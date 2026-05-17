package metrics

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPrometheusCollector(t *testing.T) {
	collector := GetPrometheusCollector()
	require.NotNil(t, collector)
	assert.NotNil(t, collector.httpRequestsTotal)
	assert.NotNil(t, collector.httpRequestDuration)
	assert.NotNil(t, collector.httpRequestsInFlight)
}

func TestPrometheusCollectorStartStop(t *testing.T) {
	collector := GetPrometheusCollector()
	require.NotNil(t, collector)

	err := collector.Start(":18090")
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	err = collector.Stop()
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
}

func TestPrometheusCollectorStartMultiple(t *testing.T) {
	collector := GetPrometheusCollector()
	require.NotNil(t, collector)

	err := collector.Start(":18091")
	require.NoError(t, err)

	err = collector.Start(":18091")
	assert.NoError(t, err)

	err = collector.Stop()
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
}

func TestRecordHTTPRequestMetrics(t *testing.T) {
	collector := GetPrometheusCollector()
	require.NotNil(t, collector)

	collector.RecordHTTPRequest("GET", "/api/test", 200, 100*time.Millisecond)
	collector.RecordHTTPRequest("POST", "/api/submit", 201, 50*time.Millisecond)
	collector.RecordHTTPRequest("GET", "/api/error", 500, 200*time.Millisecond)
	collector.RecordHTTPRequest("GET", "/api/notfound", 404, 30*time.Millisecond)
}

func TestIncrementDecrementInFlight(t *testing.T) {
	collector := GetPrometheusCollector()
	require.NotNil(t, collector)

	collector.IncrementInFlight()
	collector.IncrementInFlight()
	collector.IncrementInFlight()

	collector.DecrementInFlight()
	collector.DecrementInFlight()

	assert.NotPanics(t, func() {
		collector.DecrementInFlight()
	})
}

func TestGetMetrics(t *testing.T) {
	collector := GetPrometheusCollector()
	require.NotNil(t, collector)

	bm := collector.GetBusinessMetrics()
	require.NotNil(t, bm)

	pm := collector.GetPerformanceMetrics()
	require.NotNil(t, pm)

	sm := collector.GetSecurityMetrics()
	require.NotNil(t, sm)
}

func TestStatusCodeToString(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{299, "2xx"},
		{301, "3xx"},
		{302, "3xx"},
		{400, "4xx"},
		{401, "4xx"},
		{404, "4xx"},
		{500, "5xx"},
		{502, "5xx"},
		{503, "5xx"},
		{600, "5xx"},
		{100, "unknown"},
	}

	for _, tt := range tests {
		result := statusCodeToString(tt.status)
		assert.Equal(t, tt.expected, result, "status code %d", tt.status)
	}
}

func TestConcurrentRecordHTTPRequest(t *testing.T) {
	collector := GetPrometheusCollector()
	require.NotNil(t, collector)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			method := "GET"
			path := "/api/test"
			status := 200
			if idx%10 == 0 {
				status = 500
			}
			collector.RecordHTTPRequest(method, path, status, time.Duration(idx)*time.Millisecond)
		}(i)
	}
	wg.Wait()
}

func TestConcurrentInFlight(t *testing.T) {
	collector := GetPrometheusCollector()
	require.NotNil(t, collector)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			collector.IncrementInFlight()
			time.Sleep(1 * time.Millisecond)
			collector.DecrementInFlight()
		}()
	}
	wg.Wait()
}

func TestMetricsServerHandler(t *testing.T) {
	collector := GetPrometheusCollector()
	collector.Start(":18092")
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:18092/metrics")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	collector.Stop()
	time.Sleep(100 * time.Millisecond)
}

func TestMetricsServerHandlerNotStarted(t *testing.T) {
	_ = GetPrometheusCollector()

	_, err := http.Get("http://localhost:18093/metrics")
	assert.Error(t, err)
}

func TestCollectorThreadSafety(t *testing.T) {
	collector := GetPrometheusCollector()
	require.NotNil(t, collector)

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	wg.Add(4)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				collector.RecordHTTPRequest("GET", "/api/test", 200, 10*time.Millisecond)
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				collector.IncrementInFlight()
				collector.DecrementInFlight()
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				collector.GetBusinessMetrics()
				collector.GetPerformanceMetrics()
				collector.GetSecurityMetrics()
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				collector.Start(":18094")
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	time.Sleep(2 * time.Second)
	cancel()
	wg.Wait()
}
