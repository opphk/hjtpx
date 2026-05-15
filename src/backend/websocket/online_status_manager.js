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

class OnlineStatusManager {
  constructor() {
    this.onlineUsers = new Map();
    this.userStatuses = new Map();
    this.lastActivity = new Map();
    this.statusHistory = [];
    this.maxHistorySize = 1000;
    this.presenceTTL = 300;
    this.activityTimeout = 300000;
    this.statusCheckInterval = null;
    this.statusCheckIntervalMs = 60000;
    this.stats = {
      statusChanges: 0,
      onlineToOffline: 0,
      offlineToOnline: 0,
      statusUpdates: 0
    };
  }

  initialize(io, connectionManager) {
    this.io = io;
    this.connectionManager = connectionManager;
    this.startStatusChecker();
    logger.info('Online status manager initialized');
  }

  setUserOnline(userId, socketId, metadata = {}) {
    const previousStatus = this.userStatuses.get(userId);
    const wasOnline = this.onlineUsers.has(userId);

    this.onlineUsers.set(userId, {
      socketId,
      userId,
      status: 'online',
      connectedAt: new Date(),
      lastActivity: Date.now(),
      metadata
    });

    this.userStatuses.set(userId, 'online');
    this.updateLastActivity(userId);

    if (!wasOnline) {
      this.stats.offlineToOnline++;
      this.stats.statusChanges++;
      this.recordStatusChange(userId, 'offline', 'online', metadata);

      this.io.emit('online_status', {
        userId,
        status: 'online',
        timestamp: new Date()
      });

      this.notifyFriendsOfStatusChange(userId, 'online');

      logger.info('User went online', {
        userId,
        socketId,
        previousStatus
      });
    }

    this.persistUserStatus(userId);

    return true;
  }

  setUserOffline(userId, reason = 'disconnect', metadata = {}) {
    const wasOnline = this.onlineUsers.has(userId);
    const previousStatus = this.userStatuses.get(userId) || 'unknown';

    if (wasOnline) {
      const userData = this.onlineUsers.get(userId);
      const duration = Date.now() - userData.connectedAt.getTime();

      this.recordStatusChange(userId, 'online', 'offline', {
        ...metadata,
        duration,
        reason
      });

      this.onlineUsers.delete(userId);
      this.userStatuses.set(userId, 'offline');
      this.stats.onlineToOffline++;
      this.stats.statusChanges++;

      this.io.emit('online_status', {
        userId,
        status: 'offline',
        timestamp: new Date(),
        reason,
        duration
      });

      this.notifyFriendsOfStatusChange(userId, 'offline');

      logger.info('User went offline', {
        userId,
        reason,
        duration,
        previousStatus
      });
    }

    this.persistUserStatus(userId);

    return true;
  }

  updateUserStatus(userId, status, metadata = {}) {
    const validStatuses = ['online', 'away', 'busy', 'offline'];
    if (!validStatuses.includes(status)) {
      logger.warn('Invalid status update', { userId, status });
      return false;
    }

    const previousStatus = this.userStatuses.get(userId) || 'offline';

    if (previousStatus === status) {
      return false;
    }

    this.userStatuses.set(userId, status);

    if (this.onlineUsers.has(userId)) {
      const userData = this.onlineUsers.get(userId);
      userData.status = status;
      userData.lastActivity = Date.now();
    }

    this.stats.statusUpdates++;
    this.stats.statusChanges++;

    this.recordStatusChange(userId, previousStatus, status, metadata);

    this.io.emit('online_status', {
      userId,
      status,
      previousStatus,
      timestamp: new Date(),
      metadata
    });

    logger.info('User status updated', {
      userId,
      previousStatus,
      newStatus: status
    });

    this.persistUserStatus(userId);

    return true;
  }

  updateLastActivity(userId) {
    const now = Date.now();
    this.lastActivity.set(userId, now);

    if (this.onlineUsers.has(userId)) {
      this.onlineUsers.get(userId).lastActivity = now;
    }

    this.persistUserStatus(userId);
  }

  recordStatusChange(userId, fromStatus, toStatus, metadata = {}) {
    const record = {
      userId,
      fromStatus,
      toStatus,
      timestamp: new Date(),
      metadata
    };

    this.statusHistory.push(record);

    if (this.statusHistory.length > this.maxHistorySize) {
      this.statusHistory = this.statusHistory.slice(-this.maxHistorySize);
    }

    return record;
  }

  getUserStatus(userId) {
    return {
      userId,
      status: this.userStatuses.get(userId) || 'offline',
      isOnline: this.onlineUsers.has(userId),
      lastActivity: this.lastActivity.get(userId),
      onlineData: this.onlineUsers.get(userId) || null
    };
  }

  getOnlineUsers() {
    return Array.from(this.onlineUsers.values()).map(user => ({
      userId: user.userId,
      status: user.status,
      connectedAt: user.connectedAt,
      lastActivity: user.lastActivity,
      duration: Date.now() - user.connectedAt.getTime()
    }));
  }

  getOnlineUserCount() {
    return this.onlineUsers.size;
  }

  getOfflineUsers() {
    const allKnownUsers = new Set([...this.onlineUsers.keys(), ...this.userStatuses.keys()]);

    return Array.from(allKnownUsers)
      .filter(userId => !this.onlineUsers.has(userId))
      .map(userId => ({
        userId,
        status: this.userStatuses.get(userId) || 'offline',
        lastActivity: this.lastActivity.get(userId)
      }));
  }

  getUsersByStatus(status) {
    const users = [];

    if (status === 'online') {
      return this.getOnlineUsers();
    }

    for (const [userId, userStatus] of this.userStatuses.entries()) {
      if (userStatus === status) {
        users.push({
          userId,
          status: userStatus,
          isOnline: this.onlineUsers.has(userId),
          lastActivity: this.lastActivity.get(userId)
        });
      }
    }

    return users;
  }

  isUserOnline(userId) {
    return this.onlineUsers.has(userId);
  }

  startStatusChecker() {
    if (this.statusCheckInterval) {
      clearInterval(this.statusCheckInterval);
    }

    this.statusCheckInterval = setInterval(() => {
      this.checkIdleUsers();
      this.checkInactiveUsers();
    }, this.statusCheckIntervalMs);

    logger.debug('Status checker started', {
      interval: this.statusCheckIntervalMs
    });
  }

  stopStatusChecker() {
    if (this.statusCheckInterval) {
      clearInterval(this.statusCheckInterval);
      this.statusCheckInterval = null;
      logger.debug('Status checker stopped');
    }
  }

  checkIdleUsers() {
    const now = Date.now();
    const idleThreshold = 300000;

    for (const [userId, lastActivity] of this.lastActivity.entries()) {
      const idleTime = now - lastActivity;

      if (idleTime > idleThreshold && this.userStatuses.get(userId) === 'online') {
        if (this.onlineUsers.has(userId)) {
          const userData = this.onlineUsers.get(userId);
          if (userData.status === 'online') {
            this.updateUserStatus(userId, 'away', { reason: 'idle', idleTime });
          }
        }
      }
    }
  }

  checkInactiveUsers() {
    const now = Date.now();

    for (const [socketId, connection] of this.connectionManager.connections.entries()) {
      const timeSinceLastHeartbeat = now - connection.lastHeartbeat;

      if (timeSinceLastHeartbeat > this.activityTimeout) {
        logger.debug('Connection inactive, triggering status check', {
          socketId,
          userId: connection.userId,
          timeSinceLastHeartbeat
        });
      }
    }
  }

  notifyFriendsOfStatusChange(userId, status) {
    this.io.emit('friend_status', {
      userId,
      status,
      timestamp: new Date()
    });
  }

  async persistUserStatus(userId) {
    try {
      const statusData = this.getUserStatus(userId);

      await cacheService.set(`presence:${userId}`, statusData, this.presenceTTL, [
        'presence',
        `user:${userId}`
      ]);

      await this.persistOnlineList();

      return true;
    } catch (error) {
      logger.error('Failed to persist user status', {
        userId,
        error: error.message
      });
      return false;
    }
  }

  async persistOnlineList() {
    try {
      const onlineList = this.getOnlineUsers().map(u => ({
        userId: u.userId,
        status: u.status,
        connectedAt: u.connectedAt
      }));

      await cacheService.set(
        'presence:online_list',
        {
          users: onlineList,
          count: onlineList.length,
          timestamp: new Date()
        },
        60,
        ['presence', 'presence:online']
      );

      return true;
    } catch (error) {
      logger.error('Failed to persist online list', { error: error.message });
      return false;
    }
  }

  async getPersistedStatus(userId) {
    try {
      return await cacheService.get(`presence:${userId}`);
    } catch (error) {
      logger.error('Failed to get persisted status', {
        userId,
        error: error.message
      });
      return null;
    }
  }

  async getPersistedOnlineList() {
    try {
      return await cacheService.get('presence:online_list');
    } catch (error) {
      logger.error('Failed to get persisted online list', {
        error: error.message
      });
      return null;
    }
  }

  getStatusHistory(userId, limit = 50) {
    return this.statusHistory
      .filter(record => record.userId === userId)
      .slice(-limit)
      .reverse();
  }

  getStats() {
    const statusCounts = {
      online: 0,
      away: 0,
      busy: 0,
      offline: 0
    };

    for (const status of this.userStatuses.values()) {
      if (statusCounts.hasOwnProperty(status)) {
        statusCounts[status]++;
      }
    }

    return {
      ...this.stats,
      totalKnownUsers: this.userStatuses.size,
      currentOnline: this.onlineUsers.size,
      statusCounts,
      recentHistory: this.statusHistory.slice(-20)
    };
  }

  cleanup() {
    this.stopStatusChecker();

    this.onlineUsers.clear();
    this.userStatuses.clear();
    this.lastActivity.clear();
    this.statusHistory = [];
    this.stats = {
      statusChanges: 0,
      onlineToOffline: 0,
      offlineToOnline: 0,
      statusUpdates: 0
    };

    logger.info('Online status manager cleaned up');
  }
}

const onlineStatusManager = new OnlineStatusManager();

module.exports = onlineStatusManager;
module.exports.OnlineStatusManager = OnlineStatusManager;
