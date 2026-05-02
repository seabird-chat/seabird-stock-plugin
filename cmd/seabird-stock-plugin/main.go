package main

import (
	"os"

	stock "github.com/jaredledvina/seabird-stock-plugin"
	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

func main() {
	// Attempt to load from .env if it exists
	_ = godotenv.Load()

	var logger zerolog.Logger

	if isatty.IsTerminal(os.Stdout.Fd()) {
		logger = zerolog.New(zerolog.NewConsoleWriter())
	} else {
		logger = zerolog.New(os.Stdout)
	}

	logger = logger.With().Timestamp().Logger()

	coreURL := os.Getenv("SEABIRD_HOST")
	coreToken := os.Getenv("SEABIRD_TOKEN")
	finnhubToken := os.Getenv("FINNHUB_TOKEN")

	if coreURL == "" || coreToken == "" {
		logger.Fatal().Msg("Missing SEABIRD_HOST or SEABIRD_TOKEN")
	}

	if finnhubToken == "" {
		logger.Fatal().Msg("Missing FINNHUB_TOKEN")
	}

	c, err := stock.NewSeabirdClient(coreURL, coreToken, finnhubToken, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to dial seabird-core")
	}
	logger.Info().Str("host", coreURL).Msg("dialed seabird-core")

	if err := c.Run(); err != nil {
		logger.Fatal().Err(err).Msg("event stream terminated")
	}
}
