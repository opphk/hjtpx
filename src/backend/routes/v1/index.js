const express = require('express');

const router = express.Router();

const adminRoutes = require('./admin');
const authRoutes = require('./auth');
const healthRoutes = require('./health');
const monitoringRoutes = require('./monitoring');
const passwordRoutes = require('./password');
const performanceRoutes = require('./performance');
const securityRoutes = require('./security');
const usersRoutes = require('./users');

router.use('/health', healthRoutes);
router.use('/users', usersRoutes);
router.use('/auth', authRoutes);
router.use('/password', passwordRoutes);
router.use('/performance', performanceRoutes);
router.use('/admin', adminRoutes);
router.use('/monitoring', monitoringRoutes);
router.use('/security', securityRoutes);

router.get('/', (req, res) => {
  res.json({
    success: true,
    data: {
      version: 'v1',
      name: 'HJTPX API v1',
      description: 'HJTPX API Version 1',
      endpoints: {
        health: '/api/v1/health',
        users: '/api/v1/users',
        auth: '/api/v1/auth',
        password: '/api/v1/password',
        performance: '/api/v1/performance',
        admin: '/api/v1/admin',
        monitoring: '/api/v1/monitoring'
      },
      timestamp: new Date().toISOString()
    }
  });
});

module.exports = router;
