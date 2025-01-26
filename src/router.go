package rewind

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
)

func (appState *App) RunRecordMode(config *Config) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var err error
		var parser *RequestWriteDB
		requestRow, err := parser.FromHttpRequest(r)
		if err != nil {
			log.Error().Msgf("Error parsing request: %v", err)
			http.Error(w, "Error parsing request", http.StatusInternalServerError)
			return
		}

		log.Info().Interface("request", requestRow).Msg("Recording request")

		_, err = appState.DB.Exec(`
			INSERT INTO requests (id, method, path, headers, cookies, body, query_params)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
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
		if err := json.NewEncoder(w).Encode(requestRow); err != nil {
			log.Error().Msgf("Error encoding response: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
	})

	log.Info().Msgf("Starting server on port %s", config.ServerPort)
	log.Fatal().Err(http.ListenAndServe(":"+config.ServerPort, nil))
}

func (appState *App) RunReplayMode(config *Config) {
	client := &http.Client{}

	statement, err := appState.DB.Prepare(`SELECT id, method, path, headers, body, query_params, recorded_at FROM requests ORDER BY recorded_at`)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer statement.Close()

	var successCount, failCount int

	rows, err := statement.Query()
	if err != nil {
		log.Error().Err(err).Msg("Failed to query requests")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var req RequestWriteDB
		var idStr string

		if err := rows.Scan(&idStr, &req.Method, &req.Path, &req.Headers, &req.Body, &req.QueryParams, &req.RecordedAt); err != nil {
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
}
