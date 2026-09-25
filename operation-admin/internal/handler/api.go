package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/operation-admin/internal/auth"
	"github.com/QuantumNous/new-api/operation-admin/internal/balancequery"
	"github.com/QuantumNous/new-api/operation-admin/internal/model"
	"github.com/QuantumNous/new-api/operation-admin/internal/newapidb"
	"github.com/QuantumNous/new-api/operation-admin/internal/rbac"
	"github.com/QuantumNous/new-api/operation-admin/internal/reconcile"
	"github.com/QuantumNous/new-api/operation-admin/internal/report"
	"github.com/QuantumNous/new-api/operation-admin/internal/settings"
	syncer "github.com/QuantumNous/new-api/operation-admin/internal/sync"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type API struct {
	DB        *gorm.DB
	NewAPIDB  *gorm.DB
	Auth      *auth.Service
	Sync      *syncer.Service
	Reconcile *reconcile.Service
	RBAC      *rbac.Service
}

func (a *API) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "ok"})
}

func (a *API) authPayload(token string, claims *auth.Claims) gin.H {
	perms, _ := a.RBAC.UserPermissions(claims.UserID, claims.AuthSource)
	paths, _ := a.RBAC.UserMenuPaths(claims.UserID, claims.AuthSource)
	super := claims.SuperAdmin || a.RBAC.IsSuperAdmin(claims.UserID, claims.AuthSource)
	return gin.H{
		"token": token,
		"user": gin.H{
			"id": claims.UserID, "username": claims.Username,
			"display_name": claims.DisplayName,
			"super_admin":  super,
			"auth_source":  claims.AuthSource,
			"permissions":  perms,
			"menu_paths":   paths,
		},
	}
}

func (a *API) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		PAT      string `json:"pat"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if req.PAT != "" {
		token, claims, err := a.Auth.LoginWithPAT(req.PAT)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": a.authPayload(token, claims)})
		return
	}
	token, need2FA, flow, claims, err := a.Auth.LoginLocalOrNewAPI(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
		return
	}
	if need2FA {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    gin.H{"require_2fa": true, "flow_token": flow},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": a.authPayload(token, claims)})
}

func (a *API) Login2FA(c *gin.Context) {
	var req struct {
		FlowToken string `json:"flow_token"`
		Code      string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	token, claims, err := a.Auth.Login2FA(req.FlowToken, req.Code)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": a.authPayload(token, claims)})
}

func (a *API) Me(c *gin.Context) {
	claimsVal, _ := c.Get("claims")
	claims, _ := claimsVal.(*auth.Claims)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	perms, _ := a.RBAC.UserPermissions(claims.UserID, claims.AuthSource)
	paths, _ := a.RBAC.UserMenuPaths(claims.UserID, claims.AuthSource)
	super := claims.SuperAdmin || a.RBAC.IsSuperAdmin(claims.UserID, claims.AuthSource)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"id": claims.UserID, "username": claims.Username,
		"display_name": claims.DisplayName,
		"super_admin":  super,
		"auth_source":  claims.AuthSource,
		"permissions":  perms,
		"menu_paths":   paths,
	}})
}

func (a *API) AuthPermissions(c *gin.Context) {
	claimsVal, _ := c.Get("claims")
	claims, _ := claimsVal.(*auth.Claims)
	perms, _ := a.RBAC.UserPermissions(claims.UserID, claims.AuthSource)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": perms})
}

func (a *API) AuthMenus(c *gin.Context) {
	claimsVal, _ := c.Get("claims")
	claims, _ := claimsVal.(*auth.Claims)
	paths, _ := a.RBAC.UserMenuPaths(claims.UserID, claims.AuthSource)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": paths})
}


func (a *API) ListChannels(c *gin.Context) {
	var rows []model.Channel
	if err := a.DB.Order("id asc").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

func (a *API) SetOpeningBalance(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		OpeningBalance float64 `json:"opening_balance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	res := a.DB.Model(&model.Channel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"opening_balance": req.OpeningBalance,
		"opening_set":     true,
	})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "渠道不存在，请先同步"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *API) GetBalanceQuery(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var ch model.Channel
	if err := a.DB.First(&ch, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "渠道不存在"})
		return
	}
	cfg := ch.ResolveBalanceQuery().Masked()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"config":             cfg,
			"channel_base_url":   ch.BaseURL,
			"has_api_key":        ch.GetBalanceQuery().APIKey != "",
			"sync_balance":       ch.SyncBalance,
			"notify_enabled":     ch.NotifyEnabled,
			"alias":              ch.Alias,
			"balance":            ch.Balance,
			"balance_raw":        ch.BalanceRaw,
			"balance_currency":   ch.BalanceCurrency,
			"balance_updated_at": ch.BalanceUpdatedTime,
		},
	})
}

func (a *API) PutBalanceQuery(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var ch model.Channel
	if err := a.DB.First(&ch, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "渠道不存在"})
		return
	}
	var req struct {
		SyncBalance   *bool               `json:"sync_balance"`
		NotifyEnabled *bool               `json:"notify_enabled"`
		Alias         *string             `json:"alias"`
		Config        balancequery.Config `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	cfgReq := req.Config
	// Keep existing api_key when client sends masked/empty placeholder.
	old := ch.GetBalanceQuery()
	if cfgReq.APIKey == "" || strings.Contains(cfgReq.APIKey, "****") {
		cfgReq.APIKey = old.APIKey
	}
	// Prefer explicit base_url; otherwise keep old or fall back to channel.base_url.
	if strings.TrimSpace(cfgReq.BaseURL) == "" {
		if strings.TrimSpace(old.BaseURL) != "" {
			cfgReq.BaseURL = old.BaseURL
		} else {
			cfgReq.BaseURL = ch.BaseURL
		}
	}
	cfgReq.BaseURL = strings.TrimRight(strings.TrimSpace(cfgReq.BaseURL), "/")
	if cfgReq.Enabled && strings.TrimSpace(cfgReq.URL) != "" {
		if err := cfgReq.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
			return
		}
	}
	if err := ch.SetBalanceQuery(cfgReq); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	updates := map[string]interface{}{
		"balance_query": ch.BalanceQueryJSON,
	}
	if req.SyncBalance != nil {
		updates["sync_balance"] = *req.SyncBalance
	}
	if req.NotifyEnabled != nil {
		updates["notify_enabled"] = *req.NotifyEnabled
	}
	if req.Alias != nil {
		updates["alias"] = strings.TrimSpace(*req.Alias)
	}
	if err := a.DB.Model(&ch).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"config":         cfgReq.Masked(),
		"sync_balance":   req.SyncBalance,
		"notify_enabled": req.NotifyEnabled,
		"alias":          updates["alias"],
	}})
}

func (a *API) TestBalanceQuery(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var ch model.Channel
	if err := a.DB.First(&ch, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "渠道不存在"})
		return
	}
	var req balancequery.Config
	_ = c.ShouldBindJSON(&req)
	cfg := req
	if !cfg.Enabled && strings.TrimSpace(cfg.URL) == "" {
		cfg = ch.ResolveBalanceQuery()
	} else {
		old := ch.GetBalanceQuery()
		if cfg.APIKey == "" || strings.Contains(cfg.APIKey, "****") {
			cfg.APIKey = old.APIKey
		}
		if strings.TrimSpace(cfg.BaseURL) == "" {
			if strings.TrimSpace(old.BaseURL) != "" {
				cfg.BaseURL = old.BaseURL
			} else {
				cfg.BaseURL = ch.BaseURL
			}
		}
		cfg.Enabled = true
	}
	res, err := balancequery.Fetch(cfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": res})
}

func (a *API) RefreshChannelBalance(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var ch model.Channel
	if err := a.DB.First(&ch, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "渠道不存在"})
		return
	}
	if err := a.Sync.RefreshOneBalance(&ch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	_ = a.DB.First(&ch, id)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"balance":            ch.Balance,
		"balance_raw":        ch.BalanceRaw,
		"balance_currency":   ch.BalanceCurrency,
		"balance_updated_at": ch.BalanceUpdatedTime,
	}})
}

func (a *API) BalanceQueryPresets(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": []gin.H{
		{
			"id":          "newapi",
			"name":        "New API /api/user/self",
			"description": "Authorization: Bearer {token}；金额 data.quota；USD = quota / 500000",
			"config":      balancequery.PresetNewAPI("", ""),
		},
	}})
}

func (a *API) ListRecharges(c *gin.Context) {
	channelID, _ := strconv.Atoi(c.Query("channel_id"))
	q := a.DB.Order("recharged_at desc")
	if channelID > 0 {
		q = q.Where("channel_id = ?", channelID)
	}
	var rows []model.Recharge
	if err := q.Limit(500).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

func (a *API) CreateRecharge(c *gin.Context) {
	var req struct {
		ChannelID   int     `json:"channel_id"`
		AmountUSD   float64 `json:"amount_usd"`
		RechargedAt int64   `json:"recharged_at"`
		Voucher     string  `json:"voucher"`
		Note        string  `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ChannelID <= 0 || req.AmountUSD == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误：需要 channel_id 与 amount_usd"})
		return
	}
	if req.RechargedAt <= 0 {
		req.RechargedAt = time.Now().Unix()
	}
	username, _ := c.Get("username")
	row := model.Recharge{
		ChannelID:   req.ChannelID,
		AmountUSD:   req.AmountUSD,
		RechargedAt: req.RechargedAt,
		Voucher:     req.Voucher,
		Note:        req.Note,
		CreatedBy:   username.(string),
	}
	if err := a.DB.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": row})
}

func (a *API) DeleteRecharge(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := a.DB.Delete(&model.Recharge{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *API) LiveReconcile(c *gin.Context) {
	channelID, _ := strconv.Atoi(c.Query("channel_id"))
	res, err := a.Reconcile.Live(channelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": res})
}

func (a *API) SyncStatus(c *gin.Context) {
	var rows []model.SyncState
	_ = a.DB.Find(&rows).Error
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

func (a *API) TriggerSync(c *gin.Context) {
	kind := c.Param("kind")
	var err error
	switch kind {
	case "channels":
		err = a.Sync.SyncChannels()
	case "balances":
		err = a.Sync.RefreshBalances("manual")
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "kind 应为 channels|balances（new-api 消费日志已改为直连库，不再同步）"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *API) GetBalanceSyncSettings(c *gin.Context) {
	enabled, minutes := settings.BalanceSyncConfig(a.DB, true, 5)
	var st model.SyncState
	_ = a.DB.Where("name = ?", "balances").First(&st).Error
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"enabled":           enabled,
		"interval_minutes":  minutes,
		"last_sync_at":      st.LastCreatedAt,
		"last_sync_count":   st.LastSourceID,
		"last_sync_message": st.Message,
		"updated_at":        st.UpdatedAt,
	}})
}

func (a *API) PutBalanceSyncSettings(c *gin.Context) {
	var req struct {
		Enabled         *bool `json:"enabled"`
		IntervalMinutes *int  `json:"interval_minutes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if req.Enabled != nil {
		if err := settings.SetBool(a.DB, model.SettingBalanceSyncEnabled, *req.Enabled); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			return
		}
	}
	if req.IntervalMinutes != nil {
		m := *req.IntervalMinutes
		if m < 1 {
			m = 1
		}
		if m > 1440 {
			m = 1440
		}
		if err := settings.SetInt(a.DB, model.SettingBalanceSyncInterval, m); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			return
		}
	}
	a.Sync.NotifyBalanceConfigChanged()
	enabled, minutes := settings.BalanceSyncConfig(a.DB, true, 5)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"enabled":          enabled,
		"interval_minutes": minutes,
	}})
}

func (a *API) DailyReport(c *gin.Context) {
	fromKey := strings.TrimSpace(c.Query("from"))
	toKey := strings.TrimSpace(c.Query("to"))
	if fromKey == "" {
		fromKey = strings.TrimSpace(c.Query("day"))
	}
	if toKey == "" {
		toKey = fromKey
	}
	channelID, _ := strconv.Atoi(c.Query("channel_id"))
	onlySync := true
	if v := strings.TrimSpace(c.Query("only_sync")); v == "0" || strings.EqualFold(v, "false") {
		onlySync = false
	}
	res, err := report.DailyConsume(a.DB, fromKey, toKey, channelID, onlySync)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": res})
}

func (a *API) DailyReportSnapshots(c *gin.Context) {
	channelID, _ := strconv.Atoi(c.Query("channel_id"))
	dayKey := strings.TrimSpace(c.Query("day"))
	items, err := report.ListDaySnapshots(a.DB, channelID, dayKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

// ListConsumeLogs reads new-api system consume logs directly (read-only).
func (a *API) ListConsumeLogs(c *gin.Context) {
	channelID, _ := strconv.Atoi(c.Query("channel_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	items, total, err := newapidb.ListConsumeLogs(a.NewAPIDB, channelID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items": items, "total": total, "page": page, "page_size": pageSize, "source": "new-api-logs",
		},
	})
}

// ListExternalLogs lists local external-channel logs (finance DB consume_logs).
func (a *API) ListExternalLogs(c *gin.Context) {
	channelID, _ := strconv.Atoi(c.Query("channel_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	q := a.DB.Model(&model.ConsumeLog{})
	if channelID > 0 {
		q = q.Where("channel_id = ?", channelID)
	}
	var total int64
	_ = q.Count(&total).Error
	var rows []model.ConsumeLog
	if err := q.Order("created_at desc, source_id desc").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items": rows, "total": total, "page": page, "page_size": pageSize, "source": "external",
		},
	})
}

// UpsertExternalLogs imports external channel usage logs into finance consume_logs.
func (a *API) UpsertExternalLogs(c *gin.Context) {
	var req struct {
		Items []model.ConsumeLog `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误：需要 items"})
		return
	}
	now := time.Now().Unix()
	n := 0
	for i := range req.Items {
		item := &req.Items[i]
		if item.SourceID == 0 || item.ChannelID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "每条记录需要 source_id 与 channel_id"})
			return
		}
		if item.CreatedAt <= 0 {
			item.CreatedAt = now
		}
		item.SyncedAt = now
		item.ID = 0
		err := a.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "source_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"user_id", "created_at", "username", "token_name", "model_name", "quota",
				"prompt_tokens", "completion_tokens", "use_time", "is_stream", "channel_id",
				"token_id", "token_group", "request_id", "upstream_request_id", "note", "synced_at",
			}),
		}).Create(item).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			return
		}
		n++
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"upserted": n}})
}
