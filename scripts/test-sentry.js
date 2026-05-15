const Sentry = require('@sentry/node');

function testSentryConnection() {
  if (!process.env.SENTRY_DSN) {
    console.log('⚠️  SENTRY_DSN 未配置，无法测试 Sentry 连接');
    return Promise.resolve(false);
  }

  console.log('🔍 测试 Sentry 连接...');
  console.log(`   DSN: ${process.env.SENTRY_DSN.substring(0, 30)}...`);
  console.log(`   环境: ${process.env.SENTRY_ENVIRONMENT || process.env.NODE_ENV}`);

  return new Promise((resolve) => {
    try {
      Sentry.captureMessage('Sentry connection test', {
        level: Sentry.Severity.Info,
        tags: {
          test: 'connection-test',
          environment: process.env.SENTRY_ENVIRONMENT || process.env.NODE_ENV,
        },
      });

      setTimeout(() => {
        console.log('✅ Sentry 测试消息已发送');
        resolve(true);
      }, 1000);
    } catch (error) {
      console.error('❌ Sentry 连接测试失败:', error.message);
      resolve(false);
    }
  });
}

function testErrorCapture() {
  if (!process.env.SENTRY_DSN) {
    console.log('⚠️  SENTRY_DSN 未配置，跳过错误捕获测试');
    return Promise.resolve(false);
  }

  console.log('🔍 测试错误捕获...');

  try {
    const testError = new Error('Test error for Sentry integration');
    testError.code = 'TEST_ERROR';
    testError.context = {
      testId: Date.now(),
      timestamp: new Date().toISOString(),
    };

    Sentry.captureException(testError, {
      tags: {
        test: 'error-capture-test',
        test_type: 'integration',
      },
      extra: {
        testData: 'This is a test error for validating Sentry integration',
      },
    });

    console.log('✅ 测试错误已捕获');
    return Promise.resolve(true);
  } catch (error) {
    console.error('❌ 错误捕获测试失败:', error.message);
    return Promise.resolve(false);
  }
}

async function runAllTests() {
  console.log('🧪 开始 Sentry 集成测试\n');

  const connectionResult = await testSentryConnection();
  const errorResult = await testErrorCapture();

  console.log('\n📊 测试结果:');
  console.log(`   连接测试: ${connectionResult ? '✅ 通过' : '⏭️  跳过'}`);
  console.log(`   错误捕获测试: ${errorResult ? '✅ 通过' : '⏭️  跳过'}`);

  if (connectionResult || errorResult) {
    console.log('\n✅ 请在 Sentry 控制台中检查是否收到测试消息');
  }

  process.exit(0);
}

if (require.main === module) {
  require('dotenv').config();
  runAllTests();
}

module.exports = {
  testSentryConnection,
  testErrorCapture,
  runAllTests,
};
