package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type CoverageReport struct {
	Package       string  `json:"package"`
	Coverage      float64 `json:"coverage"`
	Timestamp     string  `json:"timestamp"`
	TotalPackages int     `json:"total_packages"`
	PassedPackages int    `json:"passed_packages"`
	FailedPackages int    `json:"failed_packages"`
}

type TestResult struct {
	Package    string
	Coverage   float64
	Passed     bool
	Output     string
}

func main() {
	fmt.Println("=== HJTPX Test Coverage Report Generator ===")
	fmt.Println()

	generateCoverageReport()
}

func generateCoverageReport() {
	fmt.Println("Generating comprehensive test coverage report...")
	fmt.Println()

	startTime := time.Now()

	cmd := exec.Command("go", "test", "-coverprofile=coverage_report.out", "-covermode=atomic", "./pkg/service-discovery/...", "./pkg/cdn/...", "./pkg/circuitbreaker/...", "./pkg/export/...", "./pkg/jwt/...")
	cmd.Dir = "/workspace/hjtpx/backend"
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Warning: Some tests failed: %v\n", err)
	}

	outputLines := strings.Split(string(output), "\n")
	
	report := CoverageReport{
		Timestamp:     time.Now().Format(time.RFC3339),
		TotalPackages: 0,
		PassedPackages: 0,
		FailedPackages: 0,
	}

	for _, line := range outputLines {
		if strings.Contains(line, "coverage:") && strings.Contains(line, "of statements") {
			parts := strings.Split(line, "coverage:")
			if len(parts) == 2 {
				coveragePart := strings.TrimSpace(parts[1])
				coveragePart = strings.TrimSuffix(coveragePart, "of statements")
				coveragePart = strings.TrimSpace(coveragePart)
				coveragePart = strings.TrimSuffix(coveragePart, "%")
				
				var coverage float64
				fmt.Sscanf(coveragePart, "%f", &coverage)
				
				packageName := strings.TrimSpace(parts[0])
				packageName = strings.TrimSuffix(packageName, "ok")
				packageName = strings.TrimSuffix(packageName, "FAIL")
				packageName = strings.TrimSpace(packageName)
				
				if strings.Contains(parts[0], "FAIL") {
					report.FailedPackages++
					fmt.Printf("❌ %s: %.1f%%\n", packageName, coverage)
				} else {
					report.PassedPackages++
					fmt.Printf("✅ %s: %.1f%%\n", packageName, coverage)
				}
				
				report.TotalPackages++
				report.Coverage = coverage
			}
		}
	}

	elapsed := time.Since(startTime)
	
	fmt.Println()
	fmt.Println("=== Summary ===")
	fmt.Printf("Total Packages Tested: %d\n", report.TotalPackages)
	fmt.Printf("Passed: %d\n", report.PassedPackages)
	fmt.Printf("Failed: %d\n", report.FailedPackages)
	fmt.Printf("Overall Coverage: %.1f%%\n", report.Coverage)
	fmt.Printf("Time Elapsed: %v\n", elapsed)
	fmt.Println()

	if report.PassedPackages > 0 {
		coveragePercentage := float64(report.PassedPackages) / float64(report.TotalPackages) * 100
		if coveragePercentage >= 80 {
			fmt.Printf("🎉 Great job! %.1f%% of packages passed testing.\n", coveragePercentage)
		} else if coveragePercentage >= 60 {
			fmt.Printf("👍 Good progress! %.1f%% of packages passed testing.\n", coveragePercentage)
		} else {
			fmt.Printf("⚠️  Need more work. %.1f%% of packages passed testing.\n", coveragePercentage)
		}
	}

	generateHTMLReport(report)
	generateJSONReport(report)

	fmt.Println("Reports generated:")
	fmt.Println("  - coverage_report.html")
	fmt.Println("  - coverage_report.json")
	fmt.Println()
}

func generateHTMLReport(report CoverageReport) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>HJTPX Test Coverage Report</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px;
            text-align: center;
        }
        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }
        .header p {
            font-size: 1.2em;
            opacity: 0.9;
        }
        .content {
            padding: 40px;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }
        .stat-card {
            background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
            padding: 30px;
            border-radius: 8px;
            text-align: center;
        }
        .stat-card h3 {
            font-size: 3em;
            color: #667eea;
            margin-bottom: 10px;
        }
        .stat-card p {
            font-size: 1.1em;
            color: #666;
        }
        .section-title {
            font-size: 2em;
            color: #333;
            margin: 40px 0 20px;
            border-bottom: 3px solid #667eea;
            padding-bottom: 10px;
        }
        .footer {
            background: #f5f7fa;
            padding: 20px;
            text-align: center;
            color: #666;
        }
        .success {
            color: #10b981;
        }
        .warning {
            color: #f59e0b;
        }
        .error {
            color: #ef4444;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎯 HJTPX Test Coverage Report</h1>
            <p>Comprehensive Testing Coverage Analysis</p>
        </div>
        <div class="content">
            <div class="stats">
                <div class="stat-card">
                    <h3 class="success">%d</h3>
                    <p>Total Packages</p>
                </div>
                <div class="stat-card">
                    <h3 class="success">%d</h3>
                    <p>Passed</p>
                </div>
                <div class="stat-card">
                    <h3 class="warning">%d</h3>
                    <p>Failed</p>
                </div>
                <div class="stat-card">
                    <h3>%.1f%%</h3>
                    <p>Coverage</p>
                </div>
            </div>
            
            <h2 class="section-title">📊 Detailed Analysis</h2>
            <p><strong>Report Generated:</strong> %s</p>
            <p><strong>Test Duration:</strong> Comprehensive</p>
            <p><strong>Quality Score:</strong> %.1f%%</p>
            
            <h2 class="section-title">📝 Recommendations</h2>
            <ul>
                <li>Continue adding unit tests for uncovered functions</li>
                <li>Focus on integration tests for critical paths</li>
                <li>Add performance benchmarks for key operations</li>
                <li>Implement property-based testing for complex algorithms</li>
            </ul>
            
            <h2 class="section-title">✅ Quality Metrics</h2>
            <ul>
                <li>Code coverage is essential for maintaining code quality</li>
                <li>Test-driven development improves code reliability</li>
                <li>Integration tests ensure system components work together</li>
                <li>Performance tests prevent regressions</li>
            </ul>
        </div>
        <div class="footer">
            <p>Generated by HJTPX Test Coverage Report Generator</p>
            <p>© 2026 HJTPX Team</p>
        </div>
    </div>
</body>
</html>`, 
		report.TotalPackages, 
		report.PassedPackages, 
		report.FailedPackages,
		report.Coverage,
		report.Timestamp,
		float64(report.PassedPackages)/float64(report.TotalPackages)*100,
	)

	file, _ := os.Create("/workspace/hjtpx/backend/coverage_report.html")
	defer file.Close()
	file.WriteString(html)
}

func generateJSONReport(report CoverageReport) {
	jsonData, _ := json.MarshalIndent(report, "", "  ")
	file, _ := os.Create("/workspace/hjtpx/backend/coverage_report.json")
	defer file.Close()
	file.Write(jsonData)
}
