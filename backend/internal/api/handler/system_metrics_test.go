package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector(1000, 60)
	assert.NotNil(t, collector)
}

func TestGetMetricsCollector(t *testing.T) {
	collector := GetMetricsCollector()
	assert.NotNil(t, collector)
}

func TestMetricsCollector_GetSystemMetrics(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	metrics := collector.GetSystemMetrics()
	assert.NotNil(t, metrics)
}

func TestMetricsCollector_GetAPIMetrics(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	metrics := collector.GetAPIMetrics()
	assert.NotNil(t, metrics)
}

func TestMetricsCollector_GetRealtimeData(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	data := collector.GetRealtimeData()
	assert.NotNil(t, data)
}

func TestMetricsCollector_GetSystemStatus(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	status := collector.GetSystemStatus()
	assert.NotNil(t, status)
}

func TestMetricsCollector_CheckAlerts(t *testing.T) {
	collector := NewMetricsCollector(100, 10)
	alerts := collector.CheckAlerts()
	assert.NotNil(t, alerts)
}

func TestNewCacheMetricsHandler(t *testing.T) {
	handler := NewCacheMetricsHandler()
	assert.NotNil(t, handler)
}

func TestNewDatabaseMetricsHandler(t *testing.T) {
	handler := NewDatabaseMetricsHandler()
	assert.NotNil(t, handler)
}
