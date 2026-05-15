const EventEmitter = require('events');

class MessageQueue extends EventEmitter {
  constructor(options = {}) {
    super();
    this.queues = new Map();
    this.subscribers = new Map();
    this.messages = new Map();
    this.maxRetries = options.maxRetries || 3;
    this.retryDelay = options.retryDelay || 5000;
    this.messageIdCounter = 0;
    this.persistent = options.persistent || false;
    this.storagePath = options.storagePath || './queue-storage';
  }

  createQueue(name, options = {}) {
    if (this.queues.has(name)) {
      return this.queues.get(name);
    }

    const queue = {
      name,
      durable: options.durable !== false,
      autoDelete: options.autoDelete || false,
      exclusive: options.exclusive || false,
      maxLength: options.maxLength || null,
      messages: [],
      subscribers: [],
      stats: {
        totalMessages: 0,
        processedMessages: 0,
        failedMessages: 0,
        avgProcessingTime: 0
      }
    };

    this.queues.set(name, queue);
    this.emit('queue:created', { queue: name });

    return queue;
  }

  async publish(queueName, message, options = {}) {
    const queue = this.queues.get(queueName);

    if (!queue) {
      throw new Error(`Queue ${queueName} does not exist`);
    }

    if (queue.maxLength && queue.messages.length >= queue.maxLength) {
      if (options.rejectOnFull) {
        throw new Error(`Queue ${queueName} is full`);
      }
      queue.messages.shift();
    }

    const messageId = `${queueName}-${++this.messageIdCounter}-${Date.now()}`;

    const msg = {
      id: messageId,
      payload: message,
      timestamp: new Date().toISOString(),
      headers: options.headers || {},
      properties: {
        persistent: options.persistent !== false,
        priority: options.priority || 0,
        replyTo: options.replyTo || null,
        correlationId: options.correlationId || null,
        expiration: options.expiration || null,
        deliveryMode: options.persistent !== false ? 2 : 1
      },
      metadata: {
        retryCount: 0,
        publishedAt: Date.now()
      }
    };

    queue.messages.push(msg);
    queue.stats.totalMessages++;

    this.emit('message:published', { queue: queueName, messageId, message: msg });

    if (this.persistent) {
      await this.persistMessage(queueName, msg);
    }

    this.deliverToSubscribers(queueName);

    return messageId;
  }

  async subscribe(queueName, handler, options = {}) {
    const queue = this.queues.get(queueName);

    if (!queue) {
      throw new Error(`Queue ${queueName} does not exist`);
    }

    const subscriptionId = `${queueName}-sub-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

    const subscription = {
      id: subscriptionId,
      handler,
      options: {
        ack: options.ack || false,
        prefetch: options.prefetch || 1,
        exclusive: options.exclusive || false
      },
      active: true,
      currentMessage: null
    };

    queue.subscribers.push(subscription);
    this.emit('subscription:created', { queue: queueName, subscriptionId });

    return subscriptionId;
  }

  async unsubscribe(queueName, subscriptionId) {
    const queue = this.queues.get(queueName);

    if (!queue) {
      return false;
    }

    const index = queue.subscribers.findIndex(s => s.id === subscriptionId);
    if (index === -1) {
      return false;
    }

    queue.subscribers.splice(index, 1);
    this.emit('subscription:removed', { queue: queueName, subscriptionId });

    return true;
  }

  deliverToSubscribers(queueName) {
    const queue = this.queues.get(queueName);
    if (!queue || queue.messages.length === 0) return;

    for (const subscriber of queue.subscribers) {
      if (!subscriber.active) continue;

      const message = queue.messages[0];

      if (
        message.properties.expiration &&
        Date.now() > message.metadata.publishedAt + message.properties.expiration
      ) {
        queue.messages.shift();
        this.emit('message:expired', { queue: queueName, messageId: message.id });
        continue;
      }

      (async () => {
        try {
          const startTime = Date.now();

          if (subscriber.options.ack) {
            subscriber.currentMessage = message;
          } else {
            await handler(message);
          }

          const processingTime = Date.now() - startTime;
          queue.stats.processedMessages++;
          queue.stats.avgProcessingTime =
            (queue.stats.avgProcessingTime * (queue.stats.processedMessages - 1) + processingTime) /
            queue.stats.processedMessages;

          if (!subscriber.options.ack) {
            queue.messages.shift();
          }

          this.emit('message:processed', {
            queue: queueName,
            messageId: message.id,
            processingTime
          });
        } catch (error) {
          console.error(`Error processing message ${message.id}:`, error);

          message.metadata.retryCount++;

          if (message.metadata.retryCount >= this.maxRetries) {
            queue.messages.shift();
            queue.stats.failedMessages++;
            this.emit('message:failed', {
              queue: queueName,
              messageId: message.id,
              error: error.message,
              retries: message.metadata.retryCount
            });
          } else {
            setTimeout(() => {
              this.deliverToSubscribers(queueName);
            }, this.retryDelay * message.metadata.retryCount);
          }
        }
      })();
    }
  }

  async ack(queueName, messageId) {
    const queue = this.queues.get(queueName);
    if (!queue) return false;

    const messageIndex = queue.messages.findIndex(m => m.id === messageId);
    if (messageIndex === -1) return false;

    queue.messages.splice(messageIndex, 1);
    this.emit('message:acked', { queue: queueName, messageId });

    return true;
  }

  async nack(queueName, messageId, requeue = true) {
    const queue = this.queues.get(queueName);
    if (!queue) return false;

    const message = queue.messages.find(m => m.id === messageId);
    if (!message) return false;

    message.metadata.retryCount++;

    if (requeue && message.metadata.retryCount < this.maxRetries) {
      this.emit('message:nacked', { queue: queueName, messageId, requeue: true });
    } else {
      queue.messages.splice(queue.messages.indexOf(message), 1);
      queue.stats.failedMessages++;
      this.emit('message:nacked', { queue: queueName, messageId, requeue: false });
    }

    return true;
  }

  purgeQueue(queueName) {
    const queue = this.queues.get(queueName);
    if (!queue) return 0;

    const count = queue.messages.length;
    queue.messages = [];
    this.emit('queue:purged', { queue: queueName, messageCount: count });

    return count;
  }

  deleteQueue(queueName) {
    if (!this.queues.has(queueName)) {
      return false;
    }

    const queue = this.queues.get(queueName);

    if (queue.subscribers.length > 0 && !queue.autoDelete) {
      throw new Error('Cannot delete queue with active subscribers');
    }

    this.queues.delete(queueName);
    this.emit('queue:deleted', { queue: queueName });

    return true;
  }

  getQueueStats(queueName) {
    const queue = this.queues.get(queueName);
    if (!queue) return null;

    return {
      name: queue.name,
      messageCount: queue.messages.length,
      subscriberCount: queue.subscribers.length,
      ...queue.stats
    };
  }

  getAllStats() {
    const stats = {};

    for (const [name, queue] of this.queues) {
      stats[name] = this.getQueueStats(name);
    }

    return stats;
  }

  async persistMessage(queueName, message) {
    if (!this.persistent) return;

    try {
      if (!this.messages.has(queueName)) {
        this.messages.set(queueName, []);
      }

      this.messages.get(queueName).push(message);
    } catch (error) {
      console.error('Failed to persist message:', error);
    }
  }

  async recover() {
    if (!this.persistent) return;

    for (const [queueName, messages] of this.messages) {
      const queue = this.queues.get(queueName);
      if (queue) {
        queue.messages.push(...messages);
      }
    }

    this.emit('queue:recovered');
  }
}

const messageQueue = new MessageQueue({
  persistent: false,
  maxRetries: 3,
  retryDelay: 5000
});

module.exports = messageQueue;
module.exports.MessageQueue = MessageQueue;
