package stock

import (
	"context"
	"errors"
	"net/http"
	"time"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go/v2"
)

var errNotFound = errors.New("ticker not found")

type marketData interface {
	Profile(ctx context.Context, symbol string) (finnhub.CompanyProfile2, error)
	Quote(ctx context.Context, symbol string) (finnhub.Quote, error)
	Metrics(ctx context.Context, symbol string) (map[string]interface{}, error)
	Peers(ctx context.Context, symbol string) ([]string, error)
	Recommendations(ctx context.Context, symbol string) ([]finnhub.RecommendationTrend, error)
	Earnings(ctx context.Context, symbol string) ([]finnhub.EarningResult, error)
	EarningsCalendar(ctx context.Context, symbol string, from, to time.Time) ([]finnhub.EarningRelease, error)
	News(ctx context.Context, symbol string, from, to time.Time) ([]finnhub.CompanyNews, error)
	InsiderSentiment(ctx context.Context, symbol string, from, to time.Time) ([]finnhub.InsiderSentimentsData, error)
	Search(ctx context.Context, query string) ([]finnhub.SymbolLookupInfo, error)
	MarketStatus(ctx context.Context, exchange string) (finnhub.MarketStatus, error)
	MarketHoliday(ctx context.Context, exchange string) ([]finnhub.MarketHolidayData, error)
}

type finnhubMarket struct {
	api *finnhub.DefaultApiService
}

const (
	apiDate     = "2006-01-02"
	httpTimeout = 10 * time.Second
)

func newFinnhubMarket(token string) *finnhubMarket {
	cfg := finnhub.NewConfiguration()
	cfg.AddDefaultHeader("X-Finnhub-Token", token)
	cfg.HTTPClient = &http.Client{Timeout: httpTimeout}
	return &finnhubMarket{api: finnhub.NewAPIClient(cfg).DefaultApi}
}

func accessible(resp *http.Response, err error) error {
	if resp != nil && resp.StatusCode == http.StatusForbidden {
		return errNotFound
	}
	return err
}

func (m *finnhubMarket) Profile(ctx context.Context, symbol string) (finnhub.CompanyProfile2, error) {
	profile, resp, err := m.api.CompanyProfile2(ctx).Symbol(symbol).Execute()
	return profile, accessible(resp, err)
}

func (m *finnhubMarket) Quote(ctx context.Context, symbol string) (finnhub.Quote, error) {
	quote, resp, err := m.api.Quote(ctx).Symbol(symbol).Execute()
	if err := accessible(resp, err); err != nil {
		return quote, err
	}
	if deref(quote.C) == 0 && deref(quote.Pc) == 0 {
		return quote, errNotFound
	}
	return quote, nil
}

func (m *finnhubMarket) Metrics(ctx context.Context, symbol string) (map[string]interface{}, error) {
	financials, _, err := m.api.CompanyBasicFinancials(ctx).Symbol(symbol).Metric("all").Execute()
	if err != nil || financials.Metric == nil {
		return nil, err
	}
	return *financials.Metric, nil
}

func (m *finnhubMarket) Peers(ctx context.Context, symbol string) ([]string, error) {
	peers, _, err := m.api.CompanyPeers(ctx).Symbol(symbol).Execute()
	return peers, err
}

func (m *finnhubMarket) Recommendations(ctx context.Context, symbol string) ([]finnhub.RecommendationTrend, error) {
	trends, _, err := m.api.RecommendationTrends(ctx).Symbol(symbol).Execute()
	return trends, err
}

func (m *finnhubMarket) Earnings(ctx context.Context, symbol string) ([]finnhub.EarningResult, error) {
	results, _, err := m.api.CompanyEarnings(ctx).Symbol(symbol).Execute()
	return results, err
}

func (m *finnhubMarket) EarningsCalendar(ctx context.Context, symbol string, from, to time.Time) ([]finnhub.EarningRelease, error) {
	calendar, _, err := m.api.EarningsCalendar(ctx).Symbol(symbol).From(from.Format(apiDate)).To(to.Format(apiDate)).Execute()
	if err != nil || calendar.EarningsCalendar == nil {
		return nil, err
	}
	return *calendar.EarningsCalendar, nil
}

func (m *finnhubMarket) News(ctx context.Context, symbol string, from, to time.Time) ([]finnhub.CompanyNews, error) {
	news, _, err := m.api.CompanyNews(ctx).Symbol(symbol).From(from.Format(apiDate)).To(to.Format(apiDate)).Execute()
	return news, err
}

func (m *finnhubMarket) InsiderSentiment(ctx context.Context, symbol string, from, to time.Time) ([]finnhub.InsiderSentimentsData, error) {
	sentiment, _, err := m.api.InsiderSentiment(ctx).Symbol(symbol).From(from.Format(apiDate)).To(to.Format(apiDate)).Execute()
	if err != nil || sentiment.Data == nil {
		return nil, err
	}
	return *sentiment.Data, nil
}

func (m *finnhubMarket) Search(ctx context.Context, query string) ([]finnhub.SymbolLookupInfo, error) {
	lookup, _, err := m.api.SymbolSearch(ctx).Q(query).Execute()
	if err != nil || lookup.Result == nil {
		return nil, err
	}
	return *lookup.Result, nil
}

func (m *finnhubMarket) MarketStatus(ctx context.Context, exchange string) (finnhub.MarketStatus, error) {
	status, _, err := m.api.MarketStatus(ctx).Exchange(exchange).Execute()
	return status, err
}

func (m *finnhubMarket) MarketHoliday(ctx context.Context, exchange string) ([]finnhub.MarketHolidayData, error) {
	holidays, _, err := m.api.MarketHoliday(ctx).Exchange(exchange).Execute()
	if err != nil || holidays.Data == nil {
		return nil, err
	}
	return *holidays.Data, nil
}
