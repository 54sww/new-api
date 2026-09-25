package notify

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/operation-admin/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	DB *gorm.DB
}

// CheckBalanceAfterRefresh evaluates active balance_below rules for one channel.
func (s *Service) CheckBalanceAfterRefresh(ch *model.Channel) {
	if s == nil || s.DB == nil || ch == nil || !ch.NotifyEnabled {
		return
	}
	var rules []model.NotifyRule
	if err := s.DB.Where("status = ? AND event_type = ?", "active", model.NotifyEventBalanceBelow).
		Find(&rules).Error; err != nil {
		log.Printf("[notify] load rules: %v", err)
		return
	}
	for _, rule := range rules {
		if !s.ruleApplies(&rule, ch.ID) {
			continue
		}
		if ch.Balance >= rule.ThresholdUSD {
			continue
		}
		if s.inCooldown(rule.ID, ch.ID, rule.CooldownMinutes) {
			continue
		}
		msg := fmt.Sprintf(
			"【余额告警】渠道 #%d %s\n当前余额: %.4f USD\n告警阈值: %.4f USD\n规则: %s",
			ch.ID, ch.DisplayName(), ch.Balance, rule.ThresholdUSD, rule.Name,
		)
		if err := s.sendToGroup(rule.NotifyGroupID, "渠道余额不足告警", msg); err != nil {
			log.Printf("[notify] rule=%d channel=%d send: %v", rule.ID, ch.ID, err)
			continue
		}
		now := time.Now().Unix()
		st := model.NotifyRuleState{
			RuleID:      rule.ID,
			ChannelID:   ch.ID,
			LastFired:   now,
			LastBalance: ch.Balance,
		}
		_ = s.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "rule_id"}, {Name: "channel_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"last_fired", "last_balance"}),
		}).Create(&st).Error
		log.Printf("[notify] fired rule=%d channel=%d balance=%.4f", rule.ID, ch.ID, ch.Balance)
	}
}

func (s *Service) ruleApplies(rule *model.NotifyRule, channelID int) bool {
	scope := rule.Scope
	if scope == "" {
		scope = model.NotifyScopeAllEnabled
	}
	if scope == model.NotifyScopeAllEnabled {
		return true
	}
	for _, id := range rule.GetChannelIDs() {
		if id == channelID {
			return true
		}
	}
	return false
}

func (s *Service) inCooldown(ruleID uint, channelID int, minutes int) bool {
	if minutes <= 0 {
		minutes = 60
	}
	var st model.NotifyRuleState
	err := s.DB.Where("rule_id = ? AND channel_id = ?", ruleID, channelID).First(&st).Error
	if err != nil {
		return false
	}
	return time.Now().Unix()-st.LastFired < int64(minutes)*60
}

func (s *Service) sendToGroup(groupID uint, title, content string) error {
	var group model.NotifyGroup
	if err := s.DB.Preload("Channels").First(&group, groupID).Error; err != nil {
		return fmt.Errorf("notify group %d: %w", groupID, err)
	}
	if group.Status != "active" {
		return fmt.Errorf("notify group %d inactive", groupID)
	}
	var lastErr error
	sent := 0
	for _, ch := range group.Channels {
		if ch.Status != "active" {
			continue
		}
		var err error
		switch ch.Type {
		case model.NotifyTypeEmail:
			err = sendEmail(&ch, title, content)
		case model.NotifyTypeDingTalk:
			err = sendDingTalk(&ch, content)
		default:
			err = fmt.Errorf("unsupported type %s", ch.Type)
		}
		if err != nil {
			lastErr = err
			log.Printf("[notify] channel %d (%s): %v", ch.ID, ch.Name, err)
			continue
		}
		sent++
	}
	if sent == 0 {
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("no active channels in group %d", groupID)
	}
	return nil
}

func sendEmail(ch *model.NotifyChannel, subject, body string) error {
	cfg := ch.GetConfigMap()
	host, _ := cfg["smtp_host"].(string)
	user, _ := cfg["username"].(string)
	pass, _ := cfg["password"].(string)
	from, _ := cfg["from"].(string)
	if from == "" {
		from = user
	}
	port := 465
	switch v := cfg["smtp_port"].(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case json.Number:
		if n, err := v.Int64(); err == nil {
			port = int(n)
		}
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			port = n
		}
	}
	to := parseRecipients(cfg["to"])
	if host == "" || user == "" || pass == "" || len(to) == 0 {
		return fmt.Errorf("email config incomplete")
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	msg := []byte("To: " + strings.Join(to, ",") + "\r\n" +
		"From: " + from + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body + "\r\n")
	auth := smtp.PlainAuth("", user, pass, host)
	return smtp.SendMail(addr, auth, from, to, msg)
}

func parseRecipients(v interface{}) []string {
	var to []string
	switch x := v.(type) {
	case []interface{}:
		for _, item := range x {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				to = append(to, strings.TrimSpace(s))
			}
		}
	case []string:
		for _, s := range x {
			if strings.TrimSpace(s) != "" {
				to = append(to, strings.TrimSpace(s))
			}
		}
	case string:
		for _, p := range strings.FieldsFunc(x, func(r rune) bool {
			return r == ',' || r == ';' || r == ' ' || r == '\n'
		}) {
			if p != "" {
				to = append(to, p)
			}
		}
	}
	return to
}

func sendDingTalk(ch *model.NotifyChannel, content string) error {
	cfg := ch.GetConfigMap()
	webhook, _ := cfg["webhook"].(string)
	secret, _ := cfg["secret"].(string)
	if strings.TrimSpace(webhook) == "" {
		return fmt.Errorf("dingtalk webhook empty")
	}
	hook := webhook
	if secret != "" {
		ts := time.Now().UnixMilli()
		stringToSign := fmt.Sprintf("%d\n%s", ts, secret)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(stringToSign))
		sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		sep := "?"
		if strings.Contains(hook, "?") {
			sep = "&"
		}
		hook = fmt.Sprintf("%s%stimestamp=%d&sign=%s", hook, sep, ts, sign)
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": content},
	})
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(hook, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("dingtalk http %d: %s", resp.StatusCode, string(body))
	}
	var res struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(body, &res)
	if res.ErrCode != 0 {
		return fmt.Errorf("dingtalk: %s", res.ErrMsg)
	}
	return nil
}
