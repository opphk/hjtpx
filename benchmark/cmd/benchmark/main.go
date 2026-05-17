package main

import (
	"bytes"
	"encoding/json"
	"flag"
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

var (
	scenarioCategory string
	concurrency      int
	duration         int
	outputFormat     string
	runProgressive   bool
)

func init() {
	flag.StringVar(&scenarioCategory, "category", "all", "Benchmark category: normal, peak, abnormal, or all")
	flag.IntVar(&concurrency, "concurrency", 100, "Number of concurrent workers")
	flag.IntVar(&duration, "duration", 60, "Test duration in seconds")
	flag.StringVar(&outputFormat, "format", "text", "Output format: text, json, or html")
	flag.BoolVar(&runProgressive, "progressive", false, "Run progressive benchmark with increasing concurrency")
}

func main() {
	flag.Parse()

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("PERFORMANCE BENCHMARK SUITE")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Started at: %s\n", time.Now().Format(time.RFC3339))
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("Initial Goroutines: %d\n", runtime.NumGoroutine())
	fmt.Printf("Category: %s\n", scenarioCategory)
	fmt.Printf("Concurrency: %d\n", concurrency)
	fmt.Printf("Duration: %ds\n", duration)
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

	var results []*benchmark.ScenarioResult

	if runProgressive {
		fmt.Println("Running Progressive Benchmark...")
		results = runProgressiveBenchmark()
	} else {
		results = runBenchmarksByCategory()
	}

	printSummary(results)
	generateReports(results)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("Benchmark completed at:", time.Now().Format(time.RFC3339))
	fmt.Println(strings.Repeat("=", 80))
}

func checkServiceHealth() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(BenchmarkBaseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func runBenchmarksByCategory() []*benchmark.ScenarioResult {
	var scenarios []benchmark.BenchmarkScenario

	switch scenarioCategory {
	case "normal":
		scenarios = benchmark.NormalScenarios
		fmt.Println("Running Normal Load Scenarios...")
	case "peak":
		scenarios = benchmark.PeakScenarios
		fmt.Println("Running Peak Load Scenarios...")
	case "abnormal":
		scenarios = benchmark.AbnormalScenarios
		fmt.Println("Running Abnormal Condition Scenarios...")
	default:
		scenarios = benchmark.Scenarios
		fmt.Println("Running All Scenarios...")
	}

	results := make([]*benchmark.ScenarioResult, 0, len(scenarios))

	for i, scenario := range scenarios {
		fmt.Printf("\n[%d/%d] Running: %s\n", i+1, len(scenarios), scenario.Name)
		fmt.Printf("Description: %s\n", scenario.Description)
		fmt.Printf("Concurrency: %d | Duration: %v\n", scenario.Concurrency, scenario.Duration)

		result := benchmark.RunScenario(scenario)
		results = append(results, result)

		metrics := result.Metrics
		fmt.Printf("Results:\n")
		fmt.Printf("  - Total Requests: %d\n", metrics.TotalRequests)
		fmt.Printf("  - Successful: %d | Failed: %d\n", metrics.SuccessfulRequests, metrics.FailedRequests)
		fmt.Printf("  - QPS: %.2f\n", metrics.QPS)
		fmt.Printf("  - P50 Latency: %v\n", metrics.LatencyP50)
		fmt.Printf("  - P95 Latency: %v\n", metrics.LatencyP95)
		fmt.Printf("  - P99 Latency: %v\n", metrics.LatencyP99)
		fmt.Printf("  - Error Rate: %.2f%%\n", metrics.ErrorRate)

		if len(result.Errors) > 0 {
			fmt.Printf("  - Errors: %d\n", len(result.Errors))
		}

		if i < len(scenarios)-1 {
			pauseDuration := 5 * time.Second
			fmt.Printf("\nPausing %v before next scenario...\n", pauseDuration)
			time.Sleep(pauseDuration)
		}
	}

	return results
}

func runProgressiveBenchmark() []*benchmark.ScenarioResult {
	results := make([]*benchmark.ScenarioResult, 0)

	concurrencyLevels := []int{10, 50, 100, 200, 500}

	for _, conc := range concurrencyLevels {
		scenario := benchmark.BenchmarkScenario{
			Name:        fmt.Sprintf("Image Captcha Generate - Concurrency %d", conc),
			Description: fmt.Sprintf("Testing with %d concurrent workers", conc),
			Endpoint:    "/api/v1/captcha/image/generate",
			Method:      "POST",
			Body: map[string]interface{}{
				"app_id": 1,
				"length": 4,
				"width":  120,
				"height": 40,
			},
			Concurrency: conc,
			Duration:    30 * time.Second,
			AppID:       1,
		}

		fmt.Printf("\nRunning: %s\n", scenario.Name)
		fmt.Printf("Concurrency: %d | Duration: %v\n", conc, scenario.Duration)

		result := benchmark.RunScenario(scenario)
		results = append(results, result)

		metrics := result.Metrics
		fmt.Printf("Results:\n")
		fmt.Printf("  - Total Requests: %d\n", metrics.TotalRequests)
		fmt.Printf("  - QPS: %.2f\n", metrics.QPS)
		fmt.Printf("  - P99 Latency: %v\n", metrics.LatencyP99)

		time.Sleep(5 * time.Second)
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
	passedScenarios := 0
	totalScenarios := len(results)

	for _, result := range results {
		metrics := result.Metrics
		totalQPS += metrics.QPS
		totalRequests += metrics.TotalRequests
		totalErrors += metrics.FailedRequests

		status := "PASS"
		targetMet := metrics.QPS >= 10000 && metrics.LatencyP99 <= 50*time.Millisecond && metrics.ErrorRate <= 1.0

		if !targetMet {
			status = "WARN"
			if metrics.QPS < 5000 || metrics.ErrorRate > 5.0 {
				status = "FAIL"
			}
		} else {
			passedScenarios++
		}

		fmt.Printf("\n[%s] %s\n", status, metrics.Name)
		fmt.Printf("  QPS: %.2f (target: >10000)\n", metrics.QPS)
		fmt.Printf("  P99 Latency: %v (target: <50ms)\n", metrics.LatencyP99)
		fmt.Printf("  Error Rate: %.2f%% (target: <1%%)\n", metrics.ErrorRate)
		fmt.Printf("  Memory: %d bytes\n", metrics.MemoryUsage)
	}

	fmt.Println("\n" + strings.Repeat("-", 80))

	totalLatency := time.Duration(0)
	for _, r := range results {
		totalLatency += r.Metrics.LatencyP99
	}
	avgP99Latency := totalLatency / time.Duration(len(results))

	fmt.Printf("\nTotal Combined QPS: %.2f\n", totalQPS)
	fmt.Printf("Average P99 Latency: %v\n", avgP99Latency)
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Total Errors: %d\n", totalErrors)
	fmt.Printf("Scenarios Passed: %d/%d\n", passedScenarios, totalScenarios)

	if totalRequests > 0 {
		overallErrorRate := float64(totalErrors) / float64(totalRequests) * 100
		fmt.Printf("Overall Error Rate: %.2f%%\n", overallErrorRate)
	}

	report := benchmark.GenerateReport(collectMetrics(results))

	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("PERFORMANCE ANALYSIS")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Overall Score: %.2f%%\n", report.Analysis.OverallScore)
	fmt.Printf("QPS Score: %.2f%%\n", report.Analysis.QPSScore)
	fmt.Printf("Latency Score: %.2f%%\n", report.Analysis.LatencyScore)
	fmt.Printf("Error Rate Score: %.2f%%\n", report.Analysis.ErrorRateScore)

	if len(report.Analysis.Bottlenecks) > 0 {
		fmt.Println("\nBottlenecks Identified:")
		for _, bottleneck := range report.Analysis.Bottlenecks {
			fmt.Printf("  - %s\n", bottleneck)
		}
	}

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
	GeneratedAt     time.Time                      `json:"generated_at"`
	SystemInfo      benchmark.SystemInfo           `json:"system_info"`
	Results         []ScenarioResultData           `json:"results"`
	Summary         SummaryData                     `json:"summary"`
	Recommendations []string                       `json:"recommendations"`
	Analysis        benchmark.PerformanceAnalysis   `json:"analysis"`
}

type ScenarioResultData struct {
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests    int64   `json:"failed_requests"`
	QPS               float64 `json:"qps"`
	LatencyP50        string  `json:"latency_p50"`
	LatencyP95        string  `json:"latency_p95"`
	LatencyP99        string  `json:"latency_p99"`
	AvgLatency        string  `json:"avg_latency"`
	ErrorRate         float64 `json:"error_rate"`
	MemoryUsage       uint64  `json:"memory_usage"`
	Duration          string  `json:"duration"`
}

type SummaryData struct {
	TotalQPS          float64 `json:"total_qps"`
	TotalRequests     int64   `json:"total_requests"`
	TotalErrors       int64   `json:"total_errors"`
	OverallErrorRate  float64 `json:"overall_error_rate"`
	ScenariosPassed   int     `json:"scenarios_passed"`
	ScenariosFailed   int     `json:"scenarios_failed"`
	AverageP99Latency string  `json:"average_p99_latency"`
}

func generateReports(results []*benchmark.ScenarioResult) {
	reportData := ReportData{
		GeneratedAt: time.Now(),
		SystemInfo:  benchmark.GetSystemInfo(),
		Results:     make([]ScenarioResultData, 0, len(results)),
	}

	totalQPS := 0.0
	totalRequests := int64(0)
	totalErrors := int64(0)
	scenariosPassed := 0
	totalP99Latency := time.Duration(0)

	for _, result := range results {
		metrics := result.Metrics
		totalQPS += metrics.QPS
		totalRequests += metrics.TotalRequests
		totalErrors += metrics.FailedRequests
		totalP99Latency += metrics.LatencyP99

		targetMet := metrics.QPS >= 10000 && metrics.LatencyP99 <= 50*time.Millisecond && metrics.ErrorRate <= 1.0
		if targetMet {
			scenariosPassed++
		}

		reportData.Results = append(reportData.Results, ScenarioResultData{
			Name:               metrics.Name,
			Description:        result.Scenario.Description,
			TotalRequests:      metrics.TotalRequests,
			SuccessfulRequests: metrics.SuccessfulRequests,
			FailedRequests:    metrics.FailedRequests,
			QPS:               metrics.QPS,
			LatencyP50:        metrics.LatencyP50.String(),
			LatencyP95:        metrics.LatencyP95.String(),
			LatencyP99:        metrics.LatencyP99.String(),
			AvgLatency:        metrics.AvgLatency.String(),
			ErrorRate:         metrics.ErrorRate,
			MemoryUsage:       metrics.MemoryUsage,
			Duration:          metrics.Duration.String(),
		})
	}

	avgP99Latency := time.Duration(0)
	if len(results) > 0 {
		avgP99Latency = totalP99Latency / time.Duration(len(results))
	}

	reportData.Summary = SummaryData{
		TotalQPS:          totalQPS,
		TotalRequests:     totalRequests,
		TotalErrors:       totalErrors,
		OverallErrorRate:  0,
		ScenariosPassed:   scenariosPassed,
		ScenariosFailed:  len(results) - scenariosPassed,
		AverageP99Latency: avgP99Latency.String(),
	}

	if totalRequests > 0 {
		reportData.Summary.OverallErrorRate = float64(totalErrors) / float64(totalRequests) * 100
	}

	report := benchmark.GenerateReport(collectMetrics(results))
	reportData.Recommendations = report.Recommendations
	reportData.Analysis = report.Analysis

	switch outputFormat {
	case "json":
		generateJSONReport(reportData)
	case "html":
		generateHTMLReport(reportData)
	default:
		generateJSONReport(reportData)
	}
}

func generateJSONReport(reportData ReportData) {
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

func generateHTMLReport(reportData ReportData) {
	filename := fmt.Sprintf("benchmark_report_%s.html", time.Now().Format("20060102_150405"))

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Performance Benchmark Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #4CAF50; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        .metric { display: inline-block; background: #e8f5e9; padding: 15px 25px; margin: 10px; border-radius: 5px; }
        .metric-value { font-size: 2em; font-weight: bold; color: #2e7d32; }
        .metric-label { color: #666; font-size: 0.9em; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #4CAF50; color: white; }
        .pass { color: #4CAF50; font-weight: bold; }
        .warn { color: #ff9800; font-weight: bold; }
        .fail { color: #f44336; font-weight: bold; }
        .summary { background: #f5f5f5; padding: 20px; border-radius: 5px; margin: 20px 0; }
        .recommendation { background: #fff3e0; padding: 15px; margin: 10px 0; border-left: 4px solid #ff9800; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Performance Benchmark Report</h1>
        <p>Generated at: ` + reportData.GeneratedAt.Format(time.RFC3339) + `</p>

        <div class="summary">
            <h2>System Information</h2>
            <p>Go Version: ` + reportData.SystemInfo.GoVersion + `</p>
            <p>CPU Cores: ` + fmt.Sprintf("%d", reportData.SystemInfo.CPUCores) + `</p>
            <p>OS: ` + reportData.SystemInfo.OS + `/` + reportData.SystemInfo.Arch + `</p>
            <p>Goroutines: ` + fmt.Sprintf("%d", reportData.SystemInfo.NumGoroutine) + `</p>
        </div>

        <h2>Summary</h2>
        <div class="metric">
            <div class="metric-value">` + fmt.Sprintf("%.2f", reportData.Summary.TotalQPS) + `</div>
            <div class="metric-label">Total QPS</div>
        </div>
        <div class="metric">
            <div class="metric-value">` + fmt.Sprintf("%d", reportData.Summary.TotalRequests) + `</div>
            <div class="metric-label">Total Requests</div>
        </div>
        <div class="metric">
            <div class="metric-value">` + reportData.Summary.AverageP99Latency + `</div>
            <div class="metric-label">Avg P99 Latency</div>
        </div>
        <div class="metric">
            <div class="metric-value">` + fmt.Sprintf("%.2f%%", reportData.Summary.OverallErrorRate) + `</div>
            <div class="metric-label">Error Rate</div>
        </div>

        <h2>Results</h2>
        <table>
            <tr>
                <th>Scenario</th>
                <th>QPS</th>
                <th>P99 Latency</th>
                <th>Error Rate</th>
                <th>Status</th>
            </tr>`

	for _, result := range reportData.Results {
		status := "pass"
		statusText := "PASS"
		if result.QPS < 5000 || result.ErrorRate > 5.0 {
			status = "fail"
			statusText = "FAIL"
		} else if result.QPS < 10000 || result.ErrorRate > 1.0 {
			status = "warn"
			statusText = "WARN"
		}

		html += fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td>%.2f</td>
                <td>%s</td>
                <td>%.2f%%</td>
                <td class="%s">%s</td>
            </tr>`, result.Name, result.QPS, result.LatencyP99, result.ErrorRate, status, statusText)
	}

	html += `
        </table>

        <h2>Analysis</h2>
        <div class="summary">
            <p><strong>Overall Score:</strong> ` + fmt.Sprintf("%.2f%%", reportData.Analysis.OverallScore) + `</p>
            <p><strong>QPS Score:</strong> ` + fmt.Sprintf("%.2f%%", reportData.Analysis.QPSScore) + `</p>
            <p><strong>Latency Score:</strong> ` + fmt.Sprintf("%.2f%%", reportData.Analysis.LatencyScore) + `</p>
            <p><strong>Error Rate Score:</strong> ` + fmt.Sprintf("%.2f%%", reportData.Analysis.ErrorRateScore) + `</p>
        </div>`

	if len(reportData.Analysis.Bottlenecks) > 0 {
		html += `
        <h2>Bottlenecks</h2>
        <ul>`
		for _, bottleneck := range reportData.Analysis.Bottlenecks {
			html += `<li>` + bottleneck + `</li>`
		}
		html += `</ul>`
	}

	if len(reportData.Recommendations) > 0 {
		html += `
        <h2>Recommendations</h2>`
		for _, rec := range reportData.Recommendations {
			html += `<div class="recommendation">` + rec + `</div>`
		}
	}

	html += `
    </div>
</body>
</html>`

	if err := os.WriteFile(filename, []byte(html), 0644); err != nil {
		fmt.Printf("Error saving HTML report: %v\n", err)
		return
	}

	fmt.Printf("\nHTML report saved to: %s\n", filename)
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
