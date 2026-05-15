const config = require('../../config/messageQueue');
const connectionManager = require('../connectionManager');
const deadLetterQueue = require('../retry/deadLetterQueue');
const { retryManager } = require('../retry/retryStrategy');

class StreamConsumer {
  constructor(queueName, options = {}) {
    this.queueName = queueName;
    this.queueConfig = config.queues[queueName];

    if (!this.queueConfig) {
      throw new Error(`Queue configuration not found for: ${queueName}`);
    }

    this.streamKey = this.queueConfig.stream;
    this.groupName = this.queueConfig.consumerGroup;
    this.consumerName = options.consumerName || this.queueConfig.consumerName;
    this.isRunning = false;
    this.handlers = new Map();
    this.messageHandler = null;
    this.errorHandler = null;
    this.blockTimeout = options.blockTimeout || config.streams.blockTimeout;
    this.batchSize = options.batchSize || 10;
    this.isInitialized = false;
  }

  async initialize() {
    if (this.isInitialized) {
      return;
    }

    await connectionManager.ensureStream(this.streamKey, config.streams.maxLen);
    await connectionManager.ensureConsumerGroup(this.streamKey, this.groupName, '0');

    await deadLetterQueue.initialize(this.queueConfig, this.queueConfig.deadLetterStream);

    this.isInitialized = true;
    console.log(
      `[StreamConsumer] Initialized consumer for queue: ${this.queueName} (group: ${this.groupName})`
    );
  }

  registerHandler(type, handler) {
    this.handlers.set(type, handler);
  }

  setMessageHandler(handler) {
    this.messageHandler = handler;
  }

  setErrorHandler(handler) {
    this.errorHandler = handler;
  }

  async consume(options = {}) {
    if (!this.isInitialized) {
      await this.initialize();
    }

    this.isRunning = true;
    const client = connectionManager.getClient();

    console.log(`[StreamConsumer] Starting consumption from ${this.queueName}`);

    while (this.isRunning) {
      try {
        const messages = await client.xreadgroup(
          'GROUP',
          this.groupName,
          this.consumerName,
          'COUNT',
          this.batchSize,
          'BLOCK',
          options.blockTimeout || this.blockTimeout,
          'STREAMS',
          this.streamKey,
          '>'
        );

        if (messages && messages.length > 0) {
          for (const [, entries] of messages) {
            for (const [messageId, fields] of entries) {
              await this.processMessage(messageId, fields);
            }
          }
        }
      } catch (error) {
        console.error(`[StreamConsumer] Error consuming messages:`, error);
        if (this.errorHandler) {
          await this.errorHandler(error);
        }
        await this.sleep(1000);
      }
    }
  }

  async processMessage(messageId, fields) {
    const client = connectionManager.getClient();
    let messageData;

    try {
      messageData = this.parseMessage(fields);
    } catch (error) {
      console.error(`[StreamConsumer] Failed to parse message ${messageId}:`, error);
      await client.xack(this.streamKey, this.groupName, messageId);
      return;
    }

    const startTime = Date.now();
    let retryCount = 0;

    while (retryCount <= this.queueConfig.maxRetries) {
      try {
        const handler = this.handlers.get(messageData.type) || this.messageHandler;

        if (handler) {
          await handler(messageData);
        } else {
          console.warn(`[StreamConsumer] No handler for message type: ${messageData.type}`);
        }

        await client.xack(this.streamKey, this.groupName, messageId);
        const processingTime = Date.now() - startTime;
        console.log(`[StreamConsumer] Processed message ${messageId} in ${processingTime}ms`);
        return;
      } catch (error) {
        retryCount++;
        console.error(
          `[StreamConsumer] Error processing message ${messageId} (attempt ${retryCount}/${this.queueConfig.maxRetries}):`,
          error.message
        );

        if (retryCount <= this.queueConfig.maxRetries) {
          const delay = this.queueConfig.retryDelay * retryCount;
          console.log(`[StreamConsumer] Retrying in ${delay}ms...`);
          await this.sleep(delay);
        } else {
          console.error(
            `[StreamConsumer] Max retries exceeded for message ${messageId}, sending to DLQ`
          );
          await deadLetterQueue.sendToDLQ(error, {
            stream: this.streamKey,
            messageId,
            data: messageData,
            retryCount
          });
          await client.xack(this.streamKey, this.groupName, messageId);
        }
      }
    }
  }

  parseMessage(fields) {
    const data = {};
    for (let i = 0; i < fields.length; i += 2) {
      const key = fields[i];
      let value = fields[i + 1];

      try {
        value = JSON.parse(value);
      } catch (e) {}

      data[key] = value;
    }

    if (data.payload) {
      try {
        data.payload = JSON.parse(data.payload);
      } catch (e) {}
    }

    return data;
  }

  async consumePending() {
    if (!this.isInitialized) {
      await this.initialize();
    }

    const client = connectionManager.getClient();

    try {
      const messages = await client.xreadgroup(
        'GROUP',
        this.groupName,
        this.consumerName,
        'COUNT',
        this.batchSize,
        'STREAMS',
        this.streamKey,
        '0'
      );

      if (messages && messages.length > 0) {
        for (const [, entries] of messages) {
          for (const [messageId, fields] of entries) {
            await this.processMessage(messageId, fields);
          }
        }
      }
    } catch (error) {
      console.error(`[StreamConsumer] Error consuming pending messages:`, error);
    }
  }

  stop() {
    console.log(`[StreamConsumer] Stopping consumer for ${this.queueName}`);
    this.isRunning = false;
  }

  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  async healthCheck() {
    try {
      const client = connectionManager.getClient();
      await client.ping();

      const groupInfo = await connectionManager.getConsumerGroupInfo(
        this.streamKey,
        this.groupName
      );
      const streamLength = await client.xlen(this.streamKey);

      return {
        healthy: true,
        queue: this.queueName,
        streamKey: this.streamKey,
        consumerGroup: this.groupName,
        consumerName: this.consumerName,
        isRunning: this.isRunning,
        isInitialized: this.isInitialized,
        streamLength,
        pendingCount: groupInfo?.pending || 0,
        consumerCount: groupInfo?.consumers || 0
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

class ConsumerManager {
  constructor() {
    this.consumers = new Map();
  }

  async createConsumer(queueName, options = {}) {
    if (this.consumers.has(queueName)) {
      return this.consumers.get(queueName);
    }

    const consumer = new StreamConsumer(queueName, options);
    await consumer.initialize();
    this.consumers.set(queueName, consumer);
    return consumer;
  }

  async getConsumer(queueName) {
    if (!this.consumers.has(queueName)) {
      return await this.createConsumer(queueName);
    }
    return this.consumers.get(queueName);
  }

  async startAll() {
    const promises = [];
    for (const [name, consumer] of this.consumers) {
      promises.push(
        consumer.consume().catch(err => {
          console.error(`[ConsumerManager] Consumer ${name} error:`, err);
        })
      );
    }
    await Promise.all(promises);
  }

  async stopAll() {
    for (const consumer of this.consumers.values()) {
      consumer.stop();
    }
    console.log(`[ConsumerManager] All consumers stopped`);
  }

  async healthCheck() {
    const results = {};
    for (const [name, consumer] of this.consumers) {
      results[name] = await consumer.healthCheck();
    }
    return results;
  }
}

const consumerManager = new ConsumerManager();

module.exports = {
  StreamConsumer,
  ConsumerManager,
  consumerManager
};
