#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');

const args = process.argv.slice(2);
const testType = args[0] || 'all';

function runCommand(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      stdio: 'inherit',
      shell: true,
      ...options
    });

    child.on('close', code => {
      if (code === 0) {
        resolve(code);
      } else {
        reject(new Error(`Command failed with exit code ${code}`));
      }
    });

    child.on('error', reject);
  });
}

async function runTests() {
  console.log('🧪 Starting test execution...\n');

  try {
    switch (testType) {
      case 'unit':
        console.log('📦 Running unit tests...');
        await runCommand('npm', ['run', 'test:unit', '--', '--coverage']);
        break;

      case 'integration':
        console.log('🔗 Running integration tests...');
        await runCommand('npm', ['run', 'test:integration', '--', '--coverage']);
        break;

      case 'e2e':
        console.log('🎭 Running E2E tests...');
        await runCommand('npm', ['run', 'test:e2e']);
        break;

      case 'e2e:headed':
        console.log('🎭 Running E2E tests (headed mode)...');
        await runCommand('npm', ['run', 'test:e2e:headed']);
        break;

      case 'api':
        console.log('🔗 Running API integration tests...');
        await runCommand('npm', ['run', 'test:integration', '--', '--testPathPattern=integration']);
        break;

      case 'auth':
        console.log('🔐 Running auth integration tests...');
        await runCommand('npm', ['run', 'test:integration', '--', '--testPathPattern=auth']);
        break;

      case 'files':
        console.log('📁 Running files integration tests...');
        await runCommand('npm', ['run', 'test:integration', '--', '--testPathPattern=files']);
        break;

      case 'notifications':
        console.log('🔔 Running notifications integration tests...');
        await runCommand('npm', ['run', 'test:integration', '--', '--testPathPattern=notifications']);
        break;

      case 'users':
        console.log('👥 Running users integration tests...');
        await runCommand('npm', ['run', 'test:integration', '--', '--testPathPattern=users']);
        break;

      case 'all':
        console.log('📦 Running unit tests...');
        await runCommand('npm', ['run', 'test:unit', '--', '--coverage']);
        console.log('\n🔗 Running integration tests...');
        await runCommand('npm', ['run', 'test:integration', '--', '--coverage']);
        console.log('\n🎭 Running E2E tests...');
        await runCommand('npm', ['run', 'test:e2e']);
        break;

      case 'ci':
        console.log('🔄 Running CI test suite...');
        await runCommand('npm', ['run', 'test:ci']);
        break;

      default:
        console.error(`Unknown test type: ${testType}`);
        console.log('\nAvailable test types:');
        console.log('  unit          - Run unit tests');
        console.log('  integration   - Run integration tests');
        console.log('  e2e           - Run E2E tests');
        console.log('  e2e:headed    - Run E2E tests in headed mode');
        console.log('  api           - Run API integration tests');
        console.log('  auth          - Run auth integration tests');
        console.log('  files         - Run files integration tests');
        console.log('  notifications - Run notifications integration tests');
        console.log('  users         - Run users integration tests');
        console.log('  all           - Run all tests');
        console.log('  ci            - Run CI test suite');
        process.exit(1);
    }

    console.log('\n✅ Tests completed successfully!');
  } catch (error) {
    console.error('\n❌ Tests failed:', error.message);
    process.exit(1);
  }
}

runTests();
