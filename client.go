package stock

import (
	"context"
	"errors"
	"fmt"
	"log"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go"
	"github.com/antihax/optional"
	seabird "github.com/seabird-chat/seabird-go"
	"github.com/seabird-chat/seabird-go/pb"
)

// SeabirdClient is a basic client for seabird
type SeabirdClient struct {
	*seabird.SeabirdClient
	FinnhubToken string
}

// NewSeabirdClient returns a new seabird client
func NewSeabirdClient(seabirdCoreURL, seabirdCoreToken, finnhubToken string) (*SeabirdClient, error) {
	seabirdClient, err := seabird.NewSeabirdClient(seabirdCoreURL, seabirdCoreToken)
	if err != nil {
		return nil, err
	}

	return &SeabirdClient{
		SeabirdClient: seabirdClient,
		FinnhubToken:  finnhubToken,
	}, nil
}

func (c *SeabirdClient) close() error {
	return c.SeabirdClient.Close()
}

func (c *SeabirdClient) stockCallback(event *pb.CommandEvent) {
	// TODO: Request debugging
	log.Printf("Processing event: %s %s %s", event.Source, event.Command, event.Arg)

	finnhubClient := finnhub.NewAPIClient(finnhub.NewConfiguration()).DefaultApi
	auth := context.WithValue(context.Background(), finnhub.ContextAPIKey, finnhub.APIKey{
		Key: c.FinnhubToken,
	})
	ticker := event.Arg
	profile2, _, err := finnhubClient.CompanyProfile2(auth, &finnhub.CompanyProfile2Opts{Symbol: optional.NewString(ticker)})
	if err != nil {
		// TODO: What do we do with the error?
		log.Println(err)
	}
	// Use the ticker from the company profile to handle mixed case queries
	ticker = profile2.Ticker
	quote, _, err := finnhubClient.Quote(auth, ticker)
	if err != nil {
		// TODO: What do we do with the error?
		log.Println(err)
	}
	var company string
	if profile2.Name != "" {
		company = fmt.Sprintf("%s (%s)", profile2.Name, ticker)
	} else {
		company = fmt.Sprintf("%s", ticker)
	}
	// TODO: Don't hardcoded USD here - currency requires premium https://finnhub.io/docs/api#company-profile
	c.Replyf(event.Source, "%s: Current price of %s is: $%+v USD", event.Source.GetUser().GetDisplayName(), company, quote.C)
}

// Run runs
func (c *SeabirdClient) Run() error {
	events, err := c.StreamEvents(map[string]*pb.CommandMetadata{
		"stock": {
			Name:      "stock",
			ShortHelp: "<ticker>",
			FullHelp:  "Returns current stock price for given ticker",
		},
	})
	if err != nil {
		return err
	}

	defer events.Close()
	for event := range events.C {
		switch v := event.GetInner().(type) {
		case *pb.Event_Command:
			if v.Command.Command == "stock" {
				go c.stockCallback(v.Command)
			}
		}
	}
	return errors.New("event stream closed")
}
