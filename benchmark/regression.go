package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RegressionConfig struct {
	QPSThreshold       float64 `json:"qps_threshold"`
	P99LatencyThresholdMs int64  `json:"p99_latency_threshold_ms"`
	ErrorRateThreshold float64 `json:"error_rate_threshold"`
	MemoryGrowthLimit  float64 `json:"memory_growth_limit"`
}

type BaselineMetrics struct {
	ScenarioName   string    `json:"scenario_name"`
	QPS            float64   `json:"qps"`
	P50Latency    string    `json:"p50_latency"`
	P95Latency    string    `json:"p95_latency"`
	P99Latency    string    `json:"p99_latency"`
	AvgLatency    string    `json:"avg_latency"`
	ErrorRate     float64   `json:"error_rate"`
	MemoryUsage   uint64    `json:"memory_usage"`
	Timestamp     time.Time `json:"timestamp"`
}

type RegressionResult struct {
	ScenarioName     string   `json:"scenario_name"`
	HasRegression    bool     `json:"has_regression"`
	CurrentQPS       float64  `json:"current_qps"`
	BaselineQPS      float64  `json:"baseline_qps"`
	QPSChange        float64  `json:"qps_change"`
	QPSChangePercent float64  `json:"qps_change_percent"`
	CurrentP99Ms     int64    `json:"current_p99_ms"`
	BaselineP99Ms    int64    `json:"baseline_p99_ms"`
	P99Change        int64    `json:"p99_change"`
	CurrentErrorRate float64 `json:"current_error_rate"`
	BaselineErrorRate float64 `json:"baseline_error_rate"`
	ErrorRateChange  float64 `json:"error_rate_change"`
	CurrentMemoryMB  float64 `json:"current_memory_mb"`
	BaselineMemoryMB float64 `json:"baseline_memory_mb"`
	MemoryChangeMB   float64 `json:"memory_change_mb"`
	Recommendations  []string `json:"recommendations"`
}

type BaselineStore struct {
	Path      string
	Baselines map[string][]BaselineMetrics
}

func NewBaselineStore(basePath string) (*BaselineStore, error) {
	store := &BaselineStore{
		Path:      basePath,
		Baselines: make(map[string][]BaselineMetrics),
	}

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create baseline directory: %w", err)
	}

	if err := store.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return store, nil
}

func (s *BaselineStore) Save() error {
	data, err := json.MarshalIndent(s.Baselines, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal baselines: %w", err)
	}

	filename := filepath.Join(s.Path, "baseline.json")
	return os.WriteFile(filename, data, 0644)
}

func (s *BaselineStore) Load() error {
	filename := filepath.Join(s.Path, "baseline.json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.Baselines)
}

func (s *BaselineStore) AddBaseline(name string, metrics *PerformanceMetrics) {
	baseline := BaselineMetrics{
		ScenarioName: name,
		QPS:          metrics.QPS,
		P50Latency:  metrics.LatencyP50.String(),
		P95Latency:  metrics.LatencyP95.String(),
		P99Latency:  metrics.LatencyP99.String(),
		AvgLatency:  metrics.AvgLatency.String(),
		ErrorRate:   metrics.ErrorRate,
		MemoryUsage: metrics.MemoryUsage,
		Timestamp:   time.Now(),
	}

	s.Baselines[name] = append(s.Baselines[name], baseline)

	if len(s.Baselines[name]) > 10 {
		s.Baselines[name] = s.Baselines[name][len(s.Baselines[name])-10:]
	}

	s.Save()
}

func (s *BaselineStore) GetLatestBaseline(name string) *BaselineMetrics {
	baselines, ok := s.Baselines[name]
	if !ok || len(baselines) == 0 {
		return nil
	}
	return &baselines[len(baselines)-1]
}

func (s *BaselineStore) GetAverageBaseline(name string) *BaselineMetrics {
	baselines, ok := s.Baselines[name]
	if !ok || len(baselines) == 0 {
		return nil
	}

	var totalQPS, totalErrorRate float64
	var totalP50Ms, totalP95Ms, totalP99Ms int64
	var totalMemory uint64

	for _, b := range baselines {
		totalQPS += b.QPS
		totalErrorRate += b.ErrorRate
		totalMemory += b.MemoryUsage

		p50, _ := parseDurationMs(b.P50Latency)
		p95, _ := parseDurationMs(b.P95Latency)
		p99, _ := parseDurationMs(b.P99Latency)

		totalP50Ms += p50
		totalP95Ms += p95
		totalP99Ms += p99
	}

	count := float64(len(baselines))
	return &BaselineMetrics{
		ScenarioName: name,
		QPS:          totalQPS / count,
		P50Latency:   formatDurationMs(totalP50Ms / int64(count)),
		P95Latency:   formatDurationMs(totalP95Ms / int64(count)),
		P99Latency:   formatDurationMs(totalP99Ms / int64(count)),
		ErrorRate:    totalErrorRate / count,
		MemoryUsage:  totalMemory / uint64(count),
		Timestamp:    time.Now(),
	}
}

func (s *BaselineStore) ClearBaselines(name string) error {
	delete(s.Baselines, name)
	return s.Save()
}

func (s *BaselineStore) ExportBaseline(name string) error {
	baselines, ok := s.Baselines[name]
	if !ok {
		return fmt.Errorf("no baselines found for scenario: %s", name)
	}

	data, err := json.MarshalIndent(baselines, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal baselines: %w", err)
	}

	filename := filepath.Join(s.Path, fmt.Sprintf("baseline_%s_%s.json", name, time.Now().Format("20060102_150405")))
	return os.WriteFile(filename, data, 0644)
}

type RegressionDetector struct {
	Store   *BaselineStore
	Config  RegressionConfig
}

func NewRegressionDetector(store *BaselineStore, config RegressionConfig) *RegressionDetector {
	if config.QPSThreshold == 0 {
		config.QPSThreshold = 0.8
	}
	if config.P99LatencyThresholdMs == 0 {
		config.P99LatencyThresholdMs = 500
	}
	if config.ErrorRateThreshold == 0 {
		config.ErrorRateThreshold = 5.0
	}
	if config.MemoryGrowthLimit == 0 {
		config.MemoryGrowthLimit = 0.2
	}

	return &RegressionDetector{
		Store:  store,
		Config: config,
	}
}

func (d *RegressionDetector) Detect(metrics *PerformanceMetrics) *RegressionResult {
	result := &RegressionResult{
		ScenarioName:  metrics.Name,
		HasRegression: false,
		CurrentQPS:    metrics.QPS,
		CurrentP99Ms:   int64(metrics.LatencyP99 / time.Millisecond),
		CurrentErrorRate: metrics.ErrorRate,
		CurrentMemoryMB:  float64(metrics.MemoryUsage) / 1024 / 1024,
		Recommendations: []string{},
	}

	baseline := d.Store.GetLatestBaseline(metrics.Name)
	if baseline == nil {
		result.Recommendations = append(result.Recommendations, "No baseline available for comparison")
		return result
	}

	result.BaselineQPS = baseline.QPS
	result.BaselineP99Ms, _ = parseDurationMs(baseline.P99Latency)
	result.BaselineErrorRate = baseline.ErrorRate
	result.BaselineMemoryMB = float64(baseline.MemoryUsage) / 1024 / 1024

	if result.BaselineQPS > 0 {
		result.QPSChange = result.CurrentQPS - result.BaselineQPS
		result.QPSChangePercent = (result.QPSChange / result.BaselineQPS) * 100
	}

	result.P99Change = result.CurrentP99Ms - result.BaselineP99Ms
	result.ErrorRateChange = result.CurrentErrorRate - result.BaselineErrorRate
	result.MemoryChangeMB = result.CurrentMemoryMB - result.BaselineMemoryMB

	if result.QPSChangePercent < -20 {
		result.HasRegression = true
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("QPS dropped by %.2f%% (baseline: %.2f, current: %.2f)",
				-result.QPSChangePercent, result.BaselineQPS, result.CurrentQPS))
	}

	if result.CurrentP99Ms > result.BaselineP99Ms*150/100 && result.CurrentP99Ms > int64(d.Config.P99LatencyThresholdMs) {
		result.HasRegression = true
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("P99 latency increased by %dms (baseline: %dms, current: %dms)",
				result.P99Change, result.BaselineP99Ms, result.CurrentP99Ms))
	}

	if result.CurrentErrorRate > result.BaselineErrorRate*2 && result.CurrentErrorRate > d.Config.ErrorRateThreshold {
		result.HasRegression = true
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Error rate increased (baseline: %.2f%%, current: %.2f%%)",
				result.BaselineErrorRate, result.CurrentErrorRate))
	}

	if result.MemoryChangeMB > result.BaselineMemoryMB*d.Config.MemoryGrowthLimit && result.MemoryChangeMB > 50 {
		result.HasRegression = true
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Memory usage increased by %.2f MB (baseline: %.2f MB, current: %.2f MB)",
				result.MemoryChangeMB, result.BaselineMemoryMB, result.CurrentMemoryMB))
	}

	if !result.HasRegression {
		result.Recommendations = append(result.Recommendations, "No significant performance regression detected")
	}

	return result
}

func (d *RegressionDetector) DetectAll(results []*ScenarioResult) []*RegressionResult {
	regressionResults := make([]*RegressionResult, 0, len(results))

	for _, result := range results {
		detection := d.Detect(result.Metrics)
		regressionResults = append(regressionResults, detection)
	}

	return regressionResults
}

func (d *RegressionDetector) SaveBaseline(metrics *PerformanceMetrics) {
	d.Store.AddBaseline(metrics.Name, metrics)
}

func parseDurationMs(s string) (int64, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return int64(d / time.Millisecond), nil
}

func formatDurationMs(ms int64) string {
	return (time.Duration(ms) * time.Millisecond).String()
}

type ReportGenerator struct {
	OutputDir string
}

func NewReportGenerator(outputDir string) (*ReportGenerator, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &ReportGenerator{
		OutputDir: outputDir,
	}, nil
}

func (g *ReportGenerator) GenerateHTMLReport(results []*ScenarioResult, regressions []*RegressionResult) error {
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
        h1 { color: #333; margin-bottom: 20px; padding: 20px; background: white; border-radius: 8px; }
        h2 { color: #555; margin: 20px 0 10px; }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin-bottom: 30px; }
        .metric-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .metric-card h3 { font-size: 14px; color: #888; margin-bottom: 5px; }
        .metric-card .value { font-size: 24px; font-weight: bold; color: #333; }
        .metric-card .value.warning { color: #f0ad4e; }
        .metric-card .value.danger { color: #d9534f; }
        .metric-card .value.success { color: #5cb85c; }
        table { width: 100%; border-collapse: collapse; background: white; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 30px; }
        th { background: #4a90d9; color: white; padding: 12px; text-align: left; }
        td { padding: 12px; border-bottom: 1px solid #eee; }
        tr:last-child td { border-bottom: none; }
        tr:hover { background: #f9f9f9; }
        .status { padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; }
        .status.pass { background: #d4edda; color: #155724; }
        .status.fail { background: #f8d7da; color: #721c24; }
        .status.warning { background: #fff3cd; color: #856404; }
        .regression-section { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 30px; }
        .regression-item { padding: 15px; border-left: 4px solid; margin-bottom: 10px; }
        .regression-item.has-regression { border-color: #d9534f; background: #fdf2f2; }
        .regression-item.no-regression { border-color: #5cb85c; background: #f0fdf4; }
        .timestamp { color: #888; font-size: 12px; }
        pre { background: #f8f8f8; padding: 15px; border-radius: 4px; overflow-x: auto; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📊 Performance Benchmark Report</h1>
        <p class="timestamp">Generated at: ` + time.Now().Format(time.RFC3339) + `</p>
`

	html += `<h2>Summary</h2>
        <div class="summary">
            <div class="metric-card">
                <h3>Total Scenarios</h3>
                <div class="value">` + fmt.Sprintf("%d", len(results)) + `</div>
            </div>
            <div class="metric-card">
                <h3>Total Requests</h3>
                <div class="value">` + fmt.Sprintf("%d", calculateTotalRequests(results)) + `</div>
            </div>
            <div class="metric-card">
                <h3>Combined QPS</h3>
                <div class="value">` + fmt.Sprintf("%.2f", calculateTotalQPS(results)) + `</div>
            </div>
            <div class="metric-card">
                <h3>Avg Error Rate</h3>
                <div class="value">` + fmt.Sprintf("%.2f%%", calculateAvgErrorRate(results)) + `</div>
            </div>
            <div class="metric-card">
                <h3>Regressions</h3>
                <div class="value ` + getRegressionStatusClass(regressions) + `">` + fmt.Sprintf("%d", countRegressions(regressions)) + `</div>
            </div>
        </div>
`

	html += `<h2>Scenario Results</h2>
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
		html += `<h2>Regression Analysis</h2>
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
                <p>Memory: %.2f MB (baseline: %.2f MB)</p>
                <ul>%s</ul>
            </div>
`, class, reg.ScenarioName, reg.CurrentQPS, reg.BaselineQPS, reg.QPSChangePercent,
				reg.CurrentP99Ms, reg.BaselineP99Ms, reg.CurrentErrorRate, reg.BaselineErrorRate,
				reg.CurrentMemoryMB, reg.BaselineMemoryMB, formatRecommendations(reg.Recommendations))
		}

		html += `        </div>
`
	}

	html += `    </div>
</body>
</html>
`

	filename := filepath.Join(g.OutputDir, fmt.Sprintf("benchmark_report_%s.html", time.Now().Format("20060102_150405")))
	return os.WriteFile(filename, []byte(html), 0644)
}

func (g *ReportGenerator) GenerateJSONReport(results []*ScenarioResult, regressions []*RegressionResult) error {
	report := struct {
		GeneratedAt        time.Time              `json:"generated_at"`
		TotalScenarios     int                    `json:"total_scenarios"`
		TotalRequests      int64                  `json:"total_requests"`
		TotalQPS           float64                `json:"total_qps"`
		AvgErrorRate       float64               `json:"avg_error_rate"`
		Scenarios          []ScenarioReport       `json:"scenarios"`
		Regressions        []*RegressionResult    `json:"regressions"`
	}{
		GeneratedAt:    time.Now(),
		TotalScenarios: len(results),
		TotalRequests:  calculateTotalRequests(results),
		TotalQPS:       calculateTotalQPS(results),
		AvgErrorRate:   calculateAvgErrorRate(results),
		Scenarios:      make([]ScenarioReport, 0, len(results)),
		Regressions:    regressions,
	}

	for _, result := range results {
		report.Scenarios = append(report.Scenarios, ScenarioReport{
			Name:              result.Metrics.Name,
			Description:       result.Scenario.Description,
			QPS:               result.Metrics.QPS,
			P50Latency:        result.Metrics.LatencyP50.String(),
			P95Latency:        result.Metrics.LatencyP95.String(),
			P99Latency:        result.Metrics.LatencyP99.String(),
			AvgLatency:        result.Metrics.AvgLatency.String(),
			MinLatency:        result.Metrics.MinLatency.String(),
			MaxLatency:        result.Metrics.MaxLatency.String(),
			ErrorRate:         result.Metrics.ErrorRate,
			TotalRequests:     result.Metrics.TotalRequests,
			SuccessfulRequests: result.Metrics.SuccessfulRequests,
			FailedRequests:    result.Metrics.FailedRequests,
			MemoryUsage:       result.Metrics.MemoryUsage,
			Duration:          result.Metrics.Duration.String(),
		})
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	filename := filepath.Join(g.OutputDir, fmt.Sprintf("benchmark_report_%s.json", time.Now().Format("20060102_150405")))
	return os.WriteFile(filename, data, 0644)
}

type ScenarioReport struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	QPS                float64 `json:"qps"`
	P50Latency        string  `json:"p50_latency"`
	P95Latency        string  `json:"p95_latency"`
	P99Latency        string  `json:"p99_latency"`
	AvgLatency        string  `json:"avg_latency"`
	MinLatency        string  `json:"min_latency"`
	MaxLatency        string  `json:"max_latency"`
	ErrorRate         float64 `json:"error_rate"`
	TotalRequests     int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests    int64   `json:"failed_requests"`
	MemoryUsage       uint64  `json:"memory_usage"`
	Duration          string  `json:"duration"`
}

func calculateTotalRequests(results []*ScenarioResult) int64 {
	var total int64
	for _, r := range results {
		total += r.Metrics.TotalRequests
	}
	return total
}

func calculateTotalQPS(results []*ScenarioResult) float64 {
	var total float64
	for _, r := range results {
		total += r.Metrics.QPS
	}
	return total
}

func calculateAvgErrorRate(results []*ScenarioResult) float64 {
	if len(results) == 0 {
		return 0
	}
	var total float64
	for _, r := range results {
		total += r.Metrics.ErrorRate
	}
	return total / float64(len(results))
}

func countRegressions(regressions []*RegressionResult) int {
	count := 0
	for _, r := range regressions {
		if r.HasRegression {
			count++
		}
	}
	return count
}

func getStatus(metrics *PerformanceMetrics) string {
	if metrics.QPS < 1000 || metrics.LatencyP99 > 200*time.Millisecond || metrics.ErrorRate > 5.0 {
		return "fail"
	}
	if metrics.QPS < 5000 || metrics.LatencyP99 > 100*time.Millisecond || metrics.ErrorRate > 1.0 {
		return "warning"
	}
	return "pass"
}

func getRegressionStatusClass(regressions []*RegressionResult) string {
	count := countRegressions(regressions)
	if count > 0 {
		return "danger"
	}
	return "success"
}

func formatRecommendations(recs []string) string {
	if len(recs) == 0 {
		return "<li>No recommendations</li>"
	}
	result := ""
	for _, r := range recs {
		result += fmt.Sprintf("<li>%s</li>", r)
	}
	return result
}
