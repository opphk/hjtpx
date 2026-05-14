const express = require('express');
const router = express.Router();
const analyticsService = require('../services/analyticsService');
const { authenticateToken, requireAdmin } = require('../middleware/auth');

/**
 * @swagger
 * /api/v1/analytics/dashboard:
 *   get:
 *     summary: Get dashboard summary
 *     tags: [Analytics]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: Dashboard summary data
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   $ref: '#/components/schemas/AnalyticsData'
 */
router.get('/dashboard', authenticateToken, async (req, res) => {
  try {
    const summary = await analyticsService.getDashboardSummary();
    res.success(summary);
  } catch (error) {
    res.error(error.message, 500, 'ANALYTICS_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/analytics/users/{userId}/activity:
 *   get:
 *     summary: Get user activity statistics
 *     tags: [Analytics]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: userId
 *         required: true
 *         schema:
 *           type: string
 *       - in: query
 *         name: days
 *         schema:
 *           type: integer
 *           default: 7
 *     responses:
 *       200:
 *         description: User activity statistics
 */
router.get('/users/:userId/activity', authenticateToken, async (req, res) => {
  try {
    const { userId } = req.params;
    const { days = 7 } = req.query;
    const stats = await analyticsService.getUserActivityStats(userId, parseInt(days));
    res.success(stats);
  } catch (error) {
    res.error(error.message, 500, 'ANALYTICS_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/analytics/features/usage:
 *   get:
 *     summary: Get feature usage statistics
 *     tags: [Analytics]
 *     security:
 *       - bearerAuth: []
 *       - requireAdmin: []
 *     parameters:
 *       - in: query
 *         name: days
 *         schema:
 *           type: integer
 *           default: 7
 *     responses:
 *       200:
 *         description: Feature usage statistics
 */
router.get('/features/usage', authenticateToken, requireAdmin, async (req, res) => {
  try {
    const { days = 7 } = req.query;
    const stats = await analyticsService.getFeatureUsageStats(parseInt(days));
    res.success(stats);
  } catch (error) {
    res.error(error.message, 500, 'ANALYTICS_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/analytics/engagement:
 *   get:
 *     summary: Get user engagement statistics
 *     tags: [Analytics]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: query
 *         name: days
 *         schema:
 *           type: integer
 *           default: 7
 *     responses:
 *       200:
 *         description: Engagement statistics
 */
router.get('/engagement', authenticateToken, async (req, res) => {
  try {
    const { days = 7 } = req.query;
    const stats = await analyticsService.getUserEngagementStats(parseInt(days));
    res.success(stats);
  } catch (error) {
    res.error(error.message, 500, 'ANALYTICS_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/analytics/realtime:
 *   get:
 *     summary: Get real-time statistics
 *     tags: [Analytics]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: Real-time analytics data
 */
router.get('/realtime', authenticateToken, async (req, res) => {
  try {
    const stats = await analyticsService.getRealTimeStats();
    res.success(stats);
  } catch (error) {
    res.error(error.message, 500, 'ANALYTICS_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/analytics/performance:
 *   get:
 *     summary: Get API performance statistics
 *     tags: [Analytics]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: query
 *         name: days
 *         schema:
 *           type: integer
 *           default: 7
 *     responses:
 *       200:
 *         description: API performance statistics
 */
router.get('/performance', authenticateToken, requireAdmin, async (req, res) => {
  try {
    const { days = 7 } = req.query;
    const stats = await analyticsService.getApiPerformanceStats(parseInt(days));
    res.success(stats);
  } catch (error) {
    res.error(error.message, 500, 'ANALYTICS_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/analytics/anomalies:
 *   get:
 *     summary: Detect anomalies in the system
 *     tags: [Analytics]
 *     security:
 *       - bearerAuth: []
 *       - requireAdmin: []
 *     responses:
 *       200:
 *         description: Detected anomalies
 */
router.get('/anomalies', authenticateToken, requireAdmin, async (req, res) => {
  try {
    const anomalies = await analyticsService.detectAnomalies();
    res.success(anomalies);
  } catch (error) {
    res.error(error.message, 500, 'ANALYTICS_ERROR');
  }
});

module.exports = router;
