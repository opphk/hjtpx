#!/usr/bin/env node

const WebSocket = require('ws');
const http = require('http');

const TEST_CONFIG = {
  host: process.env.WS_HOST || 'localhost',
  port: parseInt(process.env.WS_PORT || '3000'),
  protocol: process.env.WS_PROTOCOL || 'ws',
  testDuration: parseInt(process.env.TEST_DURATION || '60000'),
  rampUpTime: parseInt(process.env.RAMP_UP_TIME || '30000'),
  maxConnections: parseInt(process.env.MAX_CONNECTIONS || '1000'),
  messageSize: parseInt(process.env.MESSAGE_SIZE || '1024'),
  heartbeatInterval: parseInt(process.env.HEARTBEAT_INTERVAL || '30000'),
};

let connections = [];
let messageCount = 0;
let errorCount = 0;
let startTime;
let statsInterval;

const results = {
  connections: {
    attempted: 0,
    successful: 0,
    failed: 0,
    active: 0,
    peak: 0,
  },
  messages: {
    sent: 0,
    received: 0,
    broadcast: 0,
    failed: 0,
  },
  performance: {
    avgLatency: 0,
    minLatency: Infinity,
    maxLatency: 0,
    totalLatency: 0,
    latencySamples: 0,
  },
  errors: [],
};

function generateRandomMessage(size) {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < size; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

function createConnection(index) {
  return new Promise((resolve, reject) => {
    const url = `${TEST_CONFIG.protocol}://${TEST_CONFIG.host}:${TEST_CONFIG.port}`;
    const ws = new WebSocket(url, {
      handshakeTimeout: 10000,
    });

    const connectionStartTime = Date.now();

    ws.on('open', () => {
      const connectionTime = Date.now() - connectionStartTime;
      results.connections.attempted++;
      results.connections.successful++;
      results.connections.active++;
      results.connections.peak = Math.max(results.connections.peak, results.connections.active);

      connections.push({
        id: index,
        socket: ws,
        connectedAt: Date.now(),
        connectionTime,
        messagesSent: 0,
        messagesReceived: 0,
      });

      ws.send(JSON.stringify({
        type: 'auth',
        userId: `test-user-${index}`,
      }));

      setTimeout(() => {
        ws.send(JSON.stringify({
          type: 'subscribe',
          room: 'test-room',
        }));
      }, 100);

      resolve(ws);
    });

    ws.on('message', (data) => {
      const connection = connections.find(c => c.id === index);
      if (connection) {
        connection.messagesReceived++;
        results.messages.received++;

        try {
          const parsed = JSON.parse(data);
          if (parsed.type === 'broadcast' && parsed.timestamp) {
            const latency = Date.now() - parsed.timestamp;
            results.performance.latencySamples++;
            results.performance.totalLatency += latency;
            results.performance.avgLatency = results.performance.totalLatency / results.performance.latencySamples;
            results.performance.minLatency = Math.min(results.performance.minLatency, latency);
            results.performance.maxLatency = Math.max(results.performance.maxLatency, latency);
          }
        } catch (e) {
        }
      }
    });

    ws.on('error', (error) => {
      results.connections.failed++;
      results.errors.push({
        time: Date.now() - startTime,
        type: 'connection',
        message: error.message,
      });
      reject(error);
    });

    ws.on('close', () => {
      results.connections.active--;
    });

    setTimeout(() => {
      if (ws.readyState === WebSocket.CONNECTING) {
        ws.terminate();
        reject(new Error('Connection timeout'));
      }
    }, 10000);
  });
}

async function createConnectionsGradually() {
  const connectionsPerSecond = TEST_CONFIG.maxConnections / (TEST_CONFIG.rampUpTime / 1000);
  const batchSize = Math.max(1, Math.floor(connectionsPerSecond / 10));
  const batchInterval = 100;

  console.log(`Starting connection ramp-up: ${TEST_CONFIG.maxConnections} connections over ${TEST_CONFIG.rampUpTime}ms`);

  for (let i = 0; i < TEST_CONFIG.maxConnections && (Date.now() - startTime) < TEST_CONFIG.testDuration; i += batchSize) {
    const batch = [];
    for (let j = 0; j < batchSize && (i + j) < TEST_CONFIG.maxConnections; j++) {
      batch.push(
        createConnection(i + j)
          .catch(err => {
            console.error(`Connection ${i + j} failed:`, err.message);
          })
      );
    }

    await Promise.all(batch);

    if ((i + batchSize) % 100 === 0) {
      console.log(`Created ${Math.min(i + batchSize, TEST_CONFIG.maxConnections)} connections...`);
    }

    await new Promise(resolve => setTimeout(resolve, batchInterval));
  }

  console.log(`Connection ramp-up completed. Active connections: ${results.connections.active}`);
}

async function broadcastMessages() {
  const message = JSON.stringify({
    type: 'broadcast',
    room: 'test-room',
    content: generateRandomMessage(TEST_CONFIG.messageSize),
    timestamp: Date.now(),
    testId: 'load-test',
  });

  while ((Date.now() - startTime) < TEST_CONFIG.testDuration && connections.length > 0) {
    const activeConnections = connections.filter(c => c.socket.readyState === WebSocket.OPEN);

    if (activeConnections.length === 0) {
      await new Promise(resolve => setTimeout(resolve, 1000));
      continue;
    }

    const broadcastStart = Date.now();
    activeConnections.forEach(conn => {
      if (conn.socket.readyState === WebSocket.OPEN) {
        try {
          conn.socket.send(message);
          conn.messagesSent++;
          results.messages.sent++;
        } catch (e) {
          results.messages.failed++;
        }
      }
    });

    const broadcastTime = Date.now() - broadcastStart;

    if (broadcastTime > 100) {
      console.warn(`Slow broadcast detected: ${broadcastTime}ms for ${activeConnections.length} clients`);
    }

    await new Promise(resolve => setTimeout(resolve, 100));
  }
}

function sendHeartbeat() {
  connections.forEach(conn => {
    if (conn.socket.readyState === WebSocket.OPEN) {
      try {
        conn.socket.send(JSON.stringify({
          type: 'ping',
          timestamp: Date.now(),
        }));
      } catch (e) {
      }
    }
  });
}

function printStats() {
  const duration = (Date.now() - startTime) / 1000;
  const activeConnections = connections.filter(c => c.socket.readyState === WebSocket.OPEN).length;

  console.log('\n========================================');
  console.log('WebSocket Load Test - Real-time Stats');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(1)}s`);
  console.log(`Active Connections: ${activeConnections} / ${TEST_CONFIG.maxConnections}`);
  console.log(`Connection Success Rate: ${(results.connections.successful / results.connections.attempted * 100).toFixed(2)}%`);
  console.log(`Messages Sent: ${results.messages.sent}`);
  console.log(`Messages Received: ${results.messages.received}`);
  console.log(`Message Throughput: ${(results.messages.sent / duration).toFixed(2)} msg/s`);
  if (results.performance.latencySamples > 0) {
    console.log(`Avg Latency: ${results.performance.avgLatency.toFixed(2)}ms`);
    console.log(`Min Latency: ${results.performance.minLatency.toFixed(2)}ms`);
    console.log(`Max Latency: ${results.performance.maxLatency.toFixed(2)}ms`);
  }
  console.log('========================================\n');
}

function printFinalResults() {
  const duration = (Date.now() - startTime) / 1000;

  console.log('\n');
  console.log('╔══════════════════════════════════════════════╗');
  console.log('║      WebSocket Load Test - Final Report       ║');
  console.log('╚══════════════════════════════════════════════╝');
  console.log('');

  console.log('📊 Connection Statistics:');
  console.log(`   • Total Attempted: ${results.connections.attempted}`);
  console.log(`   • Successful: ${results.connections.successful}`);
  console.log(`   • Failed: ${results.connections.failed}`);
  console.log(`   • Peak Concurrent: ${results.connections.peak}`);
  console.log(`   • Success Rate: ${(results.connections.successful / results.connections.attempted * 100).toFixed(2)}%`);

  console.log('');
  console.log('📨 Message Statistics:');
  console.log(`   • Total Sent: ${results.messages.sent}`);
  console.log(`   • Total Received: ${results.messages.received}`);
  console.log(`   • Failed: ${results.messages.failed}`);
  console.log(`   • Throughput: ${(results.messages.sent / duration).toFixed(2)} msg/s`);

  console.log('');
  console.log('⏱️  Performance Metrics:');
  if (results.performance.latencySamples > 0) {
    console.log(`   • Average Latency: ${results.performance.avgLatency.toFixed(2)}ms`);
    console.log(`   • Min Latency: ${results.performance.minLatency.toFixed(2)}ms`);
    console.log(`   • Max Latency: ${results.performance.maxLatency.toFixed(2)}ms`);
  }
  console.log(`   • Test Duration: ${duration.toFixed(2)}s`);

  console.log('');
  console.log('⚠️  Errors:');
  if (results.errors.length === 0) {
    console.log('   • No errors detected');
  } else {
    console.log(`   • Total Errors: ${results.errors.length}`);
    const errorTypes = {};
    results.errors.forEach(e => {
      errorTypes[e.type] = (errorTypes[e.type] || 0) + 1;
    });
    Object.entries(errorTypes).forEach(([type, count]) => {
      console.log(`     - ${type}: ${count}`);
    });
  }

  console.log('');
  console.log('============================================\n');

  const report = {
    testConfig: TEST_CONFIG,
    timestamp: new Date().toISOString(),
    duration,
    results,
    summary: {
      status: results.errors.length === 0 ? 'PASSED' : 'PASSED_WITH_WARNINGS',
      connectionsPerSecond: (results.connections.successful / duration).toFixed(2),
      messagesPerSecond: (results.messages.sent / duration).toFixed(2),
    },
  };

  const fs = require('fs');
  const reportPath = `/workspace/hjtpx/test-results/websocket-load-test-${Date.now()}.json`;
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
  console.log(`📄 Detailed report saved to: ${reportPath}`);
}

async function runLoadTest() {
  console.log('Starting WebSocket Load Test...');
  console.log(`Target: ${TEST_CONFIG.protocol}://${TEST_CONFIG.host}:${TEST_CONFIG.port}`);
  console.log(`Duration: ${TEST_CONFIG.testDuration / 1000}s`);
  console.log(`Max Connections: ${TEST_CONFIG.maxConnections}`);
  console.log(`Ramp-up Time: ${TEST_CONFIG.rampUpTime / 1000}s`);

  startTime = Date.now();

  statsInterval = setInterval(printStats, 10000);

  const heartbeatInterval = setInterval(sendHeartbeat, TEST_CONFIG.heartbeatInterval);

  try {
    await createConnectionsGradually();

    await broadcastMessages();

  } catch (error) {
    console.error('Test error:', error);
  } finally {
    clearInterval(statsInterval);
    clearInterval(heartbeatInterval);

    console.log('\nCleaning up connections...');
    connections.forEach(conn => {
      try {
        conn.socket.close();
      } catch (e) {
      }
    });

    printFinalResults();

    process.exit(results.errors.length > 0 ? 1 : 0);
  }
}

if (require.main === module) {
  runLoadTest().catch(console.error);
}

module.exports = {
  runLoadTest,
  TEST_CONFIG,
};
