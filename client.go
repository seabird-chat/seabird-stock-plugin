package stock

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

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
		Client:        seabirdClient,
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

	query := strings.TrimSpace(strings.ToUpper(event.Arg))

	profile2, _, err := c.finnhubClient.CompanyProfile2(c.finnhubContext, &finnhub.CompanyProfile2Opts{Symbol: optional.NewString(query)})
	if err != nil {
		// TODO: What do we do with the error?
		log.Println(err)
		return
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
			return
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

			current := stonkify(fmt.Sprintf("$%.2f", quote.C))
			change := stonkify(fmt.Sprintf("%.2f", math.Abs(float64(quote.C)-float64(quote.O))))

			c.MentionReplyf(event.Source, "%s %s. %s (%s%s)", company, stonks, current, sign, change)
		} else {
			percentChange := ((quote.C - quote.O) / quote.O) * 100
			c.MentionReplyf(event.Source, "%s - Open: $%.2f, Current: $%.2f (%+.2f%%)", company, quote.O, quote.C, percentChange)
		}
	}

}

func (c *SeabirdClient) exchangeCallback(event *pb.CommandEvent) {
	cryptoExchange, _, err := c.finnhubClient.CryptoExchanges(c.finnhubContext)
	if err != nil {
		log.Println(err)
	}

	sort.Strings(cryptoExchange)

	c.MentionReplyf(event.Source, "Supported Exchanges: %v", strings.Join(cryptoExchange, ", "))
}

func (c *SeabirdClient) symbolsCallback(event *pb.CommandEvent) {
	cryptoSymbol, _, err := c.finnhubClient.CryptoSymbols(c.finnhubContext, strings.ToUpper(event.Arg))
	if err != nil {
		log.Println(err)
		return
	}

	var symbols []string
	for _, symbol := range cryptoSymbol {
		symbols = append(symbols, symbol.DisplaySymbol)
	}

	sort.Strings(symbols)

	c.MentionReplyf(
		event.Source,
		"Supported Symbols on %s: %v",
		strings.Title(strings.ToLower(event.Arg)),
		strings.Join(symbols, ", "))
}

func (c *SeabirdClient) cryptoCallback(event *pb.CommandEvent) {
	ticker := strings.ToUpper(strings.TrimSpace(event.Arg))
	exchange := "COINBASE"
	currency := "USD"

	split := strings.SplitN(ticker, ":", 2)
	if len(split) == 2 {
		exchange = split[0]
		ticker = split[1]
	}

	split = strings.SplitN(ticker, "/", 2)
	if len(split) == 2 {
		ticker = split[0]
		currency = split[1]
	}

	// TODO: we probably don't need to look this up every time - it should be
	// fine to cache this and use it later. For now, it's not a huge deal to
	// have an extra call for every lookup.
	symbols, _, err := c.finnhubClient.CryptoSymbols(c.finnhubContext, exchange)
	if err != nil {
		c.MentionReply(event.Source, "failed to look up exchange symbols")
		return
	}

	// Convert from human readable form to internal symbol. This could be more
	// efficient, but it works for now.
	var query string
	target := ticker + "/" + currency
	for _, symbol := range symbols {
		if symbol.DisplaySymbol == target {
			query = symbol.Symbol
			break
		}
	}

	if query == "" {
		c.MentionReply(event.Source, "ticker or conversion not found on that exchange")
		return
	}

	// Gets the candle sticks for the last day and returns the Closed price which appears to an accurate current price
	cryptoCandles, _, err := c.finnhubClient.CryptoCandles(c.finnhubContext, query, "D", time.Now().AddDate(0, 0, -1).Unix(), time.Now().Unix())
	if err != nil {
		log.Println(err)
		return
	}

	if len(cryptoCandles.C) == 0 {
		c.MentionReply(event.Source, "no results")
		return
	}

	current := cryptoCandles.C[0]
	c.MentionReplyf(event.Source, "%s - Current: $%.2f on %s", event.Arg, current, strings.Title(strings.ToLower(exchange)))
}

// Run runs
func (c *SeabirdClient) Run() error {
	events, err := c.StreamEvents(map[string]*pb.CommandMetadata{
		"stock": {
			Name:      "stock",
			ShortHelp: "<ticker>",
			FullHelp:  "Returns current stock price for given ticker",
		},
		"crypto": {
			Name:      "crypto",
			ShortHelp: "[<exchange:]<symbol>[/<conversion]",
			FullHelp:  "Returns current crypto price for given symbol",
		},
		"exchange": {
			Name:      "exchange",
			ShortHelp: "",
			FullHelp:  "Returns supported crypto exchanges",
		},
		"symbols": {
			Name:      "symbols",
			ShortHelp: "<exchange>",
			FullHelp:  "Returns supported crypto symbols for a given exchange",
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
			case "crypto":
				go c.cryptoCallback(v.Command)
			case "exchange":
				go c.exchangeCallback(v.Command)
			case "symbols":
				go c.symbolsCallback(v.Command)
			}
		}
	}
	return errors.New("event stream closed")
}
