package sdk

import (
	"encoding/json"
	"fmt"
)

const (
	DetectionScriptPath   = "/detect/script"
	DetectionSubmitPath = "/detect/submit"
	DetectionCheckPath   = "/detect/check"
	DetectionFingerprintPath = "/detect/fingerprint"
	DetectionStatsPath = "/detect/stats"
)

type DetectionScriptResponse struct {
	Script string `json:"script"`
}

type DetectionSubmitRequest struct {
	DetectionID string                 `json:"detection_id"`
	RiskScore   float64                `json:"risk_score"`
	Chain       []string               `json:"chain,omitempty"`
	Fingerprint string                 `json:"fingerprint,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	Timestamp   int64                  `json:"timestamp,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

type DetectionSubmitResponse struct {
	Success    bool    `json:"success"`
	RiskScore  float64 `json:"risk_score"`
	Anomalies  float64 `json:"anomalies,omitempty"`
}

type EnvironmentCheckRequest struct {
	Fingerprint   string                 `json:"fingerprint"`
	CanvasHash   string                 `json:"canvas_hash,omitempty"`
	WebGLVendor  string                 `json:"webgl_vendor,omitempty"`
	WebGLRenderer string                `json:"webgl_renderer,omitempty"`
	Fonts        []string               `json:"fonts,omitempty"`
	Plugins      []string               `json:"plugins,omitempty"`
	ProxyDetected bool                  `json:"proxy_detected,omitempty"`
	ScreenInfo  map[string]interface{} `json:"screen_info,omitempty"`
	Timezone    string                 `json:"timezone,omitempty"`
	Language    string                 `json:"language,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
}

type EnvironmentCheckResponse struct {
	IsBot        bool     `json:"is_bot"`
	RiskLevel   string   `json:"risk_level"`
	RiskScore   float64  `json:"risk_score"`
	DetectedFlags []string `json:"detected_flags,omitempty"`
	Fingerprint string   `json:"fingerprint"`
	IsUnique    bool     `json:"is_unique,omitempty"`
}

type FingerprintStats struct {
	TotalCount        int64                `json:"total_count"`
	BotCount          int64                `json:"bot_count"`
	ProxyCount        int64                `json:"proxy_count"`
	AverageRiskScore  float64              `json:"average_risk_score"`
	RiskDistribution  map[string]int64   `json:"risk_distribution,omitempty"`
	TopFingerprints   []map[string]interface{} `json:"top_fingerprints,omitempty"`
}

type DetectClient struct {
	*Client
}

func NewDetectClient(client *Client) *DetectClient {
	return &DetectClient{Client: client}
}

func (c *Client) Detect() *DetectClient {
	return NewDetectClient(c)
}

func (dc *DetectClient) GetScript(callback string) (string, error) {
	path := DetectionScriptPath
	if callback != "" {
		path = path + "?callback=" + callback
	}

	resp, err := dc.doRequestWithRetry("GET", path, nil)
	if err != nil {
		return "", err
	}

	return string(resp.Data), nil
}

func (dc *DetectClient) Submit(req *DetectionSubmitRequest) (*DetectionSubmitResponse, error) {
	if req == nil {
		return nil, NewSDKError(400, "request cannot be nil")
	}
	if req.DetectionID == "" {
		return nil, NewSDKError(400, "detection_id is required")
	}

	resp, err := dc.doRequestWithRetry("POST", DetectionSubmitPath, req)
	if err != nil {
		return nil, err
	}

	var result DetectionSubmitResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (dc *DetectClient) Check(req *EnvironmentCheckRequest) (*EnvironmentCheckResponse, error) {
	if req == nil {
		return nil, NewSDKError(400, "request cannot be nil")
	}
	if req.Fingerprint == "" {
		return nil, NewSDKError(400, "fingerprint is required")
	}

	resp, err := dc.doRequestWithRetry("POST", DetectionCheckPath, req)
	if err != nil {
		return nil, err
	}

	var result EnvironmentCheckResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (dc *DetectClient) GetFingerprintInfo(fingerprint string) (map[string]interface{}, error) {
	if fingerprint == "" {
		return nil, NewSDKError(400, "fingerprint is required")
	}

	path := fmt.Sprintf("%s?fingerprint=%s", DetectionFingerprintPath, fingerprint)
	resp, err := dc.doRequestWithRetry("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (dc *DetectClient) GetStats() (*FingerprintStats, error) {
	resp, err := dc.doRequestWithRetry("GET", DetectionStatsPath, nil)
	if err != nil {
		return nil, err
	}

	var result FingerprintStats
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
