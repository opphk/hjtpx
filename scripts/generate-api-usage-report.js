const { ApiUsageTracker, getApiUsageTracker } = require('../src/backend/middleware/apiUsage');
const fs = require('fs');
const path = require('path');

const args = process.argv.slice(2);
const command = args[0];

const outputDir = './docs/api-usage';

function printHelp() {
  console.log(`
API Usage Report Generator
===========================

Usage: node scripts/generate-api-usage-report.js [command] [options]

Commands:
  generate              Generate a usage report (JSON format)
  markdown              Generate a markdown report
  top-endpoints         Show top endpoints by usage
  slow-endpoints        Show slow endpoints
  stats                 Show overall statistics
  export [format]       Export stats (json, markdown)
  clear                 Clear all usage statistics

Options:
  --output <dir>        Output directory (default: ./docs/api-usage)
  --limit <n>           Limit number of results (default: 10)
  --format <type>       Output format: console, json, markdown (default: console)

Examples:
  node scripts/generate-api-usage-report.js stats
  node scripts/generate-api-usage-report.js top-endpoints --limit 20
  node scripts/generate-api-usage-report.js markdown
  node scripts/generate-api-usage-report.js export markdown
  `);
}

async function showStats() {
  const tracker = getApiUsageTracker();
  const stats = tracker.getOverallStats();
  
  console.log('📊 API Usage Statistics\n');
  console.log(`Total Requests: ${stats.summary.totalRequests.toLocaleString()}`);
  console.log(`Total Errors: ${stats.summary.totalErrors.toLocaleString()}`);
  console.log(`Error Rate: ${stats.summary.errorRate}`);
  console.log(`Average Response Time: ${stats.summary.averageResponseTime}`);
  console.log(`Unique Endpoints: ${stats.summary.uniqueEndpoints}`);
  console.log(`Period Start: ${stats.summary.startTime}`);
  console.log(`Last Updated: ${stats.summary.lastUpdated}`);
  
  console.log('\n📱 Request Methods:');
  Object.entries(stats.methods).forEach(([method, data]) => {
    console.log(`  ${method}: ${data.count.toLocaleString()} (${data.errors} errors)`);
  });
  
  console.log('\n📈 Status Codes:');
  Object.entries(stats.statusCodes)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 10)
    .forEach(([code, count]) => {
      console.log(`  ${code}: ${count.toLocaleString()}`);
    });
}

async function showTopEndpoints(limit = 10) {
  const tracker = getApiUsageTracker();
  const endpoints = tracker.getTopEndpoints(limit);
  
  console.log(`🏆 Top ${limit} Endpoints by Usage\n`);
  console.log('| Endpoint | Method | Calls | Errors | Error Rate | Avg Response Time |');
  console.log('|----------|--------|-------|--------|------------|-------------------|');
  
  endpoints.forEach(ep => {
    console.log(`| ${ep.path} | ${ep.method} | ${ep.count.toLocaleString()} | ${ep.errors} | ${ep.errorRate} | ${ep.avgResponseTime}ms |`);
  });
}

async function showSlowEndpoints(limit = 10) {
  const tracker = getApiUsageTracker();
  const endpoints = tracker.getSlowEndpoints(limit);
  
  if (endpoints.length === 0) {
    console.log('✅ No slow endpoints detected (>1000ms average)');
    return;
  }
  
  console.log(`🐌 Top ${limit} Slow Endpoints (>1000ms)\n`);
  console.log('| Endpoint | Method | Calls | Avg Response Time |');
  console.log('|----------|--------|-------|-------------------|');
  
  endpoints.forEach(ep => {
    console.log(`| ${ep.path} | ${ep.method} | ${ep.count.toLocaleString()} | ${ep.avgResponseTime}ms |`);
  });
}

async function generateReport(format = 'json') {
  const tracker = getApiUsageTracker();
  const report = tracker.generateUsageReport();
  
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }
  
  if (format === 'json') {
    const stats = tracker.getOverallStats();
    const filename = path.join(outputDir, `usage-report-${Date.now()}.json`);
    fs.writeFileSync(filename, JSON.stringify(stats, null, 2));
    console.log(`✅ Report saved to: ${filename}`);
  } else if (format === 'markdown') {
    const filename = path.join(outputDir, `usage-report-${Date.now()}.md`);
    fs.writeFileSync(filename, report);
    console.log(`✅ Report saved to: ${filename}`);
  } else {
    console.log(report);
  }
}

async function exportStats(format = 'json') {
  const tracker = getApiUsageTracker();
  const output = tracker.exportStats(format);
  
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }
  
  const extensions = { json: 'json', markdown: 'md' };
  const filename = path.join(outputDir, `api-stats-${Date.now()}.${extensions[format] || 'json'}`);
  fs.writeFileSync(filename, output);
  console.log(`✅ Stats exported to: ${filename}`);
}

async function clearStats() {
  const tracker = getApiUsageTracker();
  const confirmed = args[1] === '--force' || 
    await new Promise(resolve => {
      const readline = require('readline');
      const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
      rl.question('⚠️  Are you sure you want to clear all usage statistics? (y/N) ', answer => {
        rl.close();
        resolve(answer.toLowerCase() === 'y');
      });
    });
  
  if (confirmed) {
    tracker.clearStats();
    
    const dirs = [outputDir, path.join(outputDir, 'daily'), path.join(outputDir, 'hourly')];
    dirs.forEach(dir => {
      if (fs.existsSync(dir)) {
        fs.readdirSync(dir).forEach(file => {
          fs.unlinkSync(path.join(dir, file));
        });
      }
    });
    
    console.log('✅ All usage statistics cleared');
  } else {
    console.log('Cancelled.');
  }
}

if (!command || command === 'help' || command === '--help' || command === '-h') {
  printHelp();
  process.exit(0);
}

switch (command) {
  case 'stats':
    showStats();
    break;
  case 'top-endpoints':
    const limit = parseInt(args.find(a => a.startsWith('--limit='))?.split('=')[1]) || 10;
    showTopEndpoints(limit);
    break;
  case 'slow-endpoints':
    const slowLimit = parseInt(args.find(a => a.startsWith('--limit='))?.split('=')[1]) || 10;
    showSlowEndpoints(slowLimit);
    break;
  case 'generate':
    generateReport('json');
    break;
  case 'markdown':
    generateReport('markdown');
    break;
  case 'export':
    exportStats(args[1] || 'json');
    break;
  case 'clear':
    clearStats();
    break;
  default:
    printHelp();
}

module.exports = {
  showStats,
  showTopEndpoints,
  showSlowEndpoints,
  generateReport,
  exportStats,
  clearStats
};
