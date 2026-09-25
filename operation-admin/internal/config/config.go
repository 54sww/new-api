package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                string
	SQLDSN              string
	JWTSecret           string
	JWTExpire           time.Duration
	NewAPIBaseURL       string
	NewAPIAccessToken   string
	// NewAPILogSQLDSN: read-only DSN for new-api logs table (prefer LOG_SQL_DSN).
	// Falls back to NewAPISqlDSN when empty.
	NewAPILogSQLDSN     string
	NewAPISqlDSN        string
	QuotaPerUnit        float64
	ChannelSyncEvery    time.Duration
	BalanceRefreshEvery time.Duration
	AdminPassword       string
}

func Load() Config {
	_ = os.Setenv("TZ", getenv("TZ", "Asia/Shanghai"))

	logDSN := getenv("NEW_API_LOG_SQL_DSN", "")
	mainDSN := getenv("NEW_API_SQL_DSN", "")
	if logDSN == "" {
		logDSN = mainDSN
	}

	return Config{
		Port:                getenv("PORT", "3100"),
		SQLDSN:              getenv("SQL_DSN", "finance.db"),
		JWTSecret:           getenv("JWT_SECRET", "operation-admin-change-me"),
		JWTExpire:           durationEnv("JWT_EXPIRE", 12*time.Hour),
		NewAPIBaseURL:       strings.TrimRight(getenv("NEW_API_BASE_URL", "http://127.0.0.1:3000"), "/"),
		NewAPIAccessToken:   getenv("NEW_API_ACCESS_TOKEN", ""),
		NewAPILogSQLDSN:     logDSN,
		NewAPISqlDSN:        mainDSN,
		QuotaPerUnit:        floatEnv("QUOTA_PER_UNIT", 500*1000.0),
		ChannelSyncEvery:    durationEnv("CHANNEL_SYNC_EVERY", 60*time.Second),
		BalanceRefreshEvery: durationEnv("BALANCE_REFRESH_EVERY", 5*time.Minute),
		AdminPassword:       getenv("ADMIN_PASSWORD", "admin123"),
	}
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func floatEnv(key string, def float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return n
}

func durationEnv(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
