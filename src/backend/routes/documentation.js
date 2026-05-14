const express = require('express');
const router = express.Router();
const swaggerSpec = require('../config/swagger');
const apiDiffDetector = require('../utils/apiDiffDetector');
const apiUsageTracker = require('../utils/apiUsageTracker');
const apiVersionManager = require('../utils/apiVersionManager');

router.post('/update', async (req, res) => {
  try {
    const currentHash = await apiDiffDetector.saveCurrentSpec(swaggerSpec);
    const changes = await apiDiffDetector.detectChanges(swaggerSpec);
    
    if (changes.hasChanges) {
      await apiDiffDetector.recordVersion(swaggerSpec, changes);
      await apiDiffDetector.generateChangeReport(changes);
    }

    res.json({
      success: true,
      message: 'API documentation updated',
      hash: currentHash,
      changes: changes
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: {
        message: 'Failed to update documentation',
        details: error.message
      }
    });
  }
});

router.get('/diff', async (req, res) => {
  try {
    const changes = await apiDiffDetector.detectChanges(swaggerSpec);
    
    res.json({
      success: true,
      changes
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: {
        message: 'Failed to generate diff',
        details: error.message
      }
    });
  }
});

router.get('/versions', async (req, res) => {
  try {
    const versions = await apiVersionManager.listVersions();
    
    res.json({
      success: true,
      versions
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: {
        message: 'Failed to list versions',
        details: error.message
      }
    });
  }
});

router.post('/versions/:version', async (req, res) => {
  try {
    const { version } = req.params;
    const versionData = await apiVersionManager.createVersion(version, swaggerSpec, req.body.metadata || {});
    
    res.json({
      success: true,
      message: `Version ${version} created`,
      data: versionData
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: {
        message: 'Failed to create version',
        details: error.message
      }
    });
  }
});

router.get('/versions/:version', async (req, res) => {
  try {
    const { version } = req.params;
    const versionData = await apiVersionManager.getVersion(version);
    
    if (!versionData) {
      return res.status(404).json({
        success: false,
        error: {
          message: 'Version not found'
        }
      });
    }
    
    res.json({
      success: true,
      data: versionData
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: {
        message: 'Failed to get version',
        details: error.message
      }
    });
  }
});

router.get('/versions/:v1/compare/:v2', async (req, res) => {
  try {
    const { v1, v2 } = req.params;
    const comparison = await apiVersionManager.compareVersions(v1, v2);
    
    res.json({
      success: true,
      comparison
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: {
        message: 'Failed to compare versions',
        details: error.message
      }
    });
  }
});

router.post('/versions/:version/deprecate', async (req, res) => {
  try {
    const { version } = req.params;
    const { sunsetDate } = req.body;
    
    const deprecated = await apiVersionManager.deprecateVersion(version, sunsetDate);
    
    res.json({
      success: true,
      message: `Version ${version} deprecated`,
      data: deprecated
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: {
        message: 'Failed to deprecate version',
        details: error.message
      }
    });
  }
});

router.get('/usage/stats', async (req, res) => {
  try {
    const stats = apiUsageTracker.getStats();
    
    res.json({
      success: true,
      stats
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: {
        message: 'Failed to get usage stats',
        details: error.message
      }
    });
  }
});

router.get('/usage/report', async (req, res) => {
  try {
    const report = await apiUsageTracker.generateDailyReport();
    
    res.json({
      success: true,
      report
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: {
        message: 'Failed to generate report',
        details: error.message
      }
    });
  }
});

router.get('/coverage', async (req, res) => {
  try {
    const coverage = apiUsageTracker.getDocumentationStats();
    
    res.json({
      success: true,
      coverage
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: {
        message: 'Failed to get coverage',
        details: error.message
      }
    });
  }
});

module.exports = router;
