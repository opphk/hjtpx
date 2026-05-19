package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

type BudgetAlert struct {
	mu       sync.RWMutex
	budgets  map[string]*Budget
	alerts   map[string][]*BudgetAlertRecord
	channels []AlertChannel
}

type Budget struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Amount           float64           `json:"amount"`
	Period           string            `json:"period"`
	AlertThresholds  []AlertThreshold  `json:"alert_thresholds"`
	CurrentSpent     float64           `json:"current_spent"`
	StartDate        time.Time         `json:"start_date"`
	EndDate          time.Time         `json:"end_date"`
	Notifications    []Notification    `json:"notifications"`
	AutoActions      []AutoAction      `json:"auto_actions"`
	Enabled          bool              `json:"enabled"`
	Scope            *BudgetScope      `json:"scope"`
	CostModel        *CostAllocation   `json:"cost_model"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type AlertThreshold struct {
	Percentage float64 `json:"percentage"`
	Level      string  `json:"level"`
	Message    string  `json:"message"`
	Color      string  `json:"color"`
}

type BudgetScope struct {
	Projects    []string `json:"projects,omitempty"`
	Services    []string `json:"services,omitempty"`
	Regions     []string `json:"regions,omitempty"`
	Environment string   `json:"environment,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

type Notification struct {
	Channel     string   `json:"channel"`
	Recipients  []string `json:"recipients"`
	Template    string   `json:"template"`
	Enabled     bool     `json:"enabled"`
}

type AutoAction struct {
	Type        string                 `json:"type"`
	Threshold   float64                `json:"threshold"`
	Parameters  map[string]interface{} `json:"parameters"`
	Enabled     bool                   `json:"enabled"`
	LastTriggered *time.Time          `json:"last_triggered,omitempty"`
}

type BudgetAlertRecord struct {
	ID           string    `json:"id"`
	BudgetID     string    `json:"budget_id"`
	AlertLevel   string    `json:"alert_level"`
	Threshold    float64   `json:"threshold"`
	ActualUsage  float64   `json:"actual_usage"`
	Percentage   float64   `json:"percentage"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
	Acknowledged bool      `json:"acknowledged"`
	AcknowledgedBy string `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	Actions     []string   `json:"actions,omitempty"`
}

type BudgetStatus struct {
	BudgetID        string          `json:"budget_id"`
	BudgetName      string          `json:"budget_name"`
	CurrentSpent    float64         `json:"current_spent"`
	BudgetAmount    float64         `json:"budget_amount"`
	Remaining       float64         `json:"remaining"`
	Percentage      float64         `json:"percentage"`
	DailyAllowance  float64         `json:"daily_allowance"`
	DaysRemaining   int             `json:"days_remaining"`
	ProjectedSpend  float64         `json:"projected_spend"`
	OverBudget      bool            `json:"over_budget"`
	AlertLevel      string          `json:"alert_level"`
	Trend           string          `json:"trend"`
	Recommendations []string        `json:"recommendations"`
}

type AlertChannel struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
}

type BudgetReport struct {
	Period         string               `json:"period"`
	StartDate      time.Time            `json:"start_date"`
	EndDate        time.Time            `json:"end_date"`
	TotalBudget    float64              `json:"total_budget"`
	TotalSpent     float64              `json:"total_spent"`
	TotalRemaining float64              `json:"total_remaining"`
	ByCategory     map[string]float64   `json:"by_category"`
	ByProject      map[string]float64   `json:"by_project"`
	Alerts         []*BudgetAlertRecord `json:"alerts"`
	ComplianceRate float64              `json:"compliance_rate"`
}

func NewBudgetAlert() *BudgetAlert {
	alert := &BudgetAlert{
		budgets:  make(map[string]*Budget),
		alerts:   make(map[string][]*BudgetAlertRecord),
		channels: make([]AlertChannel, 0),
	}
	alert.initializeChannels()
	alert.initializeDefaultBudgets()
	go alert.startMonitoring()
	return alert
}

func (b *BudgetAlert) initializeChannels() {
	b.channels = []AlertChannel{
		{ID: "email", Type: "email", Name: "邮件通知", Enabled: true, Endpoint: "smtp://localhost"},
		{ID: "slack", Type: "slack", Name: "Slack通知", Enabled: true, Endpoint: "https://hooks.slack.com/"},
		{ID: "webhook", Type: "webhook", Name: "Webhook", Enabled: true, Endpoint: "https://api.example.com/webhook"},
		{ID: "sms", Type: "sms", Name: "短信通知", Enabled: false, Endpoint: ""},
	}
}

func (b *BudgetAlert) initializeDefaultBudgets() {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	b.budgets = map[string]*Budget{
		"monthly-total": {
			ID:       "monthly-total",
			Name:     "月度总预算",
			Amount:   5000.0,
			Period:   "monthly",
			AlertThresholds: []AlertThreshold{
				{Percentage: 50, Level: "info", Message: "预算使用达到50%", Color: "green"},
				{Percentage: 75, Level: "warning", Message: "预算使用达到75%", Color: "yellow"},
				{Percentage: 90, Level: "critical", Message: "预算使用达到90%", Color: "orange"},
				{Percentage: 100, Level: "exceeded", Message: "预算已超支", Color: "red"},
			},
			CurrentSpent: 2500.0,
			StartDate:     startOfMonth,
			EndDate:       endOfMonth,
			Notifications: []Notification{
				{Channel: "email", Recipients: []string{"admin@example.com"}, Enabled: true},
				{Channel: "slack", Recipients: []string{"#cost-alerts"}, Enabled: true},
			},
			AutoActions: []AutoAction{
				{Type: "scale_down", Threshold: 80, Parameters: map[string]interface{}{"percentage": 20}, Enabled: true},
				{Type: "notify", Threshold: 90, Parameters: map[string]interface{}{"channels": []string{"email", "slack"}}, Enabled: true},
			},
			Enabled: true,
			Scope:   &BudgetScope{Environment: "production"},
			CreatedAt: now,
			UpdatedAt: now,
		},
		"monthly-compute": {
			ID:       "monthly-compute",
			Name:     "月度计算预算",
			Amount:   2000.0,
			Period:   "monthly",
			AlertThresholds: []AlertThreshold{
				{Percentage: 60, Level: "warning", Message: "计算费用达到60%", Color: "yellow"},
				{Percentage: 80, Level: "critical", Message: "计算费用达到80%", Color: "orange"},
			},
			CurrentSpent: 800.0,
			StartDate:     startOfMonth,
			EndDate:       endOfMonth,
			Enabled:       true,
			Scope:         &BudgetScope{Services: []string{"compute", "api"}},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		"monthly-storage": {
			ID:       "monthly-storage",
			Name:     "月度存储预算",
			Amount:   1000.0,
			Period:   "monthly",
			AlertThresholds: []AlertThreshold{
				{Percentage: 70, Level: "warning", Message: "存储费用达到70%", Color: "yellow"},
				{Percentage: 85, Level: "critical", Message: "存储费用达到85%", Color: "orange"},
			},
			CurrentSpent: 350.0,
			StartDate:     startOfMonth,
			EndDate:       endOfMonth,
			Enabled:       true,
			Scope:         &BudgetScope{Services: []string{"storage", "database"}},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
}

func (b *BudgetAlert) startMonitoring() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		b.checkAllBudgets()
	}
}

func (b *BudgetAlert) checkAllBudgets() {
	b.mu.Lock()
	defer b.mu.Unlock()

	_ = time.Now()

	for _, budget := range b.budgets {
		if !budget.Enabled {
			continue
		}

		percentage := (budget.CurrentSpent / budget.Amount) * 100

		for _, threshold := range budget.AlertThresholds {
			if percentage >= threshold.Percentage {
				b.triggerAlert(budget, threshold, percentage)
			}
		}
	}
}

func (b *BudgetAlert) triggerAlert(budget *Budget, threshold AlertThreshold, percentage float64) {
	budgetID := budget.ID

	if b.alerts[budgetID] == nil {
		b.alerts[budgetID] = make([]*BudgetAlertRecord, 0)
	}

	if len(b.alerts[budgetID]) > 0 {
		lastAlert := b.alerts[budgetID][len(b.alerts[budgetID])-1]
		if lastAlert.Percentage == percentage && time.Since(lastAlert.Timestamp) < 24*time.Hour {
			return
		}
	}

	record := &BudgetAlertRecord{
		ID:          fmt.Sprintf("alert-%s-%d", budgetID, time.Now().Unix()),
		BudgetID:    budgetID,
		AlertLevel:  threshold.Level,
		Threshold:   threshold.Percentage,
		ActualUsage: budget.CurrentSpent,
		Percentage:  percentage,
		Message:     threshold.Message,
		Timestamp:   time.Now(),
	}

	b.alerts[budgetID] = append(b.alerts[budgetID], record)

	b.sendNotifications(budget, record)

	b.executeAutoActions(budget, threshold)
}

func (b *BudgetAlert) sendNotifications(budget *Budget, record *BudgetAlertRecord) {
	for _, notification := range budget.Notifications {
		if !notification.Enabled {
			continue
		}

		fmt.Printf("Sending notification via %s to %v: %s\n", notification.Channel, notification.Recipients, record.Message)
	}
}

func (b *BudgetAlert) executeAutoActions(budget *Budget, threshold AlertThreshold) {
	for _, action := range budget.AutoActions {
		if !action.Enabled {
			continue
		}

		if threshold.Percentage >= action.Threshold {
			b.executeAction(action)
			now := time.Now()
			action.LastTriggered = &now
		}
	}
}

func (b *BudgetAlert) executeAction(action AutoAction) {
	switch action.Type {
	case "scale_down":
		fmt.Printf("Executing auto action: scale_down by %v%%\n", action.Parameters["percentage"])
	case "notify":
		channels := action.Parameters["channels"]
		fmt.Printf("Executing auto action: notify via %v\n", channels)
	case "stop_resources":
		fmt.Printf("Executing auto action: stop_resources\n")
	case "alert_team":
		fmt.Printf("Executing auto action: alert_team\n")
	}
}

func (b *BudgetAlert) GetBudgetStatus(ctx context.Context) (*BudgetStatus, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	now := time.Now()

	for _, budget := range b.budgets {
		if budget.Period != "monthly" {
			continue
		}

		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endOfMonth := startOfMonth.AddDate(0, 1, 0)

		daysInMonth := endOfMonth.Day()
		daysElapsed := now.Day()
		daysRemaining := daysInMonth - daysElapsed

		dailyAllowance := budget.Amount / float64(daysInMonth)

		percentage := (budget.CurrentSpent / budget.Amount) * 100

		remaining := budget.Amount - budget.CurrentSpent
		if remaining < 0 {
			remaining = 0
		}

		var projectedSpend float64
		if daysElapsed > 0 {
			avgDailyCost := budget.CurrentSpent / float64(daysElapsed)
			projectedSpend = avgDailyCost * float64(daysInMonth)
		}

		alertLevel := "normal"
		if percentage >= 100 {
			alertLevel = "exceeded"
		} else if percentage >= 90 {
			alertLevel = "critical"
		} else if percentage >= 75 {
			alertLevel = "warning"
		} else if percentage >= 50 {
			alertLevel = "info"
		}

		trend := "stable"
		if projectedSpend > budget.Amount {
			trend = "overspending"
		} else if projectedSpend < budget.Amount*0.8 {
			trend = "under_spending"
		}

		var recommendations []string
		if percentage > 80 {
			recommendations = append(recommendations, "考虑优化资源使用")
		}
		if projectedSpend > budget.Amount {
			recommendations = append(recommendations, "预测将超出预算，建议采取节流措施")
		}
		if trend == "under_spending" {
			recommendations = append(recommendations, "当前支出低于预期，可考虑升级服务")
		}

		return &BudgetStatus{
			BudgetID:        budget.ID,
			BudgetName:      budget.Name,
			CurrentSpent:    budget.CurrentSpent,
			BudgetAmount:    budget.Amount,
			Remaining:       remaining,
			Percentage:      percentage,
			DailyAllowance:  dailyAllowance,
			DaysRemaining:   daysRemaining,
			ProjectedSpend:  projectedSpend,
			OverBudget:      percentage >= 100,
			AlertLevel:      alertLevel,
			Trend:           trend,
			Recommendations: recommendations,
		}, nil
	}

	return nil, fmt.Errorf("no monthly budget found")
}

func (b *BudgetAlert) GetAllBudgetStatus(ctx context.Context) ([]*BudgetStatus, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var statuses []*BudgetStatus
	now := time.Now()

	for _, budget := range b.budgets {
		if !budget.Enabled {
			continue
		}

		percentage := (budget.CurrentSpent / budget.Amount) * 100
		remaining := budget.Amount - budget.CurrentSpent
		if remaining < 0 {
			remaining = 0
		}

		daysRemaining := int(budget.EndDate.Sub(now).Hours() / 24)
		if daysRemaining < 0 {
			daysRemaining = 0
		}

		dailyAllowance := remaining / float64(math.Max(1, float64(daysRemaining)))

		var projectedSpend float64
		if len(b.alerts[budget.ID]) > 0 {
			avgDailyCost := budget.CurrentSpent / float64(time.Since(budget.StartDate).Hours()/24+1)
			totalDays := budget.EndDate.Sub(budget.StartDate).Hours() / 24
			projectedSpend = avgDailyCost * totalDays
		}

		alertLevel := "normal"
		if percentage >= 100 {
			alertLevel = "exceeded"
		} else if percentage >= 90 {
			alertLevel = "critical"
		} else if percentage >= 75 {
			alertLevel = "warning"
		} else if percentage >= 50 {
			alertLevel = "info"
		}

		statuses = append(statuses, &BudgetStatus{
			BudgetID:       budget.ID,
			BudgetName:     budget.Name,
			CurrentSpent:   budget.CurrentSpent,
			BudgetAmount:   budget.Amount,
			Remaining:      remaining,
			Percentage:     percentage,
			DailyAllowance: dailyAllowance,
			DaysRemaining:  daysRemaining,
			ProjectedSpend: projectedSpend,
			OverBudget:     percentage >= 100,
			AlertLevel:     alertLevel,
		})
	}

	return statuses, nil
}

func (b *BudgetAlert) CreateBudget(ctx context.Context, budget *Budget) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if budget.ID == "" {
		budget.ID = fmt.Sprintf("budget-%d", len(b.budgets)+1)
	}

	budget.CreatedAt = time.Now()
	budget.UpdatedAt = time.Now()

	b.budgets[budget.ID] = budget
	return nil
}

func (b *BudgetAlert) UpdateBudget(ctx context.Context, budget *Budget) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	existing, exists := b.budgets[budget.ID]
	if !exists {
		return fmt.Errorf("budget not found: %s", budget.ID)
	}

	budget.CreatedAt = existing.CreatedAt
	budget.UpdatedAt = time.Now()

	b.budgets[budget.ID] = budget
	return nil
}

func (b *BudgetAlert) DeleteBudget(ctx context.Context, budgetID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.budgets[budgetID]; !exists {
		return fmt.Errorf("budget not found: %s", budgetID)
	}

	delete(b.budgets, budgetID)
	delete(b.alerts, budgetID)

	return nil
}

func (b *BudgetAlert) GetBudget(ctx context.Context, budgetID string) (*Budget, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	budget, exists := b.budgets[budgetID]
	if !exists {
		return nil, fmt.Errorf("budget not found: %s", budgetID)
	}

	return budget, nil
}

func (b *BudgetAlert) GetAllBudgets(ctx context.Context) ([]*Budget, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	budgets := make([]*Budget, 0, len(b.budgets))
	for _, budget := range b.budgets {
		budgets = append(budgets, budget)
	}

	sort.Slice(budgets, func(i, j int) bool {
		return budgets[i].Name < budgets[j].Name
	})

	return budgets, nil
}

func (b *BudgetAlert) GetAlerts(ctx context.Context, budgetID string, limit int) ([]*BudgetAlertRecord, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	alerts := b.alerts[budgetID]
	if len(alerts) <= limit {
		return alerts, nil
	}

	return alerts[len(alerts)-limit:], nil
}

func (b *BudgetAlert) AcknowledgeAlert(ctx context.Context, budgetID string, alertID string, acknowledgedBy string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	alerts := b.alerts[budgetID]
	for _, alert := range alerts {
		if alert.ID == alertID {
			alert.Acknowledged = true
			alert.AcknowledgedBy = acknowledgedBy
			now := time.Now()
			alert.AcknowledgedAt = &now
			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", alertID)
}

func (b *BudgetAlert) UpdateBudgetSpending(ctx context.Context, budgetID string, spent float64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	budget, exists := b.budgets[budgetID]
	if !exists {
		return fmt.Errorf("budget not found: %s", budgetID)
	}

	budget.CurrentSpent = spent
	budget.UpdatedAt = time.Now()

	return nil
}

func (b *BudgetAlert) GetChannels(ctx context.Context) ([]AlertChannel, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.channels, nil
}

func (b *BudgetAlert) EnableChannel(ctx context.Context, channelID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := range b.channels {
		if b.channels[i].ID == channelID {
			b.channels[i].Enabled = true
			return nil
		}
	}

	return fmt.Errorf("channel not found: %s", channelID)
}

func (b *BudgetAlert) DisableChannel(ctx context.Context, channelID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := range b.channels {
		if b.channels[i].ID == channelID {
			b.channels[i].Enabled = false
			return nil
		}
	}

	return fmt.Errorf("channel not found: %s", channelID)
}

func (b *BudgetAlert) GetBudgetReport(ctx context.Context, period string) (*BudgetReport, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	now := time.Now()
	var startDate, endDate time.Time

	switch period {
	case "monthly":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, 0)
	case "weekly":
		startDate = now.AddDate(0, 0, -int(now.Weekday()))
		endDate = startDate.AddDate(0, 0, 7)
	case "quarterly":
		quarter := (int(now.Month()) - 1) / 3
		startDate = time.Date(now.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 3, 0)
	default:
		return nil, fmt.Errorf("invalid period: %s", period)
	}

	var totalBudget, totalSpent float64
	byCategory := make(map[string]float64)
	byProject := make(map[string]float64)
	var allAlerts []*BudgetAlertRecord

	for _, budget := range b.budgets {
		if budget.Period == period {
			totalBudget += budget.Amount
			totalSpent += budget.CurrentSpent

			byCategory[budget.Name] = budget.CurrentSpent

			if budget.Scope != nil && len(budget.Scope.Projects) > 0 {
				for _, project := range budget.Scope.Projects {
					byProject[project] += budget.CurrentSpent
				}
			}

			if alerts, exists := b.alerts[budget.ID]; exists {
				allAlerts = append(allAlerts, alerts...)
			}
		}
	}

	totalRemaining := totalBudget - totalSpent
	if totalRemaining < 0 {
		totalRemaining = 0
	}

	complianceRate := 0.0
	if totalBudget > 0 {
		complianceRate = (1 - totalSpent/totalBudget) * 100
		if complianceRate < 0 {
			complianceRate = 0
		}
	}

	return &BudgetReport{
		Period:          period,
		StartDate:       startDate,
		EndDate:         endDate,
		TotalBudget:     totalBudget,
		TotalSpent:      totalSpent,
		TotalRemaining:  totalRemaining,
		ByCategory:      byCategory,
		ByProject:       byProject,
		Alerts:          allAlerts,
		ComplianceRate:  complianceRate,
	}, nil
}

func (b *BudgetAlert) ExportBudgetData(ctx context.Context, format string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var data string

	data += "Budget Alert Report\n"
	data += fmt.Sprintf("Generated at: %s\n\n", time.Now().Format(time.RFC3339))

	data += "Budgets:\n"
	for _, budget := range b.budgets {
		percentage := (budget.CurrentSpent / budget.Amount) * 100
		data += fmt.Sprintf("  %s: $%.2f / $%.2f (%.1f%%)\n", budget.Name, budget.CurrentSpent, budget.Amount, percentage)
	}

	data += "\nAlerts:\n"
	for budgetID, alerts := range b.alerts {
		data += fmt.Sprintf("  Budget: %s\n", budgetID)
		for _, alert := range alerts {
			data += fmt.Sprintf("    - [%s] %s (%.1f%%)\n", alert.AlertLevel, alert.Message, alert.Percentage)
		}
	}

	return []byte(data), nil
}

func (b *BudgetAlert) GetSpendingForecast(ctx context.Context, budgetID string) (*SpendingForecast, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	budget, exists := b.budgets[budgetID]
	if !exists {
		return nil, fmt.Errorf("budget not found: %s", budgetID)
	}

	now := time.Now()
	elapsedHours := now.Sub(budget.StartDate).Hours()
	totalHours := budget.EndDate.Sub(budget.StartDate).Hours()

	avgHourlyCost := 0.0
	if elapsedHours > 0 {
		avgHourlyCost = budget.CurrentSpent / elapsedHours
	}

	forecast := &SpendingForecast{
		BudgetID:        budgetID,
		CurrentSpent:    budget.CurrentSpent,
		ProjectedSpend:  avgHourlyCost * totalHours,
		DaysRemaining:   int(budget.EndDate.Sub(now).Hours() / 24),
		RemainingBudget: budget.Amount - budget.CurrentSpent,
		OnTrack:         true,
	}

	if forecast.ProjectedSpend > budget.Amount {
		forecast.OnTrack = false
		forecast.Shortfall = forecast.ProjectedSpend - budget.Amount
	}

	return forecast, nil
}

type SpendingForecast struct {
	BudgetID        string  `json:"budget_id"`
	CurrentSpent    float64 `json:"current_spent"`
	ProjectedSpend  float64 `json:"projected_spend"`
	DaysRemaining   int     `json:"days_remaining"`
	RemainingBudget float64 `json:"remaining_budget"`
	OnTrack         bool    `json:"on_track"`
	Shortfall       float64 `json:"shortfall,omitempty"`
}

func (b *BudgetAlert) SetBudgetAmount(ctx context.Context, budgetID string, amount float64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	budget, exists := b.budgets[budgetID]
	if !exists {
		return fmt.Errorf("budget not found: %s", budgetID)
	}

	budget.Amount = amount
	budget.UpdatedAt = time.Now()

	return nil
}

func (b *BudgetAlert) AddThreshold(ctx context.Context, budgetID string, threshold AlertThreshold) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	budget, exists := b.budgets[budgetID]
	if !exists {
		return fmt.Errorf("budget not found: %s", budgetID)
	}

	budget.AlertThresholds = append(budget.AlertThresholds, threshold)
	budget.UpdatedAt = time.Now()

	return nil
}

func (b *BudgetAlert) RemoveThreshold(ctx context.Context, budgetID string, percentage float64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	budget, exists := b.budgets[budgetID]
	if !exists {
		return fmt.Errorf("budget not found: %s", budgetID)
	}

	for i, threshold := range budget.AlertThresholds {
		if threshold.Percentage == percentage {
			budget.AlertThresholds = append(budget.AlertThresholds[:i], budget.AlertThresholds[i+1:]...)
			budget.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("threshold not found: %.1f%%", percentage)
}

func (b *BudgetAlert) EnableBudget(ctx context.Context, budgetID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	budget, exists := b.budgets[budgetID]
	if !exists {
		return fmt.Errorf("budget not found: %s", budgetID)
	}

	budget.Enabled = true
	budget.UpdatedAt = time.Now()

	return nil
}

func (b *BudgetAlert) DisableBudget(ctx context.Context, budgetID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	budget, exists := b.budgets[budgetID]
	if !exists {
		return fmt.Errorf("budget not found: %s", budgetID)
	}

	budget.Enabled = false
	budget.UpdatedAt = time.Now()

	return nil
}
