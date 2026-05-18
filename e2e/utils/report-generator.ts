import * as fs from 'fs';
import * as path from 'path';

interface TestResult {
  title: string;
  status: 'passed' | 'failed' | 'skipped';
  duration: number;
  error?: string;
  screenshots?: string[];
  consoleErrors?: string[];
}

interface SuiteResult {
  name: string;
  tests: TestResult[];
  duration: number;
}

interface ReportData {
  generatedAt: string;
  totalTests: number;
  passed: number;
  failed: number;
  skipped: number;
  duration: number;
  suites: SuiteResult[];
  consoleErrors: ConsoleError[];
  screenshots: string[];
  recommendations: string[];
}

interface ConsoleError {
  message: string;
  count: number;
  pages: string[];
}

export class ReportGenerator {
  private resultsDir: string;
  private screenshotsDir: string;

  constructor(
    resultsDir: string = 'test-results',
    screenshotsDir: string = 'test-screenshots'
  ) {
    this.resultsDir = resultsDir;
    this.screenshotsDir = screenshotsDir;
    this.ensureDirectories();
  }

  private ensureDirectories(): void {
    if (!fs.existsSync(this.resultsDir)) {
      fs.mkdirSync(this.resultsDir, { recursive: true });
    }
    if (!fs.existsSync(this.screenshotsDir)) {
      fs.mkdirSync(this.screenshotsDir, { recursive: true });
    }
  }

  async generateReport(): Promise<void> {
    console.log('开始生成测试报告...');

    const reportData = await this.collectTestData();
    await this.saveHTMLReport(reportData);
    await this.saveMarkdownReport(reportData);
    await this.saveJSONReport(reportData);

    console.log('测试报告生成完成！');
    console.log(`HTML报告: ${path.join(this.resultsDir, 'e2e-test-report.html')}`);
    console.log(`Markdown报告: ${path.join(this.resultsDir, 'e2e-test-report.md')}`);
    console.log(`JSON报告: ${path.join(this.resultsDir, 'e2e-test-report.json')}`);
  }

  private async collectTestData(): Promise<ReportData> {
    const jsonResults = await this.loadJSONResults();
    const screenshots = this.loadScreenshots();
    const consoleErrors = this.analyzeConsoleErrors();

    return {
      generatedAt: new Date().toISOString(),
      totalTests: jsonResults.length,
      passed: jsonResults.filter(r => r.status === 'passed').length,
      failed: jsonResults.filter(r => r.status === 'failed').length,
      skipped: jsonResults.filter(r => r.status === 'skipped').length,
      duration: jsonResults.reduce((sum, r) => sum + (r.duration || 0), 0),
      suites: this.groupTestsBySuite(jsonResults),
      consoleErrors,
      screenshots,
      recommendations: this.generateRecommendations(jsonResults, consoleErrors)
    };
  }

  private async loadJSONResults(): Promise<any[]> {
    const jsonPath = path.join(this.resultsDir, 'results.json');
    
    if (!fs.existsSync(jsonPath)) {
      console.log('未找到results.json，使用空数据');
      return [];
    }

    try {
      const data = JSON.parse(fs.readFileSync(jsonPath, 'utf-8'));
      return data.stats?.tests || [];
    } catch (error) {
      console.error('读取results.json失败:', error);
      return [];
    }
  }

  private loadScreenshots(): string[] {
    if (!fs.existsSync(this.screenshotsDir)) {
      return [];
    }

    const files = fs.readdirSync(this.screenshotsDir);
    return files
      .filter(f => f.endsWith('.png'))
      .map(f => path.join(this.screenshotsDir, f))
      .sort()
      .reverse()
      .slice(0, 20);
  }

  private analyzeConsoleErrors(): ConsoleError[] {
    const errorMap = new Map<string, ConsoleError>();
    
    const reportFiles = fs.readdirSync(this.screenshotsDir)
      .filter(f => f.includes('-state.json'));
    
    for (const file of reportFiles) {
      try {
        const content = JSON.parse(
          fs.readFileSync(path.join(this.screenshotsDir, file), 'utf-8')
        );
        
        if (content.consoleErrors) {
          for (const error of content.consoleErrors) {
            const existing = errorMap.get(error);
            if (existing) {
              existing.count++;
              if (!existing.pages.includes(content.url)) {
                existing.pages.push(content.url);
              }
            } else {
              errorMap.set(error, {
                message: error,
                count: 1,
                pages: [content.url]
              });
            }
          }
        }
      } catch (error) {
        console.error(`读取${file}失败:`, error);
      }
    }

    return Array.from(errorMap.values())
      .sort((a, b) => b.count - a.count);
  }

  private groupTestsBySuite(results: any[]): SuiteResult[] {
    const suites = new Map<string, SuiteResult>();
    
    for (const result of results) {
      const suiteName = result.title?.split(' ')[0] || 'Unknown';
      
      if (!suites.has(suiteName)) {
        suites.set(suiteName, {
          name: suiteName,
          tests: [],
          duration: 0
        });
      }
      
      const suite = suites.get(suiteName)!;
      suite.tests.push({
        title: result.title,
        status: result.status,
        duration: result.duration || 0,
        error: result.error
      });
      suite.duration += result.duration || 0;
    }

    return Array.from(suites.values());
  }

  private generateRecommendations(results: any[], errors: ConsoleError[]): string[] {
    const recommendations: string[] = [];

    const failedTests = results.filter(r => r.status === 'failed');
    if (failedTests.length > 0) {
      recommendations.push(`有${failedTests.length}个测试失败，建议检查相关功能`);
    }

    if (errors.length > 0) {
      recommendations.push(`检测到${errors.length}种不同的控制台错误，建议修复`);
    }

    const slowTests = results.filter(r => r.duration > 30000);
    if (slowTests.length > 0) {
      recommendations.push(`有${slowTests.length}个测试执行时间超过30秒，可能需要优化`);
    }

    if (recommendations.length === 0) {
      recommendations.push('所有测试通过，控制台无错误');
    }

    return recommendations;
  }

  private async saveHTMLReport(data: ReportData): Promise<void> {
    const html = this.generateHTML(data);
    const filepath = path.join(this.resultsDir, 'e2e-test-report.html');
    fs.writeFileSync(filepath, html, 'utf-8');
  }

  private generateHTML(data: ReportData): string {
    const passRate = data.totalTests > 0 
      ? ((data.passed / data.totalTests) * 100).toFixed(1) 
      : '0';

    return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>E2E测试报告</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background: #f5f5f5;
      padding: 20px;
    }
    .container {
      max-width: 1200px;
      margin: 0 auto;
    }
    .header {
      background: white;
      padding: 30px;
      border-radius: 8px;
      margin-bottom: 20px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    }
    h1 {
      color: #333;
      margin-bottom: 10px;
    }
    .summary {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 20px;
      margin-bottom: 20px;
    }
    .stat-card {
      background: white;
      padding: 20px;
      border-radius: 8px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    }
    .stat-card h3 {
      color: #666;
      font-size: 14px;
      margin-bottom: 10px;
    }
    .stat-card .value {
      font-size: 32px;
      font-weight: bold;
      color: #333;
    }
    .stat-card.passed .value { color: #10b981; }
    .stat-card.failed .value { color: #ef4444; }
    .stat-card.skipped .value { color: #f59e0b; }
    .section {
      background: white;
      padding: 30px;
      border-radius: 8px;
      margin-bottom: 20px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    }
    .section h2 {
      color: #333;
      margin-bottom: 20px;
      padding-bottom: 10px;
      border-bottom: 2px solid #e5e5e5;
    }
    .test-item {
      padding: 15px;
      border-radius: 6px;
      margin-bottom: 10px;
      background: #f9fafb;
    }
    .test-item.passed { border-left: 4px solid #10b981; }
    .test-item.failed { border-left: 4px solid #ef4444; }
    .test-item.skipped { border-left: 4px solid #f59e0b; }
    .test-title {
      font-weight: 500;
      color: #333;
      margin-bottom: 5px;
    }
    .test-duration {
      font-size: 12px;
      color: #666;
    }
    .test-error {
      margin-top: 10px;
      padding: 10px;
      background: #fee;
      border-radius: 4px;
      color: #c00;
      font-size: 12px;
    }
    .recommendations {
      background: #f0f9ff;
      padding: 20px;
      border-radius: 8px;
    }
    .recommendations ul {
      list-style: none;
      padding: 0;
    }
    .recommendations li {
      padding: 10px 0;
      border-bottom: 1px solid #e5e5e5;
      color: #333;
    }
    .recommendations li:last-child {
      border-bottom: none;
    }
    .screenshot-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
      gap: 15px;
    }
    .screenshot-item {
      border-radius: 8px;
      overflow: hidden;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    }
    .screenshot-item img {
      width: 100%;
      height: auto;
      display: block;
    }
    .error-list {
      max-height: 400px;
      overflow-y: auto;
    }
    .error-item {
      padding: 15px;
      background: #fef2f2;
      border-radius: 6px;
      margin-bottom: 10px;
    }
    .error-message {
      font-weight: 500;
      color: #991b1b;
      margin-bottom: 5px;
    }
    .error-meta {
      font-size: 12px;
      color: #666;
    }
    .footer {
      text-align: center;
      padding: 20px;
      color: #666;
      font-size: 12px;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>E2E自动化测试报告</h1>
      <p>生成时间: ${new Date(data.generatedAt).toLocaleString('zh-CN')}</p>
    </div>

    <div class="summary">
      <div class="stat-card">
        <h3>总测试数</h3>
        <div class="value">${data.totalTests}</div>
      </div>
      <div class="stat-card passed">
        <h3>通过</h3>
        <div class="value">${data.passed}</div>
      </div>
      <div class="stat-card failed">
        <h3>失败</h3>
        <div class="value">${data.failed}</div>
      </div>
      <div class="stat-card skipped">
        <h3>跳过</h3>
        <div class="value">${data.skipped}</div>
      </div>
      <div class="stat-card">
        <h3>通过率</h3>
        <div class="value">${passRate}%</div>
      </div>
      <div class="stat-card">
        <h3>总耗时</h3>
        <div class="value">${(data.duration / 1000).toFixed(1)}s</div>
      </div>
    </div>

    <div class="section">
      <h2>测试建议</h2>
      <div class="recommendations">
        <ul>
          ${data.recommendations.map(r => `<li>${r}</li>`).join('')}
        </ul>
      </div>
    </div>

    ${data.consoleErrors.length > 0 ? `
    <div class="section">
      <h2>控制台错误 (${data.consoleErrors.length})</h2>
      <div class="error-list">
        ${data.consoleErrors.slice(0, 10).map(err => `
          <div class="error-item">
            <div class="error-message">${err.message}</div>
            <div class="error-meta">
              出现次数: ${err.count} | 影响页面: ${err.pages.length}
            </div>
          </div>
        `).join('')}
      </div>
    </div>
    ` : ''}

    <div class="section">
      <h2>测试截图 (最近20张)</h2>
      <div class="screenshot-grid">
        ${data.screenshots.map(s => `
          <div class="screenshot-item">
            <img src="${path.relative(this.resultsDir, s)}" alt="screenshot">
            <div style="padding: 10px; font-size: 12px; color: #666;">
              ${path.basename(s)}
            </div>
          </div>
        `).join('')}
      </div>
    </div>

    <div class="footer">
      <p>本报告由自动化测试系统生成 | E2E Test Report</p>
    </div>
  </div>
</body>
</html>`;
  }

  private async saveMarkdownReport(data: ReportData): Promise<void> {
    const markdown = this.generateMarkdown(data);
    const filepath = path.join(this.resultsDir, 'e2e-test-report.md');
    fs.writeFileSync(filepath, markdown, 'utf-8');
  }

  private generateMarkdown(data: ReportData): string {
    return `# E2E自动化测试报告

## 测试概览

- **生成时间**: ${new Date(data.generatedAt).toLocaleString('zh-CN')}
- **总测试数**: ${data.totalTests}
- **通过**: ${data.passed} ✅
- **失败**: ${data.failed} ❌
- **跳过**: ${data.skipped} ⏭️
- **通过率**: ${data.totalTests > 0 ? ((data.passed / data.totalTests) * 100).toFixed(1) : 0}%
- **总耗时**: ${(data.duration / 1000).toFixed(1)}秒

## 测试建议

${data.recommendations.map((r, i) => `${i + 1}. ${r}`).join('\n')}

${data.consoleErrors.length > 0 ? `

## 控制台错误分析

发现 ${data.consoleErrors.length} 种不同的控制台错误：

${data.consoleErrors.slice(0, 10).map(err => 
  `- **${err.message}** (出现 ${err.count} 次，影响 ${err.pages.length} 个页面)`
).join('\n')}
` : ''}

## 测试截图

已捕获 ${data.screenshots.length} 张测试截图，保存在 \`test-screenshots/\` 目录。

---

*本报告由自动化测试系统生成*
`;
  }

  private async saveJSONReport(data: ReportData): Promise<void> {
    const filepath = path.join(this.resultsDir, 'e2e-test-report.json');
    fs.writeFileSync(filepath, JSON.stringify(data, null, 2), 'utf-8');
  }
}

if (require.main === module) {
  const generator = new ReportGenerator();
  generator.generateReport().catch(console.error);
}
