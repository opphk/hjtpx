const { logInfo, logWarning, logError } = require('../middleware/logger');
const jwt = require('jsonwebtoken');
const { Server } = require('socket.io');
const metricsService = require('../services/metricsService');

class WebSocketServer {
  constructor(httpServer) {
    this.heartbeatConfig = {
      pingTimeout: parseInt(process.env.WS_PING_TIMEOUT) || 30000,
      pingInterval: parseInt(process.env.WS_PING_INTERVAL) || 15000,
      heartbeatCheckInterval: parseInt(process.env.WS_HEARTBEAT_CHECK_INTERVAL) || 5000,
      maxMissedHeartbeats: parseInt(process.env.WS_MAX_MISSED_HEARTBEATS) || 3
    };

    this.io = new Server(httpServer, {
      cors: {
        origin: process.env.CORS_ORIGIN || '*',
        methods: ['GET', 'POST'],
        credentials: true
      },
      pingTimeout: this.heartbeatConfig.pingTimeout,
      pingInterval: this.heartbeatConfig.pingInterval,
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
    this.clientHeartbeats = new Map();
    
    this.metrics = {
      totalConnections: 0,
      totalDisconnections: 0,
      messagesSent: 0,
      messagesReceived: 0,
      errors: 0,
      heartbeatsSent: 0,
      heartbeatsReceived: 0,
      missedHeartbeats: 0,
      connectionTimes: [],
      startTime: Date.now(),
      latencySamples: [],
      lastLatencyReport: Date.now()
    };

    this.connectionStateMonitor = {
      activeConnections: 0,
      pendingConnections: 0,
      maxConcurrentConnections: 0,
      connectionQueue: [],
      stateHistory: []
    };

    this.setupHeartbeatMonitor();
    this.setupConnectionStateMonitor();
    this.setupMiddleware();
    this.setupEventHandlers();
  }

  setupHeartbeatMonitor() {
    this.heartbeatInterval = setInterval(() => {
      this.checkClientHeartbeats();
    }, this.heartbeatConfig.heartbeatCheckInterval);

    this.io.on('connection', socket => {
      this.clientHeartbeats.set(socket.id, {
        lastPing: Date.now(),
        missedCount: 0,
        connectedAt: Date.now(),
        lastHeartbeatResponse: Date.now()
      });
    });

    this.io.engine.on('packet', (packet, socket) => {
      if (packet.type === 'pong') {
        const socketId = this.getSocketIdFromPacket(socket);
        if (socketId) {
          const heartbeat = this.clientHeartbeats.get(socketId);
          if (heartbeat) {
            const now = Date.now();
            const latency = now - heartbeat.lastPing;
            heartbeat.lastPing = now;
            heartbeat.lastHeartbeatResponse = now;
            heartbeat.missedCount = 0;
            this.metrics.heartbeatsReceived++;
            this.recordLatency(latency);
            
            if (metricsService && metricsService.recordWebSocketHeartbeat) {
              metricsService.recordWebSocketHeartbeat('received');
            }
          }
        }
      }
    });
  }

  getSocketIdFromPacket(engine) {
    for (const [id, socket] of this.io.sockets.sockets) {
      if (socket.conn && socket.conn.transport) {
        return id;
      }
    }
    return null;
  }

  recordLatency(latency) {
    this.metrics.latencySamples.push(latency);
    if (this.metrics.latencySamples.length > 1000) {
      this.metrics.latencySamples.shift();
    }
    
    const now = Date.now();
    if (now - this.metrics.lastLatencyReport > 60000) {
      this.reportLatencyStats();
      this.metrics.lastLatencyReport = now;
    }
  }

  reportLatencyStats() {
    if (this.metrics.latencySamples.length === 0) return;
    
    const sorted = [...this.metrics.latencySamples].sort((a, b) => a - b);
    const p50 = sorted[Math.floor(sorted.length * 0.5)];
    const p95 = sorted[Math.floor(sorted.length * 0.95)];
    const p99 = sorted[Math.floor(sorted.length * 0.99)];
    
    logInfo('WebSocket latency statistics', {
      p50,
      p95,
      p99,
      samples: this.metrics.latencySamples.length
    });
  }

  findSocketByEngineSocket(engine) {
    for (const [id, socket] of this.io.sockets.sockets) {
      if (socket.conn && socket.conn.transport) {
        return socket;
      }
    }
    return null;
  }

  checkClientHeartbeats() {
    const now = Date.now();
    const timeout = this.heartbeatConfig.pingInterval + this.heartbeatConfig.pingTimeout;

    for (const [socketId, heartbeat] of this.clientHeartbeats.entries()) {
      const timeSinceLastPing = now - heartbeat.lastPing;

      if (timeSinceLastPing > timeout) {
        heartbeat.missedCount++;
        this.metrics.missedHeartbeats++;

        if (metricsService && metricsService.recordWebSocketMissedHeartbeat) {
          metricsService.recordWebSocketMissedHeartbeat();
        }

        if (heartbeat.missedCount >= this.heartbeatConfig.maxMissedHeartbeats) {
          const socket = this.io.sockets.sockets.get(socketId);
          if (socket) {
            logWarning('Client missed too many heartbeats, disconnecting', {
              socketId,
              missedCount: heartbeat.missedCount,
              timeSinceLastPing
            });
            socket.disconnect(true);
            
            if (metricsService && metricsService.recordWebSocketError) {
              metricsService.recordWebSocketError('heartbeat_timeout');
            }
          }
          this.clientHeartbeats.delete(socketId);
        } else {
          logWarning('Client missed heartbeat', {
            socketId,
            missedCount: heartbeat.missedCount,
            timeSinceLastPing
          });
        }
      }
    }
  }

  setupConnectionStateMonitor() {
    this.stateMonitorInterval = setInterval(() => {
      this.updateConnectionState();
    }, 1000);
  }

  updateConnectionState() {
    const currentTime = Date.now();
    const activeConnections = this.connectedClients.size;
    
    this.connectionStateMonitor.activeConnections = activeConnections;
    this.connectionStateMonitor.maxConcurrentConnections = Math.max(
      this.connectionStateMonitor.maxConcurrentConnections,
      activeConnections
    );

    if (activeConnections > 0) {
      this.connectionStateMonitor.stateHistory.push({
        timestamp: currentTime,
        activeConnections,
        queuedConnections: this.connectionStateMonitor.pendingConnections
      });

      if (this.connectionStateMonitor.stateHistory.length > 3600) {
        this.connectionStateMonitor.stateHistory.shift();
      }
    }

    if (metricsService && metricsService.updateConnectionMetrics) {
      metricsService.updateConnectionMetrics(activeConnections);
    }
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
        logError('WebSocket authentication failed', { error: error.message });
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

    logInfo('Client connected', {
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

    socket.on('get:metrics', (callback) => {
      if (callback && typeof callback === 'function') {
        callback({ success: true, metrics: this.getDetailedMetrics() });
      }
    });

    socket.on('error', error => {
      this.metrics.errors++;
      logError('Socket error', {
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

      logInfo('Client joined room', {
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
      logError('Error joining room', { error: error.message });
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

      logInfo('Client left room', {
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
      logError('Error leaving room', { error: error.message });
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

      logInfo('Client subscribed to channel', {
        socketId: socket.id,
        userId: socket.userId,
        channel
      });

      if (callback && typeof callback === 'function') {
        callback({ success: true, channel });
      }
    } catch (error) {
      logError('Error subscribing to channel', { error: error.message });
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

      logInfo('Client unsubscribed from channel', {
        socketId: socket.id,
        userId: socket.userId,
        channel
      });

      if (callback && typeof callback === 'function') {
        callback({ success: true, channel });
      }
    } catch (error) {
      logError('Error unsubscribing from channel', { error: error.message });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handleMessage(socket, data, callback) {
    try {
      this.metrics.messagesReceived++;
      
      if (metricsService && metricsService.recordWebSocketMessageReceived) {
        metricsService.recordWebSocketMessageReceived();
      }
      
      logInfo('Message received', {
        socketId: socket.id,
        userId: socket.userId,
        type: data.type
      });

      if (callback && typeof callback === 'function') {
        callback({ success: true, received: true });
      }
    } catch (error) {
      this.metrics.errors++;
      if (metricsService && metricsService.recordWebSocketError) {
        metricsService.recordWebSocketError('message_handler_error');
      }
      logError('Error handling message', { error: error.message });
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
      
      if (metricsService && metricsService.recordWebSocketMessageSent) {
        metricsService.recordWebSocketMessageSent();
      }
      
      logInfo('Broadcast sent', {
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
      if (metricsService && metricsService.recordWebSocketError) {
        metricsService.recordWebSocketError('broadcast_error');
      }
      logError('Error sending broadcast', { error: error.message });
      if (callback && typeof callback === 'function') {
        callback({ success: false, error: error.message });
      }
    }
  }

  handleDisconnection(socket, reason) {
    const clientInfo = this.connectedClients.get(socket.id);

    if (clientInfo) {
      const connectedDuration = Date.now() - clientInfo.connectedAt.getTime();
      
      logInfo('Client disconnected', {
        socketId: socket.id,
        userId: socket.userId,
        reason,
        connectedDuration
      });

      this.metrics.totalDisconnections++;
      this.metrics.connectionTimes.push(connectedDuration);

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

  getDetailedMetrics() {
    const uptime = Date.now() - this.metrics.startTime;
    const avgConnectionTime = this.metrics.connectionTimes.length > 0
      ? this.metrics.connectionTimes.reduce((sum, t) => sum + t, 0) / this.metrics.connectionTimes.length
      : 0;
    
    return {
      uptime,
      currentConnections: this.connectedClients.size,
      onlineUsers: this.getOnlineUsers().length,
      totalConnections: this.metrics.totalConnections,
      totalDisconnections: this.metrics.totalDisconnections,
      messagesSent: this.metrics.messagesSent,
      messagesReceived: this.metrics.messagesReceived,
      errors: this.metrics.errors,
      heartbeatMetrics: {
        heartbeatsSent: this.metrics.heartbeatsSent,
        heartbeatsReceived: this.metrics.heartbeatsReceived,
        missedHeartbeats: this.metrics.missedHeartbeats,
        activeHeartbeats: this.clientHeartbeats.size,
        config: this.heartbeatConfig
      },
      avgConnectionTime,
      rooms: Array.from(this.roomSubscriptions.keys()),
      subscriptions: Array.from(this.roomSubscriptions.entries()).map(([channel, users]) => ({
        channel,
        subscriberCount: users.size
      }))
    };
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
    logInfo('Closing WebSocket server');
    
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
    
    this.clientHeartbeats.clear();
    this.io.close();
  }
}

module.exports = WebSocketServer;
