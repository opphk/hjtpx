package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/benchmark"
)

const (
	BenchmarkBaseURL = "http://localhost:8080"
)

func main() {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("PERFORMANCE BENCHMARK SUITE")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Started at: %s\n", time.Now().Format(time.RFC3339))
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("Initial Goroutines: %d\n", runtime.NumGoroutine())
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	if err := benchmark.ValidateBenchmarkConfig(); err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		fmt.Println("Please ensure the benchmark service is properly configured.")
		os.Exit(1)
	}

	if !checkServiceHealth() {
		fmt.Println("Warning: Benchmark service may not be running at", BenchmarkBaseURL)
		fmt.Println("Starting benchmarks anyway...")
	}

	results := runAllBenchmarks()

	printSummary(results)

	generateJSONReport(results)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("Benchmark completed at:", time.Now().Format(time.RFC3339))
	fmt.Println(strings.Repeat("=", 80))
}

func checkServiceHealth() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(BenchmarkBaseURL + "/api/v1/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func runAllBenchmarks() []*benchmark.ScenarioResult {
	results := make([]*benchmark.ScenarioResult, 0, len(benchmark.Scenarios))

	for i, scenario := range benchmark.Scenarios {
		fmt.Printf("\n[%d/%d] Running: %s\n", i+1, len(benchmark.Scenarios), scenario.Name)
		fmt.Printf("Description: %s\n", scenario.Description)
		fmt.Printf("Concurrency: %d | Duration: %v\n", scenario.Concurrency, scenario.Duration)

		result := benchmark.RunScenario(scenario)
		results = append(results, result)

		metrics := result.Metrics
		fmt.Printf("Results:\n")
		fmt.Printf("  - Total Requests: %d\n", metrics.TotalRequests)
		fmt.Printf("  - QPS: %.2f\n", metrics.QPS)
		fmt.Printf("  - P95 Latency: %v\n", metrics.LatencyP95)
		fmt.Printf("  - Error Rate: %.2f%%\n", metrics.ErrorRate)

		if len(result.Errors) > 0 {
			fmt.Printf("  - Errors: %d\n", len(result.Errors))
		}

		if i < len(benchmark.Scenarios)-1 {
			pauseDuration := 5 * time.Second
			fmt.Printf("\nPausing %v before next scenario...\n", pauseDuration)
			time.Sleep(pauseDuration)
		}
	}

	return results
}

func printSummary(results []*benchmark.ScenarioResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("BENCHMARK SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	totalQPS := 0.0
	totalRequests := int64(0)
	totalErrors := int64(0)

	for _, result := range results {
		metrics := result.Metrics
		totalQPS += metrics.QPS
		totalRequests += metrics.TotalRequests
		totalErrors += metrics.FailedRequests

		status := "PASS"
		if metrics.QPS < 5000 || metrics.LatencyP99 > 100*time.Millisecond || metrics.ErrorRate > 1.0 {
			status = "FAIL"
		}

		fmt.Printf("\n[%s] %s\n", status, metrics.Name)
		fmt.Printf("  QPS: %.2f (target: >5000)\n", metrics.QPS)
		fmt.Printf("  P99 Latency: %v (target: <100ms)\n", metrics.LatencyP99)
		fmt.Printf("  Error Rate: %.2f%%\n", metrics.ErrorRate)
		fmt.Printf("  Memory: %d bytes\n", metrics.MemoryUsage)
	}

	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Printf("\nTotal Combined QPS: %.2f\n", totalQPS)
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Total Errors: %d\n", totalErrors)

	if totalRequests > 0 {
		overallErrorRate := float64(totalErrors) / float64(totalRequests) * 100
		fmt.Printf("Overall Error Rate: %.2f%%\n", overallErrorRate)
	}

	report := benchmark.GenerateReport(collectMetrics(results))
	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("RECOMMENDATIONS")
	fmt.Println(strings.Repeat("-", 80))
	for _, rec := range report.Recommendations {
		fmt.Printf("- %s\n", rec)
	}
}

func collectMetrics(results []*benchmark.ScenarioResult) []*benchmark.PerformanceMetrics {
	metrics := make([]*benchmark.PerformanceMetrics, 0, len(results))
	for _, r := range results {
		metrics = append(metrics, r.Metrics)
	}
	return metrics
}

type ReportData struct {
	GeneratedAt    time.Time                     `json:"generated_at"`
	SystemInfo     benchmark.SystemInfo          `json:"system_info"`
	Results        []ScenarioResultData          `json:"results"`
	Summary        SummaryData                   `json:"summary"`
	Recommendations []string                      `json:"recommendations"`
}

type ScenarioResultData struct {
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	QPS                float64 `json:"qps"`
	LatencyP50         string  `json:"latency_p50"`
	LatencyP95         string  `json:"latency_p95"`
	LatencyP99         string  `json:"latency_p99"`
	AvgLatency         string  `json:"avg_latency"`
	ErrorRate          float64 `json:"error_rate"`
	MemoryUsage        uint64  `json:"memory_usage"`
	Duration           string  `json:"duration"`
}

type SummaryData struct {
	TotalQPS         float64 `json:"total_qps"`
	TotalRequests    int64   `json:"total_requests"`
	TotalErrors      int64   `json:"total_errors"`
	OverallErrorRate float64 `json:"overall_error_rate"`
	ScenariosPassed  int     `json:"scenarios_passed"`
	ScenariosFailed  int     `json:"scenarios_failed"`
}

func generateJSONReport(results []*benchmark.ScenarioResult) {
	reportData := ReportData{
		GeneratedAt: time.Now(),
		SystemInfo:  benchmark.GetSystemInfo(),
		Results:     make([]ScenarioResultData, 0, len(results)),
	}

	totalQPS := 0.0
	totalRequests := int64(0)
	totalErrors := int64(0)
	scenariosPassed := 0

	for _, result := range results {
		metrics := result.Metrics
		totalQPS += metrics.QPS
		totalRequests += metrics.TotalRequests
		totalErrors += metrics.FailedRequests

		if metrics.QPS >= 5000 && metrics.LatencyP99 <= 100*time.Millisecond && metrics.ErrorRate <= 1.0 {
			scenariosPassed++
		}

		reportData.Results = append(reportData.Results, ScenarioResultData{
			Name:               metrics.Name,
			Description:        result.Scenario.Description,
			TotalRequests:      metrics.TotalRequests,
			SuccessfulRequests: metrics.SuccessfulRequests,
			FailedRequests:     metrics.FailedRequests,
			QPS:                metrics.QPS,
			LatencyP50:         metrics.LatencyP50.String(),
			LatencyP95:         metrics.LatencyP95.String(),
			LatencyP99:         metrics.LatencyP99.String(),
			AvgLatency:         metrics.AvgLatency.String(),
			ErrorRate:          metrics.ErrorRate,
			MemoryUsage:        metrics.MemoryUsage,
			Duration:           metrics.Duration.String(),
		})
	}

	reportData.Summary = SummaryData{
		TotalQPS:         totalQPS,
		TotalRequests:    totalRequests,
		TotalErrors:      totalErrors,
		OverallErrorRate: 0,
	}

	if totalRequests > 0 {
		reportData.Summary.OverallErrorRate = float64(totalErrors) / float64(totalRequests) * 100
	}
	reportData.Summary.ScenariosPassed = scenariosPassed
	reportData.Summary.ScenariosFailed = len(results) - scenariosPassed

	report := benchmark.GenerateReport(collectMetrics(results))
	reportData.Recommendations = report.Recommendations

	filename := fmt.Sprintf("benchmark_report_%s.json", time.Now().Format("20060102_150405"))
	data, err := json.MarshalIndent(reportData, "", "  ")
	if err != nil {
		fmt.Printf("Error generating JSON report: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Printf("Error saving report: %v\n", err)
		return
	}

	fmt.Printf("\nJSON report saved to: %s\n", filename)
}

func executeHTTPRequest(method, url string, body interface{}, headers map[string]string) (int, []byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, bodyBytes, nil
}

type LoadTestConfig struct {
	Name        string
	BaseURL     string
	Endpoints   []LoadTestEndpoint
	Duration    time.Duration
	RampUpTime  time.Duration
	PeakLoad    int
	SustainLoad int
}

type LoadTestEndpoint struct {
	Path    string
	Method  string
	Body    map[string]interface{}
	Weight  int
}

func RunLoadTest(config LoadTestConfig) *benchmark.ScenarioResult {
	scenarios := make([]benchmark.BenchmarkScenario, 0)

	for _, endpoint := range config.Endpoints {
		scenarios = append(scenarios, benchmark.BenchmarkScenario{
			Name:        config.Name + "_" + endpoint.Path,
			Description: "Load test for " + endpoint.Path,
			Endpoint:    endpoint.Path,
			Method:      endpoint.Method,
			Body:        endpoint.Body,
			Concurrency: config.PeakLoad,
			Duration:    config.Duration,
		})
	}

	combinedResult := &benchmark.ScenarioResult{
		Metrics: benchmark.NewPerformanceMetrics(config.Name),
	}

	for _, scenario := range scenarios {
		result := benchmark.RunScenario(scenario)
		combinedResult.Metrics.TotalRequests += result.Metrics.TotalRequests
		combinedResult.Metrics.SuccessfulRequests += result.Metrics.SuccessfulRequests
		combinedResult.Metrics.FailedRequests += result.Metrics.FailedRequests
	}

	combinedResult.Metrics.CalculateFinalMetrics()
	return combinedResult
}

func printProgressBar(current, total int, width int) {
	percent := float64(current) / float64(total)
	filled := int(float64(width) * percent)
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	fmt.Printf("\r[%s] %.1f%%", bar, percent*100)

	if current == total {
		fmt.Println()
	}
}
