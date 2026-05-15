const connectionManager = require('../connectionManager');
const config = require('../../config/messageQueue');

class DeadLetterQueueManager {
  constructor() {
    this.dlqStreams = new Map();
    this.processors = new Map();
  }

  async initialize(queueConfig, dlqStreamKey) {
    const client = connectionManager.getClient();
    await connectionManager.ensureStream(dlqStreamKey, config.streams.maxLen);
    this.dlqStreams.set(queueConfig.stream, {
      mainStream: queueConfig.stream,
      dlqStream: dlqStreamKey,
      maxRetries: queueConfig.maxRetries,
      config: queueConfig
    });
    console.log(`[DLQManager] Initialized DLQ for ${queueConfig.stream}`);
  }

  async sendToDLQ(message, error, metadata = {}) {
    const dlqInfo = this.dlqStreams.get(metadata.stream);
    if (!dlqInfo) {
      console.error(`[DLQManager] No DLQ configured for stream: ${metadata.stream}`);
      return false;
    }

    const client = connectionManager.getClient();
    const dlqEntry = {
      originalStream: metadata.stream,
      originalMessageId: metadata.messageId,
      originalData: JSON.stringify(metadata.data),
      error: error.message,
      errorStack: error.stack || '',
      retryCount: metadata.retryCount || 0,
      failedAt: new Date().toISOString(),
      metadata: JSON.stringify(metadata.metadata || {})
    };

    try {
      const messageId = await client.xadd(
        dlqInfo.dlqStream,
        'MAXLEN',
        '~',
        config.streams.maxLen.toString(),
        '*',
        ...this.flattenObject(dlqEntry)
      );

      console.log(`[DLQManager] Message moved to DLQ: ${dlqInfo.dlqStream} (ID: ${messageId})`);
      return messageId;
    } catch (err) {
      console.error(`[DLQManager] Failed to send to DLQ:`, err);
      return false;
    }
  }

  flattenObject(obj, prefix = '') {
    const result = [];
    for (const [key, value] of Object.entries(obj)) {
      const fieldName = prefix ? `${prefix}:${key}` : key;
      if (typeof value === 'object' && value !== null) {
        result.push(...this.flattenObject(value, fieldName));
      } else {
        result.push(fieldName, String(value));
      }
    }
    return result;
  }

  async getDLQMessages(dlqStreamKey, count = 10) {
    const client = connectionManager.getClient();
    try {
      const messages = await client.xrange(dlqStreamKey, '-', '+', 'COUNT', count);
      return messages.map(msg => this.parseDLQEntry(msg));
    } catch (error) {
      console.error(`[DLQManager] Failed to get DLQ messages:`, error);
      return [];
    }
  }

  parseDLQEntry(entry) {
    const [id, fields] = entry;
    const data = {};
    for (let i = 0; i < fields.length; i += 2) {
      const key = fields[i];
      let value = fields[i + 1];
      try {
        if (key.includes('Data') || key.includes('metadata')) {
          value = JSON.parse(value);
        }
      } catch (e) {
      }
      data[key] = value;
    }
    return { id, ...data };
  }

  async getDLQStats(dlqStreamKey) {
    const client = connectionManager.getClient();
    try {
      const length = await client.xlen(dlqStreamKey);
      const info = await connectionManager.getStreamInfo(dlqStreamKey);
      return {
        stream: dlqStreamKey,
        length,
        firstEntry: info?.firstEntry,
        lastEntry: info?.lastEntry
      };
    } catch (error) {
      return { stream: dlqStreamKey, length: 0, error: error.message };
    }
  }

  async reprocessDLQ(dlqStreamKey, mainStream) {
    const client = connectionManager.getClient();
    const messages = await this.getDLQMessages(dlqStreamKey, 100);

    let reprocessed = 0;
    for (const message of messages) {
      try {
        const originalData = message.originalData;
        if (originalData) {
          const data = JSON.parse(originalData);
          await client.xadd(
            mainStream,
            'MAXLEN',
            '~',
            config.streams.maxLen.toString(),
            '*',
            ...this.flattenObject(data)
          );
          await client.xdel(dlqStreamKey, message.id);
          reprocessed++;
        }
      } catch (error) {
        console.error(`[DLQManager] Failed to reprocess message ${message.id}:`, error);
      }
    }

    console.log(`[DLQManager] Reprocessed ${reprocessed} messages from DLQ`);
    return reprocessed;
  }

  async purgeDLQ(dlqStreamKey) {
    const client = connectionManager.getClient();
    try {
      await client.del(dlqStreamKey);
      console.log(`[DLQManager] Purged DLQ: ${dlqStreamKey}`);
      return true;
    } catch (error) {
      console.error(`[DLQManager] Failed to purge DLQ:`, error);
      return false;
    }
  }

  async getAllDLQStats() {
    const stats = {};
    for (const [streamKey, dlqInfo] of this.dlqStreams.entries()) {
      stats[streamKey] = await this.getDLQStats(dlqInfo.dlqStream);
    }
    return stats;
  }

  registerProcessor(name, handler) {
    this.processors.set(name, handler);
  }

  async processDLQ(name, options = {}) {
    const processor = this.processors.get(name);
    if (!processor) {
      throw new Error(`No processor registered for: ${name}`);
    }

    const dlqInfo = Array.from(this.dlqStreams.values()).find(d => d.mainStream === options.stream);
    if (!dlqInfo) {
      throw new Error(`No DLQ found for stream: ${options.stream}`);
    }

    const messages = await this.getDLQMessages(dlqInfo.dlqStream, options.count || 10);
    const results = { processed: 0, failed: 0 };

    for (const message of messages) {
      try {
        await processor(message);
        await this.acknowledgeDLQMessage(dlqInfo.dlqStream, message.id);
        results.processed++;
      } catch (error) {
        console.error(`[DLQManager] Failed to process message ${message.id}:`, error);
        results.failed++;
      }
    }

    return results;
  }

  async acknowledgeDLQMessage(dlqStreamKey, messageId) {
    const client = connectionManager.getClient();
    await client.xdel(dlqStreamKey, messageId);
  }
}

const deadLetterQueueManager = new DeadLetterQueueManager();

module.exports = deadLetterQueueManager;
