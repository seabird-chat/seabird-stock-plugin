# stock-plugin
Simple plugin for [seabird](https://github.com/seabird-chat/seabird-core) adding the `!stock` command backed by https://finnhub.io/

## Commands

| Command | Output |
|---|---|
| `!stock <ticker>` | Today's open and current price |
| `!stonks <ticker>` | The same, but STONKS |
| `!stock <ticker> 5d\|mtd\|3m\|6m\|ytd\|1y` | Price change over the period |
| `!stock <ticker> 52w` | 52-week high and low with dates |
| `!stock <ticker> market-cap` | Market cap and enterprise value |
| `!stock <ticker> stats` | P/E, forward P/E, EPS, beta, dividend yield |
| `!stock <ticker> vs-sp500` (or `vs-spy`) | Price performance relative to the S&P 500 |
| `!stock <ticker> info` | Exchange, industry, IPO date, shares outstanding, website |
| `!stock <ticker> earnings` | Latest quarter EPS vs estimate and next earnings date |
| `!stock <ticker> analysts` | Latest analyst buy/hold/sell counts |
| `!stock <ticker> peers` | Peer tickers |
| `!stock <ticker> news` | Three most recent headlines |
| `!stock <ticker> insiders` | Latest month of insider net shares bought or sold and the buy/sell split |
| `!stock search <company>` | Look up tickers by company name |
| `!stock market` | Whether the US market is open and the next holiday |

Everything uses endpoints available on Finnhub's free tier.

## Configuration

| Variable | Purpose |
|---|---|
| `SEABIRD_HOST` | seabird-core gRPC address |
| `SEABIRD_TOKEN` | seabird-core auth token |
| `FINNHUB_TOKEN` | Finnhub API key |
| `SEABIRD_COMMAND` | Command name to register (default `stock`); use a different name to run a dev build alongside production |
