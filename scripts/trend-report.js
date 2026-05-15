const fs = require('fs');
const path = require('path');

class TrendReporter {
  constructor(options = {}) {
    this.historyFile = options.historyFile || path.join(__dirname, '..', 'coverage', 'coverage-history.json');
    this.outputDir = options.outputDir || path.join(__dirname, '..', 'coverage');
  }

  loadHistory() {
    if (!fs.existsSync(this.historyFile)) {
      throw new Error(`History file not found: ${this.historyFile}`);
    }
    return JSON.parse(fs.readFileSync(this.historyFile, 'utf8'));
  }

  generateHTMLReport(history) {
    const data = history.slice(-30);

    let html = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>测试覆盖率趋势报告</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 2rem;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #1e3a8a 0%, #312e81 100%);
            color: white;
            padding: 2rem;
            text-align: center;
        }
        .header h1 {
            font-size: 2rem;
            margin-bottom: 0.5rem;
        }
        .header p {
            opacity: 0.9;
        }
        .content {
            padding: 2rem;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2rem;
        }
        .stat-card {
            background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
            padding: 1.5rem;
            border-radius: 12px;
            text-align: center;
        }
        .stat-value {
            font-size: 2.5rem;
            font-weight: bold;
            color: #1e3a8a;
        }
        .stat-label {
            color: #64748b;
            margin-top: 0.5rem;
        }
        .chart-container {
            margin-bottom: 2rem;
            background: #f8fafc;
            padding: 1.5rem;
            border-radius: 12px;
        }
        .chart-title {
            font-size: 1.25rem;
            font-weight: 600;
            color: #1e293b;
            margin-bottom: 1rem;
        }
        .chart {
            position: relative;
            height: 300px;
        }
        .chart-lines {
            position: relative;
            height: 250px;
            border-left: 2px solid #e2e8f0;
            border-bottom: 2px solid #e2e8f0;
            margin-left: 50px;
        }
        .chart-y-axis {
            position: absolute;
            left: 0;
            top: 0;
            height: 250px;
            width: 50px;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            padding: 5px 0;
            font-size: 12px;
            color: #64748b;
        }
        .chart-x-axis {
            display: flex;
            justify-content: space-between;
            margin-left: 50px;
            margin-top: 5px;
            font-size: 11px;
            color: #64748b;
            overflow-x: auto;
        }
        .chart-svg {
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
        }
        .line-branches { stroke: #ef4444; stroke-width: 2; fill: none; }
        .line-functions { stroke: #f59e0b; stroke-width: 2; fill: none; }
        .line-lines { stroke: #10b981; stroke-width: 2; fill: none; }
        .line-statements { stroke: #3b82f6; stroke-width: 2; fill: none; }
        .dot { r: 4; }
        .dot-branches { fill: #ef4444; }
        .dot-functions { fill: #f59e0b; }
        .dot-lines { fill: #10b981; }
        .dot-statements { fill: #3b82f6; }
        .legend {
            display: flex;
            gap: 1.5rem;
            justify-content: center;
            margin-top: 1rem;
            flex-wrap: wrap;
        }
        .legend-item {
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }
        .legend-color {
            width: 20px;
            height: 3px;
        }
        .table-container {
            overflow-x: auto;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            padding: 0.75rem;
            text-align: left;
            border-bottom: 1px solid #e2e8f0;
        }
        th {
            background: #f8fafc;
            font-weight: 600;
            color: #1e293b;
        }
        tr:hover {
            background: #f8fafc;
        }
        .good { color: #10b981; }
        .warning { color: #f59e0b; }
        .bad { color: #ef4444; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 测试覆盖率趋势</h1>
            <p>共 ${data.length} 次历史记录</p>
        </div>
        <div class="content">`;

    if (data.length > 0) {
      const latest = data[data.length - 1];
      const oldest = data[0];
      
      html += `
            <div class="stats">
                <div class="stat-card">
                    <div class="stat-value ${latest.coverage.lines.pct >= 80 ? 'good' : latest.coverage.lines.pct >= 50 ? 'warning' : 'bad'}">${latest.coverage.lines.pct.toFixed(1)}%</div>
                    <div class="stat-label">行覆盖率</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value ${latest.coverage.branches.pct >= 80 ? 'good' : latest.coverage.branches.pct >= 50 ? 'warning' : 'bad'}">${latest.coverage.branches.pct.toFixed(1)}%</div>
                    <div class="stat-label">分支覆盖率</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value ${latest.coverage.functions.pct >= 80 ? 'good' : latest.coverage.functions.pct >= 50 ? 'warning' : 'bad'}">${latest.coverage.functions.pct.toFixed(1)}%</div>
                    <div class="stat-label">函数覆盖率</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value ${latest.coverage.statements.pct >= 80 ? 'good' : latest.coverage.statements.pct >= 50 ? 'warning' : 'bad'}">${latest.coverage.statements.pct.toFixed(1)}%</div>
                    <div class="stat-label">语句覆盖率</div>
                </div>
            </div>`;

      const chartWidth = Math.max(data.length * 50, 600);
      const chartHeight = 250;
      
      const generatePath = (values) => {
        const points = values.map((v, i) => {
          const x = (i / (data.length - 1 || 0)) * (chartWidth - 20) + 10;
          const y = chartHeight - (v / 100) * (chartHeight - 20) - 10;
          return { x, y, v };
        });
        const path = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`);
        return { path: path.join(' '), points };
      };

      const branches = generatePath(data.map(d => d.coverage.branches.pct));
      const functions = generatePath(data.map(d => d.coverage.functions.pct));
      const lines = generatePath(data.map(d => d.coverage.lines.pct));
      const statements = generatePath(data.map(d => d.coverage.statements.pct));

      html += `
            <div class="chart-container">
                <div class="chart-title">📈 覆盖率趋势</div>
                <div class="chart">
                    <div class="chart-y-axis">
                        <span>100%</span>
                        <span>75%</span>
                        <span>50%</span>
                        <span>25%</span>
                        <span>0%</span>
                    </div>
                    <div class="chart-lines" style="width: ${chartWidth}px;">
                        <svg class="chart-svg" viewBox="0 0 ${chartWidth} ${chartHeight}">
                            <path d="${branches.path}" class="line-branches"/>
                            <path d="${functions.path}" class="line-functions"/>
                            <path d="${lines.path}" class="line-lines"/>
                            <path d="${statements.path}" class="line-statements"/>
                            ${branches.points.map(p => `<circle cx="${p.x}" cy="${p.y}" class="dot dot-branches"/>`).join('')}
                            ${functions.points.map(p => `<circle cx="${p.x}" cy="${p.y}" class="dot dot-functions"/>`).join('')}
                            ${lines.points.map(p => `<circle cx="${p.x}" cy="${p.y}" class="dot dot-lines"/>`).join('')}
                            ${statements.points.map(p => `<circle cx="${p.x}" cy="${p.y}" class="dot dot-statements"/>`).join('')}
                        </svg>
                    </div>
                    <div class="chart-x-axis" style="width: ${chartWidth}px;">
                        ${data.map(d => `<span>${d.commit.substring(0, 7)}</span>`).join('')}
                    </div>
                </div>
                <div class="legend">
                    <div class="legend-item">
                        <div class="legend-color" style="background: #ef4444;"></div>
                        <span>分支</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color" style="background: #f59e0b;"></div>
                        <span>函数</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color" style="background: #10b981;"></div>
                        <span>行</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color" style="background: #3b82f6;"></div>
                        <span>语句</span>
                    </div>
                </div>
            </div>

            <div class="chart-container">
                <div class="chart-title">📋 历史记录</div>
                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th>时间</th>
                                <th>提交</th>
                                <th>分支</th>
                                <th>函数</th>
                                <th>行</th>
                                <th>语句</th>
                            </tr>
                        </thead>
                        <tbody>`;

      for (let i = data.length - 1; i >= 0; i--) {
        const entry = data[i];
        const date = new Date(entry.timestamp);
        html += `
                            <tr>
                                <td>${date.toLocaleString()}</td>
                                <td><code>${entry.commit.substring(0, 8)}</code></td>
                                <td class="${entry.coverage.branches.pct >= 80 ? 'good' : entry.coverage.branches.pct >= 50 ? 'warning' : 'bad'}">${entry.coverage.branches.pct.toFixed(1)}%</td>
                                <td class="${entry.coverage.functions.pct >= 80 ? 'good' : entry.coverage.functions.pct >= 50 ? 'warning' : 'bad'}">${entry.coverage.functions.pct.toFixed(1)}%</td>
                                <td class="${entry.coverage.lines.pct >= 80 ? 'good' : entry.coverage.lines.pct >= 50 ? 'warning' : 'bad'}">${entry.coverage.lines.pct.toFixed(1)}%</td>
                                <td class="${entry.coverage.statements.pct >= 80 ? 'good' : entry.coverage.statements.pct >= 50 ? 'warning' : 'bad'}">${entry.coverage.statements.pct.toFixed(1)}%</td>
                            </tr>`;
      }

      html += `
                        </tbody>
                    </table>
                </div>
            </div>`;
    } else {
      html += `
            <div style="text-align: center; padding: 3rem; color: #64748b;">
                <p style="font-size: 1.25rem;">暂无历史数据</p>
                <p>运行测试后会生成覆盖率历史记录</p>
            </div>`;
    }

    html += `
        </div>
    </div>
</body>
</html>`;

    return html;
  }

  generate() {
    const history = this.loadHistory();
    const html = this.generateHTMLReport(history);

    if (!fs.existsSync(this.outputDir)) {
      fs.mkdirSync(this.outputDir, { recursive: true });
    }

    const outputPath = path.join(this.outputDir, 'trend-report.html');
    fs.writeFileSync(outputPath, html, 'utf8');
    console.log(`✅ 趋势报告已生成: ${outputPath}`);

    return outputPath;
  }
}

module.exports = TrendReporter;

if (require.main === module) {
  const reporter = new TrendReporter();
  try {
    reporter.generate();
  } catch (error) {
    console.error('❌ 生成趋势报告失败:', error.message);
    process.exit(1);
  }
}
