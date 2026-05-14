const { Server } = require('socket.io');
const jwt = require('jsonwebtoken');
const { logger } = require('../middleware/logger');

class WebSocketServer {
  constructor(httpServer) {
    this.io = new Server(httpServer, {
      cors: {
        origin: process.env.CORS_ORIGIN || '*',
        methods: ['GET', 'POST'],
        credentials: true
      },
      pingTimeout: 60000,
      pingInterval: 25000,
      transports: ['websocket', 'polling']
    });

    this.connectedClients = new Map();
    this.roomSubscriptions = new Map();

    this.setupMiddleware();
    this.setupEventHandlers();
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
    this.io.on('connection', (socket) => {
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

    logger.info('Client connected', {
      socketId: socket.id,
      userId: socket.userId
    });

    socket.emit('connected', {
      socketId: socket.id,
      message: 'Successfully connected to WebSocket server'
    });

    this.setupSocketEventHandlers(socket);

    this.broadcastUserOnlineStatus(socket.userId, true);
  }

  setupSocketEventHandlers(socket) {
    socket.on('disconnect', (reason) => {
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

    socket.on('error', (error) => {
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
      logger.info('Message received', {
        socketId: socket.id,
        userId: socket.userId,
        type: data.type
      });

      if (callback && typeof callback === 'function') {
        callback({ success: true, received: true });
      }
    } catch (error) {
      logger.error('Error handling message', { error: error.message });
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
      logger.error('Error sending broadcast', { error: error.message });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handleDisconnection(socket, reason) {
    const clientInfo = this.connectedClients.get(socket.id);

    if (clientInfo) {
      logger.info('Client disconnected', {
        socketId: socket.id,
        userId: socket.userId,
        reason,
        connectedDuration: Date.now() - clientInfo.connectedAt.getTime()
      });

      this.broadcastUserOnlineStatus(socket.userId, false);

      clientInfo.rooms.forEach(room => {
        socket.to(room).emit('user:left', {
          userId: socket.userId,
          room,
          timestamp: new Date()
        });
      });

      this.connectedClients.delete(socket.id);
    }
  }

  broadcastUserOnlineStatus(userId, isOnline) {
    this.io.emit('presence:update', {
      userId,
      isOnline,
      timestamp: new Date()
    });
  }

  broadcast(event, data, room = null) {
    if (room) {
      this.io.to(room).emit(event, data);
    } else {
      this.io.emit(event, data);
    }
  }

  sendToUser(userId, event, data) {
    for (const [socketId, clientInfo] of this.connectedClients.entries()) {
      if (clientInfo.userId === userId) {
        this.io.to(socketId).emit(event, data);
        return true;
      }
    }
    return false;
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
    const onlineUsers = new Map();
    for (const [socketId, clientInfo] of this.connectedClients.entries()) {
      if (!onlineUsers.has(clientInfo.userId)) {
        onlineUsers.set(clientInfo.userId, {
          userId: clientInfo.userId,
          socketCount: 0,
          connectedAt: clientInfo.connectedAt
        });
      }
      const user = onlineUsers.get(clientInfo.userId);
      user.socketCount++;
    }
    return Array.from(onlineUsers.values());
  }

  getConnectionStats() {
    return {
      totalConnections: this.connectedClients.size,
      onlineUsers: this.getOnlineUsers().length,
      rooms: Array.from(this.roomSubscriptions.keys()),
      subscriptions: Array.from(this.roomSubscriptions.entries()).map(([channel, users]) => ({
        channel,
        subscriberCount: users.size
      }))
    };
  }

  close() {
    logger.info('Closing WebSocket server');
    this.io.close();
  }
}

module.exports = WebSocketServer;
