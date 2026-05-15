const { producerManager } = require('./producers/streamProducer');
const { consumerManager } = require('./consumers/streamConsumer');

class LoggingQueueService {
  constructor() {
    this.queueName = 'logging';
  }

  async log(level, message, metadata = {}, options = {}) {
    return await producerManager.send(this.queueName, {
      level,
      message,
      metadata,
      timestamp: new Date().toISOString(),
      source: options.source || 'application'
    }, {
      type: 'log_entry',
      priority: this.getPriorityForLevel(level)
    });
  }

  getPriorityForLevel(level) {
    const priorities = {
      error: 10,
      warn: 7,
      info: 5,
      debug: 3,
      trace: 1
    };
    return priorities[level] || 5;
  }

  async logError(message, error = null, metadata = {}) {
    return await this.log('error', message, {
      ...metadata,
      error: error ? {
        message: error.message,
        stack: error.stack,
        name: error.name
      } : null
    });
  }

  async logWarn(message, metadata = {}) {
    return await this.log('warn', message, metadata);
  }

  async logInfo(message, metadata = {}) {
    return await this.log('info', message, metadata);
  }

  async logDebug(message, metadata = {}) {
    return await this.log('debug', message, metadata);
  }

  async logSecurityEvent(eventType, details = {}) {
    return await producerManager.send(this.queueName, {
      level: 'warn',
      message: `Security event: ${eventType}`,
      metadata: {
        eventType,
        ...details,
        securityEvent: true
      },
      timestamp: new Date().toISOString(),
      source: 'security'
    }, {
      type: 'security_log',
      priority: 10
    });
  }

  async logUserAction(userId, action, details = {}) {
    return await producerManager.send(this.queueName, {
      level: 'info',
      message: `User action: ${action}`,
      metadata: {
        userId,
        action,
        ...details,
        userAction: true
      },
      timestamp: new Date().toISOString(),
      source: 'user'
    }, {
      type: 'user_action',
      priority: 5
    });
  }

  async logPerformanceMetric(metricName, value, metadata = {}) {
    return await producerManager.send(this.queueName, {
      level: 'info',
      message: `Performance metric: ${metricName}`,
      metadata: {
        metricName,
        value,
        unit: metadata.unit || 'ms',
        ...metadata,
        performanceMetric: true
      },
      timestamp: new Date().toISOString(),
      source: 'performance'
    }, {
      type: 'performance_log',
      priority: 3
    });
  }

  async startConsumer(options = {}) {
    const consumer = await consumerManager.createConsumer(this.queueName, options);

    consumer.registerHandler('log_entry', async (message) => {
      const logger = require('../../utils/logger');

      const { level, message: logMessage, metadata, timestamp, source } = message.payload;

      const logData = {
        timestamp,
        source,
        ...metadata
      };

      switch (level) {
        case 'error':
          logger.error(logMessage, logData);
          break;
        case 'warn':
          logger.warn(logMessage, logData);
          break;
        case 'info':
          logger.info(logMessage, logData);
          break;
        case 'debug':
          logger.debug(logMessage, logData);
          break;
        default:
          logger.info(logMessage, logData);
      }
    });

    consumer.registerHandler('security_log', async (message) => {
      const auditLogger = require('../../utils/security/audit_logger');
      const { message: logMessage, metadata, timestamp } = message.payload;

      await auditLogger.log({
        event: metadata.eventType,
        timestamp,
        details: metadata,
        severity: 'high'
      });
    });

    consumer.registerHandler('user_action', async (message) => {
      const analyticsService = require('../analyticsService');

      const { metadata, timestamp } = message.payload;

      await analyticsService.trackEvent({
        event: metadata.action,
        userId: metadata.userId,
        properties: metadata,
        timestamp
      });
    });

    consumer.registerHandler('performance_log', async (message) => {
      const metricsService = require('../metricsService');

      const { metadata, timestamp } = message.payload;

      await metricsService.recordMetric(
        metadata.metricName,
        metadata.value,
        {
          unit: metadata.unit,
          tags: metadata.tags,
          timestamp
        }
      );
    });

    return consumer;
  }
}

const loggingQueueService = new LoggingQueueService();

module.exports = loggingQueueService;
