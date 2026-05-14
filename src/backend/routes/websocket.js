const express = require('express');
const router = express.Router();
const websocketMonitor = require('../services/websocketMonitor');
const heartbeatManager = require('../services/heartbeatManager');
const websocketService = require('../services/websocketService');

router.get('/metrics', (req, res) => {
  try {
    const metrics = websocketMonitor.getMetrics();
    res.json({
      success: true,
      data: metrics,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/metrics/connections', (req, res) => {
  try {
    const connections = websocketMonitor.getActiveConnections();
    res.json({
      success: true,
      data: connections,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/metrics/performance', (req, res) => {
  try {
    const performance = websocketMonitor.getPerformanceMetrics();
    res.json({
      success: true,
      data: performance,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/metrics/rooms', (req, res) => {
  try {
    const rooms = websocketMonitor.getRoomMetrics();
    res.json({
      success: true,
      data: rooms,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/metrics/errors', (req, res) => {
  try {
    const count = parseInt(req.query.count) || 10;
    const errors = websocketMonitor.getRecentErrors(count);
    res.json({
      success: true,
      data: {
        errors,
        total: websocketMonitor.metrics.errors.total,
        errorRate: websocketMonitor.calculateErrorRate()
      },
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/metrics/alerts', (req, res) => {
  try {
    const count = parseInt(req.query.count) || 10;
    const alerts = websocketMonitor.getRecentAlerts(count);
    res.json({
      success: true,
      data: alerts,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/health', (req, res) => {
  try {
    const healthStatus = websocketMonitor.getHealthStatus();
    const heartbeatStats = heartbeatManager.getStats();
    const connectionStats = websocketService.getConnectionStats();

    res.json({
      success: true,
      data: {
        status: healthStatus.status,
        issues: healthStatus.issues,
        uptime: healthStatus.uptime,
        activeConnections: healthStatus.activeConnections,
        errorRate: healthStatus.errorRate,
        heartbeat: {
          totalSockets: heartbeatStats.totalSockets,
          averageResponseTime: heartbeatStats.averageResponseTime,
          missedPongs: heartbeatStats.missedPongs,
          socketHealth: heartbeatStats.socketHealth
        },
        connections: connectionStats
      },
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/heartbeat', (req, res) => {
  try {
    const report = heartbeatManager.getHealthReport();
    res.json({
      success: true,
      data: report,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/sockets', (req, res) => {
  try {
    const sockets = heartbeatManager.getAllSockets();
    res.json({
      success: true,
      data: {
        total: sockets.length,
        sockets
      },
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/history', (req, res) => {
  try {
    const count = parseInt(req.query.count) || 20;
    const history = websocketMonitor.getConnectionHistory(count);
    res.json({
      success: true,
      data: history,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.post('/thresholds', (req, res) => {
  try {
    const { name, value } = req.body;

    if (!name || value === undefined) {
      return res.status(400).json({
        success: false,
        error: 'Name and value are required'
      });
    }

    websocketMonitor.setAlertThreshold(name, value);

    res.json({
      success: true,
      message: `Threshold ${name} updated to ${value}`,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.post('/reset', (req, res) => {
  try {
    websocketMonitor.reset();
    heartbeatManager.resetStats();

    res.json({
      success: true,
      message: 'Metrics reset successfully',
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/export', (req, res) => {
  try {
    const metrics = websocketMonitor.exportMetrics();
    res.json({
      success: true,
      data: metrics
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.post('/monitoring/start', (req, res) => {
  try {
    const interval = parseInt(req.body.interval) || 5000;
    const monitorId = websocketMonitor.startMonitoring(interval);

    res.json({
      success: true,
      message: 'Monitoring started',
      monitorId,
      interval,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.post('/monitoring/stop', (req, res) => {
  try {
    const { monitorId } = req.body;

    if (monitorId) {
      websocketMonitor.stopMonitoring(monitorId);
      res.json({
        success: true,
        message: `Monitor ${monitorId} stopped`,
        timestamp: new Date().toISOString()
      });
    } else {
      websocketMonitor.stopAllMonitoring();
      res.json({
        success: true,
        message: 'All monitors stopped',
        timestamp: new Date().toISOString()
      });
    }
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

module.exports = router;
