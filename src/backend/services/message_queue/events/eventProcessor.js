const eventPublisher = require('./eventPublisher');
const eventSubscriber = require('./eventSubscriber');
const { EventTypes } = require('./eventTypes');

class EventHandlerRegistry {
  constructor() {
    this.handlers = new Map();
  }

  register(eventType, handler) {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, []);
    }
    this.handlers.get(eventType).push(handler);
    console.log(`[EventHandlerRegistry] Registered handler for: ${eventType}`);
  }

  unregister(eventType, handler) {
    const handlers = this.handlers.get(eventType);
    if (handlers) {
      const index = handlers.indexOf(handler);
      if (index > -1) {
        handlers.splice(index, 1);
      }
    }
  }

  getHandlers(eventType) {
    return this.handlers.get(eventType) || [];
  }

  async handle(event) {
    const handlers = this.getHandlers(event.type);
    const results = [];

    for (const handler of handlers) {
      try {
        const result = await handler(event);
        results.push({ success: true, result });
      } catch (error) {
        console.error(`[EventHandlerRegistry] Handler error for ${event.type}:`, error);
        results.push({ success: false, error: error.message });
      }
    }

    return results;
  }
}

class EventProcessor {
  constructor() {
    this.registry = new EventHandlerRegistry();
    this.isProcessing = false;
  }

  async initialize() {
    await this.setupDefaultHandlers();
    await eventSubscriber.initialize();
    await this.startProcessing();
    console.log('[EventProcessor] Initialized');
  }

  async setupDefaultHandlers() {
    this.registry.register(EventTypes.USER_CREATED, async (event) => {
      console.log(`[EventProcessor] User created: ${event.data.userId}`);
    });

    this.registry.register(EventTypes.USER_LOGGED_IN, async (event) => {
      console.log(`[EventProcessor] User logged in: ${event.data.userId}`);
    });

    this.registry.register(EventTypes.NOTIFICATION_CREATED, async (event) => {
      console.log(`[EventProcessor] Notification created: ${event.data.notificationId}`);
    });

    this.registry.register(EventTypes.EXPORT_COMPLETED, async (event) => {
      console.log(`[EventProcessor] Export completed: ${event.data.exportId}`);
    });

    this.registry.register(EventTypes.SECURITY_EVENT, async (event) => {
      console.log(`[EventProcessor] Security event: ${event.data.eventType}`);
    });
  }

  async startProcessing() {
    if (this.isProcessing) {
      return;
    }

    this.isProcessing = true;

    await eventSubscriber.subscribeToPattern(`${process.env.EVENTS_PUBSUB_PREFIX || 'hjtpx:events:'}*', async (event) => {
      await this.registry.handle(event);
    });

    await eventSubscriber.startListening();
    console.log('[EventProcessor] Started processing events');
  }

  async stopProcessing() {
    this.isProcessing = false;
    await eventSubscriber.stopListening();
    console.log('[EventProcessor] Stopped processing events');
  }

  registerHandler(eventType, handler) {
    this.registry.register(eventType, handler);
  }

  unregisterHandler(eventType, handler) {
    this.registry.unregister(eventType, handler);
  }

  async publishEvent(event) {
    return await eventPublisher.publish(event);
  }

  async healthCheck() {
    return {
      healthy: this.isProcessing,
      registeredHandlers: this.registry.handlers.size,
      isProcessing: this.isProcessing
    };
  }
}

const eventProcessor = new EventProcessor();

module.exports = {
  EventHandlerRegistry,
  EventProcessor,
  eventProcessor
};
