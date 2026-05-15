const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const RELEASE = process.env.SENTRY_RELEASE;
const DSN = process.env.SENTRY_DSN;

if (!RELEASE) {
  console.log('⚠️  SENTRY_RELEASE 未设置，跳过 sourcemaps 上传');
  process.exit(0);
}

if (!DSN) {
  console.log('⚠️  SENTRY_DSN 未设置，跳过 sourcemaps 上传');
  process.exit(0);
}

console.log(`📤 开始上传 sourcemaps for release: ${RELEASE}`);

try {
  const sourceMapDir = path.join(__dirname, '../../.sentry-sourcemaps');
  
  if (!fs.existsSync(sourceMapDir)) {
    fs.mkdirSync(sourceMapDir, { recursive: true });
  }
  
  execSync('find . -name "*.js.map" -type f | head -100', {
    stdio: 'inherit',
    shell: true,
  });
  
  console.log('✅ Sourcemaps 扫描完成');
  
  if (process.env.CI) {
    console.log('📤 在 CI 环境中跳过实际上传（使用 GitHub Actions）');
  }
  
} catch (error) {
  console.error('❌ Sourcemaps 上传失败:', error.message);
  process.exit(1);
}
