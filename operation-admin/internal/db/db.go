package db

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/operation-admin/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(dsn string) (*gorm.DB, error) {
	gdb, err := openRaw(dsn)
	if err != nil {
		return nil, err
	}
	if err := gdb.AutoMigrate(
		&model.Channel{},
		&model.ConsumeLog{},
		&model.Recharge{},
		&model.SyncState{},
		&model.SysSetting{},
		&model.BalanceSnapshot{},
		&model.SysUser{},
		&model.SysRole{},
		&model.SysMenu{},
		&model.NotifyChannel{},
		&model.NotifyGroup{},
		&model.NotifyRule{},
		&model.NotifyRuleState{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	// One-shot: existing finance channels keep syncing; newly synced channels stay opt-in (false).
	var marker model.SyncState
	res := gdb.Where("name = ?", "channel_sync_balance_default").FirstOrCreate(&marker, model.SyncState{
		Name:      "channel_sync_balance_default",
		UpdatedAt: 1,
		Message:   "pending",
	})
	if res.Error == nil && marker.Message == "pending" {
		_ = gdb.Model(&model.Channel{}).Where("opening_set = ?", true).Update("sync_balance", true).Error
		_ = gdb.Model(&marker).Update("message", "ok").Error
	}
	return gdb, nil
}

// OpenReadOnly connects without running AutoMigrate (for new-api / log DB).
func OpenReadOnly(dsn string) (*gorm.DB, error) {
	return openRaw(dsn)
}

func openRaw(dsn string) (*gorm.DB, error) {
	dialector, err := chooseDialector(dsn)
	if err != nil {
		return nil, err
	}
	return gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
}

func chooseDialector(dsn string) (gorm.Dialector, error) {
	lower := strings.ToLower(dsn)
	switch {
	case strings.HasPrefix(lower, "postgres://"), strings.HasPrefix(lower, "postgresql://"):
		return postgres.Open(dsn), nil
	case strings.Contains(lower, "@tcp(") || strings.HasPrefix(lower, "mysql://"):
		clean := strings.TrimPrefix(dsn, "mysql://")
		return mysql.Open(clean), nil
	case strings.HasSuffix(lower, ".db"), strings.Contains(lower, "sqlite"), !strings.Contains(dsn, "://"):
		path := dsn
		if strings.HasPrefix(lower, "sqlite://") {
			path = strings.TrimPrefix(dsn, "sqlite://")
		}
		return sqlite.Open(path), nil
	default:
		return nil, fmt.Errorf("unsupported SQL_DSN (use postgres://, mysql user:pass@tcp(...)/db, or *.db for sqlite): %s", dsn)
	}
}
