const jwt = require('jsonwebtoken');
const { Server } = require('socket.io');

const { logger } = require('../middleware/logger');
const connectionManager = require('./connection_manager');
const notificationSystem = require('./notification_system');
const onlineStatusManager = require('./online_status_manager');
const messageBroadcaster = require('./message_broadcaster');
const heartbeatSystem = require('./heartbeat_system');

class WebSocketServer {
  constructor(httpServer) {
    this.io = new Server(httpServer, {
      cors: {
        origin: process.env.CORS_ORIGIN || '*',
        methods: ['GET', 'POST'],
        credentials: true
      },
      pingTimeout: 30000,
      pingInterval: 15000,
      transports: ['websocket', 'polling'],
      maxHttpBufferSize: 1e7,
      perMessageDeflate: {
        threshold: 1024,
        serverNoContextTakeover: true,
        clientNoContextTakeover: true,
        serverMaxWindowBits: 10,
        clientMaxWindowBits: 10,
        memLevel: 7,
        level: 6
      }
    });

    this.connectedClients = new Map();
    this.roomSubscriptions = new Map();

    this.initializeSubsystems();

    this.metrics = {
      totalConnections: 0,
      totalDisconnections: 0,
      messagesSent: 0,
      messagesReceived: 0,
      errors: 0,
      connectionTimes: [],
      startTime: Date.now()
    };

    this.setupMiddleware();
    this.setupEventHandlers();
  }

  initializeSubsystems() {
    connectionManager.initialize ? connectionManager.initialize() : null;
    notificationSystem.initialize(this.io, connectionManager);
    onlineStatusManager.initialize(this.io, connectionManager);
    messageBroadcaster.initialize(this.io, connectionManager);
    heartbeatSystem.initialize(this.io, connectionManager);

    logger.info('WebSocket subsystems initialized');
  }

  setupMiddleware() {
    this.io.use(async (socket, next) => {
      try {
        const token = socket.handshake.auth.token || socket.handshake.query.token;

        if (!token) {
          return next(new Error('Authentication token required'));
        }

        const decoded = jwt.verify(token, process.env.JWT_SECRET || 'your-secret-key');
        socket.userId = decoded.id;
        socket.user = decoded;
        next();
      } catch (error) {
        logger.error('WebSocket authentication failed', { error: error.message });
        next(new Error('Authentication failed'));
      }
    });
  }

  setupEventHandlers() {
    this.io.on('connection', socket => {
      this.handleConnection(socket);
    });
  }

  handleConnection(socket) {
    const clientInfo = {
      socketId: socket.id,
      userId: socket.userId,
      connectedAt: new Date(),
      rooms: []
    };

    this.connectedClients.set(socket.id, clientInfo);
    this.metrics.totalConnections++;
    connectionManager.addConnection(socket);

    logger.info('Client connected', {
      socketId: socket.id,
      userId: socket.userId
    });

    socket.emit('connected', {
      socketId: socket.id,
      message: 'Successfully connected to WebSocket server'
    });

    socket.emit('online_status', {
      status: 'online',
      timestamp: new Date()
    });

    this.setupSocketEventHandlers(socket);

    onlineStatusManager.setUserOnline(socket.userId, socket.id, {
      socketId: socket.id,
      connectedAt: clientInfo.connectedAt
    });

    this.broadcastUserOnlineStatus(socket.userId, true);
  }

  setupSocketEventHandlers(socket) {
    socket.on('disconnect', reason => {
      this.handleDisconnection(socket, reason);
    });

    socket.on('join', (room, callback) => {
      this.handleJoinRoom(socket, room, callback);
    });

    socket.on('leave', (room, callback) => {
      this.handleLeaveRoom(socket, room, callback);
    });

    socket.on('subscribe', (channel, callback) => {
      this.handleSubscribe(socket, channel, callback);
    });

    socket.on('unsubscribe', (channel, callback) => {
      this.handleUnsubscribe(socket, channel, callback);
    });

    socket.on('message', (data, callback) => {
      this.handleMessage(socket, data, callback);
    });

    socket.on('broadcast', (data, callback) => {
      this.handleBroadcast(socket, data, callback);
    });

    socket.on('private_message', (data, callback) => {
      this.handlePrivateMessage(socket, data, callback);
    });

    socket.on('group_message', (data, callback) => {
      this.handleGroupMessage(socket, data, callback);
    });

    socket.on('heartbeat', () => {
      this.handleHeartbeat(socket);
    });

    socket.on('ping', () => {
      socket.emit('pong', { timestamp: Date.now() });
    });

    socket.on('status_update', (data, callback) => {
      this.handleStatusUpdate(socket, data, callback);
    });

    socket.on('get:online_users', callback => {
      if (callback && typeof callback === 'function') {
        callback({
          success: true,
          users: onlineStatusManager.getOnlineUsers()
        });
      }
    });

    socket.on('get:history', (data, callback) => {
      this.handleGetHistory(socket, data, callback);
    });

    socket.on('get:metrics', callback => {
      if (callback && typeof callback === 'function') {
        callback({ success: true, metrics: this.getDetailedMetrics() });
      }
    });

    socket.on('error', error => {
      this.metrics.errors++;
      logger.error('Socket error', {
        socketId: socket.id,
        userId: socket.userId,
        error: error.message
      });
    });
  }

  handleJoinRoom(socket, room, callback) {
    try {
      socket.join(room);

      const clientInfo = this.connectedClients.get(socket.id);
      if (clientInfo && !clientInfo.rooms.includes(room)) {
        clientInfo.rooms.push(room);
      }

      connectionManager.addRoomConnection(room, socket.id, socket.userId);

      logger.info('Client joined room', {
        socketId: socket.id,
        userId: socket.userId,
        room
      });

      if (callback && typeof callback === 'function') {
        callback({ success: true, room });
      }

      socket.to(room).emit('user:joined', {
        userId: socket.userId,
        room,
        timestamp: new Date()
      });
    } catch (error) {
      logger.error('Error joining room', { error: error.message });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handleLeaveRoom(socket, room, callback) {
    try {
      socket.leave(room);

      const clientInfo = this.connectedClients.get(socket.id);
      if (clientInfo) {
        clientInfo.rooms = clientInfo.rooms.filter(r => r !== room);
      }

      connectionManager.removeRoomConnection(room, socket.id);

      logger.info('Client left room', {
        socketId: socket.id,
        userId: socket.userId,
        room
      });

      if (callback && typeof callback === 'function') {
        callback({ success: true, room });
      }

      socket.to(room).emit('user:left', {
        userId: socket.userId,
        room,
        timestamp: new Date()
      });
    } catch (error) {
      logger.error('Error leaving room', { error: error.message });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handleSubscribe(socket, channel, callback) {
    try {
      socket.join(`channel:${channel}`);

      if (!this.roomSubscriptions.has(channel)) {
        this.roomSubscriptions.set(channel, new Set());
      }
      this.roomSubscriptions.get(channel).add(socket.userId);

      logger.info('Client subscribed to channel', {
        socketId: socket.id,
        userId: socket.userId,
        channel
      });

      if (callback && typeof callback === 'function') {
        callback({ success: true, channel });
      }
    } catch (error) {
      logger.error('Error subscribing to channel', { error: error.message });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handleUnsubscribe(socket, channel, callback) {
    try {
      socket.leave(`channel:${channel}`);

      if (this.roomSubscriptions.has(channel)) {
        this.roomSubscriptions.get(channel).delete(socket.userId);
      }

      logger.info('Client unsubscribed from channel', {
        socketId: socket.id,
        userId: socket.userId,
        channel
      });

      if (callback && typeof callback === 'function') {
        callback({ success: true, channel });
      }
    } catch (error) {
      logger.error('Error unsubscribing from channel', { error: error.message });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handleMessage(socket, data, callback) {
    try {
      this.metrics.messagesReceived++;

      const handler = messageBroadcaster.messageHandlers.get(data.type || 'text');
      if (handler) {
        handler(data);
      }

      logger.info('Message received', {
        socketId: socket.id,
        userId: socket.userId,
        type: data.type
      });

      if (callback && typeof callback === 'function') {
        callback({ success: true, received: true });
      }
    } catch (error) {
      this.metrics.errors++;
      logger.error('Error handling message', { error: error.message });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handlePrivateMessage(socket, data, callback) {
    try {
      const { to, content, type, metadata } = data;

      const message = messageBroadcaster.sendPrivateMessage(socket.userId, to, {
        content,
        type: type || 'text',
        metadata
      });

      this.metrics.messagesSent++;

      if (callback && typeof callback === 'function') {
        callback({
          success: true,
          messageId: message.id,
          timestamp: message.timestamp
        });
      }
    } catch (error) {
      this.metrics.errors++;
      logger.error('Error sending private message', {
        from: socket.userId,
        to: data.to,
        error: error.message
      });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handleGroupMessage(socket, data, callback) {
    try {
      const { room, content, type, metadata } = data;

      const message = messageBroadcaster.sendGroupMessage(socket.userId, room, {
        content,
        type: type || 'text',
        metadata
      });

      this.metrics.messagesSent++;

      if (callback && typeof callback === 'function') {
        callback({
          success: true,
          messageId: message.id,
          timestamp: message.timestamp
        });
      }
    } catch (error) {
      this.metrics.errors++;
      logger.error('Error sending group message', {
        from: socket.userId,
        room: data.room,
        error: error.message
      });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handleBroadcast(socket, data, callback) {
    try {
      const { room, message, type } = data;

      if (room) {
        this.io.to(room).emit(type || 'broadcast', {
          message,
          from: socket.userId,
          timestamp: new Date()
        });
      } else {
        socket.broadcast.emit(type || 'broadcast', {
          message,
          from: socket.userId,
          timestamp: new Date()
        });
      }

      this.metrics.messagesSent++;

      logger.info('Broadcast sent', {
        socketId: socket.id,
        userId: socket.userId,
        room,
        type
      });

      if (callback && typeof callback === 'function') {
        callback({ success: true });
      }
    } catch (error) {
      this.metrics.errors++;
      logger.error('Error sending broadcast', { error: error.message });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handleHeartbeat(socket) {
    heartbeatSystem.handleHeartbeatResponse(socket.id);
    onlineStatusManager.updateLastActivity(socket.userId);
    connectionManager.updateHeartbeat(socket.id);

    socket.emit('heartbeat:ack', {
      timestamp: Date.now(),
      serverTime: new Date().toISOString()
    });
  }

  handleStatusUpdate(socket, data, callback) {
    try {
      const { status } = data;

      const success = onlineStatusManager.updateUserStatus(socket.userId, status, {
        socketId: socket.id
      });

      if (callback && typeof callback === 'function') {
        callback({ success });
      }
    } catch (error) {
      logger.error('Error updating status', {
        socketId: socket.id,
        userId: socket.userId,
        error: error.message
      });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handleGetHistory(socket, data, callback) {
    try {
      const { type, room, limit } = data;

      let history;

      if (room) {
        history = messageBroadcaster.getRoomHistory(room, { limit });
      } else {
        history = messageBroadcaster.getUserHistory(socket.userId, {
          type,
          limit
        });
      }

      if (callback && typeof callback === 'function') {
        callback({ success: true, history });
      }
    } catch (error) {
      logger.error('Error getting history', {
        socketId: socket.id,
        userId: socket.userId,
        error: error.message
      });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handleDisconnection(socket, reason) {
    const clientInfo = this.connectedClients.get(socket.id);

    if (clientInfo) {
      const connectedDuration = Date.now() - clientInfo.connectedAt.getTime();

      logger.info('Client disconnected', {
        socketId: socket.id,
        userId: socket.userId,
        reason,
        connectedDuration
      });

      this.metrics.totalDisconnections++;
      this.metrics.connectionTimes.push(connectedDuration);

      const userConnections = connectionManager.getUserConnections(socket.userId);
      const isLastConnection = userConnections.length <= 1;

      if (isLastConnection) {
        onlineStatusManager.setUserOffline(socket.userId, reason, {
          socketId: socket.id,
          duration: connectedDuration
        });
      }

      clientInfo.rooms.forEach(room => {
        socket.to(room).emit('user:left', {
          userId: socket.userId,
          room,
          timestamp: new Date()
        });
      });

      connectionManager.removeConnection(socket.id);
      heartbeatSystem.unregisterSocketHeartbeat(socket.id, reason);

      this.connectedClients.delete(socket.id);

      this.broadcastUserOnlineStatus(socket.userId, isLastConnection ? false : undefined);
    }
  }

  broadcastUserOnlineStatus(userId, isOnline) {
    if (isOnline !== undefined) {
      this.io.emit('presence:update', {
        userId,
        isOnline,
        timestamp: new Date()
      });
    }
  }

  broadcast(event, data, room = null) {
    if (room) {
      this.io.to(room).emit(event, data);
    } else {
      this.io.emit(event, data);
    }
  }

  sendToUser(userId, event, data) {
    const userConnections = connectionManager.getUserConnections(userId);

    if (userConnections.length === 0) {
      return false;
    }

    userConnections.forEach(connection => {
      this.io.to(connection.socketId).emit(event, data);
    });

    return true;
  }

  sendToUsers(userIds, event, data) {
    userIds.forEach(userId => {
      this.sendToUser(userId, event, data);
    });
  }

  broadcastToChannel(channel, event, data) {
    this.io.to(`channel:${channel}`).emit(event, data);
  }

  getConnectedClients() {
    return Array.from(this.connectedClients.values());
  }

  getOnlineUsers() {
    return onlineStatusManager.getOnlineUsers();
  }

  getConnectionStats() {
    return {
      totalConnections: this.connectedClients.size,
      onlineUsers: onlineStatusManager.getOnlineUserCount(),
      rooms: Array.from(this.roomSubscriptions.keys()),
      subscriptions: Array.from(this.roomSubscriptions.entries()).map(([channel, users]) => ({
        channel,
        subscriberCount: users.size
      })),
      connectionManager: connectionManager.getStats(),
      onlineStatus: onlineStatusManager.getStats(),
      messageBroadcaster: messageBroadcaster.getStats(),
      heartbeat: heartbeatSystem.getStats(),
      notification: notificationSystem.getStats()
    };
  }

  getDetailedMetrics() {
    const uptime = Date.now() - this.metrics.startTime;
    const avgConnectionTime =
      this.metrics.connectionTimes.length > 0
        ? this.metrics.connectionTimes.reduce((sum, t) => sum + t, 0) /
          this.metrics.connectionTimes.length
        : 0;

    return {
      uptime,
      currentConnections: this.connectedClients.size,
      onlineUsers: onlineStatusManager.getOnlineUserCount(),
      totalConnections: this.metrics.totalConnections,
      totalDisconnections: this.metrics.totalDisconnections,
      messagesSent: this.metrics.messagesSent,
      messagesReceived: this.metrics.messagesReceived,
      errors: this.metrics.errors,
      avgConnectionTime,
      rooms: Array.from(this.roomSubscriptions.keys()),
      subscriptions: Array.from(this.roomSubscriptions.entries()).map(([channel, users]) => ({
        channel,
        subscriberCount: users.size
      })),
      subsystems: {
        connectionManager: connectionManager.getStats(),
        onlineStatus: onlineStatusManager.getStats(),
        messageBroadcaster: messageBroadcaster.getStats(),
        heartbeat: heartbeatSystem.getStats(),
        notification: notificationSystem.getStats()
      }
    };
  }

  async sendNotification(userId, notification) {
    return await notificationSystem.sendToUser(userId, notification);
  }

  async sendBulkNotifications(userIds, notification) {
    return await notificationSystem.sendToUsers(userIds, notification);
  }

  broadcastNotification(notification, room = 'notifications') {
    return notificationSystem.broadcast(notification, room);
  }

  close() {
    logger.info('Closing WebSocket server');

    heartbeatSystem.cleanup();
    onlineStatusManager.cleanup();
    messageBroadcaster.cleanup();
    notificationSystem.cleanup();
    connectionManager.cleanup();

    this.io.close();
  }
}

module.exports = WebSocketServer;
