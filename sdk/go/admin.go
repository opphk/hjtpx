package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	AdminLoginPath           = "/admin/login"
	AdminLogoutPath          = "/admin/logout"
	AdminDashboardStatsPath  = "/admin/dashboard/stats"
	AdminDashboardActivityPath = "/admin/dashboard/activity"
	AdminDashboardSystemStatusPath = "/admin/dashboard/system-status"
	AdminDashboardRequestTrendPath = "/admin/dashboard/request-trend"
	AdminStatsVerificationPath = "/admin/stats/verification"
	AdminStatsChartPath      = "/admin/stats/chart"
	AdminStatsTrendPath      = "/admin/stats/trend"
	AdminStatsHourlyPath     = "/admin/stats/hourly"
	AdminStatsRealtimePath   = "/admin/stats/realtime"
	AdminStatsRiskDistPath   = "/admin/stats/risk-distribution"
	AdminStatsTopIPsPath     = "/admin/stats/top-ips"
	AdminStatsAppPath        = "/admin/stats/application"
	AdminStatsCaptchaTypePath = "/admin/stats/captcha-type"
	AdminStatsReportPath     = "/admin/stats/report"
)

type AdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AdminLoginResponse struct {
	Token string    `json:"token"`
	User  AdminInfo `json:"user"`
}

type AdminInfo struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	IsSuperAdmin bool   `json:"is_super_admin"`
}

type AdminClient struct {
	*Client
	token string
}

func NewAdminClient(client *Client, token string) *AdminClient {
	return &AdminClient{Client: client, token: token}
}

func (c *Client) Admin(token string) *AdminClient {
	return NewAdminClient(c, token)
}

func (ac *AdminClient) Login(req *AdminLoginRequest) (*AdminLoginResponse, error) {
	if req == nil {
		return nil, NewSDKError(400, "request cannot be nil")
	}
	if req.Username == "" {
		return nil, NewSDKError(400, "username is required")
	}
	if req.Password == "" {
		return nil, NewSDKError(400, "password is required")
	}

	resp, err := ac.doRequestWithRetry("POST", AdminLoginPath, req)
	if err != nil {
		return nil, err
	}

	var result AdminLoginResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	ac.token = result.Token
	return &result, nil
}

func (ac *AdminClient) Logout() error {
	_, err := ac.doRequestWithRetry("POST", AdminLogoutPath, nil)
	return err
}

func (ac *AdminClient) doAuthenticatedRequest(method, path string, body interface{}) (*SDKResponse, error) {
	req, err := http.NewRequest(method, ac.buildURL(path), nil)
	if err != nil {
		return nil, err
	}

	if ac.token != "" {
		req.Header.Set("Authorization", "Bearer "+ac.token)
	}

	return ac.doRequest(method, path, body)
}

type DashboardStats struct {
	TotalUsers    int64 `json:"totalUsers"`
	TotalApps     int64 `json:"totalApps"`
	TotalRequests int64 `json:"totalRequests"`
	TotalErrors   int64 `json:"totalErrors"`
}

type ActivityItem struct {
	Time   string `json:"time"`
	Event  string `json:"event"`
	User   string `json:"user"`
	Status string `json:"status"`
}

type VerificationStats struct {
	Total        int64 `json:"total"`
	Pending      int64 `json:"pending"`
	Success      int64 `json:"success"`
	Failed       int64 `json:"failed"`
	Applications int64 `json:"applications"`
	Users        int64 `json:"users"`
}

type ChartDataPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type ChartData struct {
	Success []ChartDataPoint `json:"success"`
	Failed  []ChartDataPoint `json:"failed"`
	Total   []ChartDataPoint `json:"total"`
}

func (ac *AdminClient) GetDashboardStats() (*DashboardStats, error) {
	resp, err := ac.doAuthenticatedRequest("GET", AdminDashboardStatsPath, nil)
	if err != nil {
		return nil, err
	}

	var result DashboardStats
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (ac *AdminClient) GetRecentActivity(limit int) ([]ActivityItem, error) {
	path := AdminDashboardActivityPath
	if limit > 0 {
		path = fmt.Sprintf("%s?limit=%d", path, limit)
	}

	resp, err := ac.doAuthenticatedRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result []ActivityItem
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (ac *AdminClient) GetSystemStatus() (map[string]interface{}, error) {
	resp, err := ac.doAuthenticatedRequest("GET", AdminDashboardSystemStatusPath, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (ac *AdminClient) GetRequestTrend(startDate, endDate string, interval string) ([]map[string]interface{}, error) {
	path := AdminDashboardRequestTrendPath
	query := ""
	if startDate != "" {
		query = fmt.Sprintf("start_date=%s", startDate)
	}
	if endDate != "" {
		if query != "" {
			query += "&"
		}
		query += fmt.Sprintf("end_date=%s", endDate)
	}
	if interval != "" {
		if query != "" {
			query += "&"
		}
		query += fmt.Sprintf("interval=%s", interval)
	}
	if query != "" {
		path = path + "?" + query
	}

	resp, err := ac.doAuthenticatedRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (ac *AdminClient) GetVerificationStats(startDate, endDate string, appID int) (*VerificationStats, error) {
	path := AdminStatsVerificationPath
	query := ""
	if startDate != "" {
		query = fmt.Sprintf("start_date=%s", startDate)
	}
	if endDate != "" {
		if query != "" {
			query += "&"
		}
		query += fmt.Sprintf("end_date=%s", endDate)
	}
	if appID > 0 {
		if query != "" {
			query += "&"
		}
		query += fmt.Sprintf("app_id=%d", appID)
	}
	if query != "" {
		path = path + "?" + query
	}

	resp, err := ac.doAuthenticatedRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result VerificationStats
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (ac *AdminClient) GetChartData(chartType, period string) (*ChartData, error) {
	path := fmt.Sprintf("%s?type=%s&period=%s", AdminStatsChartPath, chartType, period)

	resp, err := ac.doAuthenticatedRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result ChartData
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type TrendDataPoint struct {
	Date        string `json:"date"`
	Total       int64  `json:"total"`
	Success     int64  `json:"success"`
	Failed      int64  `json:"failed"`
}

func (ac *AdminClient) GetTrendData(days int) ([]TrendDataPoint, error) {
	path := AdminStatsTrendPath
	if days > 0 {
		path = fmt.Sprintf("%s?days=%d", path, days)
	}

	resp, err := ac.doAuthenticatedRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result []TrendDataPoint
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

type HourlyStats struct {
	Hour   int `json:"hour"`
	Total  int64 `json:"total"`
	Success int64 `json:"success"`
	Failed  int64 `json:"failed"`
}

func (ac *AdminClient) GetHourlyStats(date string) ([]HourlyStats, error) {
	path := AdminStatsHourlyPath
	if date != "" {
		path = fmt.Sprintf("%s?date=%s", path, date)
	}

	resp, err := ac.doAuthenticatedRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result []HourlyStats
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

type RealtimeStats struct {
	CurrentQPS       float64 `json:"current_qps"`
	TotalToday       int64   `json:"total_today"`
	SuccessRate      float64 `json:"success_rate"`
	AvgResponseTime  float64 `json:"avg_response_time"`
	ActiveSessions   int     `json:"active_sessions"`
}

func (ac *AdminClient) GetRealtimeStats() (*RealtimeStats, error) {
	resp, err := ac.doAuthenticatedRequest("GET", AdminStatsRealtimePath, nil)
	if err != nil {
		return nil, err
	}

	var result RealtimeStats
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type RiskDistribution struct {
	Low      int64 `json:"low"`
	Medium   int64 `json:"medium"`
	High     int64 `json:"high"`
	Critical int64 `json:"critical"`
}

func (ac *AdminClient) GetRiskDistribution() (*RiskDistribution, error) {
	resp, err := ac.doAuthenticatedRequest("GET", AdminStatsRiskDistPath, nil)
	if err != nil {
		return nil, err
	}

	var result RiskDistribution
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type TopIPInfo struct {
	IP       string  `json:"ip"`
	Count    int64   `json:"count"`
	RiskScore float64 `json:"risk_score"`
	IsBlocked bool   `json:"is_blocked"`
}

func (ac *AdminClient) GetTopIPs(limit int, ipType string) ([]TopIPInfo, error) {
	path := AdminStatsTopIPsPath
	query := ""
	if limit > 0 {
		query = fmt.Sprintf("limit=%d", limit)
	}
	if ipType != "" {
		if query != "" {
			query += "&"
		}
		query += fmt.Sprintf("type=%s", ipType)
	}
	if query != "" {
		path = path + "?" + query
	}

	resp, err := ac.doAuthenticatedRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result []TopIPInfo
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

type AppStats struct {
	ID               uint    `json:"id"`
	Name             string  `json:"name"`
	TotalVerifications int64  `json:"total_verifications"`
	SuccessRate      float64 `json:"success_rate"`
	AvgRiskScore     float64 `json:"avg_risk_score"`
}

func (ac *AdminClient) GetApplicationStats(limit int) ([]AppStats, error) {
	path := AdminStatsAppPath
	if limit > 0 {
		path = fmt.Sprintf("%s?limit=%d", path, limit)
	}

	resp, err := ac.doAuthenticatedRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result []AppStats
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

type CaptchaTypeStats struct {
	Slider struct {
		Total       int64   `json:"total"`
		SuccessRate float64 `json:"success_rate"`
	} `json:"slider"`
	Click struct {
		Total       int64   `json:"total"`
		SuccessRate float64 `json:"success_rate"`
	} `json:"click"`
	Image struct {
		Total       int64   `json:"total"`
		SuccessRate float64 `json:"success_rate"`
	} `json:"image"`
}

func (ac *AdminClient) GetCaptchaTypeStats() (*CaptchaTypeStats, error) {
	resp, err := ac.doAuthenticatedRequest("GET", AdminStatsCaptchaTypePath, nil)
	if err != nil {
		return nil, err
	}

	var result CaptchaTypeStats
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (ac *AdminClient) GenerateReport(startDate, endDate, format string) (map[string]interface{}, error) {
	path := fmt.Sprintf("%s?start_date=%s&end_date=%s", AdminStatsReportPath, startDate, endDate)
	if format != "" {
		path = path + "&format=" + format
	}

	resp, err := ac.doAuthenticatedRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

type ClientWithToken struct {
	*Client
	accessToken string
}

func NewClientWithToken(accessToken string) *ClientWithToken {
	return &ClientWithToken{
		Client:      NewClient(),
		accessToken: accessToken,
	}
}

func (cwt *ClientWithToken) SetAccessToken(token string) {
	cwt.accessToken = token
}

func (cwt *ClientWithToken) doAuthenticatedRequest(method, path string, body interface{}) (*SDKResponse, error) {
	req, err := http.NewRequest(method, cwt.buildURL(path), nil)
	if err != nil {
		return nil, err
	}

	if cwt.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+cwt.accessToken)
	}

	return cwt.doRequest(method, path, body)
}

type TokenManager struct {
	accessToken  string
	refreshToken string
	expiresAt   time.Time
	client      *Client
}

func NewTokenManager(client *Client) *TokenManager {
	return &TokenManager{
		client: client,
	}
}

func (tm *TokenManager) SetTokens(accessToken, refreshToken string, expiresIn int) {
	tm.accessToken = accessToken
	tm.refreshToken = refreshToken
	tm.expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
}

func (tm *TokenManager) IsTokenExpired() bool {
	return time.Now().After(tm.expiresAt.Add(-time.Minute))
}

func (tm *TokenManager) GetAccessToken() string {
	return tm.accessToken
}

func (tm *TokenManager) Refresh() error {
	if tm.refreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	authClient := tm.client.Auth()
	resp, err := authClient.RefreshToken(&RefreshTokenRequest{
		RefreshToken: tm.refreshToken,
	})
	if err != nil {
		return err
	}

	tm.SetTokens(resp.AccessToken, resp.RefreshToken, resp.ExpiresIn)
	return nil
}
