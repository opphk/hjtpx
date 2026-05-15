const connectionManager = require('../connectionManager');
const config = require('../../config/messageQueue');
const { v4: uuidv4 } = require('uuid');

class StreamProducer {
  constructor(queueName) {
    this.queueName = queueName;
    this.queueConfig = config.queues[queueName];
    if (!this.queueConfig) {
      throw new Error(`Queue configuration not found for: ${queueName}`);
    }
    this.streamKey = this.queueConfig.stream;
    this.isInitialized = false;
  }

  async initialize() {
    if (this.isInitialized) {
      return;
    }

    await connectionManager.ensureStream(this.streamKey, config.streams.maxLen);
    await connectionManager.ensureConsumerGroup(
      this.streamKey,
      this.queueConfig.consumerGroup,
      '0'
    );

    this.isInitialized = true;
    console.log(`[StreamProducer] Initialized producer for queue: ${this.queueName}`);
  }

  async send(message, options = {}) {
    if (!this.isInitialized) {
      await this.initialize();
    }

    const client = connectionManager.getClient();
    const messageId = options.messageId || '*';
    const priority = options.priority || 0;

    const messageData = {
      id: uuidv4(),
      type: options.type || 'default',
      payload: JSON.stringify(message),
      priority: priority.toString(),
      timestamp: new Date().toISOString(),
      correlationId: options.correlationId || uuidv4(),
      ...(options.headers || {})
    };

    if (options.delayed && options.delay > 0) {
      messageData._delay = options.delay.toString();
      messageData._scheduledTime = new Date(Date.now() + options.delay).toISOString();
    }

    try {
      const entryId = await client.xadd(
        this.streamKey,
        'MAXLEN',
        '~',
        (options.maxLen || config.streams.maxLen).toString(),
        messageId,
        ...this.flattenMessage(messageData)
      );

      console.log(`[StreamProducer] Message sent to ${this.queueName}: ${entryId}`);
      return {
        messageId: entryId,
        queue: this.queueName,
        timestamp: messageData.timestamp,
        correlationId: messageData.correlationId
      };
    } catch (error) {
      console.error(`[StreamProducer] Failed to send message:`, error);
      throw error;
    }
  }

  flattenMessage(message) {
    const result = [];
    for (const [key, value] of Object.entries(message)) {
      if (typeof value === 'object' && value !== null) {
        result.push(key, JSON.stringify(value));
      } else {
        result.push(key, String(value));
      }
    }
    return result;
  }

  async sendBatch(messages, options = {}) {
    const results = [];
    for (const message of messages) {
      try {
        const result = await this.send(message, options);
        results.push({ success: true, ...result });
      } catch (error) {
        results.push({ success: false, error: error.message });
      }
    }
    return results;
  }

  async sendWithDelay(message, delayMs, options = {}) {
    return this.send(message, { ...options, delayed: true, delay: delayMs });
  }

  async getQueueLength() {
    const client = connectionManager.getClient();
    return await client.xlen(this.streamKey);
  }

  async getPendingCount() {
    const client = connectionManager.getClient();
    try {
      const info = await connectionManager.getConsumerGroupInfo(this.streamKey, this.queueConfig.consumerGroup);
      return info ? info.pending : 0;
    } catch (error) {
      return 0;
    }
  }

  async healthCheck() {
    try {
      const client = connectionManager.getClient();
      await client.ping();
      const length = await this.getQueueLength();
      const pending = await this.getPendingCount();
      return {
        healthy: true,
        queue: this.queueName,
        streamKey: this.streamKey,
        messages: length,
        pending: pending
      };
    } catch (error) {
      return {
        healthy: false,
        queue: this.queueName,
        error: error.message
      };
    }
  }
}

class ProducerManager {
  constructor() {
    this.producers = new Map();
  }

  async initializeProducer(queueName) {
    if (this.producers.has(queueName)) {
      return this.producers.get(queueName);
    }

    const producer = new StreamProducer(queueName);
    await producer.initialize();
    this.producers.set(queueName, producer);
    return producer;
  }

  async getProducer(queueName) {
    if (!this.producers.has(queueName)) {
      return await this.initializeProducer(queueName);
    }
    return this.producers.get(queueName);
  }

  async send(queueName, message, options = {}) {
    const producer = await this.getProducer(queueName);
    return await producer.send(message, options);
  }

  async sendBatch(queueName, messages, options = {}) {
    const producer = await this.getProducer(queueName);
    return await producer.sendBatch(messages, options);
  }

  async sendWithDelay(queueName, message, delayMs, options = {}) {
    const producer = await this.getProducer(queueName);
    return await producer.sendWithDelay(message, delayMs, options);
  }

  async healthCheck() {
    const results = {};
    for (const [name, producer] of this.producers) {
      results[name] = await producer.healthCheck();
    }
    return results;
  }

  async initializeAll() {
    const queueNames = Object.keys(config.queues);
    for (const name of queueNames) {
      await this.initializeProducer(name);
    }
    console.log(`[ProducerManager] Initialized ${this.producers.size} producers`);
  }
}

const producerManager = new ProducerManager();

module.exports = {
  StreamProducer,
  ProducerManager,
  producerManager
};
