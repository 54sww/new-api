package model

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	NotifyTypeEmail    = "email"
	NotifyTypeDingTalk = "dingtalk"
)

// NotifyChannel is a delivery endpoint (SMTP / DingTalk webhook, etc.).
type NotifyChannel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:64;not null"`
	Type      string    `json:"type" gorm:"size:32;index;not null"` // email | dingtalk
	Status    string    `json:"status" gorm:"size:16;index"`        // active | inactive
	Remark    string    `json:"remark" gorm:"size:255"`
	ConfigJSON string   `json:"-" gorm:"column:config;type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (NotifyChannel) TableName() string { return "notify_channels" }

func (c *NotifyChannel) GetConfigMap() map[string]interface{} {
	out := map[string]interface{}{}
	if c == nil || strings.TrimSpace(c.ConfigJSON) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(c.ConfigJSON), &out)
	return out
}

func (c *NotifyChannel) SetConfigMap(m map[string]interface{}) error {
	if m == nil {
		m = map[string]interface{}{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	c.ConfigJSON = string(b)
	return nil
}

func (c *NotifyChannel) MaskedConfig() map[string]interface{} {
	m := c.GetConfigMap()
	for _, k := range []string{"password", "secret", "webhook"} {
		if v, ok := m[k].(string); ok && v != "" {
			if k == "webhook" && len(v) > 12 {
				m[k] = v[:8] + "****" + v[len(v)-4:]
			} else {
				m[k] = "********"
			}
		}
	}
	return m
}

// NotifyGroup bundles one or more notify channels for reuse (e.g. finance alerts).
type NotifyGroup struct {
	ID        uint            `json:"id" gorm:"primaryKey"`
	Name      string          `json:"name" gorm:"size:64;not null"`
	Code      string          `json:"code" gorm:"uniqueIndex;size:64"`
	Status    string          `json:"status" gorm:"size:16;index"`
	Remark    string          `json:"remark" gorm:"size:255"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Channels  []NotifyChannel `json:"channels,omitempty" gorm:"many2many:notify_group_channels;"`
}

func (NotifyGroup) TableName() string { return "notify_groups" }

const (
	NotifyEventBalanceBelow = "balance_below"
	NotifyScopeAllEnabled   = "all_enabled" // channels with notify_enabled=true
	NotifyScopeSelected     = "selected"
)

// NotifyRule defines when to fire notifications (e.g. remaining balance below threshold).
type NotifyRule struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	Name            string    `json:"name" gorm:"size:64;not null"`
	Status          string    `json:"status" gorm:"size:16;index"` // active | inactive
	Remark          string    `json:"remark" gorm:"size:255"`
	EventType       string    `json:"event_type" gorm:"size:32;index"` // balance_below
	ThresholdUSD    float64   `json:"threshold_usd"`
	NotifyGroupID   uint      `json:"notify_group_id" gorm:"index"`
	Scope           string    `json:"scope" gorm:"size:32"` // all_enabled | selected
	ChannelIDsJSON  string    `json:"-" gorm:"column:channel_ids;type:text"`
	CooldownMinutes  int       `json:"cooldown_minutes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	NotifyGroup     *NotifyGroup `json:"notify_group,omitempty" gorm:"foreignKey:NotifyGroupID"`
}

func (NotifyRule) TableName() string { return "notify_rules" }

func (r *NotifyRule) GetChannelIDs() []int {
	if r == nil || strings.TrimSpace(r.ChannelIDsJSON) == "" {
		return nil
	}
	var ids []int
	_ = json.Unmarshal([]byte(r.ChannelIDsJSON), &ids)
	return ids
}

func (r *NotifyRule) SetChannelIDs(ids []int) error {
	if ids == nil {
		ids = []int{}
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	r.ChannelIDsJSON = string(b)
	return nil
}

// NotifyRuleState tracks last fire time per rule+channel for cooldown.
type NotifyRuleState struct {
	ID        uint  `json:"id" gorm:"primaryKey"`
	RuleID    uint  `json:"rule_id" gorm:"uniqueIndex:idx_rule_channel;not null"`
	ChannelID int   `json:"channel_id" gorm:"uniqueIndex:idx_rule_channel;not null"`
	LastFired int64 `json:"last_fired"`
	LastBalance float64 `json:"last_balance"`
}

func (NotifyRuleState) TableName() string { return "notify_rule_states" }
