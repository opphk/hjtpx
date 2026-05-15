const fs = require('fs');
const path = require('path');

class CoverageAlertManager {
  constructor(options = {}) {
    this.coverageDir = options.coverageDir || path.join(__dirname, '..', 'coverage');
    this.config = this.loadJestConfig();
    this.alertThreshold = this.config.coverageAlertThreshold || {
      global: {
        branches: 35,
        functions: 40,
        lines: 45,
        statements: 45
      }
    };
    this.slackWebhook = process.env.SLACK_WEBHOOK_URL;
    this.githubToken = process.env.GITHUB_TOKEN;
  }

  loadJestConfig() {
    const configPath = path.join(__dirname, '..', 'jest.config.js');
    if (!fs.existsSync(configPath)) {
      return {};
    }
    const configContent = fs.readFileSync(configPath, 'utf8');
    const configMatch = configContent.match(/coverageAlertThreshold:\s*({[\s\S]*?}),/);
    if (configMatch) {
      try {
        return { coverageAlertThreshold: eval(`(${configMatch[1]})`) };
      } catch (e) {
        console.warn('Failed to parse alert threshold config');
      }
    }
    return {};
  }

  loadCoverageSummary() {
    const summaryPath = path.join(this.coverageDir, 'coverage-summary.json');
    if (!fs.existsSync(summaryPath)) {
      throw new Error(`Coverage summary not found at ${summaryPath}`);
    }
    return JSON.parse(fs.readFileSync(summaryPath, 'utf8'));
  }

  checkAlertConditions(coverage) {
    const total = coverage.total;
    const alerts = {
      critical: [],
      warning: [],
      info: []
    };

    for (const [metric, threshold] of Object.entries(this.alertThreshold.global)) {
      const actual = total[metric].pct;
      const diff = threshold - actual;

      if (diff > 10) {
        alerts.critical.push({
          metric,
          actual,
          threshold,
          diff,
          severity: 'critical',
          message: `${metric} coverage is critically low: ${actual.toFixed(2)}% (threshold: ${threshold}%)`
        });
      } else if (diff > 5) {
        alerts.warning.push({
          metric,
          actual,
          threshold,
          diff,
          severity: 'warning',
          message: `${metric} coverage is below alert threshold: ${actual.toFixed(2)}% (threshold: ${threshold}%)`
        });
      } else if (diff > 0) {
        alerts.info.push({
          metric,
          actual,
          threshold,
          diff,
          severity: 'info',
          message: `${metric} coverage approaching threshold: ${actual.toFixed(2)}% (threshold: ${threshold}%)`
        });
      }
    }

    return alerts;
  }

  generateAlertReport(alerts, coverage, branch) {
    const total = coverage.total;
    let report = '# 🚨 测试覆盖率告警报告\n\n';

    if (alerts.critical.length === 0 && alerts.warning.length === 0 && alerts.info.length === 0) {
      report += '✅ 所有覆盖率指标正常,无告警\n';
      return report;
    }

    if (alerts.critical.length > 0) {
      report += '## 🔴 严重告警\n\n';
      report += '| 指标 | 当前 | 阈值 | 差距 |\n';
      report += '|------|------|------|------|\n';
      for (const alert of alerts.critical) {
        report += `| ${alert.metric} | ${alert.actual.toFixed(2)}% | ${alert.threshold}% | -${alert.diff.toFixed(2)}% |\n`;
      }
      report += '\n';
    }

    if (alerts.warning.length > 0) {
      report += '## 🟡 警告\n\n';
      report += '| 指标 | 当前 | 阈值 | 差距 |\n';
      report += '|------|------|------|------|\n';
      for (const alert of alerts.warning) {
        report += `| ${alert.metric} | ${alert.actual.toFixed(2)}% | ${alert.threshold}% | -${alert.diff.toFixed(2)}% |\n`;
      }
      report += '\n';
    }

    if (alerts.info.length > 0) {
      report += '## 🔵 提示\n\n';
      report += '| 指标 | 当前 | 阈值 | 差距 |\n';
      report += '|------|------|------|------|\n';
      for (const alert of alerts.info) {
        report += `| ${alert.metric} | ${alert.actual.toFixed(2)}% | ${alert.threshold}% | -${alert.diff.toFixed(2)}% |\n`;
      }
      report += '\n';
    }

    report += '## 📊 当前覆盖率详情\n\n';
    report += '| 指标 | 覆盖数 | 总数 | 百分比 |\n';
    report += '|------|--------|------|--------|\n';
    for (const metric of ['branches', 'functions', 'lines', 'statements']) {
      report += `| ${metric} | ${total[metric].covered} | ${total[metric].total} | ${total[metric].pct.toFixed(2)}% |\n`;
    }

    report += `\n**分支:** ${branch || 'unknown'}\n`;
    report += `**时间:** ${new Date().toISOString()}\n`;

    return report;
  }

  async sendSlackNotification(alerts) {
    if (!this.slackWebhook) {
      console.log('No Slack webhook configured, skipping notification');
      return;
    }

    if (alerts.critical.length === 0 && alerts.warning.length === 0) {
      return;
    }

    const payload = {
      text: '🚨 Test Coverage Alert',
      attachments: []
    };

    if (alerts.critical.length > 0) {
      const fields = alerts.critical.map(a => ({
        title: a.metric,
        value: `${a.actual.toFixed(2)}% (threshold: ${a.threshold}%)`,
        short: true
      }));

      payload.attachments.push({
        color: 'danger',
        title: '🔴 Critical Coverage Alerts',
        fields
      });
    }

    if (alerts.warning.length > 0) {
      const fields = alerts.warning.map(a => ({
        title: a.metric,
        value: `${a.actual.toFixed(2)}% (threshold: ${a.threshold}%)`,
        short: true
      }));

      payload.attachments.push({
        color: 'warning',
        title: '🟡 Warning Coverage Alerts',
        fields
      });
    }

    try {
      const response = await fetch(this.slackWebhook, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });

      if (response.ok) {
        console.log('✅ Slack notification sent successfully');
      } else {
        console.error('❌ Failed to send Slack notification:', response.status);
      }
    } catch (error) {
      console.error('❌ Error sending Slack notification:', error.message);
    }
  }

  async createGitHubCheckRun(alerts, coverage, branch) {
    if (!this.githubToken || !process.env.GITHUB_SHA) {
      console.log('No GitHub token or SHA available, skipping check run');
      return;
    }

    const status = alerts.critical.length > 0 ? 'completed' : 'completed';
    const conclusion = alerts.critical.length > 0 ? 'failure' : 'success';

    const checkRun = {
      name: 'Coverage Alert Check',
      status,
      conclusion,
      output: {
        title: alerts.critical.length > 0 ? 'Coverage Below Critical Threshold' : 'Coverage Alert Check',
        summary: this.generateAlertReport(alerts, coverage, branch),
        text: `Coverage alerts generated at ${new Date().toISOString()}`
      }
    };

    try {
      const response = await fetch(
        `https://api.github.com/repos/${process.env.GITHUB_REPOSITORY}/check-runs`,
        {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${this.githubToken}`,
            'Accept': 'application/vnd.github.v3+json',
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(checkRun)
        }
      );

      if (response.ok) {
        console.log('✅ GitHub check run created successfully');
      } else {
        console.error('❌ Failed to create GitHub check run:', response.status);
      }
    } catch (error) {
      console.error('❌ Error creating GitHub check run:', error.message);
    }
  }

  async run() {
    try {
      const coverage = this.loadCoverageSummary();
      const branch = process.env.GITHUB_REF || 'local';
      const alerts = this.checkAlertConditions(coverage);

      const report = this.generateAlertReport(alerts, coverage, branch);

      const reportPath = path.join(this.coverageDir, 'coverage-alert.md');
      fs.writeFileSync(reportPath, report, 'utf8');
      console.log(`✅ Alert report generated: ${reportPath}`);

      const alertsJsonPath = path.join(this.coverageDir, 'coverage-alerts.json');
      fs.writeFileSync(alertsJsonPath, JSON.stringify(alerts, null, 2), 'utf8');

      await this.sendSlackNotification(alerts);
      await this.createGitHubCheckRun(alerts, coverage, branch);

      if (alerts.critical.length > 0) {
        console.log('🚨 Critical coverage alerts detected!');
        process.exit(1);
      } else if (alerts.warning.length > 0) {
        console.log('🟡 Warning coverage alerts detected');
        process.exit(0);
      } else {
        console.log('✅ No coverage alerts');
        process.exit(0);
      }
    } catch (error) {
      console.error('❌ Alert check failed:', error.message);
      process.exit(1);
    }
  }
}

module.exports = CoverageAlertManager;

if (require.main === module) {
  const alertManager = new CoverageAlertManager();
  alertManager.run();
}
