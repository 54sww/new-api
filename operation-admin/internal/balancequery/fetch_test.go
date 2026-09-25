package balancequery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAmountNewAPIUserSelfQuota(t *testing.T) {
	// 1 USD = 500_000 Quota → 2_500_000 quota = $5
	raw := []byte(`{"success":true,"message":"","data":{"id":1,"username":"root","quota":2500000,"used_quota":100000}}`)
	cfg := PresetNewAPI("pat-test", "https://api.example.com")
	amount, currency, err := ParseAmount(raw, cfg)
	require.NoError(t, err)
	require.Equal(t, 2500000.0, amount)
	require.Equal(t, "USD", currency)
	require.InDelta(t, 5.0, ToUSD(amount, currency, cfg.FxToUSD), 1e-9)
}

func TestParseAmountNestedAndString(t *testing.T) {
	raw := []byte(`{"data":{"balance":"1,234.56"}}`)
	amount, currency, err := ParseAmount(raw, Config{AmountPath: "data.balance", Currency: "USD"})
	require.NoError(t, err)
	require.Equal(t, 1234.56, amount)
	require.Equal(t, "USD", currency)
}

func TestValidateRequiresURL(t *testing.T) {
	err := Config{Enabled: true, AmountPath: "data.quota"}.Validate()
	require.Error(t, err)
}

func TestApplyBaseURLPlaceholder(t *testing.T) {
	cfg := PresetNewAPI("sk-x", "https://host.example")
	n := cfg.Normalized()
	url := applyPlaceholders(n.URL, n.APIKey, n.BaseURL)
	require.Equal(t, "https://host.example/api/user/self", url)
	auth := applyPlaceholders(n.Headers["Authorization"], n.APIKey, n.BaseURL)
	require.Equal(t, "Bearer sk-x", auth)
}
