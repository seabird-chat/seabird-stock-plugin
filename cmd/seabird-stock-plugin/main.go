package main

import (
	"log"
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
	logger.Level(zerolog.InfoLevel)

	coreURL := os.Getenv("SEABIRD_HOST")
	coreToken := os.Getenv("SEABIRD_TOKEN")
	finnhubToken := os.Getenv("FINNHUB_TOKEN")

	if coreURL == "" || coreToken == "" {
		log.Fatal("Missing SEABIRD_HOST or SEABIRD_TOKEN")
	}

	if finnhubToken == "" {
		log.Fatal("Missing FINNHUB_TOKEN")
	}

	c, err := stock.NewSeabirdClient(coreURL, coreToken, finnhubToken)
	if err != nil {
		log.Fatalf("Failed to connect to seabird-core: %s", err)
	}
	log.Printf("Successfully connected to seabird-core at %s", coreURL)

	err = c.Run()
	if err != nil {
		log.Fatal(err)
	}

}
