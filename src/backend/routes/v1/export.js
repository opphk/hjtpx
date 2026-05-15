const express = require('express');

const router = express.Router();
const { authenticateToken, requireAdmin } = require('../middleware/auth');
const { exportData } = require('../services/exportService');

/**
 * @swagger
 * /api/v1/export:
 *   post:
 *     summary: Export data
 *     tags: [Export]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               data:
 *                 type: array
 *                 description: Data to export
 *               format:
 *                 type: string
 *                 enum: [csv, excel, json, pdf]
 *                 default: csv
 *               filename:
 *                 type: string
 *                 default: export
 *               fields:
 *                 type: array
 *                 items:
 *                   type: string
 *     responses:
 *       200:
 *         description: Export file
 */
router.post('/', authenticateToken, async (req, res) => {
  try {
    const { data, format = 'csv', filename = 'export', fields } = req.body;

    if (!data || !Array.isArray(data)) {
      return res.badRequest('Data must be an array');
    }

    const result = await exportData(data, format, {
      filename,
      fields
    });

    res.setHeader('Content-Type', result.mimeType);
    res.setHeader('Content-Disposition', `attachment; filename="${result.filename}"`);

    if (Buffer.isBuffer(result.content)) {
      res.send(result.content);
    } else {
      res.send(result.content);
    }
  } catch (error) {
    res.error(error.message, 500, 'EXPORT_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/export/users:
 *   post:
 *     summary: Export users data
 *     tags: [Export]
 *     security:
 *       - bearerAuth: []
 *       - requireAdmin: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               format:
 *                 type: string
 *                 enum: [csv, excel, json, pdf]
 *               fields:
 *                 type: array
 *                 items:
 *                   type: string
 *     responses:
 *       200:
 *         description: Exported users file
 */
router.post('/users', authenticateToken, requireAdmin, async (req, res) => {
  try {
    const { format = 'csv', fields } = req.body;
    const userService = require('../services/userService');

    const users = await userService.getAllUsers();
    const data = users.map(user => ({
      id: user.id,
      email: user.email,
      username: user.username,
      role: user.role,
      createdAt: user.created_at,
      lastLogin: user.last_login
    }));

    const result = await exportData(data, format, {
      filename: `users_${Date.now()}`,
      fields: fields || ['id', 'email', 'username', 'role', 'createdAt', 'lastLogin']
    });

    res.setHeader('Content-Type', result.mimeType);
    res.setHeader('Content-Disposition', `attachment; filename="${result.filename}"`);
    res.send(result.content);
  } catch (error) {
    res.error(error.message, 500, 'EXPORT_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/export/analytics:
 *   post:
 *     summary: Export analytics data
 *     tags: [Export]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               format:
 *                 type: string
 *                 enum: [csv, excel, json, pdf]
 *               dateRange:
 *                 type: object
 *                 properties:
 *                   start:
 *                     type: string
 *                     format: date
 *                   end:
 *                     type: string
 *                     format: date
 *     responses:
 *       200:
 *         description: Exported analytics file
 */
router.post('/analytics', authenticateToken, requireAdmin, async (req, res) => {
  try {
    const { format = 'csv', dateRange } = req.body;
    const analyticsService = require('../services/analyticsService');

    const stats = {
      engagement: await analyticsService.getUserEngagementStats(30),
      features: await analyticsService.getFeatureUsageStats(30),
      performance: await analyticsService.getApiPerformanceStats(30),
      exportedAt: new Date().toISOString(),
      dateRange: dateRange || 'last 30 days'
    };

    const data = [stats];
    const result = await exportData(data, format, {
      filename: `analytics_${Date.now()}`
    });

    res.setHeader('Content-Type', result.mimeType);
    res.setHeader('Content-Disposition', `attachment; filename="${result.filename}"`);
    res.send(result.content);
  } catch (error) {
    res.error(error.message, 500, 'EXPORT_ERROR');
  }
});

module.exports = router;
