package main

import (
	"log"
	"net/http"
	"os"

	stock "github.com/jaredledvina/seabird-stock-plugin"
	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

// HTTPHandler create a handler struct
type HTTPHandler struct{}

// implement `ServeHTTP` method on `HttpHandler` struct
func (h HTTPHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	// create response binary data
	data := []byte("Hello World!") // slice of bytes

	// write `data` to response
	res.Write(data)
}

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
	servicePort := os.Getenv("PORT")

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
	// create a new handler
	handler := HTTPHandler{}

	// listen and serve
	go http.ListenAndServe(servicePort, handler)

	err = c.Run()
	if err != nil {
		log.Fatal(err)
	}

}
