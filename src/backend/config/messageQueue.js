module.exports = {
  redis: {
    host: process.env.REDIS_HOST || 'localhost',
    port: parseInt(process.env.REDIS_PORT || '6379'),
    password: process.env.REDIS_PASSWORD || undefined,
    db: parseInt(process.env.REDIS_DB || '0'),
    keyPrefix: process.env.REDIS_KEY_PREFIX || 'hjtpx:',
    connectTimeout: parseInt(process.env.REDIS_CONNECT_TIMEOUT || '10000'),
    commandTimeout: parseInt(process.env.REDIS_COMMAND_TIMEOUT || '5000'),
    retryStrategy: (times) => {
      const delay = Math.min(times * 50, 2000);
      return delay;
    },
    maxRetriesPerRequest: 3,
    enableReadyCheck: true,
    enableOfflineQueue: true
  },

  streams: {
    enabled: process.env.STREAMS_ENABLED !== 'false',
    consumerGroupPrefix: process.env.STREAMS_CONSUMER_GROUP_PREFIX || 'hjtpx-consumer',
    blockTimeout: parseInt(process.env.STREAMS_BLOCK_TIMEOUT || '5000'),
    claimTimeout: parseInt(process.env.STREAMS_CLAIM_TIMEOUT || '30000'),
    maxLen: parseInt(process.env.STREAMS_MAX_LEN || '10000')
  },

  queues: {
    email: {
      stream: 'hjtpx:streams:email',
      consumerGroup: 'hjtpx:consumers:email',
      consumerName: process.env.HOSTNAME || 'worker-email',
      maxRetries: 3,
      retryDelay: 5000,
      deadLetterStream: 'hjtpx:streams:email:dlq'
    },
    notification: {
      stream: 'hjtpx:streams:notification',
      consumerGroup: 'hjtpx:consumers:notification',
      consumerName: process.env.HOSTNAME || 'worker-notification',
      maxRetries: 3,
      retryDelay: 5000,
      deadLetterStream: 'hjtpx:streams:notification:dlq'
    },
    export: {
      stream: 'hjtpx:streams:export',
      consumerGroup: 'hjtpx:consumers:export',
      consumerName: process.env.HOSTNAME || 'worker-export',
      maxRetries: 2,
      retryDelay: 10000,
      deadLetterStream: 'hjtpx:streams:export:dlq'
    },
    logging: {
      stream: 'hjtpx:streams:logging',
      consumerGroup: 'hjtpx:consumers:logging',
      consumerName: process.env.HOSTNAME || 'worker-logging',
      maxRetries: 5,
      retryDelay: 1000,
      deadLetterStream: 'hjtpx:streams:logging:dlq'
    }
  },

  events: {
    enabled: process.env.EVENTS_ENABLED !== 'false',
    pubSubPrefix: process.env.EVENTS_PUBSUB_PREFIX || 'hjtpx:events:'
  },

  monitoring: {
    enabled: process.env.MQ_MONITORING_ENABLED !== 'false',
    metricsInterval: parseInt(process.env.MQ_METRICS_INTERVAL || '30000'),
    alertThreshold: {
      queueLength: parseInt(process.env.MQ_ALERT_QUEUE_LENGTH || '1000'),
      processingTime: parseInt(process.env.MQ_ALERT_PROCESSING_TIME || '30000'),
      failureRate: parseFloat(process.env.MQ_ALERT_FAILURE_RATE || '0.1')
    }
  },

  retry: {
    maxAttempts: 5,
    initialDelay: 1000,
    maxDelay: 60000,
    backoffMultiplier: 2,
    jitter: true
  }
};
