import { renderHook, act, waitFor } from '@testing-library/react';
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { createSocketService } from '../services/socketService';

class MockWebSocket {
  constructor(url) {
    this.url = url;
    this.readyState = 0;
    this.CONNECTING = 0;
    this.OPEN = 1;
    this.CLOSING = 2;
    this.CLOSED = 3;
    this.onopen = null;
    this.onmessage = null;
    this.onclose = null;
    this.onerror = null;
    this.sentMessages = [];
  }

  set onopen(handler) {
    this._onopen = handler;
    setTimeout(() => {
      this.readyState = 1;
      if (handler) handler({ type: 'open' });
    }, 0);
  }

  get onopen() {
    return this._onopen;
  }

  send(data) {
    this.sentMessages.push(JSON.parse(data));
  }

  close(code = 1000, reason = '') {
    this.readyState = 3;
    if (this.onclose) {
      this.onclose({ code, reason, type: 'close' });
    }
  }

  mockMessage(data) {
    if (this.onmessage) {
      this.onmessage({ data: JSON.stringify(data) });
    }
  }

  mockError(error) {
    if (this.onerror) {
      this.onerror({ error });
    }
  }
}

global.WebSocket = MockWebSocket;

describe('WebSocket Service', () => {
  let socketService;

  beforeEach(() => {
    vi.useFakeTimers();
    socketService = createSocketService({
      url: 'ws://localhost:3000',
      maxReconnectAttempts: 3,
      reconnectDelay: 1000,
      heartbeatIntervalTime: 1000,
      heartbeatTimeoutTime: 500,
      autoConnect: false
    });
  });

  afterEach(() => {
    socketService.disconnect();
    socketService = null;
    vi.useRealTimers();
  });

  describe('Connection Management', () => {
    test('should initialize with default state', () => {
      expect(socketService.getConnectionState()).toBe('disconnected');
      expect(socketService.isConnected()).toBe(false);
    });

    test('should connect successfully', async () => {
      const connectionPromise = new Promise((resolve) => {
        socketService.onConnectionChange((state) => {
          if (state === 'connected') resolve();
        });
      });

      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      await act(async () => {
        await connectionPromise;
      });

      expect(socketService.getConnectionState()).toBe('connected');
      expect(socketService.isConnected()).toBe(true);
    });

    test('should not connect if already connecting', () => {
      socketService.connect('user1');
      socketService.connect('user2');

      expect(socketService.getConnectionState()).toBe('connecting');
    });

    test('should disconnect properly', async () => {
      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      socketService.disconnect();

      expect(socketService.getConnectionState()).toBe('disconnected');
      expect(socketService.isConnected()).toBe(false);
    });
  });

  describe('Message Handling', () => {
    test('should send message when connected', async () => {
      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      const result = socketService.send({ type: 'test', data: 'hello' });

      expect(result).toBe(true);
    });

    test('should queue messages when disconnected', () => {
      const result = socketService.send({ type: 'test', data: 'hello' });

      expect(result).toBe(false);
      expect(socketService.messageQueue.length).toBe(1);
    });

    test('should flush message queue on reconnection', async () => {
      socketService.send({ type: 'test', data: 'queued' });

      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      expect(socketService.messageQueue.length).toBe(0);
    });
  });

  describe('Heartbeat', () => {
    test('should start heartbeat after connection', async () => {
      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      expect(socketService.heartbeatInterval).not.toBeNull();
    });

    test('should stop heartbeat on disconnect', async () => {
      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      socketService.disconnect();

      expect(socketService.heartbeatInterval).toBeNull();
    });
  });

  describe('Event Listeners', () => {
    test('should register and emit events', async () => {
      const callback = vi.fn();
      socketService.on('test_event', callback);

      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      socketService.emit('test_event', { data: 'test' });

      expect(callback).toHaveBeenCalledWith({ data: 'test' });
    });

    test('should remove event listeners', async () => {
      const callback = vi.fn();
      socketService.on('test_event', callback);
      socketService.off('test_event', callback);

      socketService.emit('test_event', { data: 'test' });

      expect(callback).not.toHaveBeenCalled();
    });

    test('should return unsubscribe function from on()', async () => {
      const callback = vi.fn();
      const unsubscribe = socketService.on('test_event', callback);

      unsubscribe();

      socketService.emit('test_event', { data: 'test' });

      expect(callback).not.toHaveBeenCalled();
    });
  });

  describe('Connection State', () => {
    test('should notify connection state changes', async () => {
      const states = [];
      socketService.onConnectionChange((state) => {
        states.push(state);
      });

      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      expect(states).toContain('connecting');
      expect(states).toContain('connected');
    });

    test('should set reconnecting state on disconnect', async () => {
      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      const states = [];
      socketService.onConnectionChange((state) => {
        states.push(state);
      });

      socketService.disconnect();

      expect(states).toContain('disconnected');
    });
  });

  describe('Statistics', () => {
    test('should return connection statistics', async () => {
      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      const stats = socketService.getStats();

      expect(stats).toHaveProperty('connectionState', 'connected');
      expect(stats).toHaveProperty('reconnectAttempts', 0);
      expect(stats).toHaveProperty('queuedMessages', 0);
      expect(stats).toHaveProperty('heartbeatActive', true);
    });
  });

  describe('Message Types', () => {
    test('should handle pong response', async () => {
      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      socketService.socket.mockMessage({ type: 'pong' });

      expect(socketService.heartbeatTimeout).toBeNull();
    });

    test('should handle notification message', async () => {
      const callback = vi.fn();
      socketService.on('notification', callback);

      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      socketService.socket.mockMessage({
        type: 'notification',
        data: { id: '1', message: 'Test' }
      });

      expect(callback).toHaveBeenCalled();
    });
  });

  describe('Error Handling', () => {
    test('should handle message parse errors', async () => {
      const errorCallback = vi.fn();
      socketService.on('error', errorCallback);

      socketService.connect('user123');

      await act(async () => {
        vi.runAllTimers();
      });

      socketService.socket.mockMessage({ invalid: 'json' });

      expect(errorCallback).toHaveBeenCalled();
    });
  });
});
