package captchax

import (
	"fmt"
	"os"
	"text/template"
	"time"
)

type BenchmarkReport struct {
	Timestamp        time.Time
	GoVersion        string
	TotalBenchmarks   int
	TotalDuration     time.Duration
	BenchmarkResults  []BenchmarkResult
	StressTestResults []StressTestData
	StabilityResults  []StabilityTestData
	Recommendations    []string
}

type BenchmarkResult struct {
	Name       string
	OpsPerSec  float64
	NsPerOp    float64
	MBPerSec   float64
	AllocsPerOp int64
	BPerOp     int64
}

type StressTestData struct {
	Name           string
	Concurrency    int
	TotalRequests  int64
	SuccessRate    float64
	RequestsPerSec float64
	AvgLatencyMs   float64
	MaxLatencyMs   float64
	P99LatencyMs   float64
	Duration       time.Duration
}

type StabilityTestData struct {
	Name        string
	Duration    time.Duration
	Availability float64
	ErrorRate   float64
	AvgLatency  float64
	P99Latency  float64
	P999Latency float64
}

type OptimizationSuggestion struct {
	Category    string
	Issue       string
	Suggestion  string
	Impact      string
	Priority    string
}

func GeneratePerformanceReport(benchmarks []BenchmarkResult, stressTests []StressTestData, stabilityTests []StabilityTestData) string {
	report := BenchmarkReport{
		Timestamp:        time.Now(),
		GoVersion:        "1.21+",
		TotalBenchmarks:   len(benchmarks),
		BenchmarkResults:  benchmarks,
		StressTestResults: stressTests,
		StabilityResults:  stabilityTests,
	}

	report.Recommendations = generateRecommendations(benchmarks, stressTests, stabilityTests)

	return formatReport(report)
}

func generateRecommendations(benchmarks []BenchmarkResult, stressTests []StressTestData, stabilityTests []StabilityTestData) []string {
	var recommendations []string

	for _, b := range benchmarks {
		if b.NsPerOp > 1000000 {
			recommendations = append(recommendations, fmt.Sprintf(
				"[HIGH] %s is slow (%.2fms per op). Consider caching or optimizing algorithm.",
				b.Name, b.NsPerOp/1000000))
		}

		if b.AllocsPerOp > 10 {
			recommendations = append(recommendations, fmt.Sprintf(
				"[MEDIUM] %s has high allocation count (%d allocs/op). Reduce allocations.",
				b.Name, b.AllocsPerOp))
		}
	}

	for _, s := range stressTests {
		if s.SuccessRate < 99.0 {
			recommendations = append(recommendations, fmt.Sprintf(
				"[HIGH] %s has low success rate (%.2f%%). Improve error handling.",
				s.Name, s.SuccessRate))
		}

		if s.AvgLatencyMs > 100 {
			recommendations = append(recommendations, fmt.Sprintf(
				"[MEDIUM] %s has high latency (%.2fms avg). Optimize connection pooling.",
				s.Name, s.AvgLatencyMs))
		}
	}

	for _, st := range stabilityTests {
		if st.Availability < 99.5 {
			recommendations = append(recommendations, fmt.Sprintf(
				"[HIGH] %s has low availability (%.2f%%). Implement better retry logic.",
				st.Name, st.Availability))
		}

		if st.P99Latency > 500 {
			recommendations = append(recommendations, fmt.Sprintf(
				"[MEDIUM] %s has high P99 latency (%.2fms). Consider circuit breaker.",
				st.Name, st.P99Latency))
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "[INFO] All tests passed within acceptable thresholds.")
	}

	return recommendations
}

func formatReport(report BenchmarkReport) string {
	tmpl := `
================================================================================
                      CAPTCHAX GO SDK PERFORMANCE REPORT
================================================================================

Generated: {{.Timestamp.Format "2006-01-02 15:04:05 MST"}}
Go Version: {{.GoVersion}}

--------------------------------------------------------------------------------
                              BENCHMARK SUMMARY
--------------------------------------------------------------------------------

Total Benchmarks: {{.TotalBenchmarks}}

{{range .BenchmarkResults}}
{{.Name}}:
  - Operations/sec: {{printf "%.2f" .OpsPerSec}}
  - Nanoseconds/op: {{printf "%.2f" .NsPerOp}}
  - Allocations/op: {{.AllocsPerOp}}
  - Bytes/op: {{.BPerOp}}
{{end}}

--------------------------------------------------------------------------------
                            STRESS TEST RESULTS
--------------------------------------------------------------------------------

{{range .StressTestResults}}
{{.Name}}:
  - Concurrency: {{.Concurrency}}
  - Total Requests: {{.TotalRequests}}
  - Success Rate: {{printf "%.2f" .SuccessRate}}%
  - Requests/sec: {{printf "%.2f" .RequestsPerSec}}
  - Avg Latency: {{printf "%.2f" .AvgLatencyMs}}ms
  - Max Latency: {{printf "%.2f" .MaxLatencyMs}}ms
  - P99 Latency: {{printf "%.2f" .P99LatencyMs}}ms
  - Duration: {{.Duration}}
{{end}}

--------------------------------------------------------------------------------
                          STABILITY TEST RESULTS
--------------------------------------------------------------------------------

{{range .StabilityResults}}
{{.Name}}:
  - Duration: {{.Duration}}
  - Availability: {{printf "%.2f" .Availability}}%
  - Error Rate: {{printf "%.4f" .ErrorRate}}%
  - Avg Latency: {{printf "%.2f" .AvgLatency}}ms
  - P99 Latency: {{printf "%.2f" .P99Latency}}ms
  - P999 Latency: {{printf "%.2f" .P999Latency}}ms
{{end}}

--------------------------------------------------------------------------------
                            RECOMMENDATIONS
--------------------------------------------------------------------------------

{{range .Recommendations}}
{{.}}
{{end}}

================================================================================
                              END OF REPORT
================================================================================
`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return fmt.Sprintf("Error parsing template: %v", err)
	}

	var output strings.Builder
	err = t.Execute(&output, report)
	if err != nil {
		return fmt.Sprintf("Error executing template: %v", err)
	}

	return output.String()
}

var stringsBuilder = struct {
	Join func(string, []string) string
}{
	Join: func(sep string, elems []string) string {
		if len(elems) == 0 {
			return ""
		}
		result := elems[0]
		for i := 1; i < len(elems); i++ {
			result += sep + elems[i]
		}
		return result
	},
}

func SaveReport(report string, filename string) error {
	return os.WriteFile(filename, []byte(report), 0644)
}

func GenerateSummaryReport(benchmarks []BenchmarkResult) string {
	var summary strings.Builder

	summary.WriteString("=== PERFORMANCE SUMMARY ===\n\n")

	summary.WriteString("Fastest Operations:\n")
	for i, b := range benchmarks {
		if i >= 5 {
			break
		}
		summary.WriteString(fmt.Sprintf("  %d. %s: %.2f ops/sec\n", i+1, b.Name, b.OpsPerSec))
	}

	summary.WriteString("\nSlowest Operations:\n")
	for i := len(benchmarks) - 1; i >= 0; i-- {
		if len(benchmarks)-i > 5 {
			break
		}
		summary.WriteString(fmt.Sprintf("  %d. %s: %.2f ops/sec\n",
			len(benchmarks)-i, benchmarks[i].Name, benchmarks[i].OpsPerSec))
	}

	return summary.String()
}
