package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCsvSafeNeutralizesFormulasAndKeepsNumbers(t *testing.T) {
	assert.Equal(t, "", csvSafe(""))
	assert.Equal(t, "plain", csvSafe("plain"))
	assert.Equal(t, "'=cmd()", csvSafe("=cmd()"))
	assert.Equal(t, "'+1", csvSafe("+1"))
	assert.Equal(t, "'@sum", csvSafe("@sum"))
	assert.Equal(t, "-12", csvSafe("-12"))
	assert.Equal(t, "-1.5", csvSafe("-1.5"))
	assert.Equal(t, "'-name", csvSafe("-name"))
}

func TestNormalizeLogExportLanguage(t *testing.T) {
	assert.Equal(t, i18n.LangZhCN, normalizeLogExportLanguage("zhCN"))
	assert.Equal(t, i18n.LangZhCN, normalizeLogExportLanguage("zh-CN"))
	assert.Equal(t, i18n.LangZhTW, normalizeLogExportLanguage("zhTW"))
	assert.Equal(t, i18n.LangZhTW, normalizeLogExportLanguage("zh-TW"))
	assert.Equal(t, i18n.LangEn, normalizeLogExportLanguage("en"))
	assert.Equal(t, i18n.DefaultLang, normalizeLogExportLanguage("fr"))
}

func TestLogExportHeadersFollowLanguageAndOmitIPAndContent(t *testing.T) {
	require.NoError(t, i18n.Init())

	zhHeaders := logExportHeaders(i18n.LangZhCN, false)
	assert.Equal(t, "时间", zhHeaders[0])
	assert.Equal(t, "类型名称", zhHeaders[2])
	assert.Contains(t, zhHeaders, "缓存读取")
	assert.Contains(t, zhHeaders, "缓存写入")
	assert.Contains(t, zhHeaders, "费用")
	assert.NotContains(t, zhHeaders, "ip")
	assert.NotContains(t, zhHeaders, "IP")
	assert.NotContains(t, zhHeaders, "content")
	assert.NotContains(t, zhHeaders, "内容")

	enHeaders := logExportHeaders(i18n.LangEn, true)
	assert.Equal(t, "Time", enHeaders[0])
	assert.NotContains(t, enHeaders, "Username")
	assert.NotContains(t, enHeaders, "IP")
	assert.NotContains(t, enHeaders, "Content")

	assert.Equal(t, "消耗", logExportTypeName(i18n.LangZhCN, 2))
	assert.Equal(t, "Consume", logExportTypeName(i18n.LangEn, 2))
}

func TestLogExportCacheCountsSplitsReadAndWrite(t *testing.T) {
	read, write := logExportCacheCounts(`{"cache_tokens":128,"cache_creation_tokens_5m":10,"cache_creation_tokens_1h":2}`)
	assert.Equal(t, 128, read)
	assert.Equal(t, 12, write)

	read, write = logExportCacheCounts(`{"cache_creation_tokens":40}`)
	assert.Equal(t, 0, read)
	assert.Equal(t, 40, write)

	read, write = logExportCacheCounts("")
	assert.Equal(t, 0, read)
	assert.Equal(t, 0, write)
}

func TestFormatLogExportCostKeepsSixDecimalPlaces(t *testing.T) {
	previousUnit := common.QuotaPerUnit
	previousType := operation_setting.GetGeneralSetting().QuotaDisplayType
	common.QuotaPerUnit = 500000
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	t.Cleanup(func() {
		common.QuotaPerUnit = previousUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = previousType
	})

	assert.Equal(t, "0.001076", formatLogExportCost(538))
	assert.Equal(t, "0.000000", formatLogExportCost(0))
	assert.NotContains(t, formatLogExportCost(538), "$")
}
