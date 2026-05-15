const WebSocket = require('ws');
const { logInfo: loggerInfo, logWarning: loggerWarn, logError: loggerError } = require('../middleware/logger');

class WebSocketManager {
  constructor() {
    this.io = null;
    this.heartbeatInterval = null;
    this.heartbeatIntervalTime = 30000;
    this.clientLastActivity = new Map();
    this.clientHeartbeatTimeout = 60000;
    this.maxReconnectAttempts = 5;
    this.reconnectDelays = [1000, 2000, 5000, 10000, 30000];
  }

  initialize(httpServer) {
    if (this.io) {
      loggerWarn('WebSocket manager already initialized');
      return;
    }

    this.io = new WebSocket.Server({ server: httpServer, path: '/ws' });

    this.io.on('connection', (socket, req) => this.handleConnection(socket, req));

    this.startHeartbeatChecker();

    loggerInfo('WebSocket manager initialized');
  }

  handleConnection(socket, req) {
    const clientId = this.generateClientId();
    socket.clientId = clientId;
    socket.isAlive = true;
    this.clientLastActivity.set(clientId, Date.now());

    loggerInfo('New WebSocket connection', { clientId });

    socket.on('pong', () => {
      socket.isAlive = true;
      this.clientLastActivity.set(clientId, Date.now());
    });

    socket.on('message', (data) => this.handleMessage(socket, data));

    socket.on('close', (code, reason) => {
      loggerInfo('WebSocket connection closed', { clientId, code, reason: reason.toString() });
      this.clientLastActivity.delete(clientId);
    });

    socket.on('error', (error) => {
      loggerError('WebSocket error', { clientId, error: error.message });
    });

    this.sendToClient(socket, {
      type: 'connected',
      clientId,
      timestamp: new Date().toISOString()
    });
  }

  handleMessage(socket, data) {
    try {
      const message = JSON.parse(data);
      this.clientLastActivity.set(socket.clientId, Date.now());

      switch (message.type) {
        case 'ping':
          this.sendToClient(socket, { type: 'pong', timestamp: new Date().toISOString() });
          break;

        case 'auth':
          this.handleAuth(socket, message);
          break;

        case 'subscribe':
          this.handleSubscribe(socket, message);
          break;

        case 'unsubscribe':
          this.handleUnsubscribe(socket, message);
          break;

        case 'notification_read':
          this.handleNotificationRead(socket, message);
          break;

        default:
          loggerWarn('Unknown message type', { type: message.type, clientId: socket.clientId });
      }
    } catch (error) {
      loggerError('Error parsing WebSocket message', { error: error.message });
      this.sendToClient(socket, {
        type: 'error',
        message: 'Invalid message format'
      });
    }
  }

  handleAuth(socket, message) {
    const { token } = message;

    try {
      const jwt = require('jsonwebtoken');
      const decoded = jwt.verify(token, process.env.JWT_SECRET || 'your-secret-key');

      socket.userId = decoded.id;
      socket.isAuthenticated = true;

      this.sendToClient(socket, {
        type: 'auth_success',
        userId: decoded.id,
        timestamp: new Date().toISOString()
      });

      loggerInfo('WebSocket authenticated', { clientId: socket.clientId, userId: decoded.id });
    } catch (error) {
      this.sendToClient(socket, {
        type: 'auth_error',
        message: 'Authentication failed',
        timestamp: new Date().toISOString()
      });
    }
  }

  handleSubscribe(socket, message) {
    const { room } = message;

    if (!socket.rooms) {
      socket.rooms = new Set();
    }

    socket.rooms.add(room);
    socket.join(room);

    this.sendToClient(socket, {
      type: 'subscribed',
      room,
      timestamp: new Date().toISOString()
    });
  }

  handleUnsubscribe(socket, message) {
    const { room } = message;

    if (socket.rooms && socket.rooms.has(room)) {
      socket.rooms.delete(room);
      socket.leave(room);

      this.sendToClient(socket, {
        type: 'unsubscribed',
        room,
        timestamp: new Date().toISOString()
      });
    }
  }

  handleNotificationRead(socket, message) {
    const { notificationId } = message;

    if (socket.isAuthenticated) {
      const Notification = require('../models/Notification');
      Notification.findByIdAndUpdate(
        notificationId,
        { read: true, readAt: new Date() },
        { new: true }
      ).then(notification => {
        if (notification) {
          this.sendToClient(socket, {
            type: 'notification_updated',
            notification,
            timestamp: new Date().toISOString()
          });
        }
      }).catch(error => {
        loggerError('Error updating notification', { error: error.message });
      });
    }
  }

  startHeartbeatChecker() {
    this.heartbeatInterval = setInterval(() => {
      this.checkHeartbeats();
    }, this.heartbeatIntervalTime);
  }

  checkHeartbeats() {
    const now = Date.now();
    const timeoutThreshold = now - this.clientHeartbeatTimeout;

    this.io.clients.forEach(socket => {
      if (socket.isAlive === false) {
        loggerWarn('Terminating inactive WebSocket client', { clientId: socket.clientId });
        return socket.terminate();
      }

      socket.isAlive = false;
      socket.ping();

      const lastActivity = this.clientLastActivity.get(socket.clientId);
      if (lastActivity && lastActivity < timeoutThreshold) {
        loggerWarn('Client inactive timeout', { clientId: socket.clientId });
        socket.terminate();
      }
    });
  }

  broadcast(event, data, room = null) {
    if (room) {
      this.io.to(room).emit(event, data);
    } else {
      this.io.emit(event, data);
    }
  }

  sendToClient(socket, data) {
    if (socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(data));
      return true;
    }
    return false;
  }

  sendToUser(userId, event, data) {
    let sent = false;

    this.io.clients.forEach(socket => {
      if (socket.userId === userId && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(data));
        sent = true;
      }
    });

    return sent;
  }

  getConnectedClients() {
    const clients = [];

    this.io.clients.forEach(socket => {
      clients.push({
        clientId: socket.clientId,
        userId: socket.userId,
        isAuthenticated: socket.isAuthenticated || false,
        rooms: Array.from(socket.rooms || []),
        connectedAt: socket.connectedAt
      });
    });

    return clients;
  }

  getOnlineUsers() {
    const users = new Set();

    this.io.clients.forEach(socket => {
      if (socket.userId) {
        users.add(socket.userId);
      }
    });

    return Array.from(users);
  }

  getConnectionStats() {
    const clients = this.getConnectedClients();
    const onlineUsers = this.getOnlineUsers();

    return {
      totalConnections: clients.length,
      onlineUsers: onlineUsers.length,
      authenticatedConnections: clients.filter(c => c.isAuthenticated).length,
      anonymousConnections: clients.filter(c => !c.isAuthenticated).length
    };
  }

  generateClientId() {
    return `client_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  getDetailedMetrics() {
    return {
      ...this.getConnectionStats(),
      uptime: process.uptime(),
      memoryUsage: process.memoryUsage(),
      timestamp: new Date().toISOString()
    };
  }

  close() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
    }

    this.io.clients.forEach(socket => {
      socket.close();
    });

    this.io.close();
    this.io = null;

    loggerInfo('WebSocket manager closed');
  }
}

const websocketManager = new WebSocketManager();

module.exports = websocketManager;
