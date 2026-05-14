package performance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkHTTPRequests(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "test-captcha",
			"status":   "ok",
		})
	}))
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get(server.URL)
		if err == nil {
			resp.Body.Close()
		}
	}
}

func BenchmarkConcurrentHTTPRequests(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "test-captcha",
			"status":   "ok",
		})
	}))
	defer server.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := http.Get(server.URL)
			if err == nil {
				resp.Body.Close()
			}
		}
	})
}

func BenchmarkJSONMarshal(b *testing.B) {
	data := map[string]interface{}{
		"id":       "captcha-123",
		"type":     "slider",
		"target_x": 100,
		"target_y": 50,
		"success":  true,
		"timestamp": time.Now().Unix(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(data)
	}
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	jsonData := []byte(`{
		"id": "captcha-123",
		"type": "slider",
		"target_x": 100,
		"target_y": 50,
		"success": true,
		"timestamp": 1700000000
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var data map[string]interface{}
		_ = json.Unmarshal(jsonData, &data)
	}
}

func TestThroughput_SingleEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "test",
			"status": "ok",
		})
	}))
	defer server.Close()

	concurrency := 10
	iterations := 1000

	var wg sync.WaitGroup
	var successCount int64

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				resp, err := http.Get(server.URL)
				if err == nil && resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
					resp.Body.Close()
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	totalOps := int64(concurrency * iterations)
	opsPerSecond := float64(totalOps) / elapsed.Seconds()

	t.Logf("Total operations: %d", totalOps)
	t.Logf("Successful operations: %d", successCount)
	t.Logf("Elapsed time: %v", elapsed)
	t.Logf("Operations per second: %.2f", opsPerSecond)

	if successCount != totalOps {
		t.Errorf("Expected %d successful operations, got %d", totalOps, successCount)
	}
}

func TestThroughput_MultipleEndpoints(t *testing.T) {
	endpoints := []string{
		"/health",
		"/api/v1/captcha/slider/generate",
		"/api/v1/captcha/slider/verify",
		"/api/v1/captcha/click/generate",
		"/api/v1/captcha/click/verify",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	}))
	defer server.Close()

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			iterations := 500
			start := time.Now()

			for i := 0; i < iterations; i++ {
				resp, err := http.Get(server.URL + endpoint)
				if err == nil {
					resp.Body.Close()
				}
			}

			elapsed := time.Since(start)
			opsPerSecond := float64(iterations) / elapsed.Seconds()

			t.Logf("Endpoint %s: %.2f ops/sec", endpoint, opsPerSecond)

			if opsPerSecond < 100 {
				t.Logf("Warning: Low throughput for %s (%.2f ops/sec)", endpoint, opsPerSecond)
			}
		})
	}
}

func TestLatency_Distribution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond * 5)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	iterations := 1000
	latencies := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		resp, err := http.Get(server.URL)
		if err == nil {
			resp.Body.Close()
		}
		latencies[i] = time.Since(start)
	}

	var total time.Duration
	var maxLatency time.Duration
	var minLatency time.Duration = time.Hour

	for _, lat := range latencies {
		total += lat
		if lat > maxLatency {
			maxLatency = lat
		}
		if lat < minLatency {
			minLatency = lat
		}
	}

	avgLatency := total / time.Duration(iterations)

	var variance time.Duration
	for _, lat := range latencies {
		diff := lat - avgLatency
		variance += diff * diff
	}
	stdDev := time.Duration(int64(variance) / int64(iterations))

	t.Logf("Latency stats (over %d iterations):", iterations)
	t.Logf("  Average: %v", avgLatency)
	t.Logf("  Min: %v", minLatency)
	t.Logf("  Max: %v", maxLatency)
	t.Logf("  StdDev: %v", stdDev)

	if avgLatency > 50*time.Millisecond {
		t.Logf("Warning: High average latency (%v)", avgLatency)
	}
}

func TestConcurrentClients_Throughput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "test",
			"status": "ok",
		})
	}))
	defer server.Close()

	clientCount := 50
	requestsPerClient := 100

	var wg sync.WaitGroup
	results := make(chan time.Duration, clientCount*requestsPerClient)

	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < requestsPerClient; j++ {
				start := time.Now()
				resp, err := http.Get(server.URL)
				if err == nil {
					resp.Body.Close()
				}
				results <- time.Since(start)
			}
		}()
	}

	wg.Wait()
	close(results)

	var totalLatency time.Duration
	var maxLatency time.Duration
	var minLatency time.Duration = time.Hour
	count := 0

	for lat := range results {
		totalLatency += lat
		count++
		if lat > maxLatency {
			maxLatency = lat
		}
		if lat < minLatency {
			minLatency = lat
		}
	}

	avgLatency := totalLatency / time.Duration(count)

	t.Logf("Concurrent clients test (%d clients, %d requests each):", clientCount, requestsPerClient)
	t.Logf("  Total requests: %d", count)
	t.Logf("  Average latency: %v", avgLatency)
	t.Logf("  Min latency: %v", minLatency)
	t.Logf("  Max latency: %v", maxLatency)

	if maxLatency > 100*time.Millisecond {
		t.Logf("Warning: High max latency under concurrent load")
	}
}

func TestConnectionReuse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	}))
	defer server.Close()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
	}

	iterations := 500
	start := time.Now()

	for i := 0; i < iterations; i++ {
		resp, err := client.Get(server.URL)
		if err == nil {
			resp.Body.Close()
		}
	}

	elapsed := time.Since(start)
	opsPerSecond := float64(iterations) / elapsed.Seconds()

	t.Logf("Connection reuse test (%d requests):", iterations)
	t.Logf("  Elapsed time: %v", elapsed)
	t.Logf("  Operations per second: %.2f", opsPerSecond)

	if opsPerSecond < 1000 {
		t.Logf("Warning: Low throughput with connection reuse (%.2f ops/sec)", opsPerSecond)
	}
}

func TestBatchRequests_Throughput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/batch/verify" {
			var req struct {
				Items []struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"items"`
			}
			json.NewDecoder(r.Body).Decode(&req)

			results := make([]map[string]interface{}, len(req.Items))
			for i := range req.Items {
				results[i] = map[string]interface{}{
					"id":      req.Items[i].ID,
					"success": true,
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"results": results,
			})
		}
	}))
	defer server.Close()

	t.Run("Batch of 10 items", func(t *testing.T) {
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			reqBody := map[string]interface{}{
				"items": []map[string]string{
					{"id": fmt.Sprintf("captcha-%d", i)},
					{"id": fmt.Sprintf("captcha-%d", i+1)},
					{"id": fmt.Sprintf("captcha-%d", i+2)},
					{"id": fmt.Sprintf("captcha-%d", i+3)},
					{"id": fmt.Sprintf("captcha-%d", i+4)},
					{"id": fmt.Sprintf("captcha-%d", i+5)},
					{"id": fmt.Sprintf("captcha-%d", i+6)},
					{"id": fmt.Sprintf("captcha-%d", i+7)},
					{"id": fmt.Sprintf("captcha-%d", i+8)},
					{"id": fmt.Sprintf("captcha-%d", i+9)},
				},
			}
			jsonBody, _ := json.Marshal(reqBody)

			resp, err := http.Post(server.URL+"/api/v1/batch/verify", "application/json",
				bytes.NewBuffer(jsonBody))
			if err == nil {
				resp.Body.Close()
			}
		}

		elapsed := time.Since(start)
		opsPerSecond := float64(iterations) / elapsed.Seconds()

		t.Logf("Batch verification (10 items per request, %d batches):", iterations)
		t.Logf("  Elapsed time: %v", elapsed)
		t.Logf("  Operations per second: %.2f", opsPerSecond)
	})
}

func TestStressTest_SustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	}))
	defer server.Close()

	duration := 5 * time.Second
	concurrency := 20

	var totalRequests int64
	var successRequests int64
	var wg sync.WaitGroup

	done := make(chan struct{})

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					atomic.AddInt64(&totalRequests, 1)
					resp, err := http.Get(server.URL)
					if err == nil && resp.StatusCode == http.StatusOK {
						atomic.AddInt64(&successRequests, 1)
						resp.Body.Close()
					}
				}
			}
		}()
	}

	time.Sleep(duration)
	close(done)
	wg.Wait()

	rps := float64(totalRequests) / duration.Seconds()
	successRate := float64(successRequests) / float64(totalRequests) * 100

	t.Logf("Stress test results (%v, %d concurrent):", duration, concurrency)
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Successful requests: %d", successRequests)
	t.Logf("  Requests per second: %.2f", rps)
	t.Logf("  Success rate: %.2f%%", successRate)

	if successRate < 95 {
		t.Errorf("Success rate too low: %.2f%%", successRate)
	}
}
