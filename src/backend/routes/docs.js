const express = require('express');

const router = express.Router();
const swaggerUi = require('swagger-ui-express');

const swaggerSpec = require('../config/swagger');
const { generateSwaggerSpec } = require('../config/swagger-auto');
const { getStatsService } = require('../middleware/apiStats');
const ApiVersionManager = require('../utils/apiVersionManager');

const versionManager = new ApiVersionManager();
const statsService = getStatsService();

router.use(
  '/',
  swaggerUi.serve,
  swaggerUi.setup(swaggerSpec, {
    customCss: `
    .swagger-ui .topbar { display: none }
    .swagger-ui .info .title { color: #2c3e50; }
    .swagger-ui .scheme-container { background-color: #f8f9fa; padding: 15px; }
  `,
    customSiteTitle: 'HJTPX API Documentation',
    customfavIcon: '/favicon.ico',
    swaggerOptions: {
      persistAuthorization: true,
      displayRequestDuration: true,
      docExpansion: 'list',
      filter: true,
      showExtensions: true,
      showCommonExtensions: true,
      tryItOutEnabled: true
    }
  })
);

router.get('/json', (req, res) => {
  res.setHeader('Content-Type', 'application/json');
  res.send(swaggerSpec);
});

router.get('/yaml', (req, res) => {
  const yaml = require('js-yaml');
  const spec = yaml.dump(swaggerSpec);
  res.setHeader('Content-Type', 'text/yaml');
  res.send(spec);
});

router.get('/versions', (req, res) => {
  try {
    const versions = versionManager.getVersions();
    res.json({
      success: true,
      data: versions
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/versions/:version', (req, res) => {
  try {
    const { version } = req.params;
    const spec = versionManager.loadVersionSpec(version);
    if (!spec) {
      return res.status(404).json({
        success: false,
        error: 'Version not found'
      });
    }
    res.setHeader('Content-Type', 'application/json');
    res.send(spec);
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/versions/:version/ui', (req, res) => {
  try {
    const { version } = req.params;
    const spec = versionManager.loadVersionSpec(version);
    if (!spec) {
      return res.status(404).json({
        success: false,
        error: 'Version not found'
      });
    }
    const html = swaggerUi.generateHTML(spec, {
      customSiteTitle: `HJTPX API v${version}`,
      swaggerOptions: {
        persistAuthorization: true
      }
    });
    res.send(html);
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.post('/versions', (req, res) => {
  try {
    const { description } = req.body;
    const spec = generateSwaggerSpec();
    const versionInfo = versionManager.saveVersion(spec, description || '');
    res.json({
      success: true,
      data: versionInfo
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.delete('/versions/:version', (req, res) => {
  try {
    const { version } = req.params;
    const deleted = versionManager.deleteVersion(version);
    if (!deleted) {
      return res.status(404).json({
        success: false,
        error: 'Version not found'
      });
    }
    res.json({
      success: true,
      message: 'Version deleted successfully'
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/compare/:version1/:version2', (req, res) => {
  try {
    const { version1, version2 } = req.params;
    const changes = versionManager.compareVersions(version1, version2);
    if (!changes) {
      return res.status(404).json({
        success: false,
        error: 'One or both versions not found'
      });
    }
    res.json({
      success: true,
      data: changes
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/stats', (req, res) => {
  try {
    const stats = statsService.getStats();
    res.json({
      success: true,
      data: stats
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.get('/stats/endpoint', (req, res) => {
  try {
    const { method, path } = req.query;
    if (!method || !path) {
      return res.status(400).json({
        success: false,
        error: 'method and path query parameters are required'
      });
    }
    const stats = statsService.getEndpointStats(method, path);
    if (!stats) {
      return res.status(404).json({
        success: false,
        error: 'Endpoint not found'
      });
    }
    res.json({
      success: true,
      data: stats
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

router.delete('/stats', (req, res) => {
  try {
    statsService.clearStats();
    res.json({
      success: true,
      message: 'Stats cleared successfully'
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

module.exports = router;
