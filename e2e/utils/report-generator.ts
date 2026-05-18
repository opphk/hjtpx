import * as fs from 'fs';
import * as path from 'path';

interface TestResult {
  title: string;
  status: 'passed' | 'failed' | 'skipped';
  duration: number;
  error?: string;
  screenshots?: string[];
  consoleErrors?: string[];
  consoleWarnings?: string[];
}

interface SuiteResult {
  name: string;
  tests: TestResult[];
  duration: number;
}

interface ConsoleError {
  message: string;
  count: number;
  pages: string[];
  timestamp: string;
}

interface ScreenshotInfo {
  filename: string;
  path: string;
  timestamp: string;
  testName?: string;
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
  consoleWarnings: ConsoleError[];
  screenshots: ScreenshotInfo[];
  failedScreenshots: string[];
  recommendations: string[];
  environment: {
    browser: string;
    viewport: string;
    baseURL: string;
  };
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
    const { consoleErrors, consoleWarnings } = this.analyzeConsoleLogs();
    const failedScreenshots = this.findFailedScreenshots();

    return {
      generatedAt: new Date().toISOString(),
      totalTests: jsonResults.length,
      passed: jsonResults.filter(r => r.status === 'passed').length,
      failed: jsonResults.filter(r => r.status === 'failed').length,
      skipped: jsonResults.filter(r => r.status === 'skipped').length,
      duration: jsonResults.reduce((sum, r) => sum + (r.duration || 0), 0),
      suites: this.groupTestsBySuite(jsonResults),
      consoleErrors,
      consoleWarnings,
      screenshots,
      failedScreenshots,
      recommendations: this.generateRecommendations(jsonResults, consoleErrors),
      environment: {
        browser: 'Chromium',
        viewport: '1280x720',
        baseURL: 'http://localhost:8080'
      }
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

  private loadScreenshots(): ScreenshotInfo[] {
    if (!fs.existsSync(this.screenshotsDir)) {
      return [];
    }

    const files = fs.readdirSync(this.screenshotsDir);
    return files
      .filter(f => f.endsWith('.png'))
      .map(f => {
        const filepath = path.join(this.screenshotsDir, f);
        const stats = fs.statSync(filepath);
        return {
          filename: f,
          path: filepath,
          timestamp: stats.mtime.toISOString(),
          testName: this.extractTestName(f)
        };
      })
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, 30);
  }

  private extractTestName(filename: string): string {
    const parts = filename.replace('.png', '').split('-');
    if (parts.length > 2) {
      return parts.slice(2, -5).join('-') || filename;
    }
    return filename;
  }

  private analyzeConsoleLogs(): { consoleErrors: ConsoleError[]; consoleWarnings: ConsoleError[] } {
    const errorMap = new Map<string, ConsoleError>();
    const warningMap = new Map<string, ConsoleError>();

    const reportFiles = fs.readdirSync(this.screenshotsDir)
      .filter(f => f.includes('-state.json'));

    for (const file of reportFiles) {
      try {
        const content = JSON.parse(
          fs.readFileSync(path.join(this.screenshotsDir, file), 'utf-8')
        );

        if (content.consoleErrors) {
          for (const error of content.consoleErrors) {
            const key = this.normalizeErrorMessage(error);
            const existing = errorMap.get(key);
            if (existing) {
              existing.count++;
              if (!existing.pages.includes(content.url)) {
                existing.pages.push(content.url);
              }
            } else {
              errorMap.set(key, {
                message: error,
                count: 1,
                pages: [content.url],
                timestamp: content.timestamp || new Date().toISOString()
              });
            }
          }
        }

        if (content.consoleWarnings) {
          for (const warning of content.consoleWarnings) {
            const key = this.normalizeErrorMessage(warning);
            const existing = warningMap.get(key);
            if (existing) {
              existing.count++;
            } else {
              warningMap.set(key, {
                message: warning,
                count: 1,
                pages: [content.url],
                timestamp: content.timestamp || new Date().toISOString()
              });
            }
          }
        }
      } catch (error) {
        console.error(`读取${file}失败:`, error);
      }
    }

    return {
      consoleErrors: Array.from(errorMap.values())
        .sort((a, b) => b.count - a.count),
      consoleWarnings: Array.from(warningMap.values())
        .sort((a, b) => b.count - a.count)
    };
  }

  private normalizeErrorMessage(message: string): string {
    return message
      .replace(/\d+/g, 'X')
      .replace(/localhost:\d+/g, 'localhost')
      .substring(0, 100);
  }

  private findFailedScreenshots(): string[] {
    const failedPatterns = ['failed', 'error', 'exception'];
    const files = fs.readdirSync(this.screenshotsDir)
      .filter(f => f.endsWith('.png'));

    return files
      .filter(f => failedPatterns.some(pattern => f.toLowerCase().includes(pattern)))
      .map(f => path.join(this.screenshotsDir, f));
  }

  private groupTestsBySuite(results: any[]): SuiteResult[] {
    const suites = new Map<string, SuiteResult>();

    for (const result of results) {
      const suiteName = this.extractSuiteName(result.title);

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

  private extractSuiteName(title: string): string {
    if (!title) return 'Unknown';
    const parts = title.split(' ');
    return parts[0] || 'Unknown';
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

    const criticalErrors = errors.filter(e => e.count > 5);
    if (criticalErrors.length > 0) {
      recommendations.push(`发现${criticalErrors.length}个高频错误，需要优先处理`);
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
  <title>E2E自动化测试报告</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      min-height: 100vh;
      padding: 20px;
    }
    .container {
      max-width: 1400px;
      margin: 0 auto;
    }
    .header {
      background: white;
      padding: 40px;
      border-radius: 16px;
      margin-bottom: 30px;
      box-shadow: 0 10px 40px rgba(0,0,0,0.1);
    }
    h1 {
      color: #1a202c;
      margin-bottom: 15px;
      font-size: 32px;
    }
    .header-meta {
      color: #718096;
      font-size: 14px;
    }
    .summary {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 20px;
      margin-bottom: 30px;
    }
    .stat-card {
      background: white;
      padding: 25px;
      border-radius: 16px;
      box-shadow: 0 4px 20px rgba(0,0,0,0.08);
      transition: transform 0.2s;
    }
    .stat-card:hover {
      transform: translateY(-5px);
    }
    .stat-card h3 {
      color: #718096;
      font-size: 13px;
      margin-bottom: 12px;
      text-transform: uppercase;
      letter-spacing: 1px;
    }
    .stat-card .value {
      font-size: 36px;
      font-weight: bold;
      color: #1a202c;
    }
    .stat-card.passed .value { color: #48bb78; }
    .stat-card.failed .value { color: #f56565; }
    .stat-card.skipped .value { color: #ed8936; }
    .stat-card.rate .value { color: #4299e1; }
    .section {
      background: white;
      padding: 35px;
      border-radius: 16px;
      margin-bottom: 30px;
      box-shadow: 0 4px 20px rgba(0,0,0,0.08);
    }
    .section h2 {
      color: #1a202c;
      margin-bottom: 25px;
      padding-bottom: 15px;
      border-bottom: 3px solid #e2e8f0;
      font-size: 22px;
    }
    .test-item {
      padding: 20px;
      border-radius: 10px;
      margin-bottom: 12px;
      background: #f7fafc;
      border-left: 5px solid #cbd5e0;
      transition: all 0.2s;
    }
    .test-item:hover {
      background: #edf2f7;
    }
    .test-item.passed { border-left-color: #48bb78; }
    .test-item.failed { border-left-color: #f56565; background: #fff5f5; }
    .test-item.skipped { border-left-color: #ed8936; }
    .test-title {
      font-weight: 600;
      color: #2d3748;
      margin-bottom: 8px;
      font-size: 15px;
    }
    .test-duration {
      font-size: 12px;
      color: #718096;
    }
    .test-error {
      margin-top: 12px;
      padding: 12px;
      background: #fed7d7;
      border-radius: 6px;
      color: #c53030;
      font-size: 13px;
      font-family: 'Monaco', 'Menlo', monospace;
    }
    .recommendations {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      padding: 30px;
      border-radius: 16px;
      color: white;
    }
    .recommendations h2 {
      color: white !important;
      border-bottom-color: rgba(255,255,255,0.2) !important;
    }
    .recommendations ul {
      list-style: none;
      padding: 0;
    }
    .recommendations li {
      padding: 15px 0;
      border-bottom: 1px solid rgba(255,255,255,0.2);
      font-size: 15px;
    }
    .recommendations li:last-child { border-bottom: none; }
    .screenshot-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
      gap: 20px;
    }
    .screenshot-item {
      border-radius: 12px;
      overflow: hidden;
      box-shadow: 0 4px 15px rgba(0,0,0,0.1);
      transition: transform 0.2s;
    }
    .screenshot-item:hover {
      transform: scale(1.02);
    }
    .screenshot-item img {
      width: 100%;
      height: 200px;
      object-fit: cover;
      display: block;
    }
    .screenshot-info {
      padding: 15px;
      background: #f7fafc;
    }
    .screenshot-name {
      font-weight: 500;
      color: #2d3748;
      font-size: 13px;
      word-break: break-all;
    }
    .screenshot-time {
      font-size: 11px;
      color: #718096;
      margin-top: 5px;
    }
    .error-list {
      max-height: 450px;
      overflow-y: auto;
    }
    .error-item {
      padding: 18px;
      background: #fff5f5;
      border-radius: 10px;
      margin-bottom: 12px;
      border-left: 4px solid #f56565;
    }
    .error-message {
      font-weight: 600;
      color: #c53030;
      margin-bottom: 8px;
      font-size: 14px;
    }
    .error-meta {
      font-size: 12px;
      color: #718096;
      display: flex;
      gap: 20px;
    }
    .warning-item {
      background: #fffaf0;
      border-left-color: #ed8936;
    }
    .warning-message { color: #c05621; }
    .environment {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 15px;
    }
    .env-item {
      background: #edf2f7;
      padding: 15px;
      border-radius: 8px;
    }
    .env-label {
      font-size: 11px;
      color: #718096;
      text-transform: uppercase;
      margin-bottom: 5px;
    }
    .env-value {
      font-size: 16px;
      font-weight: 600;
      color: #2d3748;
    }
    .footer {
      text-align: center;
      padding: 30px;
      color: rgba(255,255,255,0.8);
      font-size: 13px;
    }
    .tabs {
      display: flex;
      gap: 10px;
      margin-bottom: 20px;
    }
    .tab {
      padding: 10px 20px;
      background: #e2e8f0;
      border-radius: 8px;
      cursor: pointer;
      transition: all 0.2s;
    }
    .tab.active {
      background: #4299e1;
      color: white;
    }
    .tab-content {
      display: none;
    }
    .tab-content.active {
      display: block;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>E2E自动化测试报告</h1>
      <p class="header-meta">
        生成时间: ${new Date(data.generatedAt).toLocaleString('zh-CN')} |
        测试环境: ${data.environment.baseURL}
      </p>
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
      <div class="stat-card rate">
        <h3>通过率</h3>
        <div class="value">${passRate}%</div>
      </div>
      <div class="stat-card">
        <h3>总耗时</h3>
        <div class="value">${(data.duration / 1000).toFixed(1)}s</div>
      </div>
    </div>

    <div class="section">
      <h2>测试环境</h2>
      <div class="environment">
        <div class="env-item">
          <div class="env-label">浏览器</div>
          <div class="env-value">${data.environment.browser}</div>
        </div>
        <div class="env-item">
          <div class="env-label">视口大小</div>
          <div class="env-value">${data.environment.viewport}</div>
        </div>
        <div class="env-item">
          <div class="env-label">基础URL</div>
          <div class="env-value">${data.environment.baseURL}</div>
        </div>
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
        ${data.consoleErrors.slice(0, 15).map(err => `
          <div class="error-item">
            <div class="error-message">${this.escapeHtml(err.message)}</div>
            <div class="error-meta">
              <span>出现次数: ${err.count}</span>
              <span>影响页面: ${err.pages.length}个</span>
            </div>
          </div>
        `).join('')}
      </div>
    </div>
    ` : ''}

    ${data.consoleWarnings.length > 0 ? `
    <div class="section">
      <h2>控制台警告 (${data.consoleWarnings.length})</h2>
      <div class="error-list">
        ${data.consoleWarnings.slice(0, 10).map(warn => `
          <div class="error-item warning-item">
            <div class="error-message warning-message">${this.escapeHtml(warn.message)}</div>
            <div class="error-meta">
              <span>出现次数: ${warn.count}</span>
            </div>
          </div>
        `).join('')}
      </div>
    </div>
    ` : ''}

    <div class="section">
      <h2>测试截图 (最近30张)</h2>
      <div class="screenshot-grid">
        ${data.screenshots.length > 0 ? data.screenshots.map(s => `
          <div class="screenshot-item">
            <img src="${path.relative(this.resultsDir, s.path)}" alt="${s.filename}"
                 onerror="this.src='data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 width=%22200%22 height=%22150%22><rect fill=%22%23f0f0f0%22 width=%22200%22 height=%22150%22/><text x=%2250%25%22 y=%2250%25%22 text-anchor=%22middle%22 dy=%22.3em%22 fill=%22%23999%22>截图加载失败</text></svg>'">
            <div class="screenshot-info">
              <div class="screenshot-name">${s.filename}</div>
              <div class="screenshot-time">${new Date(s.timestamp).toLocaleString('zh-CN')}</div>
            </div>
          </div>
        `).join('') : '<p style="color:#718096;padding:20px;">暂无截图</p>'}
      </div>
    </div>

    ${data.failedScreenshots.length > 0 ? `
    <div class="section">
      <h2>失败截图 (${data.failedScreenshots.length})</h2>
      <div class="screenshot-grid">
        ${data.failedScreenshots.map(s => `
          <div class="screenshot-item">
            <img src="${path.relative(this.resultsDir, s)}" alt="failed screenshot">
            <div class="screenshot-info">
              <div class="screenshot-name">${path.basename(s)}</div>
            </div>
          </div>
        `).join('')}
      </div>
    </div>
    ` : ''}

    <div class="section">
      <h2>测试套件详情</h2>
      ${data.suites.map(suite => `
        <div style="margin-bottom: 25px;">
          <h3 style="color: #2d3748; margin-bottom: 15px; font-size: 18px;">
            ${suite.name} (${suite.tests.length}个测试, ${(suite.duration / 1000).toFixed(1)}s)
          </h3>
          ${suite.tests.map(test => `
            <div class="test-item ${test.status}">
              <div class="test-title">${test.title}</div>
              <div class="test-duration">状态: ${test.status} | 耗时: ${(test.duration / 1000).toFixed(2)}s</div>
              ${test.error ? `<div class="test-error">错误: ${this.escapeHtml(test.error)}</div>` : ''}
            </div>
          `).join('')}
        </div>
      `).join('')}
    </div>

    <div class="footer">
      <p>本报告由自动化测试系统生成 | E2E Test Report</p>
      <p style="margin-top: 10px;">hjtpx captcha system</p>
    </div>
  </div>
</body>
</html>`;
  }

  private escapeHtml(text: string): string {
    if (!text) return '';
    return text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
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
- **测试环境**: ${data.environment.baseURL}
- **浏览器**: ${data.environment.browser}
- **视口**: ${data.environment.viewport}

### 测试统计

| 指标 | 数值 |
|------|------|
| 总测试数 | ${data.totalTests} |
| 通过 | ${data.passed} ✅ |
| 失败 | ${data.failed} ❌ |
| 跳过 | ${data.skipped} ⏭️ |
| 通过率 | ${data.totalTests > 0 ? ((data.passed / data.totalTests) * 100).toFixed(1) : 0}% |
| 总耗时 | ${(data.duration / 1000).toFixed(1)}秒 |

## 测试建议

${data.recommendations.map((r, i) => `${i + 1}. ${r}`).join('\n')}

${data.consoleErrors.length > 0 ? `

## 控制台错误分析

发现 ${data.consoleErrors.length} 种不同的控制台错误：

${data.consoleErrors.slice(0, 15).map(err =>
  `- **${err.message}** (出现 ${err.count} 次，影响 ${err.pages.length} 个页面)`
).join('\n')}
` : ''}

${data.consoleWarnings.length > 0 ? `

## 控制台警告分析

发现 ${data.consoleWarnings.length} 种不同的控制台警告：

${data.consoleWarnings.slice(0, 10).map(warn =>
  `- **${warn.message}** (出现 ${warn.count} 次)`
).join('\n')}
` : ''}

## 截图统计

- 已捕获 ${data.screenshots.length} 张测试截图
- 失败相关截图 ${data.failedScreenshots.length} 张

截图保存在 \`${this.screenshotsDir}/\` 目录。

## 测试套件

${data.suites.map(suite => `### ${suite.name}

- 测试数量: ${suite.tests.length}
- 总耗时: ${(suite.duration / 1000).toFixed(1)}秒

${suite.tests.map(test => `- **${test.title}**: ${test.status} (${(test.duration / 1000).toFixed(2)}s)`).join('\n')}
`).join('\n')}

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
