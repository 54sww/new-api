package report

import (
	"fmt"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/operation-admin/internal/model"
	"gorm.io/gorm"
)

const (
	StatusOK         = "ok"          // 有日初结转 + 日内有快照作日末
	StatusPartial    = "partial"     // 缺日初或缺日末，消耗仅供参考
	StatusInProgress = "in_progress" // 当天尚未结束，日末为最新快照
	StatusNoData     = "no_data"     // 无可用快照
)

type DayRow struct {
	ChannelID    int     `json:"channel_id"`
	ChannelName  string  `json:"channel_name"`
	SyncBalance  bool    `json:"sync_balance"`
	DayKey       string  `json:"day_key"`
	StartBalance float64 `json:"start_balance"`
	StartAt      int64   `json:"start_at"` // snapshot time used as day-open
	EndBalance   float64 `json:"end_balance"`
	EndAt        int64   `json:"end_at"`
	RechargeUSD  float64 `json:"recharge_usd"`
	ConsumeUSD   float64 `json:"consume_usd"` // start + recharge - end
	SnapshotCnt  int     `json:"snapshot_cnt"` // snapshots inside the day [start,end)
	Status       string  `json:"status"`
	StatusNote   string  `json:"status_note"`
}

type DaySummaryRow struct {
	DayKey         string  `json:"day_key"`
	ChannelCount   int     `json:"channel_count"`
	OKCount        int     `json:"ok_count"`
	PartialCount   int     `json:"partial_count"`
	InProgressCount int    `json:"in_progress_count"`
	ConsumeTotal   float64 `json:"consume_total"`
	RechargeTotal  float64 `json:"recharge_total"`
	StartTotal     float64 `json:"start_total"`
	EndTotal       float64 `json:"end_total"`
}

type DayResult struct {
	From    string          `json:"from"`
	To      string          `json:"to"`
	ByDay   []DaySummaryRow `json:"by_day"`
	Items   []DayRow        `json:"items"`
	Summary struct {
		ChannelCount    int     `json:"channel_count"`
		DayCount        int     `json:"day_count"`
		RowCount        int     `json:"row_count"`
		OKCount         int     `json:"ok_count"`
		PartialCount    int     `json:"partial_count"`
		InProgressCount int     `json:"in_progress_count"`
		ConsumeTotal    float64 `json:"consume_total"`
		RechargeTotal   float64 `json:"recharge_total"`
		StartTotal      float64 `json:"start_total"`
		EndTotal        float64 `json:"end_total"`
	} `json:"summary"`
}

type SnapshotPoint struct {
	ID         uint64  `json:"id"`
	Balance    float64 `json:"balance"`
	BalanceRaw float64 `json:"balance_raw"`
	Currency   string  `json:"currency"`
	Source     string  `json:"source"`
	SyncedAt   int64   `json:"synced_at"`
	DayKey     string  `json:"day_key"`
}

// DailyConsume builds channel×day consume for [from,to] inclusive (local calendar).
//
// 日初余额 = 当天 00:00 之前最近一次快照（结转），不是「当天第一笔」。
// 日末余额 = 次日 00:00 之前最近一次快照。
// 当日消耗 = 日初 + 当日充值 − 日末。
func DailyConsume(db *gorm.DB, fromKey, toKey string, channelID int, onlySync bool) (*DayResult, error) {
	today := time.Now().In(time.Local).Format("2006-01-02")
	if toKey == "" {
		toKey = today
	}
	if fromKey == "" {
		fromKey = toKey
	}
	fromDay, err := time.ParseInLocation("2006-01-02", fromKey, time.Local)
	if err != nil {
		return nil, fmt.Errorf("from 日期无效: %w", err)
	}
	toDay, err := time.ParseInLocation("2006-01-02", toKey, time.Local)
	if err != nil {
		return nil, fmt.Errorf("to 日期无效: %w", err)
	}
	if toDay.Before(fromDay) {
		fromDay, toDay = toDay, fromDay
		fromKey, toKey = toKey, fromKey
	}
	// Cap range to avoid huge scans
	if toDay.Sub(fromDay) > 90*24*time.Hour {
		return nil, fmt.Errorf("查询区间最长 90 天")
	}

	var channels []model.Channel
	cq := db.Order("id asc")
	if channelID > 0 {
		cq = cq.Where("id = ?", channelID)
	} else if onlySync {
		cq = cq.Where("sync_balance = ?", true)
	}
	if err := cq.Find(&channels).Error; err != nil {
		return nil, err
	}
	if len(channels) == 0 {
		out := &DayResult{From: fromKey, To: toKey, ByDay: []DaySummaryRow{}, Items: []DayRow{}}
		return out, nil
	}

	ids := make([]int, len(channels))
	nameByID := map[int]string{}
	syncByID := map[int]bool{}
	for i, ch := range channels {
		ids[i] = ch.ID
		nameByID[ch.ID] = ch.Name
		syncByID[ch.ID] = ch.SyncBalance
	}

	rangeStart := fromDay.Unix()
	rangeEnd := toDay.Add(24 * time.Hour).Unix()

	// Opening carry: last snapshot before range for each channel.
	type chMax struct {
		ChannelID int   `gorm:"column:channel_id"`
		Mx        int64 `gorm:"column:mx"`
	}
	var maxima []chMax
	if err := db.Model(&model.BalanceSnapshot{}).
		Select("channel_id, MAX(synced_at) as mx").
		Where("channel_id IN ? AND synced_at < ?", ids, rangeStart).
		Group("channel_id").Scan(&maxima).Error; err != nil {
		return nil, err
	}
	snapsByCh := map[int][]model.BalanceSnapshot{}
	for _, m := range maxima {
		var s model.BalanceSnapshot
		if err := db.Where("channel_id = ? AND synced_at = ?", m.ChannelID, m.Mx).
			Order("id desc").First(&s).Error; err == nil {
			snapsByCh[m.ChannelID] = append(snapsByCh[m.ChannelID], s)
		}
	}

	// All snapshots inside the query window.
	var inRange []model.BalanceSnapshot
	if err := db.Where("channel_id IN ? AND synced_at >= ? AND synced_at < ?", ids, rangeStart, rangeEnd).
		Order("channel_id asc, synced_at asc").
		Find(&inRange).Error; err != nil {
		return nil, err
	}
	for _, s := range inRange {
		snapsByCh[s.ChannelID] = append(snapsByCh[s.ChannelID], s)
	}

	var recharges []model.Recharge
	if err := db.Where("channel_id IN ? AND recharged_at >= ? AND recharged_at < ?", ids, rangeStart, rangeEnd).
		Find(&recharges).Error; err != nil {
		return nil, err
	}

	out := &DayResult{
		From:  fromKey,
		To:    toKey,
		Items: make([]DayRow, 0),
		ByDay: make([]DaySummaryRow, 0),
	}
	byDayMap := map[string]*DaySummaryRow{}
	channelSeen := map[int]struct{}{}

	for d := fromDay; !d.After(toDay); d = d.Add(24 * time.Hour) {
		dayKey := d.Format("2006-01-02")
		dayStart := d.Unix()
		dayEnd := d.Add(24 * time.Hour).Unix()
		isToday := dayKey == today

		daySum := &DaySummaryRow{DayKey: dayKey}
		byDayMap[dayKey] = daySum

		for _, chID := range ids {
			row := buildDayRow(chID, nameByID[chID], syncByID[chID], dayKey, dayStart, dayEnd, isToday,
				snapsByCh[chID], recharges)
			if row.Status == StatusNoData {
				continue
			}
			out.Items = append(out.Items, row)
			channelSeen[chID] = struct{}{}

			daySum.ChannelCount++
			daySum.ConsumeTotal += row.ConsumeUSD
			daySum.RechargeTotal += row.RechargeUSD
			daySum.StartTotal += row.StartBalance
			daySum.EndTotal += row.EndBalance
			switch row.Status {
			case StatusOK:
				daySum.OKCount++
			case StatusPartial:
				daySum.PartialCount++
			case StatusInProgress:
				daySum.InProgressCount++
			}

			out.Summary.RowCount++
			out.Summary.ConsumeTotal += row.ConsumeUSD
			out.Summary.RechargeTotal += row.RechargeUSD
			out.Summary.StartTotal += row.StartBalance
			out.Summary.EndTotal += row.EndBalance
			switch row.Status {
			case StatusOK:
				out.Summary.OKCount++
			case StatusPartial:
				out.Summary.PartialCount++
			case StatusInProgress:
				out.Summary.InProgressCount++
			}
		}
	}

	for d := fromDay; !d.After(toDay); d = d.Add(24 * time.Hour) {
		key := d.Format("2006-01-02")
		if s := byDayMap[key]; s != nil {
			out.ByDay = append(out.ByDay, *s)
		}
	}
	out.Summary.ChannelCount = len(channelSeen)
	out.Summary.DayCount = len(out.ByDay)

	// Prefer high consume first within same day for readability
	sort.SliceStable(out.Items, func(i, j int) bool {
		if out.Items[i].DayKey != out.Items[j].DayKey {
			return out.Items[i].DayKey > out.Items[j].DayKey
		}
		return out.Items[i].ConsumeUSD > out.Items[j].ConsumeUSD
	})

	return out, nil
}

func buildDayRow(
	chID int, name string, syncBal bool,
	dayKey string, dayStart, dayEnd int64, isToday bool,
	snaps []model.BalanceSnapshot, recharges []model.Recharge,
) DayRow {
	row := DayRow{
		ChannelID:   chID,
		ChannelName: name,
		SyncBalance: syncBal,
		DayKey:      dayKey,
		Status:      StatusNoData,
	}

	var startSnap, endSnap *model.BalanceSnapshot
	inDay := 0
	for i := range snaps {
		s := &snaps[i]
		if s.SyncedAt < dayStart {
			startSnap = s
		}
		if s.SyncedAt >= dayStart && s.SyncedAt < dayEnd {
			inDay++
			endSnap = s // last within day
		}
		// If no in-day snap, end can still be carry-forward start (no change) — handled below
	}
	row.SnapshotCnt = inDay

	if startSnap == nil && endSnap == nil {
		return row
	}

	// Opening: prefer last snap before day; if missing, use first in-day as approximate
	hasRealStart := startSnap != nil
	if startSnap != nil {
		row.StartBalance = startSnap.Balance
		row.StartAt = startSnap.SyncedAt
	} else if endSnap != nil {
		// approximate: first in-day as start (already lost morning consume)
		first := firstInDay(snaps, dayStart, dayEnd)
		row.StartBalance = first.Balance
		row.StartAt = first.SyncedAt
	}

	if endSnap != nil {
		row.EndBalance = endSnap.Balance
		row.EndAt = endSnap.SyncedAt
	} else if startSnap != nil {
		// no sync that day — balance assumed unchanged from open
		row.EndBalance = startSnap.Balance
		row.EndAt = startSnap.SyncedAt
	}

	var recharge float64
	for _, r := range recharges {
		if r.ChannelID == chID && r.RechargedAt >= dayStart && r.RechargedAt < dayEnd {
			recharge += r.AmountUSD
		}
	}
	row.RechargeUSD = recharge
	row.ConsumeUSD = row.StartBalance + recharge - row.EndBalance

	switch {
	case hasRealStart && inDay > 0 && !isToday:
		row.Status = StatusOK
		row.StatusNote = "日初结转 + 日内快照"
	case hasRealStart && inDay > 0 && isToday:
		row.Status = StatusInProgress
		row.StatusNote = "当天进行中，日末为最新快照"
	case hasRealStart && inDay == 0:
		row.Status = StatusPartial
		row.StatusNote = "当日无快照，按余额未变估算（消耗≈充值）"
	case !hasRealStart && inDay > 0:
		row.Status = StatusPartial
		row.StatusNote = "缺日初结转，用当天首笔快照近似"
	default:
		row.Status = StatusPartial
		row.StatusNote = "数据不完整"
	}
	return row
}

func firstInDay(snaps []model.BalanceSnapshot, dayStart, dayEnd int64) model.BalanceSnapshot {
	for _, s := range snaps {
		if s.SyncedAt >= dayStart && s.SyncedAt < dayEnd {
			return s
		}
	}
	return model.BalanceSnapshot{}
}

// ListDaySnapshots returns snapshots for one channel on one local day (plus the opening carry snap).
func ListDaySnapshots(db *gorm.DB, channelID int, dayKey string) ([]SnapshotPoint, error) {
	if channelID <= 0 {
		return nil, fmt.Errorf("channel_id 必填")
	}
	if dayKey == "" {
		dayKey = time.Now().In(time.Local).Format("2006-01-02")
	}
	dayStart, err := time.ParseInLocation("2006-01-02", dayKey, time.Local)
	if err != nil {
		return nil, err
	}
	dayEnd := dayStart.Add(24 * time.Hour)
	startUnix := dayStart.Unix()
	endUnix := dayEnd.Unix()

	out := make([]SnapshotPoint, 0)
	var open model.BalanceSnapshot
	if err := db.Where("channel_id = ? AND synced_at < ?", channelID, startUnix).
		Order("synced_at desc").First(&open).Error; err == nil {
		out = append(out, SnapshotPoint{
			ID: open.ID, Balance: open.Balance, BalanceRaw: open.BalanceRaw,
			Currency: open.Currency, Source: open.Source + "/open",
			SyncedAt: open.SyncedAt, DayKey: open.DayKey,
		})
	}
	var daySnaps []model.BalanceSnapshot
	if err := db.Where("channel_id = ? AND synced_at >= ? AND synced_at < ?", channelID, startUnix, endUnix).
		Order("synced_at asc").Find(&daySnaps).Error; err != nil {
		return nil, err
	}
	for _, s := range daySnaps {
		out = append(out, SnapshotPoint{
			ID: s.ID, Balance: s.Balance, BalanceRaw: s.BalanceRaw,
			Currency: s.Currency, Source: s.Source,
			SyncedAt: s.SyncedAt, DayKey: s.DayKey,
		})
	}
	return out, nil
}
