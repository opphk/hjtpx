package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/benchmark"
)

var (
	baseURL        = flag.String("url", "http://localhost:8080", "API base URL")
	duration       = flag.Int("duration", 60, "Benchmark duration in seconds")
	concurrency    = flag.Int("concurrency", 100, "Number of concurrent requests")
	scenarioFilter = flag.String("scenario", "", "Filter scenarios by name")
	reportFormat   = flag.String("format", "both", "Report format: json, html, both")
	reportDir      = flag.String("output", "./reports", "Output directory for reports")
	baselineMode   = flag.Bool("baseline", false, "Save results as baseline")
	compareMode    = flag.Bool("compare", false, "Compare with existing baseline")
	noColor        = flag.Bool("no-color", false, "Disable color output")
	quiet          = flag.Bool("quiet", false, "Suppress detailed output")
	help           = flag.Bool("help", false, "Show help")
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

type ReportData struct {
	GeneratedAt     time.Time                    `json:"generated_at"`
	SystemInfo     benchmark.SystemInfo          `json:"system_info"`
	Results        []ScenarioResultData          `json:"results"`
	Summary        SummaryData                   `json:"summary"`
	Recommendations []string                      `json:"recommendations"`
	Regressions    []*benchmark.RegressionResult  `json:"regressions,omitempty"`
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
	MinLatency         string  `json:"min_latency"`
	MaxLatency         string  `json:"max_latency"`
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

func main() {
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	printHeader()

	if err := os.MkdirAll(*reportDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	baselineStore, err := benchmark.NewBaselineStore("./baselines")
	if err != nil {
		fmt.Printf("Warning: Could not initialize baseline store: %v\n", err)
	}

	var regressionDetector *benchmark.RegressionDetector
	if baselineStore != nil {
		regressionDetector = benchmark.NewRegressionDetector(baselineStore, benchmark.RegressionConfig{
			QPSThreshold:        0.8,
			P99LatencyThresholdMs: 500,
			ErrorRateThreshold: 5.0,
			MemoryGrowthLimit:  0.2,
		})
	}

	if !checkServiceHealth() {
		colorPrint(colorYellow, "Warning: Benchmark service may not be running at %s", *baseURL)
		if os.Getenv("CI") != "true" {
			colorPrint(colorYellow, "Press Ctrl+C to cancel or wait to continue...")
			time.Sleep(5 * time.Second)
		}
	}

	scenarios := filterScenarios()
	results := runBenchmarks(scenarios, regressionDetector)

	printSummary(results, regressionDetector)

	if regressionDetector != nil && len(results) > 0 && *compareMode {
		regressions := regressionDetector.DetectAll(results)
		printRegressionAnalysis(regressions)
	}

	generateReports(results, regressionDetector)

	if *baselineMode {
		saveBaseline(results, baselineStore)
	}

	printFooter()

	hasFailure := checkResults(results)
	if hasFailure {
		os.Exit(1)
	}
}

func filterScenarios() []benchmark.BenchmarkScenario {
	var filtered []benchmark.BenchmarkScenario

	for _, scenario := range benchmark.Scenarios {
		if *scenarioFilter != "" {
			if !strings.Contains(strings.ToLower(scenario.Name), strings.ToLower(*scenarioFilter)) {
				continue
			}
		}

		scenario.Concurrency = *concurrency
		scenario.Duration = time.Duration(*duration) * time.Second

		filtered = append(filtered, scenario)
	}

	if len(filtered) == 0 {
		colorPrint(colorYellow, "No scenarios match the filter '%s', using all scenarios", *scenarioFilter)
		for i := range benchmark.Scenarios {
			benchmark.Scenarios[i].Concurrency = *concurrency
			benchmark.Scenarios[i].Duration = time.Duration(*duration) * time.Second
		}
		return benchmark.Scenarios
	}

	return filtered
}

func runBenchmarks(scenarios []benchmark.BenchmarkScenario, detector *benchmark.RegressionDetector) []*benchmark.ScenarioResult {
	results := make([]*benchmark.ScenarioResult, 0, len(scenarios))

	for i, scenario := range scenarios {
		if !*quiet {
			fmt.Printf("\n[%d/%d] Running: %s\n", i+1, len(scenarios), scenario.Name)
			fmt.Printf("    Description: %s\n", scenario.Description)
			fmt.Printf("    Concurrency: %d | Duration: %v\n", scenario.Concurrency, scenario.Duration)
		}

		result := benchmark.RunScenario(scenario)
		results = append(results, result)

		if !*quiet {
			metrics := result.Metrics
			color := colorGreen
			if metrics.QPS < 1000 || metrics.ErrorRate > 5.0 {
				color = colorRed
			} else if metrics.QPS < 5000 || metrics.ErrorRate > 1.0 {
				color = colorYellow
			}

			fmt.Printf("\n    ")
			colorPrint(color, "Results:")
			fmt.Printf("      Total Requests: %d\n", metrics.TotalRequests)
			fmt.Printf("      QPS: %.2f\n", metrics.QPS)
			fmt.Printf("      P50 Latency: %v\n", metrics.LatencyP50)
			fmt.Printf("      P95 Latency: %v\n", metrics.LatencyP95)
			fmt.Printf("      P99 Latency: %v\n", metrics.LatencyP99)
			fmt.Printf("      Error Rate: %.2f%%\n", metrics.ErrorRate)

			if len(result.Errors) > 0 {
				colorPrint(colorRed, "      Errors: %d", len(result.Errors))
			}
		}

		if i < len(scenarios)-1 && !*quiet {
			fmt.Printf("\n    Pausing 5s before next scenario...\n")
			time.Sleep(5 * time.Second)
		}
	}

	return results
}

func printSummary(results []*benchmark.ScenarioResult, detector *benchmark.RegressionDetector) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	colorPrint(colorCyan, "BENCHMARK SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	totalQPS := 0.0
	totalRequests := int64(0)
	totalErrors := int64(0)
	scenariosPassed := 0

	for _, result := range results {
		metrics := result.Metrics
		totalQPS += metrics.QPS
		totalRequests += metrics.TotalRequests
		totalErrors += metrics.FailedRequests

		status := "PASS"
		statusColor := colorGreen
		if metrics.QPS < 1000 || metrics.LatencyP99 > 200*time.Millisecond || metrics.ErrorRate > 5.0 {
			status = "FAIL"
			statusColor = colorRed
		} else if metrics.QPS < 5000 || metrics.LatencyP99 > 100*time.Millisecond || metrics.ErrorRate > 1.0 {
			status = "WARN"
			statusColor = colorYellow
		}

		if status == "PASS" {
			scenariosPassed++
		}

		fmt.Printf("\n[%s%s%s] %s\n", statusColor, status, colorReset, metrics.Name)
		fmt.Printf("  QPS: %.2f (target: >5000)\n", metrics.QPS)
		fmt.Printf("  P99 Latency: %v (target: <100ms)\n", metrics.LatencyP99)
		fmt.Printf("  Error Rate: %.2f%% (target: <1%%)\n", metrics.ErrorRate)
		fmt.Printf("  Memory: %.2f MB\n", float64(metrics.MemoryUsage)/1024/1024)
	}

	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Printf("\nTotal Combined QPS: %.2f\n", totalQPS)
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Total Errors: %d\n", totalErrors)

	if totalRequests > 0 {
		overallErrorRate := float64(totalErrors) / float64(totalRequests) * 100
		fmt.Printf("Overall Error Rate: %.2f%%\n", overallErrorRate)
	}

	fmt.Printf("\nScenarios: %d passed, %d failed\n", scenariosPassed, len(results)-scenariosPassed)

	report := benchmark.GenerateReport(collectMetrics(results))
	if len(report.Recommendations) > 0 {
		fmt.Println("\n" + strings.Repeat("-", 80))
		colorPrint(colorYellow, "RECOMMENDATIONS")
		fmt.Println(strings.Repeat("-", 80))
		for _, rec := range report.Recommendations {
			fmt.Printf("- %s\n", rec)
		}
	}
}

func printRegressionAnalysis(regressions []*benchmark.RegressionResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	colorPrint(colorCyan, "REGRESSION ANALYSIS")
	fmt.Println(strings.Repeat("=", 80))

	regressionCount := 0
	for _, reg := range regressions {
		if reg.HasRegression {
			regressionCount++
		}

		status := "OK"
		statusColor := colorGreen
		if reg.HasRegression {
			status = "REGRESSION"
			statusColor = colorRed
		}

		fmt.Printf("\n[%s%s%s] %s\n", statusColor, status, colorReset, reg.ScenarioName)

		if reg.BaselineQPS > 0 {
			fmt.Printf("  QPS Change: %.2f (%.2f%%)\n", reg.QPSChange, reg.QPSChangePercent)
		}
		fmt.Printf("  P99 Change: %dms\n", reg.P99Change)
		fmt.Printf("  Error Rate Change: %.2f%%\n", reg.ErrorRateChange)

		if len(reg.Recommendations) > 0 {
			fmt.Println("  Recommendations:")
			for _, rec := range reg.Recommendations {
				fmt.Printf("    - %s\n", rec)
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("-", 80))
	if regressionCount > 0 {
		colorPrint(colorRed, "WARNING: %d regression(s) detected!", regressionCount)
	} else {
		colorPrint(colorGreen, "No regressions detected")
	}
}

func generateReports(results []*benchmark.ScenarioResult, detector *benchmark.RegressionDetector) {
	var regressions []*benchmark.RegressionResult
	if detector != nil {
		regressions = detector.DetectAll(results)
	}

	report := buildReport(results, regressions)

	if *reportFormat == "json" || *reportFormat == "both" {
		generateJSONReport(report, results, regressions)
	}

	if *reportFormat == "html" || *reportFormat == "both" {
		generateHTMLReport(results, regressions)
	}
}

func buildReport(results []*benchmark.ScenarioResult, regressions []*benchmark.RegressionResult) *ReportData {
	report := &ReportData{
		GeneratedAt: time.Now(),
		SystemInfo:  benchmark.GetSystemInfo(),
		Results:     make([]ScenarioResultData, 0, len(results)),
		Regressions: regressions,
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

		report.Results = append(report.Results, ScenarioResultData{
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
			MinLatency:         metrics.MinLatency.String(),
			MaxLatency:         metrics.MaxLatency.String(),
			ErrorRate:          metrics.ErrorRate,
			MemoryUsage:        metrics.MemoryUsage,
			Duration:           metrics.Duration.String(),
		})
	}

	report.Summary = SummaryData{
		TotalQPS:         totalQPS,
		TotalRequests:    totalRequests,
		TotalErrors:      totalErrors,
		OverallErrorRate: 0,
	}

	if totalRequests > 0 {
		report.Summary.OverallErrorRate = float64(totalErrors) / float64(totalRequests) * 100
	}
	report.Summary.ScenariosPassed = scenariosPassed
	report.Summary.ScenariosFailed = len(results) - scenariosPassed

	metrics := collectMetrics(results)
	benchReport := benchmark.GenerateReport(metrics)
	report.Recommendations = benchReport.Recommendations

	return report
}

func generateJSONReport(report *ReportData, results []*benchmark.ScenarioResult, regressions []*benchmark.RegressionResult) {
	filename := filepath.Join(*reportDir, fmt.Sprintf("benchmark_report_%s.json", time.Now().Format("20060102_150405")))

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		colorPrint(colorRed, "Error generating JSON report: %v", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		colorPrint(colorRed, "Error saving JSON report: %v", err)
		return
	}

	colorPrint(colorGreen, "JSON report saved to: %s", filename)
}

func generateHTMLReport(results []*benchmark.ScenarioResult, regressions []*benchmark.RegressionResult) {
	filename := filepath.Join(*reportDir, fmt.Sprintf("benchmark_report_%s.html", time.Now().Format("20060102_150405")))

	totalQPS := 0.0
	totalRequests := int64(0)
	for _, r := range results {
		totalQPS += r.Metrics.QPS
		totalRequests += r.Metrics.TotalRequests
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Performance Benchmark Report</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1400px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 20px; padding: 20px; background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h2 { color: #555; margin: 20px 0 10px; padding: 15px; background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin-bottom: 30px; }
        .metric-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .metric-card h3 { font-size: 14px; color: #888; margin-bottom: 5px; }
        .metric-card .value { font-size: 24px; font-weight: bold; color: #333; }
        .metric-card .value.pass { color: #28a745; }
        .metric-card .value.warn { color: #ffc107; }
        .metric-card .value.fail { color: #dc3545; }
        table { width: 100%; border-collapse: collapse; background: white; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 30px; }
        th { background: #4a90d9; color: white; padding: 12px; text-align: left; }
        td { padding: 12px; border-bottom: 1px solid #eee; }
        tr:last-child td { border-bottom: none; }
        tr:hover { background: #f9f9f9; }
        .status { padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; }
        .status.pass { background: #d4edda; color: #155724; }
        .status.fail { background: #f8d7da; color: #721c24; }
        .status.warn { background: #fff3cd; color: #856404; }
        .regression-section { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 30px; }
        .regression-item { padding: 15px; border-left: 4px solid; margin-bottom: 10px; border-radius: 4px; }
        .regression-item.has-regression { border-color: #dc3545; background: #fdf2f2; }
        .regression-item.no-regression { border-color: #28a745; background: #f0fdf4; }
        .timestamp { color: #888; font-size: 12px; }
        pre { background: #f8f8f8; padding: 15px; border-radius: 4px; overflow-x: auto; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Performance Benchmark Report</h1>
        <p class="timestamp">Generated at: ` + time.Now().Format(time.RFC3339) + `</p>
`

	html += fmt.Sprintf(`
        <h2>Summary</h2>
        <div class="summary">
            <div class="metric-card">
                <h3>Total Scenarios</h3>
                <div class="value">%d</div>
            </div>
            <div class="metric-card">
                <h3>Total Requests</h3>
                <div class="value">%d</div>
            </div>
            <div class="metric-card">
                <h3>Combined QPS</h3>
                <div class="value %.2f">%.2f</div>
            </div>
            <div class="metric-card">
                <h3>Scenarios Passed</h3>
                <div class="value pass">%d/%d</div>
            </div>
        </div>
`, len(results), totalRequests, totalQPS, totalQPS, countPassed(results), len(results))

	html += `
        <h2>Scenario Results</h2>
        <table>
            <thead>
                <tr>
                    <th>Scenario</th>
                    <th>Status</th>
                    <th>QPS</th>
                    <th>P50</th>
                    <th>P95</th>
                    <th>P99</th>
                    <th>Error Rate</th>
                    <th>Requests</th>
                </tr>
            </thead>
            <tbody>
`

	for _, result := range results {
		metrics := result.Metrics
		status := getStatus(metrics)
		html += fmt.Sprintf(`                <tr>
                    <td>%s</td>
                    <td><span class="status %s">%s</span></td>
                    <td>%.2f</td>
                    <td>%s</td>
                    <td>%s</td>
                    <td>%s</td>
                    <td>%.2f%%</td>
                    <td>%d</td>
                </tr>
`, metrics.Name, status, strings.ToUpper(status), metrics.QPS,
			metrics.LatencyP50, metrics.LatencyP95, metrics.LatencyP99,
			metrics.ErrorRate, metrics.TotalRequests)
	}

	html += `            </tbody>
        </table>
`

	if len(regressions) > 0 {
		html += `
        <h2>Regression Analysis</h2>
        <div class="regression-section">
`

		for _, reg := range regressions {
			class := "no-regression"
			if reg.HasRegression {
				class = "has-regression"
			}
			html += fmt.Sprintf(`            <div class="regression-item %s">
                <h3>%s</h3>
                <p>QPS: %.2f (baseline: %.2f, change: %.2f%%)</p>
                <p>P99: %dms (baseline: %dms)</p>
                <p>Error Rate: %.2f%% (baseline: %.2f%%)</p>
            </div>
`, class, reg.ScenarioName, reg.CurrentQPS, reg.BaselineQPS, reg.QPSChangePercent,
				reg.CurrentP99Ms, reg.BaselineP99Ms, reg.CurrentErrorRate, reg.BaselineErrorRate)
		}

		html += `        </div>
`
	}

	html += `    </div>
</body>
</html>
`

	if err := os.WriteFile(filename, []byte(html), 0644); err != nil {
		colorPrint(colorRed, "Error saving HTML report: %v", err)
		return
	}

	colorPrint(colorGreen, "HTML report saved to: %s", filename)
}

func saveBaseline(results []*benchmark.ScenarioResult, store *benchmark.BaselineStore) {
	if store == nil {
		colorPrint(colorYellow, "Baseline store not available, skipping baseline save")
		return
	}

	for _, result := range results {
		store.AddBaseline(result.Metrics.Name, result.Metrics)
	}

	colorPrint(colorGreen, "Baseline saved successfully")
}

func checkServiceHealth() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(*baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func checkResults(results []*benchmark.ScenarioResult) bool {
	failed := false
	for _, result := range results {
		if result.Metrics.QPS < 1000 || result.Metrics.ErrorRate > 5.0 {
			failed = true
			break
		}
	}
	return failed
}

func collectMetrics(results []*benchmark.ScenarioResult) []*benchmark.PerformanceMetrics {
	metrics := make([]*benchmark.PerformanceMetrics, 0, len(results))
	for _, r := range results {
		metrics = append(metrics, r.Metrics)
	}
	return metrics
}

func countPassed(results []*benchmark.ScenarioResult) int {
	count := 0
	for _, r := range results {
		if r.Metrics.QPS >= 5000 && r.Metrics.LatencyP99 <= 100*time.Millisecond && r.Metrics.ErrorRate <= 1.0 {
			count++
		}
	}
	return count
}

func getStatus(metrics *benchmark.PerformanceMetrics) string {
	if metrics.QPS < 1000 || metrics.LatencyP99 > 200*time.Millisecond || metrics.ErrorRate > 5.0 {
		return "fail"
	}
	if metrics.QPS < 5000 || metrics.LatencyP99 > 100*time.Millisecond || metrics.ErrorRate > 1.0 {
		return "warn"
	}
	return "pass"
}

func printHeader() {
	if !*quiet {
		fmt.Println(strings.Repeat("=", 80))
		colorPrint(colorCyan, "PERFORMANCE BENCHMARK SUITE v2.0")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Printf("Started at: %s\n", time.Now().Format(time.RFC3339))
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())
		fmt.Printf("Base URL: %s\n", *baseURL)
		fmt.Printf("Duration: %ds | Concurrency: %d\n", *duration, *concurrency)
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println()
	}
}

func printFooter() {
	if !*quiet {
		fmt.Println("\n" + strings.Repeat("=", 80))
		colorPrint(colorCyan, "Benchmark completed at: %s", time.Now().Format(time.RFC3339))
		fmt.Println(strings.Repeat("=", 80))
	}
}

func colorPrint(color, format string, args ...interface{}) {
	if *noColor {
		fmt.Printf(format+"\n", args...)
		return
	}
	fmt.Printf(color+format+colorReset+"\n", args...)
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
