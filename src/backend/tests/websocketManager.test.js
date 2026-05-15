const WebSocket = require('ws');
const websocketManager = require('../../backend/services/websocketManager');

describe('WebSocket Manager', () => {
  let mockHttpServer;
  let mockSocket;

  beforeEach(() => {
    mockHttpServer = {
      on: jest.fn(),
      listen: jest.fn((port, callback) => callback())
    };

    mockSocket = {
      on: jest.fn(),
      send: jest.fn(),
      close: jest.fn(),
      terminate: jest.fn(),
      clientId: 'test-client-id',
      isAlive: true,
      readyState: WebSocket.OPEN,
      rooms: new Set(),
      userId: null,
      isAuthenticated: false
    };
  });

  afterEach(() => {
    if (websocketManager.io) {
      websocketManager.close();
    }
  });

  describe('Initialization', () => {
    test('should initialize with http server', () => {
      websocketManager.initialize(mockHttpServer);

      expect(websocketManager.io).not.toBeNull();
      expect(websocketManager.heartbeatInterval).not.toBeNull();
    });

    test('should not reinitialize if already initialized', () => {
      websocketManager.initialize(mockHttpServer);
      const firstIo = websocketManager.io;

      websocketManager.initialize(mockHttpServer);

      expect(websocketManager.io).toBe(firstIo);
    });
  });

  describe('Connection Handling', () => {
    test('should generate unique client ID', () => {
      const id1 = websocketManager.generateClientId();
      const id2 = websocketManager.generateClientId();

      expect(id1).toMatch(/^client_/);
      expect(id2).toMatch(/^client_/);
      expect(id1).not.toBe(id2);
    });

    test('should send connected message on connection', () => {
      const sendSpy = jest.spyOn(websocketManager, 'sendToClient');

      websocketManager.handleConnection(mockSocket, {});

      expect(sendSpy).toHaveBeenCalledWith(mockSocket, expect.objectContaining({
        type: 'connected',
        clientId: expect.stringMatching(/^client_/)
      }));
    });

    test('should track client last activity', () => {
      websocketManager.handleConnection(mockSocket, {});

      expect(websocketManager.clientLastActivity.has(mockSocket.clientId)).toBe(true);
    });
  });

  describe('Message Handling', () => {
    beforeEach(() => {
      websocketManager.initialize(mockHttpServer);
    });

    test('should handle ping message', () => {
      const sendSpy = jest.spyOn(websocketManager, 'sendToClient');
      const data = JSON.stringify({ type: 'ping' });

      websocketManager.handleMessage(mockSocket, data);

      expect(sendSpy).toHaveBeenCalledWith(mockSocket, expect.objectContaining({
        type: 'pong'
      }));
    });

    test('should handle auth message', () => {
      const sendSpy = jest.spyOn(websocketManager, 'sendToClient');
      const data = JSON.stringify({
        type: 'auth',
        token: 'invalid-token'
      });

      websocketManager.handleMessage(mockSocket, data);

      expect(sendSpy).toHaveBeenCalledWith(mockSocket, expect.objectContaining({
        type: 'auth_error'
      }));
    });

    test('should handle subscribe message', () => {
      const sendSpy = jest.spyOn(websocketManager, 'sendToClient');
      const data = JSON.stringify({
        type: 'subscribe',
        room: 'test-room'
      });

      websocketManager.handleMessage(mockSocket, data);

      expect(sendSpy).toHaveBeenCalledWith(mockSocket, expect.objectContaining({
        type: 'subscribed',
        room: 'test-room'
      }));
      expect(mockSocket.rooms.has('test-room')).toBe(true);
    });

    test('should handle unsubscribe message', () => {
      mockSocket.rooms.add('test-room');
      const sendSpy = jest.spyOn(websocketManager, 'sendToClient');
      const data = JSON.stringify({
        type: 'unsubscribe',
        room: 'test-room'
      });

      websocketManager.handleMessage(mockSocket, data);

      expect(sendSpy).toHaveBeenCalledWith(mockSocket, expect.objectContaining({
        type: 'unsubscribed',
        room: 'test-room'
      }));
      expect(mockSocket.rooms.has('test-room')).toBe(false);
    });

    test('should handle unknown message type', () => {
      const sendSpy = jest.spyOn(websocketManager, 'sendToClient');
      const data = JSON.stringify({ type: 'unknown_type' });

      websocketManager.handleMessage(mockSocket, data);

      expect(sendSpy).not.toHaveBeenCalled();
    });

    test('should handle invalid JSON message', () => {
      const sendSpy = jest.spyOn(websocketManager, 'sendToClient');
      const data = 'invalid-json';

      websocketManager.handleMessage(mockSocket, data);

      expect(sendSpy).toHaveBeenCalledWith(mockSocket, expect.objectContaining({
        type: 'error',
        message: 'Invalid message format'
      }));
    });
  });

  describe('Broadcasting', () => {
    beforeEach(() => {
      websocketManager.initialize(mockHttpServer);
      websocketManager.io = {
        to: jest.fn().mockReturnThis(),
        emit: jest.fn(),
        clients: jest.fn().mockReturnValue([mockSocket])
      };
    });

    test('should broadcast to all clients', () => {
      websocketManager.broadcast('test_event', { data: 'test' });

      expect(websocketManager.io.emit).toHaveBeenCalledWith('test_event', { data: 'test' });
    });

    test('should broadcast to specific room', () => {
      websocketManager.broadcast('test_event', { data: 'test' }, 'test-room');

      expect(websocketManager.io.to).toHaveBeenCalledWith('test-room');
      expect(websocketManager.io.emit).toHaveBeenCalled();
    });
  });

  describe('Send To Client', () => {
    test('should send message when socket is open', () => {
      mockSocket.readyState = WebSocket.OPEN;

      const result = websocketManager.sendToClient(mockSocket, { type: 'test' });

      expect(result).toBe(true);
      expect(mockSocket.send).toHaveBeenCalledWith(JSON.stringify({ type: 'test' }));
    });

    test('should not send message when socket is closed', () => {
      mockSocket.readyState = WebSocket.CLOSED;

      const result = websocketManager.sendToClient(mockSocket, { type: 'test' });

      expect(result).toBe(false);
      expect(mockSocket.send).not.toHaveBeenCalled();
    });
  });

  describe('Connection Statistics', () => {
    beforeEach(() => {
      websocketManager.initialize(mockHttpServer);
      websocketManager.io = {
        clients: jest.fn().mockReturnValue([
          { clientId: '1', userId: 'user1', isAuthenticated: true, rooms: new Set() },
          { clientId: '2', userId: null, isAuthenticated: false, rooms: new Set() },
          { clientId: '3', userId: 'user2', isAuthenticated: true, rooms: new Set() }
        ])
      };
    });

    test('should return connection stats', () => {
      const stats = websocketManager.getConnectionStats();

      expect(stats.totalConnections).toBe(3);
      expect(stats.onlineUsers).toBe(2);
      expect(stats.authenticatedConnections).toBe(2);
      expect(stats.anonymousConnections).toBe(1);
    });

    test('should return connected clients', () => {
      const clients = websocketManager.getConnectedClients();

      expect(clients).toHaveLength(3);
      expect(clients[0]).toHaveProperty('clientId');
      expect(clients[0]).toHaveProperty('userId');
      expect(clients[0]).toHaveProperty('isAuthenticated');
    });

    test('should return online users', () => {
      const users = websocketManager.getOnlineUsers();

      expect(users).toContain('user1');
      expect(users).toContain('user2');
      expect(users).not.toContain(null);
    });
  });

  describe('Cleanup', () => {
    beforeEach(() => {
      websocketManager.initialize(mockHttpServer);
      websocketManager.io = {
        clients: jest.fn().mockReturnValue([mockSocket]),
        close: jest.fn()
      };
    });

    test('should close all connections on close', () => {
      websocketManager.close();

      expect(mockSocket.close).toHaveBeenCalled();
      expect(websocketManager.io.close).toHaveBeenCalled();
      expect(websocketManager.io).toBeNull();
    });

    test('should clear heartbeat interval on close', () => {
      const clearIntervalSpy = jest.spyOn(global, 'clearInterval');

      websocketManager.close();

      expect(clearIntervalSpy).toHaveBeenCalled();
    });
  });

  describe('Heartbeat Management', () => {
    beforeEach(() => {
      websocketManager.initialize(mockHttpServer);
      websocketManager.io = {
        clients: jest.fn().mockReturnValue([mockSocket]),
        close: jest.fn()
      };
    });

    test('should mark socket as not alive on ping', () => {
      mockSocket.on.mock.calls[1][1]();

      expect(mockSocket.isAlive).toBe(false);
    });

    test('should terminate inactive sockets', () => {
      mockSocket.isAlive = false;
      websocketManager.io.clients.mockReturnValue([mockSocket]);

      websocketManager.checkHeartbeats();

      expect(mockSocket.terminate).toHaveBeenCalled();
    });
  });
});
