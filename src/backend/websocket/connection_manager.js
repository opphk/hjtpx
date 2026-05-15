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

let cacheService;
try {
  cacheService = require('../services/cacheService');
} catch (error) {
  cacheService = {
    get: async () => null,
    set: async () => true,
    del: async () => true
  };
}

class ConnectionManager {
  constructor() {
    this.connections = new Map();
    this.userConnections = new Map();
    this.roomConnections = new Map();
    this.connectionMetadata = new Map();
    this.heartbeatTimers = new Map();
    this.reconnectAttempts = new Map();
    this.maxReconnectAttempts = 5;
    this.heartbeatInterval = 25000;
    this.heartbeatTimeout = 10000;
    this.stats = {
      totalConnections: 0,
      totalDisconnections: 0,
      activeConnections: 0,
      uniqueUsers: 0,
      peakConnections: 0,
      avgConnectionDuration: 0,
      connectionHistory: []
    };
  }

  addConnection(socket) {
    const connectionId = socket.id;
    const userId = socket.userId;

    const connectionInfo = {
      socketId: connectionId,
      userId: userId,
      connectedAt: new Date(),
      lastHeartbeat: Date.now(),
      rooms: [],
      metadata: socket.handshake || {},
      reconnectCount: this.reconnectAttempts.get(userId) || 0,
      status: 'active'
    };

    this.connections.set(connectionId, connectionInfo);

    if (!this.userConnections.has(userId)) {
      this.userConnections.set(userId, new Set());
    }
    this.userConnections.get(userId).add(connectionId);

    this.connectionMetadata.set(connectionId, {
      ip: socket.handshake.address,
      userAgent: socket.handshake.headers['user-agent'],
      authenticated: true,
      authData: socket.user
    });

    this.stats.totalConnections++;
    this.stats.activeConnections = this.connections.size;
    this.stats.uniqueUsers = this.userConnections.size;

    if (this.stats.activeConnections > this.stats.peakConnections) {
      this.stats.peakConnections = this.stats.activeConnections;
    }

    this.startHeartbeatMonitor(connectionId);

    logger.info('Connection added', {
      socketId: connectionId,
      userId: userId,
      activeConnections: this.stats.activeConnections,
      userConnectionCount: this.getUserConnectionCount(userId)
    });

    return connectionInfo;
  }

  removeConnection(socketId) {
    const connectionInfo = this.connections.get(socketId);

    if (!connectionInfo) {
      logger.warn('Attempted to remove non-existent connection', { socketId });
      return null;
    }

    const userId = connectionInfo.userId;
    const connectedAt = connectionInfo.connectedAt;
    const duration = Date.now() - connectedAt.getTime();

    this.stopHeartbeatMonitor(socketId);

    if (this.userConnections.has(userId)) {
      this.userConnections.get(userId).delete(socketId);
      if (this.userConnections.get(userId).size === 0) {
        this.userConnections.delete(userId);
      }
    }

    connectionInfo.rooms.forEach(room => {
      this.removeRoomConnection(room, socketId);
    });

    this.connections.delete(socketId);
    this.connectionMetadata.delete(socketId);

    this.stats.totalDisconnections++;
    this.stats.activeConnections = this.connections.size;
    this.stats.uniqueUsers = this.userConnections.size;
    this.stats.connectionHistory.push({
      duration,
      timestamp: new Date(),
      reason: 'disconnect'
    });

    if (this.stats.connectionHistory.length > 1000) {
      this.stats.connectionHistory = this.stats.connectionHistory.slice(-1000);
    }

    const recentDurations = this.stats.connectionHistory.slice(-100).map(h => h.duration);
    if (recentDurations.length > 0) {
      this.stats.avgConnectionDuration =
        recentDurations.reduce((a, b) => a + b, 0) / recentDurations.length;
    }

    logger.info('Connection removed', {
      socketId,
      userId,
      duration,
      activeConnections: this.stats.activeConnections,
      remainingUserConnections: this.getUserConnectionCount(userId)
    });

    return connectionInfo;
  }

  addRoomConnection(room, socketId, userId) {
    if (!this.roomConnections.has(room)) {
      this.roomConnections.set(room, new Map());
    }

    const roomConnections = this.roomConnections.get(room);
    if (!roomConnections.has(socketId)) {
      roomConnections.set(socketId, userId);

      const connectionInfo = this.connections.get(socketId);
      if (connectionInfo && !connectionInfo.rooms.includes(room)) {
        connectionInfo.rooms.push(room);
      }

      logger.debug('Room connection added', {
        room,
        socketId,
        userId,
        roomMemberCount: roomConnections.size
      });
    }
  }

  removeRoomConnection(room, socketId) {
    if (this.roomConnections.has(room)) {
      this.roomConnections.get(room).delete(socketId);

      if (this.roomConnections.get(room).size === 0) {
        this.roomConnections.delete(room);
      }

      logger.debug('Room connection removed', {
        room,
        socketId,
        remainingMembers: this.roomConnections.get(room)?.size || 0
      });
    }
  }

  getConnection(socketId) {
    return this.connections.get(socketId);
  }

  getUserConnection(socketId) {
    const connection = this.connections.get(socketId);
    return connection ? connection.userId : null;
  }

  getConnectionMetadata(socketId) {
    return this.connectionMetadata.get(socketId);
  }

  getUserConnections(userId) {
    const socketIds = this.userConnections.get(userId);
    if (!socketIds) return [];

    return Array.from(socketIds)
      .map(socketId => this.connections.get(socketId))
      .filter(conn => conn !== undefined);
  }

  getUserConnectionCount(userId) {
    const connections = this.userConnections.get(userId);
    return connections ? connections.size : 0;
  }

  isUserOnline(userId) {
    return this.userConnections.has(userId) && this.userConnections.get(userId).size > 0;
  }

  getOnlineUsers() {
    return Array.from(this.userConnections.keys()).map(userId => ({
      userId,
      connectionCount: this.userConnections.get(userId).size,
      connections: this.getUserConnections(userId).map(conn => ({
        socketId: conn.socketId,
        connectedAt: conn.connectedAt,
        rooms: conn.rooms
      }))
    }));
  }

  getRoomMembers(room) {
    if (!this.roomConnections.has(room)) return [];
    return Array.from(this.roomConnections.get(room).entries()).map(([socketId, userId]) => ({
      socketId,
      userId
    }));
  }

  getRoomMemberCount(room) {
    if (!this.roomConnections.has(room)) return 0;
    return this.roomConnections.get(room).size;
  }

  getAllRooms() {
    return Array.from(this.roomConnections.keys()).map(room => ({
      name: room,
      memberCount: this.roomConnections.get(room).size,
      members: this.getRoomMembers(room)
    }));
  }

  updateHeartbeat(socketId) {
    const connection = this.connections.get(socketId);
    if (connection) {
      connection.lastHeartbeat = Date.now();
      return true;
    }
    return false;
  }

  startHeartbeatMonitor(socketId) {
    this.stopHeartbeatMonitor(socketId);

    const timer = setInterval(() => {
      this.checkHeartbeat(socketId);
    }, this.heartbeatInterval);

    this.heartbeatTimers.set(socketId, timer);
  }

  stopHeartbeatMonitor(socketId) {
    if (this.heartbeatTimers.has(socketId)) {
      clearInterval(this.heartbeatTimers.get(socketId));
      this.heartbeatTimers.delete(socketId);
    }
  }

  checkHeartbeat(socketId) {
    const connection = this.connections.get(socketId);
    if (!connection) {
      this.stopHeartbeatMonitor(socketId);
      return false;
    }

    const timeSinceLastHeartbeat = Date.now() - connection.lastHeartbeat;

    if (timeSinceLastHeartbeat > this.heartbeatInterval + this.heartbeatTimeout) {
      logger.warn('Connection heartbeat timeout', {
        socketId,
        userId: connection.userId,
        timeSinceLastHeartbeat
      });
      return false;
    }

    return true;
  }

  recordReconnectAttempt(userId) {
    const attempts = (this.reconnectAttempts.get(userId) || 0) + 1;
    this.reconnectAttempts.set(userId, attempts);

    logger.info('Reconnect attempt recorded', {
      userId,
      attempts,
      maxAttempts: this.maxReconnectAttempts
    });

    if (attempts >= this.maxReconnectAttempts) {
      logger.warn('Max reconnect attempts reached', {
        userId,
        attempts
      });
      return false;
    }

    return true;
  }

  clearReconnectAttempts(userId) {
    this.reconnectAttempts.delete(userId);
  }

  getReconnectAttempts(userId) {
    return this.reconnectAttempts.get(userId) || 0;
  }

  getStats() {
    return {
      totalConnections: this.stats.totalConnections,
      totalDisconnections: this.stats.totalDisconnections,
      activeConnections: this.stats.activeConnections,
      uniqueUsers: this.stats.uniqueUsers,
      peakConnections: this.stats.peakConnections,
      avgConnectionDuration: Math.round(this.stats.avgConnectionDuration),
      roomCount: this.roomConnections.size,
      rooms: this.getAllRooms().map(r => ({ name: r.name, memberCount: r.memberCount })),
      heartbeatActive: this.heartbeatTimers.size,
      reconnectAttempts: Array.from(this.reconnectAttempts.entries()).map(([userId, attempts]) => ({
        userId,
        attempts
      }))
    };
  }

  async persistOnlineStatus() {
    try {
      const onlineUsers = this.getOnlineUsers();
      const statusData = {
        users: onlineUsers.map(u => ({
          userId: u.userId,
          connectionCount: u.connectionCount,
          connectedAt: u.connections[0]?.connectedAt
        })),
        totalConnections: this.stats.activeConnections,
        timestamp: new Date()
      };

      await cacheService.set('websocket:online_status', statusData, 60, ['websocket', 'presence']);

      return statusData;
    } catch (error) {
      logger.error('Failed to persist online status', { error: error.message });
      return null;
    }
  }

  async getPersistedOnlineStatus() {
    try {
      return await cacheService.get('websocket:online_status');
    } catch (error) {
      logger.error('Failed to get persisted online status', { error: error.message });
      return null;
    }
  }

  cleanup() {
    logger.info('Cleaning up connection manager');

    for (const socketId of this.heartbeatTimers.keys()) {
      this.stopHeartbeatMonitor(socketId);
    }

    this.connections.clear();
    this.userConnections.clear();
    this.roomConnections.clear();
    this.connectionMetadata.clear();
    this.reconnectAttempts.clear();

    this.stats = {
      totalConnections: 0,
      totalDisconnections: 0,
      activeConnections: 0,
      uniqueUsers: 0,
      peakConnections: 0,
      avgConnectionDuration: 0,
      connectionHistory: []
    };

    logger.info('Connection manager cleaned up');
  }
}

const connectionManager = new ConnectionManager();

module.exports = connectionManager;
module.exports.ConnectionManager = ConnectionManager;
