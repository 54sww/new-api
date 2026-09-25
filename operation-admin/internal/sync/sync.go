package syncer

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/operation-admin/internal/balancequery"
	"github.com/QuantumNous/new-api/operation-admin/internal/config"
	"github.com/QuantumNous/new-api/operation-admin/internal/model"
	"github.com/QuantumNous/new-api/operation-admin/internal/newapi"
	"github.com/QuantumNous/new-api/operation-admin/internal/notify"
	"github.com/QuantumNous/new-api/operation-admin/internal/settings"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	DB     *gorm.DB
	Client *newapi.Client
	Cfg    config.Config
	Notify *notify.Service

	balMu       sync.Mutex
	balReloadCh chan struct{}
}

func (s *Service) Start() {
	settings.SeedDefaults(s.DB, s.Cfg.BalanceRefreshEvery > 0, int(s.Cfg.BalanceRefreshEvery.Minutes()))
	if s.balReloadCh == nil {
		s.balReloadCh = make(chan struct{}, 1)
	}
	go s.loop("channels", s.Cfg.ChannelSyncEvery, s.SyncChannels)
	go s.balanceLoop()
}

// NotifyBalanceConfigChanged wakes the balance loop so interval/enabled take effect immediately.
func (s *Service) NotifyBalanceConfigChanged() {
	if s.balReloadCh == nil {
		return
	}
	select {
	case s.balReloadCh <- struct{}{}:
	default:
	}
}

func (s *Service) loop(name string, every time.Duration, fn func() error) {
	if every <= 0 {
		return
	}
	time.Sleep(2 * time.Second)
	for {
		if err := fn(); err != nil {
			log.Printf("[sync:%s] %v", name, err)
			s.setState(name+"_error", 0, 0, err.Error())
		}
		time.Sleep(every)
	}
}

func (s *Service) balanceLoop() {
	time.Sleep(3 * time.Second)
	for {
		enabled, minutes := settings.BalanceSyncConfig(s.DB, true, 5)
		if !enabled {
			s.setState("balances", time.Now().Unix(), 0, "disabled")
			select {
			case <-s.balReloadCh:
				continue
			case <-time.After(30 * time.Second):
				continue
			}
		}
		if err := s.RefreshBalances("auto"); err != nil {
			log.Printf("[sync:balances] %v", err)
			s.setState("balances_error", 0, 0, err.Error())
		}
		wait := time.Duration(minutes) * time.Minute
		timer := time.NewTimer(wait)
		select {
		case <-s.balReloadCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
		}
	}
}

func (s *Service) SyncChannels() error {
	if s.Client.AccessToken == "" {
		return nil
	}
	page := 1
	pageSize := 100
	now := time.Now().Unix()
	totalSynced := 0
	for {
		items, total, err := s.Client.ListChannels(page, pageSize)
		if err != nil {
			return err
		}
		for _, ch := range items {
			remark := ""
			if ch.Remark != nil {
				remark = *ch.Remark
			}
			baseURL := ""
			if ch.BaseURL != nil {
				baseURL = strings.TrimRight(strings.TrimSpace(*ch.BaseURL), "/")
			}
			row := model.Channel{
				ID:                 ch.ID,
				Type:               ch.Type,
				Status:             ch.Status,
				Name:               ch.Name,
				Group:              ch.Group,
				BaseURL:            baseURL,
				Balance:            ch.Balance,
				BalanceUpdatedTime: ch.BalanceUpdatedTime,
				UsedQuota:          ch.UsedQuota,
				Remark:             remark,
				SyncedAt:           now,
			}
			// Do NOT overwrite balance_query / finance-owned balance fields on conflict.
			err := s.DB.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"type", "status", "name", "channel_group", "base_url",
					"used_quota", "remark", "synced_at",
				}),
			}).Create(&row).Error
			if err != nil {
				return err
			}
			_ = s.DB.Model(&model.Channel{}).
				Where("id = ? AND opening_set = ?", ch.ID, false).
				Updates(map[string]interface{}{
					"opening_balance": ch.Balance,
					"opening_set":     true,
				}).Error
			totalSynced++
		}
		if page*pageSize >= total || len(items) == 0 {
			break
		}
		page++
	}
	s.setState("channels", now, totalSynced, "ok")
	return nil
}

func (s *Service) RefreshBalances(source string) error {
	s.balMu.Lock()
	defer s.balMu.Unlock()
	if source == "" {
		source = "auto"
	}
	var channels []model.Channel
	if err := s.DB.Where("status = ? AND sync_balance = ?", 1, true).Find(&channels).Error; err != nil {
		return err
	}
	now := time.Now().Unix()
	ok := 0
	for i := range channels {
		if err := s.refreshOneBalance(&channels[i], source); err != nil {
			log.Printf("[sync:balance] channel %d: %v", channels[i].ID, err)
			continue
		}
		ok++
		time.Sleep(200 * time.Millisecond)
	}
	s.setState("balances", now, ok, "ok")
	return nil
}

// RefreshOneBalance refreshes a single channel (manual) and writes a snapshot.
func (s *Service) RefreshOneBalance(ch *model.Channel) error {
	return s.refreshOneBalance(ch, "manual")
}

func (s *Service) refreshOneBalance(ch *model.Channel, source string) error {
	now := time.Now().Unix()
	var err error
	if ch.HasCustomBalanceQuery() {
		var res *balancequery.Result
		res, err = balancequery.Fetch(ch.ResolveBalanceQuery())
		if err != nil {
			return err
		}
		err = s.DB.Model(&model.Channel{}).Where("id = ?", ch.ID).Updates(map[string]interface{}{
			"balance":              res.AmountUSD,
			"balance_raw":          res.AmountRaw,
			"balance_currency":     res.Currency,
			"balance_updated_time": now,
			"synced_at":            now,
		}).Error
		if err != nil {
			return err
		}
		ch.Balance = res.AmountUSD
		ch.BalanceRaw = res.AmountRaw
		ch.BalanceCurrency = res.Currency
	} else {
		if s.Client.AccessToken == "" {
			return nil
		}
		var bal float64
		bal, err = s.Client.UpdateChannelBalance(ch.ID)
		if err != nil {
			return err
		}
		err = s.DB.Model(&model.Channel{}).Where("id = ?", ch.ID).Updates(map[string]interface{}{
			"balance":              bal,
			"balance_raw":          bal,
			"balance_currency":     "USD",
			"balance_updated_time": now,
			"synced_at":            now,
		}).Error
		if err != nil {
			return err
		}
		ch.Balance = bal
		ch.BalanceRaw = bal
		ch.BalanceCurrency = "USD"
	}

	snap := model.BalanceSnapshot{
		ChannelID:  ch.ID,
		Balance:    ch.Balance,
		BalanceRaw: ch.BalanceRaw,
		Currency:   ch.BalanceCurrency,
		Source:     source,
		SyncedAt:   now,
		DayKey:     time.Unix(now, 0).In(time.Local).Format("2006-01-02"),
	}
	if err := s.DB.Create(&snap).Error; err != nil {
		log.Printf("[sync:balance] snapshot channel %d: %v", ch.ID, err)
	}

	// Reload notify_enabled / name / alias in case caller only had partial row.
	_ = s.DB.Select("id", "name", "alias", "balance", "notify_enabled").First(ch, ch.ID).Error
	if s.Notify != nil {
		s.Notify.CheckBalanceAfterRefresh(ch)
	}
	return nil
}

func (s *Service) setState(name string, lastCreated int64, lastSource int, msg string) {
	st := model.SyncState{
		Name:          name,
		LastCreatedAt: lastCreated,
		LastSourceID:  lastSource,
		UpdatedAt:     time.Now().Unix(),
		Message:       msg,
	}
	_ = s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_created_at", "last_source_id", "updated_at", "message"}),
	}).Create(&st).Error
}
