package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrMonitorNotFound = errors.New("monitor not found")
	ErrAlertNotFound   = errors.New("alert not found")
)

type ModelMonitoringService interface {
	CreateMonitor(ctx context.Context, monitor *ModelMonitor) error
	GetMonitor(ctx context.Context, monitorID string) (*ModelMonitor, error)
	UpdateMonitor(ctx context.Context, monitor *ModelMonitor) error
	DeleteMonitor(ctx context.Context, monitorID string) error
	ListMonitors(ctx context.Context, modelID string) ([]*ModelMonitor, error)
	GetMonitorData(ctx context.Context, monitorID string, period *MonitoringPeriod) (*MonitorData, error)
	CreateAlert(ctx context.Context, alert *MonitoringAlert) error
	GetAlert(ctx context.Context, alertID string) (*MonitoringAlert, error)
	UpdateAlert(ctx context.Context, alert *MonitoringAlert) error
	DeleteAlert(ctx context.Context, alertID string) error
	ListAlerts(ctx context.Context, monitorID string) ([]*MonitoringAlert, error)
	GetAlertHistory(ctx context.Context, alertID string) ([]*AlertEvent, error)
}

type ModelMonitor struct {
	MonitorID    string          `json:"monitor_id"`
	ModelID     string          `json:"model_id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Type         string         `json:"type"`
	Metrics      []string       `json:"metrics"`
	Thresholds   []Threshold     `json:"thresholds"`
	Status       string         `json:"status"`
	Schedule     string         `json:"schedule"`
	Enabled      bool           `json:"enabled"`
	Notifications []Notification `json:"notifications"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type Threshold struct {
	Metric      string   `json:"metric"`
	Operator   string   `json:"operator"`
	Value      float64  `json:"value"`
	Severity   string   `json:"severity"`
}

type Notification struct {
	Type     string   `json:"type"`
	Channel  string   `json:"channel"`
	Recipients []string `json:"recipients"`
	Template string   `json:"template"`
}

type MonitoringPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Granularity string `json:"granularity"`
}

type MonitorData struct {
	MonitorID   string       `json:"monitor_id"`
	Period     *MonitoringPeriod `json:"period"`
	DataPoints []DataPoint  `json:"data_points"`
	Statistics *DataStatistics `json:"statistics"`
	Anomalies  []Anomaly    `json:"anomalies"`
	GeneratedAt time.Time   `json:"generated_at"`
}

type DataPoint struct {
	Timestamp  time.Time              `json:"timestamp"`
	Metrics    map[string]float64     `json:"metrics"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type DataStatistics struct {
	Mean      map[string]float64 `json:"mean"`
	Median    map[string]float64 `json:"median"`
	StdDev    map[string]float64 `json:"std_dev"`
	Min       map[string]float64 `json:"min"`
	Max       map[string]float64 `json:"max"`
}

type Anomaly struct {
	Timestamp   time.Time `json:"timestamp"`
	Metric      string   `json:"metric"`
	Value       float64 `json:"value"`
	ExpectedMin float64 `json:"expected_min"`
	ExpectedMax float64 `json:"expected_max"`
	Severity    string  `json:"severity"`
}

type MonitoringAlert struct {
	AlertID     string    `json:"alert_id"`
	MonitorID   string    `json:"monitor_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Condition   string    `json:"condition"`
	Threshold   float64   `json:"threshold"`
	Severity    string    `json:"severity"`
	Status      string    `json:"status"`
	FiredAt    *time.Time `json:"fired_at,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	Count       int       `json:"count"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type AlertEvent struct {
	EventID    string    `json:"event_id"`
	AlertID    string    `json:"alert_id"`
	Type       string    `json:"type"`
	Severity   string    `json:"severity"`
	Message    string    `json:"message"`
	Metadata   map[string]interface{} `json:"metadata"`
	Timestamp  time.Time `json:"timestamp"`
}

type modelMonitoringService struct {
	monitors      map[string]*ModelMonitor
	monitorData   map[string][]*MonitorData
	alerts        map[string]*MonitoringAlert
	alertHistory  map[string][]*AlertEvent
	mu            sync.RWMutex
}

func NewModelMonitoringService() ModelMonitoringService {
	return &modelMonitoringService{
		monitors:     make(map[string]*ModelMonitor),
		monitorData:  make(map[string][]*MonitorData),
		alerts:       make(map[string]*MonitoringAlert),
		alertHistory: make(map[string][]*AlertEvent),
	}
}

func (s *modelMonitoringService) CreateMonitor(ctx context.Context, monitor *ModelMonitor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if monitor.MonitorID == "" {
		monitor.MonitorID = fmt.Sprintf("mon-%d", time.Now().UnixNano())
	}

	monitor.CreatedAt = time.Now()
	monitor.UpdatedAt = time.Now()

	if monitor.Status == "" {
		monitor.Status = "active"
	}

	s.monitors[monitor.MonitorID] = monitor
	return nil
}

func (s *modelMonitoringService) GetMonitor(ctx context.Context, monitorID string) (*ModelMonitor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	monitor, exists := s.monitors[monitorID]
	if !exists {
		return nil, ErrMonitorNotFound
	}

	return monitor, nil
}

func (s *modelMonitoringService) UpdateMonitor(ctx context.Context, monitor *ModelMonitor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.monitors[monitor.MonitorID]; !exists {
		return ErrMonitorNotFound
	}

	monitor.UpdatedAt = time.Now()
	s.monitors[monitor.MonitorID] = monitor
	return nil
}

func (s *modelMonitoringService) DeleteMonitor(ctx context.Context, monitorID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.monitors[monitorID]; !exists {
		return ErrMonitorNotFound
	}

	delete(s.monitors, monitorID)
	delete(s.monitorData, monitorID)
	return nil
}

func (s *modelMonitoringService) ListMonitors(ctx context.Context, modelID string) ([]*ModelMonitor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ModelMonitor
	for _, monitor := range s.monitors {
		if modelID == "" || monitor.ModelID == modelID {
			result = append(result, monitor)
		}
	}

	return result, nil
}

func (s *modelMonitoringService) GetMonitorData(ctx context.Context, monitorID string, period *MonitoringPeriod) (*MonitorData, error) {
	s.mu.RLock()
	monitor, exists := s.monitors[monitorID]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrMonitorNotFound
	}

	dataPoints := make([]DataPoint, 0)
	for i := 0; i < 10; i++ {
		metrics := make(map[string]float64)
		for _, metric := range monitor.Metrics {
			metrics[metric] = float64(100 + i*10)
		}

		dataPoints = append(dataPoints, DataPoint{
			Timestamp: time.Now().Add(time.Duration(-i) * time.Hour),
			Metrics:   metrics,
			Metadata:  make(map[string]interface{}),
		})
	}

	statistics := &DataStatistics{
		Mean:   make(map[string]float64),
		Median: make(map[string]float64),
		StdDev: make(map[string]float64),
		Min:    make(map[string]float64),
		Max:    make(map[string]float64),
	}

	for _, metric := range monitor.Metrics {
		statistics.Mean[metric] = 150.0
		statistics.Median[metric] = 150.0
		statistics.StdDev[metric] = 25.0
		statistics.Min[metric] = 100.0
		statistics.Max[metric] = 200.0
	}

	anomalies := make([]Anomaly, 0)

	return &MonitorData{
		MonitorID:   monitorID,
		Period:     period,
		DataPoints: dataPoints,
		Statistics: statistics,
		Anomalies:  anomalies,
		GeneratedAt: time.Now(),
	}, nil
}

func (s *modelMonitoringService) CreateAlert(ctx context.Context, alert *MonitoringAlert) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if alert.AlertID == "" {
		alert.AlertID = fmt.Sprintf("alert-%d", time.Now().UnixNano())
	}

	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()

	if alert.Status == "" {
		alert.Status = "active"
	}

	s.alerts[alert.AlertID] = alert
	s.alertHistory[alert.AlertID] = []*AlertEvent{}

	return nil
}

func (s *modelMonitoringService) GetAlert(ctx context.Context, alertID string) (*MonitoringAlert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alert, exists := s.alerts[alertID]
	if !exists {
		return nil, ErrAlertNotFound
	}

	return alert, nil
}

func (s *modelMonitoringService) UpdateAlert(ctx context.Context, alert *MonitoringAlert) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.alerts[alert.AlertID]; !exists {
		return ErrAlertNotFound
	}

	alert.UpdatedAt = time.Now()
	s.alerts[alert.AlertID] = alert
	return nil
}

func (s *modelMonitoringService) DeleteAlert(ctx context.Context, alertID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.alerts[alertID]; !exists {
		return ErrAlertNotFound
	}

	delete(s.alerts, alertID)
	delete(s.alertHistory, alertID)
	return nil
}

func (s *modelMonitoringService) ListAlerts(ctx context.Context, monitorID string) ([]*MonitoringAlert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*MonitoringAlert
	for _, alert := range s.alerts {
		if monitorID == "" || alert.MonitorID == monitorID {
			result = append(result, alert)
		}
	}

	return result, nil
}

func (s *modelMonitoringService) GetAlertHistory(ctx context.Context, alertID string) ([]*AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.alerts[alertID]; !exists {
		return nil, ErrAlertNotFound
	}

	return s.alertHistory[alertID], nil
}
