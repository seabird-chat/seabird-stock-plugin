package stock

import (
	"context"
	"errors"
	"testing"
	"time"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go/v2"
)

func f32(v float32) *float32 { return &v }
func i64(v int64) *int64     { return &v }
func str(v string) *string   { return &v }
func boolp(v bool) *bool     { return &v }

type fakeMarket struct {
	profiles         map[string]finnhub.CompanyProfile2
	quotes           map[string]finnhub.Quote
	metrics          map[string]map[string]interface{}
	peers            map[string][]string
	recommendations  map[string][]finnhub.RecommendationTrend
	earnings         map[string][]finnhub.EarningResult
	earningsCalendar map[string][]finnhub.EarningRelease
	news             map[string][]finnhub.CompanyNews
	insiders         map[string][]finnhub.InsiderSentimentsData
	search           map[string][]finnhub.SymbolLookupInfo
	status           finnhub.MarketStatus
	holidays         []finnhub.MarketHolidayData
	err              error
	windows          map[string][2]time.Time
}

func (f *fakeMarket) Profile(_ context.Context, symbol string) (finnhub.CompanyProfile2, error) {
	return f.profiles[symbol], f.err
}

func (f *fakeMarket) Quote(_ context.Context, symbol string) (finnhub.Quote, error) {
	q, ok := f.quotes[symbol]
	if f.err != nil {
		return q, f.err
	}
	if !ok {
		return q, errNotFound
	}
	return q, nil
}

func (f *fakeMarket) Metrics(_ context.Context, symbol string) (map[string]interface{}, error) {
	return f.metrics[symbol], f.err
}

func (f *fakeMarket) Peers(_ context.Context, symbol string) ([]string, error) {
	return f.peers[symbol], f.err
}

func (f *fakeMarket) Recommendations(_ context.Context, symbol string) ([]finnhub.RecommendationTrend, error) {
	return f.recommendations[symbol], f.err
}

func (f *fakeMarket) Earnings(_ context.Context, symbol string) ([]finnhub.EarningResult, error) {
	return f.earnings[symbol], f.err
}

func (f *fakeMarket) EarningsCalendar(_ context.Context, symbol string, from, to time.Time) ([]finnhub.EarningRelease, error) {
	f.record("earningsCalendar", from, to)
	return f.earningsCalendar[symbol], f.err
}

func (f *fakeMarket) News(_ context.Context, symbol string, from, to time.Time) ([]finnhub.CompanyNews, error) {
	f.record("news", from, to)
	return f.news[symbol], f.err
}

func (f *fakeMarket) InsiderSentiment(_ context.Context, symbol string, from, to time.Time) ([]finnhub.InsiderSentimentsData, error) {
	f.record("insiders", from, to)
	return f.insiders[symbol], f.err
}

func (f *fakeMarket) Search(_ context.Context, query string) ([]finnhub.SymbolLookupInfo, error) {
	return f.search[query], f.err
}

func (f *fakeMarket) MarketStatus(_ context.Context, exchange string) (finnhub.MarketStatus, error) {
	return f.status, f.err
}

func (f *fakeMarket) MarketHoliday(_ context.Context, exchange string) ([]finnhub.MarketHolidayData, error) {
	return f.holidays, f.err
}

func (f *fakeMarket) record(name string, from, to time.Time) {
	if f.windows == nil {
		f.windows = map[string][2]time.Time{}
	}
	f.windows[name] = [2]time.Time{from, to}
}

var now = time.Date(2026, time.September, 7, 15, 0, 0, 0, time.UTC)

func datadog() *fakeMarket {
	return &fakeMarket{
		profiles: map[string]finnhub.CompanyProfile2{
			"DDOG": {
				Ticker:               str("DDOG"),
				Name:                 str("Datadog Inc"),
				Exchange:             str("NASDAQ NMS - GLOBAL MARKET"),
				FinnhubIndustry:      str("Technology"),
				Ipo:                  str("2019-09-19"),
				Weburl:               str("https://www.datadoghq.com/"),
				MarketCapitalization: f32(76457.84),
				ShareOutstanding:     f32(355.96),
			},
		},
		quotes: map[string]finnhub.Quote{
			"DDOG": {O: f32(212.2), C: f32(212.97), Pc: f32(214.76)},
			"SPY":  {O: f32(640), C: f32(636.8)},
			"TINY": {O: f32(10), C: f32(11)},
			"HUGE": {O: f32(100), C: f32(101)},
			"HALT": {O: f32(0), C: f32(0), Pc: f32(12.5)},
		},
		metrics: map[string]map[string]interface{}{
			"DDOG": {
				"5DayPriceReturnDaily":        -10.1713,
				"monthToDatePriceReturnDaily": -10.1713,
				"13WeekPriceReturnDaily":      -12.5903,
				"26WeekPriceReturnDaily":      90.5073,
				"yearToDatePriceReturnDaily":  56.5777,
				"52WeekPriceReturnDaily":      56.4741,
				"52WeekHigh":                  292.72,
				"52WeekHighDate":              "2026-08-05",
				"52WeekLow":                   98.01,
				"52WeekLowDate":               "2026-02-24",
				"marketCapitalization":        76457.84,
				"enterpriseValue":             77008.428,
				"peTTM":                       430.5446,
				"forwardPE":                   99.3254,
				"epsTTM":                      0.4843,
				"beta":                        1.5233313,
				"currentDividendYieldTTM":     nil,
				"priceRelativeToS&P5004Week":  -17.9814,
				"priceRelativeToS&P50013Week": -14.3206,
				"priceRelativeToS&P50026Week": 77.299,
				"priceRelativeToS&P500Ytd":    43.6334,
				"priceRelativeToS&P50052Week": 37.4781,
			},
			"SPY": {},
			"TINY": {
				"13WeekPriceReturnDaily":      10.0,
				"priceRelativeToS&P50013Week": 0.0,
				"priceRelativeToS&P50026Week": 5.0,
				"marketCapitalization":        850.0,
				"currentDividendYieldTTM":     2.5,
				"beta":                        0.8,
			},
			"HUGE": {
				"marketCapitalization": 4_500_000.0,
				"enterpriseValue":      4_480_000.0,
			},
		},
		peers: map[string][]string{
			"DDOG": {"CRM", "APP", "ADBE", "DDOG", "SNPS"},
		},
		recommendations: map[string][]finnhub.RecommendationTrend{
			"DDOG": {
				{Period: str("2026-08-01"), StrongBuy: i64(14), Buy: i64(35), Hold: i64(5), Sell: i64(1), StrongSell: i64(1)},
				{Period: str("2026-09-01"), StrongBuy: i64(15), Buy: i64(36), Hold: i64(4), Sell: i64(1), StrongSell: i64(0)},
			},
		},
		earnings: map[string][]finnhub.EarningResult{
			"DDOG": {
				{Period: str("2026-03-31"), Year: i64(2026), Quarter: i64(1), Actual: f32(0.6), Estimate: f32(0.5225), SurprisePercent: f32(14.833)},
				{Period: str("2026-06-30"), Year: i64(2026), Quarter: i64(2), Actual: f32(0.65), Estimate: f32(0.6063), SurprisePercent: f32(7.2077)},
				{Period: str("2026-09-30"), Year: i64(2026), Quarter: i64(3), Estimate: f32(0.70)},
			},
			"TINY": {
				{Period: str("2026-06-30"), Year: i64(2026), Quarter: i64(2), Actual: f32(-0.12), Estimate: f32(-0.1), SurprisePercent: f32(-20)},
			},
		},
		earningsCalendar: map[string][]finnhub.EarningRelease{
			"DDOG": {
				{Date: str("2027-02-11")},
				{Date: str("2026-11-05")},
			},
			"HUGE": {{Date: str("2026-11-19")}},
		},
		news: map[string][]finnhub.CompanyNews{
			"DDOG": {
				{Datetime: i64(1788700000), Headline: str("Older story"), Source: str("Reuters"), Url: str("https://example.com/2")},
				{Datetime: i64(1788725153), Headline: str("Is Datadog (DDOG) Undervalued?"), Source: str("Yahoo"), Url: str("https://example.com/1")},
				{Datetime: i64(1788600000), Headline: str("Even older"), Source: str("Bloomberg"), Url: str("https://example.com/3")},
				{Datetime: i64(1788500000), Headline: str("Oldest, dropped"), Source: str("CNBC"), Url: str("https://example.com/4")},
			},
		},
		search: map[string][]finnhub.SymbolLookupInfo{
			"datadog": {
				{Description: str("Datadog Inc (Pre-Reincorporation)"), Symbol: str("DDOG")},
				{Description: str("DATADOG INC"), Symbol: str("DDOG.MX")},
			},
			"data": {
				{Description: str("One"), Symbol: str("A")},
				{Description: str("Two"), Symbol: str("B")},
				{Description: str("Three"), Symbol: str("C")},
				{Description: str("Four"), Symbol: str("D")},
				{Description: str("Five"), Symbol: str("E")},
				{Description: str("Six"), Symbol: str("F")},
				{Description: str("Seven"), Symbol: str("G")},
			},
		},
		status: finnhub.MarketStatus{IsOpen: boolp(false), Holiday: str("Labor Day"), Timezone: str("America/New_York")},
		holidays: []finnhub.MarketHolidayData{
			{EventName: str("Christmas"), AtDate: str("2027-12-24")},
			{EventName: str("Thanksgiving Day"), AtDate: str("2026-11-26")},
			{EventName: str("Labor Day"), AtDate: str("2026-09-07")},
			{EventName: str("Juneteenth"), AtDate: str("2026-06-19")},
		},
		insiders: map[string][]finnhub.InsiderSentimentsData{
			"DDOG": {
				{Year: i64(2026), Month: i64(6), Change: i64(120500), Mspr: f32(12.5)},
				{Year: i64(2026), Month: i64(8), Change: i64(-83440), Mspr: f32(-30.677824)},
				{Year: i64(2026), Month: i64(7), Change: i64(1000), Mspr: f32(1)},
			},
			"TINY": {{Year: i64(2026), Month: i64(7), Change: i64(4000), Mspr: f32(60)}},
		},
	}
}

func withStatus(m *fakeMarket, status finnhub.MarketStatus) *fakeMarket {
	m.status = status
	return m
}

func TestRespond(t *testing.T) {
	tests := []struct {
		name    string
		command string
		arg     string
		market  *fakeMarket
		now     time.Time
		want    string
	}{
		{
			name:    "default quote",
			command: "stock",
			arg:     "DDOG",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - Open: $212.20, Current: $212.97 (+0.36%)",
		},
		{
			name:    "normalizes ticker case and whitespace",
			command: "stock",
			arg:     "  ddog  ",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - Open: $212.20, Current: $212.97 (+0.36%)",
		},
		{
			name:    "stonks command",
			command: "stonks",
			arg:     "DDOG",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) is STONKS ↗️. 💲2️⃣1️⃣2️⃣⏺️9️⃣7️⃣ (➕0️⃣⏺️7️⃣7️⃣)",
		},
		{
			name:    "quote without profile uses bare ticker",
			command: "stock",
			arg:     "SPY",
			market:  datadog(),
			want:    "SPY - Open: $640.00, Current: $636.80 (-0.50%)",
		},
		{
			name:    "halted quote has no open to compare against",
			command: "stock",
			arg:     "HALT",
			market:  datadog(),
			want:    "HALT - Open: $0.00, Current: $0.00 (n/a)",
		},
		{
			name:    "unknown ticker",
			command: "stock",
			arg:     "NOPE",
			market:  datadog(),
			want:    "Unable to find NOPE.",
		},
		{
			name:    "missing ticker prints usage",
			command: "stock",
			arg:     "",
			market:  datadog(),
			want:    "Usage: !stock <ticker> [5d|mtd|3m|6m|ytd|1y|52w|market-cap|stats|vs-sp500|info|earnings|analysts|peers|news|insiders], !stock search <company>, !stock market",
		},
		{
			name:    "help argument prints usage for the invoked command",
			command: "stockdev",
			arg:     "Help",
			market:  datadog(),
			want:    "Usage: !stockdev <ticker> [5d|mtd|3m|6m|ytd|1y|52w|market-cap|stats|vs-sp500|info|earnings|analysts|peers|news|insiders], !stockdev search <company>, !stockdev market",
		},
		{
			name:    "unknown subcommand",
			command: "stock",
			arg:     "DDOG bogus",
			market:  datadog(),
			want:    `Unknown option "bogus". Try: 5d, mtd, 3m, 6m, ytd, 1y, 52w, market-cap, stats, vs-sp500, info, earnings, analysts, peers, news, insiders`,
		},
		{
			name:    "ytd return with implied start price",
			command: "stock",
			arg:     "DDOG ytd",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - YTD: $137.16 → $212.97 (+55.27%)",
		},
		{
			name:    "negative 5 day return",
			command: "stock",
			arg:     "DDOG 5d",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - 5 day: $239.08 → $212.97 (-10.92%)",
		},
		{
			name:    "3 month return maps to 13 weeks",
			command: "stock",
			arg:     "DDOG 3m",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - 3 month: $245.69 → $212.97 (-13.32%)",
		},
		{
			name:    "period return unavailable",
			command: "stock",
			arg:     "SPY ytd",
			market:  datadog(),
			want:    "SPY - YTD: n/a",
		},
		{
			name:    "52 week range",
			command: "stock",
			arg:     "DDOG 52w",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - 52w high: $292.72 (2026-08-05), low: $98.01 (2026-02-24)",
		},
		{
			name:    "market cap in billions",
			command: "stock",
			arg:     "DDOG market-cap",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - Market cap: $76.46B, Enterprise value: $77.01B",
		},
		{
			name:    "market cap in trillions",
			command: "stock",
			arg:     "HUGE market-cap",
			market:  datadog(),
			want:    "HUGE - Market cap: $4.50T, Enterprise value: $4.48T",
		},
		{
			name:    "market cap in millions without enterprise value",
			command: "stock",
			arg:     "TINY market-cap",
			market:  datadog(),
			want:    "TINY - Market cap: $850.00M, Enterprise value: n/a",
		},
		{
			name:    "stats without dividend",
			command: "stock",
			arg:     "DDOG stats",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - P/E: 430.54, Forward P/E: 99.33, EPS: $0.48, Beta: 1.52, Dividend yield: n/a",
		},
		{
			name:    "stats with dividend and missing ratios",
			command: "stock",
			arg:     "TINY stats",
			market:  datadog(),
			want:    "TINY - P/E: n/a, Forward P/E: n/a, EPS: n/a, Beta: 0.80, Dividend yield: 2.50%",
		},
		{
			name:    "relative to sp500",
			command: "stock",
			arg:     "DDOG vs-sp500",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) vs S&P 500 - 3m: DDOG -12.59% vs S&P +1.73%, 6m: DDOG +90.51% vs S&P +13.21%, YTD: DDOG +56.58% vs S&P +12.94%, 1y: DDOG +56.47% vs S&P +19.00%",
		},
		{
			name:    "vs-spy aliases vs-sp500",
			command: "stock",
			arg:     "DDOG vs-spy",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) vs S&P 500 - 3m: DDOG -12.59% vs S&P +1.73%, 6m: DDOG +90.51% vs S&P +13.21%, YTD: DDOG +56.58% vs S&P +12.94%, 1y: DDOG +56.47% vs S&P +19.00%",
		},
		{
			name:    "relative to sp500 with partial data",
			command: "stock",
			arg:     "TINY vs-sp500",
			market:  datadog(),
			want:    "TINY vs S&P 500 - 3m: TINY +10.00% vs S&P +10.00%, 6m: n/a, YTD: n/a, 1y: n/a",
		},
		{
			name:    "company info",
			command: "stock",
			arg:     "DDOG info",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - NASDAQ NMS - GLOBAL MARKET, Technology, IPO 2019-09-19, 355.96M shares, https://www.datadoghq.com/",
		},
		{
			name:    "company info without profile",
			command: "stock",
			arg:     "SPY info",
			market:  datadog(),
			want:    "SPY - No profile available.",
		},
		{
			name:    "peers excludes the company itself",
			command: "stock",
			arg:     "DDOG peers",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - Peers: CRM, APP, ADBE, SNPS",
		},
		{
			name:    "analysts uses the latest period",
			command: "stock",
			arg:     "DDOG analysts",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - Analysts (2026-09): 15 strong buy, 36 buy, 4 hold, 1 sell, 0 strong sell",
		},
		{
			name:    "earnings uses latest quarter and soonest upcoming date",
			command: "stock",
			arg:     "DDOG earnings",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - Q2 2026 EPS: $0.65 vs est. $0.61 (+7.21%). Next: 2026-11-05",
		},
		{
			name:    "earnings miss without upcoming date",
			command: "stock",
			arg:     "TINY earnings",
			market:  datadog(),
			want:    "TINY - Q2 2026 EPS: $-0.12 vs est. $-0.10 (-20.00%)",
		},
		{
			name:    "earnings upcoming date only",
			command: "stock",
			arg:     "HUGE earnings",
			market:  datadog(),
			want:    "HUGE - No earnings history. Next: 2026-11-19",
		},
		{
			name:    "no earnings data",
			command: "stock",
			arg:     "SPY earnings",
			market:  datadog(),
			want:    "SPY - No earnings data.",
		},
		{
			name:    "news shows three newest headlines",
			command: "stock",
			arg:     "DDOG news",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - Is Datadog (DDOG) Undervalued? (Yahoo) <https://example.com/1> | Older story (Reuters) <https://example.com/2> | Even older (Bloomberg) <https://example.com/3>",
		},
		{
			name:    "insiders uses the latest month",
			command: "stock",
			arg:     "DDOG insiders",
			market:  datadog(),
			want:    "Datadog Inc (DDOG) - Insiders (Aug 2026): net sold 83,440 shares, 35% buys / 65% sells",
		},
		{
			name:    "insiders net buying",
			command: "stock",
			arg:     "TINY insiders",
			market:  datadog(),
			want:    "TINY - Insiders (Jul 2026): net bought 4,000 shares, 80% buys / 20% sells",
		},
		{
			name:    "search lists symbol and description",
			command: "stock",
			arg:     "search datadog",
			market:  datadog(),
			want:    "DDOG: Datadog Inc (Pre-Reincorporation), DDOG.MX: DATADOG INC",
		},
		{
			name:    "search truncates long result lists",
			command: "stock",
			arg:     "search data",
			market:  datadog(),
			want:    "A: One, B: Two, C: Three, D: Four, E: Five (+2 more)",
		},
		{
			name:    "search with multi-word query and no matches",
			command: "stock",
			arg:     "search nothing   here",
			market:  datadog(),
			want:    `No matches for "nothing here".`,
		},
		{
			name:    "search without query prints usage",
			command: "stock",
			arg:     "search",
			market:  datadog(),
			want:    "Usage: !stock <ticker> [5d|mtd|3m|6m|ytd|1y|52w|market-cap|stats|vs-sp500|info|earnings|analysts|peers|news|insiders], !stock search <company>, !stock market",
		},
		{
			name:    "market closed for holiday shows the following holiday",
			command: "stock",
			arg:     "market",
			market:  datadog(),
			want:    "US market is closed (Labor Day). Next holiday: Thanksgiving Day (2026-11-26)",
		},
		{
			name:    "market open with session",
			command: "stock",
			arg:     "market",
			market:  withStatus(datadog(), finnhub.MarketStatus{IsOpen: boolp(true), Session: str("regular")}),
			want:    "US market is open (regular). Next holiday: Thanksgiving Day (2026-11-26)",
		},
		{
			name:    "market closed outside hours uses exchange-local date for the next holiday",
			command: "stock",
			arg:     "market",
			market:  withStatus(datadog(), finnhub.MarketStatus{IsOpen: boolp(false), Timezone: str("America/New_York")}),
			now:     time.Date(2026, time.September, 7, 3, 0, 0, 0, time.UTC),
			want:    "US market is closed. Next holiday: Labor Day (2026-09-07)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			at := tt.now
			if at.IsZero() {
				at = now
			}
			got, err := respond(context.Background(), tt.market, tt.command, tt.arg, at)
			if err != nil {
				t.Fatalf("respond() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("respond() =\n  %q\nwant\n  %q", got, tt.want)
			}
		})
	}
}

func TestRespondSurfacesAPIErrors(t *testing.T) {
	boom := errors.New("finnhub down")
	for _, arg := range []string{"DDOG", "DDOG ytd", "DDOG peers", "search datadog", "market"} {
		t.Run(arg, func(t *testing.T) {
			market := datadog()
			market.err = boom
			_, err := respond(context.Background(), market, "stock", arg, now)
			if !errors.Is(err, boom) {
				t.Fatalf("respond() error = %v, want wrapped %v", err, boom)
			}
		})
	}
}

func TestRespondQueryWindows(t *testing.T) {
	tests := []struct {
		arg   string
		query string
		from  time.Time
		to    time.Time
	}{
		{"DDOG news", "news", now.AddDate(0, 0, -7), now},
		{"DDOG insiders", "insiders", now.AddDate(0, -3, 0), now},
		{"DDOG earnings", "earningsCalendar", now, now.AddDate(0, 0, 120)},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			market := datadog()
			if _, err := respond(context.Background(), market, "stock", tt.arg, now); err != nil {
				t.Fatalf("respond() error = %v", err)
			}
			got, ok := market.windows[tt.query]
			if !ok {
				t.Fatalf("%s was never queried", tt.query)
			}
			if !got[0].Equal(tt.from) || !got[1].Equal(tt.to) {
				t.Errorf("%s window = %v..%v, want %v..%v", tt.query, got[0], got[1], tt.from, tt.to)
			}
		})
	}
}
