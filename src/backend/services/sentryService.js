require('dotenv').config({
  path: process.env.NODE_ENV === 'production' ? '.env.production' :
        process.env.NODE_ENV === 'staging' ? '.env.staging' : '.env'
});

const Sentry = require('@sentry/node');
const { ProfilingIntegration } = require('@sentry/profiling-node');

const SENTRY_DSN = process.env.SENTRY_DSN;
const NODE_ENV = process.env.NODE_ENV || 'development';
const APP_VERSION = process.env.APP_VERSION || '1.0.0';

const isSentryEnabled = SENTRY_DSN && NODE_ENV !== 'test';

if (isSentryEnabled) {
  Sentry.init({
    dsn: SENTRY_DSN,
    
    integrations: [
      new Sentry.Integrations.Http({ tracing: true, breadcrumbs: true }),
      new Sentry.Integrations.Express({ 
        app: null,
        router: null,
      }),
      new Sentry.Integrations.Mongo(),
      new Sentry.Integrations.Postgres(),
      new Sentry.Integrations.Redis(),
      new ProfilingIntegration(),
    ],

    tracesSampleRate: NODE_ENV === 'production' ? 0.1 : 1.0,
    
    profilesSampleRate: NODE_ENV === 'production' ? 0.05 : 1.0,

    environment: NODE_ENV,
    release: APP_VERSION,
    dist: `${APP_VERSION}-${NODE_ENV}`,

    ignoreErrors: [
      'Non-Error promise rejection captured',
      'UnhandledRejection',
      'Network Error',
      'Failed to fetch',
      'NetworkError when attempting to fetch resource',
    ],

    denyUrls: [
      /extensions\//i,
      /chrome-extension:\/\//i,
      /browser-extension:\/\//i,
      /webkit-masked-url/i,
      /moz-extension:\/\//i,
    ],

    beforeSend(event, hint) {
      if (NODE_ENV === 'production') {
        const error = hint?.originalException;
        if (error?.status === 401 || error?.status === 403) {
          return null;
        }
      }
      return event;
    },

    beforeSendTransaction(transaction) {
      if (transaction.contexts?.response?.status_code === 404) {
        return null;
      }
      return transaction;
    },

    maxBreadcrumbs: 100,
    maxValueLength: 1000,
    attachStacktrace: NODE_ENV !== 'production',
    sendDefaultPii: false,
    sendClientReports: true,
    normalizeDepth: 10,
    normalizeMaxBreadth: 1000,
  });

  console.log('✅ Sentry error tracking initialized');
}

const sentryService = {
  captureException(error, context = {}) {
    if (!isSentryEnabled) {
      console.error('Sentry not initialized:', error.message);
      return null;
    }
    
    return Sentry.withScope((scope) => {
      scope.setTag('application', 'hjtpx-api');
      scope.setTag('node_env', NODE_ENV);
      
      if (context.userId) {
        scope.setUser({ id: String(context.userId) });
      }
      
      if (context.requestId) {
        scope.setTag('request_id', context.requestId);
      }
      
      if (context.endpoint) {
        scope.setTag('endpoint', context.endpoint);
      }
      
      if (context.method) {
        scope.setTag('http_method', context.method);
      }

      if (context.tags) {
        Object.entries(context.tags).forEach(([key, value]) => {
          scope.setTag(key, value);
        });
      }

      if (context.extra) {
        Object.entries(context.extra).forEach(([key, value]) => {
          scope.setExtra(key, value);
        });
      }

      return Sentry.captureException(error, {
        contexts: context.contexts,
      });
    });
  },

  captureMessage(message, level = 'info', context = {}) {
    if (!isSentryEnabled) {
      console.log(`[${level.toUpperCase()}] ${message}`);
      return null;
    }

    return Sentry.withScope((scope) => {
      scope.setTag('application', 'hjtpx-api');
      scope.setTag('node_env', NODE_ENV);
      
      if (context.userId) {
        scope.setUser({ id: String(context.userId) });
      }

      if (context.tags) {
        Object.entries(context.tags).forEach(([key, value]) => {
          scope.setTag(key, value);
        });
      }

      if (context.extra) {
        Object.entries(context.extra).forEach(([key, value]) => {
          scope.setExtra(key, value);
        });
      }

      return Sentry.captureMessage(message, level);
    });
  },

  startTransaction(name, op, context = {}) {
    if (!isSentryEnabled) {
      const startTime = Date.now();
      return {
        end: () => ({ duration: Date.now() - startTime }),
        setTag: () => {},
        setData: () => {},
        setStatus: () => {},
        finish: () => ({ duration: Date.now() - startTime }),
      };
    }

    const transaction = Sentry.startTransaction({
      name,
      op,
      description: context.description,
      data: context.data,
    });

    return transaction;
  },

  addBreadcrumb(message, category, data = {}, level = 'info') {
    if (!isSentryEnabled) return;

    Sentry.addBreadcrumb({
      message,
      category,
      data,
      level,
      timestamp: Date.now() / 1000,
    });
  },

  setUser(user) {
    if (!isSentryEnabled) return;
    Sentry.setUser(user);
  },

  setTag(key, value) {
    if (!isSentryEnabled) return;
    Sentry.setTag(key, value);
  },

  setContext(key, value) {
    if (!isSentryEnabled) return;
    Sentry.setContext(key, value);
  },

  configureScope(callback) {
    if (!isSentryEnabled) return;
    Sentry.configureScope(callback);
  },

  getCurrentHub() {
    return isSentryEnabled ? Sentry.getCurrentHub() : null;
  },

  isEnabled() {
    return isSentryEnabled;
  },
};

module.exports = sentryService;
