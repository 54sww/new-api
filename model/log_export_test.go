package model

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestClampLogExportWindow(t *testing.T) {
	now := int64(1_700_000_000)
	earliest := now - int64(LogExportRecentWindow/time.Second)

	tests := []struct {
		name    string
		role    int
		start   int64
		end     int64
		want    LogExportWindow
		wantErr error
	}{
		{
			name:  "admin keeps an unbounded window",
			role:  common.RoleAdminUser,
			start: 0,
			end:   0,
			want:  LogExportWindow{},
		},
		{
			name:  "root keeps a historical window",
			role:  common.RoleRootUser,
			start: earliest - 86400,
			end:   earliest - 1,
			want:  LogExportWindow{Start: earliest - 86400, End: earliest - 1},
		},
		{
			name:  "user empty start is clamped to the recent window",
			role:  common.RoleCommonUser,
			start: 0,
			end:   now,
			want:  LogExportWindow{Start: earliest, End: now, Clamped: true},
		},
		{
			name:  "user start before the window is clamped",
			role:  common.RoleCommonUser,
			start: earliest - 10,
			end:   now,
			want:  LogExportWindow{Start: earliest, End: now, Clamped: true},
		},
		{
			name:    "user range entirely before the window is rejected",
			role:    common.RoleCommonUser,
			start:   earliest - 86400,
			end:     earliest - 1,
			wantErr: ErrLogExportOutsideWindow,
		},
		{
			name:  "user range inside the window is unchanged",
			role:  common.RoleCommonUser,
			start: earliest + 10,
			end:   now,
			want:  LogExportWindow{Start: earliest + 10, End: now},
		},
		{
			name:    "guest is rejected for a historical range",
			role:    common.RoleGuestUser,
			start:   1,
			end:     earliest - 1,
			wantErr: ErrLogExportOutsideWindow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClampLogExportWindow(tt.role, tt.start, tt.end, now)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestListLogsForExportPagesByUserAndTime(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&Log{}))

	previousDB := LOG_DB
	previousGroupCol := logGroupCol
	LOG_DB = database
	logGroupCol = "`group`"
	t.Cleanup(func() {
		LOG_DB = previousDB
		logGroupCol = previousGroupCol
	})

	rows := []*Log{
		{UserId: 1, CreatedAt: 300, Type: LogTypeConsume, ModelName: "new", TokenName: "mine", Content: "newest"},
		{UserId: 1, CreatedAt: 200, Type: LogTypeConsume, ModelName: "mid", TokenName: "mine", Content: "middle"},
		{UserId: 1, CreatedAt: 100, Type: LogTypeConsume, ModelName: "old", TokenName: "mine", Content: "oldest"},
		{UserId: 2, CreatedAt: 250, Type: LogTypeConsume, ModelName: "other", TokenName: "theirs", Content: "other-user"},
	}
	require.NoError(t, database.Create(&rows).Error)

	ctx := context.Background()
	query := LogExportQuery{UserID: 1, StartTimestamp: 150}
	first, err := ListLogsForExport(ctx, query, nil, 1)
	require.NoError(t, err)
	require.Len(t, first, 1)
	assert.Equal(t, "newest", first[0].Content)

	cursor := LogExportCursor{CreatedAt: first[0].CreatedAt, ID: first[0].Id, RequestID: first[0].RequestId}
	second, err := ListLogsForExport(ctx, query, &cursor, 10)
	require.NoError(t, err)
	require.Len(t, second, 1)
	assert.Equal(t, "middle", second[0].Content)
	assert.NotEqual(t, 2, second[0].UserId)
}
