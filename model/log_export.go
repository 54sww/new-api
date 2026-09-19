package model

import (
	"context"
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"

	"gorm.io/gorm"
)

// LogExportRecentWindow is the maximum age of logs a non-admin may export.
const LogExportRecentWindow = 30 * 24 * time.Hour

// ErrLogExportOutsideWindow is returned when a non-admin export range ends
// entirely before the recent window.
var ErrLogExportOutsideWindow = errors.New("log export outside allowed window")

// LogExportWindow is the created_at range applied to an export query.
// Start or End of 0 means that side is unbounded. Clamped is true when a
// non-admin request had its start pulled forward to the recent window.
type LogExportWindow struct {
	Start   int64
	End     int64
	Clamped bool
}

// ClampLogExportWindow keeps admin and root exports unchanged. Everyone else
// can only export logs created within the last 30 days. A range that ends
// before that window is rejected; a range that starts before it is clamped.
func ClampLogExportWindow(role int, start, end, now int64) (LogExportWindow, error) {
	if role >= common.RoleAdminUser {
		return LogExportWindow{Start: start, End: end}, nil
	}

	earliest := now - int64(LogExportRecentWindow/time.Second)
	if end != 0 && end < earliest {
		return LogExportWindow{}, ErrLogExportOutsideWindow
	}

	window := LogExportWindow{Start: start, End: end}
	if start == 0 || start < earliest {
		window.Start = earliest
		window.Clamped = true
	}
	if window.End != 0 && window.Start > window.End {
		return LogExportWindow{}, ErrLogExportOutsideWindow
	}
	return window, nil
}

// LogExportQuery selects common logs for CSV export. UserID > 0 restricts the
// query to that user and ignores Username and Channel, so a self export cannot
// widen its scope through query parameters.
type LogExportQuery struct {
	UserID            int
	LogType           int
	StartTimestamp    int64
	EndTimestamp      int64
	ModelName         string
	Username          string
	TokenName         string
	Channel           int
	Group             string
	RequestID         string
	UpstreamRequestID string
}

// LogExportCursor is the keyset position of the last row already exported.
type LogExportCursor struct {
	CreatedAt int64
	ID        int
	RequestID string
}

func (cursor LogExportCursor) Same(other LogExportCursor) bool {
	return cursor.CreatedAt == other.CreatedAt && cursor.ID == other.ID && cursor.RequestID == other.RequestID
}

const logExportPageLimit = 1000

// ListLogsForExport returns one page of logs, newest first. Pass a nil cursor
// for the first page. The caller advances the cursor from the last returned row.
func ListLogsForExport(ctx context.Context, query LogExportQuery, cursor *LogExportCursor, limit int) ([]*Log, error) {
	if limit <= 0 || limit > logExportPageLimit {
		limit = logExportPageLimit
	}

	tx, err := applyLogExportFilters(LOG_DB.WithContext(ctx).Model(&Log{}), query)
	if err != nil {
		return nil, err
	}
	tx = applyLogExportCursor(tx, cursor)

	order := "logs.created_at desc, logs.id desc"
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		order = clickHouseLogOrder("logs.")
	}

	var logs []*Log
	if err = tx.Order(order).Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func applyLogExportFilters(tx *gorm.DB, query LogExportQuery) (*gorm.DB, error) {
	var err error
	if query.UserID != 0 {
		tx = tx.Where("logs.user_id = ?", query.UserID)
	}
	if query.LogType != LogTypeUnknown {
		tx = tx.Where("logs.type = ?", query.LogType)
	}
	if tx, err = applyExplicitLogTextFilter(tx, "logs.model_name", query.ModelName); err != nil {
		return nil, err
	}
	if query.UserID == 0 {
		if tx, err = applyExplicitLogTextFilter(tx, "logs.username", query.Username); err != nil {
			return nil, err
		}
		if query.Channel != 0 {
			tx = tx.Where("logs.channel_id = ?", query.Channel)
		}
	}
	if query.TokenName != "" {
		tx = tx.Where("logs.token_name = ?", query.TokenName)
	}
	if query.RequestID != "" {
		tx = tx.Where("logs.request_id = ?", query.RequestID)
	}
	if query.UpstreamRequestID != "" {
		tx = tx.Where("logs.upstream_request_id = ?", query.UpstreamRequestID)
	}
	if query.StartTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", query.StartTimestamp)
	}
	if query.EndTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", query.EndTimestamp)
	}
	if query.Group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", query.Group)
	}
	return tx, nil
}

func applyLogExportCursor(tx *gorm.DB, cursor *LogExportCursor) *gorm.DB {
	if cursor == nil {
		return tx
	}
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		return tx.Where(
			"(logs.created_at < ?) OR (logs.created_at = ? AND logs.request_id < ?)",
			cursor.CreatedAt, cursor.CreatedAt, cursor.RequestID,
		)
	}
	return tx.Where(
		"(logs.created_at < ?) OR (logs.created_at = ? AND logs.id < ?)",
		cursor.CreatedAt, cursor.CreatedAt, cursor.ID,
	)
}

// PopulateLogChannelNames fills ChannelName the same way the log list does.
func PopulateLogChannelNames(logs []*Log) error {
	channelIds := types.NewSet[int]()
	for _, log := range logs {
		if log.ChannelId != 0 {
			channelIds.Add(log.ChannelId)
		}
	}
	if channelIds.Len() == 0 {
		return nil
	}

	var channels []struct {
		Id   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if common.MemoryCacheEnabled {
		for _, channelId := range channelIds.Items() {
			cacheChannel, err := CacheGetChannel(channelId)
			if err != nil {
				continue
			}
			channels = append(channels, struct {
				Id   int    `gorm:"column:id"`
				Name string `gorm:"column:name"`
			}{
				Id:   channelId,
				Name: cacheChannel.Name,
			})
		}
	} else if err := DB.Table("channels").Select("id, name").Where("id IN ?", channelIds.Items()).Find(&channels).Error; err != nil {
		return err
	}

	channelMap := make(map[int]string, len(channels))
	for _, channel := range channels {
		channelMap[channel.Id] = channel.Name
	}
	for i := range logs {
		logs[i].ChannelName = channelMap[logs[i].ChannelId]
	}
	return nil
}
