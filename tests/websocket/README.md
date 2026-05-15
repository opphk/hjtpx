# HJTPX WebSocket Stress Testing

## Overview

This directory contains scripts and tools for testing the HJTPX WebSocket service performance under load.

## File Description

- `package.json` - Test script dependencies configuration
- `test-connections.js` - Concurrent connection stress test script
- `test-broadcast.js` - Message broadcast performance test script
- `test-monitor.js` - Monitoring test script

## Installation

```bash
cd tests/websocket
npm install
```

## Usage

### 1. Start the Main Server

Run in the project root directory:

```bash
npm start
```

### 2. Concurrent Connection Stress Test

```bash
cd tests/websocket
node test-connections.js
```

This test will:
- Establish a large number of concurrent connections
- Measure connection success rate and connection time
- Keep connections for a period before disconnecting
- Output detailed test results including latency percentiles (P50, P95, P99)

**Configuration Options** (modify in `test-connections.js`):
- `TOTAL_CONNECTIONS` - Total number of connections (default: 100)
- `CONNECTIONS_PER_BATCH` - Connections per batch (default: 20)
- `BATCH_DELAY` - Batch interval (ms, default: 500)
- `STAY_CONNECTED_TIME` - Connection duration (ms, default: 30000)
- `USE_NATIVE_WS` - Use native WebSocket library (default: false)

### 3. Broadcast Performance Test

```bash
cd tests/websocket
node test-broadcast.js
```

This test will:
- Create multiple connections
- Send multiple broadcast messages
- Measure message delivery success rate and throughput
- Calculate latency metrics (avg, min, max, P50, P95, P99)

**Configuration Options** (modify in `test-broadcast.js`):
- `TOTAL_CLIENTS` - Number of clients (default: 50)
- `MESSAGES_PER_TEST` - Number of messages to send (default: 100)
- `MESSAGE_INTERVAL` - Message interval (ms, default: 100)
- `MESSAGE_SIZE` - Message size in bytes (default: 1024)
- `ENABLE_DELAY_MEASUREMENT` - Enable latency measurement (default: true)

### 4. Monitoring Test

```bash
cd tests/websocket
node test-monitor.js
```

This test will:
- Connect to the server
- Monitor memory usage and network statistics
- Request server metrics periodically
- Output comprehensive monitoring statistics

**Configuration Options** (modify in `test-monitor.js`):
- `TEST_DURATION` - Test duration in ms (default: 60000)
- `UPDATE_INTERVAL` - Statistics update interval (ms, default: 5000)
- `METRICS_INTERVAL` - Server metrics request interval (ms, default: 10000)

## API Monitoring Endpoints

### Get WebSocket Detailed Metrics

```
GET /api/v1/monitoring/websocket
```

Response Example:
```json
{
  "success": true,
  "data": {
    "uptime": 3600000,
    "currentConnections": 10,
    "onlineUsers": 5,
    "totalConnections": 100,
    "totalDisconnections": 90,
    "messagesSent": 500,
    "messagesReceived": 480,
    "errors": 5,
    "avgConnectionTime": 120000,
    "rooms": ["room1", "room2"],
    "subscriptions": [
      {"channel": "channel1", "subscriberCount": 3}
    ],
    "heartbeatMetrics": {
      "heartbeatsSent": 1000,
      "heartbeatsReceived": 995,
      "missedHeartbeats": 5,
      "activeHeartbeats": 10,
      "config": {
        "pingTimeout": 30000,
        "pingInterval": 15000,
        "heartbeatCheckInterval": 5000,
        "maxMissedHeartbeats": 3
      }
    }
  }
}
```

### Get WebSocket Health Status

```
GET /api/v1/monitoring/websocket/health
```

Response Example:
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "uptime": 3600000,
    "currentConnections": 10,
    "errorRate": "2.50",
    "heartbeatStatus": {
      "heartbeatsSent": 1000,
      "heartbeatsReceived": 995,
      "missedHeartbeats": 5,
      "activeHeartbeats": 10,
      "config": {...}
    }
  }
}
```

### Get WebSocket Performance Metrics

```
GET /api/v1/monitoring/websocket/performance
```

Response Example:
```json
{
  "success": true,
  "data": {
    "throughput": {
      "messagesPerSecond": 50.5,
      "connectionsPerSecond": 2.3
    },
    "latency": {
      "avgMessageLatency": "1.25"
    },
    "reliability": {
      "totalMessages": 500,
      "totalReceived": 480,
      "messageDeliveryRate": "96.00",
      "errorCount": 5
    },
    "heartbeat": {...}
  }
}
```

### Get Basic Statistics

```
GET /api/v1/monitoring/websocket/stats
```

### Get Online Users

```
GET /api/v1/monitoring/websocket/online-users
```

### Get Prometheus Metrics

```
GET /api/v1/monitoring/metrics
```

## WebSocket Events

### Get Metrics (Client-side)

```javascript
socket.emit('get:metrics', (response) => {
  console.log(response.metrics);
});
```

## Performance Optimization

### Heartbeat Mechanism Optimization
- pingTimeout: 30 seconds (optimized from 60s)
- pingInterval: 15 seconds (optimized from 25s)
- Configurable via environment variables:
  - `WS_PING_TIMEOUT`
  - `WS_PING_INTERVAL`
  - `WS_HEARTBEAT_CHECK_INTERVAL`
  - `WS_MAX_MISSED_HEARTBEATS`
- Faster connection state detection

### Message Compression
- Enabled perMessageDeflate compression
- Threshold set to 1KB
- Reduces network bandwidth consumption

### Buffer Size
- maxHttpBufferSize: 10MB
- Supports larger message transmission

## Monitoring Capabilities

### Real-time Metrics
- Connection count and status
- Message throughput (sent/received)
- Error tracking
- Heartbeat monitoring
- Memory usage
- Network statistics

### Performance Indicators
- Latency percentiles (P50, P95, P99)
- Message delivery rate
- Connection success rate
- System resource utilization

### Health Checks
- Automatic health status evaluation
- Error rate threshold monitoring
- Heartbeat health validation

## Test Scripts

### Run All Tests

```bash
cd tests/websocket
npm run test:all
```

### Run Stress Test Suite

```bash
cd tests/websocket
npm run test:stress
```

This will execute:
1. Connection stress test
2. Broadcast performance test
3. Monitoring test

## Environment Variables

The WebSocket server supports the following environment variables:

```bash
WS_PING_TIMEOUT=30000
WS_PING_INTERVAL=15000
WS_HEARTBEAT_CHECK_INTERVAL=5000
WS_MAX_MISSED_HEARTBEATS=3
```

## Performance Best Practices

1. **Connection Pooling**: Use batch connections to avoid overwhelming the server
2. **Message Batching**: Send messages in batches for better throughput
3. **Heartbeat Tuning**: Adjust heartbeat intervals based on your network conditions
4. **Compression**: Enable compression for large messages to reduce bandwidth
5. **Monitoring**: Regular monitoring helps identify bottlenecks early
