package stock

import (
	"context"
	"fmt"
	"strings"
	"time"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go/v2"
)

const searchLimit = 5

func searchCompanies(ctx context.Context, api marketData, command, query string) (string, error) {
	if query == "" {
		return usage(command), nil
	}
	matches, err := api.Search(ctx, query)
	if err != nil {
		return "", fmt.Errorf("search %q: %w", query, err)
	}
	if len(matches) == 0 {
		return fmt.Sprintf("No matches for %q.", query), nil
	}

	shown := matches[:min(searchLimit, len(matches))]
	parts := make([]string, 0, len(shown))
	for _, m := range shown {
		parts = append(parts, fmt.Sprintf("%s: %s", derefString(m.Symbol), derefString(m.Description)))
	}
	text := strings.Join(parts, ", ")
	if extra := len(matches) - len(shown); extra > 0 {
		text += fmt.Sprintf(" (+%d more)", extra)
	}
	return text, nil
}

const exchange = "US"

func marketStatus(ctx context.Context, api marketData, now time.Time) (string, error) {
	status, err := api.MarketStatus(ctx, exchange)
	if err != nil {
		return "", fmt.Errorf("market status: %w", err)
	}
	holidays, err := api.MarketHoliday(ctx, exchange)
	if err != nil {
		return "", fmt.Errorf("market holidays: %w", err)
	}

	state := "closed"
	if status.IsOpen != nil && *status.IsOpen {
		state = "open"
	}
	reason := ""
	if detail := firstNonEmpty(derefString(status.Holiday), derefString(status.Session)); detail != "" {
		reason = fmt.Sprintf(" (%s)", detail)
	}

	text := fmt.Sprintf("%s market is %s%s.", exchange, state, reason)
	if next, ok := nextHoliday(holidays, now.In(exchangeZone(status))); ok {
		text += fmt.Sprintf(" Next holiday: %s (%s)", derefString(next.EventName), derefString(next.AtDate))
	}
	return text, nil
}

func nextHoliday(holidays []finnhub.MarketHolidayData, now time.Time) (finnhub.MarketHolidayData, bool) {
	today := now.Format(apiDate)
	var next finnhub.MarketHolidayData
	found := false
	for _, h := range holidays {
		date := derefString(h.AtDate)
		if date <= today {
			continue
		}
		if !found || date < derefString(next.AtDate) {
			next, found = h, true
		}
	}
	return next, found
}

func exchangeZone(status finnhub.MarketStatus) *time.Location {
	zone, err := time.LoadLocation(derefString(status.Timezone))
	if err != nil {
		return time.UTC
	}
	return zone
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
