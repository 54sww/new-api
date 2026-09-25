package model

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/operation-admin/internal/balancequery"
)

// Channel is a local mirror of new-api channels (no API keys stored from new-api sync).
// Balance query credentials live only in BalanceQueryJSON when configured here.
type Channel struct {
	ID                 int     `json:"id" gorm:"primaryKey"`
	Type               int     `json:"type"`
	Status             int     `json:"status"`
	Name               string  `json:"name" gorm:"index;size:255"`
	Group              string  `json:"group" gorm:"column:channel_group;size:64"`
	BaseURL            string  `json:"base_url" gorm:"column:base_url;size:512"` // from new-api channel.base_url
	Balance            float64 `json:"balance"` // upstream balance USD (for reconcile)
	BalanceRaw         float64 `json:"balance_raw"`
	BalanceCurrency    string  `json:"balance_currency" gorm:"size:16"`
	BalanceUpdatedTime int64   `json:"balance_updated_time"`
	BalanceQueryJSON   string  `json:"-" gorm:"column:balance_query;type:text"` // secret config
	UsedQuota          int64   `json:"used_quota"`
	OpeningBalance     float64 `json:"opening_balance"`
	OpeningSet         bool    `json:"opening_set"`
	// SyncBalance: when true, included in scheduled / batch balance refresh (opt-in for prod channels).
	SyncBalance bool `json:"sync_balance"`
	// NotifyEnabled: when true, channel is eligible for balance/diff notifications (test channels stay off).
	NotifyEnabled bool `json:"notify_enabled"`
	// Alias: display name for notifications; empty falls back to Name.
	Alias  string `json:"alias" gorm:"size:128"`
	Remark string `json:"remark" gorm:"size:255"`
	SyncedAt int64 `json:"synced_at"`
}

func (Channel) TableName() string { return "channels" }

// DisplayName returns alias if set, otherwise the synced channel name.
func (ch *Channel) DisplayName() string {
	if ch == nil {
		return ""
	}
	if a := strings.TrimSpace(ch.Alias); a != "" {
		return a
	}
	return ch.Name
}

func (ch *Channel) GetBalanceQuery() balancequery.Config {
	if ch == nil || strings.TrimSpace(ch.BalanceQueryJSON) == "" {
		return balancequery.Config{}
	}
	var cfg balancequery.Config
	_ = json.Unmarshal([]byte(ch.BalanceQueryJSON), &cfg)
	return cfg
}

func (ch *Channel) SetBalanceQuery(cfg balancequery.Config) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	ch.BalanceQueryJSON = string(b)
	return nil
}

func (ch *Channel) HasCustomBalanceQuery() bool {
	cfg := ch.GetBalanceQuery()
	return cfg.Enabled && strings.TrimSpace(cfg.URL) != ""
}

// ResolveBalanceQuery fills empty BaseURL from the synced channel base_url.
func (ch *Channel) ResolveBalanceQuery() balancequery.Config {
	cfg := ch.GetBalanceQuery()
	if strings.TrimSpace(cfg.BaseURL) == "" && ch != nil {
		cfg.BaseURL = strings.TrimRight(strings.TrimSpace(ch.BaseURL), "/")
	}
	return cfg
}

// ConsumeLog stores EXTERNAL channel usage logs only.
// New-api system consume (logs type=2) is read live via NEW_API_LOG_SQL_DSN / NEW_API_SQL_DSN — never copied here.
type ConsumeLog struct {
	ID                uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	SourceID          int    `json:"source_id" gorm:"uniqueIndex;not null"` // external idempotency key
	UserID            int    `json:"user_id" gorm:"index"`
	CreatedAt         int64  `json:"created_at" gorm:"index:idx_channel_created,priority:2;index"`
	Username          string `json:"username" gorm:"size:64"`
	TokenName         string `json:"token_name" gorm:"size:64"`
	ModelName         string `json:"model_name" gorm:"size:128;index"`
	Quota             int    `json:"quota"`
	PromptTokens      int    `json:"prompt_tokens"`
	CompletionTokens  int    `json:"completion_tokens"`
	UseTime           int    `json:"use_time"`
	IsStream          bool   `json:"is_stream"`
	ChannelID         int    `json:"channel_id" gorm:"index:idx_channel_created,priority:1;index"`
	TokenID           int    `json:"token_id"`
	Group             string `json:"group" gorm:"column:token_group;size:64"`
	RequestID         string `json:"request_id" gorm:"size:64;index"`
	UpstreamRequestID string `json:"upstream_request_id" gorm:"size:128"`
	Note              string `json:"note" gorm:"size:512"`
	SyncedAt          int64  `json:"synced_at"`
}

func (ConsumeLog) TableName() string { return "consume_logs" }

// Recharge records upstream top-ups entered by finance.
type Recharge struct {
	ID          uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	ChannelID   int       `json:"channel_id" gorm:"index;not null"`
	AmountUSD   float64   `json:"amount_usd" gorm:"not null"` // positive = top-up
	RechargedAt int64     `json:"recharged_at" gorm:"index;not null"`
	Voucher     string    `json:"voucher" gorm:"size:255"`
	Note        string    `json:"note" gorm:"size:512"`
	CreatedBy   string    `json:"created_by" gorm:"size:64"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Recharge) TableName() string { return "recharges" }

// SyncState tracks incremental sync cursors.
type SyncState struct {
	Name          string `json:"name" gorm:"primaryKey;size:64"`
	LastCreatedAt int64  `json:"last_created_at"`
	LastSourceID  int    `json:"last_source_id"`
	UpdatedAt     int64  `json:"updated_at"`
	Message       string `json:"message" gorm:"size:512"`
}

func (SyncState) TableName() string { return "sync_states" }
