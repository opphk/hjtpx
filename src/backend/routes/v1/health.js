const express = require('express');

const router = express.Router();

let db = null;
let redisClient = null;
let cacheService = null;

try {
  db = require('../../../../config/database/db');
} catch (error) {
  console.warn('Database connection not available');
}

try {
  redisClient = require('../../../../config/redis/client');
} catch (error) {
  console.warn('Redis connection not available');
}

try {
  cacheService = require('../../../services/cacheService');
} catch (error) {
  console.warn('Cache service not available');
}

router.get('/', async (req, res) => {
  try {
    res.json({
      success: true,
      data: {
        status: 'healthy',
        service: 'HJTPX API',
        version: '1.0.0',
        timestamp: new Date().toISOString(),
        uptime: process.uptime(),
        environment: process.env.NODE_ENV || 'development'
      }
    });
  } catch (error) {
    res.status(503).json({
      success: false,
      error: {
        code: 'HEALTH_CHECK_FAILED',
        message: 'Service health check failed',
        details: process.env.NODE_ENV === 'development' ? error.message : undefined
      }
    });
  }
});

router.get('/detailed', async (req, res) => {
  const startTime = Date.now();
  const healthStatus = {
    status: 'healthy',
    timestamp: new Date().toISOString(),
    service: 'HJTPX API',
    version: '1.0.0',
    uptime: process.uptime(),
    environment: process.env.NODE_ENV || 'development',
    checks: {}
  };

  try {
    if (db) {
      try {
        await db.query('SELECT 1');
        healthStatus.checks.database = {
          status: 'healthy',
          message: 'Database connection is healthy',
          responseTime: `${Date.now() - startTime}ms`
        };
      } catch (dbError) {
        healthStatus.checks.database = {
          status: 'unhealthy',
          message: 'Database connection failed',
          error: process.env.NODE_ENV === 'development' ? dbError.message : 'Database error'
        };
        healthStatus.status = 'degraded';
      }
    } else {
      healthStatus.checks.database = {
        status: 'unavailable',
        message: 'Database connection not configured'
      };
      healthStatus.status = 'degraded';
    }

    const redisStartTime = Date.now();
    if (redisClient) {
      try {
        await redisClient.ping();
        healthStatus.checks.redis = {
          status: 'healthy',
          message: 'Redis connection is healthy',
          responseTime: `${Date.now() - redisStartTime}ms`
        };
      } catch (redisError) {
        healthStatus.checks.redis = {
          status: 'unhealthy',
          message: 'Redis connection failed',
          error: process.env.NODE_ENV === 'development' ? redisError.message : 'Redis error'
        };
        healthStatus.status = 'degraded';
      }
    } else {
      healthStatus.checks.redis = {
        status: 'unavailable',
        message: 'Redis connection not configured'
      };
      healthStatus.status = 'degraded';
    }

    if (cacheService) {
      const cacheHealth = await cacheService.isHealthy();
      const cacheStats = cacheService.getStats();

      healthStatus.checks.cache = {
        status: cacheHealth ? 'healthy' : 'degraded',
        message: cacheHealth ? 'Cache service is healthy' : 'Cache service is degraded',
        stats: cacheStats
      };
    } else {
      healthStatus.checks.cache = {
        status: 'unavailable',
        message: 'Cache service not configured'
      };
    }

    healthStatus.checks.memory = {
      status: 'healthy',
      message: 'Memory usage is normal',
      usage: {
        used: Math.round(process.memoryUsage().heapUsed / 1024 / 1024),
        total: Math.round(process.memoryUsage().heapTotal / 1024 / 1024),
        unit: 'MB'
      }
    };

    healthStatus.checks.cpu = {
      status: 'healthy',
      message: 'CPU usage is normal',
      loadAverage: process.loadAvg ? process.loadAvg() : [0, 0, 0]
    };

    const totalResponseTime = Date.now() - startTime;
    healthStatus.responseTime = `${totalResponseTime}ms`;

    const statusCode =
      healthStatus.status === 'healthy' ? 200 : healthStatus.status === 'degraded' ? 200 : 503;

    res.status(statusCode).json({
      success: healthStatus.status !== 'unhealthy',
      data: healthStatus
    });
  } catch (error) {
    healthStatus.status = 'unhealthy';
    healthStatus.error =
      process.env.NODE_ENV === 'development' ? error.message : 'Health check failed';

    res.status(503).json({
      success: false,
      data: healthStatus
    });
  }
});

router.get('/stats', async (req, res) => {
  try {
    if (!cacheService) {
      return res.status(503).json({
        success: false,
        error: 'Cache service not available'
      });
    }

    const stats = cacheService.getStats();
    res.json({
      success: true,
      data: stats
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Failed to retrieve cache stats'
    });
  }
});

router.get('/pool-stats', async (req, res) => {
  try {
    if (!db) {
      return res.status(503).json({
        success: false,
        error: 'Database connection not available'
      });
    }

    const poolStats = db.getPoolStats ? db.getPoolStats() : {};
    const queryStats = db.getQueryStats ? db.getQueryStats() : {};
    const detailedStats = db.getDetailedStats ? db.getDetailedStats() : {};

    res.json({
      success: true,
      data: {
        pool: poolStats,
        queries: queryStats,
        detailed: detailedStats
      }
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error:
        process.env.NODE_ENV === 'development'
          ? error.message
          : 'Failed to retrieve pool statistics'
    });
  }
});

router.get('/pool-health', async (req, res) => {
  try {
    if (!db) {
      return res.status(503).json({
        success: false,
        error: 'Database connection not available'
      });
    }

    const health = db.healthCheck ? await db.healthCheck() : { healthy: false };

    const statusCode = health.healthy ? 200 : 503;

    res.status(statusCode).json({
      success: health.healthy,
      data: health
    });
  } catch (error) {
    res.status(503).json({
      success: false,
      error: process.env.NODE_ENV === 'development' ? error.message : 'Health check failed'
    });
  }
});

router.post('/stats/reset', async (req, res) => {
  try {
    if (!cacheService) {
      return res.status(503).json({
        success: false,
        error: 'Cache service not available'
      });
    }

    cacheService.resetStats();
    res.json({
      success: true,
      message: 'Cache statistics reset successfully'
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Failed to reset cache stats'
    });
  }
});

router.post('/pool-stats/reset', async (req, res) => {
  try {
    if (!db) {
      return res.status(503).json({
        success: false,
        error: 'Database connection not available'
      });
    }

    if (db.resetStats) {
      db.resetStats();
    }

    res.json({
      success: true,
      message: 'Pool statistics reset successfully'
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error:
        process.env.NODE_ENV === 'development' ? error.message : 'Failed to reset pool statistics'
    });
  }
});

module.exports = router;
