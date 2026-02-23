package stocks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Stock struct {
	Ticker    string
	Price     string
	Change    string
	ChangePct string
	Volume    string
}

type Response struct {
	TopGainers []Stock `json:"top_gainers"`
	TopLosers  []Stock `json:"top_losers"`
}

type rawStock struct {
	Ticker    string `json:"ticker"`
	Price     string `json:"price"`
	Change    string `json:"change_amount"`
	ChangePct string `json:"change_percentage"`
	Volume    string `json:"volume"`
}

type rawResponse struct {
	TopGainers []rawStock `json:"top_gainers"`
	TopLosers  []rawStock `json:"top_losers"`
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

func FetchGainers(apiKey string) ([]Stock, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key not configured. Add [api] section with alpha_vantage key to config")
	}

	url := fmt.Sprintf("https://www.alphavantage.co/query?function=TOP_GAINERS_LOSERS&apikey=%s", apiKey)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var raw rawResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	var stocks []Stock
	for _, r := range raw.TopGainers {
		stocks = append(stocks, Stock{
			Ticker:    r.Ticker,
			Price:     r.Price,
			Change:    r.Change,
			ChangePct: r.ChangePct,
			Volume:    r.Volume,
		})
	}

	return stocks, nil
}

func FormatTable(stocks []Stock) string {
	if len(stocks) == 0 {
		return "No gainers found"
	}

	var sb strings.Builder
	sb.WriteString("\n TOP GAINERS\n")
	sb.WriteString(" \x1b[1;36m─────────────────────────────────────────────────────────\x1b[0m\n")
	sb.WriteString(fmt.Sprintf(" \x1b[1m%-6s %10s %12s %15s\x1b[0m\n", "Ticker", "Price", "Change", "Volume"))
	sb.WriteString(" \x1b[36m─────────────────────────────────────────────────────────\x1b[0m\n")

	for _, s := range stocks {
		change := s.ChangePct
		if !strings.HasPrefix(change, "-") && change != "" {
			change = "+" + change
		}
		sb.WriteString(fmt.Sprintf(" \x1b[1;32m%-6s\x1b[0m %10s \x1b[32m%12s\x1b[0m %15s\n",
			s.Ticker, s.Price, change, formatVolume(s.Volume)))
	}

	return sb.String()
}

func formatVolume(v string) string {
	vol := strings.TrimSpace(v)
	if vol == "" {
		return "-"
	}
	return vol
}
