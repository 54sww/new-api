package controller

import (
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

const logExportBatchSize = 500

func ExportAllLogs(c *gin.Context) {
	exportLogs(c, false)
}

func ExportUserLogs(c *gin.Context) {
	exportLogs(c, true)
}

func exportLogs(c *gin.Context, selfOnly bool) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	window, err := model.ClampLogExportWindow(c.GetInt("role"), startTimestamp, endTimestamp, time.Now().Unix())
	if errors.Is(err, model.ErrLogExportOutsideWindow) {
		common.ApiErrorI18n(c, i18n.MsgLogExportOutsideWindow)
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	logType, _ := strconv.Atoi(c.Query("type"))
	query := model.LogExportQuery{
		LogType:           logType,
		StartTimestamp:    window.Start,
		EndTimestamp:      window.End,
		ModelName:         c.Query("model_name"),
		TokenName:         c.Query("token_name"),
		Group:             c.Query("group"),
		RequestID:         c.Query("request_id"),
		UpstreamRequestID: c.Query("upstream_request_id"),
	}
	if selfOnly {
		query.UserID = c.GetInt("id")
		if query.UserID == 0 {
			common.ApiErrorI18n(c, i18n.MsgUnauthorized)
			return
		}
	} else {
		query.Username = c.Query("username")
		query.Channel, _ = strconv.Atoi(c.Query("channel"))
	}

	filename := fmt.Sprintf("usage-logs-%s.csv", time.Now().Format("20060102-150405"))
	lang := resolveLogExportLanguage(c)
	var cursor *model.LogExportCursor
	headerWritten := false
	writer := csv.NewWriter(c.Writer)

	for {
		if c.Request.Context().Err() != nil {
			return
		}
		logs, listErr := model.ListLogsForExport(c.Request.Context(), query, cursor, logExportBatchSize)
		if listErr != nil {
			if !headerWritten {
				common.ApiError(c, listErr)
				return
			}
			common.SysError("failed to export logs: " + listErr.Error())
			return
		}
		if !selfOnly && len(logs) > 0 {
			if nameErr := model.PopulateLogChannelNames(logs); nameErr != nil {
				if !headerWritten {
					common.ApiError(c, nameErr)
					return
				}
				common.SysError("failed to export log channel names: " + nameErr.Error())
				return
			}
		}
		if !headerWritten {
			if window.Clamped {
				c.Header("X-Log-Export-Clamped", "1")
			}
			c.Header("Content-Type", "text/csv; charset=utf-8")
			c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
			c.Header("Cache-Control", "no-store")
			c.Status(http.StatusOK)
			if _, writeErr := c.Writer.Write([]byte{0xEF, 0xBB, 0xBF}); writeErr != nil {
				return
			}
			if writeErr := writer.Write(logExportHeaders(lang, selfOnly)); writeErr != nil {
				return
			}
			headerWritten = true
		}
		if len(logs) == 0 {
			writer.Flush()
			c.Writer.Flush()
			return
		}

		if writeErr := writeLogExportRows(writer, logs, lang, selfOnly); writeErr != nil {
			return
		}
		writer.Flush()
		if writeErr := writer.Error(); writeErr != nil {
			return
		}
		c.Writer.Flush()

		next := logExportCursor(logs[len(logs)-1])
		if cursor != nil && cursor.Same(next) {
			return
		}
		cursor = &next
		if len(logs) < logExportBatchSize {
			return
		}
	}
}

func logExportCursor(log *model.Log) model.LogExportCursor {
	return model.LogExportCursor{
		CreatedAt: log.CreatedAt,
		ID:        log.Id,
		RequestID: log.RequestId,
	}
}

func logExportHeaders(lang string, selfOnly bool) []string {
	headers := []string{
		logExportText(lang, i18n.MsgLogExportTime),
		logExportText(lang, i18n.MsgLogExportType),
		logExportText(lang, i18n.MsgLogExportTypeName),
	}
	if !selfOnly {
		headers = append(headers,
			logExportText(lang, i18n.MsgLogExportUsername),
			logExportText(lang, i18n.MsgLogExportUserID),
		)
	}
	headers = append(headers,
		logExportText(lang, i18n.MsgLogExportTokenName),
		logExportText(lang, i18n.MsgLogExportModelName),
		logExportText(lang, i18n.MsgLogExportGroup),
	)
	if !selfOnly {
		headers = append(headers,
			logExportText(lang, i18n.MsgLogExportChannelID),
			logExportText(lang, i18n.MsgLogExportChannelName),
		)
	}
	headers = append(headers,
		logExportText(lang, i18n.MsgLogExportPromptTokens),
		logExportText(lang, i18n.MsgLogExportCompletionTokens),
		logExportText(lang, i18n.MsgLogExportCacheRead),
		logExportText(lang, i18n.MsgLogExportCacheWrite),
		logExportText(lang, i18n.MsgLogExportCost),
		logExportText(lang, i18n.MsgLogExportUseTime),
		logExportText(lang, i18n.MsgLogExportStream),
		logExportText(lang, i18n.MsgLogExportRequestID),
		logExportText(lang, i18n.MsgLogExportUpstreamRequestID),
	)
	return headers
}

func writeLogExportRows(writer *csv.Writer, logs []*model.Log, lang string, selfOnly bool) error {
	for _, log := range logs {
		if err := writer.Write(logExportRow(log, lang, selfOnly)); err != nil {
			return err
		}
	}
	return nil
}

func logExportRow(log *model.Log, lang string, selfOnly bool) []string {
	row := []string{
		time.Unix(log.CreatedAt, 0).UTC().Format(time.RFC3339),
		strconv.Itoa(log.Type),
		logExportTypeName(lang, log.Type),
	}
	if !selfOnly {
		row = append(row,
			csvSafe(log.Username),
			strconv.Itoa(log.UserId),
		)
	}
	row = append(row,
		csvSafe(log.TokenName),
		csvSafe(log.ModelName),
		csvSafe(log.Group),
	)
	if !selfOnly {
		row = append(row,
			strconv.Itoa(log.ChannelId),
			csvSafe(log.ChannelName),
		)
	}
	cacheRead, cacheWrite := logExportCacheCounts(log.Other)
	row = append(row,
		strconv.Itoa(log.PromptTokens),
		strconv.Itoa(log.CompletionTokens),
		strconv.Itoa(cacheRead),
		strconv.Itoa(cacheWrite),
		formatLogExportCost(log.Quota),
		strconv.Itoa(log.UseTime),
		strconv.FormatBool(log.IsStream),
		csvSafe(log.RequestId),
		csvSafe(log.UpstreamRequestId),
	)
	return row
}

func logExportTypeName(lang string, logType int) string {
	switch logType {
	case model.LogTypeTopup:
		return logExportText(lang, i18n.MsgLogExportTypeTopup)
	case model.LogTypeConsume:
		return logExportText(lang, i18n.MsgLogExportTypeConsume)
	case model.LogTypeManage:
		return logExportText(lang, i18n.MsgLogExportTypeManage)
	case model.LogTypeSystem:
		return logExportText(lang, i18n.MsgLogExportTypeSystem)
	case model.LogTypeError:
		return logExportText(lang, i18n.MsgLogExportTypeError)
	case model.LogTypeRefund:
		return logExportText(lang, i18n.MsgLogExportTypeRefund)
	case model.LogTypeLogin:
		return logExportText(lang, i18n.MsgLogExportTypeLogin)
	default:
		return logExportText(lang, i18n.MsgLogExportTypeUnknown)
	}
}

func resolveLogExportLanguage(c *gin.Context) string {
	if userSetting, ok := common.GetContextKeyType[dto.UserSetting](c, constant.ContextKeyUserSetting); ok && userSetting.Language != "" {
		return normalizeLogExportLanguage(userSetting.Language)
	}
	if userID := c.GetInt("id"); userID > 0 {
		if lang := model.GetUserLanguage(userID); lang != "" {
			return normalizeLogExportLanguage(lang)
		}
	}
	return normalizeLogExportLanguage(i18n.GetLangFromContext(c))
}

func normalizeLogExportLanguage(lang string) string {
	folded := strings.ToLower(strings.TrimSpace(lang))
	compact := strings.ReplaceAll(strings.ReplaceAll(folded, "-", ""), "_", "")
	switch {
	case compact == "zhtw" || strings.HasPrefix(folded, "zh-hk") || strings.HasPrefix(folded, "zh-mo") || strings.Contains(folded, "hant"):
		return i18n.LangZhTW
	case strings.HasPrefix(folded, "zh"):
		return i18n.LangZhCN
	case strings.HasPrefix(folded, "en"):
		return i18n.LangEn
	default:
		return i18n.DefaultLang
	}
}

func logExportText(lang, key string) string {
	return i18n.Translate(lang, key)
}

func logExportCacheCounts(other string) (int, int) {
	if other == "" {
		return 0, 0
	}
	parsed, err := common.StrToMap(other)
	if err != nil || parsed == nil {
		return 0, 0
	}
	read := logExportJSONCount(parsed["cache_tokens"])
	write5m := logExportJSONCount(parsed["cache_creation_tokens_5m"])
	write1h := logExportJSONCount(parsed["cache_creation_tokens_1h"])
	write := write5m + write1h
	if write == 0 {
		write = logExportJSONCount(parsed["cache_creation_tokens"])
	}
	return read, write
}

func logExportJSONCount(value any) int {
	count, ok := value.(float64)
	if !ok || math.IsNaN(count) || math.IsInf(count, 0) || count <= 0 {
		return 0
	}
	return int(count)
}

// formatLogExportCost converts quota into the site display currency as a plain
// decimal. A currency symbol makes spreadsheets treat the cell as money and
// round it to two places, so small costs such as 0.001076 show up as 0.00.
func formatLogExportCost(quota int) string {
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens || common.QuotaPerUnit <= 0 {
		return strconv.Itoa(quota)
	}
	amount := float64(quota) / common.QuotaPerUnit * operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	return strconv.FormatFloat(amount, 'f', 6, 64)
}

// csvSafe neutralizes spreadsheet formulas without turning numeric values,
// including negatives, into text.
func csvSafe(value string) string {
	if value == "" {
		return ""
	}
	switch value[0] {
	case '=', '+', '@', '\t', '\r':
		return "'" + value
	case '-':
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return value
		}
		return "'" + value
	default:
		return value
	}
}
