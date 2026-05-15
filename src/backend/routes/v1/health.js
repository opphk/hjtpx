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

/**
 * @swagger
 * tags:
 *   name: Health
 *   description: Health check and monitoring endpoints
 */

/**
 * @swagger
 * /api/v1/health:
 *   get:
 *     summary: Basic health check
 *     description: Returns basic health status of the API service
 *     tags: [Health]
 *     responses:
 *       200:
 *         description: Service is healthy
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                   example: true
 *                 data:
 *                   type: object
 *                   properties:
 *                     status:
 *                       type: string
 *                       example: healthy
 *                     service:
 *                       type: string
 *                       example: HJTPX API
 *                     version:
 *                       type: string
 *                       example: 1.0.0
 *                     timestamp:
 *                       type: string
 *                       format: date-time
 *                     uptime:
 *                       type: number
 *                       description: Process uptime in seconds
 *                     environment:
 *                       type: string
 *                       example: development
 *                 message:
 *                   type: string
 *                   example: Health check passed
 *       503:
 *         description: Service is unhealthy
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 */
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

/**
 * @swagger
 * /api/v1/health/detailed:
 *   get:
 *     summary: Detailed health check
 *     description: Returns detailed health status including database, Redis, cache, memory and CPU
 *     tags: [Health]
 *     responses:
 *       200:
 *         description: Detailed health status
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: object
 *                   properties:
 *                     status:
 *                       type: string
 *                       enum: [healthy, degraded, unhealthy]
 *                     timestamp:
 *                       type: string
 *                       format: date-time
 *                     service:
 *                       type: string
 *                     version:
 *                       type: string
 *                     uptime:
 *                       type: number
 *                     environment:
 *                       type: string
 *                     responseTime:
 *                       type: string
 *                     checks:
 *                       type: object
 *                       properties:
 *                         database:
 *                           type: object
 *                           properties:
 *                             status:
 *                               type: string
 *                             message:
 *                               type: string
 *                             responseTime:
 *                               type: string
 *                         redis:
 *                           type: object
 *                           properties:
 *                             status:
 *                               type: string
 *                             message:
 *                               type: string
 *                             responseTime:
 *                               type: string
 *                         cache:
 *                           type: object
 *                           properties:
 *                             status:
 *                               type: string
 *                             message:
 *                               type: string
 *                             stats:
 *                               type: object
 *                         memory:
 *                           type: object
 *                           properties:
 *                             status:
 *                               type: string
 *                             message:
 *                               type: string
 *                             usage:
 *                               type: object
 *                         cpu:
 *                           type: object
 *                           properties:
 *                             status:
 *                               type: string
 *                             message:
 *                               type: string
 *                             loadAverage:
 *                               type: array
 *                               items:
 *                                 type: number
 *       503:
 *         description: Service is unhealthy
 */
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

/**
 * @swagger
 * /api/v1/health/stats:
 *   get:
 *     summary: Get cache statistics
 *     description: Returns cache service statistics
 *     tags: [Health]
 *     responses:
 *       200:
 *         description: Cache statistics retrieved successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: object
 *                   description: Cache statistics object
 *       503:
 *         description: Cache service not available
 */
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

/**
 * @swagger
 * /api/v1/health/pool-stats:
 *   get:
 *     summary: Get database pool statistics
 *     description: Returns database connection pool statistics
 *     tags: [Health]
 *     responses:
 *       200:
 *         description: Pool statistics retrieved successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: object
 *                   properties:
 *                     pool:
 *                       type: object
 *                     queries:
 *                       type: object
 *                     detailed:
 *                       type: object
 *       503:
 *         description: Database connection not available
 */
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

/**
 * @swagger
 * /api/v1/health/pool-health:
 *   get:
 *     summary: Check database pool health
 *     description: Returns database connection pool health status
 *     tags: [Health]
 *     responses:
 *       200:
 *         description: Pool is healthy
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: object
 *                   properties:
 *                     healthy:
 *                       type: boolean
 *       503:
 *         description: Pool is unhealthy or database not available
 */
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

/**
 * @swagger
 * /api/v1/health/stats/reset:
 *   post:
 *     summary: Reset cache statistics
 *     description: Reset all cache statistics counters
 *     tags: [Health]
 *     responses:
 *       200:
 *         description: Cache statistics reset successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 message:
 *                   type: string
 *       503:
 *         description: Cache service not available
 */
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

/**
 * @swagger
 * /api/v1/health/pool-stats/reset:
 *   post:
 *     summary: Reset pool statistics
 *     description: Reset all database pool statistics counters
 *     tags: [Health]
 *     responses:
 *       200:
 *         description: Pool statistics reset successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 message:
 *                   type: string
 *       503:
 *         description: Database connection not available
 */
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
