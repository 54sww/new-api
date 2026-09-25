package reconcile

import (
	"github.com/QuantumNous/new-api/operation-admin/internal/model"
	"github.com/QuantumNous/new-api/operation-admin/internal/newapidb"
	"gorm.io/gorm"
)

type Row struct {
	ChannelID       int     `json:"channel_id"`
	Name            string  `json:"name"`
	Type            int     `json:"type"`
	Status          int     `json:"status"`
	UpstreamBalance float64 `json:"upstream_balance"`
	BalanceRaw      float64 `json:"balance_raw"`
	BalanceCurrency string  `json:"balance_currency"`
	BalanceSource   string  `json:"balance_source"` // custom | new-api
	BalanceUpdated  int64   `json:"balance_updated_time"`
	HasBalanceQuery bool    `json:"has_balance_query"`
	SyncBalance     bool    `json:"sync_balance"`
	NotifyEnabled   bool    `json:"notify_enabled"`
	OpeningBalance  float64 `json:"opening_balance"`
	RechargeTotal   float64 `json:"recharge_total"`
	ConsumeQuota    int64   `json:"consume_quota"`
	ConsumeUSD      float64 `json:"consume_usd"`
	TheoryBalance   float64 `json:"theory_balance"` // opening + recharge - consume
	DiffUSD         float64 `json:"diff_usd"`       // upstream - theory
	AbsDiffUSD      float64 `json:"abs_diff_usd"`
	SyncedAt        int64   `json:"synced_at"`
}

type Summary struct {
	ChannelCount  int     `json:"channel_count"`
	UpstreamTotal float64 `json:"upstream_total"`
	RechargeTotal float64 `json:"recharge_total"`
	ConsumeUSD    float64 `json:"consume_usd"`
	TheoryTotal   float64 `json:"theory_total"`
	DiffTotal     float64 `json:"diff_total"`
	AbsDiffTotal  float64 `json:"abs_diff_total"`
	QuotaPerUnit  float64 `json:"quota_per_unit"`
	ConsumeSource string  `json:"consume_source"` // new-api-logs
}

type Result struct {
	Summary Summary `json:"summary"`
	Items   []Row   `json:"items"`
}

type Service struct {
	DB           *gorm.DB
	NewAPIDB     *gorm.DB // read-only new-api log/main db
	QuotaPerUnit float64
}

func (s *Service) Live(channelID int) (*Result, error) {
	qpu := s.QuotaPerUnit
	if qpu <= 0 {
		qpu = 500000
	}

	var channels []model.Channel
	q := s.DB.Order("id asc")
	if channelID > 0 {
		q = q.Where("id = ?", channelID)
	}
	if err := q.Find(&channels).Error; err != nil {
		return nil, err
	}

	type agg struct {
		ChannelID int
		Total     float64
	}
	var rechargeAggs []agg
	rq := s.DB.Model(&model.Recharge{}).Select("channel_id, COALESCE(SUM(amount_usd),0) as total").Group("channel_id")
	if channelID > 0 {
		rq = rq.Where("channel_id = ?", channelID)
	}
	if err := rq.Scan(&rechargeAggs).Error; err != nil {
		return nil, err
	}
	rechargeMap := map[int]float64{}
	for _, a := range rechargeAggs {
		rechargeMap[a.ChannelID] = a.Total
	}

	consumeMap, err := newapidb.SumConsumeByChannel(s.NewAPIDB, channelID)
	if err != nil {
		return nil, err
	}

	items := make([]Row, 0, len(channels))
	var sum Summary
	sum.QuotaPerUnit = qpu
	sum.ConsumeSource = "new-api-logs"

	for _, ch := range channels {
		recharge := rechargeMap[ch.ID]
		quota := consumeMap[ch.ID]
		consumeUSD := float64(quota) / qpu
		theory := ch.OpeningBalance + recharge - consumeUSD
		diff := ch.Balance - theory
		if diff < 0 {
			sum.AbsDiffTotal += -diff
		} else {
			sum.AbsDiffTotal += diff
		}
		src := "new-api"
		if ch.HasCustomBalanceQuery() {
			src = "custom"
		}
		row := Row{
			ChannelID:       ch.ID,
			Name:            ch.Name,
			Type:            ch.Type,
			Status:          ch.Status,
			UpstreamBalance: ch.Balance,
			BalanceRaw:      ch.BalanceRaw,
			BalanceCurrency: ch.BalanceCurrency,
			BalanceSource:   src,
			BalanceUpdated:  ch.BalanceUpdatedTime,
			HasBalanceQuery: ch.HasCustomBalanceQuery(),
			SyncBalance:     ch.SyncBalance,
			NotifyEnabled:   ch.NotifyEnabled,
			OpeningBalance:  ch.OpeningBalance,
			RechargeTotal:   recharge,
			ConsumeQuota:    quota,
			ConsumeUSD:      consumeUSD,
			TheoryBalance:   theory,
			DiffUSD:         diff,
			AbsDiffUSD:      abs(diff),
			SyncedAt:        ch.SyncedAt,
		}
		items = append(items, row)
		sum.ChannelCount++
		sum.UpstreamTotal += ch.Balance
		sum.RechargeTotal += recharge
		sum.ConsumeUSD += consumeUSD
		sum.TheoryTotal += theory
		sum.DiffTotal += diff
	}

	return &Result{Summary: sum, Items: items}, nil
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
