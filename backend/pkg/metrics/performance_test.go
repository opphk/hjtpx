package metrics

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPerformanceMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)
	assert.NotNil(t, pm.requestDuration)
	assert.NotNil(t, pm.goroutineCount)
	assert.NotNil(t, pm.memoryUsageHeap)
	assert.NotNil(t, pm.databaseConnectionsActive)
	assert.NotNil(t, pm.redisConnectionsActive)
	assert.NotNil(t, pm.cacheHitTotal)
	assert.NotNil(t, pm.bandwidthIn)
}

func TestRecordHTTPRequest(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)

	pm.RecordHTTPRequest("GET", "/api/test", 200, 100*time.Millisecond)
	pm.RecordHTTPRequest("POST", "/api/submit", 201, 50*time.Millisecond)
	pm.RecordHTTPRequest("GET", "/api/error", 500, 200*time.Millisecond)
	pm.RecordHTTPRequest("DELETE", "/api/delete", 204, 30*time.Millisecond)
}

func TestUpdateLatencyPercentiles(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)

	pm.UpdateLatencyPercentiles(100*time.Millisecond, 500*time.Millisecond, 1000*time.Millisecond)
	pm.UpdateLatencyPercentiles(50*time.Millisecond, 200*time.Millisecond, 500*time.Millisecond)
}

func TestUpdateDatabaseConnections(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)

	pm.UpdateDatabaseConnections(10, 5, 5)
	pm.UpdateDatabaseConnections(15, 0, 15)
	pm.UpdateDatabaseConnections(8, 7, 1)
}

func TestRecordDatabaseQuery(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)

	pm.RecordDatabaseQuery("SELECT", "users", 50*time.Millisecond, nil)
	pm.RecordDatabaseQuery("INSERT", "logs", 100*time.Millisecond, nil)
	pm.RecordDatabaseQuery("UPDATE", "sessions", 75*time.Millisecond, nil)
	pm.RecordDatabaseQuery("SELECT", "users", 500*time.Millisecond, assert.AnError)
}

func TestUpdateRedisConnections(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)

	pm.UpdateRedisConnections(5)
	pm.UpdateRedisConnections(10)
	pm.UpdateRedisConnections(2)
}

func TestRecordRedisCommand(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)

	pm.RecordRedisCommand("GET", 5*time.Millisecond, nil)
	pm.RecordRedisCommand("SET", 10*time.Millisecond, nil)
	pm.RecordRedisCommand("DEL", 3*time.Millisecond, nil)
	pm.RecordRedisCommand("GET", 100*time.Millisecond, assert.AnError)
}

func TestRecordCacheHit(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)

	pm.RecordCacheHit()
	pm.RecordCacheHit()
	pm.RecordCacheHit()
}

func TestRecordCacheMiss(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)

	pm.RecordCacheMiss()
	pm.RecordCacheMiss()
}

func TestRecordBandwidthIn(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)

	pm.RecordBandwidthIn(1024)
	pm.RecordBandwidthIn(2048)
	pm.RecordBandwidthIn(4096)
}

func TestRecordBandwidthOut(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)

	pm.RecordBandwidthOut(1024)
	pm.RecordBandwidthOut(2048)
	pm.RecordBandwidthOut(4096)
}

func TestConcurrentPerformanceMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			pm.RecordHTTPRequest("GET", "/api/test", 200, time.Duration(idx)*time.Millisecond)
			pm.RecordDatabaseQuery("SELECT", "users", time.Duration(idx)*time.Millisecond, nil)
			pm.RecordRedisCommand("GET", time.Duration(idx)*time.Millisecond, nil)
			pm.RecordCacheHit()
			pm.RecordBandwidthIn(1024)
			pm.RecordBandwidthOut(512)
		}(i)
	}
	wg.Wait()
}

func TestPerformanceMetricsUpdateRuntimeMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	pm := newPerformanceMetrics(registry)
	require.NotNil(t, pm)

	var lastNumGC uint32
	var lastMemStats runtime.MemStats

	pm.updateRuntimeMetrics(&lastNumGC, &lastMemStats)
	pm.updateRuntimeMetrics(&lastNumGC, &lastMemStats)
}
