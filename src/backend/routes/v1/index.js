const express = require('express');
const router = express.Router();

const healthRoutes = require('./health');
const usersRoutes = require('./users');
const authRoutes = require('./auth');

router.use('/health', healthRoutes);
router.use('/users', usersRoutes);
router.use('/auth', authRoutes);

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
        auth: '/api/v1/auth'
      },
      timestamp: new Date().toISOString()
    }
  });
});

module.exports = router;
