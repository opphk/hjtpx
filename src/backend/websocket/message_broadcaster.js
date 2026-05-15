let logger;
try {
  const loggerModule = require('../middleware/logger');
  logger = loggerModule.logger || loggerModule;
} catch (error) {
  logger = {
    info: () => {},
    error: () => {},
    warn: () => {},
    debug: () => {}
  };
}

if (!logger.info) logger.info = () => {};
if (!logger.error) logger.error = () => {};
if (!logger.warn) logger.warn = () => {};
if (!logger.debug) logger.debug = () => {};

class MessageBroadcaster {
  constructor() {
    this.io = null;
    this.connectionManager = null;
    this.messageHandlers = new Map();
    this.messageHistory = new Map();
    this.maxHistoryPerUser = 100;
    this.maxHistoryPerRoom = 500;
    this.stats = {
      totalMessages: 0,
      systemMessages: 0,
      groupMessages: 0,
      privateMessages: 0,
      broadcastMessages: 0,
      failedMessages: 0,
      byType: {}
    };
  }

  initialize(io, connectionManager) {
    this.io = io;
    this.connectionManager = connectionManager;
    this.setupDefaultHandlers();
    logger.info('Message broadcaster initialized');
  }

  setupDefaultHandlers() {
    this.registerHandler('text', this.handleTextMessage.bind(this));
    this.registerHandler('system', this.handleSystemMessage.bind(this));
    this.registerHandler('notification', this.handleNotificationMessage.bind(this));
    this.registerHandler('alert', this.handleAlertMessage.bind(this));
    this.registerHandler('data_update', this.handleDataUpdateMessage.bind(this));
  }

  registerHandler(type, handler) {
    this.messageHandlers.set(type, handler);
    logger.debug(`Message handler registered for type: ${type}`);
  }

  sendPrivateMessage(fromUserId, toUserId, message) {
    this.stats.totalMessages++;
    this.stats.privateMessages++;

    const messageData = {
      id: this.generateMessageId(),
      type: 'private',
      messageType: message.type || 'text',
      from: fromUserId,
      to: toUserId,
      content: message.content,
      metadata: message.metadata || {},
      timestamp: new Date(),
      status: 'sent'
    };

    try {
      const delivered = this.deliverToUser(toUserId, 'message', messageData);

      if (!delivered) {
        messageData.status = 'queued';
        this.queueMessage(toUserId, messageData);
      }

      this.addToHistory(fromUserId, messageData);
      this.updateStats(messageData);

      logger.info('Private message sent', {
        messageId: messageData.id,
        from: fromUserId,
        to: toUserId,
        delivered
      });

      const ackData = {
        messageId: messageData.id,
        status: delivered ? 'delivered' : 'queued',
        timestamp: new Date()
      };

      this.deliverToUser(fromUserId, 'message:ack', ackData);

      return messageData;
    } catch (error) {
      this.stats.failedMessages++;
      logger.error('Failed to send private message', {
        from: fromUserId,
        to: toUserId,
        error: error.message
      });
      throw error;
    }
  }

  sendGroupMessage(fromUserId, room, message) {
    this.stats.totalMessages++;
    this.stats.groupMessages++;

    const messageData = {
      id: this.generateMessageId(),
      type: 'group',
      messageType: message.type || 'text',
      from: fromUserId,
      room,
      content: message.content,
      metadata: message.metadata || {},
      timestamp: new Date()
    };

    try {
      this.io.to(room).emit('message', messageData);

      this.addToRoomHistory(room, messageData);
      this.updateStats(messageData);

      logger.info('Group message sent', {
        messageId: messageData.id,
        room,
        from: fromUserId,
        memberCount: this.connectionManager.getRoomMemberCount(room)
      });

      return messageData;
    } catch (error) {
      this.stats.failedMessages++;
      logger.error('Failed to send group message', {
        room,
        from: fromUserId,
        error: error.message
      });
      throw error;
    }
  }

  broadcastSystemMessage(message) {
    this.stats.totalMessages++;
    this.stats.systemMessages++;

    const messageData = {
      id: this.generateMessageId(),
      type: 'system',
      messageType: 'system',
      content: message.content,
      priority: message.priority || 'normal',
      metadata: message.metadata || {},
      timestamp: new Date()
    };

    try {
      if (message.room) {
        this.io.to(message.room).emit('message', messageData);
      } else {
        this.io.emit('message', messageData);
      }

      this.updateStats(messageData);

      logger.info('System message broadcast', {
        messageId: messageData.id,
        room: message.room || 'global'
      });

      return messageData;
    } catch (error) {
      this.stats.failedMessages++;
      logger.error('Failed to broadcast system message', {
        error: error.message
      });
      throw error;
    }
  }

  broadcastToChannel(channel, fromUserId, message) {
    this.stats.totalMessages++;
    this.stats.broadcastMessages++;

    const messageData = {
      id: this.generateMessageId(),
      type: 'channel',
      messageType: message.type || 'text',
      channel,
      from: fromUserId,
      content: message.content,
      metadata: message.metadata || {},
      timestamp: new Date()
    };

    try {
      this.io.to(`channel:${channel}`).emit('message', messageData);

      this.updateStats(messageData);

      logger.info('Channel message broadcast', {
        messageId: messageData.id,
        channel,
        from: fromUserId
      });

      return messageData;
    } catch (error) {
      this.stats.failedMessages++;
      logger.error('Failed to broadcast to channel', {
        channel,
        from: fromUserId,
        error: error.message
      });
      throw error;
    }
  }

  deliverToUser(userId, event, data) {
    const userConnections = this.connectionManager.getUserConnections(userId);

    if (userConnections.length === 0) {
      return false;
    }

    userConnections.forEach(connection => {
      this.io.to(connection.socketId).emit(event, data);
    });

    return true;
  }

  queueMessage(userId, message) {
    if (!this.messageHistory.has(userId)) {
      this.messageHistory.set(userId, {
        queued: [],
        history: []
      });
    }

    const userHistory = this.messageHistory.get(userId);
    userHistory.queued.push({
      ...message,
      queuedAt: new Date()
    });

    logger.debug('Message queued', {
      userId,
      messageId: message.id,
      queueSize: userHistory.queued.length
    });
  }

  addToHistory(userId, message) {
    if (!this.messageHistory.has(userId)) {
      this.messageHistory.set(userId, {
        queued: [],
        history: []
      });
    }

    const userHistory = this.messageHistory.get(userId);
    userHistory.history.push(message);

    if (userHistory.history.length > this.maxHistoryPerUser) {
      userHistory.history = userHistory.history.slice(-this.maxHistoryPerUser);
    }
  }

  addToRoomHistory(room, message) {
    const roomKey = `room:${room}`;

    if (!this.messageHistory.has(roomKey)) {
      this.messageHistory.set(roomKey, []);
    }

    const roomHistory = this.messageHistory.get(roomKey);
    roomHistory.push(message);

    if (roomHistory.length > this.maxHistoryPerRoom) {
      this.messageHistory.set(roomKey, roomHistory.slice(-this.maxHistoryPerRoom));
    }
  }

  getUserHistory(userId, options = {}) {
    const userHistory = this.messageHistory.get(userId);
    if (!userHistory) return { queued: [], history: [] };

    let history = userHistory.history;

    if (options.type) {
      history = history.filter(m => m.type === options.type);
    }

    if (options.messageType) {
      history = history.filter(m => m.messageType === options.messageType);
    }

    if (options.since) {
      history = history.filter(m => new Date(m.timestamp) >= new Date(options.since));
    }

    if (options.limit) {
      history = history.slice(-options.limit);
    }

    return {
      queued: userHistory.queued,
      history: history.reverse()
    };
  }

  getRoomHistory(room, options = {}) {
    const roomKey = `room:${room}`;
    let history = this.messageHistory.get(roomKey) || [];

    if (options.since) {
      history = history.filter(m => new Date(m.timestamp) >= new Date(options.since));
    }

    if (options.limit) {
      history = history.slice(-options.limit);
    }

    return history.reverse();
  }

  handleTextMessage(message) {
    logger.debug('Handling text message', {
      messageId: message.id,
      from: message.from
    });
  }

  handleSystemMessage(message) {
    logger.info('System message received', {
      messageId: message.id,
      content: message.content
    });
  }

  handleNotificationMessage(message) {
    logger.debug('Handling notification message', {
      messageId: message.id,
      metadata: message.metadata
    });
  }

  handleAlertMessage(message) {
    logger.warn('Alert message received', {
      messageId: message.id,
      priority: message.priority,
      content: message.content
    });
  }

  handleDataUpdateMessage(message) {
    logger.debug('Handling data update message', {
      messageId: message.id,
      metadata: message.metadata
    });
  }

  updateStats(message) {
    const type = message.type || 'unknown';
    if (!this.stats.byType[type]) {
      this.stats.byType[type] = 0;
    }
    this.stats.byType[type]++;
  }

  generateMessageId() {
    return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  getStats() {
    return {
      ...this.stats,
      rooms: Array.from(this.messageHistory.keys())
        .filter(k => k.startsWith('room:'))
        .map(k => ({
          room: k.replace('room:', ''),
          messageCount: this.messageHistory.get(k).length
        })),
      usersWithHistory: this.messageHistory.size
    };
  }

  cleanup() {
    this.messageHistory.clear();
    this.messageHandlers.clear();
    this.stats = {
      totalMessages: 0,
      systemMessages: 0,
      groupMessages: 0,
      privateMessages: 0,
      broadcastMessages: 0,
      failedMessages: 0,
      byType: {}
    };
    logger.info('Message broadcaster cleaned up');
  }
}

const messageBroadcaster = new MessageBroadcaster();

module.exports = messageBroadcaster;
module.exports.MessageBroadcaster = MessageBroadcaster;
