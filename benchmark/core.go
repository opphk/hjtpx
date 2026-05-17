package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"runtime"
	"runtime/pprof"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	BenchmarkBaseURL   = "http://localhost:8080"
	BenchmarkDBDSN     = "host=localhost user=postgres password=postgres dbname=hjtpx port=5432 sslmode=disable"
	BenchmarkRedisAddr = "localhost:6379"
	BenchmarkRedisDB   = 0
	BenchmarkRedisPool = 100
)

type BenchmarkScenario struct {
	Name        string
	Description string
	Endpoint    string
	Method      string
	Body        map[string]interface{}
	Headers     map[string]string
	Concurrency int
	Duration    time.Duration
	AppID       uint
}

type ScenarioCategory string

const (
	CategoryNormal  ScenarioCategory = "normal"
	CategoryPeak    ScenarioCategory = "peak"
	CategoryAbnormal ScenarioCategory = "abnormal"
)

type ScenarioType string

const (
	TypeSliderCaptcha    ScenarioType = "slider_captcha"
	TypeClickCaptcha     ScenarioType = "click_captcha"
	TypeImageCaptcha     ScenarioType = "image_captcha"
	TypeGestureCaptcha   ScenarioType = "gesture_captcha"
	TypeJigsawCaptcha    ScenarioType = "jigsaw_captcha"
	TypeSeamlessVerify   ScenarioType = "seamless_verify"
	TypeAdminAPI         ScenarioType = "admin_api"
	TypeHealthCheck      ScenarioType = "health_check"
)

var NormalScenarios = []BenchmarkScenario{
	{
		Name:        "Slider Captcha Generate - Normal",
		Description: "Slider captcha generation with background image and slider",
		Endpoint:    "/api/v1/captcha/slider",
		Method:      "GET",
		Body: map[string]interface{}{
			"app_id":      1,
			"width":       320,
			"height":      160,
			"slider_size": 40,
		},
		Concurrency: 50,
		Duration:    1 * time.Minute,
		AppID:       1,
	},
	{
		Name:        "Click Captcha Generate - Normal",
		Description: "Click-based captcha generation with target icons",
		Endpoint:    "/api/v1/captcha/click",
		Method:      "GET",
		Body: map[string]interface{}{
			"app_id":       1,
			"width":        320,
			"height":       160,
			"target_count": 4,
		},
		Concurrency: 50,
		Duration:    1 * time.Minute,
		AppID:       1,
	},
	{
		Name:        "Image Captcha Generate - Normal",
		Description: "Traditional image captcha generation",
		Endpoint:    "/api/v1/captcha/image/generate",
		Method:      "POST",
		Body: map[string]interface{}{
			"app_id": 1,
			"length": 4,
			"width":  120,
			"height": 40,
		},
		Concurrency: 100,
		Duration:    1 * time.Minute,
		AppID:       1,
	},
	{
		Name:        "Health Check - Normal",
		Description: "Health check endpoint",
		Endpoint:    "/health",
		Method:      "GET",
		Body:        nil,
		Concurrency: 200,
		Duration:    1 * time.Minute,
		AppID:       0,
	},
}

var PeakScenarios = []BenchmarkScenario{
	{
		Name:        "Slider Captcha Generate - Peak",
		Description: "Slider captcha generation under peak load",
		Endpoint:    "/api/v1/captcha/slider",
		Method:      "GET",
		Body: map[string]interface{}{
			"app_id":      1,
			"width":       320,
			"height":      160,
			"slider_size": 40,
		},
		Concurrency: 500,
		Duration:    2 * time.Minute,
		AppID:       1,
	},
	{
		Name:        "Click Captcha Generate - Peak",
		Description: "Click captcha generation under peak load",
		Endpoint:    "/api/v1/captcha/click",
		Method:      "GET",
		Body: map[string]interface{}{
			"app_id":       1,
			"width":        320,
			"height":       160,
			"target_count": 4,
		},
		Concurrency: 500,
		Duration:    2 * time.Minute,
		AppID:       1,
	},
	{
		Name:        "Image Captcha Generate - Peak",
		Description: "Image captcha generation under peak load",
		Endpoint:    "/api/v1/captcha/image/generate",
		Method:      "POST",
		Body: map[string]interface{}{
			"app_id": 1,
			"length": 4,
			"width":  120,
			"height": 40,
		},
		Concurrency: 800,
		Duration:    2 * time.Minute,
		AppID:       1,
	},
	{
		Name:        "Mixed Captcha - Peak",
		Description: "Mixed captcha types under peak load",
		Endpoint:    "/api/v1/captcha/image/generate",
		Method:      "POST",
		Body: map[string]interface{}{
			"app_id": 1,
			"length": 4,
			"width":  120,
			"height": 40,
		},
		Concurrency: 1000,
		Duration:    2 * time.Minute,
		AppID:       1,
	},
}

var AbnormalScenarios = []BenchmarkScenario{
	{
		Name:        "Invalid Captcha Verify - Abnormal",
		Description: "Verify with invalid session",
		Endpoint:    "/api/v1/captcha/verify",
		Method:      "POST",
		Body: map[string]interface{}{
			"app_id":     1,
			"session_id": "invalid_session",
			"captcha":    "WRONG",
		},
		Concurrency: 100,
		Duration:    30 * time.Second,
		AppID:       1,
	},
	{
		Name:        "Large Payload - Abnormal",
		Description: "Send request with oversized payload",
		Endpoint:    "/api/v1/captcha/image/generate",
		Method:      "POST",
		Body: map[string]interface{}{
			"app_id": 1,
			"length": 1000,
			"width":  10000,
			"height": 10000,
		},
		Concurrency: 50,
		Duration:    30 * time.Second,
		AppID:       1,
	},
	{
		Name:        "Rapid Fire - Abnormal",
		Description: "Extremely rapid requests to test rate limiting",
		Endpoint:    "/api/v1/captcha/image/generate",
		Method:      "POST",
		Body: map[string]interface{}{
			"app_id": 1,
			"length": 4,
			"width":  120,
			"height": 40,
		},
		Concurrency: 2000,
		Duration:    30 * time.Second,
		AppID:       1,
	},
	{
		Name:        "Timeout Test - Abnormal",
		Description: "Requests with timeout simulation",
		Endpoint:    "/api/v1/captcha/slider",
		Method:      "GET",
		Body: map[string]interface{}{
			"app_id":      1,
			"width":       320,
			"height":      160,
			"slider_size": 40,
		},
		Concurrency: 100,
		Duration:    30 * time.Second,
		AppID:       1,
	},
}

var Scenarios = []BenchmarkScenario{
	{
		Name:        "Slider Captcha Generate",
		Description: "Slider captcha generation with background image and slider",
		Endpoint:    "/api/v1/captcha/slider",
		Method:      "GET",
		Body: map[string]interface{}{
			"app_id":      1,
			"width":       320,
			"height":      160,
			"slider_size": 40,
		},
		Concurrency: 100,
		Duration:    1 * time.Minute,
		AppID:       1,
	},
	{
		Name:        "Slider Captcha Verify",
		Description: "Slider captcha verification with track data",
		Endpoint:    "/api/v1/captcha/verify",
		Method:      "POST",
		Body: map[string]interface{}{
			"app_id":     1,
			"session_id": "benchmark_session",
			"x":          150,
			"track_data": []map[string]interface{}{
				{"x": 10, "y": 5, "t": 50},
				{"x": 30, "y": 8, "t": 100},
				{"x": 60, "y": 10, "t": 150},
				{"x": 100, "y": 12, "t": 200},
				{"x": 150, "y": 15, "t": 300},
			},
		},
		Concurrency: 100,
		Duration:    1 * time.Minute,
		AppID:       1,
	},
	{
		Name:        "Click Captcha Generate",
		Description: "Click-based captcha generation with target icons",
		Endpoint:    "/api/v1/captcha/click",
		Method:      "GET",
		Body: map[string]interface{}{
			"app_id":       1,
			"width":        320,
			"height":       160,
			"target_count": 4,
		},
		Concurrency: 50,
		Duration:    1 * time.Minute,
		AppID:       1,
	},
	{
		Name:        "Click Captcha Verify",
		Description: "Click-based captcha verification with click coordinates",
		Endpoint:    "/api/v1/captcha/verify",
		Method:      "POST",
		Body: map[string]interface{}{
			"app_id":     1,
			"session_id": "benchmark_session",
			"clicks": []map[string]interface{}{
				{"x": 100, "y": 80},
				{"x": 200, "y": 40},
				{"x": 150, "y": 120},
				{"x": 250, "y": 60},
			},
		},
		Concurrency: 50,
		Duration:    1 * time.Minute,
		AppID:       1,
	},
	{
		Name:        "Image Captcha Generate",
		Description: "Traditional image captcha generation",
		Endpoint:    "/api/v1/captcha/image/generate",
		Method:      "POST",
		Body: map[string]interface{}{
			"app_id": 1,
			"length": 4,
			"width":  120,
			"height": 40,
		},
		Concurrency: 200,
		Duration:    1 * time.Minute,
		AppID:       1,
	},
	{
		Name:        "Image Captcha Verify",
		Description: "Traditional image captcha verification",
		Endpoint:    "/api/v1/captcha/verify",
		Method:      "POST",
		Body: map[string]interface{}{
			"app_id":     1,
			"session_id": "benchmark_session",
			"captcha":    "ABCD",
		},
		Concurrency: 200,
		Duration:    1 * time.Minute,
		AppID:       1,
	},
}

type PerformanceMetrics struct {
	Name               string
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TimeoutRequests    int64
	QPS                float64
	LatencyP50         time.Duration
	LatencyP95         time.Duration
	LatencyP99         time.Duration
	AvgLatency         time.Duration
	MinLatency         time.Duration
	MaxLatency         time.Duration
	ErrorRate          float64
	TimeoutRate        float64
	CPUUsage           float64
	MemoryUsage        uint64
	MemoryAlloc        uint64
	StartTime          time.Time
	EndTime            time.Time
	Duration           time.Duration
	Latencies          []time.Duration
	mu                 sync.Mutex
}

func NewPerformanceMetrics(name string) *PerformanceMetrics {
	return &PerformanceMetrics{
		Name:       name,
		StartTime:  time.Now(),
		Latencies:  make([]time.Duration, 0, 100000),
	}
}

func (m *PerformanceMetrics) RecordLatency(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Latencies = append(m.Latencies, d)
	m.TotalRequests++
	if len(m.Latencies)%5000 == 0 {
		m.calculateLatencyPercentiles()
	}
}

func (m *PerformanceMetrics) RecordSuccess() {
	atomic.AddInt64(&m.SuccessfulRequests, 1)
}

func (m *PerformanceMetrics) RecordFailure() {
	atomic.AddInt64(&m.FailedRequests, 1)
}

func (m *PerformanceMetrics) RecordTimeout() {
	atomic.AddInt64(&m.TimeoutRequests, 1)
	atomic.AddInt64(&m.FailedRequests, 1)
}

func (m *PerformanceMetrics) calculateLatencyPercentiles() {
	if len(m.Latencies) == 0 {
		return
	}

	sorted := make([]time.Duration, len(m.Latencies))
	copy(sorted, m.Latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	n := len(sorted)
	if n == 0 {
		return
	}

	p50Idx := n * 50 / 100
	p95Idx := n * 95 / 100
	p99Idx := n * 99 / 100

	if p50Idx < n {
		m.LatencyP50 = sorted[p50Idx]
	}
	if p95Idx < n {
		m.LatencyP95 = sorted[p95Idx]
	}
	if p99Idx < n {
		m.LatencyP99 = sorted[p99Idx]
	}

	var totalLatency time.Duration
	for _, l := range sorted {
		totalLatency += l
	}
	m.AvgLatency = totalLatency / time.Duration(n)
	m.MinLatency = sorted[0]
	m.MaxLatency = sorted[n-1]
}

func (m *PerformanceMetrics) CalculateFinalMetrics() {
	m.calculateLatencyPercentiles()
	m.EndTime = time.Now()
	m.Duration = m.EndTime.Sub(m.StartTime)

	if m.Duration > 0 {
		m.QPS = float64(m.TotalRequests) / m.Duration.Seconds()
	}

	if m.TotalRequests > 0 {
		m.ErrorRate = float64(m.FailedRequests) / float64(m.TotalRequests) * 100
		m.TimeoutRate = float64(m.TimeoutRequests) / float64(m.TotalRequests) * 100
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	m.MemoryUsage = memStats.Sys
	m.MemoryAlloc = memStats.Alloc

	m.CPUUsage = getCPUUsage()
}

func getCPUUsage() float64 {
	var start time.Time
	start = time.Now()
	time.Sleep(100 * time.Millisecond)
	elapsed := time.Since(start)
	return float64(elapsed.Milliseconds()) / 100.0 * 100.0
}

type ScenarioResult struct {
	Scenario      BenchmarkScenario
	Metrics       *PerformanceMetrics
	Errors        []error
	StartTime     time.Time
	EndTime       time.Time
	Category      ScenarioCategory
	ScenarioType  ScenarioType
}

func RunScenario(scenario BenchmarkScenario) *ScenarioResult {
	result := &ScenarioResult{
		Scenario:  scenario,
		Metrics:   NewPerformanceMetrics(scenario.Name),
		StartTime: time.Now(),
		Errors:    make([]error, 0),
	}

	var wg sync.WaitGroup
	workerCount := scenario.Concurrency

	requestChan := make(chan struct{}, workerCount)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-requestChan:
					executeRequest(scenario, result)
				case <-time.After(scenario.Duration):
					return
				}
			}
		}()
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			requestChan <- struct{}{}
			if time.Since(result.StartTime) >= scenario.Duration {
				close(requestChan)
				break
			}
		default:
			if time.Since(result.StartTime) >= scenario.Duration {
				close(requestChan)
				goto done
			}
		}
	}

done:
	wg.Wait()
	result.Metrics.CalculateFinalMetrics()
	result.EndTime = time.Now()

	return result
}

func RunScenarioWithRampUp(scenario BenchmarkScenario) *ScenarioResult {
	result := &ScenarioResult{
		Scenario:  scenario,
		Metrics:   NewPerformanceMetrics(scenario.Name),
		StartTime: time.Now(),
		Errors:    make([]error, 0),
	}

	rampUpDuration := scenario.Duration / 5
	steadyStateDuration := scenario.Duration - rampUpDuration

	steadyStateConcurrency := scenario.Concurrency
	rampUpSteps := 10
	stepDuration := rampUpDuration / time.Duration(rampUpSteps)

	var wg sync.WaitGroup

	for step := 0; step < rampUpSteps; step++ {
		currentConcurrency := (steadyStateConcurrency * (step + 1)) / rampUpSteps

		for i := 0; i < currentConcurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ticker := time.NewTicker(stepDuration)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						executeRequest(scenario, result)
					}
				}
			}()
		}

		time.Sleep(stepDuration)
	}

	for i := 0; i < steadyStateConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					executeRequest(scenario, result)
				}
			}
		}()
	}

	time.Sleep(steadyStateDuration)
	result.Metrics.CalculateFinalMetrics()
	result.EndTime = time.Now()

	return result
}

func executeRequest(scenario BenchmarkScenario, result *ScenarioResult) {
	startTime := time.Now()

	var body []byte
	var err error
	if scenario.Body != nil {
		body, err = json.Marshal(scenario.Body)
		if err != nil {
			result.Errors = append(result.Errors, err)
			result.Metrics.RecordFailure()
			return
		}
	}

	reqURL := BenchmarkBaseURL + scenario.Endpoint
	if scenario.Method == "GET" && scenario.Body != nil {
		if params, err := json.Marshal(scenario.Body); err == nil {
			reqURL = fmt.Sprintf("%s?params=%s", reqURL, url.QueryEscape(string(params)))
		}
	}

	var req *http.Request
	if scenario.Method == "GET" {
		req, err = http.NewRequest(scenario.Method, reqURL, nil)
	} else {
		req, err = http.NewRequest(scenario.Method, reqURL, bytes.NewBuffer(body))
	}

	if err != nil {
		result.Errors = append(result.Errors, err)
		result.Metrics.RecordFailure()
		return
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range scenario.Headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	resp, err := client.Do(req)
	latency := time.Since(startTime)

	if err != nil {
		if timeoutErr, ok := err.(interface{ Timeout() bool }); ok && timeoutErr.Timeout() {
			result.Metrics.RecordTimeout()
		} else {
			result.Metrics.RecordFailure()
		}
		result.Errors = append(result.Errors, err)
		result.Metrics.RecordLatency(latency)
		return
	}
	defer resp.Body.Close()

	result.Metrics.RecordLatency(latency)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Metrics.RecordSuccess()
	} else {
		result.Metrics.RecordFailure()
	}
}

func RunScenarioOnce(scenario BenchmarkScenario) *ScenarioResult {
	result := &ScenarioResult{
		Scenario:  scenario,
		Metrics:   NewPerformanceMetrics(scenario.Name),
		StartTime: time.Now(),
		Errors:    make([]error, 0),
	}

	executeRequest(scenario, result)

	result.Metrics.CalculateFinalMetrics()
	result.EndTime = time.Now()

	return result
}

type BenchmarkReport struct {
	Summary          string
	Metrics          []*PerformanceMetrics
	Recommendations  []string
	GeneratedAt      time.Time
	SystemInfo       SystemInfo
	PerformanceGoals PerformanceGoals
	Analysis         PerformanceAnalysis
}

type PerformanceGoals struct {
	TargetQPS         float64
	TargetP99Latency  time.Duration
	TargetErrorRate   float64
	TargetCPULimit    float64
	TargetMemoryLimit uint64
}

type PerformanceAnalysis struct {
	OverallScore      float64
	QPSScore          float64
	LatencyScore      float64
	ErrorRateScore    float64
	Bottlenecks       []string
	Recommendations   []Recommendation
}

type Recommendation struct {
	Category    string
	Priority    string
	Title       string
	Description string
	Impact      string
}

type SystemInfo struct {
	CPUCores        int
	CPUModel        string
	GoVersion       string
	OS              string
	Arch            string
	TotalMemory     uint64
	AvailableMemory uint64
	NumGoroutine    int
}

func GetSystemInfo() SystemInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return SystemInfo{
		CPUCores:        runtime.NumCPU(),
		GoVersion:       runtime.Version(),
		OS:               runtime.GOOS,
		Arch:             runtime.GOARCH,
		TotalMemory:      memStats.Sys,
		AvailableMemory:  memStats.Sys - memStats.Alloc,
		NumGoroutine:     runtime.NumGoroutine(),
	}
}

func GenerateReport(metrics []*PerformanceMetrics) *BenchmarkReport {
	report := &BenchmarkReport{
		GeneratedAt: time.Now(),
		Metrics:      metrics,
		PerformanceGoals: PerformanceGoals{
			TargetQPS:        10000,
			TargetP99Latency: 50 * time.Millisecond,
			TargetErrorRate:  1.0,
			TargetCPULimit:   80.0,
			TargetMemoryLimit: 512 * 1024 * 1024,
		},
	}

	totalQPS := 0.0
	totalErrors := 0.0
	totalRequests := int64(0)
	avgP99Latency := time.Duration(0)

	for _, m := range metrics {
		totalQPS += m.QPS
		totalErrors += m.ErrorRate
		totalRequests += m.TotalRequests
		avgP99Latency += m.LatencyP99
	}

	if len(metrics) > 0 {
		avgP99Latency = avgP99Latency / time.Duration(len(metrics))
	}

	report.Summary = fmt.Sprintf("Benchmark completed with total QPS: %.2f, average error rate: %.2f%%, average P99 latency: %v",
		totalQPS, totalErrors/float64(len(metrics)), avgP99Latency)

	report.Recommendations = generateRecommendations(metrics)
	report.SystemInfo = GetSystemInfo()
	report.Analysis = analyzePerformance(metrics, report.PerformanceGoals)

	return report
}

func generateRecommendations(metrics []*PerformanceMetrics) []string {
	var recommendations []string

	for _, m := range metrics {
		if m.LatencyP99 > 100*time.Millisecond {
			recommendations = append(recommendations,
				fmt.Sprintf("Optimize %s: P99 latency is %.2fms (target: <50ms)",
					m.Name, float64(m.LatencyP99)/float64(time.Millisecond)))
		}

		if m.ErrorRate > 1.0 {
			recommendations = append(recommendations,
				fmt.Sprintf("Investigate errors in %s: Error rate is %.2f%%",
					m.Name, m.ErrorRate))
		}

		if m.QPS < 10000 {
			recommendations = append(recommendations,
				fmt.Sprintf("Scale %s: QPS is %.2f (target: >10000)",
					m.Name, m.QPS))
		}

		if m.MemoryUsage > 512*1024*1024 {
			recommendations = append(recommendations,
				fmt.Sprintf("Memory optimization needed for %s: Using %.2f MB",
					m.Name, float64(m.MemoryUsage)/1024/1024))
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "All metrics within acceptable ranges")
	}

	return recommendations
}

func analyzePerformance(metrics []*PerformanceMetrics, goals PerformanceGoals) PerformanceAnalysis {
	analysis := PerformanceAnalysis{
		Bottlenecks:     make([]string, 0),
		Recommendations: make([]Recommendation, 0),
	}

	var totalQPS, totalP99, totalErrorRate float64
	for _, m := range metrics {
		totalQPS += m.QPS
		totalP99 += float64(m.LatencyP99.Milliseconds())
		totalErrorRate += m.ErrorRate
	}

	if len(metrics) > 0 {
		avgQPS := totalQPS / float64(len(metrics))
		avgP99 := totalP99 / float64(len(metrics))
		avgErrorRate := totalErrorRate / float64(len(metrics))

		analysis.QPSScore = math.Min(100, (avgQPS/goals.TargetQPS)*100)
		analysis.LatencyScore = math.Max(0, 100-(avgP99/float64(goals.TargetP99Latency.Milliseconds())*100))
		analysis.ErrorRateScore = math.Max(0, 100-avgErrorRate)

		analysis.OverallScore = (analysis.QPSScore + analysis.LatencyScore + analysis.ErrorRateScore) / 3

		if analysis.QPSScore < 50 {
			analysis.Bottlenecks = append(analysis.Bottlenecks, "Low QPS - need horizontal scaling or caching")
			analysis.Recommendations = append(analysis.Recommendations, Recommendation{
				Category:  "scaling",
				Priority:  "high",
				Title:     "Implement horizontal scaling",
				Impact:    "Can increase QPS by 2-5x",
			})
		}

		if analysis.LatencyScore < 50 {
			analysis.Bottlenecks = append(analysis.Bottlenecks, "High latency - need code optimization")
			analysis.Recommendations = append(analysis.Recommendations, Recommendation{
				Category:  "optimization",
				Priority:  "high",
				Title:     "Optimize database queries and add caching",
				Impact:    "Can reduce latency by 30-50%",
			})
		}

		if analysis.ErrorRateScore < 50 {
			analysis.Bottlenecks = append(analysis.Bottlenecks, "High error rate - need error handling")
			analysis.Recommendations = append(analysis.Recommendations, Recommendation{
				Category:  "reliability",
				Priority:  "high",
				Title:     "Improve error handling and retry logic",
				Impact:    "Can reduce error rate by 50-80%",
			})
		}
	}

	return analysis
}

type DBOptimizer struct {
	enableQueryCache     bool
	queryCacheSize       int
	enableConnectionPool bool
	maxOpenConns         int
	maxIdleConns         int
	connMaxLifetime      time.Duration
	enablePreparedStmt   bool
}

func NewDBOptimizer() *DBOptimizer {
	return &DBOptimizer{
		enableQueryCache:     true,
		queryCacheSize:       1000,
		enableConnectionPool: true,
		maxOpenConns:         100,
		maxIdleConns:         10,
		connMaxLifetime:      time.Hour,
		enablePreparedStmt:   true,
	}
}

func (o *DBOptimizer) AddIndexes() []string {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_verifications_app_id ON verifications(application_id)`,
		`CREATE INDEX IF NOT EXISTS idx_verifications_user_id ON verifications(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_verifications_session_id ON verifications(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_verifications_created_at ON verifications(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_verifications_captcha_type ON verifications(captcha_type)`,
		`CREATE INDEX IF NOT EXISTS idx_verification_logs_verification_id ON verification_logs(verification_id)`,
		`CREATE INDEX IF NOT EXISTS idx_verification_logs_session_id ON verification_logs(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_verification_logs_application_id ON verification_logs(application_id)`,
		`CREATE INDEX IF NOT EXISTS idx_verification_logs_created_at ON verification_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_blacklist_target ON blacklists(target)`,
		`CREATE INDEX IF NOT EXISTS idx_blacklist_type ON blacklists(type)`,
		`CREATE INDEX IF NOT EXISTS idx_device_fingerprints_fingerprint ON device_fingerprints(fingerprint)`,
		`CREATE INDEX IF NOT EXISTS idx_device_fingerprints_ip_address ON device_fingerprints(ip_address)`,
	}

	return indexes
}

func (o *DBOptimizer) CreateCompositeIndexes() []string {
	compositeIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_verifications_app_status ON verifications(application_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_verifications_app_created ON verifications(application_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_verification_logs_app_created ON verification_logs(application_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_device_fingerprints_ip_fp ON device_fingerprints(ip_address, fingerprint)`,
	}

	return compositeIndexes
}

type CacheOptimizer struct {
	keyPrefix      string
	defaultTTL     time.Duration
	enableWarmup   bool
	enablePrefetch bool
	prefetchSize   int
	hitRateHistory []float64
	mu             sync.RWMutex
}

func NewCacheOptimizer() *CacheOptimizer {
	return &CacheOptimizer{
		keyPrefix:      "hjtpx:cache:",
		defaultTTL:     5 * time.Minute,
		enableWarmup:   true,
		enablePrefetch: true,
		prefetchSize:   100,
		hitRateHistory: make([]float64, 0, 1000),
	}
}

func (o *CacheOptimizer) OptimizeTTL(accessCount int64) time.Duration {
	baseTTL := o.defaultTTL

	if accessCount > 10000 {
		baseTTL = 30 * time.Minute
	} else if accessCount > 1000 {
		baseTTL = 15 * time.Minute
	} else if accessCount > 100 {
		baseTTL = 10 * time.Minute
	}

	return baseTTL
}

func (o *CacheOptimizer) RecordHitRate(hitRate float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.hitRateHistory = append(o.hitRateHistory, hitRate)
	if len(o.hitRateHistory) > 1000 {
		o.hitRateHistory = o.hitRateHistory[1:]
	}
}

func (o *CacheOptimizer) GetAverageHitRate() float64 {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if len(o.hitRateHistory) == 0 {
		return 0
	}

	var sum float64
	for _, hr := range o.hitRateHistory {
		sum += hr
	}

	return sum / float64(len(o.hitRateHistory))
}

func (o *CacheOptimizer) ShouldWarmup() bool {
	return o.enableWarmup && o.GetAverageHitRate() < 0.8
}

func (o *CacheOptimizer) GenerateCacheKey(parts ...string) string {
	result := o.keyPrefix
	for i, part := range parts {
		if i > 0 {
			result += ":"
		}
		result += part
	}
	return result
}

type WorkerPool struct {
	workers     int
	jobQueue    chan func() interface{}
	resultQueue chan interface{}
	wg          sync.WaitGroup
	running     atomic.Bool
}

func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	return &WorkerPool{
		workers:     workers,
		jobQueue:    make(chan func() interface{}, queueSize),
		resultQueue: make(chan interface{}, queueSize),
	}
}

func (wp *WorkerPool) Start() {
	if wp.running.Load() {
		return
	}

	wp.running.Store(true)

	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()

	for job := range wp.jobQueue {
		result := job()
		wp.resultQueue <- result
	}
}

func (wp *WorkerPool) Submit(job func() interface{}) {
	wp.jobQueue <- job
}

func (wp *WorkerPool) SubmitWithTimeout(job func() interface{}, timeout time.Duration) (interface{}, bool) {
	resultChan := make(chan interface{}, 1)

	select {
	case wp.jobQueue <- func() interface{} {
		result := job()
		resultChan <- result
		return result
	}:
		select {
		case result := <-resultChan:
			return result, true
		case <-time.After(timeout):
			return nil, false
		}
	default:
		return nil, false
	}
}

func (wp *WorkerPool) Stop() {
	if !wp.running.Load() {
		return
	}

	wp.running.Store(false)
	close(wp.jobQueue)
	wp.wg.Wait()
	close(wp.resultQueue)
}

func (wp *WorkerPool) GetQueueLength() int {
	return len(wp.jobQueue)
}

func (wp *WorkerPool) IsRunning() bool {
	return wp.running.Load()
}

type ResponsePool struct {
	pool sync.Pool
}

func NewResponsePool() *ResponsePool {
	return &ResponsePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &CachedResponse{
					StatusCode: 200,
					Headers:    make(map[string]string),
					Body:       make([]byte, 0, 4096),
				}
			},
		},
	}
}

type CachedResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	CreatedAt  time.Time
}

func (rp *ResponsePool) Get() *CachedResponse {
	resp := rp.pool.Get().(*CachedResponse)
	resp.CreatedAt = time.Now()
	resp.Headers = make(map[string]string)
	resp.Body = resp.Body[:0]
	return resp
}

func (rp *ResponsePool) Put(resp *CachedResponse) {
	rp.pool.Put(resp)
}

type BatchOperation struct {
	batchSize    int
	maxBatchSize int
	mu           sync.Mutex
}

func NewBatchOperation(batchSize int) *BatchOperation {
	return &BatchOperation{
		batchSize:    batchSize,
		maxBatchSize: 1000,
	}
}

func (bo *BatchOperation) SetBatchSize(size int) {
	bo.mu.Lock()
	defer bo.mu.Unlock()

	if size > 0 && size <= bo.maxBatchSize {
		bo.batchSize = size
	}
}

type CachedQuery struct {
	Result      interface{}
	ExpiresAt   time.Time
	AccessCount int64
	LastAccess  time.Time
}

type QueryCache struct {
	cache     map[string]*CachedQuery
	mu        sync.RWMutex
	maxSize   int
	hits      int64
	misses    int64
	evictions int64
}

func NewQueryCache(maxSize int) *QueryCache {
	return &QueryCache{
		cache:   make(map[string]*CachedQuery),
		maxSize: maxSize,
	}
}

func (qc *QueryCache) Get(key string) (interface{}, bool) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	cached, exists := qc.cache[key]
	if !exists {
		atomic.AddInt64(&qc.misses, 1)
		return nil, false
	}

	if time.Now().After(cached.ExpiresAt) {
		return nil, false
	}

	atomic.AddInt64(&qc.hits, 1)
	cached.AccessCount++
	cached.LastAccess = time.Now()

	return cached.Result, true
}

func (qc *QueryCache) Set(key string, result interface{}, ttl time.Duration) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	if len(qc.cache) >= qc.maxSize {
		qc.evict()
	}

	qc.cache[key] = &CachedQuery{
		Result:      result,
		ExpiresAt:   time.Now().Add(ttl),
		AccessCount: 1,
		LastAccess:  time.Now(),
	}
}

func (qc *QueryCache) evict() {
	var oldestKey string
	var oldestTime time.Time

	for key, cached := range qc.cache {
		if oldestTime.IsZero() || cached.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = cached.LastAccess
		}
	}

	if oldestKey != "" {
		delete(qc.cache, oldestKey)
		atomic.AddInt64(&qc.evictions, 1)
	}
}

func (qc *QueryCache) GetHitRate() float64 {
	total := atomic.LoadInt64(&qc.hits) + atomic.LoadInt64(&qc.misses)
	if total == 0 {
		return 0
	}
	return float64(atomic.LoadInt64(&qc.hits)) / float64(total)
}

func (qc *QueryCache) Clear() {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	qc.cache = make(map[string]*CachedQuery)
	atomic.StoreInt64(&qc.hits, 0)
	atomic.StoreInt64(&qc.misses, 0)
}

type RateLimiter struct {
	requests    map[string][]time.Time
	mu          sync.RWMutex
	maxRequests int
	window      time.Duration
}

func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests:    make(map[string][]time.Time),
		maxRequests: maxRequests,
		window:      window,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	requests := rl.requests[key]
	var validRequests []time.Time

	for _, t := range requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	if len(validRequests) >= rl.maxRequests {
		rl.requests[key] = validRequests
		return false
	}

	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests

	return true
}

func (rl *RateLimiter) GetRemaining(key string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	requests := rl.requests[key]
	count := 0

	for _, t := range requests {
		if t.After(windowStart) {
			count++
		}
	}

	remaining := rl.maxRequests - count
	if remaining < 0 {
		return 0
	}

	return remaining
}

func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.requests, key)
}

type CapacityPlanner struct {
	currentCapacity     int
	targetQPS          float64
	peakQPS            float64
	scalingThreshold   float64
	costPerInstance    float64
	maxInstances       int
}

func NewCapacityPlanner(targetQPS float64) *CapacityPlanner {
	return &CapacityPlanner{
		currentCapacity:   1,
		targetQPS:         targetQPS,
		peakQPS:           targetQPS * 2,
		scalingThreshold:  0.7,
		costPerInstance:   0.05,
		maxInstances:      100,
	}
}

func (cp *CapacityPlanner) CalculateRequiredInstances(currentQPS float64) int {
	if currentQPS <= 0 {
		return cp.currentCapacity
	}

	instances := int(math.Ceil(cp.targetQPS / currentQPS))
	if instances < 1 {
		instances = 1
	}
	if instances > cp.maxInstances {
		instances = cp.maxInstances
	}

	return instances
}

func (cp *CapacityPlanner) ShouldScale(currentQPS float64) bool {
	utilization := currentQPS / cp.targetQPS
	return utilization > cp.scalingThreshold || utilization < 0.3
}

func (cp *CapacityPlanner) EstimateCost(instances int, duration time.Duration) float64 {
	hours := duration.Hours()
	return float64(instances) * cp.costPerInstance * hours
}

func (cp *CapacityPlanner) GetScalingRecommendation(currentQPS float64) string {
	instances := cp.CalculateRequiredInstances(currentQPS)

	if instances > cp.currentCapacity {
		return fmt.Sprintf("Scale up: Need %d instances (current: %d)", instances, cp.currentCapacity)
	} else if instances < cp.currentCapacity {
		return fmt.Sprintf("Scale down: Can reduce to %d instances (current: %d)", instances, cp.currentCapacity)
	}

	return "No scaling needed"
}

type ConnectionPool struct {
	maxOpen      int
	maxIdle      int
	maxLifetime  time.Duration
	currentOpen  int64
	mu           sync.RWMutex
}

func NewConnectionPool(maxOpen, maxIdle int, maxLifetime time.Duration) *ConnectionPool {
	return &ConnectionPool{
		maxOpen:     maxOpen,
		maxIdle:     maxIdle,
		maxLifetime: maxLifetime,
	}
}

func (cp *ConnectionPool) Acquire() bool {
	atomic.AddInt64(&cp.currentOpen, 1)
	return true
}

func (cp *ConnectionPool) Release() {
	atomic.AddInt64(&cp.currentOpen, -1)
}

func (cp *ConnectionPool) GetStats() (open, idle int) {
	open = int(atomic.LoadInt64(&cp.currentOpen))
	return open, cp.maxIdle
}

func ValidateBenchmarkConfig() error {
	if BenchmarkBaseURL == "" {
		return fmt.Errorf("benchmark base URL is required")
	}

	_, err := url.Parse(BenchmarkBaseURL)
	if err != nil {
		return fmt.Errorf("invalid benchmark base URL: %w", err)
	}

	return nil
}

func RunAllScenarios() []*ScenarioResult {
	results := make([]*ScenarioResult, 0, len(Scenarios))

	for _, scenario := range Scenarios {
		result := RunScenario(scenario)
		results = append(results, result)
	}

	return results
}

func RunScenariosByCategory(category ScenarioCategory) []*ScenarioResult {
	var scenarios []BenchmarkScenario

	switch category {
	case CategoryNormal:
		scenarios = NormalScenarios
	case CategoryPeak:
		scenarios = PeakScenarios
	case CategoryAbnormal:
		scenarios = AbnormalScenarios
	default:
		scenarios = Scenarios
	}

	results := make([]*ScenarioResult, 0, len(scenarios))

	for _, scenario := range scenarios {
		result := RunScenario(scenario)
		result.Category = category
		results = append(results, result)
	}

	return results
}

func RunProgressiveBenchmark() []*ScenarioResult {
	results := make([]*ScenarioResult, 0)

	concurrencyLevels := []int{10, 50, 100, 200, 500}

	for _, concurrency := range concurrencyLevels {
		scenario := BenchmarkScenario{
			Name:        fmt.Sprintf("Image Captcha Generate - Concurrency %d", concurrency),
			Description: fmt.Sprintf("Testing with %d concurrent workers", concurrency),
			Endpoint:    "/api/v1/captcha/image/generate",
			Method:      "POST",
			Body: map[string]interface{}{
				"app_id": 1,
				"length": 4,
				"width":  120,
				"height": 40,
			},
			Concurrency: concurrency,
			Duration:    30 * time.Second,
			AppID:       1,
		}

		result := RunScenario(scenario)
		results = append(results, result)
	}

	return results
}

type Profiler struct {
	enabled bool
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewProfiler() *Profiler {
	return &Profiler{
		enabled: false,
	}
}

func (p *Profiler) Start() error {
	if p.enabled {
		return fmt.Errorf("profiler already running")
	}

	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.enabled = true

	return nil
}

func (p *Profiler) Stop() {
	if !p.enabled {
		return
	}

	p.cancel()
	p.enabled = false
}

func (p *Profiler) GetCPUProfile() ([]byte, error) {
	if !p.enabled {
		return nil, fmt.Errorf("profiler not running")
	}

	var buf bytes.Buffer
	if err := pprof.Lookup("cpu").WriteTo(&buf, 0); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Profiler) GetMemProfile() ([]byte, error) {
	if !p.enabled {
		return nil, fmt.Errorf("profiler not running")
	}

	var buf bytes.Buffer
	if err := pprof.Lookup("heap").WriteTo(&buf, 0); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Profiler) WriteProfile(name string, w io.Writer) error {
	profile := pprof.Lookup(name)
	if profile == nil {
		return fmt.Errorf("profile %s not found", name)
	}

	return profile.WriteTo(w, 0)
}
