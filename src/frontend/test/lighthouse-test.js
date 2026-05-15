import { defineConfig } from 'lighthouse';
import { writeFileSync, existsSync, mkdirSync } from 'fs';
import { join } from 'path';

export const PERFORMANCE_THRESHOLDS = {
  performance: 90,
  'first-contentful-paint': 1800,
  'largest-contentful-paint': 2500,
  'cumulative-layout-shift': 0.1,
  'total-blocking-time': 200,
  'speed-index': 1800,
  'interactive': 3000
};

export const MOBILE_PERFORMANCE_THRESHOLDS = {
  performance: 85,
  'first-contentful-paint': 3000,
  'largest-contentful-paint': 4000,
  'cumulative-layout-shift': 0.25,
  'total-blocking-time': 300,
  'speed-index': 3000,
  'interactive': 5000
};

export async function runLighthouseAudit(url, options = {}) {
  const lighthouse = await import('lighthouse');
  
  const config = {
    extends: 'lighthouse:default',
    settings: {
      onlyCategories: ['performance', 'accessibility', 'best-practices', 'seo'],
      output: 'json',
      logLevel: 'info',
      ...options
    }
  };

  const runnerResult = await lighthouse.default(url, {
    port: 9222,
    output: 'json',
    logLevel: 'info',
    config: config
  });

  const report = runnerResult.report;
  const lhr = runnerResult.lhr;

  const resultsDir = join(process.cwd(), 'lighthouse-results');
  if (!existsSync(resultsDir)) {
    mkdirSync(resultsDir, { recursive: true });
  }

  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  const reportPath = join(resultsDir, `lighthouse-${timestamp}.json`);
  writeFileSync(reportPath, JSON.stringify(lhr, null, 2));

  console.log('\n📊 Lighthouse Audit Results\n');
  console.log('═'.repeat(80));
  console.log(`\n🔗 URL: ${url}`);
  console.log(`📅 Timestamp: ${new Date().toISOString()}`);
  console.log(`\n📈 Performance Score: ${(lhr.categories.performance.score * 100).toFixed(0)}/100`);
  
  const thresholds = options.emulatedFormFactor === 'mobile' 
    ? MOBILE_PERFORMANCE_THRESHOLDS 
    : PERFORMANCE_THRESHOLDS;

  const metrics = [
    {
      name: 'First Contentful Paint',
      key: 'first-contentful-paint',
      value: lhr.audits['first-contentful-paint']?.numericValue,
      threshold: thresholds['first-contentful-paint']
    },
    {
      name: 'Largest Contentful Paint',
      key: 'largest-contentful-paint',
      value: lhr.audits['largest-contentful-paint']?.numericValue,
      threshold: thresholds['largest-contentful-paint']
    },
    {
      name: 'Cumulative Layout Shift',
      key: 'cumulative-layout-shift',
      value: lhr.audits['cumulative-layout-shift']?.numericValue,
      threshold: thresholds['cumulative-layout-shift']
    },
    {
      name: 'Total Blocking Time',
      key: 'total-blocking-time',
      value: lhr.audits['total-blocking-time']?.numericValue,
      threshold: thresholds['total-blocking-time']
    },
    {
      name: 'Speed Index',
      key: 'speed-index',
      value: lhr.audits['speed-index']?.numericValue,
      threshold: thresholds['speed-index']
    }
  ];

  console.log('\n🎯 Core Web Vitals:\n');
  metrics.forEach(metric => {
    const value = metric.value ? (metric.value / 1000).toFixed(2) : 'N/A';
    const threshold = (metric.threshold / 1000).toFixed(2);
    const status = metric.value <= metric.threshold ? '✅' : '❌';
    
    console.log(`${status} ${metric.name}`);
    console.log(`   Current: ${value}s | Target: <${threshold}s\n`);
  });

  console.log('\n💡 Recommendations:\n');
  const opportunities = lhr.audits['diagnostics']?.details?.items?.[0] || {};
  
  Object.entries(opportunities).forEach(([key, value]) => {
    if (value && typeof value === 'number' && value > 0) {
      console.log(`   • ${formatMetricName(key)}: ${value}`);
    }
  });

  console.log('\n' + '='.repeat(80) + '\n');

  const passed = metrics.filter(m => m.value <= m.threshold).length;
  const total = metrics.length;
  
  console.log(`\n✅ Passed: ${passed}/${total} metrics`);
  console.log(`📊 Report saved to: ${reportPath}\n`);

  return {
    score: lhr.categories.performance.score * 100,
    metrics: metrics.reduce((acc, m) => {
      acc[m.key] = {
        value: m.value,
        threshold: m.threshold,
        passed: m.value <= m.threshold
      };
      return acc;
    }, {}),
    reportPath,
    lhr
  };
}

function formatMetricName(name) {
  return name
    .replace(/([A-Z])/g, ' $1')
    .replace(/^./, str => str.toUpperCase())
    .replace(/Mainthread Work/g, 'Main Thread Work')
    .replace(/Rebuilds/g, 'Rebuilds')
    .replace(/Style Tasks/g, 'Style Tasks');
}

export function validatePerformanceResults(results, thresholds = PERFORMANCE_THRESHOLDS) {
  const validation = {
    performance: {
      score: results.score,
      threshold: thresholds.performance,
      passed: results.score >= thresholds.performance
    },
    metrics: {}
  };

  Object.entries(results.metrics).forEach(([key, data]) => {
    const threshold = thresholds[key];
    if (threshold !== undefined) {
      validation.metrics[key] = {
        ...data,
        threshold,
        passed: data.passed
      };
    }
  });

  const allPassed = validation.performance.passed && 
                   Object.values(validation.metrics).every(m => m.passed);

  console.log('\n📋 Performance Validation Results\n');
  console.log('═'.repeat(80));
  
  console.log(`\n${validation.performance.passed ? '✅' : '❌'} Performance Score: ${validation.performance.score.toFixed(0)}/100 (Target: ${validation.performance.threshold}+)`);
  
  Object.entries(validation.metrics).forEach(([key, data]) => {
    const icon = data.passed ? '✅' : '❌';
    const value = (data.value / 1000).toFixed(2);
    const threshold = (data.threshold / 1000).toFixed(2);
    console.log(`${icon} ${formatMetricName(key)}: ${value}s (Target: <${threshold}s)`);
  });

  console.log('\n' + '='.repeat(80) + '\n');

  if (allPassed) {
    console.log('✅ All performance thresholds met!\n');
  } else {
    console.log('❌ Some performance thresholds not met. See above for details.\n');
  }

  return allPassed;
}

export default {
  runLighthouseAudit,
  validatePerformanceResults,
  PERFORMANCE_THRESHOLDS,
  MOBILE_PERFORMANCE_THRESHOLDS
};
