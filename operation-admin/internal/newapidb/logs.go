package newapidb

import (
	"fmt"

	"github.com/QuantumNous/new-api/operation-admin/internal/db"
	"gorm.io/gorm"
)

const LogTypeConsume = 2

// Open opens a read-oriented connection to new-api main or log database.
// Prefer pointing this at LOG_SQL_DSN when new-api stores logs separately.
func Open(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("NEW_API_SQL_DSN / NEW_API_LOG_SQL_DSN is empty")
	}
	return db.OpenReadOnly(dsn)
}

type ConsumeAgg struct {
	ChannelID int   `gorm:"column:channel_id"`
	Total     int64 `gorm:"column:total"`
}

// SumConsumeByChannel aggregates new-api logs (type=consume) by channel_id.
func SumConsumeByChannel(gdb *gorm.DB, channelID int) (map[int]int64, error) {
	if gdb == nil {
		return nil, fmt.Errorf("new-api log db is not configured")
	}
	q := gdb.Table("logs").
		Select("channel_id as channel_id, COALESCE(SUM(quota),0) as total").
		Where("type = ?", LogTypeConsume)
	if channelID > 0 {
		q = q.Where("channel_id = ?", channelID)
	}
	var rows []ConsumeAgg
	if err := q.Group("channel_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[int]int64, len(rows))
	for _, r := range rows {
		out[r.ChannelID] = r.Total
	}
	return out, nil
}

// LogRow is a slim projection of new-api logs for browse APIs.
type LogRow struct {
	ID               int    `json:"id" gorm:"column:id"`
	UserID           int    `json:"user_id" gorm:"column:user_id"`
	CreatedAt        int64  `json:"created_at" gorm:"column:created_at"`
	Username         string `json:"username" gorm:"column:username"`
	TokenName        string `json:"token_name" gorm:"column:token_name"`
	ModelName        string `json:"model_name" gorm:"column:model_name"`
	Quota            int    `json:"quota" gorm:"column:quota"`
	PromptTokens     int    `json:"prompt_tokens" gorm:"column:prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens" gorm:"column:completion_tokens"`
	ChannelID        int    `json:"channel_id" gorm:"column:channel_id"`
	RequestID        string `json:"request_id" gorm:"column:request_id"`
}

func ListConsumeLogs(gdb *gorm.DB, channelID, page, pageSize int) (items []LogRow, total int64, err error) {
	if gdb == nil {
		return nil, 0, fmt.Errorf("new-api log db is not configured")
	}
	q := gdb.Table("logs").Where("type = ?", LogTypeConsume)
	if channelID > 0 {
		q = q.Where("channel_id = ?", channelID)
	}
	if err = q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err = q.Select("id, user_id, created_at, username, token_name, model_name, quota, prompt_tokens, completion_tokens, channel_id, request_id").
		Order("created_at desc, id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&items).Error
	return items, total, err
}
