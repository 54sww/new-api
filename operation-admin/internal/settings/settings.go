package settings

import (
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/operation-admin/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Get(db *gorm.DB, key, def string) string {
	var row model.SysSetting
	if err := db.Where("setting_key = ?", key).First(&row).Error; err != nil {
		return def
	}
	if strings.TrimSpace(row.Value) == "" {
		return def
	}
	return row.Value
}

func Set(db *gorm.DB, key, value string) error {
	row := model.SysSetting{Key: key, Value: value, UpdatedAt: time.Now()}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "setting_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&row).Error
}

func GetBool(db *gorm.DB, key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(Get(db, key, "")))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func GetInt(db *gorm.DB, key string, def int) int {
	v := strings.TrimSpace(Get(db, key, ""))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func SetBool(db *gorm.DB, key string, v bool) error {
	if v {
		return Set(db, key, "true")
	}
	return Set(db, key, "false")
}

func SetInt(db *gorm.DB, key string, v int) error {
	return Set(db, key, strconv.Itoa(v))
}

// SeedDefaults writes initial balance sync settings if missing.
func SeedDefaults(db *gorm.DB, enabled bool, intervalMinutes int) {
	var n int64
	db.Model(&model.SysSetting{}).Where("setting_key = ?", model.SettingBalanceSyncEnabled).Count(&n)
	if n == 0 {
		_ = SetBool(db, model.SettingBalanceSyncEnabled, enabled)
	}
	db.Model(&model.SysSetting{}).Where("setting_key = ?", model.SettingBalanceSyncInterval).Count(&n)
	if n == 0 {
		if intervalMinutes < 1 {
			intervalMinutes = 5
		}
		_ = SetInt(db, model.SettingBalanceSyncInterval, intervalMinutes)
	}
}

// BalanceSyncConfig reads enabled + interval from DB.
func BalanceSyncConfig(db *gorm.DB, defaultEnabled bool, defaultMinutes int) (enabled bool, minutes int) {
	enabled = GetBool(db, model.SettingBalanceSyncEnabled, defaultEnabled)
	minutes = GetInt(db, model.SettingBalanceSyncInterval, defaultMinutes)
	if minutes < 1 {
		minutes = 1
	}
	if minutes > 1440 {
		minutes = 1440
	}
	return enabled, minutes
}
