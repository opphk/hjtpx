const express = require('express');
const router = express.Router();
const { versionNegotiationMiddleware } = require('../../middleware/apiVersionNegotiation');
const apiVersionManager = require('../../middleware/apiVersionManager');

router.use(versionNegotiationMiddleware);

router.get('/', (req, res) => {
  const report = apiVersionManager.generateVersionReport();

  res.json({
    success: true,
    data: {
      currentVersion: req.apiVersion,
      versionInfo: req.versionInfo,
      report,
      supportedVersions: report.supportedVersions.map(v => v.version),
      deprecatedVersions: report.deprecatedVersions.map(v => ({
        version: v.version,
        sunsetDate: v.sunsetDate
      }))
    }
  });
});

router.get('/supported', (req, res) => {
  const versions = apiVersionManager.getSupportedVersions();

  res.json({
    success: true,
    data: {
      versions: versions.map(v => ({
        version: v.version,
        status: v.status,
        releaseDate: v.releaseDate
      }))
    }
  });
});

router.get('/deprecated', (req, res) => {
  const deprecated = apiVersionManager.getAllVersions().filter(v => v.isDeprecated);

  const deprecationDetails = deprecated.map(v => {
    const warnings = apiVersionManager.getDeprecationWarnings(v.version);
    return {
      version: v.version,
      status: v.status,
      sunsetDate: v.sunsetDate,
      warnings: warnings || []
    };
  });

  res.json({
    success: true,
    data: {
      deprecatedVersions: deprecationDetails
    }
  });
});

router.get('/migration/:from/:to', (req, res) => {
  const { from, to } = req.params;
  const strategy = apiVersionManager.getMigrationStrategy(from, to);

  if (!strategy) {
    return res.status(404).json({
      success: false,
      error: {
        code: 'MIGRATION_NOT_FOUND',
        message: `Migration path from ${from} to ${to} not found`
      }
    });
  }

  res.json({
    success: true,
    data: {
      migration: strategy
    }
  });
});

router.post('/deprecate/:version', (req, res) => {
  const { version } = req.params;
  const { sunsetDate } = req.body;

  if (!version || !sunsetDate) {
    return res.status(400).json({
      success: false,
      error: {
        code: 'INVALID_REQUEST',
        message: 'Version and sunset date are required'
      }
    });
  }

  apiVersionManager.deprecateVersion(version, sunsetDate);

  res.json({
    success: true,
    data: {
      message: `Version ${version} deprecated successfully`,
      sunsetDate
    }
  });
});

module.exports = router;
