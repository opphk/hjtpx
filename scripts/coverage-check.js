#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const COVERAGE_HISTORY_FILE = '.coverage_history.json';
const COVERAGE_DIR = 'coverage';
const BRANCHES_THRESHOLD = 50;
const FUNCTIONS_THRESHOLD = 50;
const LINES_THRESHOLD = 50;
const STATEMENTS_THRESHOLD = 50;
const COVERAGE_DROP_THRESHOLD = 5;

function loadCoverageHistory() {
  try {
    if (fs.existsSync(COVERAGE_HISTORY_FILE)) {
      const data = fs.readFileSync(COVERAGE_HISTORY_FILE, 'utf8');
      return JSON.parse(data);
    }
  } catch (error) {
    console.warn('Warning: Could not load coverage history:', error.message);
  }
  return { entries: [] };
}

function saveCoverageHistory(history) {
  try {
    fs.writeFileSync(COVERAGE_HISTORY_FILE, JSON.stringify(history, null, 2));
  } catch (error) {
    console.error('Error saving coverage history:', error.message);
  }
}

function parseCoverageReport(coverageDir) {
  const lcovFile = path.join(coverageDir, 'lcov.info');
  
  if (!fs.existsSync(lcovFile)) {
    console.error('Coverage report not found. Run tests with coverage first.');
    process.exit(1);
  }

  const lcovContent = fs.readFileSync(lcovFile, 'utf8');
  const lines = lcovContent.split('\n');
  
  let totals = {
    branches: { total: 0, covered: 0 },
    functions: { total: 0, covered: 0 },
    lines: { total: 0, covered: 0 },
    statements: { total: 0, covered: 0 }
  };

  let currentFile = null;
  
  lines.forEach(line => {
    if (line.startsWith('SF:')) {
      currentFile = line.substring(3);
    } else if (line.startsWith('BRF:')) {
      totals.branches.total += parseInt(line.substring(4), 10) || 0;
    } else if (line.startsWith('BRH:')) {
      totals.branches.covered += parseInt(line.substring(4), 10) || 0;
    } else if (line.startsWith('FNF:')) {
      totals.functions.total += parseInt(line.substring(4), 10) || 0;
    } else if (line.startsWith('FNH:')) {
      totals.functions.covered += parseInt(line.substring(4), 10) || 0;
    } else if (line.startsWith('LF:')) {
      totals.lines.total += parseInt(line.substring(3), 10) || 0;
    } else if (line.startsWith('LH:')) {
      totals.lines.covered += parseInt(line.substring(3), 10) || 0;
    } else if (line.startsWith('SF:')) {
      totals.statements.total += 0;
    } else if (line.startsWith('DA:')) {
      totals.statements.total += 1;
    }
  });

  const calcPercentage = (covered, total) => {
    if (total === 0) return 100;
    return Math.round((covered / total) * 100 * 100) / 100;
  };

  return {
    branches: calcPercentage(totals.branches.covered, totals.branches.total),
    functions: calcPercentage(totals.functions.covered, totals.functions.total),
    lines: calcPercentage(totals.lines.covered, totals.lines.total),
    statements: calcPercentage(totals.statements.covered, totals.statements.total),
    timestamp: new Date().toISOString()
  };
}

function checkThresholds(coverage) {
  const failures = [];
  
  if (coverage.branches < BRANCHES_THRESHOLD) {
    failures.push(`Branches: ${coverage.branches}% (minimum: ${BRANCHES_THRESHOLD}%)`);
  }
  
  if (coverage.functions < FUNCTIONS_THRESHOLD) {
    failures.push(`Functions: ${coverage.functions}% (minimum: ${FUNCTIONS_THRESHOLD}%)`);
  }
  
  if (coverage.lines < LINES_THRESHOLD) {
    failures.push(`Lines: ${coverage.lines}% (minimum: ${LINES_THRESHOLD}%)`);
  }
  
  if (coverage.statements < STATEMENTS_THRESHOLD) {
    failures.push(`Statements: ${coverage.statements}% (minimum: ${STATEMENTS_THRESHOLD}%)`);
  }
  
  return failures;
}

function checkCoverageDrop(currentCoverage, history) {
  const alerts = [];
  
  if (history.entries.length === 0) {
    console.log('First coverage entry - no baseline comparison available.');
    return alerts;
  }

  const lastEntry = history.entries[history.entries.length - 1];
  
  const checks = [
    { name: 'Branches', current: currentCoverage.branches, previous: lastEntry.branches },
    { name: 'Functions', current: currentCoverage.functions, previous: lastEntry.functions },
    { name: 'Lines', current: currentCoverage.lines, previous: lastEntry.lines },
    { name: 'Statements', current: currentCoverage.statements, previous: lastEntry.statements }
  ];

  checks.forEach(({ name, current, previous }) => {
    const drop = previous - current;
    if (drop > COVERAGE_DROP_THRESHOLD) {
      alerts.push({
        type: 'warning',
        message: `⚠️  Coverage drop detected: ${name} dropped by ${drop.toFixed(2)}% (${previous}% → ${current}%)`
      });
    }
  });

  return alerts;
}

function generateTrendReport(history) {
  if (history.entries.length < 2) {
    console.log('\nTrend Report: Not enough data points for trend analysis.');
    return;
  }

  console.log('\n📈 Coverage Trend Report:');
  console.log('─'.repeat(80));
  
  const metrics = ['branches', 'functions', 'lines', 'statements'];
  const metricNames = {
    branches: 'Branches',
    functions: 'Functions',
    lines: 'Lines',
    statements: 'Statements'
  };

  metrics.forEach(metric => {
    const values = history.entries.map(e => e[metric]);
    const first = values[0];
    const last = values[values.length - 1];
    const trend = last - first;
    const trendSymbol = trend > 0 ? '📈' : trend < 0 ? '📉' : '➡️';
    
    console.log(`${metricNames[metric]}: ${trendSymbol} ${trend >= 0 ? '+' : ''}${trend.toFixed(2)}% (${first}% → ${last}%)`);
  });

  console.log('─'.repeat(80));
}

function main() {
  console.log('🔍 Running Coverage Check...\n');

  const coverage = parseCoverageReport(COVERAGE_DIR);
  const history = loadCoverageHistory();

  console.log('Current Coverage:');
  console.log(`  Branches:    ${coverage.branches}%`);
  console.log(`  Functions:  ${coverage.functions}%`);
  console.log(`  Lines:      ${coverage.lines}%`);
  console.log(`  Statements: ${coverage.statements}%`);
  console.log();

  const thresholdFailures = checkThresholds(coverage);
  const coverageAlerts = checkCoverageDrop(coverage, history);

  if (thresholdFailures.length > 0) {
    console.error('❌ Coverage Threshold Failures:');
    thresholdFailures.forEach(failure => {
      console.error(`  - ${failure}`);
    });
    console.error();
  } else {
    console.log('✅ All coverage thresholds passed.');
  }

  if (coverageAlerts.length > 0) {
    console.warn('⚠️  Coverage Alerts:');
    coverageAlerts.forEach(alert => {
      console.warn(`  ${alert.message}`);
    });
    console.warn();
  }

  history.entries.push({
    ...coverage,
    commit: process.env.GITHUB_SHA || 'local',
    branch: process.env.GITHUB_REF || 'local'
  });
  
  if (history.entries.length > 30) {
    history.entries = history.entries.slice(-30);
  }
  
  saveCoverageHistory(history);

  generateTrendReport(history);

  if (thresholdFailures.length > 0) {
    console.error('\n❌ Coverage check FAILED. Please improve coverage before merging.');
    process.exit(1);
  }

  if (coverageAlerts.length > 0) {
    console.warn('\n⚠️  Coverage check passed but with warnings. Please review coverage trends.');
  } else {
    console.log('\n✅ Coverage check PASSED.');
  }

  process.exit(0);
}

main();
