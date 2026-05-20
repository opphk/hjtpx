package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrPluginNotFound      = errors.New("plugin not found")
	ErrPluginAlreadyExists = errors.New("plugin already exists")
	ErrPluginDisabled      = errors.New("plugin is disabled")
	ErrInvalidPluginType   = errors.New("invalid plugin type")
	ErrPluginExecutionFailed = errors.New("plugin execution failed")
)

type PluginType string

const (
	PluginTypeCaptcha     PluginType = "captcha"
	PluginTypeAnalytics   PluginType = "analytics"
	PluginTypeSecurity    PluginType = "security"
	PluginTypeIntegration PluginType = "integration"
	PluginTypeCustom      PluginType = "custom"
)

type PluginStatus string

const (
	PluginStatusActive    PluginStatus = "active"
	PluginStatusInactive  PluginStatus = "inactive"
	PluginStatusSuspended PluginStatus = "suspended"
	PluginStatusPending   PluginStatus = "pending"
)

type PluginSystemService interface {
	CreatePlugin(ctx context.Context, plugin *Plugin) error
	GetPlugin(ctx context.Context, pluginID string) (*Plugin, error)
	UpdatePlugin(ctx context.Context, plugin *Plugin) error
	DeletePlugin(ctx context.Context, pluginID string) error
	ListPlugins(ctx context.Context, filters *PluginFilters) ([]*Plugin, error)
	EnablePlugin(ctx context.Context, pluginID string) error
	DisablePlugin(ctx context.Context, pluginID string) error
	ExecutePlugin(ctx context.Context, pluginID string, input *PluginInput) (*PluginOutput, error)
	UploadPlugin(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*Plugin, error)
	DownloadPlugin(ctx context.Context, pluginID string) ([]byte, error)
	GetPluginMetrics(ctx context.Context, pluginID string) (*PluginMetrics, error)
	RegisterHook(ctx context.Context, hook *PluginHook) error
	UnregisterHook(ctx context.Context, hookID string) error
}

type Plugin struct {
	PluginID      string          `json:"plugin_id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Version       string          `json:"version"`
	Type          PluginType      `json:"type"`
	Author        string          `json:"author"`
	Homepage      string          `json:"homepage"`
	License       string          `json:"license"`
	Status        PluginStatus    `json:"status"`
	Configuration json.RawMessage `json:"configuration"`
	Permissions   []string        `json:"permissions"`
	Dependencies  []PluginDependency `json:"dependencies"`
	Hooks         []string        `json:"hooks"`
	IconURL       string          `json:"icon_url"`
	Screenshots   []string        `json:"screenshots"`
	DownloadCount int             `json:"download_count"`
	Rating        float64         `json:"rating"`
	Tags          []string        `json:"tags"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	PublishedAt   *time.Time      `json:"published_at,omitempty"`
}

type PluginDependency struct {
	PluginID string `json:"plugin_id"`
	Version  string `json:"version"`
	Required bool   `json:"required"`
}

type PluginFilters struct {
	Type       PluginType   `json:"type,omitempty"`
	Status     PluginStatus `json:"status,omitempty"`
	Author     string       `json:"author,omitempty"`
	Tags       []string     `json:"tags,omitempty"`
	Search     string       `json:"search,omitempty"`
	SortBy     string       `json:"sort_by,omitempty"`
	Order      string       `json:"order,omitempty"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
}

type PluginInput struct {
	Context    map[string]interface{} `json:"context"`
	Data       map[string]interface{} `json:"data"`
	Parameters map[string]interface{} `json:"parameters"`
	AuthToken  string                 `json:"auth_token,omitempty"`
	UserID     string                `json:"user_id,omitempty"`
	AppID      string                `json:"app_id,omitempty"`
}

type PluginOutput struct {
	Success    bool                   `json:"success"`
	Data       map[string]interface{} `json:"data"`
	Error      string                 `json:"error,omitempty"`
	Metrics    *PluginExecutionMetrics `json:"metrics,omitempty"`
	DurationMs int64                 `json:"duration_ms"`
}

type PluginExecutionMetrics struct {
	ExecutionTimeMs int64  `json:"execution_time_ms"`
	MemoryUsedMB    int64  `json:"memory_used_mb"`
	CPUUsedPercent  float64 `json:"cpu_used_percent"`
	CacheHit       bool   `json:"cache_hit"`
}

type PluginMetrics struct {
	PluginID       string             `json:"plugin_id"`
	TotalExecutions int64            `json:"total_executions"`
	SuccessCount   int64             `json:"success_count"`
	FailureCount   int64             `json:"failure_count"`
	AvgLatencyMs   float64           `json:"avg_latency_ms"`
	LastExecutedAt *time.Time        `json:"last_executed_at,omitempty"`
	TodayMetrics   *DailyMetrics     `json:"today_metrics"`
	WeeklyMetrics  []*DailyMetrics   `json:"weekly_metrics"`
}

type DailyMetrics struct {
	Date         time.Time `json:"date"`
	Executions   int64    `json:"executions"`
	SuccessRate  float64  `json:"success_rate"`
	AvgLatencyMs float64  `json:"avg_latency_ms"`
}

type PluginHook struct {
	HookID     string   `json:"hook_id"`
	PluginID   string   `json:"plugin_id"`
	Event      string   `json:"event"`
	Endpoint   string   `json:"endpoint"`
	Method     string   `json:"method"`
	Headers    map[string]string `json:"headers,omitempty"`
	Enabled    bool     `json:"enabled"`
	Priority   int      `json:"priority"`
	Filters    map[string]string `json:"filters,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type pluginSystemService struct {
	plugins    map[string]*Plugin
	hooks      map[string]*PluginHook
	metrics    map[string]*PluginMetrics
	configs    map[string]json.RawMessage
	mu         sync.RWMutex
	httpClient *http.Client
}

func NewPluginSystemService() PluginSystemService {
	service := &pluginSystemService{
		plugins:  make(map[string]*Plugin),
		hooks:    make(map[string]*PluginHook),
		metrics:  make(map[string]*PluginMetrics),
		configs:  make(map[string]json.RawMessage),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	service.initializeDefaultPlugins()
	return service
}

func (s *pluginSystemService) initializeDefaultPlugins() {
	defaultPlugins := []*Plugin{
		{
			PluginID: "builtin-analytics",
			Name:     "Analytics Plugin",
			Description: "Built-in analytics and reporting",
			Version:  "1.0.0",
			Type:     PluginTypeAnalytics,
			Author:   "HJTPX Team",
			Status:   PluginStatusActive,
			Tags:     []string{"analytics", "reporting"},
		},
		{
			PluginID: "builtin-security",
			Name:     "Security Plugin",
			Description: "Built-in security features",
			Version:  "1.0.0",
			Type:     PluginTypeSecurity,
			Author:   "HJTPX Team",
			Status:   PluginStatusActive,
			Tags:     []string{"security", "protection"},
		},
	}

	for _, plugin := range defaultPlugins {
		s.plugins[plugin.PluginID] = plugin
		s.metrics[plugin.PluginID] = &PluginMetrics{
			PluginID:        plugin.PluginID,
			TotalExecutions: 0,
			SuccessCount:    0,
			FailureCount:    0,
			AvgLatencyMs:    0,
		}
	}
}

func (s *pluginSystemService) CreatePlugin(ctx context.Context, plugin *Plugin) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.plugins[plugin.PluginID]; exists {
		return ErrPluginAlreadyExists
	}

	if plugin.PluginID == "" {
		plugin.PluginID = fmt.Sprintf("plugin-%d", time.Now().UnixNano())
	}

	plugin.CreatedAt = time.Now()
	plugin.UpdatedAt = time.Now()
	plugin.Status = PluginStatusPending

	s.plugins[plugin.PluginID] = plugin
	s.metrics[plugin.PluginID] = &PluginMetrics{
		PluginID:        plugin.PluginID,
		TotalExecutions: 0,
		SuccessCount:    0,
		FailureCount:    0,
		AvgLatencyMs:    0,
	}

	return nil
}

func (s *pluginSystemService) GetPlugin(ctx context.Context, pluginID string) (*Plugin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plugin, exists := s.plugins[pluginID]
	if !exists {
		return nil, ErrPluginNotFound
	}

	return plugin, nil
}

func (s *pluginSystemService) UpdatePlugin(ctx context.Context, plugin *Plugin) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.plugins[plugin.PluginID]
	if !exists {
		return ErrPluginNotFound
	}

	plugin.UpdatedAt = time.Now()
	plugin.CreatedAt = existing.CreatedAt

	s.plugins[plugin.PluginID] = plugin
	return nil
}

func (s *pluginSystemService) DeletePlugin(ctx context.Context, pluginID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.plugins[pluginID]; !exists {
		return ErrPluginNotFound
	}

	delete(s.plugins, pluginID)
	delete(s.metrics, pluginID)
	delete(s.configs, pluginID)

	return nil
}

func (s *pluginSystemService) ListPlugins(ctx context.Context, filters *PluginFilters) ([]*Plugin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Plugin
	for _, plugin := range s.plugins {
		if s.matchesFilters(plugin, filters) {
			result = append(result, plugin)
		}
	}

	return result, nil
}

func (s *pluginSystemService) matchesFilters(plugin *Plugin, filters *PluginFilters) bool {
	if filters == nil {
		return true
	}

	if filters.Type != "" && plugin.Type != filters.Type {
		return false
	}

	if filters.Status != "" && plugin.Status != filters.Status {
		return false
	}

	if filters.Author != "" && plugin.Author != filters.Author {
		return false
	}

	if len(filters.Tags) > 0 {
		hasTag := false
		for _, tag := range filters.Tags {
			for _, pluginTag := range plugin.Tags {
				if strings.Contains(strings.ToLower(pluginTag), strings.ToLower(tag)) {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	if filters.Search != "" {
		searchLower := strings.ToLower(filters.Search)
		if !strings.Contains(strings.ToLower(plugin.Name), searchLower) &&
			!strings.Contains(strings.ToLower(plugin.Description), searchLower) {
			return false
		}
	}

	return true
}

func (s *pluginSystemService) EnablePlugin(ctx context.Context, pluginID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plugin, exists := s.plugins[pluginID]
	if !exists {
		return ErrPluginNotFound
	}

	plugin.Status = PluginStatusActive
	plugin.UpdatedAt = time.Now()

	return nil
}

func (s *pluginSystemService) DisablePlugin(ctx context.Context, pluginID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plugin, exists := s.plugins[pluginID]
	if !exists {
		return ErrPluginNotFound
	}

	plugin.Status = PluginStatusInactive
	plugin.UpdatedAt = time.Now()

	return nil
}

func (s *pluginSystemService) ExecutePlugin(ctx context.Context, pluginID string, input *PluginInput) (*PluginOutput, error) {
	s.mu.RLock()
	plugin, exists := s.plugins[pluginID]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrPluginNotFound
	}

	if plugin.Status != PluginStatusActive {
		return nil, ErrPluginDisabled
	}

	startTime := time.Now()
	output := &PluginOutput{
		Success: true,
		Data:    make(map[string]interface{}),
	}

	defer func() {
		output.DurationMs = time.Since(startTime).Milliseconds()

		s.mu.Lock()
		if m, ok := s.metrics[pluginID]; ok {
			m.TotalExecutions++
			if output.Success {
				m.SuccessCount++
			} else {
				m.FailureCount++
			}
			now := time.Now()
			m.LastExecutedAt = &now

			total := m.SuccessCount + m.FailureCount
			if total > 0 {
				m.AvgLatencyMs = (m.AvgLatencyMs*float64(total-1) + float64(output.DurationMs)) / float64(total)
			}
		}
		s.mu.Unlock()
	}()

	output.Data["result"] = "success"
	output.Data["plugin_id"] = pluginID
	output.Data["input_received"] = len(input.Data) > 0

	return output, nil
}

func (s *pluginSystemService) UploadPlugin(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*Plugin, error) {
	ext := filepath.Ext(header.Filename)
	if ext != ".zip" && ext != ".tar.gz" {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	_, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin file: %w", err)
	}

	plugin := &Plugin{
		PluginID:    fmt.Sprintf("uploaded-%d", time.Now().UnixNano()),
		Name:        strings.TrimSuffix(header.Filename, ext),
		Description: "Uploaded plugin",
		Version:     "1.0.0",
		Type:        PluginTypeCustom,
		Author:      "user",
		Status:      PluginStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.CreatePlugin(ctx, plugin); err != nil {
		return nil, err
	}

	return plugin, nil
}

func (s *pluginSystemService) DownloadPlugin(ctx context.Context, pluginID string) ([]byte, error) {
	s.mu.RLock()
	plugin, exists := s.plugins[pluginID]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrPluginNotFound
	}

	plugin.DownloadCount++
	return []byte(fmt.Sprintf("Plugin archive for %s v%s", plugin.Name, plugin.Version)), nil
}

func (s *pluginSystemService) GetPluginMetrics(ctx context.Context, pluginID string) (*PluginMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics, exists := s.metrics[pluginID]
	if !exists {
		return nil, ErrPluginNotFound
	}

	return metrics, nil
}

func (s *pluginSystemService) RegisterHook(ctx context.Context, hook *PluginHook) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.plugins[hook.PluginID]; !exists {
		return ErrPluginNotFound
	}

	if hook.HookID == "" {
		hook.HookID = fmt.Sprintf("hook-%d", time.Now().UnixNano())
	}

	hook.CreatedAt = time.Now()
	s.hooks[hook.HookID] = hook

	return nil
}

func (s *pluginSystemService) UnregisterHook(ctx context.Context, hookID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.hooks[hookID]; !exists {
		return ErrPluginNotFound
	}

	delete(s.hooks, hookID)
	return nil
}
