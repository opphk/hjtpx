const Sentry = require('@sentry/node');
const { ProfilingIntegration } = require('@sentry/profiling-node');

const monitoringConfig = {
  sentry: {
    dsn: process.env.SENTRY_DSN || '',
    environment: process.env.NODE_ENV || 'development',
    release: process.env.APP_VERSION || '1.0.0',
    tracesSampleRate: parseFloat(process.env.SENTRY_TRACES_SAMPLE_RATE) || 0.1,
    profilesSampleRate: parseFloat(process.env.SENTRY_PROFILES_SAMPLE_RATE) || 0.1,
    enableTracing: process.env.SENTRY_ENABLE_TRACING === 'true',
  },
  errorGroups: {
    database: ['MongoError', 'MongooseError', 'MongoServerError'],
    validation: ['ValidationError', 'JoiError', 'ValidatorError'],
    auth: ['UnauthorizedError', 'JsonWebTokenError', 'TokenExpiredError'],
    network: ['ECONNREFUSED', 'ETIMEDOUT', 'ENOTFOUND'],
    syntax: ['SyntaxError', 'ParseError'],
  },
  alertRules: {
    critical: {
      threshold: 5,
      window: 300,
      severity: 'critical',
    },
    warning: {
      threshold: 20,
      window: 600,
      severity: 'warning',
    },
  },
};

function initializeSentry(app) {
  if (!monitoringConfig.sentry.dsn) {
    console.warn('Sentry DSN not configured, error tracking disabled');
    return null;
  }

  Sentry.init({
    dsn: monitoringConfig.sentry.dsn,
    environment: monitoringConfig.sentry.environment,
    release: monitoringConfig.sentry.release,
    tracesSampleRate: monitoringConfig.sentry.tracesSampleRate,
    profilesSampleRate: monitoringConfig.sentry.profilesSampleRate,
    enableTracing: monitoringConfig.sentry.enableTracing,
    integrations: [
      new Sentry.Integrations.Http({ tracing: true }),
      new Sentry.Integrations.Express({ app }),
      new Sentry.Integrations.Mongo(),
      new Sentry.Integrations.Redis(),
      new ProfilingIntegration(),
    ],
    beforeSend(event, hint) {
      const error = hint?.originalException;
      if (error) {
        const group = categorizeError(error);
        event.tags = { ...event.tags, errorGroup: group };
      }
      return event;
    },
    ignoreErrors: [
      /Non-Error promise rejection/,
      /ResizeObserver/,
      /Warning: /,
    ],
  });

  console.log('✅ Sentry error tracking initialized');
  return Sentry;
}

function categorizeError(error) {
  const errorName = error.name || error.constructor?.name || 'Unknown';
  const errorMessage = error.message || '';

  for (const [group, patterns] of Object.entries(monitoringConfig.errorGroups)) {
    if (patterns.some(pattern => {
      if (pattern instanceof RegExp) {
        return pattern.test(errorName) || pattern.test(errorMessage);
      }
      return errorName === pattern || errorMessage.includes(pattern);
    })) {
      return group;
    }
  }
  return 'general';
}

function captureException(error, context = {}) {
  if (!Sentry.getClient()) {
    console.error('Sentry not initialized:', error);
    return null;
  }

  const scope = new Sentry.Scope();
  if (context.user) {
    scope.setUser({
      id: context.user.id,
      username: context.user.username,
      email: context.user.email,
    });
  }

  if (context.request) {
    scope.setExtra('requestPath', context.request.path);
    scope.setExtra('requestMethod', context.request.method);
    scope.setTag('requestId', context.request.requestId);
  }

  if (context.operation) {
    scope.setTag('operation', context.operation);
  }

  return Sentry.captureException(error, scope);
}

function captureMessage(message, level = 'info', context = {}) {
  if (!Sentry.getClient()) {
    console.log(`[${level.toUpperCase()}] ${message}`);
    return null;
  }

  const scope = new Sentry.Scope();
  if (context.tags) {
    scope.setTags(context.tags);
  }
  if (context.extra) {
    scope.setExtras(context.extra);
  }

  return Sentry.captureMessage(message, level, scope);
}

function startTransaction(name, op = 'custom') {
  if (!Sentry.getClient()) {
    return null;
  }
  return Sentry.startTransaction({ name, op });
}

function addBreadcrumb(message, category, data = {}) {
  if (!Sentry.getClient()) {
    return;
  }
  Sentry.addBreadcrumb({
    message,
    category,
    data,
    timestamp: Date.now(),
  });
}

function setContext(key, value) {
  if (Sentry.getClient()) {
    Sentry.setContext(key, value);
  }
}

function setTag(key, value) {
  if (Sentry.getClient()) {
    Sentry.setTag(key, value);
  }
}

function getErrorStats() {
  if (!Sentry.getClient()) {
    return null;
  }

  const client = Sentry.getClient();
  const transport = client?.getTransport();

  return {
    environment: monitoringConfig.sentry.environment,
    dsn: monitoringConfig.sentry.dsn ? 'configured' : 'not configured',
    sampleRate: monitoringConfig.sentry.tracesSampleRate,
    profilingEnabled: monitoringConfig.sentry.profilesSampleRate > 0,
  };
}

module.exports = {
  initializeSentry,
  captureException,
  captureMessage,
  startTransaction,
  addBreadcrumb,
  setContext,
  setTag,
  categorizeError,
  getErrorStats,
  monitoringConfig,
};
