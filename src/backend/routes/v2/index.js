const express = require('express');

const router = express.Router();

const adminRoutes = require('./admin');
const authRoutes = require('./auth');
const healthRoutes = require('./health');
const passwordRoutes = require('./password');
const performanceRoutes = require('./performance');
const usersRoutes = require('./users');
const versionsRoutes = require('./versions');

router.use('/health', healthRoutes);
router.use('/users', usersRoutes);
router.use('/auth', authRoutes);
router.use('/password', passwordRoutes);
router.use('/performance', performanceRoutes);
router.use('/admin', adminRoutes);
router.use('/versions', versionsRoutes);

router.get('/', (req, res) => {
  res.json({
    success: true,
    data: {
      version: 'v2',
      name: 'HJTPX API v2',
      description: 'HJTPX API Version 2 (Beta)',
      endpoints: {
        health: '/api/v2/health',
        users: '/api/v2/users',
        auth: '/api/v2/auth',
        password: '/api/v2/password',
        performance: '/api/v2/performance',
        admin: '/api/v2/admin',
        versions: '/api/v2/versions'
      },
      changes: {
        responseFormat: 'Updated user response to include metadata object',
        pagination: 'Changed from page-based to offset-based pagination',
        errorFormat: 'Enhanced error responses with code and details fields'
      },
      timestamp: new Date().toISOString()
    }
  });
});

module.exports = router;
