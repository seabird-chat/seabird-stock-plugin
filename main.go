package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go"
	"github.com/jaredledvina/stock-plugin/pb"
	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

// SeabirdClient is a basic client for seabird
type SeabirdClient struct {
	grpcChannel  *grpc.ClientConn
	inner        pb.SeabirdClient
	FinnhubToken string
}

func newSeabirdClient(seabirdCoreURL, seabirdCoreToken, finnhubToken string) (*SeabirdClient, error) {
	grpcChannel, err := newGRPCClient(seabirdCoreURL, seabirdCoreToken)
	if err != nil {
		return nil, err
	}

	return &SeabirdClient{
		grpcChannel:  grpcChannel,
		inner:        pb.NewSeabirdClient(grpcChannel),
		FinnhubToken: finnhubToken,
	}, nil
}

func (c *SeabirdClient) close() error {
	return c.grpcChannel.Close()
}

func (c *SeabirdClient) reply(source *pb.ChannelSource, msg string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.inner.SendMessage(ctx, &pb.SendMessageRequest{
		ChannelId: source.GetChannelId(),
		Text:      msg,
	})
	return err
}

func (c *SeabirdClient) stockCallback(event *pb.CommandEvent) {
	go func() {
		finnhubClient := finnhub.NewAPIClient(finnhub.NewConfiguration()).DefaultApi
		auth := context.WithValue(context.Background(), finnhub.ContextAPIKey, finnhub.APIKey{
			Key: c.FinnhubToken,
		})
		ticker := event.Arg
		quote, _, err := finnhubClient.Quote(auth, ticker)
		if err != nil {
			// TODO: What do we do with the error?
			fmt.Println(err)
		}
		msg := fmt.Sprintf("%s: Current price of %s is: %+v", event.Source.GetUser().GetDisplayName(), ticker, quote.C)
		c.reply(event.Source, msg)
	}()
}

func (c *SeabirdClient) run() error {
	events, err := c.inner.StreamEvents(
		context.Background(),
		&pb.StreamEventsRequest{
			Commands: map[string]*pb.CommandMetadata{
				"stock": {
					Name:      "stock",
					ShortHelp: "<ticker>",
					FullHelp:  "Returns current stock price for given ticker",
				},
			},
		},
	)
	if err != nil {
		return err
	}

	for {
		event, err := events.Recv()
		if err != nil {
			return err
		}

		switch v := event.GetInner().(type) {
		case *pb.Event_Command:
			if v.Command.Command == "stock" {
				c.stockCallback(v.Command)
			}
		}
	}
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

	if coreURL == "" || coreToken == "" {
		log.Fatal("Missing SEABIRD_HOST or SEABIRD_TOKEN")
	}

	if finnhubToken == "" {
		log.Fatal("Missing FINNHUB_TOKEN")
	}

	c, err := newSeabirdClient(
		coreURL,
		coreToken,
		finnhubToken,
	)
	if err != nil {
		log.Fatal(err)
	}

	err = c.run()
	if err != nil {
		log.Fatal(err)
	}

}
