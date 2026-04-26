package stock

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go/v2"
	"github.com/seabird-chat/seabird-go"
	"github.com/seabird-chat/seabird-go/pb"
)

// parsePeriod parses a period string like "30d", "2w", "6m", "1y", or "ytd"
// into a start time and display label. Returns zero time if invalid.
func parsePeriod(period string, now time.Time) (time.Time, string) {
	period = strings.ToLower(strings.TrimSpace(period))

	if period == "ytd" {
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()), "YTD"
	}

	if len(period) < 2 {
		return time.Time{}, ""
	}

	unit := period[len(period)-1]
	numStr := period[:len(period)-1]
	n := 0
	for _, c := range numStr {
		if c < '0' || c > '9' {
			return time.Time{}, ""
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return time.Time{}, ""
	}

	label := strings.ToUpper(period)
	switch unit {
	case 'd':
		return now.AddDate(0, 0, -n), label
	case 'w':
		return now.AddDate(0, 0, -n*7), label
	case 'm':
		return now.AddDate(0, -n, 0), label
	case 'y':
		return now.AddDate(-n, 0, 0), label
	default:
		return time.Time{}, ""
	}
}

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
	context.Context
	*seabird.Client
	finnhubClient *finnhub.DefaultApiService
}

// NewSeabirdClient returns a new seabird client
func NewSeabirdClient(seabirdCoreURL, seabirdCoreToken, finnhubToken string) (*SeabirdClient, error) {
	seabirdClient, err := seabird.NewClient(seabirdCoreURL, seabirdCoreToken)
	if err != nil {
		return nil, err
	}

	finnhubCfg := finnhub.NewConfiguration()
	finnhubCfg.AddDefaultHeader("X-Finnhub-Token", finnhubToken)

	return &SeabirdClient{
		Context:       context.Background(),
		Client:        seabirdClient,
		finnhubClient: finnhub.NewAPIClient(finnhubCfg).DefaultApi,
	}, nil
}

func (c *SeabirdClient) close() error {
	return c.Client.Close()
}

func parseTickerAndPeriod(arg string) (string, string) {
	parts := strings.Fields(arg)
	if len(parts) < 1 {
		return "", ""
	}

	ticker := strings.ToUpper(parts[0])

	if len(parts) >= 2 {
		periodInput := strings.ToLower(parts[1])
		if _, label := parsePeriod(periodInput, time.Now()); label != "" {
			return ticker, periodInput
		}
	}

	return ticker, ""
}

func (c *SeabirdClient) stockPeriodCallback(event *pb.CommandEvent, ticker, company, period string) {
	now := time.Now()
	from, label := parsePeriod(period, now)
	if label == "" {
		c.MentionReplyf(event.Source, "Invalid period: %s. Use formats like 1d, 2w, 3m, 1y, or ytd.", period)
		return
	}

	candles, _, err := c.finnhubClient.StockCandles(c.Context).
		Symbol(ticker).
		Resolution("D").
		From(from.Unix()).
		To(now.Unix()).
		Execute()
	if err != nil {
		log.Println(err)
		c.MentionReplyf(event.Source, "Unable to fetch historical data for %s.", ticker)
		return
	}

	if candles.S != nil && *candles.S == "no_data" {
		c.MentionReplyf(event.Source, "No historical data available for %s.", ticker)
		return
	}

	if candles.O == nil || candles.C == nil || len(*candles.O) == 0 || len(*candles.C) == 0 {
		c.MentionReplyf(event.Source, "No historical data available for %s.", ticker)
		return
	}

	startPrice := (*candles.O)[0]
	endPrice := (*candles.C)[len(*candles.C)-1]
	percentChange := ((endPrice - startPrice) / startPrice) * 100

	c.MentionReplyf(event.Source, "%s — %s: %+.2f%% ($%.2f → $%.2f)", company, label, percentChange, startPrice, endPrice)
}

func (c *SeabirdClient) stockCallback(event *pb.CommandEvent) {
	// TODO: Request debugging
	log.Printf("Processing event: %s %s %s", event.Source, event.Command, event.Arg)
	ticker, period := parseTickerAndPeriod(event.Arg)
	if ticker == "" {
		return
	}

	profile2, _, err := c.finnhubClient.CompanyProfile2(c.Context).Symbol(ticker).Execute()
	if err != nil {
		// TODO: What do we do with the error?
		log.Println(err)
		return
	}

	log.Printf("profile2 is: %+v\n", profile2)

	// If Finnhub fails to find ticker, we get a 200 back with empty values, so
	// we set a default ticker/company and only use the profile response if it
	// has valid values.
	if profile2.Ticker != nil {
		ticker = *profile2.Ticker
	}

	company := ticker
	if profile2.Name != nil {
		company = fmt.Sprintf("%s (%s)", *profile2.Name, ticker)
	}

	if period != "" {
		c.stockPeriodCallback(event, ticker, company, period)
		return
	}

	quote, quoteResp, err := c.finnhubClient.Quote(c.Context).Symbol(ticker).Execute()

	// XXX: it's pretty terrible, but a content-length of -1 seems to be the
	// only consistent way to determine if a stock actually exists.
	if err != nil || quoteResp.ContentLength != -1 {
		if err != nil {
			log.Println(err)
			return
		}
		c.MentionReplyf(event.Source, "Unable to find %s.", ticker)
	} else {
		// TODO: Don't hardcoded USD here - currency requires premium https://finnhub.io/docs/api#company-profile
		if event.Command == "stonk" || event.Command == "stonks" {
			stonks := "is STONKS ↗️"
			sign := stonkReplacements["+"]
			if *quote.C <= *quote.O {
				stonks = "is NOT STONKS ↘️"
				sign = stonkReplacements["-"]
			}

			current := stonkify(fmt.Sprintf("$%.2f", *quote.C))
			change := stonkify(fmt.Sprintf("%.2f", math.Abs(float64(*quote.C)-float64(*quote.O))))

			c.MentionReplyf(event.Source, "%s %s. %s (%s%s)", company, stonks, current, sign, change)
		} else {
			percentChange := ((*quote.C - *quote.O) / *quote.O) * 100
			c.MentionReplyf(event.Source, "%s - Open: $%.2f, Current: $%.2f (%+.2f%%)", company, *quote.O, *quote.C, percentChange)
		}
	}

}

// Run runs
func (c *SeabirdClient) Run() error {
	events, err := c.StreamEvents(map[string]*pb.CommandMetadata{
		"stock": {
			Name:      "stock",
			ShortHelp: "<ticker> [period]",
			FullHelp:  "Returns current stock price for given ticker. Optionally specify a period (e.g. 1d, 2w, 3m, 1y, ytd) to see price change.",
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
