package stock

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go/v2"
	"github.com/rs/zerolog"
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
	context.Context
	*seabird.Client
	finnhubClient *finnhub.DefaultApiService
	logger        zerolog.Logger
}

// NewSeabirdClient returns a new seabird client
func NewSeabirdClient(seabirdCoreURL, seabirdCoreToken, finnhubToken string, logger zerolog.Logger) (*SeabirdClient, error) {
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
		logger:        logger,
	}, nil
}

func (c *SeabirdClient) close() error {
	return c.Client.Close()
}

func (c *SeabirdClient) reply(source *pb.ChannelSource, format string, args ...interface{}) {
	if err := c.MentionReplyf(source, format, args...); err != nil {
		c.logger.Error().Err(err).Str("channel_id", source.GetChannelId()).Msg("failed to send reply")
	}
}

func (c *SeabirdClient) stockCallback(event *pb.CommandEvent) {
	ticker := strings.ToUpper(strings.TrimSpace(event.Arg))

	cmdLog := c.logger.With().
		Str("command", event.Command).
		Str("ticker", ticker).
		Str("channel_id", event.Source.GetChannelId()).
		Logger()

	profile2, _, err := c.finnhubClient.CompanyProfile2(c.Context).Symbol(ticker).Execute()
	if err != nil {
		cmdLog.Error().Err(err).Msg("finnhub CompanyProfile2 failed")
		c.reply(event.Source, "Unable to look up %s.", ticker)
		return
	}

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

	quote, quoteResp, err := c.finnhubClient.Quote(c.Context).Symbol(ticker).Execute()

	// XXX: it's pretty terrible, but a content-length of -1 seems to be the
	// only consistent way to determine if a stock actually exists.
	if err != nil || quoteResp.ContentLength != -1 {
		if err != nil {
			cmdLog.Error().Err(err).Msg("finnhub Quote failed")
			c.reply(event.Source, "Unable to fetch quote for %s.", ticker)
			return
		}
		c.reply(event.Source, "Unable to find %s.", ticker)
		return
	}

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

		c.reply(event.Source, "%s %s. %s (%s%s)", company, stonks, current, sign, change)
	} else {
		percentChange := ((*quote.C - *quote.O) / *quote.O) * 100
		c.reply(event.Source, "%s - Open: $%.2f, Current: $%.2f (%+.2f%%)", company, *quote.O, *quote.C, percentChange)
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

	c.logger.Info().Msg("event stream open")

	for event := range events.C {
		switch v := event.GetInner().(type) {
		case *pb.Event_Command:
			switch v.Command.Command {
			case "stock", "stocks", "stonk", "stonks":
				go c.stockCallback(v.Command)
			}
		}
	}

	if closeErr := events.Close(); closeErr != nil {
		return fmt.Errorf("event stream closed: %w", closeErr)
	}
	return errors.New("event stream closed without error")
}
