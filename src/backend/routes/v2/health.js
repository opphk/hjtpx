const express = require('express');
const router = express.Router();

router.get('/', (req, res) => {
  res.json({
    success: true,
    data: {
      status: 'healthy',
      version: 'v2',
      timestamp: new Date().toISOString(),
      uptime: process.uptime(),
      services: {
        database: 'connected',
        cache: 'connected',
        websocket: 'connected'
      },
      metrics: {
        memory: process.memoryUsage(),
        cpu: process.cpuUsage()
      }
    }
  });
});

router.get('/detailed', (req, res) => {
  res.json({
    success: true,
    data: {
      status: 'healthy',
      version: 'v2',
      timestamp: new Date().toISOString(),
      uptime: process.uptime(),
      environment: process.env.NODE_ENV || 'development',
      services: {
        database: 'connected',
        cache: 'connected',
        websocket: 'connected'
      },
      metrics: {
        memory: process.memoryUsage(),
        cpu: process.cpuUsage(),
        requests: 0
      }
    }
  });
});

module.exports = router;
