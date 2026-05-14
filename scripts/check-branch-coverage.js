#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const COVERAGE_REQUIREMENTS_FILE = '.coverage-requirements.json';

function loadCoverageRequirements() {
  try {
    const data = fs.readFileSync(COVERAGE_REQUIREMENTS_FILE, 'utf8');
    return JSON.parse(data);
  } catch (error) {
    console.error('Error loading coverage requirements:', error.message);
    return null;
  }
}

function parseBranchType(branchName) {
  const branchPatterns = [
    { pattern: /^main$/, type: 'main' },
    { pattern: /^develop$/, type: 'develop' },
    { pattern: /^feature\/.*/, type: 'feature/*' },
    { pattern: /^release\/.*/, type: 'release/*' },
    { pattern: /^hotfix\/.*/, type: 'hotfix/*' },
    { pattern: /^bugfix\/.*/, type: 'feature/*' },
    { pattern: /^refactor\/.*/, type: 'feature/*' }
  ];

  for (const { pattern, type } of branchPatterns) {
    if (pattern.test(branchName)) {
      return type;
    }
  }

  return 'feature/*';
}

function loadCoverage() {
  const lcovFile = path.join('coverage', 'lcov.info');
  
  if (!fs.existsSync(lcovFile)) {
    console.error('Coverage report not found. Run tests with coverage first.');
    return null;
  }

  const lcovContent = fs.readFileSync(lcovFile, 'utf8');
  const lines = lcovContent.split('\n');
  
  const totals = {
    branches: { total: 0, covered: 0 },
    functions: { total: 0, covered: 0 },
    lines: { total: 0, covered: 0 },
    statements: { total: 0, covered: 0 }
  };

  lines.forEach(line => {
    if (line.startsWith('BRF:')) {
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
    statements: calcPercentage(totals.statements.covered, totals.statements.total)
  };
}

function checkBranchCoverage(coverage, requirements, branchName) {
  const branchType = parseBranchType(branchName);
  const requirements_config = requirements.branches[branchType] || requirements.global;
  
  console.log(`\n📋 Branch: ${branchName} (Type: ${branchType})`);
  console.log(`   Description: ${requirements_config.description || 'No description'}`);
  console.log('\n   Coverage Requirements vs Actual:');
  
  const failures = [];
  const metrics = ['branches', 'functions', 'lines', 'statements'];
  
  metrics.forEach(metric => {
    const required = requirements_config.minimum[metric];
    const actual = coverage[metric];
    const passed = actual >= required;
    const symbol = passed ? '✅' : '❌';
    
    console.log(`   ${symbol} ${metric.charAt(0).toUpperCase() + metric.slice(1)}: ${actual}% (required: ${required}%)`);
    
    if (!passed) {
      failures.push(`${metric}: ${actual}% < ${required}%`);
    }
  });

  return { branchType, requirements_config, failures };
}

function main() {
  console.log('🔍 Branch Coverage Requirements Checker\n');
  console.log('='.repeat(80));

  const requirements = loadCoverageRequirements();
  if (!requirements) {
    console.error('Failed to load coverage requirements. Exiting.');
    process.exit(1);
  }

  const coverage = loadCoverage();
  if (!coverage) {
    console.error('Failed to load coverage data. Exiting.');
    process.exit(1);
  }

  const branchName = process.env.GITHUB_REF_NAME || process.env.GIT_BRANCH || 'main';
  
  const result = checkBranchCoverage(coverage, requirements, branchName);

  console.log('\n' + '='.repeat(80));

  if (result.failures.length > 0) {
    console.error('\n❌ Branch coverage requirements NOT MET:');
    result.failures.forEach(failure => {
      console.error(`   - ${failure}`);
    });
    console.error(`\n   Branch type: ${result.branchType}`);
    console.error('   Please improve coverage or update requirements for this branch type.');
    console.error('\n💡 Tip: Run `npm run test:coverage` to see detailed coverage report.');
    process.exit(1);
  } else {
    console.log('\n✅ Branch coverage requirements MET!');
    console.log(`   All metrics meet or exceed the ${result.branchType} requirements.`);
    process.exit(0);
  }
}

main();
