package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

type BenchmarkResult struct {
	Timestamp           time.Time           `json:"timestamp"`
	TargetURL           string              `json:"target_url"`
	Duration            string              `json:"duration"`
	Rate               int                 `json:"rate"`
	Requests           int                 `json:"requests"`
	Throughput         float64             `json:"throughput_requests_per_second"`
	Latency            LatencyStats        `json:"latency"`
	SuccessRate        float64             `json:"success_rate_percent"`
	StatusCodes        map[string]int      `json:"status_codes"`
	Errors             []string            `json:"errors"`
}

type LatencyStats struct {
	Min      float64 `json:"min_ms"`
	Mean     float64 `json:"mean_ms"`
	Max      float64 `json:"max_ms"`
	P50      float64 `json:"p50_ms"`
	P90      float64 `json:"p90_ms"`
	P95      float64 `json:"p95_ms"`
	P99      float64 `json:"p99_ms"`
}

type TestCase struct {
	Name        string
	URL         string
	Method      string
	Headers     map[string]string
	Body        string
	Duration    time.Duration
	Rate        int
}

func main() {
	log.Println("Starting HJTPX Performance Benchmark...")

	baseURL := "http://localhost:8080"
	if envURL := os.Getenv("TEST_URL"); envURL != "" {
		baseURL = envURL
	}

	testCases := []TestCase{
		{
			Name:     "Health Check",
			URL:      baseURL + "/",
			Method:   "GET",
			Duration: 30 * time.Second,
			Rate:     100,
		},
		{
			Name:     "Metrics",
			URL:      baseURL + "/metrics",
			Method:   "GET",
			Duration: 30 * time.Second,
			Rate:     50,
		},
		{
			Name:     "Slider Captcha",
			URL:      baseURL + "/api/v1/captcha/slider",
			Method:   "GET",
			Duration: 60 * time.Second,
			Rate:     50,
		},
		{
			Name:     "Click Captcha",
			URL:      baseURL + "/api/v1/captcha/click",
			Method:   "GET",
			Duration: 60 * time.Second,
			Rate:     50,
		},
	}

	results := make([]BenchmarkResult, 0)

	for _, tc := range testCases {
		log.Printf("Running benchmark: %s", tc.Name)
		result := runBenchmark(tc)
		results = append(results, result)
		printResult(result)
	}

	reportPath := generateReport(results)
	log.Printf("Benchmark report generated: %s", reportPath)

	log.Println("Benchmark completed!")
}

func runBenchmark(tc TestCase) BenchmarkResult {
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: tc.Method,
		URL:    tc.URL,
	})

	rate := vegeta.Rate{Freq: tc.Rate, Per: time.Second}
	attacker := vegeta.NewAttacker()

	var metrics vegeta.Metrics
	var errors []string

	for res := range attacker.Attack(targeter, rate, tc.Duration, tc.Name) {
		metrics.Add(res)
		if res.Error != "" {
			errors = append(errors, res.Error)
		}
	}

	metrics.Close()

	statusCodes := make(map[string]int)
	statusCodes["2xx"] = int(metrics.StatusCodes["200"]) + int(metrics.StatusCodes["201"]) + int(metrics.StatusCodes["204"])
	statusCodes["4xx"] = int(metrics.StatusCodes["400"]) + int(metrics.StatusCodes["401"]) + int(metrics.StatusCodes["403"]) + int(metrics.StatusCodes["404"])
	statusCodes["5xx"] = int(metrics.StatusCodes["500"]) + int(metrics.StatusCodes["502"]) + int(metrics.StatusCodes["503"]) + int(metrics.StatusCodes["504"])

	return BenchmarkResult{
		Timestamp:   time.Now(),
		TargetURL:   tc.URL,
		Duration:    tc.Duration.String(),
		Rate:        tc.Rate,
		Requests:    int(metrics.Requests),
		Throughput:  metrics.Throughput,
		Latency: LatencyStats{
			Min:  metrics.Latencies.Min.Seconds() * 1000,
			Mean: metrics.Latencies.Mean.Seconds() * 1000,
			Max:  metrics.Latencies.Max.Seconds() * 1000,
			P50:  metrics.Latencies.P50.Seconds() * 1000,
			P90:  metrics.Latencies.P90.Seconds() * 1000,
			P95:  metrics.Latencies.P95.Seconds() * 1000,
			P99:  metrics.Latencies.P99.Seconds() * 1000,
		},
		SuccessRate: metrics.Success * 100,
		StatusCodes: statusCodes,
		Errors:      errors,
	}
}

func printResult(result BenchmarkResult) {
	fmt.Printf("\n=== %s ===\n", result.TargetURL)
	fmt.Printf("Requests: %d\n", result.Requests)
	fmt.Printf("Duration: %s\n", result.Duration)
	fmt.Printf("Throughput: %.2f requests/sec\n", result.Throughput)
	fmt.Printf("Success Rate: %.2f%%\n", result.SuccessRate)
	fmt.Printf("Latency (ms): Min=%.2f, Mean=%.2f, Max=%.2f\n",
		result.Latency.Min, result.Latency.Mean, result.Latency.Max)
	fmt.Printf("Percentiles (ms): P50=%.2f, P90=%.2f, P95=%.2f, P99=%.2f\n",
		result.Latency.P50, result.Latency.P90, result.Latency.P95, result.Latency.P99)
	fmt.Printf("Status Codes: 2xx=%d, 4xx=%d, 5xx=%d\n",
		result.StatusCodes["2xx"], result.StatusCodes["4xx"], result.StatusCodes["5xx"])
	if len(result.Errors) > 0 {
		fmt.Printf("Errors: %d\n", len(result.Errors))
	}
}

func generateReport(results []BenchmarkResult) string {
	timestamp := time.Now().Format("20060102_150405")
	reportPath := fmt.Sprintf("benchmark_report_%s.json", timestamp)

	report := map[string]interface{}{
		"report_version": "1.0",
		"generated_at":   time.Now().Format(time.RFC3339),
		"total_tests":    len(results),
		"results":        results,
		"summary":        generateSummary(results),
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Printf("Failed to generate report: %v", err)
		return ""
	}

	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		log.Printf("Failed to write report: %v", err)
		return ""
	}

	htmlReportPath := generateHTMLReport(report, timestamp)
	return htmlReportPath
}

func generateSummary(results []BenchmarkResult) map[string]interface{} {
	totalRequests := 0
	totalSuccess := 0.0
	sumLatency := 0.0
	sumThroughput := 0.0

	for _, r := range results {
		totalRequests += r.Requests
		totalSuccess += r.SuccessRate
		sumLatency += r.Latency.Mean
		sumThroughput += r.Throughput
	}

	count := float64(len(results))

	return map[string]interface{}{
		"total_requests":       totalRequests,
		"average_success_rate": totalSuccess / count,
		"average_latency_ms":   sumLatency / count,
		"average_throughput":   sumThroughput / count,
		"passed_tests":         countPassed(results),
		"failed_tests":         countFailed(results),
	}
}

func countPassed(results []BenchmarkResult) int {
	count := 0
	for _, r := range results {
		if r.SuccessRate >= 95 && r.Latency.Mean < 500 {
			count++
		}
	}
	return count
}

func countFailed(results []BenchmarkResult) int {
	return len(results) - countPassed(results)
}

func generateHTMLReport(report map[string]interface{}, timestamp string) string {
	htmlPath := fmt.Sprintf("benchmark_report_%s.html", timestamp)

	summary := report["summary"].(map[string]interface{})
	results := report["results"].([]BenchmarkResult)

	failedTests := summary["failed_tests"].(int)
	failedClass := ""
	if failedTests > 0 {
		failedClass = "danger"
	}

	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>HJTPX Performance Benchmark Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 2px solid #4CAF50; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        .summary { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; margin: 20px 0; }
        .summary-item { background: #f9f9f9; padding: 20px; border-radius: 8px; text-align: center; }
        .summary-item .label { color: #888; font-size: 14px; }
        .summary-item .value { font-size: 28px; font-weight: bold; color: #4CAF50; }
        .summary-item.warning .value { color: #ff9800; }
        .summary-item.danger .value { color: #f44336; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f5f5f5; font-weight: bold; }
        tr:hover { background: #f9f9f9; }
        .status-pass { color: #4CAF50; }
        .status-fail { color: #f44336; }
        .metrics { margin-top: 20px; }
        .metric-row { display: flex; gap: 20px; margin-bottom: 10px; }
        .metric { flex: 1; background: #f9f9f9; padding: 10px; border-radius: 4px; }
        .metric label { color: #888; font-size: 12px; }
        .metric value { font-weight: bold; }
        .error-list { color: #f44336; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>HJTPX Performance Benchmark Report</h1>
        <p>Generated at: ` + report["generated_at"].(string) + `</p>
        <p>Total Tests: ` + strconv.Itoa(report["total_tests"].(int)) + `</p>
        <div class="summary">
            <div class="summary-item">
                <div class="label">Total Requests</div>
                <div class="value">` + strconv.Itoa(summary["total_requests"].(int)) + `</div>
            </div>
            <div class="summary-item">
                <div class="label">Avg Success Rate</div>
                <div class="value">` + fmt.Sprintf("%.2f%%", summary["average_success_rate"].(float64)) + `</div>
            </div>
            <div class="summary-item">
                <div class="label">Avg Latency</div>
                <div class="value">` + fmt.Sprintf("%.2fms", summary["average_latency_ms"].(float64)) + `</div>
            </div>
            <div class="summary-item">
                <div class="label">Avg Throughput</div>
                <div class="value">` + fmt.Sprintf("%.2f/s", summary["average_throughput"].(float64)) + `</div>
            </div>
            <div class="summary-item">
                <div class="label">Passed Tests</div>
                <div class="value">` + strconv.Itoa(summary["passed_tests"].(int)) + `</div>
            </div>
            <div class="summary-item">
                <div class="label">Failed Tests</div>
                <div class="value ` + failedClass + `">` + strconv.Itoa(failedTests) + `</div>
            </div>
        </div>`

	for _, result := range results {
		statusClass := "pass"
		if result.SuccessRate < 95 {
			statusClass = "fail"
		}

		html += `
        <h2>` + result.TargetURL + `</h2>
        <table>
            <tr><th>Metric</th><th>Value</th></tr>
            <tr><td>Duration</td><td>` + result.Duration + `</td></tr>
            <tr><td>Rate</td><td>` + strconv.Itoa(result.Rate) + ` req/s</td></tr>
            <tr><td>Requests</td><td>` + strconv.Itoa(result.Requests) + `</td></tr>
            <tr><td>Throughput</td><td>` + fmt.Sprintf("%.2f req/s", result.Throughput) + `</td></tr>
            <tr><td>Success Rate</td><td><span class="status-` + statusClass + `">` + fmt.Sprintf("%.2f%%", result.SuccessRate) + `</span></td></tr>
            <tr><td>2xx Responses</td><td>` + strconv.Itoa(result.StatusCodes["2xx"]) + `</td></tr>
            <tr><td>4xx Responses</td><td>` + strconv.Itoa(result.StatusCodes["4xx"]) + `</td></tr>
            <tr><td>5xx Responses</td><td>` + strconv.Itoa(result.StatusCodes["5xx"]) + `</td></tr>
        </table>
        <div class="metrics">
            <div class="metric-row">
                <div class="metric"><label>Min Latency</label><br><value>` + fmt.Sprintf("%.2fms", result.Latency.Min) + `</value></div>
                <div class="metric"><label>Mean Latency</label><br><value>` + fmt.Sprintf("%.2fms", result.Latency.Mean) + `</value></div>
                <div class="metric"><label>Max Latency</label><br><value>` + fmt.Sprintf("%.2fms", result.Latency.Max) + `</value></div>
            </div>
            <div class="metric-row">
                <div class="metric"><label>P50</label><br><value>` + fmt.Sprintf("%.2fms", result.Latency.P50) + `</value></div>
                <div class="metric"><label>P90</label><br><value>` + fmt.Sprintf("%.2fms", result.Latency.P90) + `</value></div>
                <div class="metric"><label>P95</label><br><value>` + fmt.Sprintf("%.2fms", result.Latency.P95) + `</value></div>
                <div class="metric"><label>P99</label><br><value>` + fmt.Sprintf("%.2fms", result.Latency.P99) + `</value></div>
            </div>
        </div>`

		if len(result.Errors) > 0 {
			html += `<div class="error-list">Errors: ` + strconv.Itoa(len(result.Errors)) + `</div>`
		}
	}

	html += `
    </div>
</body>
</html>`

	if err := os.WriteFile(htmlPath, []byte(html), 0644); err != nil {
		log.Printf("Failed to write HTML report: %v", err)
		return ""
	}

	return htmlPath
}
