package captchax

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type StressTestResult struct {
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	TimeoutRequests   int64
	AvgLatencyMs      float64
	MinLatencyMs      float64
	MaxLatencyMs      float64
	RequestsPerSecond float64
	Duration          time.Duration
	Errors            []error
}

type StabilityTestResult struct {
	TotalDuration    time.Duration
	SuccessCount     int64
	FailureCount     int64
	TimeoutCount     int64
	AvgLatencyMs     float64
	P99LatencyMs     float64
	P999LatencyMs    float64
	ErrorRate        float64
	Availability     float64
	Recoveries       int64
	Panics           int64
}

func RunStressTest(client *Client, concurrency int, totalRequests int, timeout time.Duration) *StressTestResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := &StressTestResult{
		Errors: make([]error, 0),
	}

	var wg sync.WaitGroup
	requestChan := make(chan struct{}, concurrency)
	startTime := time.Now()

	var latencySum float64
	var latencyCount int64
	var latencyMutex sync.Mutex
	var minLatency float64 = float64(^uint64(0) >> 1)
	var maxLatency float64

	recordLatency := func(latency time.Duration) {
		latencyMutex.Lock()
		defer latencyMutex.Unlock()

		latencyMs := float64(latency.Milliseconds())
		latencySum += latencyMs
		latencyCount++

		if latencyMs < minLatency {
			minLatency = latencyMs
		}
		if latencyMs > maxLatency {
			maxLatency = latencyMs
		}
	}

	for i := 0; i < totalRequests; i++ {
		select {
		case <-ctx.Done():
			break
		case requestChan <- struct{}{}:
			wg.Add(1)
			go func(requestID int) {
				defer wg.Done()
				defer func() {
					<-requestChan
				}()

				start := time.Now()
				_, err := client.GenerateSliderCaptcha(context.Background(), nil)
				latency := time.Since(start)

				atomic.AddInt64(&result.TotalRequests, 1)
				recordLatency(latency)

				if err != nil {
					if err == context.DeadlineExceeded {
						atomic.AddInt64(&result.TimeoutRequests, 1)
					} else {
						atomic.AddInt64(&result.FailedRequests, 1)
					}
					result.Errors = append(result.Errors, err)
				} else {
					atomic.AddInt64(&result.SuccessRequests, 1)
				}
			}(i)
		}
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	if latencyCount > 0 {
		result.AvgLatencyMs = latencySum / float64(latencyCount)
	}
	result.MinLatencyMs = minLatency
	result.MaxLatencyMs = maxLatency

	if result.Duration > 0 {
		result.RequestsPerSecond = float64(result.TotalRequests) / result.Duration.Seconds()
	}

	return result
}

func RunStabilityTest(client *Client, duration time.Duration) *StabilityTestResult {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	result := &StabilityTestResult{}

	var latencies []float64
	var latenciesMutex sync.Mutex
	var lastErrorTime time.Time
	var consecutiveErrors int

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	addLatency := func(latencyMs float64) {
		latenciesMutex.Lock()
		defer latenciesMutex.Unlock()
		latencies = append(latencies, latencyMs)
	}

	for {
		select {
		case <-ctx.Done():
			goto calculate
		case <-ticker.C:
			start := time.Now()
			_, err := client.GenerateSliderCaptcha(context.Background(), nil)
			latency := time.Since(start)

			atomic.AddInt64(&result.TotalDuration, latency)

			if err != nil {
				atomic.AddInt64(&result.FailureCount, 1)

				if lastErrorTime.IsZero() || time.Since(lastErrorTime) > 5*time.Second {
					consecutiveErrors = 0
				}
				consecutiveErrors++

				if consecutiveErrors > 3 {
					atomic.AddInt64(&result.Recoveries, 1)
					consecutiveErrors = 0
				}

				lastErrorTime = time.Now()
			} else {
				atomic.AddInt64(&result.SuccessCount, 1)
				consecutiveErrors = 0
				addLatency(float64(latency.Milliseconds()))
			}
		}
	}

calculate:
	latenciesMutex.Lock()
	totalOps := result.SuccessCount + result.FailureCount
	if totalOps > 0 {
		result.Availability = float64(result.SuccessCount) / float64(totalOps) * 100
		result.ErrorRate = float64(result.FailureCount) / float64(totalOps) * 100
	}

	if len(latencies) > 0 {
		var sum float64
		for _, l := range latencies {
			sum += l
		}
		result.AvgLatencyMs = sum / float64(len(latencies))

		n := len(latencies)
		if n > 0 {
			for i := 0; i < n-1; i++ {
				for j := i + 1; j < n; j++ {
					if latencies[i] > latencies[j] {
						latencies[i], latencies[j] = latencies[j], latencies[i]
					}
				}
			}

			result.P99LatencyMs = latencies[int(float64(n)*0.99)]
			if n >= 1000 {
				result.P999LatencyMs = latencies[int(float64(n)*0.999)]
			}
		}
	}
	latenciesMutex.Unlock()

	return result
}

func TestStressTest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"code":0,"message":"success","data":{"id":"test","image":"data"}}`)
	}))
	defer server.Close()

	client, err := NewClient(NewConfig(server.URL).WithAppID("stress-test"))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	testCases := []struct {
		name          string
		concurrency   int
		totalRequests int
		timeout       time.Duration
	}{
		{"LowLoad", 10, 100, 30 * time.Second},
		{"MediumLoad", 50, 500, 30 * time.Second},
		{"HighLoad", 100, 1000, 30 * time.Second},
		{"VeryHighLoad", 200, 2000, 60 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := RunStressTest(client, tc.concurrency, tc.totalRequests, tc.timeout)

			t.Logf("\n=== Stress Test Results: %s ===", tc.name)
			t.Logf("Total Requests: %d", result.TotalRequests)
			t.Logf("Successful: %d", result.SuccessRequests)
			t.Logf("Failed: %d", result.FailedRequests)
			t.Logf("Timeouts: %d", result.TimeoutRequests)
			t.Logf("Success Rate: %.2f%%", float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
			t.Logf("Avg Latency: %.2f ms", result.AvgLatencyMs)
			t.Logf("Min Latency: %.2f ms", result.MinLatencyMs)
			t.Logf("Max Latency: %.2f ms", result.MaxLatencyMs)
			t.Logf("Requests/sec: %.2f", result.RequestsPerSecond)
			t.Logf("Duration: %v", result.Duration)

			if result.SuccessRequests < int64(float64(tc.totalRequests)*0.95) {
				t.Errorf("Success rate below 95%%: %.2f%%",
					float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
			}

			if result.AvgLatencyMs > 100 {
				t.Errorf("Average latency too high: %.2f ms", result.AvgLatencyMs)
			}
		})
	}
}

func TestStabilityTest(t *testing.T) {
	errorCount := atomic.Int32{}
	recoveryCount := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rand.Intn(100) < 5 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"code":500,"message":"error"}`)
			return
		}

		time.Sleep(5 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"code":0,"message":"success","data":{"id":"test"}}`)
	}))
	defer server.Close()

	client, err := NewClient(NewConfig(server.URL).WithAppID("stability-test"))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	duration := 10 * time.Second
	result := RunStabilityTest(client, duration)

	t.Logf("\n=== Stability Test Results ===")
	t.Logf("Duration: %v", duration)
	t.Logf("Total Operations: %d", result.SuccessCount+result.FailureCount)
	t.Logf("Successful: %d", result.SuccessCount)
	t.Logf("Failed: %d", result.FailureCount)
	t.Logf("Error Rate: %.2f%%", result.ErrorRate)
	t.Logf("Availability: %.2f%%", result.Availability)
	t.Logf("Avg Latency: %.2f ms", result.AvgLatencyMs)
	t.Logf("P99 Latency: %.2f ms", result.P99LatencyMs)
	t.Logf("P999 Latency: %.2f ms", result.P999LatencyMs)
	t.Logf("Recoveries: %d", result.Recoveries)
	t.Logf("Panics: %d", result.Panics)

	if result.Availability < 95.0 {
		t.Errorf("Availability below 95%%: %.2f%%", result.Availability)
	}

	if result.P99LatencyMs > 500 {
		t.Errorf("P99 latency too high: %.2f ms", result.P99LatencyMs)
	}

	_ = errorCount
	_ = recoveryCount
}

func TestConcurrentClientCreation(t *testing.T) {
	concurrency := 100
	var wg sync.WaitGroup
	clients := make([]*Client, concurrency)
	errors := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			client, err := NewClient(NewConfig("https://example.com").WithAppID("test"))
			clients[index] = client
			errors[index] = err
		}(i)
	}

	wg.Wait()

	for i, err := range errors {
		if err != nil {
			t.Errorf("Client %d creation failed: %v", i, err)
		}
		if clients[i] == nil {
			t.Errorf("Client %d is nil", i)
		}
	}
}

func TestMemoryLeaks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"code":0,"message":"success","data":{"id":"test","image":"data"}}`)
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test"))

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	initialAlloc := memStats.Alloc

	for i := 0; i < 10000; i++ {
		_, _ = client.GenerateSliderCaptcha(context.Background(), nil)
	}

	runtime.ReadMemStats(&memStats)
	finalAlloc := memStats.Alloc

	increase := finalAlloc - initialAlloc
	increaseMB := float64(increase) / 1024 / 1024

	t.Logf("Memory increase after 10000 requests: %.2f MB", increaseMB)

	if increaseMB > 50 {
		t.Errorf("Potential memory leak: %.2f MB increase", increaseMB)
	}
}

func TestConnectionPoolEfficiency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"code":0,"message":"success","data":{"id":"test"}}`)
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test"))

	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.HealthCheck(context.Background())
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("100 concurrent health checks completed in %v", duration)
	t.Logf("Average per request: %v", duration/100)

	if duration > 5*time.Second {
		t.Errorf("Connection pool seems inefficient: %v for 100 requests", duration)
	}
}

func TestRetryResilience(t *testing.T) {
	attempt := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempt++
		currentAttempt := attempt
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		if currentAttempt < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"code":503,"message":"unavailable"}`)
			time.Sleep(10 * time.Millisecond)
			return
		}

		fmt.Fprintf(w, `{"code":0,"message":"success","data":{"id":"test"}}`)
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test").WithRetryTimes(5))

	start := time.Now()
	_, err := client.HealthCheck(context.Background())
	duration := time.Since(start)

	mu.Lock()
	totalAttempts := attempt
	mu.Unlock()

	t.Logf("Succeeded after %d attempts in %v", totalAttempts, duration)

	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}

	if totalAttempts < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", totalAttempts)
	}
}

func TestTimeoutBehavior(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"code":0,"message":"success"}`)
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test").WithTimeout(50 * time.Millisecond))

	_, err := client.HealthCheck(context.Background())

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestHighConcurrencyStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high concurrency stress test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"code":0,"message":"success"}`)
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test"))

	result := RunStressTest(client, 500, 5000, 2*time.Minute)

	t.Logf("High concurrency results:")
	t.Logf("Success rate: %.2f%%", float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	t.Logf("Requests/sec: %.2f", result.RequestsPerSecond)
	t.Logf("Avg latency: %.2f ms", result.AvgLatencyMs)

	if float64(result.SuccessRequests)/float64(result.TotalRequests) < 0.99 {
		t.Errorf("Success rate below 99%%: %.2f%%",
			float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	}
}

func TestLongRunningStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long running test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"code":0,"message":"success"}`)
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test"))

	duration := 30 * time.Second
	result := RunStabilityTest(client, duration)

	t.Logf("Long running stability results:")
	t.Logf("Availability: %.2f%%", result.Availability)
	t.Logf("Error rate: %.2f%%", result.ErrorRate)
	t.Logf("Avg latency: %.2f ms", result.AvgLatencyMs)
	t.Logf("P99 latency: %.2f ms", result.P99LatencyMs)

	if result.Availability < 99.0 {
		t.Errorf("Availability below 99%%: %.2f%%", result.Availability)
	}
}

func BenchmarkStressTest(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"code":0,"message":"success"}`)
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test"))

	b.ResetTimer()
	result := RunStressTest(client, 100, 1000, 60*time.Second)
	b.StopTimer()

	b.Logf("Stress test results:")
	b.Logf("Requests/sec: %.2f", result.RequestsPerSecond)
	b.Logf("Success rate: %.2f%%", float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	b.Logf("Avg latency: %.2f ms", result.AvgLatencyMs)
}

func BenchmarkStabilityTest(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"code":0,"message":"success"}`)
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test"))

	b.ResetTimer()
	result := RunStabilityTest(client, 10*time.Second)
	b.StopTimer()

	b.Logf("Stability test results:")
	b.Logf("Availability: %.2f%%", result.Availability)
	b.Logf("Avg latency: %.2f ms", result.AvgLatencyMs)
	b.Logf("P99 latency: %.2f ms", result.P99LatencyMs)
}
