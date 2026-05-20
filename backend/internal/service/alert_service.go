package service

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type AlertService struct {
	rules      []*AlertRule
	channels   []AlertChannel
	alerts     map[string]*Alert
	history    []*AlertHistory
	notifiers  map[string]Notifier
	mu         sync.RWMutex
	stopChan   chan struct{}
	config     *AlertConfig
}

type AlertConfig struct {
	EnableAlerting       bool
	AlertRetention       int
	CheckInterval        int
	MaxAlertsPerMinute   int
	EnableDeduplication  bool
	DeduplicationWindow  int
}

type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Condition   *AlertCondition        `json:"condition"`
	Actions     []*AlertAction         `json:"actions"`
	Enabled     bool                  `json:"enabled"`
	Severity    string                 `json:"severity"`
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]string      `json:"annotations"`
	Matchings   int                    `json:"matchings"`
	LastFired   time.Time              `json:"last_fired"`
}

type AlertCondition struct {
	Metric     string   `json:"metric"`
	Operator   string   `json:"operator"`
	Threshold  float64  `json:"threshold"`
	Duration   int     `json:"duration"`
	Comparison string   `json:"comparison"`
}

type AlertAction struct {
	Type       string                 `json:"type"`
	Channel    string                 `json:"channel"`
	Template   string                 `json:"template"`
	Settings   map[string]interface{}  `json:"settings"`
}

type AlertChannel struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Enabled   bool                   `json:"enabled"`
	Settings  map[string]interface{}  `json:"settings"`
}

type Notifier interface {
	Send(alert *Alert) error
	Close() error
}

type AlertHistory struct {
	AlertID     string    `json:"alert_id"`
	Name        string    `json:"name"`
	Severity    string    `json:"severity"`
	Status      string    `json:"status"`
	FiredAt     time.Time `json:"fired_at"`
	ResolvedAt  time.Time `json:"resolved_at,omitempty"`
	Duration    int64     `json:"duration_seconds"`
	Count       int       `json:"count"`
}

type AlertNotification struct {
	ID          string                 `json:"id"`
	AlertID     string                 `json:"alert_id"`
	Channel     string                 `json:"channel"`
	SentAt      time.Time              `json:"sent_at"`
	Status      string                 `json:"status"`
	Error       string                 `json:"error,omitempty"`
}

func NewAlertService() *AlertService {
	return &AlertService{
		rules:     make([]*AlertRule, 0),
		channels:  make([]AlertChannel, 0),
		alerts:    make(map[string]*Alert),
		history:   make([]*AlertHistory, 0),
		notifiers: make(map[string]Notifier),
		stopChan:  make(chan struct{}),
		config: &AlertConfig{
			EnableAlerting:      true,
			AlertRetention:      1000,
			CheckInterval:       30,
			MaxAlertsPerMinute:  100,
			EnableDeduplication: true,
			DeduplicationWindow: 300,
		},
	}
}

func (s *AlertService) Start() {
	if !s.config.EnableAlerting {
		log.Println("Alerting is disabled")
		return
	}

	go s.ruleChecker()
	log.Println("Alert service started")
}

func (s *AlertService) Stop() {
	close(s.stopChan)
	for _, notifier := range s.notifiers {
		notifier.Close()
	}
	log.Println("Alert service stopped")
}

func (s *AlertService) CreateRule(rule *AlertRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule-%d-%s", time.Now().UnixNano(), rule.Name)
	}
	if rule.Labels == nil {
		rule.Labels = make(map[string]string)
	}
	if rule.Annotations == nil {
		rule.Annotations = make(map[string]string)
	}

	s.rules = append(s.rules, rule)
	log.Printf("Alert rule created: %s", rule.Name)
	return nil
}

func (s *AlertService) UpdateRule(ruleID string, rule *AlertRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.rules {
		if r.ID == ruleID {
			rule.ID = ruleID
			s.rules[i] = rule
			return nil
		}
	}
	return fmt.Errorf("rule not found")
}

func (s *AlertService) DeleteRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.rules {
		if r.ID == ruleID {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("rule not found")
}

func (s *AlertService) GetRules() []*AlertRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*AlertRule, len(s.rules))
	copy(rules, s.rules)
	return rules
}

func (s *AlertService) EnableRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rule := range s.rules {
		if rule.ID == ruleID {
			rule.Enabled = true
			return nil
		}
	}
	return fmt.Errorf("rule not found")
}

func (s *AlertService) DisableRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rule := range s.rules {
		if rule.ID == ruleID {
			rule.Enabled = false
			return nil
		}
	}
	return fmt.Errorf("rule not found")
}

func (s *AlertService) ruleChecker() {
	ticker := time.NewTicker(time.Duration(s.config.CheckInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkRules()
		}
	}
}

func (s *AlertService) checkRules() {
	s.mu.RLock()
	rules := s.rules
	s.mu.RUnlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		if s.evaluateCondition(rule.Condition) {
			s.fireAlert(rule)
		}
	}
}

func (s *AlertService) evaluateCondition(condition *AlertCondition) bool {
	return true
}

func (s *AlertService) fireAlert(rule *AlertRule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	alertKey := rule.ID

	if existing, ok := s.alerts[alertKey]; ok {
		if existing.Status == "firing" {
			return
		}
	}

	alert := &Alert{
		ID:          fmt.Sprintf("alert-%d-%s", time.Now().UnixNano(), rule.Name),
		Name:        rule.Name,
		Severity:    rule.Severity,
		Message:     rule.Description,
		Source:      "alert_rule",
		Timestamp:   time.Now(),
		Status:      "firing",
		Labels:      rule.Labels,
		Annotations: rule.Annotations,
	}

	s.alerts[alertKey] = alert
	rule.Matchings++
	rule.LastFired = time.Now()

	history := &AlertHistory{
		AlertID:  alert.ID,
		Name:     rule.Name,
		Severity: rule.Severity,
		Status:   "firing",
		FiredAt:  time.Now(),
		Count:    rule.Matchings,
	}
	s.history = append(s.history, history)

	if len(s.history) > s.config.AlertRetention {
		s.history = s.history[len(s.history)-s.config.AlertRetention:]
	}

	go s.executeActions(rule, alert)
}

func (s *AlertService) executeActions(rule *AlertRule, alert *Alert) {
	for _, action := range rule.Actions {
		if action.Type == "notification" {
			s.sendNotification(action, alert)
		}
	}
}

func (s *AlertService) sendNotification(action *AlertAction, alert *Alert) {
	s.mu.RLock()
	notifier, ok := s.notifiers[action.Channel]
	s.mu.RUnlock()

	if !ok {
		log.Printf("Notifier not found for channel: %s", action.Channel)
		return
	}

	if err := notifier.Send(alert); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

func (s *AlertService) RegisterChannel(channel AlertChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.channels {
		if c.ID == channel.ID {
			return fmt.Errorf("channel already exists")
		}
	}

	s.channels = append(s.channels, channel)

	notifier, err := s.createNotifier(channel)
	if err != nil {
		return err
	}

	s.notifiers[channel.ID] = notifier
	return nil
}

func (s *AlertService) createNotifier(channel AlertChannel) (Notifier, error) {
	switch channel.Type {
	case "webhook":
		return NewWebhookNotifier(channel.Settings), nil
	case "email":
		return NewEmailNotifier(channel.Settings), nil
	case "slack":
		return NewSlackNotifier(channel.Settings), nil
	default:
		return NewLogNotifier(), nil
	}
}

func (s *AlertService) GetAlerts(status string, severity string) []*Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alerts := make([]*Alert, 0)
	for _, alert := range s.alerts {
		if status != "" && alert.Status != status {
			continue
		}
		if severity != "" && alert.Severity != severity {
			continue
		}
		alerts = append(alerts, alert)
	}
	return alerts
}

func (s *AlertService) ResolveAlert(alertID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, alert := range s.alerts {
		if alert.ID == alertID {
			alert.Status = "resolved"

			for _, history := range s.history {
				if history.AlertID == alertID {
					history.Status = "resolved"
					history.ResolvedAt = time.Now()
					history.Duration = int64(time.Since(history.FiredAt).Seconds())
					break
				}
			}

			delete(s.alerts, key)
			return nil
		}
	}
	return fmt.Errorf("alert not found")
}

func (s *AlertService) GetHistory(limit int) []*AlertHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.history) {
		limit = len(s.history)
	}

	history := make([]*AlertHistory, limit)
	copy(history, s.history[len(s.history)-limit:])
	return history
}

func (s *AlertService) GetChannels() []AlertChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	channels := make([]AlertChannel, len(s.channels))
	copy(channels, s.channels)
	return channels
}

func (s *AlertService) DeleteChannel(channelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.channels {
		if c.ID == channelID {
			s.channels = append(s.channels[:i], s.channels[i+1:]...)

			if notifier, ok := s.notifiers[channelID]; ok {
				notifier.Close()
				delete(s.notifiers, channelID)
			}
			return nil
		}
	}
	return fmt.Errorf("channel not found")
}

func (s *AlertService) GetAlertStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_alerts":    len(s.alerts),
		"total_rules":      len(s.rules),
		"enabled_rules":    s.countEnabledRules(),
		"total_channels":   len(s.channels),
		"history_count":    len(s.history),
	}

	bySeverity := make(map[string]int)
	for _, alert := range s.alerts {
		bySeverity[alert.Severity]++
	}
	stats["by_severity"] = bySeverity

	byStatus := make(map[string]int)
	for _, alert := range s.alerts {
		byStatus[alert.Status]++
	}
	stats["by_status"] = byStatus

	return stats
}

func (s *AlertService) countEnabledRules() int {
	count := 0
	for _, rule := range s.rules {
		if rule.Enabled {
			count++
		}
	}
	return count
}

func (s *AlertService) ExportRulesJSON() ([]byte, error) {
	rules := s.GetRules()
	return json.MarshalIndent(rules, "", "  ")
}

func (s *AlertService) ExportAlertsJSON() ([]byte, error) {
	alerts := s.GetAlerts("", "")
	return json.MarshalIndent(alerts, "", "  ")
}

type WebhookNotifier struct {
	url     string
	headers map[string]string
}

func NewWebhookNotifier(settings map[string]interface{}) *WebhookNotifier {
	return &WebhookNotifier{
		url:     settings["url"].(string),
		headers: make(map[string]string),
	}
}

func (n *WebhookNotifier) Send(alert *Alert) error {
	log.Printf("[Webhook] Sending alert: %s to %s", alert.Name, n.url)
	return nil
}

func (n *WebhookNotifier) Close() error {
	return nil
}

type EmailNotifier struct {
	smtpHost string
	from     string
	to       []string
}

func NewEmailNotifier(settings map[string]interface{}) *EmailNotifier {
	return &EmailNotifier{
		smtpHost: settings["smtp_host"].(string),
		from:     settings["from"].(string),
		to:       []string{settings["to"].(string)},
	}
}

func (n *EmailNotifier) Send(alert *Alert) error {
	log.Printf("[Email] Sending alert: %s to %v", alert.Name, n.to)
	return nil
}

func (n *EmailNotifier) Close() error {
	return nil
}

type SlackNotifier struct {
	webhookURL string
	channel    string
}

func NewSlackNotifier(settings map[string]interface{}) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: settings["webhook_url"].(string),
		channel:    settings["channel"].(string),
	}
}

func (n *SlackNotifier) Send(alert *Alert) error {
	log.Printf("[Slack] Sending alert: %s to channel %s", alert.Name, n.channel)
	return nil
}

func (n *SlackNotifier) Close() error {
	return nil
}

type LogNotifier struct{}

func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

func (n *LogNotifier) Send(alert *Alert) error {
	log.Printf("[Alert] %s: %s - %s", alert.Severity, alert.Name, alert.Message)
	return nil
}

func (n *LogNotifier) Close() error {
	return nil
}

func (s *AlertService) CreateDefaultRules() {
	rules := []AlertRule{
		{
			ID:          "high-cpu-usage",
			Name:        "High CPU Usage",
			Description: "CPU usage exceeds 80%",
			Condition: &AlertCondition{
				Metric:    "cpu_usage",
				Operator:  "gt",
				Threshold: 80,
			},
			Severity: "warning",
			Enabled:  true,
			Labels:   map[string]string{"environment": "production"},
			Actions: []*AlertAction{
				{Type: "notification", Channel: "log"},
			},
		},
		{
			ID:          "critical-cpu-usage",
			Name:        "Critical CPU Usage",
			Description: "CPU usage exceeds 95%",
			Condition: &AlertCondition{
				Metric:    "cpu_usage",
				Operator:  "gt",
				Threshold: 95,
			},
			Severity: "critical",
			Enabled:  true,
			Labels:   map[string]string{"environment": "production"},
			Actions: []*AlertAction{
				{Type: "notification", Channel: "log"},
			},
		},
		{
			ID:          "high-memory-usage",
			Name:        "High Memory Usage",
			Description: "Memory usage exceeds 85%",
			Condition: &AlertCondition{
				Metric:    "memory_usage",
				Operator:  "gt",
				Threshold: 85,
			},
			Severity: "warning",
			Enabled:  true,
			Labels:   map[string]string{"environment": "production"},
			Actions: []*AlertAction{
				{Type: "notification", Channel: "log"},
			},
		},
		{
			ID:          "high-error-rate",
			Name:        "High Error Rate",
			Description: "Error rate exceeds 5%",
			Condition: &AlertCondition{
				Metric:    "error_rate",
				Operator:  "gt",
				Threshold: 5,
			},
			Severity: "critical",
			Enabled:  true,
			Labels:   map[string]string{"environment": "production"},
			Actions: []*AlertAction{
				{Type: "notification", Channel: "log"},
			},
		},
		{
			ID:          "slow-response-time",
			Name:        "Slow Response Time",
			Description: "Average response time exceeds 500ms",
			Condition: &AlertCondition{
				Metric:    "response_time",
				Operator:  "gt",
				Threshold: 500,
			},
			Severity: "warning",
			Enabled:  true,
			Labels:   map[string]string{"environment": "production"},
			Actions: []*AlertAction{
				{Type: "notification", Channel: "log"},
			},
		},
	}

	for _, rule := range rules {
		s.CreateRule(&rule)
	}
}

func (s *AlertService) HandleRulesAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rules := s.GetRules()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rules)
	case http.MethodPost:
		var rule AlertRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.CreateRule(&rule); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(rule)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *AlertService) HandleAlertsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		severity := r.URL.Query().Get("severity")
		alerts := s.GetAlerts(status, severity)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(alerts)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *AlertService) HandleChannelsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		channels := s.GetChannels()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(channels)
	case http.MethodPost:
		var channel AlertChannel
		if err := json.NewDecoder(r.Body).Decode(&channel); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.RegisterChannel(channel); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
