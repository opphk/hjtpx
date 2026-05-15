package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type LoadTestConfig struct {
	TargetQPS       int
	Duration        time.Duration
	NumWorkers      int
	BaseURL         string
	Timeout         time.Duration
	EnableTLS       bool
	RequestBodySize int
}

type LoadTestResult struct {
	TotalRequests    int64
	SuccessfulReqs   int64
	FailedReqs       int64
	TimeoutReqs      int64
	ErrorReqs        int64
	TotalDuration    time.Duration
	RequestsPerSec   float64
	AvgLatency       time.Duration
	MinLatency       time.Duration
	MaxLatency       time.Duration
	P50Latency       time.Duration
	P90Latency       time.Duration
	P99Latency       time.Duration
	P999Latency      time.Duration
	LatencyArray     []time.Duration
	ErrorsByType     map[string]int64
	BytesSent        int64
	BytesReceived    int64
}

type LoadTestScenario interface {
	Name() string
	Request() (*http.Request, error)
	ProcessResponse(*http.Response, time.Duration) error
}

type CaptchaGenerateScenario struct {
	BaseURL  string
	AppID    string
	ClientIP string
}

func (s *CaptchaGenerateScenario) Name() string {
	return "captcha_generate"
}

func (s *CaptchaGenerateScenario) Request() (*http.Request, error) {
	body := map[string]interface{}{
		"app_id":      s.AppID,
		"client_info": s.ClientIP,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", s.BaseURL+"/api/v2/captcha/slider", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-IP", s.ClientIP)
	return req, nil
}

func (s *CaptchaGenerateScenario) ProcessResponse(resp *http.Response, duration time.Duration) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

type CaptchaVerifyScenario struct {
	BaseURL    string
	CaptchaID  string
	TargetX    int
	TargetY    int
	ClientIP   string
}

func (s *CaptchaVerifyScenario) Name() string {
	return "captcha_verify"
}

func (s *CaptchaVerifyScenario) Request() (*http.Request, error) {
	body := map[string]interface{}{
		"captcha_id": s.CaptchaID,
		"target_x":   s.TargetX,
		"target_y":   s.TargetY,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", s.BaseURL+"/api/v2/captcha/slider/verify", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-IP", s.ClientIP)
	return req, nil
}

func (s *CaptchaVerifyScenario) ProcessResponse(resp *http.Response, duration time.Duration) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

type HealthCheckScenario struct {
	BaseURL string
}

func (s *HealthCheckScenario) Name() string {
	return "health_check"
}

func (s *HealthCheckScenario) Request() (*http.Request, error) {
	req, err := http.NewRequest("GET", s.BaseURL+"/health", nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (s *HealthCheckScenario) ProcessResponse(resp *http.Response, duration time.Duration) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

type MixedLoadScenario struct {
	BaseURL  string
	AppID    string
	Scenarios []LoadTestScenario
	Weights  []int
}

func (s *MixedLoadScenario) Name() string {
	return "mixed_load"
}

func (s *MixedLoadScenario) Request() (*http.Request, error) {
	sum := 0
	for _, w := range s.Weights {
		sum += w
	}
	r := time.Now().UnixNano() % int64(sum)
	for i, w := range s.Weights {
		r -= int64(w)
		if r < 0 {
			return s.Scenarios[i].Request()
		}
	}
	return s.Scenarios[0].Request()
}

func (s *MixedLoadScenario) ProcessResponse(resp *http.Response, duration time.Duration) error {
	return nil
}

type LoadTester struct {
	config    LoadTestConfig
	client    *http.Client
	transport *http.Transport
	result    *LoadTestResult
	stopChan  chan struct{}
	mu        sync.Mutex
}

func NewLoadTester(config LoadTestConfig) *LoadTester {
	transport := &http.Transport{
		MaxIdleConns:        config.NumWorkers * 2,
		MaxIdleConnsPerHost: config.NumWorkers,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  false,
		MaxResponseHeaderBytes: 4096,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	result := &LoadTestResult{
		ErrorsByType: make(map[string]int64),
		LatencyArray: make([]time.Duration, 0, 100000),
	}

	return &LoadTester{
		config:    config,
		client:    client,
		transport: transport,
		result:    result,
		stopChan:  make(chan struct{}),
	}
}

func (lt *LoadTester) Run(ctx context.Context, scenario LoadTestScenario) (*LoadTestResult, error) {
	lt.result = &LoadTestResult{
		ErrorsByType:  make(map[string]int64),
		LatencyArray: make([]time.Duration, 0, 100000),
	}

	concurrency := lt.config.NumWorkers
	requestsPerWorker := lt.config.TargetQPS / concurrency

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			lt.runWorker(ctx, workerID, requestsPerWorker, scenario)
		}(i)
	}

	interval := time.Duration(1000000000/lt.config.TargetQPS) * time.Nanosecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	select {
	case <-ctx.Done():
		return lt.result, ctx.Err()
	case <-time.After(lt.config.Duration):
		lt.stopChan <- struct{}{}
	}

	wg.Wait()
	lt.result.TotalDuration = time.Since(startTime)

	lt.calculatePercentiles()
	lt.result.RequestsPerSec = float64(lt.result.TotalRequests) / lt.result.TotalDuration.Seconds()

	return lt.result, nil
}

func (lt *LoadTester) runWorker(ctx context.Context, workerID int, requestsPerWorker int, scenario LoadTestScenario) {
	requestInterval := time.Duration(1000000000/lt.config.TargetQPS) * time.Nanosecond * time.Duration(lt.config.NumWorkers)

	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	iteration := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-lt.stopChan:
			return
		case <-ticker.C:
			lt.executeRequest(ctx, scenario)
			iteration++
			if iteration >= requestsPerWorker && requestsPerWorker > 0 {
				return
			}
		}
	}
}

func (lt *LoadTester) executeRequest(ctx context.Context, scenario LoadTestScenario) {
	startTime := time.Now()

	req, err := scenario.Request()
	if err != nil {
		atomic.AddInt64(&lt.result.ErrorReqs, 1)
		lt.result.ErrorsByType["request_prepare"]++
		return
	}

	resp, err := lt.client.Do(req.WithContext(ctx))
	latency := time.Since(startTime)

	if err != nil {
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			atomic.AddInt64(&lt.result.TimeoutReqs, 1)
			lt.result.ErrorsByType["timeout"]++
		} else {
			atomic.AddInt64(&lt.result.ErrorReqs, 1)
			lt.result.ErrorsByType["request_error"]++
		}
		return
	}

	defer resp.Body.Close()

	lt.result.BytesSent += req.ContentLength
	if resp.ContentLength > 0 {
		lt.result.BytesReceived += resp.ContentLength
	} else {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
		lt.result.BytesReceived += int64(len(body))
	}

	if err := scenario.ProcessResponse(resp, latency); err != nil {
		atomic.AddInt64(&lt.result.FailedReqs, 1)
		lt.result.ErrorsByType["process_error"]++
		return
	}

	atomic.AddInt64(&lt.result.TotalRequests, 1)
	atomic.AddInt64(&lt.result.SuccessfulReqs, 1)

	lt.mu.Lock()
	lt.result.LatencyArray = append(lt.result.LatencyArray, latency)
	lt.mu.Unlock()
}

func (lt *LoadTester) calculatePercentiles() {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if len(lt.result.LatencyArray) == 0 {
		return
	}

	lt.result.AvgLatency = calculateAverage(lt.result.LatencyArray)
	lt.result.MinLatency = lt.result.LatencyArray[0]
	lt.result.MaxLatency = lt.result.LatencyArray[len(lt.result.LatencyArray)-1]

	n := len(lt.result.LatencyArray)
	lt.result.P50Latency = lt.result.LatencyArray[n*50/100]
	lt.result.P90Latency = lt.result.LatencyArray[n*90/100]
	lt.result.P99Latency = lt.result.LatencyArray[n*99/100]
	lt.result.P999Latency = lt.result.LatencyArray[n*999/1000]
}

func calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var sum int64
	for _, d := range durations {
		sum += int64(d)
	}
	return time.Duration(sum / int64(len(durations)))
}

func sortDurations(durations []time.Duration) {
	for i := 1; i < len(durations); i++ {
		for j := i; j > 0 && durations[j] < durations[j-1]; j-- {
			durations[j], durations[j-1] = durations[j-1], durations[j]
		}
	}
}

func RunLoadTest(baseURL string, targetQPS int, duration time.Duration) (*LoadTestResult, error) {
	config := LoadTestConfig{
		TargetQPS:  targetQPS,
		Duration:   duration,
		NumWorkers: targetQPS / 100,
		BaseURL:   baseURL,
		Timeout:   10 * time.Second,
	}

	if config.NumWorkers < 10 {
		config.NumWorkers = 10
	}
	if config.NumWorkers > 1000 {
		config.NumWorkers = 1000
	}

	scenarios := []LoadTestScenario{
		&HealthCheckScenario{BaseURL: baseURL},
		&CaptchaGenerateScenario{BaseURL: baseURL, AppID: "test-app", ClientIP: "127.0.0.1"},
	}

	mixedScenario := &MixedLoadScenario{
		BaseURL:   baseURL,
		AppID:     "test-app",
		Scenarios: scenarios,
		Weights:   []int{30, 70},
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration+30*time.Second)
	defer cancel()

	tester := NewLoadTester(config)
	result, err := tester.Run(ctx, mixedScenario)

	return result, err
}

func RunCaptchaLoadTest(baseURL string, targetQPS int, duration time.Duration) (*LoadTestResult, error) {
	config := LoadTestConfig{
		TargetQPS:  targetQPS,
		Duration:   duration,
		NumWorkers: targetQPS / 100,
		BaseURL:   baseURL,
		Timeout:   10 * time.Second,
	}

	if config.NumWorkers < 10 {
		config.NumWorkers = 10
	}
	if config.NumWorkers > 1000 {
		config.NumWorkers = 1000
	}

	scenario := &CaptchaGenerateScenario{
		BaseURL:  baseURL,
		AppID:    "benchmark-app",
		ClientIP: "127.0.0.1",
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration+30*time.Second)
	defer cancel()

	tester := NewLoadTester(config)
	result, err := tester.Run(ctx, scenario)

	return result, err
}

func RunHealthCheckLoadTest(baseURL string, targetQPS int, duration time.Duration) (*LoadTestResult, error) {
	config := LoadTestConfig{
		TargetQPS:  targetQPS,
		Duration:   duration,
		NumWorkers: targetQPS / 100,
		BaseURL:   baseURL,
		Timeout:   5 * time.Second,
	}

	if config.NumWorkers < 10 {
		config.NumWorkers = 10
	}
	if config.NumWorkers > 1000 {
		config.NumWorkers = 1000
	}

	scenario := &HealthCheckScenario{BaseURL: baseURL}

	ctx, cancel := context.WithTimeout(context.Background(), duration+30*time.Second)
	defer cancel()

	tester := NewLoadTester(config)
	result, err := tester.Run(ctx, scenario)

	return result, err
}

func PrintLoadTestResult(result *LoadTestResult, scenarioName string) {
	fmt.Printf("\n=== Load Test Results: %s ===\n", scenarioName)
	fmt.Printf("Total Duration:      %v\n", result.TotalDuration)
	fmt.Printf("Total Requests:       %d\n", result.TotalRequests)
	fmt.Printf("Successful Requests:  %d\n", result.SuccessfulReqs)
	fmt.Printf("Failed Requests:      %d\n", result.FailedReqs)
	fmt.Printf("Timeout Requests:     %d\n", result.TimeoutReqs)
	fmt.Printf("Error Requests:       %d\n", result.ErrorReqs)
	fmt.Printf("Requests/sec (QPS):   %.2f\n", result.RequestsPerSec)

	if result.TotalRequests > 0 {
		errorRate := float64(result.FailedReqs+result.TimeoutReqs+result.ErrorReqs) / float64(result.TotalRequests) * 100
		fmt.Printf("Error Rate:           %.2f%%\n", errorRate)
	}

	fmt.Printf("\nLatency Statistics:\n")
	fmt.Printf("  Average:   %v\n", result.AvgLatency)
	fmt.Printf("  Min:      %v\n", result.MinLatency)
	fmt.Printf("  Max:      %v\n", result.MaxLatency)
	fmt.Printf("  P50:      %v\n", result.P50Latency)
	fmt.Printf("  P90:      %v\n", result.P90Latency)
	fmt.Printf("  P99:      %v\n", result.P99Latency)
	fmt.Printf("  P99.9:    %v\n", result.P999Latency)

	fmt.Printf("\nData Transfer:\n")
	fmt.Printf("  Bytes Sent:     %d\n", result.BytesSent)
	fmt.Printf("  Bytes Received: %d\n", result.BytesReceived)

	fmt.Printf("\nErrors by Type:\n")
	for errType, count := range result.ErrorsByType {
		fmt.Printf("  %s: %d\n", errType, count)
	}
}
