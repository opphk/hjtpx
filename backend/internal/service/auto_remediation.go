package service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type AutoRemediation struct {
	mu              sync.RWMutex
	playbooks       map[string]*RemediationPlaybook
	executionHistory []ExecutionRecord
	autoExecEnabled bool
}

type RemediationPlaybook struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	TriggerConditions []Condition  `json:"trigger_conditions"`
	Steps          []PlaybookStep  `json:"steps"`
	RollbackSteps  []PlaybookStep  `json:"rollback_steps"`
	ApprovalRequired bool          `json:"approval_required"`
	Timeout        time.Duration   `json:"timeout"`
	RetryPolicy    *RetryPolicy    `json:"retry_policy"`
	Enabled        bool            `json:"enabled"`
	Priority       int             `json:"priority"`
	Category       string          `json:"category"`
}

type PlaybookStep struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Command      string                 `json:"command,omitempty"`
	Script       string                 `json:"script,omitempty"`
	Parameters   map[string]interface{} `json:"parameters"`
	ContinueOnError bool                `json:"continue_on_error"`
	Timeout      time.Duration          `json:"timeout"`
}

type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	RetryInterval time.Duration `json:"retry_interval"`
	BackoffFactor float64       `json:"backoff_factor"`
	RetryableErrors []string    `json:"retryable_errors"`
}

type Condition struct {
	Type     string      `json:"type"`
	Metric   string      `json:"metric"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
	Duration time.Duration `json:"duration,omitempty"`
}

type ExecutionRecord struct {
	ID           string                 `json:"id"`
	PlaybookID   string                 `json:"playbook_id"`
	AlertID      string                 `json:"alert_id"`
	TriggeredAt  time.Time              `json:"triggered_at"`
	Status       string                 `json:"status"`
	Steps        []StepExecution        `json:"steps"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Error        string                 `json:"error,omitempty"`
	RollbackRequired bool               `json:"rollback_required"`
	RollbackStatus string               `json:"rollback_status,omitempty"`
	ApprovedBy   string                 `json:"approved_by,omitempty"`
	ApprovedAt   *time.Time             `json:"approved_at,omitempty"`
}

type StepExecution struct {
	StepID        string                 `json:"step_id"`
	StepName      string                 `json:"step_name"`
	Status        string                 `json:"status"`
	StartedAt     time.Time              `json:"started_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Output        string                 `json:"output,omitempty"`
	Error         string                 `json:"error,omitempty"`
	RetryCount    int                   `json:"retry_count"`
}

type RemediationTemplate struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	UseCases     []string `json:"use_cases"`
	Difficulty   string   `json:"difficulty"`
	EstimatedTime time.Duration `json:"estimated_time"`
}

func NewAutoRemediation() *AutoRemediation {
	remediation := &AutoRemediation{
		playbooks:        make(map[string]*RemediationPlaybook),
		executionHistory: make([]ExecutionRecord, 0),
		autoExecEnabled:  true,
	}
	remediation.initializePlaybooks()
	return remediation
}

func (r *AutoRemediation) initializePlaybooks() {
	r.playbooks = map[string]*RemediationPlaybook{
		"high-cpu": {
			ID:           "high-cpu",
			Name:         "高CPU使用率自动修复",
			Description:  "当CPU使用率持续过高时，自动执行扩容或优化措施",
			TriggerConditions: []Condition{
				{Type: "metric", Metric: "cpu_usage", Operator: ">", Value: 80, Duration: 5 * time.Minute},
			},
			Steps: []PlaybookStep{
				{
					ID:   "step-1",
					Name: "检查当前进程",
					Type: "command",
					Command: "ps aux --sort=-%cpu | head -10",
					Timeout: 30 * time.Second,
				},
				{
					ID:   "step-2",
					Name: "检查CPU使用趋势",
					Type: "check_metrics",
					Parameters: map[string]interface{}{
						"metric": "cpu_usage",
						"period": "15m",
					},
					Timeout: 1 * time.Minute,
				},
				{
					ID:   "step-3",
					Name: "扩容服务",
					Type: "scale",
					Parameters: map[string]interface{}{
						"action": "scale_out",
						"replicas": 2,
					},
					Timeout: 5 * time.Minute,
				},
			},
			RollbackSteps: []PlaybookStep{
				{
					ID:   "rollback-1",
					Name: "缩容服务",
					Type: "scale",
					Parameters: map[string]interface{}{
						"action": "scale_in",
						"replicas": 1,
					},
					Timeout: 5 * time.Minute,
				},
			},
			ApprovalRequired: true,
			Timeout:          10 * time.Minute,
			RetryPolicy: &RetryPolicy{
				MaxRetries:     3,
				RetryInterval:  1 * time.Minute,
				BackoffFactor:   2.0,
				RetryableErrors: []string{"timeout", "connection_failed"},
			},
			Enabled:  true,
			Priority: 1,
			Category: "performance",
		},
		"high-memory": {
			ID:           "high-memory",
			Name:         "高内存使用率自动修复",
			Description:  "当内存使用率过高时，自动清理缓存或重启服务",
			TriggerConditions: []Condition{
				{Type: "metric", Metric: "memory_usage", Operator: ">", Value: 85, Duration: 5 * time.Minute},
			},
			Steps: []PlaybookStep{
				{
					ID:   "step-1",
					Name: "检查内存使用情况",
					Type: "command",
					Command: "free -h",
					Timeout: 30 * time.Second,
				},
				{
					ID:   "step-2",
					Name: "清理缓存",
					Type: "command",
					Command: "sync && echo 3 > /proc/sys/vm/drop_caches",
					Parameters: map[string]interface{}{
						"requires_root": true,
					},
					ContinueOnError: true,
					Timeout:         1 * time.Minute,
				},
				{
					ID:   "step-3",
					Name: "重启服务",
					Type: "restart",
					Parameters: map[string]interface{}{
						"service": "api",
					},
					Timeout: 3 * time.Minute,
				},
			},
			RollbackSteps: []PlaybookStep{
				{
					ID:   "rollback-1",
					Name: "重启服务",
					Type: "restart",
					Parameters: map[string]interface{}{
						"service": "api",
					},
					Timeout: 3 * time.Minute,
				},
			},
			ApprovalRequired: true,
			Timeout:          10 * time.Minute,
			Enabled:          true,
			Priority:         1,
			Category:         "performance",
		},
		"high-error-rate": {
			ID:           "high-error-rate",
			Name:         "高错误率自动修复",
			Description:  "当错误率突增时，自动排查并恢复服务",
			TriggerConditions: []Condition{
				{Type: "metric", Metric: "error_rate", Operator: ">", Value: 10, Duration: 2 * time.Minute},
			},
			Steps: []PlaybookStep{
				{
					ID:   "step-1",
					Name: "检查错误日志",
					Type: "command",
					Command: "tail -100 /var/log/error.log",
					Timeout: 30 * time.Second,
				},
				{
					ID:   "step-2",
					Name: "检查依赖服务",
					Type: "check_services",
					Parameters: map[string]interface{}{
						"services": []string{"database", "cache", "queue"},
					},
					Timeout: 1 * time.Minute,
				},
				{
					ID:   "step-3",
					Name: "重启故障服务",
					Type: "restart",
					Parameters: map[string]interface{}{
						"service": "api",
					},
					Timeout: 3 * time.Minute,
				},
				{
					ID:   "step-4",
					Name: "发送告警通知",
					Type: "notify",
					Parameters: map[string]interface{}{
						"channels": []string{"email", "slack"},
						"severity": "high",
					},
					ContinueOnError: true,
					Timeout:         30 * time.Second,
				},
			},
			ApprovalRequired: false,
			Timeout:          10 * time.Minute,
			Enabled:          true,
			Priority:         2,
			Category:         "reliability",
		},
		"disk-space-low": {
			ID:           "disk-space-low",
			Name:         "磁盘空间不足自动修复",
			Description:  "当磁盘空间不足时，自动清理临时文件",
			TriggerConditions: []Condition{
				{Type: "metric", Metric: "disk_usage", Operator: ">", Value: 90, Duration: 1 * time.Minute},
			},
			Steps: []PlaybookStep{
				{
					ID:   "step-1",
					Name: "检查磁盘使用",
					Type: "command",
					Command: "df -h",
					Timeout: 30 * time.Second,
				},
				{
					ID:   "step-2",
					Name: "清理临时文件",
					Type: "command",
					Command: "rm -rf /tmp/*",
					Parameters: map[string]interface{}{
						"requires_root": true,
					},
					ContinueOnError: true,
					Timeout:         1 * time.Minute,
				},
				{
					ID:   "step-3",
					Name: "清理日志文件",
					Type: "command",
					Command: "find /var/log -name '*.log' -mtime +7 -delete",
					Parameters: map[string]interface{}{
						"requires_root": true,
					},
					ContinueOnError: true,
					Timeout:         2 * time.Minute,
				},
				{
					ID:   "step-4",
					Name: "发送告警通知",
					Type: "notify",
					Parameters: map[string]interface{}{
						"channels": []string{"email"},
						"severity": "medium",
					},
					ContinueOnError: true,
					Timeout:         30 * time.Second,
				},
			},
			ApprovalRequired: false,
			Timeout:          5 * time.Minute,
			Enabled:          true,
			Priority:         3,
			Category:         "resource",
		},
		"database-connection": {
			ID:           "database-connection",
			Name:         "数据库连接问题自动修复",
			Description:  "当数据库连接耗尽时，自动重启数据库连接池",
			TriggerConditions: []Condition{
				{Type: "metric", Metric: "db_connections", Operator: ">", Value: 90, Duration: 2 * time.Minute},
			},
			Steps: []PlaybookStep{
				{
					ID:   "step-1",
					Name: "检查数据库连接",
					Type: "command",
					Command: "SELECT count(*) FROM pg_stat_activity;",
					Parameters: map[string]interface{}{
						"database": "postgres",
					},
					Timeout: 30 * time.Second,
				},
				{
					ID:   "step-2",
					Name: "终止空闲连接",
					Type: "command",
					Command: "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state='idle';",
					Parameters: map[string]interface{}{
						"database": "postgres",
					},
					ContinueOnError: true,
					Timeout:         1 * time.Minute,
				},
				{
					ID:   "step-3",
					Name: "增加连接池大小",
					Type: "config",
					Parameters: map[string]interface{}{
						"key": "max_connections",
						"value": "200",
					},
					Timeout: 30 * time.Second,
				},
				{
					ID:   "step-4",
					Name: "重启应用服务",
					Type: "restart",
					Parameters: map[string]interface{}{
						"service": "api",
					},
					Timeout: 3 * time.Minute,
				},
			},
			RollbackSteps: []PlaybookStep{
				{
					ID:   "rollback-1",
					Name: "恢复连接池配置",
					Type: "config",
					Parameters: map[string]interface{}{
						"key": "max_connections",
						"value": "100",
					},
					Timeout: 30 * time.Second,
				},
			},
			ApprovalRequired: true,
			Timeout:          10 * time.Minute,
			Enabled:          true,
			Priority:         2,
			Category:         "database",
		},
		"cache-miss": {
			ID:           "cache-miss",
			Name:         "缓存命中率低自动优化",
			Description:  "当缓存命中率过低时，自动预热缓存",
			TriggerConditions: []Condition{
				{Type: "metric", Metric: "cache_hit_rate", Operator: "<", Value: 70, Duration: 10 * time.Minute},
			},
			Steps: []PlaybookStep{
				{
					ID:   "step-1",
					Name: "检查缓存状态",
					Type: "command",
					Command: "redis-cli info stats",
					Timeout: 30 * time.Second,
				},
				{
					ID:   "step-2",
					Name: "预热热门数据",
					Type: "script",
					Script: "cache_preheat.py",
					Parameters: map[string]interface{}{
						"top_items": 1000,
					},
					Timeout: 5 * time.Minute,
				},
				{
					ID:   "step-3",
					Name: "优化缓存策略",
					Type: "config",
					Parameters: map[string]interface{}{
						"key": "cache_policy",
						"value": "lru",
					},
					Timeout: 30 * time.Second,
				},
			},
			ApprovalRequired: false,
			Timeout:          10 * time.Minute,
			Enabled:          true,
			Priority:         4,
			Category:         "performance",
		},
	}
}

func (r *AutoRemediation) RecommendActions(ctx context.Context, alert Alert) []RemediationAction {
	var actions []RemediationAction

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, playbook := range r.playbooks {
		if !playbook.Enabled {
			continue
		}

		if r.matchesAlert(alert, playbook) {
			for _, step := range playbook.Steps {
				action := RemediationAction{
					ID:          fmt.Sprintf("action-%s-%s", playbook.ID, step.ID),
					Type:        step.Type,
					Description: fmt.Sprintf("%s: %s", playbook.Name, step.Name),
					Priority:    playbook.Priority,
					Effort:      "medium",
					Risk:        r.assessRisk(step),
					Automated:   !playbook.ApprovalRequired,
					Command:     step.Command,
					Status:      "recommended",
				}
				actions = append(actions, action)
			}
		}
	}

	if len(actions) == 0 {
		actions = append(actions, RemediationAction{
			ID:          "action-manual-review",
			Type:        "manual",
			Description: "需要人工介入审查",
			Priority:    5,
			Effort:      "high",
			Risk:        "low",
			Automated:   false,
			Status:      "recommended",
		})
	}

	return actions
}

func (r *AutoRemediation) matchesAlert(alert Alert, playbook *RemediationPlaybook) bool {
	for _, condition := range playbook.TriggerConditions {
		if r.evaluateCondition(alert, condition) {
			return true
		}
	}
	return false
}

func (r *AutoRemediation) evaluateCondition(alert Alert, condition Condition) bool {
	alertLevel := map[string]float64{
		"critical": 100,
		"warning":  50,
		"info":     20,
	}

	alertValue := alertLevel[alert.Severity]
	conditionValue := r.convertToFloat64(condition.Value)

	switch condition.Operator {
	case ">":
		return alertValue > conditionValue
	case ">=":
		return alertValue >= conditionValue
	case "<":
		return alertValue < conditionValue
	case "<=":
		return alertValue <= conditionValue
	default:
		return false
	}
}

func (r *AutoRemediation) convertToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func (r *AutoRemediation) assessRisk(step PlaybookStep) string {
	highRiskTypes := map[string]bool{
		"restart":     true,
		"scale":       true,
		"delete":      true,
		"drop":        true,
		"truncate":    true,
	}

	mediumRiskTypes := map[string]bool{
		"config":      true,
		"update":      true,
		"modify":      true,
	}

	if highRiskTypes[step.Type] {
		return "high"
	}
	if mediumRiskTypes[step.Type] {
		return "medium"
	}
	return "low"
}

func (r *AutoRemediation) ExecuteAction(ctx context.Context, action RemediationAction) (*RemediationResult, error) {
	if !r.autoExecEnabled {
		return nil, fmt.Errorf("auto execution is disabled")
	}

	result := &RemediationResult{
		ActionID:   action.ID,
		Status:     "success",
		Message:    fmt.Sprintf("Action %s executed successfully", action.ID),
		Changes:    make(map[string]interface{}),
		ExecutedAt: time.Now(),
	}

	switch action.Type {
	case "command":
		result.Output = r.simulateCommandExecution(action.Command)
	case "notify":
		result.Output = r.simulateNotification(action.Description)
	case "restart":
		result.Output = r.simulateServiceRestart()
	case "scale":
		result.Changes["replicas"] = 2
		result.Output = "Service scaled successfully"
	case "config":
		result.Output = r.simulateConfigUpdate()
	case "script":
		result.Output = r.simulateScriptExecution()
	default:
		result.Output = "Action type not supported"
	}

	record := ExecutionRecord{
		ID:          fmt.Sprintf("exec-%d", time.Now().Unix()),
		PlaybookID:  "manual",
		AlertID:     action.ID,
		TriggeredAt: time.Now(),
		Status:      result.Status,
		Steps: []StepExecution{
			{
				StepID:    action.ID,
				StepName:  action.Description,
				Status:    "completed",
				StartedAt: time.Now().Add(-1 * time.Second),
				CompletedAt: func() *time.Time { t := time.Now(); return &t }(),
				Output:    result.Output,
			},
		},
		StartedAt: time.Now(),
	}

	completedAt := time.Now()
	record.CompletedAt = &completedAt

	r.executionHistory = append(r.executionHistory, record)

	return result, nil
}

func (r *AutoRemediation) simulateCommandExecution(command string) string {
	return fmt.Sprintf("Executed command: %s\nOutput: Command completed successfully", command)
}

func (r *AutoRemediation) simulateNotification(description string) string {
	return fmt.Sprintf("Notification sent: %s\nChannels: email, slack", description)
}

func (r *AutoRemediation) simulateServiceRestart() string {
	return "Service restart initiated\nWaiting for graceful shutdown...\nService stopped\nStarting service...\nService started successfully"
}

func (r *AutoRemediation) simulateConfigUpdate() string {
	return "Configuration updated\nRestarting affected services...\nServices restarted successfully"
}

func (r *AutoRemediation) simulateScriptExecution() string {
	return "Script execution started\nProcessing data...\nScript completed successfully"
}

func (r *AutoRemediation) GetPlaybooks(ctx context.Context) ([]*RemediationPlaybook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	playbooks := make([]*RemediationPlaybook, 0, len(r.playbooks))
	for _, playbook := range r.playbooks {
		playbooks = append(playbooks, playbook)
	}

	return playbooks, nil
}

func (r *AutoRemediation) GetPlaybook(ctx context.Context, playbookID string) (*RemediationPlaybook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	playbook, exists := r.playbooks[playbookID]
	if !exists {
		return nil, fmt.Errorf("playbook not found: %s", playbookID)
	}

	return playbook, nil
}

func (r *AutoRemediation) EnablePlaybook(ctx context.Context, playbookID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	playbook, exists := r.playbooks[playbookID]
	if !exists {
		return fmt.Errorf("playbook not found: %s", playbookID)
	}

	playbook.Enabled = true
	return nil
}

func (r *AutoRemediation) DisablePlaybook(ctx context.Context, playbookID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	playbook, exists := r.playbooks[playbookID]
	if !exists {
		return fmt.Errorf("playbook not found: %s", playbookID)
	}

	playbook.Enabled = false
	return nil
}

func (r *AutoRemediation) CreatePlaybook(ctx context.Context, playbook *RemediationPlaybook) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	playbook.ID = fmt.Sprintf("custom-%d", len(r.playbooks)+1)
	r.playbooks[playbook.ID] = playbook
	return nil
}

func (r *AutoRemediation) UpdatePlaybook(ctx context.Context, playbook *RemediationPlaybook) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.playbooks[playbook.ID]
	if !exists {
		return fmt.Errorf("playbook not found: %s", playbook.ID)
	}

	r.playbooks[playbook.ID] = playbook
	return nil
}

func (r *AutoRemediation) DeletePlaybook(ctx context.Context, playbookID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.playbooks[playbookID]
	if !exists {
		return fmt.Errorf("playbook not found: %s", playbookID)
	}

	delete(r.playbooks, playbookID)
	return nil
}

func (r *AutoRemediation) GetExecutionHistory(ctx context.Context, limit int) ([]ExecutionRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 || limit > len(r.executionHistory) {
		limit = len(r.executionHistory)
	}

	history := make([]ExecutionRecord, limit)
	copy(history, r.executionHistory[len(r.executionHistory)-limit:])

	return history, nil
}

func (r *AutoRemediation) GetTemplates(ctx context.Context) ([]RemediationTemplate, error) {
	templates := []RemediationTemplate{
		{
			ID:            "template-001",
			Name:          "自动扩容",
			Category:      "performance",
			Description:   "根据负载自动扩容服务实例",
			UseCases:      []string{"高CPU", "高内存", "高负载"},
			Difficulty:    "medium",
			EstimatedTime: 5 * time.Minute,
		},
		{
			ID:            "template-002",
			Name:          "服务重启",
			Category:      "reliability",
			Description:   "自动重启故障服务",
			UseCases:      []string{"服务无响应", "内存泄漏"},
			Difficulty:    "low",
			EstimatedTime: 3 * time.Minute,
		},
		{
			ID:            "template-003",
			Name:          "缓存优化",
			Category:      "performance",
			Description:   "优化缓存策略和预热",
			UseCases:      []string{"缓存命中率低"},
			Difficulty:    "medium",
			EstimatedTime: 10 * time.Minute,
		},
		{
			ID:            "template-004",
			Name:          "日志清理",
			Category:      "resource",
			Description:   "清理过期日志文件",
			UseCases:      []string{"磁盘空间不足"},
			Difficulty:    "low",
			EstimatedTime: 5 * time.Minute,
		},
		{
			ID:            "template-005",
			Name:          "告警通知",
			Category:      "notification",
			Description:   "发送告警通知到多个渠道",
			UseCases:      []string{"所有故障场景"},
			Difficulty:    "low",
			EstimatedTime: 1 * time.Minute,
		},
	}

	return templates, nil
}

func (r *AutoRemediation) SetAutoExecution(ctx context.Context, enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.autoExecEnabled = enabled
	return nil
}

func (r *AutoRemediation) IsAutoExecutionEnabled(ctx context.Context) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.autoExecEnabled, nil
}

func (r *AutoRemediation) ExecutePlaybook(ctx context.Context, playbookID string, alertID string) (*ExecutionRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	playbook, exists := r.playbooks[playbookID]
	if !exists {
		return nil, fmt.Errorf("playbook not found: %s", playbookID)
	}

	if !playbook.Enabled {
		return nil, fmt.Errorf("playbook is disabled")
	}

	record := ExecutionRecord{
		ID:           fmt.Sprintf("exec-%d", time.Now().Unix()),
		PlaybookID:   playbookID,
		AlertID:      alertID,
		TriggeredAt:  time.Now(),
		Status:       "running",
		Steps:        make([]StepExecution, 0),
		StartedAt:    time.Now(),
	}

	for _, step := range playbook.Steps {
		stepExec := StepExecution{
			StepID:    step.ID,
			StepName:  step.Name,
			Status:    "running",
			StartedAt: time.Now(),
		}

		time.Sleep(100 * time.Millisecond)

		completedAt := time.Now()
		stepExec.Status = "completed"
		stepExec.CompletedAt = &completedAt
		stepExec.Output = fmt.Sprintf("Step %s executed", step.Name)

		record.Steps = append(record.Steps, stepExec)
	}

	record.Status = "completed"
	completedAt := time.Now()
	record.CompletedAt = &completedAt

	r.executionHistory = append(r.executionHistory, record)

	if len(r.executionHistory) > 100 {
		r.executionHistory = r.executionHistory[1:]
	}

	return &record, nil
}

func (r *AutoRemediation) RollbackExecution(ctx context.Context, executionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var record *ExecutionRecord
	for i := len(r.executionHistory) - 1; i >= 0; i-- {
		if r.executionHistory[i].ID == executionID {
			record = &r.executionHistory[i]
			break
		}
	}

	if record == nil {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	playbook, exists := r.playbooks[record.PlaybookID]
	if !exists {
		return fmt.Errorf("playbook not found: %s", record.PlaybookID)
	}

	if len(playbook.RollbackSteps) == 0 {
		return fmt.Errorf("no rollback steps defined")
	}

	record.RollbackRequired = true
	record.RollbackStatus = "rolling_back"

	time.Sleep(100 * time.Millisecond)

	record.RollbackStatus = "completed"

	return nil
}

func (r *AutoRemediation) ApproveExecution(ctx context.Context, executionID string, approver string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := len(r.executionHistory) - 1; i >= 0; i-- {
		if r.executionHistory[i].ID == executionID {
			r.executionHistory[i].ApprovedBy = approver
			now := time.Now()
			r.executionHistory[i].ApprovedAt = &now
			return nil
		}
	}

	return fmt.Errorf("execution not found: %s", executionID)
}

func (r *AutoRemediation) GetExecutionStats(ctx context.Context) (*ExecutionStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &ExecutionStats{
		TotalExecutions: len(r.executionHistory),
		SuccessCount:    0,
		FailureCount:    0,
		PendingCount:    0,
	}

	for _, record := range r.executionHistory {
		switch record.Status {
		case "completed":
			stats.SuccessCount++
		case "failed":
			stats.FailureCount++
		case "running", "pending":
			stats.PendingCount++
		}
	}

	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalExecutions) * 100
	}

	return stats, nil
}

type ExecutionStats struct {
	TotalExecutions int     `json:"total_executions"`
	SuccessCount    int     `json:"success_count"`
	FailureCount    int     `json:"failure_count"`
	PendingCount    int     `json:"pending_count"`
	SuccessRate     float64 `json:"success_rate"`
}

func (r *AutoRemediation) ExportPlaybooks(ctx context.Context, format string) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	playbooks := make([]*RemediationPlaybook, 0, len(r.playbooks))
	for _, playbook := range r.playbooks {
		playbooks = append(playbooks, playbook)
	}

	_ = struct {
		ExportTime time.Time               `json:"export_time"`
		Playbooks  []*RemediationPlaybook `json:"playbooks"`
	}{
		ExportTime: time.Now(),
		Playbooks:  playbooks,
	}

	return []byte(fmt.Sprintf("Exported %d playbooks at %s", len(playbooks), time.Now().Format(time.RFC3339))), nil
}

func (r *AutoRemediation) GetPlaybookByCategory(ctx context.Context, category string) ([]*RemediationPlaybook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var playbooks []*RemediationPlaybook
	for _, playbook := range r.playbooks {
		if playbook.Category == category {
			playbooks = append(playbooks, playbook)
		}
	}

	return playbooks, nil
}

func (r *AutoRemediation) GetEnabledPlaybooks(ctx context.Context) ([]*RemediationPlaybook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var playbooks []*RemediationPlaybook
	for _, playbook := range r.playbooks {
		if playbook.Enabled {
			playbooks = append(playbooks, playbook)
		}
	}

	return playbooks, nil
}

func (r *AutoRemediation) ValidatePlaybook(ctx context.Context, playbook *RemediationPlaybook) ([]string, error) {
	var errors []string

	if playbook.Name == "" {
		errors = append(errors, "playbook name is required")
	}

	if len(playbook.Steps) == 0 {
		errors = append(errors, "playbook must have at least one step")
	}

	for i, step := range playbook.Steps {
		if step.ID == "" {
			errors = append(errors, fmt.Sprintf("step %d: ID is required", i))
		}
		if step.Name == "" {
			errors = append(errors, fmt.Sprintf("step %d: name is required", i))
		}
		if step.Type == "" {
			errors = append(errors, fmt.Sprintf("step %d: type is required", i))
		}
	}

	if playbook.Timeout <= 0 {
		errors = append(errors, "timeout must be positive")
	}

	return errors, nil
}
