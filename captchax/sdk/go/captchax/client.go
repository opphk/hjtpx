package captchax

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Client struct {
	config     *Config
	httpClient *HTTPClient
	mu         sync.RWMutex
}

func NewClient(config *Config) (*Client, error) {
	if config.BaseURL == "" {
		return nil, NewError("baseURL is required")
	}

	client := &Client{
		config:     config,
		httpClient: NewHTTPClient(config.BaseURL, config.Timeout, config.RetryTimes),
	}

	if config.AppID != "" {
		client.httpClient.SetHeader("X-App-ID", config.AppID)
	}

	return client, nil
}

func NewClientWithDefault(baseURL string) (*Client, error) {
	return NewClient(NewConfig(baseURL))
}

func (c *Client) SetAppID(appID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config.AppID = appID
	c.httpClient.SetHeader("X-App-ID", appID)
}

func (c *Client) SetAPIVersion(version APIVersion) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config.APIVersion = version
}

func (c *Client) GetAPIVersion() APIVersion {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config.APIVersion
}

func (c *Client) getAPIPrefix() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return fmt.Sprintf("/api/%s", c.config.APIVersion)
}

func (c *Client) requireAppID() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.config.AppID == "" {
		return NewError("appId is required for captcha generation")
	}
	return nil
}

func (c *Client) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	body, err := c.httpClient.Get(ctx, "/health")
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, NewError(fmt.Sprintf("failed to parse response: %v", err))
	}

	var health HealthStatus
	dataBytes, _ := json.Marshal(apiResp.Data)
	if err := json.Unmarshal(dataBytes, &health); err != nil {
		return nil, NewError(fmt.Sprintf("failed to unmarshal health: %v", err))
	}

	return &health, nil
}

type SliderGenerateOptions struct {
	Width       *int
	Height      *int
	ClientInfo  string
	ScenarioID  string
}

func (c *Client) GenerateSliderCaptcha(ctx context.Context, opts *SliderGenerateOptions) (*SliderCaptchaResult, error) {
	if err := c.requireAppID(); err != nil {
		return nil, err
	}

	type requestBody struct {
		AppID      string `json:"app_id"`
		Width      *int   `json:"width,omitempty"`
		Height     *int   `json:"height,omitempty"`
		ClientInfo string `json:"client_info,omitempty"`
		ScenarioID string `json:"scenario_id,omitempty"`
	}

	body := requestBody{AppID: c.config.AppID}
	if opts != nil {
		if opts.Width != nil {
			body.Width = opts.Width
		}
		if opts.Height != nil {
			body.Height = opts.Height
		}
		body.ClientInfo = opts.ClientInfo
		body.ScenarioID = opts.ScenarioID
	}

	respBody, err := c.httpClient.Post(ctx, c.getAPIPrefix()+"/captcha/slider", body, "")
	if err != nil {
		return nil, err
	}

	var result SliderCaptchaResult
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) VerifySliderCaptcha(ctx context.Context, captchaID string, targetX int, targetY *int) (*SliderVerifyResult, error) {
	type requestBody struct {
		CaptchaID string `json:"captcha_id"`
		TargetX   int    `json:"target_x"`
		TargetY   *int   `json:"target_y,omitempty"`
	}

	body := requestBody{
		CaptchaID: captchaID,
		TargetX:   targetX,
		TargetY:   targetY,
	}

	respBody, err := c.httpClient.Post(ctx, c.getAPIPrefix()+"/captcha/slider/verify", body, "")
	if err != nil {
		return nil, err
	}

	var result SliderVerifyResult
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

type ClickGenerateOptions struct {
	CharCount   *int
	ClientInfo  string
	ScenarioID  string
}

func (c *Client) GenerateClickCaptcha(ctx context.Context, opts *ClickGenerateOptions) (*ClickCaptchaResult, error) {
	if err := c.requireAppID(); err != nil {
		return nil, err
	}

	type requestBody struct {
		AppID      string `json:"app_id"`
		CharCount  *int   `json:"char_count,omitempty"`
		ClientInfo string `json:"client_info,omitempty"`
		ScenarioID string `json:"scenario_id,omitempty"`
	}

	body := requestBody{AppID: c.config.AppID}
	if opts != nil {
		if opts.CharCount != nil {
			body.CharCount = opts.CharCount
		}
		body.ClientInfo = opts.ClientInfo
		body.ScenarioID = opts.ScenarioID
	}

	respBody, err := c.httpClient.Post(ctx, c.getAPIPrefix()+"/captcha/click", body, "")
	if err != nil {
		return nil, err
	}

	var result ClickCaptchaResult
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) VerifyClickCaptcha(ctx context.Context, captchaID string, clicks []CharPosition) (*ClickVerifyResult, error) {
	type requestBody struct {
		CaptchaID string         `json:"captcha_id"`
		Clicks    []CharPosition `json:"clicks"`
	}

	body := requestBody{
		CaptchaID: captchaID,
		Clicks:    clicks,
	}

	respBody, err := c.httpClient.Post(ctx, c.getAPIPrefix()+"/captcha/click/verify", body, "")
	if err != nil {
		return nil, err
	}

	var result ClickVerifyResult
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GeneratePuzzleCaptcha(ctx context.Context, opts *SliderGenerateOptions) (*PuzzleCaptchaResult, error) {
	if err := c.requireAppID(); err != nil {
		return nil, err
	}

	type requestBody struct {
		AppID      string `json:"app_id"`
		Width      *int   `json:"width,omitempty"`
		Height     *int   `json:"height,omitempty"`
		ClientInfo string `json:"client_info,omitempty"`
		ScenarioID string `json:"scenario_id,omitempty"`
	}

	body := requestBody{AppID: c.config.AppID}
	if opts != nil {
		if opts.Width != nil {
			body.Width = opts.Width
		}
		if opts.Height != nil {
			body.Height = opts.Height
		}
		body.ClientInfo = opts.ClientInfo
		body.ScenarioID = opts.ScenarioID
	}

	respBody, err := c.httpClient.Post(ctx, c.getAPIPrefix()+"/captcha/puzzle", body, "")
	if err != nil {
		return nil, err
	}

	var result PuzzleCaptchaResult
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) VerifyPuzzleCaptcha(ctx context.Context, captchaID string, targetX int, targetY *int) (*PuzzleVerifyResult, error) {
	type requestBody struct {
		CaptchaID string `json:"captcha_id"`
		TargetX   int    `json:"target_x"`
		TargetY   *int   `json:"target_y,omitempty"`
	}

	body := requestBody{
		CaptchaID: captchaID,
		TargetX:   targetX,
		TargetY:   targetY,
	}

	respBody, err := c.httpClient.Post(ctx, c.getAPIPrefix()+"/captcha/puzzle/verify", body, "")
	if err != nil {
		return nil, err
	}

	var result PuzzleVerifyResult
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) BatchVerify(ctx context.Context, items []BatchVerifyItem, deduplicationID string) (*BatchVerifyResponse, error) {
	type requestBody struct {
		Items []BatchVerifyItem `json:"items"`
	}

	body := requestBody{Items: items}

	respBody, err := c.httpClient.Post(ctx, c.getAPIPrefix()+"/captcha/batch/verify", body, deduplicationID)
	if err != nil {
		return nil, err
	}

	var result BatchVerifyResponse
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) ListScenarios(ctx context.Context) (*ScenarioListResponse, error) {
	respBody, err := c.httpClient.Get(ctx, c.getAPIPrefix()+"/captcha/scenarios")
	if err != nil {
		return nil, err
	}

	var result ScenarioListResponse
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) CreateScenario(ctx context.Context, scenario *Scenario) (*Scenario, error) {
	respBody, err := c.httpClient.Post(ctx, c.getAPIPrefix()+"/captcha/scenarios", scenario, "")
	if err != nil {
		return nil, err
	}

	var result Scenario
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GetScenario(ctx context.Context, scenarioID string) (*Scenario, error) {
	respBody, err := c.httpClient.Get(ctx, c.getAPIPrefix()+"/captcha/scenarios/"+scenarioID)
	if err != nil {
		return nil, err
	}

	var result Scenario
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) UpdateScenario(ctx context.Context, scenarioID string, updates map[string]interface{}) (*Scenario, error) {
	respBody, err := c.httpClient.Put(ctx, c.getAPIPrefix()+"/captcha/scenarios/"+scenarioID, updates)
	if err != nil {
		return nil, err
	}

	var result Scenario
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) DeleteScenario(ctx context.Context, scenarioID string) (*DeleteResponse, error) {
	respBody, err := c.httpClient.Delete(ctx, c.getAPIPrefix()+"/captcha/scenarios/"+scenarioID)
	if err != nil {
		return nil, err
	}

	var result DeleteResponse
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) RegisterWebhook(ctx context.Context, webhook *Webhook) (*Webhook, error) {
	respBody, err := c.httpClient.Post(ctx, c.getAPIPrefix()+"/captcha/webhook/register", webhook, "")
	if err != nil {
		return nil, err
	}

	var result Webhook
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) ListWebhooks(ctx context.Context, appID string) (*WebhookListResponse, error) {
	endpoint := c.getAPIPrefix() + "/captcha/webhook"
	if appID != "" {
		endpoint += "?app_id=" + appID
	}

	respBody, err := c.httpClient.Get(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var result WebhookListResponse
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) UpdateWebhook(ctx context.Context, webhookID string, updates map[string]interface{}) (*Webhook, error) {
	respBody, err := c.httpClient.Put(ctx, c.getAPIPrefix()+"/captcha/webhook/"+webhookID, updates)
	if err != nil {
		return nil, err
	}

	var result Webhook
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) UnregisterWebhook(ctx context.Context, webhookID string) (*DeleteResponse, error) {
	respBody, err := c.httpClient.Delete(ctx, c.getAPIPrefix()+"/captcha/webhook/"+webhookID)
	if err != nil {
		return nil, err
	}

	var result DeleteResponse
	if err := c.parseAPIResponse(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) CreateClientInfo(extra map[string]interface{}) string {
	info := map[string]interface{}{
		"platform":  "go",
		"timestamp": time.Now().UnixMilli(),
	}
	for k, v := range extra {
		info[k] = v
	}
	jsonBytes, _ := json.Marshal(info)
	return string(jsonBytes)
}

func (c *Client) parseAPIResponse(body []byte, v interface{}) error {
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return NewError(fmt.Sprintf("failed to parse response: %v", err))
	}

	if apiResp.Code != 0 && apiResp.Code != 200 {
		return NewErrorWithCode(apiResp.Message, apiResp.Code, 400)
	}

	if apiResp.Data != nil {
		dataBytes, err := json.Marshal(apiResp.Data)
		if err != nil {
			return NewError(fmt.Sprintf("failed to marshal data: %v", err))
		}
		if err := json.Unmarshal(dataBytes, v); err != nil {
			return NewError(fmt.Sprintf("failed to unmarshal data: %v", err))
		}
	}

	return nil
}
