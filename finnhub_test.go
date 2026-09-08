package stock

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go/v2"
)

func TestLookupCompanyAgainstFinnhubResponses(t *testing.T) {
	forbidden := `{"error":"You don't have access to this resource."}`
	responses := map[string]map[string]string{
		"/stock/profile2": {
			"DDOG":    `{"ticker":"DDOG","name":"Datadog Inc","marketCapitalization":76457.8}`,
			"HELP":    `{"ticker":"HELP.NE","name":"Help Co"}`,
			"HELP.NE": forbidden,
			"HALTED":  `{}`,
			"NOPEXYZ": `{}`,
		},
		"/quote": {
			"DDOG":    `{"c":212.97,"d":-1.79,"dp":-0.8335,"h":214.71,"l":210.95,"o":212.2,"pc":214.76,"t":1788552000}`,
			"HELP.NE": forbidden,
			"HALTED":  `{"c":0,"d":null,"dp":null,"h":0,"l":0,"o":0,"pc":12.5,"t":1788552000}`,
			"NOPEXYZ": `{"c":0,"d":null,"dp":null,"h":0,"l":0,"o":0,"pc":0,"t":0}`,
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Finnhub-Token") != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		body := responses[r.URL.Path][r.URL.Query().Get("symbol")]
		w.Header().Set("Content-Type", "application/json")
		if body == forbidden {
			w.WriteHeader(http.StatusForbidden)
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	market := &finnhubMarket{api: finnhub.NewAPIClient(&finnhub.Configuration{
		DefaultHeader: map[string]string{"X-Finnhub-Token": "secret"},
		Servers:       finnhub.ServerConfigurations{{URL: srv.URL}},
	}).DefaultApi}

	tests := []struct {
		symbol string
		label  string
		close  float64
	}{
		{"DDOG", "Datadog Inc (DDOG)", 212.97},
		{"HALTED", "HALTED", 0},
		{"NOPEXYZ", "", 0},
		{"HELP.NE", "", 0},
		{"HELP", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.symbol, func(t *testing.T) {
			c, err := lookupCompany(context.Background(), market, tt.symbol)
			if tt.label == "" {
				if !errors.Is(err, errNotFound) {
					t.Fatalf("lookupCompany() error = %v, want errNotFound", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("lookupCompany() error = %v", err)
			}
			if c.Label() != tt.label {
				t.Errorf("Label() = %q, want %q", c.Label(), tt.label)
			}
			if got := deref(c.Quote.C); got != float64(float32(tt.close)) {
				t.Errorf("close = %v, want %v", got, tt.close)
			}
		})
	}
}
