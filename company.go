package stock

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go/v2"
)

func peers(ctx context.Context, api marketData, c *company, _ time.Time) (string, error) {
	symbols, err := api.Peers(ctx, c.Ticker)
	if err != nil {
		return "", fmt.Errorf("peers for %s: %w", c.Ticker, err)
	}
	others := slices.DeleteFunc(slices.Clone(symbols), func(s string) bool { return s == c.Ticker })
	if len(others) == 0 {
		return fmt.Sprintf("%s - No peers found.", c.Label()), nil
	}
	return fmt.Sprintf("%s - Peers: %s", c.Label(), strings.Join(others, ", ")), nil
}

func analysts(ctx context.Context, api marketData, c *company, _ time.Time) (string, error) {
	trends, err := api.Recommendations(ctx, c.Ticker)
	if err != nil {
		return "", fmt.Errorf("recommendations for %s: %w", c.Ticker, err)
	}
	if len(trends) == 0 {
		return fmt.Sprintf("%s - No analyst recommendations.", c.Label()), nil
	}
	latest := slices.MaxFunc(trends, func(a, b finnhub.RecommendationTrend) int {
		return strings.Compare(derefString(a.Period), derefString(b.Period))
	})
	month := strings.TrimSuffix(derefString(latest.Period), "-01")
	return fmt.Sprintf("%s - Analysts (%s): %d strong buy, %d buy, %d hold, %d sell, %d strong sell",
		c.Label(), month, derefInt(latest.StrongBuy), derefInt(latest.Buy), derefInt(latest.Hold), derefInt(latest.Sell), derefInt(latest.StrongSell)), nil
}

const earningsLookahead = 120

func earnings(ctx context.Context, api marketData, c *company, now time.Time) (string, error) {
	results, err := api.Earnings(ctx, c.Ticker)
	if err != nil {
		return "", fmt.Errorf("earnings for %s: %w", c.Ticker, err)
	}
	upcoming, err := api.EarningsCalendar(ctx, c.Ticker, now, now.AddDate(0, 0, earningsLookahead))
	if err != nil {
		return "", fmt.Errorf("earnings calendar for %s: %w", c.Ticker, err)
	}

	next := ""
	if len(upcoming) > 0 {
		soonest := slices.MinFunc(upcoming, func(a, b finnhub.EarningRelease) int {
			return strings.Compare(derefString(a.Date), derefString(b.Date))
		})
		next = ". Next: " + derefString(soonest.Date)
	}

	reported := slices.DeleteFunc(slices.Clone(results), func(r finnhub.EarningResult) bool { return r.Actual == nil })
	if len(reported) == 0 {
		if next == "" {
			return fmt.Sprintf("%s - No earnings data.", c.Label()), nil
		}
		return fmt.Sprintf("%s - No earnings history%s", c.Label(), next), nil
	}

	latest := slices.MaxFunc(reported, func(a, b finnhub.EarningResult) int {
		return strings.Compare(derefString(a.Period), derefString(b.Period))
	})
	return fmt.Sprintf("%s - Q%d %d EPS: %s vs est. %s (%s)%s",
		c.Label(), derefInt(latest.Quarter), derefInt(latest.Year),
		money(deref(latest.Actual)), money(deref(latest.Estimate)), percent(deref(latest.SurprisePercent)), next), nil
}

const (
	newsLookbackDays = 7
	newsHeadlines    = 3
)

func news(ctx context.Context, api marketData, c *company, now time.Time) (string, error) {
	stories, err := api.News(ctx, c.Ticker, now.AddDate(0, 0, -newsLookbackDays), now)
	if err != nil {
		return "", fmt.Errorf("news for %s: %w", c.Ticker, err)
	}
	if len(stories) == 0 {
		return fmt.Sprintf("%s - No recent news.", c.Label()), nil
	}

	sorted := slices.Clone(stories)
	slices.SortFunc(sorted, func(a, b finnhub.CompanyNews) int {
		return cmp.Compare(derefInt(b.Datetime), derefInt(a.Datetime))
	})
	headlines := make([]string, 0, newsHeadlines)
	for _, story := range sorted[:min(newsHeadlines, len(sorted))] {
		headlines = append(headlines, fmt.Sprintf("%s (%s) <%s>", derefString(story.Headline), derefString(story.Source), derefString(story.Url)))
	}
	return fmt.Sprintf("%s - %s", c.Label(), strings.Join(headlines, " | ")), nil
}

const insiderLookbackMonths = 3

func insiders(ctx context.Context, api marketData, c *company, now time.Time) (string, error) {
	months, err := api.InsiderSentiment(ctx, c.Ticker, now.AddDate(0, -insiderLookbackMonths, 0), now)
	if err != nil {
		return "", fmt.Errorf("insider sentiment for %s: %w", c.Ticker, err)
	}
	if len(months) == 0 {
		return fmt.Sprintf("%s - No insider activity.", c.Label()), nil
	}

	latest := slices.MaxFunc(months, func(a, b finnhub.InsiderSentimentsData) int {
		return cmp.Compare(derefInt(a.Year)*12+derefInt(a.Month), derefInt(b.Year)*12+derefInt(b.Month))
	})
	month := time.Date(int(derefInt(latest.Year)), time.Month(derefInt(latest.Month)), 1, 0, 0, 0, 0, time.UTC).Format("Jan 2006")
	buys := (100 + deref(latest.Mspr)) / 2
	return fmt.Sprintf("%s - Insiders (%s): %s, %.0f%% buys / %.0f%% sells",
		c.Label(), month, netShares(derefInt(latest.Change)), buys, 100-buys), nil
}

func netShares(change int64) string {
	switch {
	case change > 0:
		return fmt.Sprintf("net bought %s shares", thousands(change))
	case change < 0:
		return fmt.Sprintf("net sold %s shares", thousands(-change))
	default:
		return "net 0 shares"
	}
}
