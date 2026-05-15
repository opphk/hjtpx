const NODE_ENV = process.env.NODE_ENV || 'development';

const errorGroupingRules = {
  version: '2024-01',
  
  groupingRules: [
    {
      id: 'database-errors',
      matchers: [
        ['type', 'DatabaseError'],
        ['type', 'MongoError'],
        ['type', 'PostgresError'],
        ['type', 'ConnectionError'],
      ],
      strategy: 'logentry',
      config: {
        message: '{{ default }}',
      },
    },
    {
      id: 'validation-errors',
      matchers: [
        ['type', 'ValidationError'],
        ['type', 'JoiError'],
        ['message', 'contains', 'validation'],
        ['message', 'startswith', 'Invalid'],
      ],
      strategy: 'logentry',
      config: {
        message: 'Validation failed: {{ params.field }}',
      },
    },
    {
      id: 'authentication-errors',
      matchers: [
        ['type', 'AuthenticationError'],
        ['type', 'UnauthorizedError'],
        ['message', 'contains', 'Invalid token'],
        ['message', 'contains', 'Token expired'],
      ],
      strategy: 'logentry',
      config: {
        message: 'Authentication failed',
      },
    },
    {
      id: 'rate-limit-errors',
      matchers: [
        ['type', 'RateLimitError'],
        ['message', 'contains', 'Too many requests'],
        ['message', 'contains', 'rate limit'],
      ],
      strategy: 'logentry',
      config: {
        message: 'Rate limit exceeded',
      },
    },
    {
      id: 'http-client-errors',
      matchers: [
        ['type', 'FetchError'],
        ['type', 'RequestError'],
        ['type', 'NetworkError'],
        ['message', 'contains', 'Failed to fetch'],
      ],
      strategy: 'logentry',
      config: {
        message: 'HTTP client error',
      },
    },
    {
      id: 'timeout-errors',
      matchers: [
        ['type', 'TimeoutError'],
        ['type', 'ETIMEDOUT'],
        ['message', 'contains', 'timeout'],
      ],
      strategy: 'logentry',
      config: {
        message: 'Operation timeout',
      },
    },
    {
      id: 'syntax-errors',
      matchers: [
        ['type', 'SyntaxError'],
        ['type', 'ParseError'],
        ['message', 'contains', 'Unexpected token'],
        ['message', 'contains', 'JSON.parse'],
      ],
      strategy: 'logentry',
      config: {
        message: 'Parse error',
      },
    },
  ],

  enhancements: [
    {
      id: 'performance-tag',
      type: 'enhance-measurement',
      metric: 'duration',
      tags: {
        'performance.category': 'function',
      },
    },
    {
      id: 'environment-tag',
      type: 'enhance-context',
      tags: {
        environment: NODE_ENV,
      },
    },
  ],
};

const alertRules = {
  critical: {
    name: 'Critical Error Alert',
    conditions: [
      { type: 'event', op: 'eq', name: 'level', value: 'fatal' },
      { type: 'event', op: 'eq', name: 'is_crashed', value: true },
    ],
    filters: [
      { type: 'name', value: 'browser-extension' },
    ],
    actions: [
      {
        type: 'notification',
        target: 'critical-pagerduty',
        priority: 'high',
      },
      {
        type: 'slack',
        channel: '#alerts-critical',
        message: '🚨 Critical error detected in {{ project }}',
      },
    ],
    aggregation: {
      type: 'count',
      window: '5m',
      threshold: 1,
    },
    urgency: 'high',
    cooldown: '30m',
  },

  errorSpike: {
    name: 'Error Spike Detection',
    conditions: [
      { type: 'event', op: 'eq', name: 'level', value: 'error' },
    ],
    filters: [],
    actions: [
      {
        type: 'notification',
        target: 'pagerduty',
        priority: 'medium',
      },
      {
        type: 'slack',
        channel: '#alerts-errors',
        message: '⚠️ Error spike detected: {{ count }} errors in {{ window }}',
      },
      {
        type: 'email',
        recipients: ['devops@company.com'],
      },
    ],
    aggregation: {
      type: 'count',
      window: '5m',
      threshold: 10,
    },
    comparison: {
      type: 'threshold',
      comparisonWindow: '1h',
      percentChange: 200,
    },
    urgency: 'high',
    cooldown: '1h',
  },

  performanceDegradation: {
    name: 'Performance Degradation Alert',
    conditions: [
      { type: 'transaction', op: 'gt', name: 'duration_p95', value: 2000 },
      { type: 'transaction', op: 'gt', name: 'failure_rate', value: 5 },
    ],
    filters: [
      { type: 'name', value: 'health-check' },
    ],
    actions: [
      {
        type: 'slack',
        channel: '#alerts-performance',
        message: '📊 Performance degradation: {{ endpoint }} - p95: {{ duration_p95 }}ms',
      },
    ],
    aggregation: {
      type: 'avg',
      window: '10m',
      threshold: 2000,
    },
    urgency: 'medium',
    cooldown: '2h',
  },

  memoryLeak: {
    name: 'Memory Leak Detection',
    conditions: [
      { type: 'measurement', op: 'gt', name: 'memory.usage', value: 90 },
      { type: 'profile', op: 'contains', name: 'allocation', value: 'growing' },
    ],
    filters: [],
    actions: [
      {
        type: 'notification',
        target: 'pagerduty',
        priority: 'low',
      },
      {
        type: 'slack',
        channel: '#alerts-performance',
        message: '🔍 Potential memory leak detected in {{ service }}',
      },
    ],
    aggregation: {
      type: 'avg',
      window: '30m',
      threshold: 85,
    },
    urgency: 'medium',
    cooldown: '4h',
  },

  databaseConnection: {
    name: 'Database Connection Alert',
    conditions: [
      { type: 'event', op: 'regex', name: 'message', value: 'ECONNREFUSED|ETIMEDOUT|ENOTFOUND' },
      { type: 'event', op: 'regex', name: 'type', value: 'MongoError|PostgresError|DatabaseError' },
    ],
    filters: [],
    actions: [
      {
        type: 'notification',
        target: 'critical-pagerduty',
        priority: 'high',
      },
      {
        type: 'slack',
        channel: '#alerts-database',
        message: '🗄️ Database connection issue: {{ error_type }}',
      },
      {
        type: 'webhook',
        url: process.env.DATABASE_ALERT_WEBHOOK || '',
        payload: {
          event: 'database_connection_error',
          timestamp: '{{ timestamp }}',
          error: '{{ error_message }}',
        },
      },
    ],
    aggregation: {
      type: 'count',
      window: '5m',
      threshold: 3,
    },
    urgency: 'critical',
    cooldown: '15m',
  },

  apiAvailability: {
    name: 'API Availability Monitor',
    conditions: [
      { type: 'transaction', op: 'eq', name: 'status', value: 'ok' },
      { type: 'event', op: 'eq', name: 'type', value: 'missing_page' },
    ],
    filters: [
      { type: 'name', value: 'health-check' },
    ],
    actions: [
      {
        type: 'slack',
        channel: '#alerts-availability',
        message: '📉 API availability dropped: {{ availability }}%',
      },
    ],
    aggregation: {
      type: 'percentage',
      window: '5m',
      threshold: 99,
    },
    urgency: 'medium',
    cooldown: '1h',
  },
};

const monitoringConfig = {
  enabled: NODE_ENV === 'production',
  
  performance: {
    transactionTracing: true,
    traceRequests: true,
    traceServerTime: true,
    traceWaitTime: true,
    traceConnectTime: true,
    traceResponseTime: true,
    maxTransactionDuration: 60000,
    slowQueryThreshold: 1000,
  },

  profiling: {
    enabled: NODE_ENV === 'production',
    captureStackTrace: true,
    maxDepth: 50,
    sampleRate: 0.1,
  },

  metrics: {
    capture: true,
    intervals: {
      cpu: 1000,
      memory: 1000,
      eventLoop: 1000,
    },
  },

  sampling: {
    transactionSampleRate: NODE_ENV === 'production' ? 0.1 : 1.0,
    errorSampleRate: 1.0,
    profileSampleRate: NODE_ENV === 'production' ? 0.05 : 1.0,
  },

  sourceMaps: {
    enabled: NODE_ENV === 'production',
    release: process.env.APP_VERSION || '1.0.0',
    paths: [
      '~/src',
      '~/.next',
      './dist',
      './build',
    ],
    ignore: [
      'node_modules',
      'webpack',
      'jest',
    ],
    rewrite: true,
    validate: true,
  },
};

module.exports = {
  errorGroupingRules,
  alertRules,
  monitoringConfig,
};
