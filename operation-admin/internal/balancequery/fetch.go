package balancequery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

const maxResponseBytes = 256 << 10

// Config describes how to fetch upstream balance for one channel.
// Example (new-api):
//
//	{
//	  "enabled": true,
//	  "base_url": "https://your-new-api.example.com",
//	  "url": "{{base_url}}/api/user/self",
//	  "method": "GET",
//	  "headers": {"Authorization": "Bearer {token}"},
//	  "amount_path": "data.quota",
//	  "currency": "USD",
//	  "fx_to_usd": 0.000002,
//	  "api_key": "access-token"
//	}
type Config struct {
	Enabled      bool              `json:"enabled"`
	BaseURL      string            `json:"base_url"` // optional; expands {{base_url}} in url/headers/body
	URL          string            `json:"url"`
	Method       string            `json:"method"` // GET/POST, default GET
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body"`          // optional raw body; supports {{api_key}} / {{base_url}}
	AmountPath   string            `json:"amount_path"`   // gjson path, e.g. data.quota
	CurrencyPath string            `json:"currency_path"` // optional gjson path
	Currency     string            `json:"currency"`      // fallback currency when path empty
	FxToUSD      float64           `json:"fx_to_usd"`     // amount * fx = USD; 0 or 1 means treat as USD
	APIKey       string            `json:"api_key"`
	TimeoutSec   int               `json:"timeout_sec"`
}

type Result struct {
	AmountRaw  float64 `json:"amount_raw"`
	Currency   string  `json:"currency"`
	AmountUSD  float64 `json:"amount_usd"`
	RawBody    string  `json:"raw_body,omitempty"`
	StatusCode int     `json:"status_code"`
}

func (c Config) Normalized() Config {
	out := c
	out.Method = strings.ToUpper(strings.TrimSpace(out.Method))
	if out.Method == "" {
		out.Method = http.MethodGet
	}
	out.URL = strings.TrimSpace(out.URL)
	out.BaseURL = strings.TrimRight(strings.TrimSpace(out.BaseURL), "/")
	out.AmountPath = strings.TrimSpace(out.AmountPath)
	if out.AmountPath == "" {
		out.AmountPath = "data.quota"
	}
	out.CurrencyPath = strings.TrimSpace(out.CurrencyPath)
	out.Currency = strings.ToUpper(strings.TrimSpace(out.Currency))
	if out.TimeoutSec <= 0 {
		out.TimeoutSec = 30
	}
	if out.Headers == nil {
		out.Headers = map[string]string{}
	}
	return out
}

func (c Config) Validate() error {
	n := c.Normalized()
	if !n.Enabled {
		return fmt.Errorf("balance query disabled")
	}
	if n.URL == "" {
		return fmt.Errorf("url is required")
	}
	if n.Method != http.MethodGet && n.Method != http.MethodPost && n.Method != http.MethodPut {
		return fmt.Errorf("unsupported method: %s", n.Method)
	}
	if n.AmountPath == "" {
		return fmt.Errorf("amount_path is required")
	}
	return nil
}

func (c Config) Masked() Config {
	out := c
	if out.APIKey != "" {
		out.APIKey = maskSecret(out.APIKey)
	}
	if out.Headers != nil {
		h := make(map[string]string, len(out.Headers))
		for k, v := range out.Headers {
			if strings.Contains(strings.ToLower(k), "authorization") || strings.Contains(strings.ToLower(k), "key") || strings.Contains(v, "{{api_key}}") {
				h[k] = v // template ok; real key is in api_key field
				continue
			}
			h[k] = v
		}
		out.Headers = h
	}
	return out
}

func maskSecret(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

func Fetch(cfg Config) (*Result, error) {
	n := cfg.Normalized()
	if err := n.Validate(); err != nil {
		return nil, err
	}

	url := applyPlaceholders(n.URL, n.APIKey, n.BaseURL)
	bodyStr := applyPlaceholders(n.Body, n.APIKey, n.BaseURL)
	var reader io.Reader
	if bodyStr != "" && n.Method != http.MethodGet {
		reader = bytes.NewBufferString(bodyStr)
	}

	req, err := http.NewRequest(n.Method, url, reader)
	if err != nil {
		return nil, err
	}
	for k, v := range n.Headers {
		req.Header.Set(k, applyPlaceholders(v, n.APIKey, n.BaseURL))
	}
	if bodyStr != "" && req.Header.Get("Content-Type") == "" && n.Method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: time.Duration(n.TimeoutSec) * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxResponseBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", maxResponseBytes)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		snippet := string(raw)
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		return nil, fmt.Errorf("HTTP %d: %s", res.StatusCode, snippet)
	}

	amount, currency, err := ParseAmount(raw, n)
	if err != nil {
		return nil, err
	}
	usd := ToUSD(amount, currency, n.FxToUSD)
	return &Result{
		AmountRaw:  amount,
		Currency:   currency,
		AmountUSD:  usd,
		RawBody:    string(raw),
		StatusCode: res.StatusCode,
	}, nil
}

func ParseAmount(raw []byte, cfg Config) (amount float64, currency string, err error) {
	n := cfg.Normalized()
	if !gjson.ValidBytes(raw) {
		return 0, "", fmt.Errorf("response is not valid JSON")
	}
	val := gjson.GetBytes(raw, n.AmountPath)
	if !val.Exists() {
		return 0, "", fmt.Errorf("amount_path %q not found in response", n.AmountPath)
	}
	amount, err = asFloat(val)
	if err != nil {
		return 0, "", fmt.Errorf("amount_path %q: %w", n.AmountPath, err)
	}
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, "", fmt.Errorf("invalid amount value")
	}

	currency = n.Currency
	if n.CurrencyPath != "" {
		cv := gjson.GetBytes(raw, n.CurrencyPath)
		if cv.Exists() && cv.String() != "" {
			currency = strings.ToUpper(strings.TrimSpace(cv.String()))
		}
	}
	if currency == "" {
		currency = "USD"
	}
	return amount, currency, nil
}

func ToUSD(amount float64, currency string, fx float64) float64 {
	// fx_to_usd > 0 always scales (e.g. quota→USD = 1/500000, CNY→USD = 0.14).
	// fx <= 0 means the amount is already in the desired unit.
	if fx > 0 {
		return amount * fx
	}
	_ = currency
	return amount
}

func asFloat(v gjson.Result) (float64, error) {
	switch v.Type {
	case gjson.Number:
		return v.Float(), nil
	case gjson.String:
		s := strings.TrimSpace(v.String())
		s = strings.ReplaceAll(s, ",", "")
		return strconv.ParseFloat(s, 64)
	default:
		// try JSON number in raw
		var n float64
		if err := json.Unmarshal([]byte(v.Raw), &n); err == nil {
			return n, nil
		}
		return 0, fmt.Errorf("value is not a number (%s)", v.Type.String())
	}
}

func applyPlaceholders(s, apiKey, baseURL string) string {
	s = strings.ReplaceAll(s, "{{api_key}}", apiKey)
	s = strings.ReplaceAll(s, "{{API_KEY}}", apiKey)
	s = strings.ReplaceAll(s, "{{token}}", apiKey)
	s = strings.ReplaceAll(s, "{{TOKEN}}", apiKey)
	s = strings.ReplaceAll(s, "{token}", apiKey)
	s = strings.ReplaceAll(s, "{{base_url}}", baseURL)
	s = strings.ReplaceAll(s, "{{BASE_URL}}", baseURL)
	return s
}

// DefaultQuotaPerUnit matches new-api common.QuotaPerUnit (500 * 1000).
const DefaultQuotaPerUnit = 500 * 1000.0

// PresetNewAPI is the template for upstream new-api / One API instances.
// Balance: GET {base}/api/user/self → data.quota (remaining quota units).
// Auth: Authorization: Bearer {token}  (user Access Token / PAT)
// Convert: USD = quota / 500000  (1 USD = 500,000 Quota)
func PresetNewAPI(apiKey, baseURL string) Config {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return Config{
		Enabled:    true,
		BaseURL:    baseURL,
		URL:        "{{base_url}}/api/user/self",
		Method:     http.MethodGet,
		Headers:    map[string]string{"Authorization": "Bearer {token}"},
		AmountPath: "data.quota",
		Currency:   "USD",
		FxToUSD:    1.0 / DefaultQuotaPerUnit,
		APIKey:     apiKey,
		TimeoutSec: 30,
	}
}
