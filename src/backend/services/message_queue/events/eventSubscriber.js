const config = require('../../config/messageQueue');
const connectionManager = require('../connectionManager');

const { Event, EventTypes } = require('./eventTypes');

class EventSubscription {
  constructor(eventType, handler, options = {}) {
    this.eventType = eventType;
    this.handler = handler;
    this.options = {
      channel: options.channel || `${config.events.pubSubPrefix}${eventType}`,
      pattern: options.pattern || null,
      filter: options.filter || null,
      concurrency: options.concurrency || 1,
      ...options
    };
    this.isActive = false;
    this.subscriptionId = null;
  }

  async activate(client) {
    if (this.isActive) {
      return;
    }

    if (this.options.pattern) {
      const pattern = await client.psubscribe(this.options.channel);
      this.subscriptionId = this.options.channel;
    } else {
      const subscription = await client.subscribe(this.options.channel);
      this.subscriptionId = this.options.channel;
    }

    this.isActive = true;
    console.log(`[EventSubscription] Activated: ${this.eventType}`);
  }

  async deactivate(client) {
    if (!this.isActive) {
      return;
    }

    if (this.options.pattern) {
      await client.punsubscribe(this.options.channel);
    } else {
      await client.unsubscribe(this.options.channel);
    }

    this.isActive = false;
    console.log(`[EventSubscription] Deactivated: ${this.eventType}`);
  }

  async handleMessage(message) {
    if (this.options.filter) {
      const shouldProcess = await this.options.filter(message);
      if (!shouldProcess) {
        return;
      }
    }

    try {
      await this.handler(message);
    } catch (error) {
      console.error(`[EventSubscription] Handler error for ${this.eventType}:`, error);
      if (this.options.onError) {
        await this.options.onError(error, message);
      }
    }
  }
}

class EventSubscriber {
  constructor() {
    this.subscriptions = new Map();
    this.client = null;
    this.isListening = false;
    this.messageHandlers = new Map();
  }

  async initialize() {
    this.client = connectionManager.getClient();
    console.log('[EventSubscriber] Initialized');
  }

  async subscribe(eventType, handler, options = {}) {
    if (!this.client) {
      await this.initialize();
    }

    const channel = options.channel || `${config.events.pubSubPrefix}${eventType}`;
    const subscription = new EventSubscription(eventType, handler, options);

    this.subscriptions.set(eventType, subscription);
    this.messageHandlers.set(channel, async (channel, message) => {
      const event = this.deserializeEvent(message);
      const sub = this.subscriptions.get(eventType);
      if (sub) {
        await sub.handleMessage(event);
      }
    });

    await subscription.activate(this.client);
    return subscription;
  }

  async subscribeToPattern(pattern, handler, options = {}) {
    if (!this.client) {
      await this.initialize();
    }

    const subscription = new EventSubscription(null, handler, {
      ...options,
      pattern: true,
      channel: pattern
    });

    this.subscriptions.set(pattern, subscription);
    this.messageHandlers.set(pattern, async (pattern, channel, message) => {
      const event = this.deserializeEvent(message);
      await subscription.handleMessage(event);
    });

    await subscription.activate(this.client);
    return subscription;
  }

  async unsubscribe(eventType) {
    const subscription = this.subscriptions.get(eventType);
    if (subscription) {
      await subscription.deactivate(this.client);
      this.subscriptions.delete(eventType);
    }
  }

  async unsubscribeAll() {
    for (const [eventType] of this.subscriptions) {
      await this.unsubscribe(eventType);
    }
  }

  deserializeEvent(message) {
    try {
      const data = JSON.parse(message);
      return new Event(data.type, data.data, {
        id: data.id,
        source: data.source,
        correlationId: data.correlationId,
        causationId: data.causationId,
        metadata: data.metadata,
        version: data.version
      });
    } catch (error) {
      console.error('[EventSubscriber] Failed to deserialize event:', error);
      return message;
    }
  }

  async startListening() {
    if (this.isListening || !this.client) {
      return;
    }

    this.isListening = true;

    this.client.on('message', async (channel, message) => {
      const handler = this.messageHandlers.get(channel);
      if (handler) {
        await handler(channel, message);
      }
    });

    this.client.on('pmessage', async (pattern, channel, message) => {
      const handler = this.messageHandlers.get(pattern);
      if (handler) {
        await handler(pattern, channel, message);
      }
    });

    console.log('[EventSubscriber] Started listening for events');
  }

  async stopListening() {
    this.isListening = false;
    await this.unsubscribeAll();
    console.log('[EventSubscriber] Stopped listening');
  }

  getSubscription(eventType) {
    return this.subscriptions.get(eventType);
  }

  getActiveSubscriptions() {
    return Array.from(this.subscriptions.values()).filter(sub => sub.isActive);
  }

  async healthCheck() {
    return {
      healthy: this.isListening,
      activeSubscriptions: this.subscriptions.size,
      isListening: this.isListening
    };
  }
}

const eventSubscriber = new EventSubscriber();

module.exports = {
  EventSubscription,
  EventSubscriber,
  eventSubscriber
};
