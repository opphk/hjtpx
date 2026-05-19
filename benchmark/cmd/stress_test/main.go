package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type StressTestConfig struct {
	BaseURL        string
	Concurrency    int
	TotalRequests  int
	Timeout        time.Duration
	RequestType    string
	WarmupRequests int
}

type StressTestResult struct {
	TotalRequests   int64
	SuccessCount   int64
	FailureCount   int64
	TotalDuration  time.Duration
	AvgLatency     time.Duration
	MinLatency     time.Duration
	MaxLatency     time.Duration
	P50Latency     time.Duration
	P95Latency     time.Duration
	P99Latency     time.Duration
	RequestsPerSec float64
	Errors         map[string]int64
}

type LatencyDistribution struct {
	latencies []time.Duration
	mu        sync.Mutex
}

func NewLatencyDistribution() *LatencyDistribution {
	return &LatencyDistribution{
		latencies: make([]time.Duration, 0, 10000),
	}
}

func (ld *LatencyDistribution) Add(latency time.Duration) {
	ld.mu.Lock()
	defer ld.mu.Unlock()
	ld.latencies = append(ld.latencies, latency)
}

func (ld *LatencyDistribution) Percentile(p float64) time.Duration {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	if len(ld.latencies) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(ld.latencies))
	copy(sorted, ld.latencies)
	sortDurations(sorted)

	idx := int(float64(len(sorted)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

func sortDurations(d []time.Duration) {
	for i := 1; i < len(d); i++ {
		for j := i; j > 0 && d[j] < d[j-1]; j-- {
			d[j], d[j-1] = d[j-1], d[j]
		}
	}
}

func RunStressTest(cfg StressTestConfig) *StressTestResult {
	log.Printf("开始压力测试...")
	log.Printf("配置: 并发数=%d, 总请求数=%d, 超时=%v", cfg.Concurrency, cfg.TotalRequests, cfg.Timeout)

	result := &StressTestResult{
		Errors: make(map[string]int64),
	}

	latencyDist := NewLatencyDistribution()

	var wg sync.WaitGroup
	requestsCh := make(chan int, cfg.TotalRequests)

	startTime := time.Now()

	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for reqNum := range requestsCh {
				reqStart := time.Now()

				var err error
				var statusCode int

				switch cfg.RequestType {
				case "slider":
					statusCode, err = sendSliderRequest(cfg.BaseURL)
				case "click":
					statusCode, err = sendClickRequest(cfg.BaseURL)
				case "verify":
					statusCode, err = sendVerifyRequest(cfg.BaseURL)
				case "mixed":
					statusCode, err = sendMixedRequest(cfg.BaseURL, reqNum)
				default:
					statusCode, err = sendSliderRequest(cfg.BaseURL)
				}

				latency := time.Since(reqStart)
				latencyDist.Add(latency)

				atomic.AddInt64(&result.TotalRequests, 1)

				if err != nil {
					atomic.AddInt64(&result.FailureCount, 1)
					errMsg := fmt.Sprintf("%v", err)
					atomic.AddInt64(&result.Errors[errMsg], 1)
					log.Printf("Worker %d: 请求 %d 失败 - %v", workerID, reqNum, err)
				} else if statusCode >= 400 {
					atomic.AddInt64(&result.FailureCount, 1)
					errMsg := fmt.Sprintf("HTTP %d", statusCode)
					atomic.AddInt64(&result.Errors[errMsg], 1)
				} else {
					atomic.AddInt64(&result.SuccessCount, 1)
				}
			}
		}(i)
	}

	for i := 0; i < cfg.TotalRequests; i++ {
		requestsCh <- i
	}
	close(requestsCh)

	wg.Wait()
	result.TotalDuration = time.Since(startTime)

	result.P50Latency = latencyDist.Percentile(0.50)
	result.P95Latency = latencyDist.Percentile(0.95)
	result.P99Latency = latencyDist.Percentile(0.99)

	result.RequestsPerSec = float64(result.TotalRequests) / result.TotalDuration.Seconds()

	return result
}

func sendSliderRequest(baseURL string) (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Post(fmt.Sprintf("%s/api/v1/captcha/slider/create", baseURL),
		"application/json",
		bytes.NewBuffer([]byte("{}")))

	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func sendClickRequest(baseURL string) (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Post(fmt.Sprintf("%s/api/v1/captcha/click/create", baseURL),
		"application/json",
		bytes.NewBuffer([]byte("{\"mode\":\"number\"}")))

	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func sendVerifyRequest(baseURL string) (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	body := map[string]interface{}{
		"session_id": fmt.Sprintf("stress_test_%d", time.Now().UnixNano()),
		"type":       "slider",
		"x":          100,
		"y":          50,
	}

	jsonBody, _ := json.Marshal(body)
	resp, err := client.Post(fmt.Sprintf("%s/api/v1/captcha/verify", baseURL),
		"application/json",
		bytes.NewBuffer(jsonBody))

	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func sendMixedRequest(baseURL string, reqNum int) (int, error) {
	switch reqNum % 4 {
	case 0:
		return sendSliderRequest(baseURL)
	case 1:
		return sendClickRequest(baseURL)
	case 2:
		return sendVerifyRequest(baseURL)
	default:
		return sendHealthCheck(baseURL)
	}
}

func sendHealthCheck(baseURL string) (int, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(fmt.Sprintf("%s/health", baseURL))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func PrintStressTestResult(result *StressTestResult) {
	fmt.Println("\n========== 压力测试结果 ==========")
	fmt.Printf("总请求数:     %d\n", result.TotalRequests)
	fmt.Printf("成功数:       %d (%.2f%%)\n", result.SuccessCount, float64(result.SuccessCount)/float64(result.TotalRequests)*100)
	fmt.Printf("失败数:       %d (%.2f%%)\n", result.FailureCount, float64(result.FailureCount)/float64(result.TotalRequests)*100)
	fmt.Printf("总耗时:       %v\n", result.TotalDuration)
	fmt.Printf("吞吐量:       %.2f req/s\n", result.RequestsPerSec)
	fmt.Printf("P50延迟:      %v\n", result.P50Latency)
	fmt.Printf("P95延迟:      %v\n", result.P95Latency)
	fmt.Printf("P99延迟:      %v\n", result.P99Latency)

	if len(result.Errors) > 0 {
		fmt.Println("\n错误分布:")
		for err, count := range result.Errors {
			fmt.Printf("  %s: %d\n", err, count)
		}
	}
	fmt.Println("================================\n")
}

func ExportStressTestResult(result *StressTestResult, filename string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	log.Printf("导出结果到 %s", filename)
	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	configs := []StressTestConfig{
		{
			BaseURL:       "http://localhost:8080",
			Concurrency:   10,
			TotalRequests: 100,
			Timeout:       60 * time.Second,
			RequestType:   "slider",
		},
		{
			BaseURL:       "http://localhost:8080",
			Concurrency:   50,
			TotalRequests: 500,
			Timeout:       60 * time.Second,
			RequestType:   "mixed",
		},
		{
			BaseURL:       "http://localhost:8080",
			Concurrency:   100,
			TotalRequests: 1000,
			Timeout:       120 * time.Second,
			RequestType:   "mixed",
		},
	}

	for i, cfg := range configs {
		log.Printf("\n--- 测试 %d/%d ---", i+1, len(configs))
		result := RunStressTest(cfg)
		PrintStressTestResult(result)
		ExportStressTestResult(result, fmt.Sprintf("stress_test_result_%d.json", i+1))
	}
}
