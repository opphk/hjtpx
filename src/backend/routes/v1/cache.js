const express = require('express');
const router = express.Router();
const cacheService = require('../../services/cacheService');

router.get('/cache/stats', async (req, res) => {
  try {
    const stats = cacheService.getStats();
    
    res.json({
      success: true,
      data: {
        metrics: stats,
        summary: {
          totalHits: stats.hits.total,
          totalMisses: stats.misses.total,
          overallHitRate: stats.hitRate.overall,
          totalSets: stats.sets.total,
          totalDeletes: stats.deletes.total,
          totalErrors: stats.errors.total,
          latency: {
            average: stats.latency.avg,
            p50: stats.latency.p50,
            p95: stats.latency.p95,
            p99: stats.latency.p99
          },
          uptime: stats.uptimeFormatted
        }
      }
    });
  } catch (error) {
    console.error('Cache stats error:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to retrieve cache statistics'
    });
  }
});

router.get('/cache/health', async (req, res) => {
  try {
    const isHealthy = await cacheService.isHealthy();
    const stats = cacheService.getStats();
    
    res.json({
      success: true,
      data: {
        healthy: isHealthy,
        redisConnected: stats.isRedisConnected,
        memoryCacheSize: stats.memoryUsage?.used || 0,
        peakMemory: stats.memoryUsage?.peak || 0,
        hitRate: stats.hitRate.overall,
        errors: stats.errors.total
      }
    });
  } catch (error) {
    console.error('Cache health check error:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to check cache health'
    });
  }
});

router.post('/cache/clear', async (req, res) => {
  try {
    const { pattern } = req.body;
    
    if (pattern) {
      await cacheService.invalidatePattern(pattern);
      res.json({
        success: true,
        message: `Cache invalidated for pattern: ${pattern}`
      });
    } else {
      await cacheService.clear();
      res.json({
        success: true,
        message: 'Cache cleared successfully'
      });
    }
  } catch (error) {
    console.error('Cache clear error:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to clear cache'
    });
  }
});

router.post('/cache/reset-stats', async (req, res) => {
  try {
    cacheService.resetStats();
    res.json({
      success: true,
      message: 'Cache statistics reset successfully'
    });
  } catch (error) {
    console.error('Cache stats reset error:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to reset cache statistics'
    });
  }
});

module.exports = router;
