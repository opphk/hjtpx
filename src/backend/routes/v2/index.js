const express = require('express');

const router = express.Router();

const healthRoutes = require('./health');
const usersRoutes = require('./users');

router.use('/health', healthRoutes);
router.use('/users', usersRoutes);

router.get('/', (req, res) => {
  res.json({
    success: true,
    data: {
      version: 'v2',
      name: 'HJTPX API v2',
      description: 'HJTPX API Version 2 (Improved)',
      endpoints: {
        health: '/api/v2/health',
        users: '/api/v2/users'
      },
      features: [
        'Enhanced response format',
        'Better error handling',
        'Pagination by default',
        'Rate limiting improvements'
      ],
      timestamp: new Date().toISOString()
    }
  });
});

module.exports = router;
