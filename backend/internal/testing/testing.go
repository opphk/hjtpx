package testing

import (
	"context"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/internal/testing/chaos"
	"github.com/hjtpx/hjtpx/internal/testing/coverage"
	"github.com/hjtpx/hjtpx/internal/testing/fuzzing"
	"github.com/hjtpx/hjtpx/internal/testing/pentest"
)

type TestSuiteResult struct {
	Name     string
	Passed   bool
	Duration time.Duration
	Details  interface{}
	Error    error
}

type TestReport struct {
	Timestamp      time.Time
	FuzzingResults map[string]*fuzzing.FuzzTestResult
	ChaosResults   map[string]*chaos.ExperimentResult
	SecurityResult *pentest.PentestReport
	CoverageReport *coverage.CoverageReport
}

func RunAllTests(rootDir, targetURL, authToken string) *TestReport {
	report := &TestReport{
		Timestamp: time.Now(),
	}

	report.FuzzingResults = fuzzing.RunAllFuzzTests()

	report.ChaosResults = chaos.RunChaosTests()

	report.SecurityResult = pentest.RunAutomatedPentest(targetURL, authToken)

	coverageReport, _ := coverage.RunCoverageEnhancement(rootDir)
	report.CoverageReport = coverageReport

	return report
}

func RunFuzzingTests() map[string]*fuzzing.FuzzTestResult {
	return fuzzing.RunAllFuzzTests()
}

func RunChaosTests() map[string]*chaos.ExperimentResult {
	return chaos.RunChaosTests()
}

func RunSecurityTests(targetURL, authToken string) *pentest.PentestReport {
	return pentest.RunAutomatedPentest(targetURL, authToken)
}

func RunCoverageAnalysis(rootDir string) (*coverage.CoverageReport, error) {
	return coverage.RunCoverageEnhancement(rootDir)
}

func PrintTestSummary(results map[string]*TestSuiteResult) {
	fmt.Println("\n========================================")
	fmt.Println("       Test Summary")
	fmt.Println("========================================\n")

	total := len(results)
	passed := 0

	for name, result := range results {
		status := "✓ PASS"
		if !result.Passed {
			status = "✗ FAIL"
		} else {
			passed++
		}

		fmt.Printf("%s: %s (Duration: %s)\n", status, name, result.Duration)
	}

	fmt.Printf("\nTotal: %d | Passed: %d\n", total, passed)
	fmt.Println("========================================\n")
}

func RunTestSuite(ctx context.Context, name string, fn func() interface{}) *TestSuiteResult {
	result := &TestSuiteResult{
		Name: name,
	}

	start := time.Now()

	done := make(chan struct{})
	go func() {
		defer close(done)
		result.Details = fn()
	}()

	select {
	case <-done:
		result.Duration = time.Since(start)
		result.Passed = true
	case <-ctx.Done():
		result.Duration = time.Since(start)
		result.Passed = false
		result.Error = fmt.Errorf("test suite timed out")
	}

	return result
}
