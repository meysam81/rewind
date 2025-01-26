package rewind

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/zerolog/log"
)

func InitApp(ctx context.Context, config *Config) (*App, error) {
	db, err := sql.Open("postgres", config.DSN)
	if err != nil {
		return nil, err
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	err = db.PingContext(ctxWithTimeout)
	if err != nil {
		return nil, err
	}

	log.Info().Interface("stats", db.Stats()).Msg("Database connection established")

	return &App{DB: db, Config: config}, nil
}
