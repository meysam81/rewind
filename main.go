package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Request struct {
	ID          ulid.ULID `json:"id"`
	Method      string    `json:"method"`
	Path        string    `json:"path"`
	Headers     []byte    `json:"headers"`
	Body        []byte    `json:"body"`
	QueryParams []byte    `json:"query_params"`
	RecordedAt  time.Time `json:"recorded_at"`
}

func NewRequest(r *http.Request) *Request {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Warn().Msgf("Error reading body: %v", err)
	}
	defer r.Body.Close()

	headerJSON, err := json.Marshal(r.Header)
	if err != nil {
		log.Warn().Msgf("Error marshaling headers: %v", err)
	}

	params, err := json.Marshal(r.URL.RawQuery)
	if err != nil {
		log.Warn().Msgf("Error marshaling params: %v", err)
	}

	return &Request{
		ID:          ulid.Make(),
		Method:      r.Method,
		Path:        r.URL.Path,
		Headers:     headerJSON,
		Body:        body,
		QueryParams: params,
		RecordedAt:  time.Now(),
	}
}

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	ServerPort string
	TestDomain string
}

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Caller().Logger()

	serverPort := flag.String("p", "8080", "Port to run the server on")
	dbHost := flag.String("dbhost", "localhost", "Database host")
	dbPort := flag.String("dbport", "5432", "Database port")
	dbUser := flag.String("dbuser", "requestrewind", "Database user")
	dbPass := flag.String("dbpass", "requestrewind", "Database password")
	dbName := flag.String("dbname", "requestrewind", "Database name")
	testDomain := flag.String("d", "", "Domain to test against")
	flag.Parse()

	config := &Config{
		DBHost:     *dbHost,
		DBPort:     *dbPort,
		DBUser:     *dbUser,
		DBPassword: *dbPass,
		DBName:     *dbName,
		ServerPort: *serverPort,
		TestDomain: *testDomain,
	}

	configNonConfidential := *config
	configNonConfidential.DBPassword = "REDACTED"

	log.Info().Interface("config", &configNonConfidential).Msg("Starting Request Rewind")

	db := initDB(config)
	defer db.Close()

	if config.TestDomain != "" {
		log.Info().Str("domain", config.TestDomain).Msg("Running in replay mode")
		runReplayMode(db, config)
	} else {
		log.Info().Msg("Running in record mode")
		runRecordMode(db, config)
	}
}

func initDB(config *Config) *sql.DB {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal().Err(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal().Err(err)
	}

	return db
}

func runRecordMode(db *sql.DB, config *Config) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var err error
		requestRow := NewRequest(r)
		log.Info().Interface("request", requestRow).Msg("Recording request")

		_, err = db.Exec(`
			INSERT INTO requests (id, method, path, headers, body, query_params)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`,
			requestRow.ID.String(),
			requestRow.Method,
			requestRow.Path,
			requestRow.Headers,
			requestRow.Body,
			requestRow.QueryParams,
		)
		if err != nil {
			log.Error().Msgf("Error storing request: %v", err)
			http.Error(w, "Error storing request", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Request recorded successfully")
	})

	log.Info().Msgf("Starting server on port %s", config.ServerPort)
	log.Fatal().Err(http.ListenAndServe(":"+config.ServerPort, nil))
}

func runReplayMode(db *sql.DB, config *Config) {
	client := &http.Client{}

	statement, err := db.Prepare(`SELECT id, method, path, headers, body, query_params FROM requests ORDER BY recorded_at`)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer statement.Close()

	var successCount, failCount int

	for row, _ := statement.Query(); row.Next(); {
		var req Request
		var idStr string

		if err := row.Scan(&idStr, &req.Method, &req.Path, &req.Headers, &req.Body, &req.QueryParams); err != nil {
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

		var params string
		if err := json.Unmarshal(req.QueryParams, &params); err != nil {
			log.Error().Msgf("Error unmarshaling query params: %v", err)
			failCount++
			continue
		}
		if params != "" {
			remoteUrl = fmt.Sprintf("%s?%s", remoteUrl, params)
		}

		httpReq, err := http.NewRequest(req.Method, remoteUrl, nil)
		if err != nil {
			log.Error().Msgf("Error creating request: %v", err)
			failCount++
			continue
		}

		var headers map[string][]string
		if len(req.Headers) > 0 {
			if err := json.Unmarshal(req.Headers, &headers); err != nil {
				log.Error().Msgf("Error unmarshaling headers: %v", err)
				failCount++
				continue
			}
		}
		for key, values := range headers {
			for _, value := range values {
				httpReq.Header.Add(key, value)
			}
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
