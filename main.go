package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	rr "github.com/meysam81/requestrewind/src"

	_ "github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Caller().Logger()
	config := rr.NewConfig()

	configNonConfidential := *config
	configNonConfidential.DBPassword = "REDACTED"

	log.Debug().Interface("originalConfig", &config).Msg("Starting Request Rewind")
	log.Info().Interface("config", &configNonConfidential).Msg("Starting Request Rewind")

	db := initDB(config)
	defer db.Close()

	if config.TestDomain != "" {
		log.Info().Str("domain", config.TestDomain).Msg("Running in replay mode")
		runReplayMode(db, config)
	} else {
		log.Debug().Msg("Running in record mode")
		runRecordMode(db, config)
	}
}

func initDB(config *rr.Config) *sql.DB {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName,
	)

	log.Debug().Msgf("Connecting to database: %s", connStr)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal().Err(err)
	}

	db.SetMaxOpenConns(config.DBMaxConnections)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Set timeout to 5 seconds
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatal().Msg("Failed to connect to database")
		os.Exit(1)
	}

	log.Info().Interface("stats", db.Stats()).Msg("Database connection established")

	return db
}

func runRecordMode(db *sql.DB, config *rr.Config) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var err error
		var parser *rr.Request
		requestRow, err := parser.FromHttpRequest(r)
		if err != nil {
			log.Error().Msgf("Error parsing request: %v", err)
			http.Error(w, "Error parsing request", http.StatusInternalServerError)
			return
		}

		log.Info().Interface("request", requestRow).Msg("Recording request")

		_, err = db.Exec(`
			INSERT INTO requests (id, method, path, headers, cookies, body, query_params)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`,
			requestRow.ID.String(),
			requestRow.Method,
			requestRow.Path,
			requestRow.Headers,
			requestRow.Cookies,
			requestRow.Body,
			requestRow.QueryParams,
		)
		if err != nil {
			log.Error().Msgf("Error storing request: %v", err)
			http.Error(w, "Error storing request", http.StatusInternalServerError)
			return
		}

		log.Debug().Interface("response", requestRow).Msg("Request recorded")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(requestRow)
	})

	log.Info().Msgf("Starting server on port %s", config.ServerPort)
	log.Fatal().Err(http.ListenAndServe(":"+config.ServerPort, nil))
}

func runReplayMode(db *sql.DB, config *rr.Config) {
	client := &http.Client{}

	statement, err := db.Prepare(`SELECT id, method, path, headers, body, query_params, recorded_at FROM requests ORDER BY recorded_at`)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer statement.Close()

	var successCount, failCount int

	for row, _ := statement.Query(); row.Next(); {
		var req rr.Request
		var idStr string

		if err := row.Scan(&idStr, &req.Method, &req.Path, &req.Headers, &req.Body, &req.QueryParams, &req.RecordedAt); err != nil {
			log.Error().Msgf("Error scanning row: %v", err)
			failCount++
			continue
		}

		if req.ID, err = ulid.Parse(idStr); err != nil {
			log.Error().Msgf("Error parsing ID: %v", err)
			failCount++
			continue
		}

		remoteUrl := fmt.Sprintf("%s%s", config.TestDomain, req.Path)
		httpReq, err := req.ToHttpRequest(config.TestDomain)
		if err != nil {
			log.Error().Msgf("Error creating request: %v", err)
			failCount++
			continue
		}

		log.Info().Msgf("Sending request (ID: %s): %s %s", idStr, req.Method, remoteUrl)

		resp, err := client.Do(httpReq)
		if err != nil {
			log.Error().Msgf("Error sending request: %v", err)
			failCount++
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			successCount++
		} else {
			failCount++
			log.Error().Msgf("Request failed (ID: %s): Status %d", idStr, resp.StatusCode)
		}
	}

	log.Info().Msgf("Replay completed. Success: %d, Failed: %d", successCount, failCount)
	os.Exit(failCount)
}
