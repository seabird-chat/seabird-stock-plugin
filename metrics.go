package stock

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const unavailable = "n/a"

type metrics map[string]interface{}

func (m metrics) number(key string) (float64, bool) {
	v, ok := m[key].(float64)
	return v, ok
}

func (m metrics) text(key string) (string, bool) {
	v, ok := m[key].(string)
	return v, ok
}

func (m metrics) format(key string, render func(float64) string) string {
	v, ok := m.number(key)
	if !ok {
		return unavailable
	}
	return render(v)
}

func fetchMetrics(ctx context.Context, api marketData, c *company) (metrics, error) {
	m, err := api.Metrics(ctx, c.Ticker)
	if err != nil {
		return nil, fmt.Errorf("metrics for %s: %w", c.Ticker, err)
	}
	return metrics(m), nil
}

type period struct {
	label  string
	metric string
}

var periods = map[string]period{
	"5d":  {"5 day", "5DayPriceReturnDaily"},
	"mtd": {"MTD", "monthToDatePriceReturnDaily"},
	"3m":  {"3 month", "13WeekPriceReturnDaily"},
	"6m":  {"6 month", "26WeekPriceReturnDaily"},
	"ytd": {"YTD", "yearToDatePriceReturnDaily"},
	"1y":  {"1 year", "52WeekPriceReturnDaily"},
}

func periodReturn(option string) tickerHandler {
	p := periods[option]
	return func(ctx context.Context, api marketData, c *company, _ time.Time) (string, error) {
		m, err := fetchMetrics(ctx, api, c)
		if err != nil {
			return "", err
		}
		closeReturn, ok := m.number(p.metric)
		if !ok || closeReturn <= -100 {
			return fmt.Sprintf("%s - %s: %s", c.Label(), p.label, unavailable), nil
		}
		start := deref(c.Quote.Pc) / (1 + closeReturn/100)
		current := deref(c.Quote.C)
		change := (current - start) / start * 100
		return fmt.Sprintf("%s - %s: %s → %s (%s)", c.Label(), p.label, money(start), money(current), percent(change)), nil
	}
}

func fiftyTwoWeekRange(ctx context.Context, api marketData, c *company, _ time.Time) (string, error) {
	m, err := fetchMetrics(ctx, api, c)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s - 52w high: %s, low: %s", c.Label(), datedPrice(m, "52WeekHigh"), datedPrice(m, "52WeekLow")), nil
}

func datedPrice(m metrics, key string) string {
	price, ok := m.number(key)
	if !ok {
		return unavailable
	}
	if date, ok := m.text(key + "Date"); ok {
		return fmt.Sprintf("%s (%s)", money(price), date)
	}
	return money(price)
}

func marketCap(ctx context.Context, api marketData, c *company, _ time.Time) (string, error) {
	m, err := fetchMetrics(ctx, api, c)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s - Market cap: %s, Enterprise value: %s",
		c.Label(), m.format("marketCapitalization", millions), m.format("enterpriseValue", millions)), nil
}

func stats(ctx context.Context, api marketData, c *company, _ time.Time) (string, error) {
	m, err := fetchMetrics(ctx, api, c)
	if err != nil {
		return "", err
	}
	ratio := func(v float64) string { return fmt.Sprintf("%.2f", v) }
	yield := func(v float64) string { return fmt.Sprintf("%.2f%%", v) }
	return fmt.Sprintf("%s - P/E: %s, Forward P/E: %s, EPS: %s, Beta: %s, Dividend yield: %s",
		c.Label(),
		m.format("peTTM", ratio),
		m.format("forwardPE", ratio),
		m.format("epsTTM", money),
		m.format("beta", ratio),
		m.format("currentDividendYieldTTM", yield)), nil
}

type comparison struct {
	label    string
	stock    string
	relative string
}

var sp500Windows = []comparison{
	{"3m", "13WeekPriceReturnDaily", "priceRelativeToS&P50013Week"},
	{"6m", "26WeekPriceReturnDaily", "priceRelativeToS&P50026Week"},
	{"YTD", "yearToDatePriceReturnDaily", "priceRelativeToS&P500Ytd"},
	{"1y", "52WeekPriceReturnDaily", "priceRelativeToS&P50052Week"},
}

func relativeToSP500(ctx context.Context, api marketData, c *company, _ time.Time) (string, error) {
	m, err := fetchMetrics(ctx, api, c)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(sp500Windows))
	for _, w := range sp500Windows {
		stock, haveStock := m.number(w.stock)
		relative, haveRelative := m.number(w.relative)
		if !haveStock || !haveRelative {
			parts = append(parts, fmt.Sprintf("%s: %s", w.label, unavailable))
			continue
		}
		index := stock - relative
		parts = append(parts, fmt.Sprintf("%s: %s %s vs S&P %s", w.label, c.Ticker, percent(stock), percent(index)))
	}
	return fmt.Sprintf("%s vs S&P 500 - %s", c.Label(), strings.Join(parts, ", ")), nil
}

func companyInfo(_ context.Context, _ marketData, c *company, _ time.Time) (string, error) {
	p := c.Profile
	if p.Name == nil {
		return fmt.Sprintf("%s - No profile available.", c.Label()), nil
	}
	return fmt.Sprintf("%s - %s, %s, IPO %s, %.2fM shares, %s",
		c.Label(), derefString(p.Exchange), derefString(p.FinnhubIndustry), derefString(p.Ipo), deref(p.ShareOutstanding), derefString(p.Weburl)), nil
}
