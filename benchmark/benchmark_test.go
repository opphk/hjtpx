package benchmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	benchmarkBaseURL = "http://localhost:8080"
	benchmarkTimeout = 30 * time.Second
)

type BenchmarkHTTPClient struct {
	client  *http.Client
	baseURL string
}

func NewBenchmarkHTTPClient(baseURL string) *BenchmarkHTTPClient {
	return &BenchmarkHTTPClient{
		client: &http.Client{
			Timeout: benchmarkTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
			},
		},
		baseURL: baseURL,
	}
}

func (c *BenchmarkHTTPClient) DoRequest(method, endpoint string, body interface{}) (int, time.Duration, error) {
	start := time.Now()

	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return 0, 0, err
		}
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, time.Since(start), err
	}
	defer resp.Body.Close()

	return resp.StatusCode, time.Since(start), nil
}

func BenchmarkSliderGenerate(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	body := map[string]interface{}{
		"app_id":      1,
		"width":       320,
		"height":      160,
		"slider_size": 40,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		statusCode, _, err := client.DoRequest("POST", "/api/v1/captcha/slider/generate", body)
		if err != nil {
			b.Fatal(err)
		}
		if statusCode != http.StatusOK && statusCode != http.StatusCreated {
			b.Errorf("Unexpected status code: %d", statusCode)
		}
	}
}

func BenchmarkSliderVerify(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	body := map[string]interface{}{
		"app_id":     1,
		"session_id": fmt.Sprintf("bench_session_%d", time.Now().UnixNano()),
		"x":          150,
		"track_data": []map[string]interface{}{
			{"x": 10, "y": 5, "t": 50},
			{"x": 30, "y": 8, "t": 100},
			{"x": 60, "y": 10, "t": 150},
			{"x": 100, "y": 12, "t": 200},
			{"x": 150, "y": 15, "t": 300},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		statusCode, _, err := client.DoRequest("POST", "/api/v1/captcha/slider/verify", body)
		if err != nil {
			b.Fatal(err)
		}
		if statusCode != http.StatusOK && statusCode != http.StatusCreated {
			b.Errorf("Unexpected status code: %d", statusCode)
		}
	}
}

func BenchmarkClickGenerate(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	body := map[string]interface{}{
		"app_id":       1,
		"width":        320,
		"height":       160,
		"target_count": 4,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		statusCode, _, err := client.DoRequest("POST", "/api/v1/captcha/click/generate", body)
		if err != nil {
			b.Fatal(err)
		}
		if statusCode != http.StatusOK && statusCode != http.StatusCreated {
			b.Errorf("Unexpected status code: %d", statusCode)
		}
	}
}

func BenchmarkClickVerify(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	body := map[string]interface{}{
		"app_id":     1,
		"session_id": fmt.Sprintf("bench_session_%d", time.Now().UnixNano()),
		"clicks": []map[string]interface{}{
			{"x": 100, "y": 80},
			{"x": 200, "y": 40},
			{"x": 150, "y": 120},
			{"x": 250, "y": 60},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		statusCode, _, err := client.DoRequest("POST", "/api/v1/captcha/click/verify", body)
		if err != nil {
			b.Fatal(err)
		}
		if statusCode != http.StatusOK && statusCode != http.StatusCreated {
			b.Errorf("Unexpected status code: %d", statusCode)
		}
	}
}

func BenchmarkImageGenerate(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	body := map[string]interface{}{
		"app_id": 1,
		"length": 4,
		"width":  120,
		"height": 40,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		statusCode, _, err := client.DoRequest("POST", "/api/v1/captcha/image/generate", body)
		if err != nil {
			b.Fatal(err)
		}
		if statusCode != http.StatusOK && statusCode != http.StatusCreated {
			b.Errorf("Unexpected status code: %d", statusCode)
		}
	}
}

func BenchmarkImageVerify(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	body := map[string]interface{}{
		"app_id":     1,
		"session_id": fmt.Sprintf("bench_session_%d", time.Now().UnixNano()),
		"captcha":    "ABCD",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		statusCode, _, err := client.DoRequest("POST", "/api/v1/captcha/image/verify", body)
		if err != nil {
			b.Fatal(err)
		}
		if statusCode != http.StatusOK && statusCode != http.StatusCreated {
			b.Errorf("Unexpected status code: %d", statusCode)
		}
	}
}

func BenchmarkSliderGenerateParallel(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	body := map[string]interface{}{
		"app_id":      1,
		"width":       320,
		"height":      160,
		"slider_size": 40,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			statusCode, _, err := client.DoRequest("POST", "/api/v1/captcha/slider/generate", body)
			if err != nil {
				b.Fatal(err)
			}
			if statusCode != http.StatusOK && statusCode != http.StatusCreated {
				b.Errorf("Unexpected status code: %d", statusCode)
			}
		}
	})
}

func BenchmarkSliderVerifyParallel(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	body := map[string]interface{}{
		"app_id":     1,
		"session_id": fmt.Sprintf("bench_session_%d", time.Now().UnixNano()),
		"x":          150,
		"track_data": []map[string]interface{}{
			{"x": 10, "y": 5, "t": 50},
			{"x": 30, "y": 8, "t": 100},
			{"x": 60, "y": 10, "t": 150},
			{"x": 100, "y": 12, "t": 200},
			{"x": 150, "y": 15, "t": 300},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			statusCode, _, err := client.DoRequest("POST", "/api/v1/captcha/slider/verify", body)
			if err != nil {
				b.Fatal(err)
			}
			if statusCode != http.StatusOK && statusCode != http.StatusCreated {
				b.Errorf("Unexpected status code: %d", statusCode)
			}
		}
	})
}

func BenchmarkImageGenerateParallel(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	body := map[string]interface{}{
		"app_id": 1,
		"length": 4,
		"width":  120,
		"height": 40,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			statusCode, _, err := client.DoRequest("POST", "/api/v1/captcha/image/generate", body)
			if err != nil {
				b.Fatal(err)
			}
			if statusCode != http.StatusOK && statusCode != http.StatusCreated {
				b.Errorf("Unexpected status code: %d", statusCode)
			}
		}
	})
}

func BenchmarkClickGenerateParallel(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	body := map[string]interface{}{
		"app_id":       1,
		"width":        320,
		"height":       160,
		"target_count": 4,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			statusCode, _, err := client.DoRequest("POST", "/api/v1/captcha/click/generate", body)
			if err != nil {
				b.Fatal(err)
			}
			if statusCode != http.StatusOK && statusCode != http.StatusCreated {
				b.Errorf("Unexpected status code: %d", statusCode)
			}
		}
	})
}

func BenchmarkMixedScenarios(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	scenarios := []struct {
		name      string
		endpoint  string
		method    string
		body      map[string]interface{}
		weight    int
	}{
		{
			name:     "Slider Generate",
			endpoint: "/api/v1/captcha/slider/generate",
			method:   "POST",
			body: map[string]interface{}{
				"app_id":      1,
				"width":       320,
				"height":      160,
				"slider_size": 40,
			},
			weight: 40,
		},
		{
			name:     "Image Generate",
			endpoint: "/api/v1/captcha/image/generate",
			method:   "POST",
			body: map[string]interface{}{
				"app_id": 1,
				"length": 4,
				"width":  120,
				"height": 40,
			},
			weight: 40,
		},
		{
			name:     "Click Generate",
			endpoint: "/api/v1/captcha/click/generate",
			method:   "POST",
			body: map[string]interface{}{
				"app_id":       1,
				"width":        320,
				"height":       160,
				"target_count": 4,
			},
			weight: 20,
		},
	}

	var totalWeight int
	for _, s := range scenarios {
		totalWeight += s.weight
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		sessionCounter := int64(0)
		for pb.Next() {
			weight := atomic.AddInt64(&sessionCounter, 1) % int64(totalWeight)
			currentWeight := 0

			for _, scenario := range scenarios {
				currentWeight += scenario.weight
				if weight < int64(currentWeight) {
					sessionBody := scenario.body
					if body, ok := sessionBody["app_id"]; ok {
						sessionBody = deepCopyMap(sessionBody)
						sessionBody["session_id"] = fmt.Sprintf("bench_%d", atomic.AddInt64(&sessionCounter, 1))
						_ = body
					}

					statusCode, _, err := client.DoRequest(scenario.method, scenario.endpoint, sessionBody)
					if err != nil {
						b.Fatal(err)
					}
					if statusCode != http.StatusOK && statusCode != http.StatusCreated {
						b.Errorf("Unexpected status code: %d for %s", statusCode, scenario.name)
					}
					break
				}
			}
		}
	})
}

func BenchmarkPeakLoad(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	peakConcurrency := 500
	duration := 30 * time.Second

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	var totalRequests int64
	var successfulRequests int64
	var failedRequests int64
	var latencyMutex sync.Mutex
	latencies := make([]time.Duration, 0, 100000)

	stopChan := make(chan struct{})

	for i := 0; i < peakConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					sessionID := fmt.Sprintf("peak_%d_%d", time.Now().UnixNano(), i)

					body := map[string]interface{}{
						"app_id":     1,
						"session_id": sessionID,
						"length":     4,
						"width":      120,
						"height":     40,
					}

					start := time.Now()
					statusCode, _, err := client.DoRequest("POST", "/api/v1/captcha/image/generate", body)
					latency := time.Since(start)

					atomic.AddInt64(&totalRequests, 1)

					latencyMutex.Lock()
					latencies = append(latencies, latency)
					latencyMutex.Unlock()

					if err != nil || (statusCode != http.StatusOK && statusCode != http.StatusCreated) {
						atomic.AddInt64(&failedRequests, 1)
					} else {
						atomic.AddInt64(&successfulRequests, 1)
					}

				case <-stopChan:
					return
				}
			}
		}()
	}

	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	start := time.Now()
	elapsed := time.Since(start)
	qps := float64(totalRequests) / elapsed.Seconds()

	latencyMutex.Lock()
	p50, p95, p99 := calculatePercentiles(latencies)
	latencyMutex.Unlock()

	b.ReportMetric(qps, "queries/op")
	b.ReportMetric(float64(p50.Microseconds()), "p50-latency/us")
	b.ReportMetric(float64(p95.Microseconds()), "p95-latency/us")
	b.ReportMetric(float64(p99.Microseconds()), "p99-latency/us")
	b.ReportMetric(float64(failedRequests)/float64(totalRequests)*100, "error-rate/%")

	fmt.Printf("\nPeak Load Results:\n")
	fmt.Printf("  Total Requests: %d\n", totalRequests)
	fmt.Printf("  QPS: %.2f\n", qps)
	fmt.Printf("  P50 Latency: %v\n", p50)
	fmt.Printf("  P95 Latency: %v\n", p95)
	fmt.Printf("  P99 Latency: %v\n", p99)
	fmt.Printf("  Error Rate: %.2f%%\n", float64(failedRequests)/float64(totalRequests)*100)
}

func BenchmarkSustainedLoad(b *testing.B) {
	client := NewBenchmarkHTTPClient(benchmarkBaseURL)

	concurrency := 50
	duration := 60 * time.Second

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	var totalRequests int64
	var successfulRequests int64
	var failedRequests int64
	var mutex sync.Mutex
	latencies := make([]time.Duration, 0, 100000)

	stopChan := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					sessionID := fmt.Sprintf("sustained_%d_%d_%d", time.Now().UnixNano(), workerID, i)

					scenario := workerID % 3
					var endpoint string
					var body map[string]interface{}

					switch scenario {
					case 0:
						endpoint = "/api/v1/captcha/image/generate"
						body = map[string]interface{}{
							"app_id":     1,
							"session_id": sessionID,
							"length":     4,
							"width":      120,
							"height":     40,
						}
					case 1:
						endpoint = "/api/v1/captcha/slider/generate"
						body = map[string]interface{}{
							"app_id":      1,
							"session_id": sessionID,
							"width":       320,
							"height":      160,
							"slider_size": 40,
						}
					case 2:
						endpoint = "/api/v1/captcha/click/generate"
						body = map[string]interface{}{
							"app_id":       1,
							"session_id":   sessionID,
							"width":        320,
							"height":       160,
							"target_count": 4,
						}
					}

					start := time.Now()
					statusCode, _, err := client.DoRequest("POST", endpoint, body)
					latency := time.Since(start)

					atomic.AddInt64(&totalRequests, 1)

					mutex.Lock()
					latencies = append(latencies, latency)
					mutex.Unlock()

					if err != nil || (statusCode != http.StatusOK && statusCode != http.StatusCreated) {
						atomic.AddInt64(&failedRequests, 1)
					} else {
						atomic.AddInt64(&successfulRequests, 1)
					}

				case <-stopChan:
					return
				}
			}
		}(i)
	}

	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	start := time.Now()
	elapsed := time.Since(start)
	qps := float64(totalRequests) / elapsed.Seconds()

	mutex.Lock()
	p50, p95, p99 := calculatePercentiles(latencies)
	mutex.Unlock()

	b.ReportMetric(qps, "queries/op")
	b.ReportMetric(float64(p50.Microseconds()), "p50-latency/us")
	b.ReportMetric(float64(p95.Microseconds()), "p95-latency/us")
	b.ReportMetric(float64(p99.Microseconds()), "p99-latency/us")
	b.ReportMetric(float64(failedRequests)/float64(totalRequests)*100, "error-rate/%")

	fmt.Printf("\nSustained Load Results:\n")
	fmt.Printf("  Duration: %v\n", elapsed)
	fmt.Printf("  Total Requests: %d\n", totalRequests)
	fmt.Printf("  Successful: %d\n", successfulRequests)
	fmt.Printf("  Failed: %d\n", failedRequests)
	fmt.Printf("  QPS: %.2f\n", qps)
	fmt.Printf("  P50 Latency: %v\n", p50)
	fmt.Printf("  P95 Latency: %v\n", p95)
	fmt.Printf("  P99 Latency: %v\n", p99)
	fmt.Printf("  Error Rate: %.2f%%\n", float64(failedRequests)/float64(totalRequests)*100)
}

func calculatePercentiles(latencies []time.Duration) (p50, p95, p99 time.Duration) {
	if len(latencies) == 0 {
		return 0, 0, 0
	}

	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	quickSort(sorted, 0, len(sorted)-1)

	n := len(sorted)
	p50 = sorted[n*50/100]
	p95 = sorted[n*95/100]
	if n > 100 {
		p99 = sorted[n*99/100]
	} else if n > 0 {
		p99 = sorted[n-1]
	}

	return p50, p95, p99
}

func quickSort(arr []time.Duration, low, high int) {
	if low < high {
		p := partition(arr, low, high)
		quickSort(arr, low, p-1)
		quickSort(arr, p+1, high)
	}
}

func partition(arr []time.Duration, low, high int) int {
	pivot := arr[high]
	i := low - 1

	for j := low; j < high; j++ {
		if arr[j] < pivot {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}

	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

func deepCopyMap(original map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range original {
		result[key] = value
	}
	return result
}

func BenchmarkDatabaseQuery(b *testing.B) {
	b.Run("Query Verifications", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
		}
	})

	b.Run("Query Logs", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
		}
	})
}

func BenchmarkCachePerformance(b *testing.B) {
	cache := NewQueryCache(10000)

	b.Run("Cache Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("benchmark_key_%d", i%10000)
			cache.Set(key, "test_value", 5*time.Minute)
		}
	})

	b.Run("Cache Get Hit", func(b *testing.B) {
		for i := 0; i < 10000; i++ {
			key := fmt.Sprintf("benchmark_key_%d", i)
			cache.Set(key, "test_value", 5*time.Minute)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("benchmark_key_%d", i%10000)
			cache.Get(key)
		}
	})

	b.Run("Cache Get Miss", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("nonexistent_key_%d", i)
			cache.Get(key)
		}
	})

	b.Run("Cache Hit Rate", func(b *testing.B) {
		for i := 0; i < 10000; i++ {
			key := fmt.Sprintf("benchmark_key_%d", i)
			cache.Set(key, "test_value", 5*time.Minute)
		}

		for i := 0; i < 5000; i++ {
			key := fmt.Sprintf("benchmark_key_%d", i%10000)
			cache.Get(key)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.GetHitRate()
		}
	})
}

func BenchmarkWorkerPool(b *testing.B) {
	pool := NewWorkerPool(10, 1000)
	pool.Start()
	defer pool.Stop()

	b.Run("Submit Jobs", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pool.Submit(func() interface{} {
				return "result"
			})
		}
	})

	b.Run("Submit and Wait", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pool.SubmitWithTimeout(func() interface{} {
				time.Sleep(1 * time.Millisecond)
				return "result"
			}, 5*time.Second)
		}
	})
}

func BenchmarkResponsePool(b *testing.B) {
	pool := NewResponsePool()

	b.Run("Get Response", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp := pool.Get()
			resp.StatusCode = 200
			resp.Body = append(resp.Body, make([]byte, 1024)...)
			pool.Put(resp)
		}
	})
}

func BenchmarkConcurrentMap(b *testing.B) {
	var counter int64
	var mutex sync.Mutex
	counterMap := make(map[string]int64)

	b.Run("With Mutex", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mutex.Lock()
			counterMap[fmt.Sprintf("key_%d", i)]++
			mutex.Unlock()
		}
	})

	b.Run("With Atomic", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			atomic.AddInt64(&counter, 1)
		}
	})
}

func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("Slice Prealloc", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slice := make([]int, 0, 1000)
			for j := 0; j < 1000; j++ {
				slice = append(slice, j)
			}
		}
	})

	b.Run("Slice No Prealloc", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var slice []int
			for j := 0; j < 1000; j++ {
				slice = append(slice, j)
			}
		}
	})

	b.Run("Map Prealloc", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			m := make(map[string]int, 1000)
			for j := 0; j < 1000; j++ {
				m[fmt.Sprintf("key_%d", j)] = j
			}
		}
	})

	b.Run("Map No Prealloc", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			m := make(map[string]int)
			for j := 0; j < 1000; j++ {
				m[fmt.Sprintf("key_%d", j)] = j
			}
		}
	})
}

func BenchmarkStringOperations(b *testing.B) {
	b.Run("String Concatenation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := ""
			for j := 0; j < 10; j++ {
				result += fmt.Sprintf("part_%d_", j)
			}
		}
	})

	b.Run("Strings Builder", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var builder strings.Builder
			for j := 0; j < 10; j++ {
				builder.WriteString(fmt.Sprintf("part_%d_", j))
			}
		}
	})

	b.Run("String Join", func(b *testing.B) {
		parts := make([]string, 10)
		for i := 0; i < 10; i++ {
			parts[i] = fmt.Sprintf("part_%d", i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = strings.Join(parts, "_")
		}
	})
}

func BenchmarkJSONMarshaling(b *testing.B) {
	type TestStruct struct {
		ID        uint      `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
		Data      []byte    `json:"data"`
	}

	data := TestStruct{
		ID:        1,
		Name:      "Test User",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		Data:      make([]byte, 1024),
	}

	b.Run("JSON Marshal", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(data)
		}
	})

	b.Run("JSON Unmarshal", func(b *testing.B) {
		jsonData, _ := json.Marshal(data)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result TestStruct
			_ = json.Unmarshal(jsonData, &result)
		}
	})
}

func BenchmarkHTTPClientReuse(b *testing.B) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
		},
	}

	body := []byte(`{"app_id": 1, "length": 4}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", benchmarkBaseURL+"/api/v1/captcha/image/generate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		_, _ = client.Do(req)
	}
}

func BenchmarkHTTPClientCreation(b *testing.B) {
	body := []byte(`{"app_id": 1, "length": 4}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := &http.Client{Timeout: 30 * time.Second}
		req, _ := http.NewRequest("POST", benchmarkBaseURL+"/api/v1/captcha/image/generate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		_, _ = client.Do(req)
	}
}

func printSystemInfo() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	fmt.Printf("\nSystem Information:\n")
	fmt.Printf("  CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("  Go Version: %s\n", runtime.Version())
	fmt.Printf("  OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  Goroutines: %d\n", runtime.NumGoroutine())
	fmt.Printf("  Memory Alloc: %d MB\n", memStats.Alloc/1024/1024)
	fmt.Printf("  Memory Sys: %d MB\n", memStats.Sys/1024/1024)
}
