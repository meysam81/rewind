package main

import (
	"context"
	"os"

	rewind "rewind/src"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Caller().Logger()
	config := rewind.NewConfig()

	log.Debug().Interface("config", &config).Msg("Starting Request Rewind")

	app, err := rewind.InitApp(ctx, config)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing app")
	}
	defer app.DB.Close()

	log.Debug().Interface("app", app).Msg("App initialized")

	if config.TestDomain != "" {
		log.Info().Str("domain", config.TestDomain).Msg("Running in replay mode")
		app.RunReplayMode(config)
	} else {
		log.Debug().Msg("Running in record mode")
		app.RunRecordMode(config)
	}
}
