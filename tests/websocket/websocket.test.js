const { describe, test, expect } = require('@jest/globals');

describe('WebSocket Modules', () => {
  describe('ConnectionManager', () => {
    let ConnectionManager;
    let connectionManager;

    beforeAll(async () => {
      ConnectionManager = require('../../src/backend/websocket/connection_manager').ConnectionManager;
      connectionManager = new ConnectionManager();
    });

    test('should create instance with correct initial state', () => {
      expect(connectionManager.connections).toBeInstanceOf(Map);
      expect(connectionManager.userConnections).toBeInstanceOf(Map);
      expect(connectionManager.roomConnections).toBeInstanceOf(Map);
      expect(connectionManager.stats.totalConnections).toBe(0);
      expect(connectionManager.stats.activeConnections).toBe(0);
    });

    test('should add connection correctly', () => {
      const mockSocket = {
        id: 'socket-123',
        userId: 'user-456',
        handshake: {
          address: '127.0.0.1',
          headers: { 'user-agent': 'test-agent' }
        }
      };

      const connectionInfo = connectionManager.addConnection(mockSocket);

      expect(connectionInfo.socketId).toBe('socket-123');
      expect(connectionInfo.userId).toBe('user-456');
      expect(connectionInfo.status).toBe('active');
      expect(connectionManager.connections.size).toBe(1);
      expect(connectionManager.userConnections.has('user-456')).toBe(true);
    });

    test('should remove connection correctly', () => {
      const mockSocket = {
        id: 'socket-789',
        userId: 'user-999',
        handshake: {
          address: '127.0.0.1',
          headers: { 'user-agent': 'test-agent' }
        }
      };

      connectionManager.addConnection(mockSocket);
      const removed = connectionManager.removeConnection('socket-789');

      expect(removed).not.toBeNull();
      expect(connectionManager.connections.size).toBe(1);
      expect(connectionManager.stats.totalDisconnections).toBe(1);
    });

    test('should track room connections', () => {
      connectionManager.addRoomConnection('room-1', 'socket-123', 'user-456');

      expect(connectionManager.roomConnections.has('room-1')).toBe(true);
      expect(connectionManager.getRoomMemberCount('room-1')).toBe(1);
    });

    test('should check user online status', () => {
      expect(connectionManager.isUserOnline('user-456')).toBe(true);
      expect(connectionManager.isUserOnline('user-nonexistent')).toBe(false);
    });

    test('should get correct stats', () => {
      const stats = connectionManager.getStats();

      expect(stats.totalConnections).toBeGreaterThanOrEqual(1);
      expect(stats.activeConnections).toBeGreaterThanOrEqual(1);
    });

    test('should update heartbeat', () => {
      const result = connectionManager.updateHeartbeat('socket-123');

      expect(result).toBe(true);
    });

    test('should get all rooms', () => {
      connectionManager.addRoomConnection('room-2', 'socket-2', 'user-2');

      const rooms = connectionManager.getAllRooms();

      expect(rooms.length).toBeGreaterThanOrEqual(2);
    });
  });

  describe('NotificationSystem', () => {
    let NotificationSystem;
    let notificationSystem;

    beforeAll(async () => {
      NotificationSystem = require('../../src/backend/websocket/notification_system').NotificationSystem;
      notificationSystem = new NotificationSystem();
    });

    test('should create instance with correct initial state', () => {
      expect(notificationSystem.stats.totalNotifications).toBe(0);
      expect(notificationSystem.stats.sentNotifications).toBe(0);
      expect(notificationSystem.stats.failedNotifications).toBe(0);
    });

    test('should generate notification ID', () => {
      const id = notificationSystem.generateNotificationId();

      expect(id).toMatch(/^notif_/);
    });

    test('should get correct stats', () => {
      const stats = notificationSystem.getStats();

      expect(stats).toHaveProperty('totalNotifications');
      expect(stats).toHaveProperty('sentNotifications');
      expect(stats).toHaveProperty('failedNotifications');
    });
  });

  describe('OnlineStatusManager', () => {
    let OnlineStatusManager;
    let onlineStatusManager;
    const mockIO = { emit: jest.fn() };
    const mockConnectionManager = { connections: new Map() };

    beforeAll(async () => {
      OnlineStatusManager = require('../../src/backend/websocket/online_status_manager').OnlineStatusManager;
      onlineStatusManager = new OnlineStatusManager();
      onlineStatusManager.initialize(mockIO, mockConnectionManager);
    });

    test('should create instance with correct initial state', () => {
      expect(onlineStatusManager.onlineUsers).toBeInstanceOf(Map);
      expect(onlineStatusManager.userStatuses).toBeInstanceOf(Map);
      expect(onlineStatusManager.stats.statusChanges).toBe(0);
    });

    test('should set user online', () => {
      onlineStatusManager.setUserOnline('user-online-test', 'socket-online');

      expect(onlineStatusManager.onlineUsers.has('user-online-test')).toBe(true);
      expect(onlineStatusManager.userStatuses.get('user-online-test')).toBe('online');
      expect(onlineStatusManager.stats.offlineToOnline).toBe(1);
    });

    test('should set user offline', () => {
      onlineStatusManager.setUserOffline('user-online-test');

      expect(onlineStatusManager.onlineUsers.has('user-online-test')).toBe(false);
      expect(onlineStatusManager.userStatuses.get('user-online-test')).toBe('offline');
      expect(onlineStatusManager.stats.onlineToOffline).toBe(1);
    });

    test('should update user status', () => {
      onlineStatusManager.setUserOnline('user-status-test', 'socket-status');
      const result = onlineStatusManager.updateUserStatus('user-status-test', 'away');

      expect(result).toBe(true);
      expect(onlineStatusManager.onlineUsers.get('user-status-test').status).toBe('away');
    });

    test('should reject invalid status', () => {
      const result = onlineStatusManager.updateUserStatus('user-status-test', 'invalid_status');

      expect(result).toBe(false);
    });

    test('should get online users count', () => {
      onlineStatusManager.setUserOnline('user-count-1', 'socket-count-1');
      onlineStatusManager.setUserOnline('user-count-2', 'socket-count-2');

      expect(onlineStatusManager.getOnlineUserCount()).toBeGreaterThanOrEqual(2);
    });

    test('should check if user is online', () => {
      expect(onlineStatusManager.isUserOnline('user-count-1')).toBe(true);
      expect(onlineStatusManager.isUserOnline('user-nonexistent')).toBe(false);
    });

    test('should get status history', () => {
      const history = onlineStatusManager.getStatusHistory('user-status-test');

      expect(history.length).toBeGreaterThan(0);
    });
  });

  describe('MessageBroadcaster', () => {
    let MessageBroadcaster;
    let messageBroadcaster;

    beforeAll(async () => {
      MessageBroadcaster = require('../../src/backend/websocket/message_broadcaster').MessageBroadcaster;
      messageBroadcaster = new MessageBroadcaster();
    });

    test('should create instance with correct initial state', () => {
      expect(messageBroadcaster.stats.totalMessages).toBe(0);
    });

    test('should generate message ID', () => {
      const id = messageBroadcaster.generateMessageId();

      expect(id).toMatch(/^msg_/);
    });

    test('should get correct stats', () => {
      const stats = messageBroadcaster.getStats();

      expect(stats).toHaveProperty('totalMessages');
      expect(stats).toHaveProperty('privateMessages');
      expect(stats).toHaveProperty('groupMessages');
    });

    test('should add to user history', () => {
      messageBroadcaster.addToHistory('user-history-test', { id: 'msg-1', content: 'Test' });

      const history = messageBroadcaster.getUserHistory('user-history-test');

      expect(history.history.length).toBeGreaterThan(0);
    });
  });

  describe('HeartbeatSystem', () => {
    let HeartbeatSystem;
    let heartbeatSystem;

    beforeAll(async () => {
      HeartbeatSystem = require('../../src/backend/websocket/heartbeat_system').HeartbeatSystem;
      heartbeatSystem = new HeartbeatSystem();
    });

    test('should create instance with correct initial state', () => {
      expect(heartbeatSystem.heartbeatInterval).toBe(25000);
      expect(heartbeatSystem.heartbeatTimeout).toBe(10000);
      expect(heartbeatSystem.stats.totalHeartbeats).toBe(0);
    });

    test('should calculate reconnect delay correctly', () => {
      expect(heartbeatSystem.calculateReconnectDelay(1)).toBe(1000);
      expect(heartbeatSystem.calculateReconnectDelay(2)).toBe(2000);
      expect(heartbeatSystem.calculateReconnectDelay(3)).toBe(4000);
    });

    test('should get heartbeat status', () => {
      const status = heartbeatSystem.getHeartbeatStatus('non-existent-socket');

      expect(status).toBe('unknown');
    });

    test('should get correct stats', () => {
      const stats = heartbeatSystem.getStats();

      expect(stats).toHaveProperty('totalHeartbeats');
      expect(stats).toHaveProperty('heartbeatInterval');
      expect(stats).toHaveProperty('connectionHealth');
    });
  });
});

describe('WebSocket Integration', () => {
  test('should export all required module classes', async () => {
    const connectionModule = await import('../../src/backend/websocket/connection_manager.js');
    const notificationModule = await import('../../src/backend/websocket/notification_system.js');
    const onlineStatusModule = await import('../../src/backend/websocket/online_status_manager.js');
    const messageModule = await import('../../src/backend/websocket/message_broadcaster.js');
    const heartbeatModule = await import('../../src/backend/websocket/heartbeat_system.js');

    expect(connectionModule.ConnectionManager).toBeDefined();
    expect(notificationModule.NotificationSystem).toBeDefined();
    expect(onlineStatusModule.OnlineStatusManager).toBeDefined();
    expect(messageModule.MessageBroadcaster).toBeDefined();
    expect(heartbeatModule.HeartbeatSystem).toBeDefined();
  });
});
