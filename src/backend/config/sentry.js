const Sentry = require('@sentry/node');
const { nodeProfilingIntegration } = require('@sentry/profiling-node');

function initSentry(app) {
  if (!process.env.SENTRY_DSN) {
    console.log('⚠️ SENTRY_DSN 未配置，Sentry 将不会启动');
    return;
  }

  const environment = process.env.SENTRY_ENVIRONMENT || process.env.NODE_ENV || 'development';
  const tracesSampleRate = parseFloat(process.env.SENTRY_TRACES_SAMPLE_RATE);
  const profilesSampleRate = parseFloat(process.env.SENTRY_PROFILES_SAMPLE_RATE);

  Sentry.init({
    dsn: process.env.SENTRY_DSN,
    environment,
    release: process.env.SENTRY_RELEASE || 'hjtpx@1.0.0',

    integrations: [
      new Sentry.Integrations.Http({ tracing: true }),
      new Sentry.Integrations.Express({ app }),
      new Sentry.Integrations.Mongo(),
      new Sentry.Integrations.Postgres(),
      nodeProfilingIntegration()
    ],

    tracesSampleRate: !isNaN(tracesSampleRate)
      ? tracesSampleRate
      : environment === 'production'
        ? 0.1
        : 1.0,
    profilesSampleRate: !isNaN(profilesSampleRate)
      ? profilesSampleRate
      : environment === 'production'
        ? 0.05
        : 1.0,

    maxBreadcrumbs: 50,
    debug: process.env.SENTRY_DEBUG === 'true',

    normalizeDepth: 5,

    sendDefaultPii: false,

    ignoreErrors: [
      'Network Error',
      'Failed to fetch',
      'Network request failed',
      'ECONNREFUSED',
      'ETIMEDOUT',
      'socket hang up'
    ],

    beforeSend(event, hint) {
      const error = hint.originalException;
      if (error && error.message) {
        if (
          error.message.includes('Network Error') ||
          error.message.includes('timeout') ||
          error.message.includes('ECONNREFUSED') ||
          error.message.includes('ETIMEDOUT')
        ) {
          event.tags = event.tags || {};
          event.tags.network_error = 'true';
          event.level = 'warning';
        }
      }
      return event;
    },

    beforeSendTransaction(event) {
      if (!event.transaction) return null;

      const ignoredPaths = ['/health', '/metrics', '/favicon.ico'];
      if (ignoredPaths.some(path => event.transaction.includes(path))) {
        return null;
      }

      if (event.spans && event.spans.length > 100) {
        event.spans = event.spans.slice(-100);
      }

      return event;
    },

    tracePropagationTargets: [
      'localhost',
      new RegExp(
        process.env.API_BASE_URL ? `^${process.env.API_BASE_URL}` : '^http://localhost:3000'
      )
    ],

    environment: environment,
    serverName: process.env.HOSTNAME || require('os').hostname(),

    initialScope: {
      tags: {
        'app.name': 'hjtpx',
        'app.version': '2.0.0',
        'node.version': process.version
      }
    }
  });

  console.log('✅ Sentry APM 已初始化');
  console.log(`   环境: ${environment}`);
  console.log(`   采样率: ${environment === 'production' ? '10%' : '100%'}`);
  console.log(`   性能分析: ${environment === 'production' ? '5%' : '100%'}`);
}

module.exports = {
  initSentry,
  Sentry
};
