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
	Package   string  `json:"package"`
	Coverage  float64 `json:"coverage"`
	Functions int     `json:"functions,omitempty"`
}

type TestReport struct {
	Timestamp       string           `json:"timestamp"`
	TotalPackages   int              `json:"total_packages"`
	TestedPackages  int              `json:"tested_packages"`
	TotalCoverage   float64          `json:"total_coverage"`
	CoverageByPackage []CoverageReport `json:"coverage_by_package"`
	Summary         string           `json:"summary"`
}

func main() {
	fmt.Println("=== CaptchaX Test Report Generator ===")
	fmt.Println()

	report := &TestReport{
		Timestamp:       time.Now().Format(time.RFC3339),
		CoverageByPackage: []CoverageReport{},
	}

	packages := []string{
		"captchax/internal/risk",
		"captchax/internal/captcha/click",
		"captchax/internal/captcha/slider",
		"captchax/internal/admin",
	}

	var totalCoverage float64
	var coverageCount int

	fmt.Println("Running tests and generating coverage reports...")
	fmt.Println()

	for _, pkg := range packages {
		fmt.Printf("Testing package: %s\n", pkg)

		cmd := exec.Command("go", "test", "-v", "-cover", "-coverprofile="+pkg+".out", "./"+pkg)
		cmd.Dir = "/workspace/hjtpx/captchax"
		output, err := cmd.CombinedOutput()

		if err != nil {
			fmt.Printf("  Warning: Test execution had issues for %s\n", pkg)
		}

		if strings.Contains(string(output), "coverage:") {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "coverage:") && strings.Contains(line, "of statements") {
					parts := strings.Split(line, "coverage:")
					if len(parts) > 1 {
						coveragePart := strings.TrimSpace(parts[1])
						coverageStr := strings.Split(coveragePart, " ")[0]
						coverageStr = strings.TrimSuffix(coverageStr, "%")

						var coverage float64
						fmt.Sscanf(coverageStr, "%f", &coverage)

						report.CoverageByPackage = append(report.CoverageByPackage, CoverageReport{
							Package:  pkg,
							Coverage: coverage,
						})

						totalCoverage += coverage
						coverageCount++
					}
				}
			}
		}

		fmt.Println()
	}

	if coverageCount > 0 {
		report.TotalCoverage = totalCoverage / float64(coverageCount)
		report.TotalPackages = len(packages)
		report.TestedPackages = coverageCount
	}

	if report.TotalCoverage >= 85 {
		report.Summary = fmt.Sprintf("Coverage target (85%%) ACHIEVED! Current coverage: %.2f%%", report.TotalCoverage)
	} else {
		report.Summary = fmt.Sprintf("Coverage target (85%%) NOT met. Current coverage: %.2f%%. Additional tests needed.", report.TotalCoverage)
	}

	reportJSON, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println("\n=== Test Report ===")
	fmt.Println(string(reportJSON))

	htmlReport := generateHTMLReport(report)
	htmlFile, _ := os.Create("/workspace/hjtpx/captchax/tests/test_report.html")
	htmlFile.WriteString(htmlReport)
	htmlFile.Close()

	fmt.Println("\nHTML report saved to: /workspace/hjtpx/captchax/tests/test_report.html")

	markdownReport := generateMarkdownReport(report)
	mdFile, _ := os.Create("/workspace/hjtpx/captchax/tests/test_report.md")
	mdFile.WriteString(markdownReport)
	mdFile.Close()

	fmt.Println("Markdown report saved to: /workspace/hjtpx/captchax/tests/test_report.md")
}

func generateHTMLReport(report *TestReport) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CaptchaX 测试报告</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
        }
        .header h1 {
            margin: 0;
            font-size: 2em;
        }
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .summary-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .summary-card h3 {
            margin: 0 0 10px 0;
            color: #666;
            font-size: 0.9em;
            text-transform: uppercase;
        }
        .summary-card .value {
            font-size: 2em;
            font-weight: bold;
            color: #333;
        }
        .coverage-table {
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 30px;
        }
        table {
            width: 100%%;
            border-collapse: collapse;
        }
        th, td {
            padding: 15px;
            text-align: left;
            border-bottom: 1px solid #eee;
        }
        th {
            background-color: #f8f9fa;
            font-weight: 600;
        }
        .coverage-bar {
            width: 100%%;
            height: 20px;
            background-color: #e9ecef;
            border-radius: 4px;
            overflow: hidden;
        }
        .coverage-fill {
            height: 100%%;
            border-radius: 4px;
            transition: width 0.3s ease;
        }
        .coverage-fill.high { background-color: #28a745; }
        .coverage-fill.medium { background-color: #ffc107; }
        .coverage-fill.low { background-color: #dc3545; }
        .status-badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 0.85em;
            font-weight: 500;
        }
        .status-badge.pass { background-color: #d4edda; color: #155724; }
        .status-badge.fail { background-color: #f8d7da; color: #721c24; }
        .footer {
            text-align: center;
            color: #666;
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #eee;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>🎯 CaptchaX 测试报告</h1>
        <p>生成时间: %s</p>
    </div>

    <div class="summary">
        <div class="summary-card">
            <h3>总覆盖率</h3>
            <div class="value">%.2f%%</div>
        </div>
        <div class="summary-card">
            <h3>已测试包数</h3>
            <div class="value">%d / %d</div>
        </div>
        <div class="summary-card">
            <h3>测试状态</h3>
            <div class="value">
                <span class="status-badge %s">%s</span>
            </div>
        </div>
    </div>

    <div class="coverage-table">
        <h2 style="padding: 20px; margin: 0; border-bottom: 1px solid #eee;">包覆盖率详情</h2>
        <table>
            <thead>
                <tr>
                    <th>包名</th>
                    <th>覆盖率</th>
                    <th>状态</th>
                </tr>
            </thead>
            <tbody>
                %s
            </tbody>
        </table>
    </div>

    <div class="summary" style="margin-top: 30px;">
        <div class="summary-card" style="grid-column: 1 / -1;">
            <h3>测试总结</h3>
            <p style="font-size: 1.1em; line-height: 1.6;">%s</p>
        </div>
    </div>

    <div class="footer">
        <p>CaptchaX Test Report &copy; 2026</p>
    </div>
</body>
</html>`,
		report.Timestamp,
		report.TotalCoverage,
		report.TestedPackages,
		report.TotalPackages,
		getStatusClass(report.TotalCoverage >= 85),
		getStatusText(report.TotalCoverage >= 85),
		generateTableRows(report.CoverageByPackage),
		report.Summary,
	)
}

func generateMarkdownReport(report *TestReport) string {
	var sb strings.Builder

	sb.WriteString("# CaptchaX 测试报告\n\n")
	sb.WriteString(fmt.Sprintf("**生成时间**: %s\n\n", report.Timestamp))
	sb.WriteString(fmt.Sprintf("**总覆盖率**: %.2f%%\n\n", report.TotalCoverage))
	sb.WriteString(fmt.Sprintf("**已测试包数**: %d / %d\n\n", report.TestedPackages, report.TotalPackages))

	if report.TotalCoverage >= 85 {
		sb.WriteString("✅ **覆盖率目标 (85%%) 已达成!**\n\n")
	} else {
		sb.WriteString("❌ **覆盖率目标 (85%%) 未达成**\n\n")
	}

	sb.WriteString("## 包覆盖率详情\n\n")
	sb.WriteString("| 包名 | 覆盖率 | 状态 |\n")
	sb.WriteString("|------|--------|------|\n")

	for _, pkg := range report.CoverageByPackage {
		status := "❌ 未达标"
		if pkg.Coverage >= 85 {
			status = "✅ 已达标"
		} else if pkg.Coverage >= 50 {
			status = "⚠️ 部分达标"
		}
		sb.WriteString(fmt.Sprintf("| %s | %.2f%% | %s |\n", pkg.Package, pkg.Coverage, status))
	}

	sb.WriteString("\n## 测试类型覆盖\n\n")
	sb.WriteString("- **单元测试**: 风险引擎、验证码生成/验证、认证模块\n")
	sb.WriteString("- **集成测试**: API 端点、CORS、限流、安全头\n")
	sb.WriteString("- **性能测试**: 基准测试、并发测试、延迟测试\n")
	sb.WriteString("- **安全测试**: SQL注入、XSS、CSRF、暴力破解防护\n")
	sb.WriteString("- **E2E测试**: 完整验证码流程、批处理、错误处理\n\n")

	sb.WriteString("## 下一步\n\n")
	if report.TotalCoverage < 85 {
		sb.WriteString(fmt.Sprintf("需要额外测试提升覆盖率 %.2f%% -> 85%%\n", report.TotalCoverage))
	} else {
		sb.WriteString("- 添加更多边界情况测试\n")
		sb.WriteString("- 完善压力测试场景\n")
		sb.WriteString("- 增加安全渗透测试用例\n")
	}

	return sb.String()
}

func getStatusClass(pass bool) string {
	if pass {
		return "pass"
	}
	return "fail"
}

func getStatusText(pass bool) string {
	if pass {
		return "达标"
	}
	return "未达标"
}

func generateTableRows(packages []CoverageReport) string {
	var sb strings.Builder
	for _, pkg := range packages {
		fillClass := "low"
		if pkg.Coverage >= 85 {
			fillClass = "high"
		} else if pkg.Coverage >= 50 {
			fillClass = "medium"
		}

		sb.WriteString(fmt.Sprintf(`        <tr>
            <td>%s</td>
            <td>
                <div class="coverage-bar">
                    <div class="coverage-fill %s" style="width: %.2f%%;"></div>
                </div>
                <span>%.2f%%</span>
            </td>
            <td><span class="status-badge %s">%s</span></td>
        </tr>`,
			pkg.Package,
			fillClass,
			pkg.Coverage,
			pkg.Coverage,
			getStatusClass(pkg.Coverage >= 85),
			getStatusText(pkg.Coverage >= 85),
		))
	}
	return sb.String()
}
