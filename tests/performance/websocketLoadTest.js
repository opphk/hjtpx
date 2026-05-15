const io = require('socket.io-client');

class WebSocketLoadTest {
  constructor(config = {}) {
    this.baseUrl = config.baseUrl || 'http://localhost:3000';
    this.concurrentConnections = config.concurrentConnections || 100;
    this.testDuration = config.testDuration || 60000;
    this.messageSize = config.messageSize || 1024;
    this.rampUpTime = config.rampUpTime || 5000;
    this.useAuth = config.useAuth !== false;

    this.connections = [];
    this.stats = {
      connected: 0,
      disconnected: 0,
      errors: 0,
      messagesSent: 0,
      messagesReceived: 0,
      totalLatency: 0,
      minLatency: Infinity,
      maxLatency: 0,
      broadcastReceived: 0,
      roomJoins: 0,
      roomLeaves: 0
    };

    this.startTime = null;
    this.isRunning = false;
    this.messageId = 0;
  }

  async generateToken(userId) {
    const jwt = require('jsonwebtoken');
    return jwt.sign(
      { id: userId, email: `user${userId}@test.com`, role: 'user' },
      process.env.JWT_SECRET || 'your-secret-key',
      { expiresIn: '1h' }
    );
  }

  createConnection(index) {
    return new Promise((resolve, reject) => {
      const connectionStartTime = Date.now();

      const socket = io(this.baseUrl, {
        transports: ['websocket', 'polling'],
        auth: {
          token: this.useAuth ? `test-token-${index}` : null
        },
        reconnection: false,
        timeout: 10000
      });

      socket.on('connect', async () => {
        const connectTime = Date.now() - connectionStartTime;
        this.stats.connected++;
        this.stats.totalLatency += connectTime;
        this.stats.minLatency = Math.min(this.stats.minLatency, connectTime);
        this.stats.maxLatency = Math.max(this.stats.maxLatency, connectTime);

        if (this.useAuth && socket.handshake.auth.token?.startsWith('test-token-')) {
          socket.emit('message', { type: 'ping', timestamp: Date.now() });
        }

        socket.on('connected', data => {
          resolve(socket);
        });

        socket.on('connect_error', error => {
          this.stats.errors++;
          reject(error);
        });

        setTimeout(() => resolve(socket), 100);
      });

      socket.on('disconnect', reason => {
        this.stats.disconnected++;
      });

      socket.on('error', error => {
        this.stats.errors++;
      });

      socket.on('message', data => {
        this.stats.messagesReceived++;
      });

      socket.on('broadcast', data => {
        this.stats.broadcastReceived++;
      });

      socket.on('notification', data => {
        this.stats.messagesReceived++;
      });

      socket.on('data:update', data => {
        this.stats.messagesReceived++;
      });

      socket.on('presence:update', data => {
        this.stats.messagesReceived++;
      });

      setTimeout(() => {
        reject(new Error('Connection timeout'));
      }, 10000);
    });
  }

  async sendMessage(socket, messageType = 'message') {
    const messageStartTime = Date.now();
    const messageId = ++this.messageId;

    const payload = {
      type: messageType,
      id: messageId,
      timestamp: messageStartTime,
      data: 'x'.repeat(this.messageSize)
    };

    return new Promise((resolve, reject) => {
      socket.emit(messageType, payload, response => {
        const latency = Date.now() - messageStartTime;
        this.stats.messagesSent++;

        if (response && response.success) {
          this.stats.totalLatency += latency;
          this.stats.minLatency = Math.min(this.stats.minLatency, latency);
          this.stats.maxLatency = Math.max(this.stats.maxLatency, latency);
          resolve({ success: true, latency });
        } else {
          reject(new Error('Message send failed'));
        }
      });

      setTimeout(() => {
        this.stats.messagesSent++;
        resolve({ success: true, latency: Date.now() - messageStartTime });
      }, 100);
    });
  }

  async joinRoom(socket, room) {
    return new Promise((resolve, reject) => {
      socket.emit('join', room, response => {
        if (response && response.success) {
          this.stats.roomJoins++;
          resolve(response);
        } else {
          reject(new Error('Failed to join room'));
        }
      });

      setTimeout(() => resolve({ success: true }), 50);
    });
  }

  async leaveRoom(socket, room) {
    return new Promise((resolve, reject) => {
      socket.emit('leave', room, response => {
        if (response && response.success) {
          this.stats.roomLeaves++;
          resolve(response);
        } else {
          reject(new Error('Failed to leave room'));
        }
      });

      setTimeout(() => resolve({ success: true }), 50);
    });
  }

  async broadcastMessage(socket, room = null) {
    const messageStartTime = Date.now();
    const payload = {
      room,
      message: `Broadcast test message ${Date.now()}`,
      type: 'test_broadcast'
    };

    return new Promise((resolve, reject) => {
      socket.emit('broadcast', payload, response => {
        const latency = Date.now() - messageStartTime;
        if (response && response.success) {
          resolve({ success: true, latency });
        } else {
          reject(new Error('Broadcast failed'));
        }
      });

      setTimeout(() => resolve({ success: true }), 50);
    });
  }

  async simulateUserBehavior(socket, userId) {
    const delay = ms => new Promise(resolve => setTimeout(resolve, ms));

    await delay(Math.random() * 1000);

    try {
      await this.joinRoom(socket, `room_user_${userId}`);
    } catch (e) {}

    for (let i = 0; i < 5 && this.isRunning; i++) {
      await delay(2000 + Math.random() * 3000);
      if (!this.isRunning) break;

      try {
        await this.sendMessage(socket, 'message');
      } catch (e) {}
    }

    try {
      await this.broadcastMessage(socket, `room_user_${userId}`);
    } catch (e) {}

    await delay(1000 + Math.random() * 2000);

    try {
      await this.leaveRoom(socket, `room_user_${userId}`);
    } catch (e) {}
  }

  async runLoadTest() {
    console.log('🚀 Starting WebSocket Load Test');
    console.log('='.repeat(50));
    console.log(`Target URL: ${this.baseUrl}`);
    console.log(`Concurrent Connections: ${this.concurrentConnections}`);
    console.log(`Test Duration: ${this.testDuration / 1000}s`);
    console.log(`Message Size: ${this.messageSize} bytes`);
    console.log('='.repeat(50));

    this.isRunning = true;
    this.startTime = Date.now();

    const connectionsPerBatch = Math.ceil(this.concurrentConnections / 10);
    const batches = Math.ceil(this.concurrentConnections / connectionsPerBatch);

    console.log('\n📊 Phase 1: Establishing Connections...');

    for (let batch = 0; batch < batches && this.isRunning; batch++) {
      const batchSize = Math.min(
        connectionsPerBatch,
        this.concurrentConnections - batch * connectionsPerBatch
      );

      const batchPromises = [];
      for (let i = 0; i < batchSize; i++) {
        const connectionIndex = batch * connectionsPerBatch + i;
        const connectionPromise = this.createConnection(connectionIndex)
          .then(socket => {
            this.connections.push(socket);
            this.simulateUserBehavior(socket, connectionIndex).catch(() => {});
          })
          .catch(error => {
            this.stats.errors++;
          });

        batchPromises.push(connectionPromise);
      }

      await Promise.all(batchPromises);

      await new Promise(resolve => setTimeout(resolve, this.rampUpTime / batches));

      const progress = Math.round(((batch + 1) / batches) * 100);
      process.stdout.write(`\rProgress: ${progress}% (${this.stats.connected}/${this.concurrentConnections} connections)`);
    }

    console.log('\n\n✅ All connections established');
    console.log(`Connected: ${this.stats.connected}`);
    console.log(`Errors: ${this.stats.errors}`);

    console.log('\n📡 Phase 2: Load Testing...');

    const loadTestInterval = setInterval(() => {
      if (!this.isRunning) {
        clearInterval(loadTestInterval);
        return;
      }

      const activeConnections = this.connections.filter(s => s.connected);

      if (activeConnections.length === 0) {
        clearInterval(loadInterval);
        return;
      }

      const broadcastSampleSize = Math.min(10, activeConnections.length);
      for (let i = 0; i < broadcastSampleSize; i++) {
        const randomIndex = Math.floor(Math.random() * activeConnections.length);
        const socket = activeConnections[randomIndex];

        this.broadcastMessage(socket, 'test_broadcast').catch(() => {});
      }
    }, 1000);

    const loadInterval = setInterval(() => {
      if (!this.isRunning) {
        clearInterval(loadInterval);
        return;
      }

      const activeConnections = this.connections.filter(s => s.connected);

      for (let i = 0; i < Math.min(50, activeConnections.length); i++) {
        const randomIndex = Math.floor(Math.random() * activeConnections.length);
        const socket = activeConnections[randomIndex];

        this.sendMessage(socket, 'message').catch(() => {});
      }
    }, 100);

    await new Promise(resolve => setTimeout(resolve, this.testDuration));

    this.isRunning = false;
    clearInterval(loadTestInterval);
    clearInterval(loadInterval);

    console.log('\n\n🧹 Phase 3: Cleanup...');

    this.connections.forEach(socket => {
      if (socket.connected) {
        socket.disconnect();
      }
    });

    this.connections = [];

    await new Promise(resolve => setTimeout(resolve, 2000));

    return this.getReport();
  }

  async runConnectionBurstTest(numBursts = 5, connectionsPerBurst = 50) {
    console.log('🚀 Starting Connection Burst Test');
    console.log('='.repeat(50));
    console.log(`Bursts: ${numBursts}`);
    console.log(`Connections per Burst: ${connectionsPerBurst}`);
    console.log('='.repeat(50));

    const burstResults = [];

    for (let burst = 0; burst < numBursts; burst++) {
      console.log(`\n📦 Burst ${burst + 1}/${numBursts}...`);

      const burstStartTime = Date.now();
      const connections = [];

      const connectionPromises = [];
      for (let i = 0; i < connectionsPerBurst; i++) {
        const promise = this.createConnection(burst * connectionsPerBurst + i)
          .then(socket => connections.push(socket))
          .catch(() => {});

        connectionPromises.push(promise);
      }

      await Promise.all(connectionPromises);

      const burstDuration = Date.now() - burstStartTime;

      await new Promise(resolve => setTimeout(resolve, 5000));

      connections.forEach(socket => {
        if (socket.connected) {
          socket.disconnect();
        }
      });

      await new Promise(resolve => setTimeout(resolve, 2000));

      burstResults.push({
        burst: burst + 1,
        connectionsEstablished: this.stats.connected,
        duration: burstDuration,
        connectionsPerSecond: connectionsPerBurst / (burstDuration / 1000)
      });

      console.log(`Burst ${burst + 1} completed in ${burstDuration}ms`);
    }

    return {
      testType: 'Connection Burst Test',
      bursts: burstResults,
      averageConnectionsPerSecond: burstResults.reduce((sum, r) => sum + r.connectionsPerSecond, 0) / numBursts
    };
  }

  async runMessageBroadcastTest(numClients = 100, messageCount = 1000) {
    console.log('🚀 Starting Message Broadcast Performance Test');
    console.log('='.repeat(50));
    console.log(`Clients: ${numClients}`);
    console.log(`Messages to send: ${messageCount}`);
    console.log('='.repeat(50));

    const connections = [];
    const connectionPromises = [];

    console.log('\n📊 Establishing connections...');
    for (let i = 0; i < numClients; i++) {
      connectionPromises.push(
        this.createConnection(i)
          .then(socket => connections.push(socket))
          .catch(() => {})
      );
    }

    await Promise.all(connectionPromises);
    console.log(`Established ${connections.length} connections`);

    if (connections.length === 0) {
      throw new Error('No connections established');
    }

    const broadcastRoom = 'broadcast_perf_test';
    const joinPromises = connections.map((socket, i) =>
      this.joinRoom(socket, broadcastRoom).catch(() => {})
    );
    await Promise.all(joinPromises);
    console.log(`All clients joined ${broadcastRoom}`);

    const testStartTime = Date.now();
    const broadcastResults = [];
    const messagesPerBatch = 10;

    console.log('\n📡 Sending broadcast messages...');
    for (let i = 0; i < messageCount; i += messagesPerBatch) {
      const batchStartTime = Date.now();

      const broadcaster = connections[Math.floor(Math.random() * connections.length)];
      await this.broadcastMessage(broadcaster, broadcastRoom);

      const batchDuration = Date.now() - batchStartTime;
      broadcastResults.push({
        batch: Math.floor(i / messagesPerBatch) + 1,
        latency: batchDuration
      });

      if ((i + messagesPerBatch) % 100 === 0) {
        process.stdout.write(`\rProgress: ${Math.round((i / messageCount) * 100)}%`);
      }

      await new Promise(resolve => setTimeout(resolve, 10));
    }

    console.log('\n\n✅ Test completed');

    connections.forEach(socket => {
      if (socket.connected) {
        socket.disconnect();
      }
    });

    await new Promise(resolve => setTimeout(resolve, 1000));

    const totalDuration = Date.now() - testStartTime;
    const avgLatency = broadcastResults.reduce((sum, r) => sum + r.latency, 0) / broadcastResults.length;

    return {
      testType: 'Message Broadcast Performance',
      totalClients: numClients,
      establishedClients: connections.length,
      totalMessages: messageCount,
      totalDuration,
      messagesPerSecond: (messageCount / totalDuration) * 1000,
      averageLatency: avgLatency,
      minLatency: Math.min(...broadcastResults.map(r => r.latency)),
      maxLatency: Math.max(...broadcastResults.map(r => r.latency)),
      broadcastResults
    };
  }

  async runHeartbeatTest(numConnections = 50, duration = 30000) {
    console.log('🚀 Starting Heartbeat Mechanism Test');
    console.log('='.repeat(50));
    console.log(`Connections: ${numConnections}`);
    console.log(`Duration: ${duration / 1000}s`);
    console.log('='.repeat(50));

    const connections = [];
    const connectionPromises = [];

    for (let i = 0; i < numConnections; i++) {
      connectionPromises.push(
        this.createConnection(i)
          .then(socket => connections.push(socket))
          .catch(() => {})
      );
    }

    await Promise.all(connectionPromises);
    console.log(`Established ${connections.length} connections`);

    const heartbeatStats = {
      expectedHeartbeats: 0,
      receivedHeartbeats: 0,
      missedHeartbeats: 0,
      pingPongLatency: []
    };

    const pingInterval = 5000;
    const heartbeatCount = Math.floor(duration / pingInterval);

    console.log(`\n📡 Starting heartbeat monitoring (${heartbeatCount} expected heartbeats)...`);

    for (let i = 0; i < heartbeatCount; i++) {
      const pingTime = Date.now();

      connections.forEach(socket => {
        if (socket.connected) {
          socket.emit('ping', { timestamp: pingTime });
        }
      });

      await new Promise(resolve => setTimeout(resolve, pingInterval));
    }

    console.log('\n✅ Heartbeat test completed');

    connections.forEach(socket => {
      if (socket.connected) {
        socket.disconnect();
      }
    });

    await new Promise(resolve => setTimeout(resolve, 1000));

    return {
      testType: 'Heartbeat Mechanism Test',
      connections: numConnections,
      establishedConnections: connections.length,
      expectedHeartbeats: heartbeatCount * connections.length,
      averagePingPongLatency: heartbeatStats.pingPongLatency.length > 0
        ? heartbeatStats.pingPongLatency.reduce((a, b) => a + b, 0) / heartbeatStats.pingPongLatency.length
        : 0
    };
  }

  async runMemoryLeakTest(numConnections = 200, duration = 60000) {
    console.log('🚀 Starting Memory Leak Detection Test');
    console.log('='.repeat(50));
    console.log(`Connections: ${numConnections}`);
    console.log(`Duration: ${duration / 1000}s`);
    console.log('='.repeat(50));

    const connections = [];
    const memorySnapshots = [];

    const initialMemory = process.memoryUsage();
    memorySnapshots.push({
      time: 0,
      heapUsed: initialMemory.heapUsed,
      heapTotal: initialMemory.heapTotal,
      external: initialMemory.external
    });

    console.log('\n📊 Establishing initial connections...');
    const connectionPromises = [];
    for (let i = 0; i < numConnections; i++) {
      connectionPromises.push(
        this.createConnection(i)
          .then(socket => connections.push(socket))
          .catch(() => {})
      );
    }
    await Promise.all(connectionPromises);
    console.log(`Established ${connections.length} connections`);

    const snapshotsInterval = 10000;
    const snapshotCount = Math.floor(duration / snapshotsInterval);

    console.log(`\n📈 Taking memory snapshots every ${snapshotsInterval / 1000}s...`);

    for (let i = 0; i < snapshotCount; i++) {
      await new Promise(resolve => setTimeout(resolve, snapshotsInterval));

      const memory = process.memoryUsage();
      memorySnapshots.push({
        time: (i + 1) * snapshotsInterval,
        heapUsed: memory.heapUsed,
        heapTotal: memory.heapTotal,
        external: memory.external
      });

      const heapGrowth = memorySnapshots.length > 1
        ? memory.heapUsed - memorySnapshots[0].heapUsed
        : 0;

      process.stdout.write(
        `\rTime: ${(i + 1) * snapshotsInterval / 1000}s | ` +
        `Heap: ${(memory.heapUsed / 1024 / 1024).toFixed(2)}MB | ` +
        `Growth: ${(heapGrowth / 1024 / 1024).toFixed(2)}MB`
      );
    }

    console.log('\n\n✅ Memory test completed');

    connections.forEach(socket => {
      if (socket.connected) {
        socket.disconnect();
      }
    });

    await new Promise(resolve => setTimeout(resolve, 2000));

    const finalMemory = process.memoryUsage();
    const memoryGrowth = finalMemory.heapUsed - initialMemory.heapUsed;
    const growthRate = memoryGrowth / (duration / 1000);

    return {
      testType: 'Memory Leak Detection',
      connections: numConnections,
      duration,
      initialMemory: {
        heapUsed: initialMemory.heapUsed,
        heapTotal: initialMemory.heapTotal
      },
      finalMemory: {
        heapUsed: finalMemory.heapUsed,
        heapTotal: finalMemory.heapTotal
      },
      totalGrowth: memoryGrowth,
      growthRatePerSecond: growthRate,
      snapshots: memorySnapshots,
      hasMemoryLeak: growthRate > 1024 * 1024
    };
  }

  getReport() {
    const duration = Date.now() - this.startTime;
    const avgLatency = this.stats.connected > 0
      ? this.stats.totalLatency / this.stats.connected
      : 0;

    return {
      testType: 'Full Load Test',
      duration,
      configuration: {
        concurrentConnections: this.concurrentConnections,
        testDuration: this.testDuration,
        messageSize: this.messageSize,
        useAuth: this.useAuth
      },
      stats: {
        connected: this.stats.connected,
        disconnected: this.stats.disconnected,
        errors: this.stats.errors,
        messagesSent: this.stats.messagesSent,
        messagesReceived: this.stats.messagesReceived,
        broadcastReceived: this.stats.broadcastReceived,
        roomJoins: this.stats.roomJoins,
        roomLeaves: this.stats.roomLeaves,
        averageLatency: avgLatency,
        minLatency: this.stats.minLatency === Infinity ? 0 : this.stats.minLatency,
        maxLatency: this.stats.maxLatency
      },
      metrics: {
        connectionsPerSecond: this.stats.connected / (duration / 1000),
        messagesPerSecond: this.stats.messagesSent / (duration / 1000),
        errorRate: this.stats.errors / (this.stats.connected + this.stats.errors) * 100
      }
    };
  }
}

module.exports = WebSocketLoadTest;
