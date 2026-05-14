#!/bin/bash

set -e

echo "=================================================="
echo "Generating Test Coverage Report"
echo "=================================================="
echo ""

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "Project root: $PROJECT_ROOT"
echo ""

echo "Step 1: Running tests with coverage..."
npm run test:coverage

echo ""
echo "Step 2: Checking coverage results..."

COVERAGE_FILE="$PROJECT_ROOT/coverage/coverage-summary.json"

if [ -f "$COVERAGE_FILE" ]; then
  echo "Coverage report generated successfully!"
  echo ""

  echo "Coverage Summary:"
  echo "-----------------"

  BRANCHES=$(cat "$COVERAGE_FILE" | grep -o '"branches":{"pct":[0-9.]*' | grep -o '[0-9.]*$' | head -1)
  FUNCTIONS=$(cat "$COVERAGE_FILE" | grep -o '"fnTotal":[0-9]*' | grep -o '[0-9]*$' | head -1)
  LINES=$(cat "$COVERAGE_FILE" | grep -o '"lines":{"pct":[0-9.]*' | grep -o '[0-9.]*$' | head -1)
  STATEMENTS=$(cat "$COVERAGE_FILE" | grep -o '"statements":{"pct":[0-9.]*' | grep -o '[0-9.]*$' | head -1)

  echo "  Branches: ${BRANCHES:-0}%"
  echo "  Functions: ${FUNCTIONS:-0}%"
  echo "  Lines: ${LINES:-0}%"
  echo "  Statements: ${STATEMENTS:-0}%"
  echo ""

  if command -v node &> /dev/null; then
    echo "Checking coverage thresholds..."
    node -e "
      const coverage = require('./coverage/coverage-summary.json');
      const thresholds = {
        branches: 80,
        functions: 80,
        lines: 80,
        statements: 80
      };

      let allPassed = true;
      const results = {
        branches: coverage.total.branches.pct >= thresholds.branches,
        functions: coverage.total.functions.pct >= thresholds.functions,
        lines: coverage.total.lines.pct >= thresholds.lines,
        statements: coverage.total.statements.pct >= thresholds.statements
      };

      Object.keys(results).forEach(key => {
        const passed = results[key];
        const symbol = passed ? '✓' : '✗';
        console.log('  ' + symbol + ' ' + key + ': ' + coverage.total[key].pct.toFixed(2) + '% (required: ' + thresholds[key] + '%)');
        if (!passed) allPassed = false;
      });

      console.log('');
      if (allPassed) {
        console.log('All coverage thresholds met!');
        process.exit(0);
      } else {
        console.log('Some coverage thresholds not met.');
        process.exit(1);
      }
    "
  fi
else
  echo "Warning: Coverage file not found at $COVERAGE_FILE"
fi

echo ""
echo "Step 3: Generating HTML report location..."
echo "HTML report available at: $PROJECT_ROOT/coverage/lcov-report/index.html"

echo ""
echo "Step 4: Generating Cobertura XML..."
if [ -f "$PROJECT_ROOT/coverage/cobertura-coverage.xml" ]; then
  echo "Cobertura XML report generated at: $PROJECT_ROOT/coverage/cobertura-coverage.xml"
else
  echo "Cobertura XML report not found (may not be available in current Jest configuration)"
fi

echo ""
echo "=================================================="
echo "Coverage Report Generation Complete"
echo "=================================================="
echo ""
echo "Report Locations:"
echo "  - HTML Report: coverage/lcov-report/index.html"
echo "  - LCOV Report: coverage/lcov.info"
echo "  - JSON Summary: coverage/coverage-summary.json"
echo "  - Cobertura XML: coverage/cobertura-coverage.xml"
echo ""
