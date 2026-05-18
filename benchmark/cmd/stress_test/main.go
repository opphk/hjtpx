package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/benchmark"
)

type StressTestConfig struct {
	BaseURL       string
	Concurrency   int
	Requests      int
	Duration      time.Duration
	Warmup        bool
	WarmupTime    time.Duration
	ReportFile    string
	Verbose       bool
	TestType      string
}

type StressTestResult struct {
	Timestamp           time.Time     `json:"timestamp"`
	Duration            time.Duration `json:"duration"`
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	QPS                 float64       `json:"qps"`
	AvgLatency          float64       `json:"avg_latency_ms"`
	P50Latency          float64       `json:"p50_latency_ms"`
	P95Latency          float64       `json:"p95_latency_ms"`
	P99Latency          float64       `json:"p99_latency_ms"`
	MaxLatency          float64       `json:"max_latency_ms"`
	MinLatency          float64       `json:"min_latency_ms"`
	ErrorRate           float64       `json:"error_rate_percent"`
	Concurrency         int           `json:"concurrency"`
	SystemInfo          SystemInfo    `json:"system_info"`
	Errors              []ErrorInfo   `json:"errors,omitempty"`
}

type SystemInfo struct {
	CPUCores      int       `json:"cpu_cores"`
	GoVersion     string    `json:"go_version"`
	OS            string    `json:"os"`
	Arch          string    `json:"arch"`
	Goroutines    int       `json:"goroutines"`
	MemoryAllocMB uint64    `json:"memory_alloc_mb"`
	MemorySysMB   uint64    `json:"memory_sys_mb"`
}

type ErrorInfo struct {
	Type     string `json:"type"`
	Count    int64  `json:"count"`
	Sample   string `json:"sample,omitempty"`
}

type WorkerResult struct {
	WorkerID        int
	TotalRequests   int64
	SuccessfulReqs  int64
	FailedReqs      int64
	Latencies       []time.Duration
	Errors          map[string]int64
	mu              sync.Mutex
}

func NewWorkerResult(workerID int) *WorkerResult {
	return &WorkerResult{
		WorkerID:  workerID,
		Latencies: make([]time.Duration, 0, 10000),
		Errors:    make(map[string]int64),
	}
}

func (wr *WorkerResult) Record(latency time.Duration, success bool, errorType string) {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	wr.TotalRequests++
	if success {
		wr.SuccessfulReqs++
	} else {
		wr.FailedReqs++
		if errorType != "" {
			wr.Errors[errorType]++
		}
	}

	if len(wr.Latencies) < 10000 {
		wr.Latencies = append(wr.Latencies, latency)
	}
}

func (wr *WorkerResult) MergeLatencies(latencies []time.Duration) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	wr.Latencies = append(wr.Latencies, latencies...)
}

func (wr *WorkerResult) GetStats() (int64, int64, int64, map[string]int64) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	return wr.TotalRequests, wr.SuccessfulReqs, wr.FailedReqs, wr.Errors
}

func RunStressTest(config StressTestConfig) *StressTestResult {
	log.Printf("Starting stress test with %d workers, %s duration",
		config.Concurrency, config.Duration)

	startTime := time.Now()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	startMemAlloc := memStats.Alloc
	startGCCount := memStats.NumGC

	client := benchmark.NewBenchmarkHTTPClient(config.BaseURL)

	var wg sync.WaitGroup
	workers := make([]*WorkerResult, config.Concurrency)
	var globalWg sync.WaitGroup

	for i := 0; i < config.Concurrency; i++ {
		workers[i] = NewWorkerResult(i)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	stopChan := make(chan struct{})
	latencyChan := make(chan time.Duration, 10000)
	errorChan := make(chan ErrorInfo, 100)

	if config.Warmup {
		log.Println("Warming up...")
		warmupWg := sync.WaitGroup{}
		for i := 0; i < config.Concurrency/2; i++ {
			warmupWg.Add(1)
			go func(workerID int) {
				defer warmupWg.Done()
				for j := 0; j < 100; j++ {
					client.DoRequest("POST", "/api/v1/captcha/image/generate", map[string]interface{}{
						"app_id": 1,
						"length": 4,
					})
				}
			}(i)
		}
		warmupWg.Wait()
		log.Println("Warmup completed")
	}

	var totalRequests int64
	var successfulRequests int64
	var failedRequests int64
	var totalLatency int64
	var maxLatency int64
	var minLatency int64 = -1
	var mutex sync.Mutex
	allErrors := make(map[string]int64)

	globalWg.Add(1)
	go func() {
		defer globalWg.Done()
		for latency := range latencyChan {
			atomic.AddInt64(&totalRequests, 1)
			atomic.AddInt64(&totalLatency, latency.Nanoseconds())

			mutex.Lock()
			if minLatency == -1 || latency.Nanoseconds() < minLatency {
				minLatency = latency.Nanoseconds()
			}
			if latency.Nanoseconds() > maxLatency {
				maxLatency = latency.Nanoseconds()
			}
			mutex.Unlock()
		}
	}()

	var errorMutex sync.Mutex
	globalWg.Add(1)
	go func() {
		defer globalWg.Done()
		for err := range errorChan {
			errorMutex.Lock()
			allErrors[err.Type] += err.Count
			errorMutex.Unlock()
		}
	}()

	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(worker *WorkerResult) {
			defer wg.Done()

			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					select {
					case <-ctx.Done():
						return
					default:
						start := time.Now()
						statusCode, _, err := client.DoRequest("POST", "/api/v1/captcha/image/generate", map[string]interface{}{
							"app_id":     1,
							"session_id": fmt.Sprintf("stress_%d_%d", worker.WorkerID, time.Now().UnixNano()),
							"length":     4,
							"width":      120,
							"height":     40,
						})
						latency := time.Since(start)

						success := statusCode >= 200 && statusCode < 300 && err == nil
						worker.Record(latency, success, "")

						latencyChan <- latency

						if !success {
							atomic.AddInt64(&failedRequests, 1)
							if err != nil {
								errorChan <- ErrorInfo{Type: err.Error(), Count: 1}
							} else {
								errorChan <- ErrorInfo{Type: fmt.Sprintf("HTTP_%d", statusCode), Count: 1}
							}
						} else {
							atomic.AddInt64(&successfulRequests, 1)
						}
					}
				case <-stopChan:
					return
				case <-ctx.Done():
					return
				}
			}
		}(workers[i])
	}

	time.Sleep(config.Duration)
	cancel()
	close(stopChan)
	close(latencyChan)
	close(errorChan)
	wg.Wait()
	globalWg.Wait()

	elapsed := time.Since(startTime)
	runtime.ReadMemStats(&memStats)

	result := &StressTestResult{
		Timestamp:          startTime,
		Duration:           elapsed,
		TotalRequests:      atomic.LoadInt64(&totalRequests),
		SuccessfulRequests: atomic.LoadInt64(&successfulRequests),
		FailedRequests:     atomic.LoadInt64(&failedRequests),
		Concurrency:        config.Concurrency,
		SystemInfo: SystemInfo{
			CPUCores:      runtime.NumCPU(),
			GoVersion:    runtime.Version(),
			OS:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			Goroutines:   runtime.NumGoroutine(),
			MemoryAllocMB: memStats.Alloc / 1024 / 1024,
			MemorySysMB:  memStats.Sys / 1024 / 1024,
		},
	}

	if result.TotalRequests > 0 {
		result.QPS = float64(result.TotalRequests) / elapsed.Seconds()
		result.AvgLatency = float64(totalLatency) / float64(result.TotalRequests) / 1e6
		result.ErrorRate = float64(result.FailedRequests) / float64(result.TotalRequests) * 100
	}

	allLatencies := make([]time.Duration, 0)
	for _, worker := range workers {
		_, _, _, _ = worker.GetStats()
		allLatencies = append(allLatencies, worker.Latencies...)
	}

	if len(allLatencies) > 0 {
		sorted := make([]time.Duration, len(allLatencies))
		copy(sorted, allLatencies)
		bucketSortDurations(sorted)

		n := len(sorted)
		result.P50Latency = float64(sorted[n*50/100]) / 1e6
		result.P95Latency = float64(sorted[n*95/100]) / 1e6
		result.P99Latency = float64(sorted[n*99/100]) / 1e6
		result.MaxLatency = float64(maxLatency) / 1e6
		result.MinLatency = float64(minLatency) / 1e6
	}

	if len(allErrors) > 0 {
		for errType, count := range allErrors {
			result.Errors = append(result.Errors, ErrorInfo{
				Type:  errType,
				Count: count,
			})
		}
	}

	log.Printf("Stress test completed:")
	log.Printf("  Total Requests: %d", result.TotalRequests)
	log.Printf("  QPS: %.2f", result.QPS)
	log.Printf("  Avg Latency: %.2fms", result.AvgLatency)
	log.Printf("  P95 Latency: %.2fms", result.P95Latency)
	log.Printf("  Error Rate: %.2f%%", result.ErrorRate)

	return result
}

func bucketSortDurations(arr []time.Duration) {
	if len(arr) <= 1 {
		return
	}

	min := arr[0]
	max := arr[0]
	for _, v := range arr {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	if max == min {
		return
	}

	buckets := 100
	bucketSize := float64(max-min) / float64(buckets)
	if bucketSize == 0 {
		bucketSize = 1
	}

	bucketCounts := make([]int, buckets)
	for _, v := range arr {
		idx := int(float64(v-min) / bucketSize)
		if idx >= buckets {
			idx = buckets - 1
		}
		bucketCounts[idx]++
	}

	for i := 1; i < buckets; i++ {
		bucketCounts[i] += bucketCounts[i-1]
	}

	sorted := make([]time.Duration, len(arr))
	for i := len(arr) - 1; i >= 0; i-- {
		idx := int(float64(arr[i]-min) / bucketSize)
		if idx >= buckets {
			idx = buckets - 1
		}
		bucketCounts[idx]--
		sorted[bucketCounts[idx]] = arr[i]
	}

	copy(arr, sorted)
}

func SaveResult(result *StressTestResult, filename string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("Results saved to %s", filename)
	return nil
}

func main() {
	config := StressTestConfig{
		BaseURL:     "http://localhost:8080",
		Concurrency: 50,
		Duration:    30 * time.Second,
		Warmup:      true,
		WarmupTime:  5 * time.Second,
		ReportFile:  "stress_test_report.json",
		Verbose:     true,
	}

	flag.StringVar(&config.BaseURL, "url", config.BaseURL, "Base URL for testing")
	flag.IntVar(&config.Concurrency, "c", config.Concurrency, "Number of concurrent workers")
	flag.DurationVar(&config.Duration, "d", config.Duration, "Test duration")
	flag.StringVar(&config.ReportFile, "o", config.ReportFile, "Output report file")
	flag.BoolVar(&config.Warmup, "warmup", config.Warmup, "Enable warmup")
	flag.BoolVar(&config.Verbose, "v", config.Verbose, "Verbose output")
	flag.Parse()

	result := RunStressTest(config)

	if config.ReportFile != "" {
		if err := SaveResult(result, config.ReportFile); err != nil {
			log.Printf("Failed to save report: %v", err)
		}
	}

	if result.QPS < 100 {
		log.Println("WARNING: QPS is below expected threshold")
	}
	if result.ErrorRate > 1.0 {
		log.Println("WARNING: Error rate is above 1%")
	}
	if result.P95Latency > 100 {
		log.Println("WARNING: P95 latency exceeds 100ms")
	}

	if len(os.Args) > 1 && os.Args[1] == "--serve" {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "ok",
				"result": result,
			})
		})

		log.Printf("Starting result server on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}
}
