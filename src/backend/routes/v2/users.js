const express = require('express');

const router = express.Router();

router.get('/', (req, res) => {
  const page = parseInt(req.query.page) || 1;
  const limit = parseInt(req.query.limit) || 10;

  res.json({
    success: true,
    data: {
      users: [
        {
          id: 1,
          name: 'John Doe',
          email: 'john@example.com',
          role: 'user',
          created_at: '2024-01-01'
        },
        {
          id: 2,
          name: 'Jane Smith',
          email: 'jane@example.com',
          role: 'admin',
          created_at: '2024-01-02'
        }
      ],
      pagination: {
        page,
        limit,
        total: 2,
        total_pages: 1
      }
    },
    meta: {
      version: 'v2',
      timestamp: new Date().toISOString()
    }
  });
});

router.get('/:id', (req, res) => {
  res.json({
    success: true,
    data: {
      id: parseInt(req.params.id),
      name: 'John Doe',
      email: 'john@example.com',
      role: 'user',
      created_at: '2024-01-01',
      profile: {
        avatar: null,
        bio: null,
        location: null
      }
    },
    meta: {
      version: 'v2',
      timestamp: new Date().toISOString()
    }
  });
});

module.exports = router;
