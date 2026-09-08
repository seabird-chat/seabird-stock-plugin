package stock

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
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

func quoteHandler(command string) tickerHandler {
	if command == "stonk" || command == "stonks" {
		return stonksQuote
	}
	return dailyQuote
}

// TODO: Don't hardcoded USD here - currency requires premium https://finnhub.io/docs/api#company-profile
func dailyQuote(_ context.Context, _ marketData, c *company, _ time.Time) (string, error) {
	open, current := deref(c.Quote.O), deref(c.Quote.C)
	change := unavailable
	if open != 0 {
		change = percent((current - open) / open * 100)
	}
	return fmt.Sprintf("%s - Open: %s, Current: %s (%s)", c.Label(), money(open), money(current), change), nil
}

func stonksQuote(_ context.Context, _ marketData, c *company, _ time.Time) (string, error) {
	open, current := deref(c.Quote.O), deref(c.Quote.C)
	stonks := "is STONKS ↗️"
	sign := stonkReplacements["+"]
	if current <= open {
		stonks = "is NOT STONKS ↘️"
		sign = stonkReplacements["-"]
	}

	price := stonkify(money(current))
	change := stonkify(fmt.Sprintf("%.2f", math.Abs(current-open)))
	return fmt.Sprintf("%s %s. %s (%s%s)", c.Label(), stonks, price, sign, change), nil
}
