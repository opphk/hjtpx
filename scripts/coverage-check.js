const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class CoverageChecker {
  constructor(options = {}) {
    this.coverageDir = options.coverageDir || path.join(__dirname, '..', 'coverage');
    this.frontendCoverageDir = options.frontendCoverageDir || path.join(__dirname, '..', 'src', 'frontend', 'coverage');
    this.config = this.loadConfig();
    this.backendThreshold = options.backendThreshold || this.config.backend?.threshold || {
      branches: 80,
      functions: 80,
      lines: 80,
      statements: 80
    };
    this.frontendThreshold = options.frontendThreshold || this.config.frontend?.threshold || {
      branches: 70,
      functions: 70,
      lines: 70,
      statements: 70
    };
  }

  loadConfig() {
    const configPath = path.join(__dirname, '..', 'coverage-config.json');
    if (!fs.existsSync(configPath)) {
      console.warn('Coverage config not found, using defaults');
      return {};
    }
    try {
      return JSON.parse(fs.readFileSync(configPath, 'utf8'));
    } catch (e) {
      console.warn('Failed to parse coverage config:', e.message);
      return {};
    }
  }

  loadCoverageSummary(coverageDir) {
    const summaryPath = path.join(coverageDir, 'coverage-summary.json');
    if (!fs.existsSync(summaryPath)) {
      throw new Error(`Coverage summary not found at ${summaryPath}`);
    }
    return JSON.parse(fs.readFileSync(summaryPath, 'utf8'));
  }

  checkThresholds(coverage, threshold, type) {
    const total = coverage.total;
    const results = {
      type,
      passed: true,
      details: {}
    };

    for (const [metric, minValue] of Object.entries(threshold)) {
      const actual = total[metric]?.pct || 0;
      const passed = actual >= minValue;
      results.details[metric] = {
        actual,
        expected: minValue,
        passed,
        diff: actual - minValue
      };
      if (!passed) {
        results.passed = false;
      }
    }

    return results;
  }

  generateReport(backendResult, frontendResult) {
    let md = '# 📊 测试覆盖率检查报告\n\n';
    md += `**检查时间:** ${new Date().toISOString()}\n\n`;

    // Backend section
    md += '## 🔧 后端覆盖率 (目标: 80%)\n\n';
    md += '| 指标 | 当前值 | 目标值 | 差距 | 状态 |\n';
    md += '|------|--------|--------|------|------|\n';

    for (const [metric, data] of Object.entries(backendResult.details)) {
      const status = data.passed ? '✅ 通过' : '❌ 未通过';
      const diffDisplay = data.diff >= 0 ? `+${data.diff.toFixed(2)}%` : `${data.diff.toFixed(2)}%`;
      md += `| ${metric} | ${data.actual.toFixed(2)}% | ${data.expected}% | ${diffDisplay} | ${status} |\n`;
    }

    md += `\n**后端状态:** ${backendResult.passed ? '✅ 通过' : '❌ 未通过'}\n\n`;

    // Frontend section
    md += '## 🎨 前端覆盖率 (目标: 70%)\n\n';
    md += '| 指标 | 当前值 | 目标值 | 差距 | 状态 |\n';
    md += '|------|--------|--------|------|------|\n';

    for (const [metric, data] of Object.entries(frontendResult.details)) {
      const status = data.passed ? '✅ 通过' : '❌ 未通过';
      const diffDisplay = data.diff >= 0 ? `+${data.diff.toFixed(2)}%` : `${data.diff.toFixed(2)}%`;
      md += `| ${metric} | ${data.actual.toFixed(2)}% | ${data.expected}% | ${diffDisplay} | ${status} |\n`;
    }

    md += `\n**前端状态:** ${frontendResult.passed ? '✅ 通过' : '❌ 未通过'}\n\n`;

    // Summary
    const allPassed = backendResult.passed && frontendResult.passed;
    md += '## 📋 总结\n\n';
    md += `**总体状态:** ${allPassed ? '✅ 全部通过' : '❌ 存在未通过的检查'}\n\n`;

    if (!allPassed) {
      md += '### 🚨 未通过的指标\n\n';
      if (!backendResult.passed) {
        md += '**后端:**\n';
        for (const [metric, data] of Object.entries(backendResult.details)) {
          if (!data.passed) {
            md += `- ${metric}: ${data.actual.toFixed(2)}% < ${data.expected}% (差距: ${data.diff.toFixed(2)}%)\n`;
          }
        }
        md += '\n';
      }
      if (!frontendResult.passed) {
        md += '**前端:**\n';
        for (const [metric, data] of Object.entries(frontendResult.details)) {
          if (!data.passed) {
            md += `- ${metric}: ${data.actual.toFixed(2)}% < ${data.expected}% (差距: ${data.diff.toFixed(2)}%)\n`;
          }
        }
      }
    }

    return { md, allPassed };
  }

  check() {
    try {
      console.log('🔍 开始检查覆盖率...');

      // Check backend coverage
      let backendResult;
      try {
        const backendCoverage = this.loadCoverageSummary(this.coverageDir);
        backendResult = this.checkThresholds(backendCoverage, this.backendThreshold, 'backend');
        console.log(`✅ 后端覆盖率检查完成: ${backendResult.passed ? '通过' : '未通过'}`);
      } catch (error) {
        console.error('❌ 后端覆盖率检查失败:', error.message);
        backendResult = {
          type: 'backend',
          passed: false,
          details: {},
          error: error.message
        };
      }

      // Check frontend coverage
      let frontendResult;
      try {
        const frontendCoverage = this.loadCoverageSummary(this.frontendCoverageDir);
        frontendResult = this.checkThresholds(frontendCoverage, this.frontendThreshold, 'frontend');
        console.log(`✅ 前端覆盖率检查完成: ${frontendResult.passed ? '通过' : '未通过'}`);
      } catch (error) {
        console.error('❌ 前端覆盖率检查失败:', error.message);
        frontendResult = {
          type: 'frontend',
          passed: false,
          details: {},
          error: error.message
        };
      }

      const { md, allPassed } = this.generateReport(backendResult, frontendResult);

      // Save report
      const reportPath = path.join(this.coverageDir, 'coverage-check-report.md');
      fs.writeFileSync(reportPath, md, 'utf8');
      console.log(`✅ 报告已生成: ${reportPath}`);

      // Save JSON result
      const jsonPath = path.join(this.coverageDir, 'coverage-check-result.json');
      fs.writeFileSync(jsonPath, JSON.stringify({
        timestamp: new Date().toISOString(),
        backend: backendResult,
        frontend: frontendResult,
        allPassed
      }, null, 2), 'utf8');

      if (!allPassed) {
        console.log('\n❌ 覆盖率检查未通过');
        process.exit(1);
      } else {
        console.log('\n✅ 覆盖率检查全部通过');
        process.exit(0);
      }
    } catch (error) {
      console.error('❌ 覆盖率检查失败:', error.message);
      process.exit(1);
    }
  }
}

module.exports = CoverageChecker;

if (require.main === module) {
  const checker = new CoverageChecker();
  checker.check();
}
