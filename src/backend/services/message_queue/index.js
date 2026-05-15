const connectionManager = require('./connectionManager');
const { producerManager } = require('./producers/streamProducer');
const { consumerManager } = require('./consumers/streamConsumer');
const { retryManager } = require('./retry/retryStrategy');
const deadLetterQueue = require('./retry/deadLetterQueue');
const eventPublisher = require('./events/eventPublisher');
const eventSubscriber = require('./events/eventSubscriber');
const { eventProcessor } = require('./events/eventProcessor');
const queueMonitor = require('./monitoring/queueMonitor');

const emailQueueService = require('./emailQueueService');
const notificationQueueService = require('./notificationQueueService');
const exportQueueService = require('./exportQueueService');
const loggingQueueService = require('./loggingQueueService');

class MessageQueueManager {
  constructor() {
    this.isInitialized = false;
    this.isRunning = false;
    this.consumers = new Map();
  }

  async initialize() {
    if (this.isInitialized) {
      return;
    }

    console.log('[MessageQueueManager] Initializing...');

    await connectionManager.connect();

    await producerManager.initializeAll();

    await eventPublisher.connect();
    await eventSubscriber.initialize();

    await queueMonitor.start();

    this.isInitialized = true;
    console.log('[MessageQueueManager] Initialized successfully');
  }

  async startConsumers(options = {}) {
    if (!this.isInitialized) {
      await this.initialize();
    }

    console.log('[MessageQueueManager] Starting consumers...');

    if (options.email) {
      const emailConsumer = await emailQueueService.startConsumer();
      this.consumers.set('email', emailConsumer);
      emailConsumer.consume().catch(err => {
        console.error('[MessageQueueManager] Email consumer error:', err);
      });
    }

    if (options.notification) {
      const notificationConsumer = await notificationQueueService.startConsumer();
      this.consumers.set('notification', notificationConsumer);
      notificationConsumer.consume().catch(err => {
        console.error('[MessageQueueManager] Notification consumer error:', err);
      });
    }

    if (options.export) {
      const exportConsumer = await exportQueueService.startConsumer();
      this.consumers.set('export', exportConsumer);
      exportConsumer.consume().catch(err => {
        console.error('[MessageQueueManager] Export consumer error:', err);
      });
    }

    if (options.logging) {
      const loggingConsumer = await loggingQueueService.startConsumer();
      this.consumers.set('logging', loggingConsumer);
      loggingConsumer.consume().catch(err => {
        console.error('[MessageQueueManager] Logging consumer error:', err);
      });
    }

    if (options.events) {
      await eventProcessor.initialize();
    }

    this.isRunning = true;
    console.log('[MessageQueueManager] Consumers started');
  }

  async stop() {
    console.log('[MessageQueueManager] Stopping...');

    for (const [name, consumer] of this.consumers) {
      consumer.stop();
    }
    this.consumers.clear();

    await eventProcessor.stopProcessing();
    await eventSubscriber.stopListening();

    queueMonitor.stop();
    await connectionManager.disconnect();

    this.isRunning = false;
    this.isInitialized = false;
    console.log('[MessageQueueManager] Stopped');
  }

  async healthCheck() {
    const checks = {
      connection: await connectionManager.healthCheck(),
      producers: await producerManager.healthCheck(),
      consumers: await consumerManager.healthCheck(),
      monitor: await queueMonitor.healthCheck(),
      eventPublisher: await eventPublisher.healthCheck(),
      eventSubscriber: await eventSubscriber.healthCheck(),
      eventProcessor: await eventProcessor.healthCheck(),
      dlqStats: await deadLetterQueue.getAllDLQStats()
    };

    const allHealthy = Object.values(checks).every(check => {
      if (typeof check === 'object' && 'healthy' in check) {
        return check.healthy;
      }
      return true;
    });

    return {
      healthy: allHealthy,
      isRunning: this.isRunning,
      isInitialized: this.isInitialized,
      checks
    };
  }

  getEmailService() {
    return emailQueueService;
  }

  getNotificationService() {
    return notificationQueueService;
  }

  getExportService() {
    return exportQueueService;
  }

  getLoggingService() {
    return loggingQueueService;
  }

  getEventPublisher() {
    return eventPublisher;
  }

  getEventSubscriber() {
    return eventSubscriber;
  }

  getEventProcessor() {
    return eventProcessor;
  }

  getMonitor() {
    return queueMonitor;
  }

  getRetryManager() {
    return retryManager;
  }

  getDLQManager() {
    return deadLetterQueue;
  }
}

const messageQueueManager = new MessageQueueManager();

module.exports = messageQueueManager;
