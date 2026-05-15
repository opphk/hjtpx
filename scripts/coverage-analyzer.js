const fs = require('fs');
const path = require('path');

class CoverageAnalyzer {
  constructor(options = {}) {
    this.coverageDir = options.coverageDir || path.join(__dirname, '..', 'coverage');
    this.historyFile = options.historyFile || path.join(this.coverageDir, 'coverage-history.json');
    this.minThreshold = options.minThreshold || {
      branches: 80,
      functions: 80,
      lines: 80,
      statements: 80
    };
    this.dropThreshold = options.dropThreshold || 5;
  }

  loadCoverageSummary() {
    const summaryPath = path.join(this.coverageDir, 'coverage-summary.json');
    if (!fs.existsSync(summaryPath)) {
      throw new Error(`Coverage summary not found at ${summaryPath}`);
    }
    return JSON.parse(fs.readFileSync(summaryPath, 'utf8'));
  }

  loadCoverageHistory() {
    if (!fs.existsSync(this.historyFile)) {
      return [];
    }
    return JSON.parse(fs.readFileSync(this.historyFile, 'utf8'));
  }

  saveCoverageHistory(history) {
    const dir = path.dirname(this.historyFile);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
    fs.writeFileSync(this.historyFile, JSON.stringify(history, null, 2), 'utf8');
  }

  addCoverageToHistory(coverage) {
    const history = this.loadCoverageHistory();
    const entry = {
      timestamp: new Date().toISOString(),
      commit: process.env.GITHUB_SHA || 'local',
      branch: process.env.GITHUB_REF || 'local',
      coverage: coverage.total
    };
    history.push(entry);
    this.saveCoverageHistory(history);
    return entry;
  }

  checkThresholds(coverage) {
    const total = coverage.total;
    const results = {
      passed: true,
      details: {}
    };

    for (const [metric, threshold] of Object.entries(this.minThreshold)) {
      const actual = total[metric].pct;
      const passed = actual >= threshold;
      results.details[metric] = {
        actual,
        expected: threshold,
        passed
      };
      if (!passed) {
        results.passed = false;
      }
    }

    return results;
  }

  checkCoverageDrop(coverage, history) {
    if (history.length < 2) {
      return { dropped: false, details: null };
    }

    const prevEntry = history[history.length - 2];
    const current = coverage.total;
    const previous = prevEntry.coverage;

    const results = {
      dropped: false,
      details: {}
    };

    for (const metric of ['branches', 'functions', 'lines', 'statements']) {
      const diff = previous[metric].pct - current[metric].pct;
      if (diff > this.dropThreshold) {
        results.dropped = true;
        results.details[metric] = {
          previous: previous[metric].pct,
          current: current[metric].pct,
          drop: diff
        };
      }
    }

    return results;
  }

  generateMarkdownReport(coverage, thresholdCheck, dropCheck, history) {
    const total = coverage.total;
    let md = '# 📊 测试覆盖率报告\n\n';

    md += '## 📈 总体覆盖率\n\n';
    md += '| 指标 | 当前 | 阈值 | 状态 |\n';
    md += '|------|------|------|------|\n';

    for (const [metric, data] of Object.entries(thresholdCheck.details)) {
      const status = data.passed ? '✅ 通过' : '❌ 未通过';
      md += `| ${metric} | ${data.actual.toFixed(2)}% | ${data.expected}% | ${status} |\n`;
    }

    md += '\n## 🔍 详细数据\n\n';
    md += '| 指标 | 覆盖数 | 总数 | 百分比 |\n';
    md += '|------|--------|------|--------|\n';

    for (const [metric, data] of Object.entries(total)) {
      if (metric !== 'branches' && metric !== 'functions' && metric !== 'lines' && metric !== 'statements') continue;
      md += `| ${metric} | ${data.covered} | ${data.total} | ${data.pct.toFixed(2)}% |\n`;
    }

    if (dropCheck.dropped) {
      md += '\n## ⚠️ 覆盖率下降警告\n\n';
      md += '| 指标 | 之前 | 当前 | 下降 |\n';
      md += '|------|------|------|------|\n';

      for (const [metric, data] of Object.entries(dropCheck.details)) {
        md += `| ${metric} | ${data.previous.toFixed(2)}% | ${data.current.toFixed(2)}% | -${data.drop.toFixed(2)}% |\n`;
      }
    }

    if (history.length > 0) {
      md += '\n## 📜 历史趋势\n\n';
      md += '最近 5 次提交的覆盖率趋势：\n\n';

      const recentHistory = history.slice(-5);
      for (const entry of recentHistory.reverse()) {
        const date = new Date(entry.timestamp).toLocaleString();
        const commit = entry.commit.substring(0, 8);
        const lines = entry.coverage.lines.pct.toFixed(2);
        md += `- ${date} (${commit}): ${lines}% 行覆盖率\n`;
      }
    }

    return md;
  }

  analyze() {
    const coverage = this.loadCoverageSummary();
    const history = this.loadCoverageHistory();
    const thresholdCheck = this.checkThresholds(coverage);
    const dropCheck = this.checkCoverageDrop(coverage, history);

    this.addCoverageToHistory(coverage);

    const report = this.generateMarkdownReport(coverage, thresholdCheck, dropCheck, history);

    return {
      coverage,
      thresholdCheck,
      dropCheck,
      history,
      report
    };
  }
}

module.exports = CoverageAnalyzer;

if (require.main === module) {
  const analyzer = new CoverageAnalyzer();
  try {
    const result = analyzer.analyze();
    
    console.log('✅ 覆盖率分析完成');
    console.log(`- 阈值检查: ${result.thresholdCheck.passed ? '✅ 通过' : '❌ 未通过'}`);
    console.log(`- 覆盖率下降: ${result.dropCheck.dropped ? '⚠️ 是' : '✅ 否'}`);
    
    const reportPath = path.join(analyzer.coverageDir, 'coverage-report.md');
    fs.writeFileSync(reportPath, result.report, 'utf8');
    console.log(`- 报告已生成: ${reportPath}`);
    
    const jsonPath = path.join(analyzer.coverageDir, 'analysis-result.json');
    fs.writeFileSync(jsonPath, JSON.stringify({
      thresholdCheck: result.thresholdCheck,
      dropCheck: result.dropCheck
    }, null, 2), 'utf8');
    
    if (!result.thresholdCheck.passed || result.dropCheck.dropped) {
      process.exit(1);
    }
  } catch (error) {
    console.error('❌ 覆盖率分析失败:', error.message);
    process.exit(1);
  }
}
