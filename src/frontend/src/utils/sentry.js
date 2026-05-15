import * as Sentry from "@sentry/react";
import { browserTracingIntegration, replayIntegration } from "@sentry/react";

const environment = import.meta.env.MODE || 'development';

const tracesSampleRate = parseFloat(import.meta.env.VITE_SENTRY_TRACES_SAMPLE_RATE);
const replaysSampleRate = parseFloat(import.meta.env.VITE_SENTRY_REPLAYS_SAMPLE_RATE);

export function initSentry() {
  const dsn = import.meta.env.VITE_SENTRY_DSN;
  
  if (!dsn) {
    console.log('⚠️ VITE_SENTRY_DSN 未配置，Sentry 前端监控将不会启动');
    return;
  }

  Sentry.init({
    dsn,
    environment,
    release: import.meta.env.VITE_APP_VERSION || 'hjtpx-frontend@1.0.0',
    
    integrations: [
      browserTracingIntegration({
        tracePropagationTargets: ['localhost', /^\/api\//],
      }),
      replayIntegration({
        maskAllText: true,
        maskAllInputs: true,
        blockAllMedia: true,
      }),
    ],
    
    tracesSampleRate: !isNaN(tracesSampleRate) ? tracesSampleRate : (environment === 'production' ? 0.1 : 1.0),
    replaysSessionSampleRate: !isNaN(replaysSampleRate) ? replaysSampleRate : (environment === 'production' ? 0.05 : 0.1),
    replaysOnErrorSampleRate: 1.0,
    
    ignoreErrors: [
      'Network Error',
      'Failed to fetch',
      'Network request failed',
      'Non-Error promise rejection captured',
      'ResizeObserver loop',
    ],
    
    denyUrls: [
      /localhost/,
      /127\.0\.0\.1/,
    ],
    
    beforeSend(event) {
      if (event.request && event.request.headers) {
        delete event.request.headers['Authorization'];
        delete event.request.headers['Cookie'];
      }
      return event;
    },
    
    initialScope: {
      tags: {
        'app.name': 'hjtpx-frontend',
        'app.version': '1.0.0',
        'react.version': window.React?.version || 'unknown',
      },
    },
  });

  console.log('✅ Sentry 前端监控已初始化');
  console.log(`   环境: ${environment}`);
  console.log(`   采样率: ${environment === 'production' ? '10%' : '100%'}`);
}

export function setUserContext(user) {
  if (user) {
    Sentry.setUser({
      id: user.id,
      email: user.email,
      username: user.username,
    });
  } else {
    Sentry.setUser(null);
  }
}

export function setTagContext(key, value) {
  Sentry.setTag(key, value);
}

export function addBreadcrumb(message, category, level = 'info', data = {}) {
  Sentry.addBreadcrumb({
    message,
    category,
    level,
    data,
    timestamp: Date.now() / 1000,
  });
}

export function recordPageLoad(pageName, loadTime) {
  addBreadcrumb(`页面加载: ${pageName}`, 'navigation', 'info', { loadTime: `${loadTime.toFixed(2)}ms` });
}

export function recordApiCall(endpoint, method, status, duration) {
  addBreadcrumb(
    `${method} ${endpoint}`,
    'api',
    status >= 400 ? 'warning' : 'info',
    { status, duration: `${duration.toFixed(2)}ms` }
  );
}

export function recordUserInteraction(element, action) {
  addBreadcrumb(`用户交互: ${action}`, 'interaction', 'info', { element });
}

export function recordError(error, context = {}) {
  Sentry.captureException(error, {
    extra: context,
  });
}

export function recordMetric(name, value, unit = 'none') {
  Sentry.metrics.increment(name, value, { unit });
}

export const SentryLogger = {
  info: (message, data) => addBreadcrumb(message, 'info', 'info', data),
  warn: (message, data) => addBreadcrumb(message, 'warning', 'warning', data),
  error: (message, data) => {
    addBreadcrumb(message, 'error', 'error', data);
    recordError(new Error(message), data);
  },
  debug: (message, data) => addBreadcrumb(message, 'debug', 'debug', data),
};

export default Sentry;
