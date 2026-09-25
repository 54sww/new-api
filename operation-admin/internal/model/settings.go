package model

import "time"

// SysSetting is a simple key-value store for runtime configuration.
type SysSetting struct {
	Key       string    `json:"key" gorm:"column:setting_key;primaryKey;size:64"`
	Value     string    `json:"value" gorm:"type:text"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SysSetting) TableName() string { return "sys_settings" }

const (
	SettingBalanceSyncEnabled  = "balance_sync_enabled"
	SettingBalanceSyncInterval = "balance_sync_interval_minutes" // minutes, min 1
)

// BalanceSnapshot records each successful balance fetch for daily consume stats.
// Day consume ≈ start_balance + recharges − end_balance.
type BalanceSnapshot struct {
	ID         uint64  `json:"id" gorm:"primaryKey;autoIncrement"`
	ChannelID  int     `json:"channel_id" gorm:"index:idx_snap_ch_time,priority:1;not null"`
	Balance    float64 `json:"balance"` // USD
	BalanceRaw float64 `json:"balance_raw"`
	Currency   string  `json:"currency" gorm:"size:16"`
	Source     string  `json:"source" gorm:"size:16"` // auto | manual
	SyncedAt   int64   `json:"synced_at" gorm:"index:idx_snap_ch_time,priority:2;index;not null"`
	DayKey     string  `json:"day_key" gorm:"size:10;index"` // YYYY-MM-DD (local TZ)
}

func (BalanceSnapshot) TableName() string { return "balance_snapshots" }
