package newapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	BaseURL     string
	AccessToken string
	HTTP        *http.Client
}

func New(baseURL, accessToken string) *Client {
	return &Client{
		BaseURL:     baseURL,
		AccessToken: accessToken,
		HTTP:        &http.Client{Timeout: 60 * time.Second},
	}
}

type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type LoginResult struct {
	Require2FA bool
	FlowToken  string
	AccessToken string
	UserID     int
	Username   string
	Role       int
	DisplayName string
	Raw        json.RawMessage
}

type UserSelf struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        int    `json:"role"`
}

type ChannelDTO struct {
	ID                 int     `json:"id"`
	Type               int     `json:"type"`
	Status             int     `json:"status"`
	Name               string  `json:"name"`
	Group              string  `json:"group"`
	BaseURL            *string `json:"base_url"`
	Balance            float64 `json:"balance"`
	BalanceUpdatedTime int64   `json:"balance_updated_time"`
	UsedQuota          int64   `json:"used_quota"`
	Remark             *string `json:"remark"`
}

type LogDTO struct {
	ID                int    `json:"id"`
	UserID            int    `json:"user_id"`
	CreatedAt         int64  `json:"created_at"`
	Type              int    `json:"type"`
	Username          string `json:"username"`
	TokenName         string `json:"token_name"`
	ModelName         string `json:"model_name"`
	Quota             int    `json:"quota"`
	PromptTokens      int    `json:"prompt_tokens"`
	CompletionTokens  int    `json:"completion_tokens"`
	UseTime           int    `json:"use_time"`
	IsStream          bool   `json:"is_stream"`
	ChannelID         int    `json:"channel"`
	TokenID           int    `json:"token_id"`
	Group             string `json:"group"`
	RequestID         string `json:"request_id"`
	UpstreamRequestID string `json:"upstream_request_id"`
}

const (
	LogTypeConsume = 2
	RoleRootUser   = 100
)

func (c *Client) Login(username, password string) (*LoginResult, error) {
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	var resp apiResponse
	if err := c.doJSON(http.MethodPost, "/api/user/login", body, "", &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(resp.Message)
	}
	return parseLoginData(resp.Data)
}

func (c *Client) Login2FA(flowToken, code string) (*LoginResult, error) {
	body, _ := json.Marshal(map[string]string{
		"flow_token": flowToken,
		"code":       code,
	})
	var resp apiResponse
	if err := c.doJSON(http.MethodPost, "/api/user/login/2fa", body, "", &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(resp.Message)
	}
	return parseLoginData(resp.Data)
}

func parseLoginData(raw json.RawMessage) (*LoginResult, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	if v, ok := probe["require_2fa"]; ok {
		var need bool
		_ = json.Unmarshal(v, &need)
		if need {
			var flow string
			_ = json.Unmarshal(probe["flow_token"], &flow)
			return &LoginResult{Require2FA: true, FlowToken: flow, Raw: raw}, nil
		}
	}
	out := &LoginResult{Raw: raw}
	if v, ok := probe["access_token"]; ok {
		_ = json.Unmarshal(v, &out.AccessToken)
	}
	if v, ok := probe["user"]; ok {
		var u UserSelf
		if err := json.Unmarshal(v, &u); err == nil {
			out.UserID = u.ID
			out.Username = u.Username
			out.DisplayName = u.DisplayName
			out.Role = u.Role
		}
	}
	return out, nil
}

func (c *Client) GetSelf(token string) (*UserSelf, error) {
	var resp apiResponse
	if err := c.doJSON(http.MethodGet, "/api/user/self", nil, token, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(resp.Message)
	}
	var u UserSelf
	if err := json.Unmarshal(resp.Data, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (c *Client) ListChannels(page, pageSize int) ([]ChannelDTO, int, error) {
	q := url.Values{}
	q.Set("p", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))
	q.Set("status", "") // all
	path := "/api/channel/?" + q.Encode()

	var resp apiResponse
	if err := c.doJSON(http.MethodGet, path, nil, c.AccessToken, &resp); err != nil {
		return nil, 0, err
	}
	if !resp.Success {
		return nil, 0, fmt.Errorf(resp.Message)
	}
	var payload struct {
		Items []ChannelDTO `json:"items"`
		Total int          `json:"total"`
	}
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		return nil, 0, err
	}
	return payload.Items, payload.Total, nil
}

func (c *Client) UpdateChannelBalance(id int) (float64, error) {
	path := fmt.Sprintf("/api/channel/update_balance/%d", id)
	var resp apiResponse
	if err := c.doJSON(http.MethodGet, path, nil, c.AccessToken, &resp); err != nil {
		return 0, err
	}
	if !resp.Success {
		return 0, fmt.Errorf(resp.Message)
	}
	// data may be {balance: x} or a bare number depending on version
	var asObj struct {
		Balance float64 `json:"balance"`
	}
	if err := json.Unmarshal(resp.Data, &asObj); err == nil && (asObj.Balance != 0 || bytes.Contains(resp.Data, []byte("balance"))) {
		return asObj.Balance, nil
	}
	var asNum float64
	if err := json.Unmarshal(resp.Data, &asNum); err == nil {
		return asNum, nil
	}
	return 0, nil
}

func (c *Client) ListConsumeLogs(page, pageSize int, startTs, endTs int64, channelID int) ([]LogDTO, int, error) {
	q := url.Values{}
	q.Set("p", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))
	q.Set("type", strconv.Itoa(LogTypeConsume))
	if startTs > 0 {
		q.Set("start_timestamp", strconv.FormatInt(startTs, 10))
	}
	if endTs > 0 {
		q.Set("end_timestamp", strconv.FormatInt(endTs, 10))
	}
	if channelID > 0 {
		q.Set("channel", strconv.Itoa(channelID))
	}
	path := "/api/log/?" + q.Encode()

	var resp apiResponse
	if err := c.doJSON(http.MethodGet, path, nil, c.AccessToken, &resp); err != nil {
		return nil, 0, err
	}
	if !resp.Success {
		return nil, 0, fmt.Errorf(resp.Message)
	}
	var payload struct {
		Items    []LogDTO `json:"items"`
		Total    int      `json:"total"`
		Page     int      `json:"page"`
		PageSize int      `json:"page_size"`
	}
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		return nil, 0, err
	}
	return payload.Items, payload.Total, nil
}

func (c *Client) doJSON(method, path string, body []byte, token string, out *apiResponse) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	tok := token
	if tok == "" {
		tok = c.AccessToken
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("new-api HTTP %d: %s", res.StatusCode, string(raw))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w; body=%s", err, string(raw))
	}
	return nil
}
