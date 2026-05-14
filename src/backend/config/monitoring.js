module.exports = {
  alerts: {
    critical: {
      name: 'Critical Error Spike',
      condition: 'error_count > 5',
      window: 300,
      severity: 'critical',
      channels: ['email', 'slack', 'pagerduty'],
      message: 'Critical error spike detected: {{error_count}} errors in {{window}} seconds',
      cooldown: 900,
    },
    warning: {
      name: 'Warning Error Rate',
      condition: 'error_rate > 0.05',
      window: 600,
      severity: 'warning',
      channels: ['email', 'slack'],
      message: 'High error rate detected: {{error_rate}}% errors in {{window}} seconds',
      cooldown: 600,
    },
    performance: {
      name: 'Performance Degradation',
      condition: 'avg_response_time > 2000',
      window: 300,
      severity: 'warning',
      channels: ['slack'],
      message: 'Performance degradation: average response time {{avg_response_time}}ms',
      cooldown: 600,
    },
    slowQueries: {
      name: 'Slow Database Queries',
      condition: 'slow_query_count > 10',
      window: 300,
      severity: 'warning',
      channels: ['slack'],
      message: 'Multiple slow queries detected: {{slow_query_count}} queries over {{window}}s',
      cooldown: 300,
    },
    memory: {
      name: 'Memory Usage High',
      condition: 'memory_usage > 0.85',
      window: 60,
      severity: 'warning',
      channels: ['email', 'slack'],
      message: 'High memory usage: {{memory_usage}}%',
      cooldown: 300,
    },
    cpu: {
      name: 'CPU Usage High',
      condition: 'cpu_usage > 0.90',
      window: 60,
      severity: 'warning',
      channels: ['email', 'slack'],
      message: 'High CPU usage: {{cpu_usage}}%',
      cooldown: 300,
    },
  },

  errorGroups: {
    database: {
      patterns: ['MongoError', 'MongooseError', 'MongoServerError', 'connection'],
      threshold: 10,
      aggregation: 'count',
    },
    authentication: {
      patterns: ['UnauthorizedError', 'JsonWebTokenError', 'TokenExpiredError', 'invalid token'],
      threshold: 5,
      aggregation: 'count',
    },
    validation: {
      patterns: ['ValidationError', 'JoiError', 'ValidatorError'],
      threshold: 20,
      aggregation: 'count',
    },
    network: {
      patterns: ['ECONNREFUSED', 'ETIMEDOUT', 'ENOTFOUND', 'network'],
      threshold: 5,
      aggregation: 'count',
    },
  },

  notifications: {
    email: {
      enabled: process.env.ALERT_EMAIL_ENABLED === 'true',
      recipients: (process.env.ALERT_EMAIL_RECIPIENTS || '').split(',').filter(Boolean),
    },
    slack: {
      enabled: process.env.SLACK_WEBHOOK_ENABLED === 'true',
      webhookUrl: process.env.SLACK_WEBHOOK_URL,
      channel: process.env.SLACK_CHANNEL || '#alerts',
    },
    pagerduty: {
      enabled: process.env.PAGERDUTY_ENABLED === 'true',
      integrationKey: process.env.PAGERDUTY_INTEGRATION_KEY,
    },
  },

  sourceMaps: {
    enabled: process.env.SENTRY_SOURCEMAPS_ENABLED === 'true',
    uploadPath: process.env.SENTRY_SOURCEMAPS_PATH || './dist',
    urlPrefix: process.env.SENTRY_SOURCEMAPS_URL_PREFIX || '~/',
    ignore: ['node_modules', 'tests', 'migrations'],
  },

  performance: {
    traces: {
      sampleRate: parseFloat(process.env.SENTRY_TRACES_SAMPLE_RATE) || 0.1,
      maxTransactionDuration: 60000,
    },
    profiling: {
      enabled: process.env.SENTRY_PROFILING_ENABLED === 'true',
      sampleRate: parseFloat(process.env.SENTRY_PROFILES_SAMPLE_RATE) || 0.1,
    },
  },
};
