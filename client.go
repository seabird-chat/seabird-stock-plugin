package stock

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go"
	"github.com/antihax/optional"
	seabird "github.com/seabird-chat/seabird-go"
	"github.com/seabird-chat/seabird-go/pb"
)

// SeabirdClient is a basic client for seabird
type SeabirdClient struct {
	*seabird.SeabirdClient

	finnhubClient  *finnhub.DefaultApiService
	finnhubContext context.Context
}

// NewSeabirdClient returns a new seabird client
func NewSeabirdClient(seabirdCoreURL, seabirdCoreToken, finnhubToken string) (*SeabirdClient, error) {
	seabirdClient, err := seabird.NewSeabirdClient(seabirdCoreURL, seabirdCoreToken)
	if err != nil {
		return nil, err
	}

	return &SeabirdClient{
		SeabirdClient: seabirdClient,
		finnhubClient: finnhub.NewAPIClient(finnhub.NewConfiguration()).DefaultApi,
		finnhubContext: context.WithValue(context.Background(), finnhub.ContextAPIKey, finnhub.APIKey{
			Key: finnhubToken,
		}),
	}, nil
}

func (c *SeabirdClient) close() error {
	return c.SeabirdClient.Close()
}

func (c *SeabirdClient) stockCallback(event *pb.CommandEvent) {
	// TODO: Request debugging
	log.Printf("Processing event: %s %s %s", event.Source, event.Command, event.Arg)

	query := event.Arg

	profile2, _, err := c.finnhubClient.CompanyProfile2(c.finnhubContext, &finnhub.CompanyProfile2Opts{Symbol: optional.NewString(query)})
	if err != nil {
		// TODO: What do we do with the error?
		log.Println(err)
	}

	log.Printf("profile2 is: %+v\n", profile2)

	// If Finnhub fails to find ticker, we get a 200 back with empty values, so
	// we set a default ticker/company and only use the profile response if it
	// has valid values.
	ticker := strings.ToUpper(query)
	if profile2.Ticker != "" {
		ticker = profile2.Ticker
	}

	company := ticker
	if profile2.Name != "" {
		company = fmt.Sprintf("%s (%s)", profile2.Name, ticker)
	}

	quote, quoteResp, err := c.finnhubClient.Quote(c.finnhubContext, query)

	// XXX: it's pretty terrible, but a content-length of -1 seems to be the
	// only consistent way to determine if a stock actually exists.
	if err != nil || quoteResp.ContentLength != -1 {
		if err != nil {
			log.Println(err)
		}
		c.Replyf(event.Source, "%s: Unable to find %s.", event.Source.GetUser().GetDisplayName(), query)
	} else {
		percentChange := ((quote.C - quote.O) / quote.O) * 100
		// TODO: Don't hardcoded USD here - currency requires premium https://finnhub.io/docs/api#company-profile
		c.Replyf(event.Source, "%s: %s - Open: $%.2f, Current: $%.2f (%+.2f%%)", event.Source.GetUser().GetDisplayName(), company, quote.O, quote.C, percentChange)
	}

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
