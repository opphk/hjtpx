import * as fs from 'fs';
import * as path from 'path';

interface TestResult {
  title: string;
  status: 'passed' | 'failed' | 'skipped';
  duration: number;
  error?: string;
}

interface TestSuite {
  name: string;
  tests: TestResult[];
  duration: number;
}

interface TestReport {
  timestamp: string;
  summary: {
    total: number;
    passed: number;
    failed: number;
    skipped: number;
    duration: number;
  };
  suites: TestSuite[];
  screenshots: string[];
  recommendations: string[];
}

class ReportGenerator {
  private resultsDir: string;
  private reportDir: string;
  private screenshotsDir: string;

  constructor() {
    this.resultsDir = path.join(process.cwd(), 'test-results');
    this.reportDir = path.join(process.cwd(), 'test-reports');
    this.screenshotsDir = path.join(process.cwd(), 'test-screenshots');
    
    this.ensureDir(this.reportDir);
  }

  private ensureDir(dir: string): void {
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
  }

  public generateReport(): TestReport {
    console.log('开始生成测试报告...');
    
    const testResults = this.loadTestResults();
    const screenshots = this.getScreenshots();
    const report = this.createReport(testResults, screenshots);
    
    this.saveReport(report);
    this.saveHTMLReport(report);
    
    console.log(`测试报告已生成: ${path.join(this.reportDir, 'test-report.json')}`);
    console.log(`HTML报告已生成: ${path.join(this.reportDir, 'test-report.html')}`);
    
    return report;
  }

  private loadTestResults(): any {
    const resultsFile = path.join(this.resultsDir, 'results.json');
    
    if (fs.existsSync(resultsFile)) {
      const content = fs.readFileSync(resultsFile, 'utf-8');
      return JSON.parse(content);
    }
    
    return { stats: { total: 0, passed: 0, failed: 0, skipped: 0 } };
  }

  private getScreenshots(): string[] {
    if (!fs.existsSync(this.screenshotsDir)) {
      return [];
    }
    
    return fs.readdirSync(this.screenshotsDir)
      .filter(file => file.endsWith('.png'))
      .map(file => path.join(this.screenshotsDir, file));
  }

  private createReport(testResults: any, screenshots: string[]): TestReport {
    const stats = testResults.stats || { total: 0, passed: 0, failed: 0, skipped: 0 };
    
    const report: TestReport = {
      timestamp: new Date().toISOString(),
      summary: {
        total: stats.total,
        passed: stats.passed,
        failed: stats.failed,
        skipped: stats.skipped,
        duration: stats.duration || 0
      },
      suites: this.extractSuites(testResults),
      screenshots: screenshots.map(s => path.basename(s)),
      recommendations: this.generateRecommendations(stats)
    };
    
    return report;
  }

  private extractSuites(testResults: any): TestSuite[] {
    const suites: TestSuite[] = [];
    
    if (testResults.suites) {
      for (const suite of testResults.suites) {
        suites.push({
          name: suite.title || 'Unknown Suite',
          tests: suite.specs?.map((spec: any) => ({
            title: spec.title,
            status: this.mapStatus(spec.ok),
            duration: spec.tests?.[0]?.results?.[0]?.duration || 0,
            error: spec.tests?.[0]?.results?.[0]?.error?.message
          })) || [],
          duration: suite.specs?.reduce((sum: number, spec: any) => 
            sum + (spec.tests?.[0]?.results?.[0]?.duration || 0), 0) || 0
        });
      }
    }
    
    return suites;
  }

  private mapStatus(ok: boolean | undefined): 'passed' | 'failed' | 'skipped' {
    if (ok === undefined) return 'skipped';
    return ok ? 'passed' : 'failed';
  }

  private generateRecommendations(stats: any): string[] {
    const recommendations: string[] = [];
    
    if (stats.failed > 0) {
      recommendations.push('存在失败的测试用例，需要修复后再继续开发');
    }
    
    if (stats.duration > 60000) {
      recommendations.push('测试执行时间较长，建议优化测试性能');
    }
    
    if (stats.passed > 0 && stats.failed === 0) {
      recommendations.push('所有测试通过！系统基本功能运行正常');
    }
    
    recommendations.push('建议定期执行E2E测试，确保系统稳定性');
    recommendations.push('建议添加更多边界条件和异常场景测试');
    
    return recommendations;
  }

  private saveReport(report: TestReport): void {
    const filename = `test-report-${new Date().toISOString().split('T')[0]}.json`;
    const filepath = path.join(this.reportDir, filename);
    
    fs.writeFileSync(filepath, JSON.stringify(report, null, 2), 'utf-8');
  }

  private saveHTMLReport(report: TestReport): void {
    const html = this.generateHTML(report);
    const filename = `test-report-${new Date().toISOString().split('T')[0]}.html`;
    const filepath = path.join(this.reportDir, filename);
    
    fs.writeFileSync(filepath, html, 'utf-8');
  }

  private generateHTML(report: TestReport): string {
    const passRate = report.summary.total > 0 
      ? ((report.summary.passed / report.summary.total) * 100).toFixed(2)
      : '0.00';

    return `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>E2E测试报告 - ${report.timestamp}</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
    .container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
    .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; border-radius: 8px 8px 0 0; }
    .header h1 { font-size: 28px; margin-bottom: 10px; }
    .header .timestamp { opacity: 0.9; font-size: 14px; }
    .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; padding: 30px; }
    .stat-card { background: #f8f9fa; padding: 20px; border-radius: 8px; text-align: center; }
    .stat-card.passed { background: #d4edda; }
    .stat-card.failed { background: #f8d7da; }
    .stat-card .number { font-size: 36px; font-weight: bold; margin-bottom: 5px; }
    .stat-card .label { color: #666; font-size: 14px; }
    .content { padding: 30px; }
    .section { margin-bottom: 30px; }
    .section h2 { font-size: 20px; margin-bottom: 15px; color: #333; border-bottom: 2px solid #667eea; padding-bottom: 10px; }
    .test-list { list-style: none; }
    .test-item { padding: 15px; margin-bottom: 10px; background: #f8f9fa; border-radius: 5px; border-left: 4px solid #ddd; }
    .test-item.passed { border-left-color: #28a745; }
    .test-item.failed { border-left-color: #dc3545; }
    .test-item .title { font-weight: 500; margin-bottom: 5px; }
    .test-item .meta { font-size: 12px; color: #666; }
    .test-item .error { color: #dc3545; font-size: 12px; margin-top: 5px; }
    .screenshot-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(250px, 1fr)); gap: 15px; }
    .screenshot-item { background: #f8f9fa; padding: 10px; border-radius: 5px; text-align: center; }
    .screenshot-item img { width: 100%; height: auto; border-radius: 5px; margin-bottom: 10px; }
    .screenshot-item .name { font-size: 12px; color: #666; }
    .recommendations { background: #e7f3ff; padding: 20px; border-radius: 5px; border-left: 4px solid #2196F3; }
    .recommendations ul { margin-left: 20px; }
    .recommendations li { margin-bottom: 10px; line-height: 1.6; }
    .pass-rate { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px; border-radius: 8px; text-align: center; margin: 20px 0; }
    .pass-rate .rate { font-size: 48px; font-weight: bold; }
    .pass-rate .label { font-size: 14px; opacity: 0.9; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>E2E测试报告</h1>
      <div class="timestamp">生成时间: ${report.timestamp}</div>
    </div>
    
    <div class="summary">
      <div class="stat-card">
        <div class="number">${report.summary.total}</div>
        <div class="label">总测试数</div>
      </div>
      <div class="stat-card passed">
        <div class="number">${report.summary.passed}</div>
        <div class="label">通过</div>
      </div>
      <div class="stat-card failed">
        <div class="number">${report.summary.failed}</div>
        <div class="label">失败</div>
      </div>
      <div class="stat-card">
        <div class="number">${(report.summary.duration / 1000).toFixed(2)}s</div>
        <div class="label">总耗时</div>
      </div>
    </div>
    
    <div class="content">
      <div class="pass-rate">
        <div class="rate">${passRate}%</div>
        <div class="label">测试通过率</div>
      </div>
      
      <div class="section">
        <h2>测试套件</h2>
        ${report.suites.map(suite => `
          <div style="margin-bottom: 20px;">
            <h3 style="color: #667eea; margin-bottom: 10px;">${suite.name}</h3>
            <ul class="test-list">
              ${suite.tests.map(test => `
                <li class="test-item ${test.status}">
                  <div class="title">${test.title}</div>
                  <div class="meta">状态: ${test.status === 'passed' ? '✅ 通过' : test.status === 'failed' ? '❌ 失败' : '⏭️ 跳过'} | 耗时: ${(test.duration / 1000).toFixed(2)}s</div>
                  ${test.error ? `<div class="error">错误: ${test.error}</div>` : ''}
                </li>
              `).join('')}
            </ul>
          </div>
        `).join('')}
      </div>
      
      <div class="section">
        <h2>截图证据 (${report.screenshots.length}张)</h2>
        ${report.screenshots.length > 0 ? `
          <div class="screenshot-grid">
            ${report.screenshots.map(screenshot => `
              <div class="screenshot-item">
                <div class="name">${screenshot}</div>
              </div>
            `).join('')}
          </div>
        ` : '<p>暂无截图</p>'}
      </div>
      
      <div class="section">
        <h2>建议</h2>
        <div class="recommendations">
          <ul>
            ${report.recommendations.map(rec => `<li>${rec}</li>`).join('')}
          </ul>
        </div>
      </div>
    </div>
  </div>
</body>
</html>
    `.trim();
  }
}

if (require.main === module) {
  const generator = new ReportGenerator();
  const report = generator.generateReport();
  console.log('\n测试报告摘要:');
  console.log(`总测试数: ${report.summary.total}`);
  console.log(`通过: ${report.summary.passed}`);
  console.log(`失败: ${report.summary.failed}`);
  console.log(`通过率: ${((report.summary.passed / report.summary.total) * 100).toFixed(2)}%`);
}

export { ReportGenerator, TestReport, TestSuite, TestResult };
