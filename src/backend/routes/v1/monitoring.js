const express = require('express');
const router = express.Router();

const { healthMonitor } = require('../services/healthMonitor');
const { alertManager } = require('../services/alertService');
const { getMetrics, getContentType } = require('../services/metricsService');

router.get('/health', (req, res) => {
  const health = healthMonitor.getHealthStatus();

  const statusCode = health.healthy ? 200 : 503;

  res.status(statusCode).json({
    success: health.healthy,
    data: health
  });
});

router.get('/health/detailed', (req, res) => {
  const detailed = healthMonitor.getDetailedHealth();

  const statusCode = detailed.healthy ? 200 : 503;

  res.status(statusCode).json({
    success: detailed.healthy,
    data: detailed
  });
});

router.get('/health/live', (req, res) => {
  res.status(200).json({
    success: true,
    data: {
      alive: true,
      timestamp: new Date().toISOString()
    }
  });
});

router.get('/health/ready', async (req, res) => {
  const checks = await healthMonitor.runAllChecks();
  const allHealthy = checks.every(c => c.healthy);

  res.status(allHealthy ? 200 : 503).json({
    success: allHealthy,
    data: {
      ready: allHealthy,
      checks: checks.map(c => ({
        name: c.name,
        healthy: c.healthy,
        message: c.message
      })),
      timestamp: new Date().toISOString()
    }
  });
});

router.get('/metrics', async (req, res) => {
  try {
    const metrics = await getMetrics();
    res.set('Content-Type', getContentType());
    res.send(metrics);
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Failed to collect metrics'
    });
  }
});

router.get('/metrics/prometheus', async (req, res) => {
  try {
    const metrics = await getMetrics();
    res.set('Content-Type', getContentType());
    res.send(metrics);
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Failed to collect metrics'
    });
  }
});

router.get('/alerts', (req, res) => {
  const activeAlerts = alertManager.getActiveAlerts();

  res.json({
    success: true,
    data: {
      activeAlerts,
      total: activeAlerts.length,
      thresholds: alertManager.thresholds
    }
  });
});

router.get('/alerts/:type', (req, res) => {
  const alerts = alertManager.getAlertsByType(req.params.type);

  res.json({
    success: true,
    data: {
      alerts,
      type: req.params.type,
      total: alerts.length
    }
  });
});

router.post('/alerts/:id/acknowledge', (req, res) => {
  const alert = alertManager.acknowledgeAlert(req.params.id);

  if (!alert) {
    return res.status(404).json({
      success: false,
      error: 'Alert not found'
    });
  }

  res.json({
    success: true,
    data: alert
  });
});

router.post('/alerts/:id/resolve', (req, res) => {
  const alert = alertManager.resolveAlert(req.params.id);

  if (!alert) {
    return res.status(404).json({
      success: false,
      error: 'Alert not found'
    });
  }

  res.json({
    success: true,
    data: alert
  });
});

router.get('/status', (req, res) => {
  const metrics = healthMonitor.getMetrics();

  res.json({
    success: true,
    data: metrics
  });
});

module.exports = router;
