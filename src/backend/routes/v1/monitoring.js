const express = require('express');
const router = express.Router();

const websocketService = require('../../services/websocketService');
const metricsService = require('../../services/metricsService');

router.get('/websocket', (req, res) => {
  try {
    const metrics = websocketService.getDetailedMetrics();
    res.json({
      success: true,
      data: metrics
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Failed to get WebSocket metrics'
    });
  }
});

router.get('/websocket/stats', (req, res) => {
  try {
    const stats = websocketService.getConnectionStats();
    res.json({
      success: true,
      data: stats
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Failed to get WebSocket stats'
    });
  }
});

router.get('/websocket/online-users', (req, res) => {
  try {
    const users = websocketService.getOnlineUsers();
    res.json({
      success: true,
      data: {
        users,
        count: users.length
      }
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Failed to get online users'
    });
  }
});

router.get('/websocket/health', (req, res) => {
  try {
    const metrics = websocketService.getDetailedMetrics();
    const uptime = metrics.uptime;
    const errorRate = metrics.totalConnections > 0 
      ? (metrics.errors / metrics.totalConnections) * 100 
      : 0;
    
    const isHealthy = errorRate < 5 && uptime > 0;
    
    res.json({
      success: true,
      data: {
        status: isHealthy ? 'healthy' : 'degraded',
        uptime,
        currentConnections: metrics.currentConnections,
        errorRate: errorRate.toFixed(2),
        heartbeatStatus: metrics.heartbeatMetrics
      }
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Failed to get WebSocket health status'
    });
  }
});

router.get('/metrics', async (req, res) => {
  try {
    const metrics = await metricsService.getMetrics();
    res.set('Content-Type', metricsService.getContentType());
    res.send(metrics);
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Failed to get metrics'
    });
  }
});

router.get('/websocket/performance', (req, res) => {
  try {
    const metrics = websocketService.getDetailedMetrics();
    
    const avgMessageLatency = metrics.messagesReceived > 0 
      ? (metrics.messagesSent / metrics.messagesReceived) * 100 
      : 0;
    
    const messagesPerSecond = metrics.uptime > 0 
      ? (metrics.messagesSent / (metrics.uptime / 1000)).toFixed(2) 
      : 0;
    
    const connectionsPerSecond = metrics.uptime > 0 
      ? (metrics.totalConnections / (metrics.uptime / 1000)).toFixed(2) 
      : 0;
    
    res.json({
      success: true,
      data: {
        throughput: {
          messagesPerSecond: parseFloat(messagesPerSecond),
          connectionsPerSecond: parseFloat(connectionsPerSecond)
        },
        latency: {
          avgMessageLatency: avgMessageLatency.toFixed(2)
        },
        reliability: {
          totalMessages: metrics.messagesSent,
          totalReceived: metrics.messagesReceived,
          messageDeliveryRate: metrics.messagesSent > 0 
            ? ((metrics.messagesReceived / metrics.messagesSent) * 100).toFixed(2) 
            : '100.00',
          errorCount: metrics.errors
        },
        heartbeat: metrics.heartbeatMetrics
      }
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Failed to get WebSocket performance metrics'
    });
  }
});

module.exports = router;
