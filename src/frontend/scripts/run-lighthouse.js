import { spawn } from 'child_process';
import { existsSync, mkdirSync, writeFileSync, readFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));

const PERFORMANCE_THRESHOLDS = {
  performance: 90,
  'first-contentful-paint': 1800,
  'largest-contentful-paint': 2500,
  'cumulative-layout-shift': 0.1,
  'total-blocking-time': 200,
  'speed-index': 1800,
  'interactive': 3000
};

async function startDevServer() {
  console.log('\n🚀 Starting development server...\n');
  
  return new Promise((resolve, reject) => {
    const server = spawn('npm', ['run', 'dev'], {
      cwd: process.cwd(),
      stdio: 'pipe',
      shell: true
    });

    let serverReady = false;
    const timeout = setTimeout(() => {
      if (!serverReady) {
        reject(new Error('Server failed to start within 30 seconds'));
      }
    }, 30000);

    server.stdout.on('data', (data) => {
      const output = data.toString();
      console.log(output);
      
      if (output.includes('Local:') || output.includes('localhost:3001')) {
        serverReady = true;
        clearTimeout(timeout);
        setTimeout(() => resolve(server), 2000);
      }
    });

    server.stderr.on('data', (data) => {
      console.error(data.toString());
    });

    server.on('error', reject);
  });
}

async function runLighthouseAudit(url) {
  console.log('\n📊 Running Lighthouse audit...\n');
  
  return new Promise((resolve, reject) => {
    const lighthouse = spawn(
      'npx',
      [
        'lighthouse',
        url,
        '--output=json',
        '--output-path=./lighthouse-results/report.json',
        '--chrome-flags=--headless',
        '--only-categories=performance',
        '--quiet'
      ],
      {
        cwd: process.cwd(),
        stdio: 'pipe',
        shell: true
      }
    );

    let stderr = '';
    
    lighthouse.stderr.on('data', (data) => {
      stderr += data.toString();
    });

    lighthouse.on('close', (code) => {
      if (code === 0) {
        resolve();
      } else {
        console.error('Lighthouse error:', stderr);
        reject(new Error(`Lighthouse exited with code ${code}`));
      }
    });

    lighthouse.on('error', reject);
  });
}

function analyzeResults() {
  const reportPath = join(process.cwd(), 'lighthouse-results', 'report.json');
  
  if (!existsSync(reportPath)) {
    throw new Error('Lighthouse report not found');
  }

  const report = JSON.parse(readFileSync(reportPath, 'utf-8'));
  
  console.log('\n📊 Lighthouse Audit Results\n');
  console.log('═'.repeat(80));
  
  const score = Math.round(report.categories.performance.score * 100);
  const status = score >= PERFORMANCE_THRESHOLDS.performance ? '✅' : '❌';
  
  console.log(`\n${status} Performance Score: ${score}/100`);
  console.log(`   Target: ${PERFORMANCE_THRESHOLDS.performance}+`);
  
  const metrics = [
    {
      name: 'First Contentful Paint',
      key: 'first-contentful-paint',
      threshold: PERFORMANCE_THRESHOLDS['first-contentful-paint'],
      unit: 'ms'
    },
    {
      name: 'Largest Contentful Paint',
      key: 'largest-contentful-paint',
      threshold: PERFORMANCE_THRESHOLDS['largest-contentful-paint'],
      unit: 'ms'
    },
    {
      name: 'Cumulative Layout Shift',
      key: 'cumulative-layout-shift',
      threshold: PERFORMANCE_THRESHOLDS['cumulative-layout-shift'],
      unit: ''
    },
    {
      name: 'Total Blocking Time',
      key: 'total-blocking-time',
      threshold: PERFORMANCE_THRESHOLDS['total-blocking-time'],
      unit: 'ms'
    },
    {
      name: 'Speed Index',
      key: 'speed-index',
      threshold: PERFORMANCE_THRESHOLDS['speed-index'],
      unit: 'ms'
    }
  ];

  console.log('\n🎯 Core Web Vitals:\n');
  
  let allPassed = true;
  
  metrics.forEach(metric => {
    const audit = report.audits[metric.key];
    const value = audit?.numericValue || 0;
    const passed = value <= metric.threshold;
    
    if (!passed) allPassed = false;
    
    const statusIcon = passed ? '✅' : '❌';
    const valueStr = metric.unit === 'ms' ? `${(value / 1000).toFixed(2)}s` : value.toFixed(3);
    const thresholdStr = metric.unit === 'ms' ? `${(metric.threshold / 1000).toFixed(2)}s` : metric.threshold;
    
    console.log(`${statusIcon} ${metric.name}`);
    console.log(`   Current: ${valueStr} | Target: <${thresholdStr}`);
    console.log(`   ${passed ? '✅ Passed' : '❌ Failed'}\n`);
  });

  console.log('\n💡 Opportunities:\n');
  
  const opportunities = report.audits['diagnostics']?.details?.items?.[0] || {};
  
  Object.entries(opportunities).forEach(([key, value]) => {
    if (value && typeof value === 'number' && value > 0) {
      console.log(`   • ${formatMetricName(key)}: ${value}`);
    }
  });

  console.log('\n' + '='.repeat(80) + '\n');
  
  if (allPassed && score >= PERFORMANCE_THRESHOLDS.performance) {
    console.log('✅ All performance thresholds met!\n');
    return true;
  } else {
    console.log('❌ Some performance thresholds not met.\n');
    return false;
  }
}

function formatMetricName(name) {
  return name
    .replace(/([A-Z])/g, ' $1')
    .replace(/^./, str => str.toUpperCase())
    .replace(/Mainthread Work/g, 'Main Thread Work')
    .replace(/Rebuilds/g, 'Rebuilds')
    .replace(/Style Tasks/g, 'Style Tasks')
    .replace(/Long Tasks/g, 'Long Tasks');
}

async function main() {
  let server = null;
  
  try {
    const resultsDir = join(process.cwd(), 'lighthouse-results');
    if (!existsSync(resultsDir)) {
      mkdirSync(resultsDir, { recursive: true });
    }

    server = await startDevServer();
    
    await runLighthouseAudit('http://localhost:3001');
    
    const passed = analyzeResults();
    
    process.exit(passed ? 0 : 1);
  } catch (error) {
    console.error('\n❌ Error:', error.message);
    process.exit(1);
  } finally {
    if (server) {
      console.log('\n🛑 Stopping development server...\n');
      server.kill();
    }
  }
}

main();
