const express = require('express');
const router = express.Router();
const dbPoolManager = require('../../config/database/dbPoolManager');

router.get('/pool/stats', async (req, res) => {
  try {
    const stats = dbPoolManager.getPoolStats();
    res.json({
      success: true,
      data: stats
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/pool/query-stats', async (req, res) => {
  try {
    const stats = dbPoolManager.getQueryStats();
    res.json({
      success: true,
      data: stats
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/pool/metrics', async (req, res) => {
  try {
    const metrics = dbPoolManager.getMetrics();
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

router.get('/pool/health', async (req, res) => {
  try {
    const health = await dbPoolManager.healthCheck();
    res.json({
      success: true,
      data: health
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/pool/leaks', async (req, res) => {
  try {
    const leaks = dbPoolManager.getConnectionLeaks();
    res.json({
      success: true,
      data: {
        count: leaks.length,
        leaks: leaks
      }
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.post('/pool/leaks/clear', async (req, res) => {
  try {
    dbPoolManager.clearLeakHistory();
    res.json({
      success: true,
      message: 'Connection leak history cleared'
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.post('/pool/stats/reset', async (req, res) => {
  try {
    dbPoolManager.resetStats();
    res.json({
      success: true,
      message: 'Statistics reset successfully'
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.post('/pool/refresh', async (req, res) => {
  try {
    await dbPoolManager.refreshPool();
    res.json({
      success: true,
      message: 'Pool refreshed successfully'
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.put('/pool/config', async (req, res) => {
  try {
    const { max, min, idleTimeoutMillis, connectionTimeoutMillis } = req.body;
    
    const newConfig = {};
    if (max !== undefined) newConfig.max = max;
    if (min !== undefined) newConfig.min = min;
    if (idleTimeoutMillis !== undefined) newConfig.idleTimeoutMillis = idleTimeoutMillis;
    if (connectionTimeoutMillis !== undefined) newConfig.connectionTimeoutMillis = connectionTimeoutMillis;
    
    if (Object.keys(newConfig).length === 0) {
      return res.status(400).json({
        success: false,
        error: 'No valid configuration provided'
      });
    }
    
    dbPoolManager.setConfig(newConfig);
    res.json({
      success: true,
      message: 'Configuration updated successfully',
      config: dbPoolManager.config
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

module.exports = router;
