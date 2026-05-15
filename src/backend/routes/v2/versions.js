const express = require('express');
const router = express.Router();

const { deprecatedVersionMiddleware } = require('../../middleware/apiVersionNegotiation');

router.use(deprecatedVersionMiddleware);

router.get('/', (req, res) => {
  res.json({
    success: true,
    data: {
      currentVersion: 'v2',
      status: 'stable',
      message: 'This is the v2 versions endpoint'
    }
  });
});

module.exports = router;
