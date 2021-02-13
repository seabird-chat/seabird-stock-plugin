package stock

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go"
	"github.com/antihax/optional"
	"github.com/seabird-chat/seabird-go"
	"github.com/seabird-chat/seabird-go/pb"
)

var stonkReplacements = map[string]string{
	"1": "1️⃣",
	"2": "2️⃣",
	"3": "3️⃣",
	"4": "4️⃣",
	"5": "5️⃣",
	"6": "6️⃣",
	"7": "7️⃣",
	"8": "8️⃣",
	"9": "9️⃣",
	"0": "0️⃣",
	"-": "➖",
	"+": "➕",
	".": "⏺️",
	"$": "💲",
}

func stonkify(in string) string {
	for k, v := range stonkReplacements {
		in = strings.ReplaceAll(in, k, v)
	}
	return in
}

// SeabirdClient is a basic client for seabird
type SeabirdClient struct {
	*seabird.Client

	finnhubClient  *finnhub.DefaultApiService
	finnhubContext context.Context
}

// NewSeabirdClient returns a new seabird client
func NewSeabirdClient(seabirdCoreURL, seabirdCoreToken, finnhubToken string) (*SeabirdClient, error) {
	seabirdClient, err := seabird.NewClient(seabirdCoreURL, seabirdCoreToken)
	if err != nil {
		return nil, err
	}

	return &SeabirdClient{
		Client: seabirdClient,
		finnhubClient: finnhub.NewAPIClient(finnhub.NewConfiguration()).DefaultApi,
		finnhubContext: context.WithValue(context.Background(), finnhub.ContextAPIKey, finnhub.APIKey{
			Key: finnhubToken,
		}),
	}, nil
}

func (c *SeabirdClient) close() error {
	return c.Client.Close()
}

func (c *SeabirdClient) stockCallback(event *pb.CommandEvent) {
	// TODO: Request debugging
	log.Printf("Processing event: %s %s %s", event.Source, event.Command, event.Arg)

	query := strings.TrimSpace(event.Arg)

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
		c.MentionReplyf(event.Source, "Unable to find %s.", query)
	} else {
		// TODO: Don't hardcoded USD here - currency requires premium https://finnhub.io/docs/api#company-profile
		if event.Command == "stonk" || event.Command == "stonks" {
			stonks := "is STONKS ↗️"
			sign := stonkReplacements["+"]
			if quote.C <= quote.O {
				stonks = "is NOT STONKS ↘️"
				sign = stonkReplacements["-"]
			}

			current := stonkify(fmt.Sprintf("$%.2f", quote.O))
			change := stonkify(fmt.Sprintf("%.2f", math.Abs(float64(quote.C)-float64(quote.O))))

			c.MentionReplyf(event.Source, "%s %s. %s (%s%s)", company, stonks, current, sign, change)
		} else {
			percentChange := ((quote.C - quote.O) / quote.O) * 100
			c.MentionReplyf(event.Source, "%s - Open: $%.2f, Current: $%.2f (%+.2f%%)", company, quote.O, quote.C, percentChange)
		}
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
			switch v.Command.Command {
			case "stock", "stocks", "stonk", "stonks":
				go c.stockCallback(v.Command)
			}
		}
	}
	return errors.New("event stream closed")
}
