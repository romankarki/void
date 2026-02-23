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

type GoldPrice struct {
	Price      string
	Change     string
	ChangePct  string
	LastUpdate string
}

type goldResponse struct {
	Price      string `json:"price"`
	Change     string `json:"change_amount"`
	ChangePct  string `json:"change_percentage"`
	LastUpdate string `json:"last_updated"`
}

type ExchangeRate struct {
	From       string
	To         string
	Rate       string
	LastUpdate string
}

type exchangeResponse struct {
	RealTimeCurrencyExchangeRate struct {
		FromCurrencyCode string `json:"1. From_Currency Code"`
		FromCurrencyName string `json:"2. From_Currency Name"`
		ToCurrencyCode   string `json:"3. To_Currency Code"`
		ToCurrencyName   string `json:"4. To_Currency Name"`
		ExchangeRate     string `json:"5. Exchange Rate"`
		LastRefreshed    string `json:"6. Last Refreshed"`
		TimeZone         string `json:"7. Time Zone"`
		BidPrice         string `json:"8. Bid Price"`
		AskPrice         string `json:"9. Ask Price"`
	} `json:"Realtime Currency Exchange Rate"`
	ErrorMessage string `json:"Error Message"`
	Note         string `json:"Note"`
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

func FetchGoldPrice(apiKey, metal string) (*GoldPrice, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key not configured. Add [api] section with alpha_vantage key to config")
	}

	symbol := "GOLD"
	if strings.ToUpper(metal) == "SILVER" || strings.ToUpper(metal) == "S" {
		symbol = "SILVER"
	}

	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GOLD_SILVER_SPOT&symbol=%s&apikey=%s", symbol, apiKey)
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

	var raw goldResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		var errResp struct {
			Information string `json:"Information"`
			Note        string `json:"Note"`
		}
		if json.Unmarshal(body, &errResp); errResp.Information != "" || errResp.Note != "" {
			return nil, fmt.Errorf("API rate limit or error: %s%s", errResp.Information, errResp.Note)
		}
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &GoldPrice{
		Price:      raw.Price,
		Change:     raw.Change,
		ChangePct:  raw.ChangePct,
		LastUpdate: raw.LastUpdate,
	}, nil
}

func FetchExchangeRate(apiKey, from, to string) (*ExchangeRate, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key not configured. Add [api] section with alpha_vantage key to config")
	}

	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))

	if from == "" || to == "" {
		return nil, fmt.Errorf("both from and to currency codes are required")
	}

	url := fmt.Sprintf("https://www.alphavantage.co/query?function=CURRENCY_EXCHANGE_RATE&from_currency=%s&to_currency=%s&apikey=%s", from, to, apiKey)
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

	var raw exchangeResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if raw.ErrorMessage != "" {
		return nil, fmt.Errorf("API error: %s", raw.ErrorMessage)
	}

	if raw.Note != "" {
		return nil, fmt.Errorf("API rate limit: %s", raw.Note)
	}

	if raw.RealTimeCurrencyExchangeRate.ExchangeRate == "" {
		return nil, fmt.Errorf("no exchange rate data available")
	}

	return &ExchangeRate{
		From:       raw.RealTimeCurrencyExchangeRate.FromCurrencyCode,
		To:         raw.RealTimeCurrencyExchangeRate.ToCurrencyCode,
		Rate:       raw.RealTimeCurrencyExchangeRate.ExchangeRate,
		LastUpdate: raw.RealTimeCurrencyExchangeRate.LastRefreshed,
	}, nil
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

func FormatGoldPrice(gold *GoldPrice, metal string) string {
	metalName := "GOLD"
	if strings.ToUpper(metal) == "SILVER" || strings.ToUpper(metal) == "S" {
		metalName = "SILVER"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n %s SPOT PRICE\n", metalName))
	sb.WriteString(" \x1b[1;36m─────────────────────────────────────────────────────────\x1b[0m\n")
	sb.WriteString(fmt.Sprintf(" \x1b[1;33mPrice:        $%s\x1b[0m\n", gold.Price))
	change := gold.ChangePct
	if !strings.HasPrefix(change, "-") && change != "" {
		change = "+" + change
		sb.WriteString(fmt.Sprintf(" \x1b[32mChange:       %s (%s%%)\x1b[0m\n", gold.Change, change))
	} else {
		sb.WriteString(fmt.Sprintf(" \x1b[31mChange:       %s (%s%%)\x1b[0m\n", gold.Change, change))
	}
	sb.WriteString(fmt.Sprintf(" \x1b[90mLast Update:  %s\x1b[0m\n", gold.LastUpdate))
	return sb.String()
}

func FormatExchangeRate(ex *ExchangeRate) string {
	var sb strings.Builder
	sb.WriteString("\n EXCHANGE RATE\n")
	sb.WriteString(" \x1b[1;36m─────────────────────────────────────────────────────────\x1b[0m\n")
	sb.WriteString(fmt.Sprintf(" \x1b[1;33m%s → %s\x1b[0m\n", ex.From, ex.To))
	sb.WriteString(" \x1b[36m─────────────────────────────────────────────────────────\x1b[0m\n")
	sb.WriteString(fmt.Sprintf(" \x1b[1;32mRate:         %s\x1b[0m\n", ex.Rate))
	sb.WriteString(fmt.Sprintf(" \x1b[90mLast Update:  %s\x1b[0m\n", ex.LastUpdate))
	return sb.String()
}

func formatVolume(v string) string {
	vol := strings.TrimSpace(v)
	if vol == "" {
		return "-"
	}
	return vol
}
