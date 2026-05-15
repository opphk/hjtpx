const fs = require('fs');
const path = require('path');

class BranchCoverageChecker {
  constructor(options = {}) {
    this.coverageDir = options.coverageDir || path.join(__dirname, '..', 'coverage');
    this.config = this.loadConfig();
    this.branchRequirements = this.config.branchRequirements || this.getDefaultRequirements();
  }

  getDefaultRequirements() {
    return {
      main: {
        branches: 80,
        functions: 80,
        lines: 80,
        statements: 80
      },
      develop: {
        branches: 75,
        functions: 75,
        lines: 75,
        statements: 75
      },
      feature: {
        branches: 70,
        functions: 70,
        lines: 70,
        statements: 70
      },
      hotfix: {
        branches: 75,
        functions: 75,
        lines: 75,
        statements: 75
      }
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

  getBranchType(branchName) {
    if (!branchName) return 'feature';

    if (branchName === 'main' || branchName === 'master') {
      return 'main';
    } else if (branchName === 'develop' || branchName === 'development') {
      return 'develop';
    } else if (branchName.startsWith('hotfix/') || branchName.startsWith('fix/')) {
      return 'hotfix';
    } else if (branchName.startsWith('feature/') || branchName.startsWith('feat/')) {
      return 'feature';
    } else if (branchName.startsWith('release/')) {
      return 'main';
    }

    return 'feature';
  }

  loadCoverageSummary() {
    const summaryPath = path.join(this.coverageDir, 'coverage-summary.json');
    if (!fs.existsSync(summaryPath)) {
      throw new Error(`Coverage summary not found at ${summaryPath}`);
    }
    return JSON.parse(fs.readFileSync(summaryPath, 'utf8'));
  }

  checkBranchCoverage(coverage, branchType) {
    const requirements = this.branchRequirements[branchType];
    if (!requirements) {
      console.warn(`No specific requirements for branch type: ${branchType}, using feature requirements`);
      return {
        passed: true,
        branchType,
        requirements,
        details: {}
      };
    }

    const total = coverage.total;
    const results = {
      passed: true,
      branchType,
      requirements,
      details: {},
      summary: []
    };

    for (const [metric, threshold] of Object.entries(requirements)) {
      const actual = total[metric].pct;
      const passed = actual >= threshold;
      const diff = threshold - actual;

      results.details[metric] = {
        actual,
        required: threshold,
        passed,
        diff
      };

      if (!passed) {
        results.passed = false;
        results.summary.push({
          metric,
          status: '❌',
          actual: actual.toFixed(2),
          required: threshold,
          diff: diff.toFixed(2)
        });
      } else {
        results.summary.push({
          metric,
          status: '✅',
          actual: actual.toFixed(2),
          required: threshold,
          diff: diff.toFixed(2)
        });
      }
    }

    return results;
  }

  generateMarkdownReport(checkResult, branchName) {
    const { details, summary, passed, branchType, requirements } = checkResult;

    let md = `# 🌿 分支覆盖率检查报告\n\n`;
    md += `**分支:** ${branchName}\n`;
    md += `**分支类型:** ${branchType}\n`;
    md += `**检查时间:** ${new Date().toISOString()}\n`;
    md += `**状态:** ${passed ? '✅ 通过' : '❌ 未通过'}\n\n`;

    md += `## 📋 覆盖率要求\n\n`;
    md += '| 指标 | 要求值 | 当前值 | 差距 | 状态 |\n';
    md += '|------|--------|--------|------|------|\n';

    for (const item of summary) {
      const status = item.diff <= 0 ? '✅ 通过' : '❌ 未通过';
      const diffDisplay = item.diff <= 0 ? `+${Math.abs(item.diff)}%` : `-${item.diff}%`;
      md += `| ${item.metric} | ${item.required}% | ${item.actual}% | ${diffDisplay} | ${status} |\n`;
    }

    md += `\n## 📊 详细数据\n\n`;
    md += '| 指标 | 覆盖数 | 总数 | 百分比 |\n';
    md += '|------|--------|------|--------|\n';

    const coverage = this.loadCoverageSummary();
    for (const metric of ['branches', 'functions', 'lines', 'statements']) {
      const data = coverage.total[metric];
      md += `| ${metric} | ${data.covered} | ${data.total} | ${data.pct.toFixed(2)}% |\n`;
    }

    if (!passed) {
      md += `\n## 🚨 未通过的检查\n\n`;
      md += `分支 **${branchName}** 的覆盖率未达到要求。\n\n`;
      md += `**要求:**\n`;
      for (const [metric, threshold] of Object.entries(requirements)) {
        md += `- ${metric}: ${threshold}%\n`;
      }
      md += `\n**建议:**\n`;
      md += `- 增加对未覆盖代码路径的测试\n`;
      md += `- 编写更多边界条件的测试用例\n`;
      md += `- 在合并前提高覆盖率\n`;
    }

    return md;
  }

  check() {
    try {
      const branchName = process.env.GITHUB_REF?.replace('refs/heads/', '') || 'local';
      const branchType = this.getBranchType(branchName);
      const coverage = this.loadCoverageSummary();
      const result = this.checkBranchCoverage(coverage, branchType);

      const report = this.generateMarkdownReport(result, branchName);

      const reportPath = path.join(this.coverageDir, 'branch-coverage-check.md');
      fs.writeFileSync(reportPath, report, 'utf8');
      console.log(`✅ Branch coverage check report generated: ${reportPath}`);

      const jsonPath = path.join(this.coverageDir, 'branch-coverage-result.json');
      fs.writeFileSync(jsonPath, JSON.stringify({
        passed: result.passed,
        branchName,
        branchType,
        requirements: result.requirements,
        details: result.details,
        summary: result.summary
      }, null, 2), 'utf8');

      if (!result.passed) {
        console.log(`❌ Branch coverage check failed for ${branchName}`);
        console.log('Failed metrics:');
        for (const item of result.summary) {
          if (item.diff > 0) {
            console.log(`  - ${item.metric}: ${item.actual}% < ${item.required}% (need +${item.diff}%)`);
          }
        }
        process.exit(1);
      } else {
        console.log(`✅ Branch coverage check passed for ${branchName} (${branchType})`);
        process.exit(0);
      }

      return result;
    } catch (error) {
      console.error('❌ Branch coverage check failed:', error.message);
      process.exit(1);
    }
  }
}

module.exports = BranchCoverageChecker;

if (require.main === module) {
  const checker = new BranchCoverageChecker();
  checker.check();
}
