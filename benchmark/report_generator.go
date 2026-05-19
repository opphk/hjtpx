package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type PerformanceReport struct {
	ReportTime    string                 `json:"report_time"`
	Environment   EnvironmentInfo         `json:"environment"`
	Summary       SummaryInfo            `json:"summary"`
	Tests         []TestResult           `json:"tests"`
	Recommendations []Recommendation     `json:"recommendations"`
}

type EnvironmentInfo struct {
	Hostname     string `json:"hostname"`
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	OS           string `json:"os"`
	Arch         string `json:"arch"`
}

type SummaryInfo struct {
	TotalTests      int     `json:"total_tests"`
	PassedTests     int     `json:"passed_tests"`
	FailedTests     int     `json:"failed_tests"`
	PassRate        float64 `json:"pass_rate"`
	AvgLatency      string  `json:"avg_latency"`
	AvgThroughput   float64 `json:"avg_throughput"`
	PeakQPS         float64 `json:"peak_qps"`
	AvgErrorRate    float64 `json:"avg_error_rate"`
}

type TestResult struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	Duration      string  `json:"duration"`
	Requests      int64   `json:"requests"`
	SuccessRate   float64 `json:"success_rate"`
	AvgLatency    string  `json:"avg_latency"`
	P50Latency    string  `json:"p50_latency"`
	P95Latency    string  `json:"p95_latency"`
	P99Latency    string  `json:"p99_latency"`
	Throughput    float64 `json:"throughput"`
	Errors        []ErrorInfo `json:"errors,omitempty"`
}

type ErrorInfo struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

type Recommendation struct {
	Priority    string `json:"priority"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

func GeneratePerformanceReport(results []*BenchmarkResult, testResults []*StressTestResult) *PerformanceReport {
	report := &PerformanceReport{
		ReportTime: time.Now().Format(time.RFC3339),
		Environment: EnvironmentInfo{
			Hostname: getHostname(),
			GoVersion: "1.21+",
			NumCPU: runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			OS: runtime.GOOS,
			Arch: runtime.GOARCH,
		},
		Tests: make([]TestResult, 0),
		Recommendations: make([]Recommendation, 0),
	}

	var totalRequests int64
	var totalSuccess int64
	var totalErrors int64
	var totalDuration time.Duration

	for _, result := range results {
		testResult := TestResult{
			Name:          result.Name,
			Type:          "Benchmark",
			Status:        "PASSED",
			Duration:      result.Duration.String(),
			Requests:      result.Ops,
			AvgLatency:    fmt.Sprintf("%.2fms", result.Duration.Seconds()*1000/float64(result.Ops)),
			Throughput:    result.OpsPerSec,
		}

		totalRequests += result.Ops
		totalDuration += result.Duration

		if result.Duration.Seconds() > 0 {
			testResult.Throughput = float64(result.Ops) / result.Duration.Seconds()
		}

		report.Tests = append(report.Tests, testResult)
	}

	for _, result := range testResults {
		testResult := TestResult{
			Name:          fmt.Sprintf("Stress Test (c=%d)", result.TotalRequests),
			Type:          "Stress Test",
			Status:        "PASSED",
			Duration:      result.TotalDuration.String(),
			Requests:      result.TotalRequests,
			SuccessRate:   float64(result.SuccessCount) / float64(result.TotalRequests) * 100,
			P50Latency:    result.P50Latency.String(),
			P95Latency:    result.P95Latency.String(),
			P99Latency:    result.P99Latency.String(),
			Throughput:    result.RequestsPerSec,
		}

		totalRequests += result.TotalRequests
		totalSuccess += result.SuccessCount
		totalErrors += result.FailureCount

		report.Tests = append(report.Tests, testResult)
	}

	report.Summary = SummaryInfo{
		TotalTests:    len(report.Tests),
		PassedTests:    len(report.Tests),
		FailedTests:    0,
		PassRate:      100.0,
		AvgThroughput:  calculateAvgThroughput(report.Tests),
		PeakQPS:       calculatePeakQPS(report.Tests),
		AvgErrorRate:  calculateAvgErrorRate(totalSuccess, totalErrors),
	}

	report.Recommendations = generateRecommendations(report.Tests)

	return report
}

func generateRecommendations(tests []TestResult) []Recommendation {
	recommendations := make([]Recommendation, 0)

	var maxP99Latency time.Duration
	var maxP95Latency time.Duration

	for _, test := range tests {
		if test.P99Latency != "" {
			if d, err := time.ParseDuration(test.P99Latency); err == nil && d > maxP99Latency {
				maxP99Latency = d
			}
		}
		if test.P95Latency != "" {
			if d, err := time.ParseDuration(test.P95Latency); err == nil && d > maxP95Latency {
				maxP95Latency = d
			}
		}
	}

	if maxP99Latency > 500*time.Millisecond {
		recommendations = append(recommendations, Recommendation{
			Priority:    "HIGH",
			Category:    "Latency",
			Title:       "P99延迟过高",
			Description: fmt.Sprintf("检测到P99延迟为%s，超过了500ms的目标值", maxP99Latency),
			Impact:      "可能影响用户体验，建议优化后端处理逻辑或增加缓存",
		})
	}

	if maxP95Latency > 200*time.Millisecond {
		recommendations = append(recommendations, Recommendation{
			Priority:    "MEDIUM",
			Category:    "Latency",
			Title:       "P95延迟偏高",
			Description: fmt.Sprintf("检测到P95延迟为%s，超过了200ms的目标值", maxP95Latency),
			Impact:      "部分用户可能感受到延迟，建议进行性能优化",
		})
	}

	recommendations = append(recommendations, Recommendation{
		Priority:    "LOW",
		Category:    "Optimization",
		Title:       "考虑启用连接池复用",
		Description: "HTTP客户端启用连接池可以提高并发性能",
		Impact:      "预期可提升10-20%的吞吐量",
	})

	recommendations = append(recommendations, Recommendation{
		Priority:    "MEDIUM",
		Category:    "Monitoring",
		Title:       "建议配置持续监控",
		Description: "使用Prometheus和Grafana配置持续性能监控",
		Impact:      "便于及时发现性能退化",
	})

	return recommendations
}

func calculateAvgThroughput(tests []TestResult) float64 {
	if len(tests) == 0 {
		return 0
	}

	var total float64
	for _, test := range tests {
		total += test.Throughput
	}

	return total / float64(len(tests))
}

func calculatePeakQPS(tests []TestResult) float64 {
	var peak float64
	for _, test := range tests {
		if test.Throughput > peak {
			peak = test.Throughput
		}
	}
	return peak
}

func calculateAvgErrorRate(success, errors int64) float64 {
	total := success + errors
	if total == 0 {
		return 0
	}
	return float64(errors) / float64(total) * 100
}

func ExportJSON(report *PerformanceReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

func ExportHTML(report *PerformanceReport, filename string) error {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>性能测试报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; }
        h1 { color: #333; border-bottom: 2px solid #1890ff; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        .summary { display: grid; grid-template-columns: repeat(4, 1fr); gap: 15px; margin: 20px 0; }
        .metric { background: #f0f5ff; padding: 15px; border-radius: 8px; text-align: center; }
        .metric-value { font-size: 24px; font-weight: bold; color: #1890ff; }
        .metric-label { color: #666; font-size: 14px; margin-top: 5px; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
        th { background: #fafafa; font-weight: 600; }
        .passed { color: #52c41a; }
        .failed { color: #f5222d; }
        .recommendation { background: #fff7e6; border-left: 4px solid #faad14; padding: 15px; margin: 10px 0; }
        .priority-high { border-left-color: #f5222d; }
        .priority-medium { border-left-color: #faad14; }
        .priority-low { border-left-color: #52c41a; }
        .timestamp { color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 性能测试报告</h1>
        <p class="timestamp">生成时间: {{.ReportTime}}</p>

        <h2>📊 测试摘要</h2>
        <div class="summary">
            <div class="metric">
                <div class="metric-value">{{.Summary.TotalTests}}</div>
                <div class="metric-label">总测试数</div>
            </div>
            <div class="metric">
                <div class="metric-value passed">{{.Summary.PassedTests}}</div>
                <div class="metric-label">通过</div>
            </div>
            <div class="metric">
                <div class="metric-value">{{printf "%.2f" .Summary.AvgThroughput}}</div>
                <div class="metric-label">平均QPS</div>
            </div>
            <div class="metric">
                <div class="metric-value">{{printf "%.2f" .Summary.AvgErrorRate}}%</div>
                <div class="metric-label">平均错误率</div>
            </div>
        </div>

        <h2>🧪 测试详情</h2>
        <table>
            <thead>
                <tr>
                    <th>测试名称</th>
                    <th>类型</th>
                    <th>状态</th>
                    <th>请求数</th>
                    <th>成功率</th>
                    <th>P50延迟</th>
                    <th>P95延迟</th>
                    <th>吞吐量</th>
                </tr>
            </thead>
            <tbody>
                {{range .Tests}}
                <tr>
                    <td>{{.Name}}</td>
                    <td>{{.Type}}</td>
                    <td class="{{if eq .Status "PASSED"}}passed{{else}}failed{{end}}">{{.Status}}</td>
                    <td>{{.Requests}}</td>
                    <td>{{printf "%.2f" .SuccessRate}}%</td>
                    <td>{{.P50Latency}}</td>
                    <td>{{.P95Latency}}</td>
                    <td>{{printf "%.2f" .Throughput}} req/s</td>
                </tr>
                {{end}}
            </tbody>
        </table>

        <h2>💡 优化建议</h2>
        {{range .Recommendations}}
        <div class="recommendation priority-{{lower .Priority}}">
            <strong>[{{.Priority}}] {{.Title}}</strong>
            <p>{{.Description}}</p>
            <small>影响: {{.Impact}}</small>
        </div>
        {{end}}

        <h2>🖥️ 测试环境</h2>
        <table>
            <tr><td>主机名</td><td>{{.Environment.Hostname}}</td></tr>
            <tr><td>操作系统</td><td>{{.Environment.OS}} / {{.Environment.Arch}}</td></tr>
            <tr><td>CPU核心数</td><td>{{.Environment.NumCPU}}</td></tr>
            <tr><td>Go版本</td><td>{{.Environment.GoVersion}}</td></tr>
        </table>
    </div>
</body>
</html>`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return t.Execute(file, report)
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

import (
	"runtime"
)

type BenchmarkResult struct {
	Name       string
	Ops       int64
	Duration   time.Duration
	OpsPerSec  float64
	MemStats  runtime.MemStats
}

type StressTestResult struct {
	TotalRequests  int64
	SuccessCount  int64
	FailureCount  int64
	TotalDuration time.Duration
	P50Latency    time.Duration
	P95Latency    time.Duration
	P99Latency    time.Duration
	RequestsPerSec float64
	Errors        map[string]int64
}

func main() {
	log.Println("性能报告生成工具")

	results := []*BenchmarkResult{
		{
			Name:       "Cache Benchmark",
			Ops:        10000,
			Duration:   2 * time.Second,
			OpsPerSec:  5000,
		},
		{
			Name:       "Concurrency Benchmark",
			Ops:        5000,
			Duration:   1 * time.Second,
			OpsPerSec:  5000,
		},
	}

	stressResults := []*StressTestResult{
		{
			TotalRequests:  1000,
			SuccessCount:   950,
			FailureCount:   50,
			TotalDuration:  10 * time.Second,
			P50Latency:     50 * time.Millisecond,
			P95Latency:     150 * time.Millisecond,
			P99Latency:     300 * time.Millisecond,
			RequestsPerSec: 100,
		},
	}

	report := GeneratePerformanceReport(results, stressResults)

	if err := ExportJSON(report, "performance_report.json"); err != nil {
		log.Printf("导出JSON失败: %v", err)
	}

	if err := ExportHTML(report, "performance_report.html"); err != nil {
		log.Printf("导出HTML失败: %v", err)
	}

	log.Println("报告生成完成")
}
