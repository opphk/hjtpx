const io = require('socket.io-client');
const http = require('http');
const jwt = require('jsonwebtoken');

const TEST_CONFIG = {
  baseUrl: process.env.WS_TEST_URL || 'http://localhost:3001',
  concurrentConnections: parseInt(process.env.WS_CONCURRENT_CONNECTIONS) || 100,
  testDuration: parseInt(process.env.WS_TEST_DURATION) || 60000,
  messageSize: parseInt(process.env.WS_MESSAGE_SIZE) || 1024,
  broadcastInterval: parseInt(process.env.WS_BROADCAST_INTERVAL) || 1000,
  heartbeatInterval: parseInt(process.env.WS_HEARTBEAT_INTERVAL) || 5000
};

class WebSocketLoadTest {
  constructor() {
    this.sockets = [];
    this.metrics = {
      connectionSuccess: 0,
      connectionFailure: 0,
      messagesSent: 0,
      messagesReceived: 0,
      broadcastsReceived: 0,
      heartbeatsSent: 0,
      heartbeatsReceived: 0,
      errors: 0,
      disconnections: 0,
      connectionTimes: [],
      messageLatencies: [],
      startTime: null,
      endTime: null
    };
    this.testResults = [];
    this.isRunning = false;
    this.broadcastInterval = null;
    this.heartbeatInterval = null;
  }

  generateTestToken(userId) {
    return jwt.sign(
      { id: userId, email: `test${userId}@example.com` },
      process.env.JWT_SECRET || 'your-secret-key',
      { expiresIn: '1h' }
    );
  }

  generateMessage(size) {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < size; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  }

  createSocket(index) {
    return new Promise((resolve, reject) => {
      const userId = `test-user-${index}`;
      const token = this.generateTestToken(userId);
      
      const startTime = Date.now();
      
      const socket = io(TEST_CONFIG.baseUrl, {
        auth: { token },
        transports: ['websocket'],
        reconnection: false,
        timeout: 10000
      });

      socket.on('connect', () => {
        const connectionTime = Date.now() - startTime;
        this.metrics.connectionTimes.push(connectionTime);
        this.metrics.connectionSuccess++;
        
        socket.userId = userId;
        socket.connectionTime = connectionTime;
        socket.messagesReceived = 0;
        socket.broadcastsReceived = 0;
        socket.heartbeatsReceived = 0;
        socket.lastPingTime = Date.now();
        
        socket.emit('join', `test-room-${index % 10}`, (response) => {
          if (response.success) {
            socket.roomJoined = true;
          }
        });

        resolve(socket);
      });

      socket.on('connected', (data) => {
        this.metrics.messagesReceived++;
        socket.messagesReceived++;
      });

      socket.on('broadcast', (data) => {
        this.metrics.broadcastsReceived++;
        socket.broadcastsReceived++;
      });

      socket.on('pong', () => {
        const latency = Date.now() - socket.lastPingTime;
        this.metrics.heartbeatsReceived++;
        socket.heartbeatsReceived++;
        this.metrics.messageLatencies.push(latency);
      });

      socket.on('message', (data) => {
        this.metrics.messagesReceived++;
        socket.messagesReceived++;
      });

      socket.on('disconnect', (reason) => {
        this.metrics.disconnections++;
        socket._customDisconnected = true;
      });

      socket.on('connect_error', (error) => {
        this.metrics.connectionFailure++;
        this.metrics.errors++;
        console.error(`Connection failed for ${userId}:`, error.message);
        reject(error);
      });

      socket.on('error', (error) => {
        this.metrics.errors++;
        console.error(`Socket error for ${userId}:`, error.message);
      });

      setTimeout(() => {
        if (!socket.connected) {
          this.metrics.connectionFailure++;
          reject(new Error('Connection timeout'));
        }
      }, 10000);
    });
  }

  async connectClients(count) {
    console.log(`\nConnecting ${count} clients...`);
    const startTime = Date.now();
    
    const connections = [];
    for (let i = 0; i < count; i++) {
      connections.push(
        this.createSocket(i)
          .then(socket => {
            this.sockets.push(socket);
            if ((i + 1) % 10 === 0) {
              console.log(`  Connected ${i + 1}/${count} clients...`);
            }
            return socket;
          })
          .catch(error => {
            console.error(`  Failed to connect client ${i}:`, error.message);
          })
      );

      if (i % 20 === 0) {
        await new Promise(resolve => setTimeout(resolve, 100));
      }
    }

    await Promise.allSettled(connections);
    
    const duration = Date.now() - startTime;
    
    console.log(`\nConnection Summary:`);
    console.log(`  Total attempted: ${count}`);
    console.log(`  Successful: ${this.metrics.connectionSuccess}`);
    console.log(`  Failed: ${this.metrics.connectionFailure}`);
    console.log(`  Time taken: ${duration}ms`);
    console.log(`  Avg connection time: ${this.calculateAverage(this.metrics.connectionTimes)}ms`);
  }

  async testMessageBroadcast() {
    console.log(`\n--- Message Broadcast Test ---`);
    console.log(`Broadcasting messages to all connected clients...`);
    
    const testMessage = this.generateMessage(TEST_CONFIG.messageSize);
    const broadcastsPerClient = 5;
    
    const broadcastStart = Date.now();
    
    for (let i = 0; i < broadcastsPerClient; i++) {
      const promises = this.sockets.map(socket => {
        return new Promise((resolve) => {
          socket.emit('broadcast', {
            room: `test-room-${i % 10}`,
            message: testMessage,
            type: 'test-broadcast'
          }, (response) => {
            if (response.success) {
              this.metrics.messagesSent++;
            }
            resolve();
          });
        });
      });

      await Promise.all(promises);
      console.log(`  Broadcast round ${i + 1}/${broadcastsPerClient} completed`);
      await new Promise(resolve => setTimeout(resolve, 100));
    }

    const broadcastDuration = Date.now() - broadcastStart;
    
    console.log(`\nBroadcast Results:`);
    console.log(`  Total broadcasts sent: ${this.metrics.messagesSent}`);
    console.log(`  Total broadcasts received: ${this.metrics.broadcastsReceived}`);
    console.log(`  Time taken: ${broadcastDuration}ms`);
    console.log(`  Avg latency: ${this.calculateAverage(this.metrics.messageLatencies)}ms`);
    
    return {
      totalSent: this.metrics.messagesSent,
      totalReceived: this.metrics.broadcastsReceived,
      duration: broadcastDuration,
      avgLatency: this.calculateAverage(this.metrics.messageLatencies)
    };
  }

  async testHeartbeatStability() {
    console.log(`\n--- Heartbeat Stability Test ---`);
    console.log(`Testing heartbeat mechanism...`);
    
    const heartbeatDuration = 10000;
    const heartbeatInterval = 2000;
    const heartbeatsExpected = Math.floor(heartbeatDuration / heartbeatInterval);
    
    this.metrics.heartbeatsSent = 0;
    this.metrics.heartbeatsReceived = 0;
    
    const heartbeatTestStart = Date.now();
    
    return new Promise((resolve) => {
      this.heartbeatInterval = setInterval(() => {
        const pingStart = Date.now();
        
        this.sockets.forEach(socket => {
          if (socket.connected) {
            socket.lastPingTime = Date.now();
            socket.emit('ping', () => {
              const latency = Date.now() - socket.lastPingTime;
              this.metrics.heartbeatsSent++;
              this.metrics.heartbeatsReceived++;
              this.metrics.messageLatencies.push(latency);
            });
          }
        });

        const elapsed = Date.now() - heartbeatTestStart;
        const heartbeatsDone = Math.floor(elapsed / heartbeatInterval);
        
        console.log(`  Heartbeat round ${heartbeatsDone}/${heartbeatsExpected} - Sent: ${this.sockets.length}`);
        
        if (elapsed >= heartbeatDuration) {
          clearInterval(this.heartbeatInterval);
          
          const duration = Date.now() - heartbeatTestStart;
          const heartbeatRate = this.metrics.heartbeatsReceived / (this.metrics.heartbeatsSent || 1) * 100;
          
          console.log(`\nHeartbeat Results:`);
          console.log(`  Expected heartbeats: ${this.metrics.heartbeatsSent}`);
          console.log(`  Received heartbeats: ${this.metrics.heartbeatsReceived}`);
          console.log(`  Heartbeat success rate: ${heartbeatRate.toFixed(2)}%`);
          console.log(`  Test duration: ${duration}ms`);
          console.log(`  Avg heartbeat latency: ${this.calculateAverage(this.metrics.messageLatencies)}ms`);
          
          resolve({
            expected: this.metrics.heartbeatsSent,
            received: this.metrics.heartbeatsReceived,
            successRate: heartbeatRate,
            duration,
            avgLatency: this.calculateAverage(this.metrics.messageLatencies)
          });
        }
      }, heartbeatInterval);
    });
  }

  async testLatencyUnderLoad() {
    console.log(`\n--- Latency Under Load Test ---`);
    console.log(`Measuring latency while under message load...`);
    
    const testDuration = 10000;
    const messageInterval = 100;
    const latencyMeasurements = [];
    
    const startTime = Date.now();
    let messageCount = 0;
    
    const sendMessages = async () => {
      while (Date.now() - startTime < testDuration) {
        const sendStart = Date.now();
        
        this.sockets.forEach(socket => {
          if (socket.connected) {
            socket.emit('message', {
              type: 'latency-test',
              timestamp: sendStart,
              size: TEST_CONFIG.messageSize
            });
            messageCount++;
          }
        });
        
        await new Promise(resolve => setTimeout(resolve, messageInterval));
      }
    };

    const receiveAcks = () => {
      this.sockets.forEach(socket => {
        socket.on('message', (data) => {
          if (data.type === 'latency-test') {
            const latency = Date.now() - data.timestamp;
            latencyMeasurements.push(latency);
          }
        });
      });
    };

    receiveAcks();
    await sendMessages();
    
    const sortedLatencies = [...latencyMeasurements].sort((a, b) => a - b);
    const p50 = sortedLatencies[Math.floor(sortedLatencies.length * 0.5)] || 0;
    const p95 = sortedLatencies[Math.floor(sortedLatencies.length * 0.95)] || 0;
    const p99 = sortedLatencies[Math.floor(sortedLatencies.length * 0.99)] || 0;
    const avg = this.calculateAverage(latencyMeasurements);
    
    console.log(`\nLatency Results:`);
    console.log(`  Messages sent: ${messageCount}`);
    console.log(`  Latency samples: ${latencyMeasurements.length}`);
    console.log(`  Average latency: ${avg.toFixed(2)}ms`);
    console.log(`  P50 latency: ${p50.toFixed(2)}ms`);
    console.log(`  P95 latency: ${p95.toFixed(2)}ms`);
    console.log(`  P99 latency: ${p99.toFixed(2)}ms`);
    
    return {
      samples: latencyMeasurements.length,
      avg,
      p50,
      p95,
      p99
    };
  }

  calculateAverage(arr) {
    if (arr.length === 0) return 0;
    return arr.reduce((sum, val) => sum + val, 0) / arr.length;
  }

  async cleanup() {
    console.log('\n--- Cleanup ---');
    console.log('Disconnecting all clients...');
    
    const disconnects = this.sockets.map(socket => {
      return new Promise(resolve => {
        if (socket.connected) {
          socket.disconnect();
        }
        resolve();
      });
    });

    await Promise.all(disconnects);
    this.sockets = [];
    
    console.log('All clients disconnected');
  }

  generateReport() {
    console.log('\n========== WebSocket Load Test Report ==========\n');
    
    const totalDuration = this.metrics.endTime - this.metrics.startTime;
    const activeConnections = this.metrics.connectionSuccess - this.metrics.disconnections;
    
    console.log('Connection Metrics:');
    console.log(`  Total connections attempted: ${this.metrics.connectionSuccess + this.metrics.connectionFailure}`);
    console.log(`  Successful connections: ${this.metrics.connectionSuccess}`);
    console.log(`  Failed connections: ${this.metrics.connectionFailure}`);
    console.log(`  Active connections at end: ${activeConnections}`);
    console.log(`  Disconnections: ${this.metrics.disconnections}`);
    console.log(`  Avg connection time: ${this.calculateAverage(this.metrics.connectionTimes).toFixed(2)}ms`);
    
    console.log('\nMessage Metrics:');
    console.log(`  Messages sent: ${this.metrics.messagesSent}`);
    console.log(`  Messages received: ${this.metrics.messagesReceived}`);
    console.log(`  Broadcasts received: ${this.metrics.broadcastsReceived}`);
    
    console.log('\nHeartbeat Metrics:');
    console.log(`  Heartbeats sent: ${this.metrics.heartbeatsSent}`);
    console.log(`  Heartbeats received: ${this.metrics.heartbeatsReceived}`);
    const heartbeatRate = this.metrics.heartbeatsSent > 0 
      ? (this.metrics.heartbeatsReceived / this.metrics.heartbeatsSent * 100).toFixed(2) 
      : '0.00';
    console.log(`  Heartbeat success rate: ${heartbeatRate}%`);
    
    console.log('\nLatency Metrics:');
    if (this.metrics.messageLatencies.length > 0) {
      const sorted = [...this.metrics.messageLatencies].sort((a, b) => a - b);
      console.log(`  Samples collected: ${this.metrics.messageLatencies.length}`);
      console.log(`  Average: ${this.calculateAverage(this.metrics.messageLatencies).toFixed(2)}ms`);
      console.log(`  P50: ${sorted[Math.floor(sorted.length * 0.5)].toFixed(2)}ms`);
      console.log(`  P95: ${sorted[Math.floor(sorted.length * 0.95)].toFixed(2)}ms`);
      console.log(`  P99: ${sorted[Math.floor(sorted.length * 0.99)].toFixed(2)}ms`);
    } else {
      console.log('  No latency data collected');
    }
    
    console.log('\nError Metrics:');
    console.log(`  Total errors: ${this.metrics.errors}`);
    
    console.log('\nPerformance Summary:');
    console.log(`  Test duration: ${(totalDuration / 1000).toFixed(2)}s`);
    console.log(`  Messages per second: ${((this.metrics.messagesSent + this.metrics.messagesReceived) / (totalDuration / 1000)).toFixed(2)}`);
    console.log(`  Avg connection rate: ${(this.metrics.connectionSuccess / (totalDuration / 1000)).toFixed(2)} conn/s`);
    
    const successRate = ((this.metrics.connectionSuccess / (this.metrics.connectionSuccess + this.metrics.connectionFailure)) * 100).toFixed(2);
    console.log(`  Overall success rate: ${successRate}%`);
    
    console.log('\n' + '='.repeat(50));
    
    const passed = this.metrics.connectionSuccess >= TEST_CONFIG.concurrentConnections * 0.9 &&
                   this.metrics.errors < this.metrics.connectionSuccess * 0.1;
    
    console.log(`\nTest Result: ${passed ? 'PASSED ✓' : 'FAILED ✗'}`);
    
    return {
      passed,
      metrics: this.metrics,
      config: TEST_CONFIG
    };
  }

  async run() {
    console.log('Starting WebSocket Load Test...');
    console.log('Configuration:', TEST_CONFIG);
    
    this.isRunning = true;
    this.metrics.startTime = Date.now();
    
    try {
      await this.connectClients(TEST_CONFIG.concurrentConnections);
      
      if (this.metrics.connectionSuccess > 0) {
        await this.testMessageBroadcast();
        
        await this.testHeartbeatStability();
        
        await this.testLatencyUnderLoad();
      } else {
        console.error('No connections established. Skipping other tests.');
      }
      
    } catch (error) {
      console.error('Test error:', error);
      this.metrics.errors++;
    } finally {
      this.metrics.endTime = Date.now();
      const result = this.generateReport();
      await this.cleanup();
      
      this.isRunning = false;
      
      process.exit(result.passed ? 0 : 1);
    }
  }
}

if (require.main === module) {
  const test = new WebSocketLoadTest();
  test.run().catch(error => {
    console.error('Fatal error:', error);
    process.exit(1);
  });
}

module.exports = WebSocketLoadTest;
