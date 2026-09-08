package stock

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go/v2"
)

type company struct {
	Ticker  string
	Profile finnhub.CompanyProfile2
	Quote   finnhub.Quote
}

func (c *company) Label() string {
	if c.Profile.Name == nil {
		return c.Ticker
	}
	return fmt.Sprintf("%s (%s)", *c.Profile.Name, c.Ticker)
}

type tickerHandler func(ctx context.Context, api marketData, c *company, now time.Time) (string, error)

type subcommand struct {
	name string
	run  tickerHandler
}

var subcommands = []subcommand{
	{"5d", periodReturn("5d")},
	{"mtd", periodReturn("mtd")},
	{"3m", periodReturn("3m")},
	{"6m", periodReturn("6m")},
	{"ytd", periodReturn("ytd")},
	{"1y", periodReturn("1y")},
	{"52w", fiftyTwoWeekRange},
	{"market-cap", marketCap},
	{"stats", stats},
	{"vs-sp500", relativeToSP500},
	{"info", companyInfo},
	{"earnings", earnings},
	{"analysts", analysts},
	{"peers", peers},
	{"news", news},
	{"insiders", insiders},
}

var subcommandAliases = map[string]string{
	"vs-spy": "vs-sp500",
}

func subcommandNames() []string {
	names := make([]string, 0, len(subcommands))
	for _, s := range subcommands {
		names = append(names, s.name)
	}
	return names
}

func findSubcommand(option string) (tickerHandler, bool) {
	if canonical, ok := subcommandAliases[option]; ok {
		option = canonical
	}
	for _, s := range subcommands {
		if s.name == option {
			return s.run, true
		}
	}
	return nil, false
}

func usage(command string) string {
	return fmt.Sprintf("Usage: !%[1]s <ticker> [%[2]s], !%[1]s search <company>, !%[1]s market", command, strings.Join(subcommandNames(), "|"))
}

func respond(ctx context.Context, api marketData, command, arg string, now time.Time) (string, error) {
	fields := strings.Fields(arg)
	if len(fields) == 0 {
		return usage(command), nil
	}

	switch strings.ToLower(fields[0]) {
	case "help":
		return usage(command), nil
	case "search":
		return searchCompanies(ctx, api, command, strings.Join(fields[1:], " "))
	case "market":
		return marketStatus(ctx, api, now)
	}

	ticker := strings.ToUpper(fields[0])
	handler := quoteHandler(command)
	if len(fields) > 1 {
		option := strings.ToLower(fields[1])
		sub, ok := findSubcommand(option)
		if !ok {
			return fmt.Sprintf("Unknown option %q. Try: %s", option, strings.Join(subcommandNames(), ", ")), nil
		}
		handler = sub
	}

	c, err := lookupCompany(ctx, api, ticker)
	if errors.Is(err, errNotFound) {
		return fmt.Sprintf("Unable to find %s.", ticker), nil
	}
	if err != nil {
		return "", err
	}
	return handler(ctx, api, c, now)
}

func lookupCompany(ctx context.Context, api marketData, ticker string) (*company, error) {
	profile, err := api.Profile(ctx, ticker)
	if err != nil {
		return nil, fmt.Errorf("company profile for %s: %w", ticker, err)
	}
	// If Finnhub fails to find ticker, we get a 200 back with empty values, so
	// we set a default ticker/company and only use the profile response if it
	// has valid values.
	if profile.Ticker != nil {
		ticker = *profile.Ticker
	}

	quote, err := api.Quote(ctx, ticker)
	if err != nil {
		return nil, fmt.Errorf("quote for %s: %w", ticker, err)
	}
	return &company{Ticker: ticker, Profile: profile, Quote: quote}, nil
}
