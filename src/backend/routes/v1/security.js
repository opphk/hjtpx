const express = require('express');

const { createCSPReportEndpoint } = require('../../middleware/securityHeaders');

const router = express.Router();

router.post(
  '/csp-report',
  express.json({ type: ['application/csp-report', 'application/json'] }),
  createCSPReportEndpoint()
);

router.get('/headers', (req, res) => {
  res.json({
    success: true,
    data: {
      message: 'Security headers are properly configured',
      headers: {
        'X-Content-Type-Options': 'nosniff',
        'X-Frame-Options': 'DENY/SAMEORIGIN',
        'X-XSS-Protection': '1; mode=block',
        'Strict-Transport-Security': 'max-age=31536000; includeSubDomains; preload',
        'Content-Security-Policy': 'Configured with nonce',
        'Permissions-Policy': 'Restricted'
      }
    }
  });
});

module.exports = router;
