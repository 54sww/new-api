package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/operation-admin/internal/model"
	"github.com/gin-gonic/gin"
)

func (a *API) ListNotifyChannels(c *gin.Context) {
	var rows []model.NotifyChannel
	q := a.DB.Order("id asc")
	if t := c.Query("type"); t != "" {
		q = q.Where("type = ?", t)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if name := c.Query("name"); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"id": r.ID, "name": r.Name, "type": r.Type, "status": r.Status,
			"remark": r.Remark, "config": r.MaskedConfig(),
			"created_at": r.CreatedAt, "updated_at": r.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

func (a *API) GetNotifyChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var row model.NotifyChannel
	if err := a.DB.First(&row, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "通知渠道不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"id": row.ID, "name": row.Name, "type": row.Type, "status": row.Status,
		"remark": row.Remark, "config": row.MaskedConfig(),
	}})
}

func mergeNotifyConfig(oldCfg, incoming map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range oldCfg {
		out[k] = v
	}
	for k, v := range incoming {
		if s, ok := v.(string); ok {
			if s == "" || strings.Contains(s, "****") || s == "********" {
				continue // keep old secret
			}
		}
		out[k] = v
	}
	return out
}

func (a *API) CreateNotifyChannel(c *gin.Context) {
	var req struct {
		Name   string                 `json:"name"`
		Type   string                 `json:"type"`
		Status string                 `json:"status"`
		Remark string                 `json:"remark"`
		Config map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "名称必填"})
		return
	}
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	if req.Type != model.NotifyTypeEmail && req.Type != model.NotifyTypeDingTalk {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "type 应为 email 或 dingtalk"})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	row := model.NotifyChannel{Name: req.Name, Type: req.Type, Status: req.Status, Remark: req.Remark}
	if err := row.SetConfigMap(req.Config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := a.DB.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": row.ID}})
}

func (a *API) UpdateNotifyChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var row model.NotifyChannel
	if err := a.DB.First(&row, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "通知渠道不存在"})
		return
	}
	var req struct {
		Name   string                 `json:"name"`
		Type   string                 `json:"type"`
		Status string                 `json:"status"`
		Remark string                 `json:"remark"`
		Config map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = req.Name
	}
	if t := strings.ToLower(strings.TrimSpace(req.Type)); t != "" {
		if t != model.NotifyTypeEmail && t != model.NotifyTypeDingTalk {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "type 应为 email 或 dingtalk"})
			return
		}
		row.Type = t
	}
	if req.Status != "" {
		row.Status = req.Status
	}
	row.Remark = req.Remark
	merged := mergeNotifyConfig(row.GetConfigMap(), req.Config)
	if err := row.SetConfigMap(merged); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := a.DB.Save(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *API) DeleteNotifyChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var n int64
	a.DB.Table("notify_group_channels").Where("notify_channel_id = ?", id).Count(&n)
	if n > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "渠道仍被通知组引用，请先解除"})
		return
	}
	if err := a.DB.Delete(&model.NotifyChannel{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *API) ListNotifyGroups(c *gin.Context) {
	var rows []model.NotifyGroup
	q := a.DB.Preload("Channels").Order("id asc")
	if name := c.Query("name"); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		chIDs := make([]uint, 0, len(r.Channels))
		chNames := make([]string, 0, len(r.Channels))
		for _, ch := range r.Channels {
			chIDs = append(chIDs, ch.ID)
			chNames = append(chNames, ch.Name)
		}
		out = append(out, gin.H{
			"id": r.ID, "name": r.Name, "code": r.Code, "status": r.Status,
			"remark": r.Remark, "channel_ids": chIDs, "channel_names": chNames,
			"created_at": r.CreatedAt, "updated_at": r.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

func (a *API) GetNotifyGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var row model.NotifyGroup
	if err := a.DB.Preload("Channels").First(&row, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "通知组不存在"})
		return
	}
	chIDs := make([]uint, 0, len(row.Channels))
	for _, ch := range row.Channels {
		chIDs = append(chIDs, ch.ID)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"id": row.ID, "name": row.Name, "code": row.Code, "status": row.Status,
		"remark": row.Remark, "channel_ids": chIDs,
	}})
}

func (a *API) CreateNotifyGroup(c *gin.Context) {
	var req struct {
		Name       string `json:"name"`
		Code       string `json:"code"`
		Status     string `json:"status"`
		Remark     string `json:"remark"`
		ChannelIDs []uint `json:"channel_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "名称必填"})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	row := model.NotifyGroup{Name: req.Name, Code: req.Code, Status: req.Status, Remark: req.Remark}
	if err := a.DB.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := a.replaceGroupChannels(row.ID, req.ChannelIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": row.ID}})
}

func (a *API) UpdateNotifyGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var row model.NotifyGroup
	if err := a.DB.First(&row, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "通知组不存在"})
		return
	}
	var req struct {
		Name       string `json:"name"`
		Code       string `json:"code"`
		Status     string `json:"status"`
		Remark     string `json:"remark"`
		ChannelIDs []uint `json:"channel_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = req.Name
	}
	row.Code = req.Code
	if req.Status != "" {
		row.Status = req.Status
	}
	row.Remark = req.Remark
	if err := a.DB.Save(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if req.ChannelIDs != nil {
		if err := a.replaceGroupChannels(row.ID, req.ChannelIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *API) DeleteNotifyGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var row model.NotifyGroup
	if err := a.DB.First(&row, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "通知组不存在"})
		return
	}
	_ = a.DB.Model(&row).Association("Channels").Clear()
	if err := a.DB.Delete(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *API) replaceGroupChannels(groupID uint, channelIDs []uint) error {
	var group model.NotifyGroup
	if err := a.DB.First(&group, groupID).Error; err != nil {
		return err
	}
	var channels []model.NotifyChannel
	if len(channelIDs) > 0 {
		if err := a.DB.Where("id IN ?", channelIDs).Find(&channels).Error; err != nil {
			return err
		}
	}
	return a.DB.Model(&group).Association("Channels").Replace(channels)
}

func (a *API) ListNotifyRules(c *gin.Context) {
	var rows []model.NotifyRule
	q := a.DB.Preload("NotifyGroup").Order("id asc")
	if name := c.Query("name"); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		groupName := ""
		if r.NotifyGroup != nil {
			groupName = r.NotifyGroup.Name
		}
		out = append(out, gin.H{
			"id": r.ID, "name": r.Name, "status": r.Status, "remark": r.Remark,
			"event_type": r.EventType, "threshold_usd": r.ThresholdUSD,
			"notify_group_id": r.NotifyGroupID, "notify_group_name": groupName,
			"scope": r.Scope, "channel_ids": r.GetChannelIDs(),
			"cooldown_minutes": r.CooldownMinutes,
			"created_at": r.CreatedAt, "updated_at": r.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

func (a *API) CreateNotifyRule(c *gin.Context) {
	var req struct {
		Name            string  `json:"name"`
		Status          string  `json:"status"`
		Remark          string  `json:"remark"`
		EventType       string  `json:"event_type"`
		ThresholdUSD    float64 `json:"threshold_usd"`
		NotifyGroupID   uint    `json:"notify_group_id"`
		Scope           string  `json:"scope"`
		ChannelIDs      []int   `json:"channel_ids"`
		CooldownMinutes  int     `json:"cooldown_minutes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "名称必填"})
		return
	}
	if req.NotifyGroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请选择通知组"})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if req.EventType == "" {
		req.EventType = model.NotifyEventBalanceBelow
	}
	if req.Scope == "" {
		req.Scope = model.NotifyScopeAllEnabled
	}
	if req.CooldownMinutes <= 0 {
		req.CooldownMinutes = 60
	}
	row := model.NotifyRule{
		Name: req.Name, Status: req.Status, Remark: req.Remark,
		EventType: req.EventType, ThresholdUSD: req.ThresholdUSD,
		NotifyGroupID: req.NotifyGroupID, Scope: req.Scope,
		CooldownMinutes: req.CooldownMinutes,
	}
	if err := row.SetChannelIDs(req.ChannelIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := a.DB.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": row.ID}})
}

func (a *API) UpdateNotifyRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var row model.NotifyRule
	if err := a.DB.First(&row, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "通知规则不存在"})
		return
	}
	var req struct {
		Name            string  `json:"name"`
		Status          string  `json:"status"`
		Remark          string  `json:"remark"`
		EventType       string  `json:"event_type"`
		ThresholdUSD    float64 `json:"threshold_usd"`
		NotifyGroupID   uint    `json:"notify_group_id"`
		Scope           string  `json:"scope"`
		ChannelIDs      []int   `json:"channel_ids"`
		CooldownMinutes  int     `json:"cooldown_minutes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = req.Name
	}
	if req.Status != "" {
		row.Status = req.Status
	}
	row.Remark = req.Remark
	if req.EventType != "" {
		row.EventType = req.EventType
	}
	row.ThresholdUSD = req.ThresholdUSD
	if req.NotifyGroupID > 0 {
		row.NotifyGroupID = req.NotifyGroupID
	}
	if req.Scope != "" {
		row.Scope = req.Scope
	}
	if req.CooldownMinutes > 0 {
		row.CooldownMinutes = req.CooldownMinutes
	}
	if req.ChannelIDs != nil {
		_ = row.SetChannelIDs(req.ChannelIDs)
	}
	if err := a.DB.Save(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *API) DeleteNotifyRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	a.DB.Where("rule_id = ?", id).Delete(&model.NotifyRuleState{})
	if err := a.DB.Delete(&model.NotifyRule{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

