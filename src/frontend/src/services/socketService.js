class WebSocketService {
  constructor(options = {}) {
    this.url = options.url || 'ws://localhost:3000';
    this.socket = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = options.maxReconnectAttempts || 5;
    this.reconnectDelay = options.reconnectDelay || 1000;
    this.heartbeatInterval = null;
    this.heartbeatTimeout = null;
    this.heartbeatIntervalTime = options.heartbeatIntervalTime || 30000;
    this.heartbeatTimeoutTime = options.heartbeatTimeoutTime || 5000;
    this.listeners = new Map();
    this.connectionState = 'disconnected';
    this.connectionListeners = new Set();
    this.messageQueue = [];
    this.autoConnect = options.autoConnect !== false;
    this.userId = null;
  }

  connect(userId = null) {
    if (this.connectionState === 'connected' || this.connectionState === 'connecting') {
      console.warn('WebSocket already connected or connecting');
      return;
    }

    this.userId = userId;
    this.setConnectionState('connecting');

    try {
      const wsUrl = userId ? `${this.url}?userId=${userId}` : this.url;
      this.socket = new WebSocket(wsUrl);

      this.socket.onopen = this.handleOpen.bind(this);
      this.socket.onmessage = this.handleMessage.bind(this);
      this.socket.onclose = this.handleClose.bind(this);
      this.socket.onerror = this.handleError.bind(this);
    } catch (error) {
      console.error('WebSocket connection error:', error);
      this.setConnectionState('error');
      this.scheduleReconnect();
    }
  }

  disconnect() {
    this.stopHeartbeat();
    this.reconnectAttempts = this.maxReconnectAttempts;

    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }

    this.setConnectionState('disconnected');
    this.clearMessageQueue();
  }

  handleOpen() {
    console.log('WebSocket connected');
    this.setConnectionState('connected');
    this.reconnectAttempts = 0;
    this.startHeartbeat();
    this.flushMessageQueue();
  }

  handleMessage(event) {
    try {
      const data = JSON.parse(event.data);

      if (data.type === 'pong') {
        this.handlePong();
        return;
      }

      if (data.type === 'notification' || data.type === 'data:update' || data.type === 'reconnected') {
        this.emit(data.type, data);
      }

      this.emit('message', data);
    } catch (error) {
      console.error('Error parsing WebSocket message:', error);
      this.emit('error', { message: 'Failed to parse message', error });
    }
  }

  handleClose(event) {
    console.log('WebSocket closed:', event.code, event.reason);
    this.stopHeartbeat();
    this.setConnectionState('disconnected');

    if (event.code !== 1000) {
      this.scheduleReconnect();
    }
  }

  handleError(error) {
    console.error('WebSocket error:', error);
    this.setConnectionState('error');
    this.emit('error', error);
  }

  handlePong() {
    if (this.heartbeatTimeout) {
      clearTimeout(this.heartbeatTimeout);
      this.heartbeatTimeout = null;
    }
  }

  startHeartbeat() {
    this.stopHeartbeat();

    this.heartbeatInterval = setInterval(() => {
      if (this.connectionState === 'connected' && this.socket?.readyState === WebSocket.OPEN) {
        this.send({ type: 'ping' });

        this.heartbeatTimeout = setTimeout(() => {
          console.warn('Heartbeat timeout, reconnecting...');
          this.socket?.close();
          this.scheduleReconnect();
        }, this.heartbeatTimeoutTime);
      }
    }, this.heartbeatIntervalTime);
  }

  stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }

    if (this.heartbeatTimeout) {
      clearTimeout(this.heartbeatTimeout);
      this.heartbeatTimeout = null;
    }
  }

  scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      this.setConnectionState('failed');
      this.emit('reconnect_failed', {
        attempts: this.reconnectAttempts,
        maxAttempts: this.maxReconnectAttempts
      });
      return;
    }

    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts);
    console.log(`Scheduling reconnect in ${delay}ms (attempt ${this.reconnectAttempts + 1})`);

    this.setConnectionState('reconnecting');

    setTimeout(() => {
      this.reconnectAttempts++;
      this.connect(this.userId);
    }, delay);
  }

  send(data) {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(data));
      return true;
    } else {
      this.queueMessage(data);
      return false;
    }
  }

  queueMessage(data) {
    if (this.messageQueue.length < 100) {
      this.messageQueue.push(data);
    }
  }

  flushMessageQueue() {
    while (this.messageQueue.length > 0) {
      const message = this.messageQueue.shift();
      this.send(message);
    }
  }

  clearMessageQueue() {
    this.messageQueue = [];
  }

  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event).add(callback);

    return () => this.off(event, callback);
  }

  off(event, callback) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).delete(callback);
    }
  }

  emit(event, data) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).forEach(callback => {
        try {
          callback(data);
        } catch (error) {
          console.error(`Error in listener for ${event}:`, error);
        }
      });
    }
  }

  onConnectionChange(callback) {
    this.connectionListeners.add(callback);
    return () => this.connectionListeners.delete(callback);
  }

  setConnectionState(state) {
    if (this.connectionState !== state) {
      this.connectionState = state;
      this.connectionListeners.forEach(callback => {
        try {
          callback(state);
        } catch (error) {
          console.error('Error in connection listener:', error);
        }
      });
    }
  }

  getConnectionState() {
    return this.connectionState;
  }

  isConnected() {
    return this.connectionState === 'connected';
  }

  getStats() {
    return {
      connectionState: this.connectionState,
      reconnectAttempts: this.reconnectAttempts,
      queuedMessages: this.messageQueue.length,
      heartbeatActive: this.heartbeatInterval !== null
    };
  }
}

let socketServiceInstance = null;

export function getSocketService(options = {}) {
  if (!socketServiceInstance) {
    socketServiceInstance = new WebSocketService(options);
  }
  return socketServiceInstance;
}

export function createSocketService(options = {}) {
  return new WebSocketService(options);
}

export default WebSocketService;
