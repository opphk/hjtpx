const Sentry = require('@sentry/node');
const { nodeProfilingIntegration } = require('@sentry/profiling-node');

function initSentry(app) {
  if (!process.env.SENTRY_DSN) {
    console.log('⚠️ SENTRY_DSN 未配置，Sentry 将不会启动');
    return;
  }

  Sentry.init({
    dsn: process.env.SENTRY_DSN,
    environment: process.env.SENTRY_ENVIRONMENT || process.env.NODE_ENV || 'development',
    release: process.env.SENTRY_RELEASE || 'hjtpx@1.0.0',

    integrations: [
      new Sentry.Integrations.Http({ tracing: true }),
      new Sentry.Integrations.Express({ app }),
      nodeProfilingIntegration(),
      new Sentry.Integrations.Mongo({ useMongoose: true })
    ],

    tracesSampleRate:
      parseFloat(process.env.SENTRY_TRACES_SAMPLE_RATE) || getDefaultTracesSampleRate(),
    profilesSampleRate:
      parseFloat(process.env.SENTRY_PROFILES_SAMPLE_RATE) || getDefaultProfilesSampleRate(),

    maxBreadcrumbs: 50,
    debug: process.env.SENTRY_DEBUG === 'true',

    normalizeDepth: 6,

    ignoreErrors: [
      'Network Error',
      'net::ERR_',
      'ECONNREFUSED',
      'ETIMEDOUT',
      'Connection refused',
      'timeout',
      'Network request failed',
      'request timeout',
      'Navigation aborted',
      'Non-Error promise rejection captured'
    ],

    denyUrls: [
      /chrome-extension:\/\//i,
      /safari-extension:\/\//i,
      /moz-extension:\/\//i,
      /webkit-masked-url/i
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
          event.level = Sentry.Severity.Info;
        }
      }

      if (error && error.name === 'ValidationError') {
        event.fingerprint = ['validation-error', error.message.split(':')[0]];
        event.level = Sentry.Severity.Warning;
      }

      if (error && error.code === 'EBADCSRFTOKEN') {
        event.fingerprint = ['csrf-token-error'];
        event.tags = event.tags || {};
        event.tags.security = 'csrf';
      }

      if (event.request && event.request.url) {
        const url = new URL(event.request.url, 'http://localhost');
        if (url.pathname.includes('/health') || url.pathname.includes('/ping')) {
          return null;
        }
      }

      return event;
    },

    beforeSendTransaction(event) {
      if (
        event.transaction &&
        (event.transaction.includes('/health') ||
          event.transaction.includes('/ping') ||
          event.transaction.includes('/favicon'))
      ) {
        return null;
      }

      if (event.spans && event.spans.length > 100) {
        event.spans = event.spans.slice(0, 100);
      }

      return event;
    },

    beforeBreadcrumb(breadcrumb) {
      if (breadcrumb.category === 'http' && breadcrumb.data && breadcrumb.data.url) {
        if (breadcrumb.data.url.includes('/health')) {
          return null;
        }
      }
      return breadcrumb;
    },

    defaultIntegrations: true,
    sendDefaultPii: false,

    attachStacktrace: true,
    maxValueLength: 1000,

    sampleRate: 1.0,

    enableMetrics: true,
    enableSpotlight: process.env.SENTRY_SPOTLIGHT === 'true'
  });

  Sentry.setTag('application', 'hjtpx-api');
  Sentry.setTag('application_version', process.env.SENTRY_RELEASE || 'hjtpx@1.0.0');

  if (process.env.NODE_ENV === 'production') {
    Sentry.setTag('deployment', 'production');
  } else if (process.env.NODE_ENV === 'staging') {
    Sentry.setTag('deployment', 'staging');
  }

  console.log('✅ Sentry 已初始化');
  console.log(`   环境: ${process.env.SENTRY_ENVIRONMENT || process.env.NODE_ENV}`);
  console.log(`   发布版本: ${process.env.SENTRY_RELEASE || 'hjtpx@1.0.0'}`);
  console.log(
    `   采样率: ${process.env.SENTRY_TRACES_SAMPLE_RATE || getDefaultTracesSampleRate()}`
  );
}

function getDefaultTracesSampleRate() {
  const env = process.env.NODE_ENV;
  if (env === 'production') return 0.1;
  if (env === 'staging') return 0.3;
  return 1.0;
}

function getDefaultProfilesSampleRate() {
  const env = process.env.NODE_ENV;
  if (env === 'production') return 0.05;
  if (env === 'staging') return 0.1;
  return 0.5;
}

function setupPerformanceMonitoring(app) {
  if (!process.env.SENTRY_DSN) return;

  app.use((req, res, next) => {
    if (req.path.includes('/health') || req.path.includes('/ping')) {
      return next();
    }

    const startTime = Date.now();

    res.on('finish', () => {
      const duration = Date.now() - startTime;

      if (duration > 1000) {
        Sentry.addBreadcrumb({
          category: 'performance',
          message: `Slow request: ${req.method} ${req.path}`,
          level: Sentry.Severity.Warning,
          data: {
            duration,
            method: req.method,
            path: req.path,
            status: res.statusCode
          }
        });
      }
    });

    next();
  });
}

function captureDatabaseError(error, operation, collection) {
  Sentry.withScope(scope => {
    scope.setTag('error_category', 'database');
    scope.setTag('database_operation', operation);
    scope.setTag('database_collection', collection);
    scope.setLevel(Sentry.Severity.Error);
    Sentry.captureException(error);
  });
}

function captureWebSocketError(error, event, socketId) {
  Sentry.withScope(scope => {
    scope.setTag('error_category', 'websocket');
    scope.setTag('websocket_event', event);
    scope.setTag('websocket_socket_id', socketId);
    scope.setLevel(Sentry.Severity.Error);
    Sentry.captureException(error);
  });
}

function captureGraphQLError(error, operationName, variables) {
  Sentry.withScope(scope => {
    scope.setTag('error_category', 'graphql');
    scope.setTag('graphql_operation', operationName);
    if (variables) {
      scope.setExtra('graphql_variables', sanitizeVariables(variables));
    }
    scope.setLevel(Sentry.Severity.Error);
    Sentry.captureException(error);
  });
}

function sanitizeVariables(variables) {
  if (!variables) return {};
  const sanitized = { ...variables };
  const sensitiveKeys = ['password', 'token', 'secret', 'apiKey', 'api_key', 'authorization'];

  for (const key of Object.keys(sanitized)) {
    if (sensitiveKeys.some(sk => key.toLowerCase().includes(sk.toLowerCase()))) {
      sanitized[key] = '[REDACTED]';
    }
  }

  return sanitized;
}

function setUserContext(user) {
  if (!user) {
    Sentry.setUser(null);
    return;
  }

  Sentry.setUser({
    id: user.id || user._id,
    email: user.email,
    username: user.username,
    role: user.role
  });
}

function addPerformanceTag(name, value) {
  Sentry.setTag(`perf_${name}`, value);
}

function recordMetric(name, value, unit = 'none') {
  Sentry.metrics.increment(name, value, { unit });
}

function createTransaction(name, op, callback) {
  return Sentry.startTransaction({ name, op }, callback);
}

module.exports = {
  initSentry,
  setupPerformanceMonitoring,
  captureDatabaseError,
  captureWebSocketError,
  captureGraphQLError,
  setUserContext,
  addPerformanceTag,
  recordMetric,
  createTransaction,
  Sentry
};
