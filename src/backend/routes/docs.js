const express = require('express');
const swaggerUi = require('swagger-ui-express');

const swaggerSpec = require('../config/swagger');
const { generateSwaggerSpec } = require('../config/swagger-auto');
const { getStatsService } = require('../middleware/apiStats');
const ApiVersionManager = require('../utils/apiVersionManager');

const router = express.Router();

const versionManager = new ApiVersionManager();
const statsService = getStatsService();

/**
 * @swagger
 * tags:
 *   name: Docs
 *   description: API documentation and version management endpoints
 */

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

/**
 * @swagger
 * /api-docs/json:
 *   get:
 *     summary: Get OpenAPI JSON specification
 *     description: Returns the API specification in JSON format
 *     tags: [Docs]
 *     responses:
 *       200:
 *         description: OpenAPI JSON specification
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 */
router.get('/json', (req, res) => {
  res.setHeader('Content-Type', 'application/json');
  res.send(swaggerSpec);
});

/**
 * @swagger
 * /api-docs/yaml:
 *   get:
 *     summary: Get OpenAPI YAML specification
 *     description: Returns the API specification in YAML format
 *     tags: [Docs]
 *     responses:
 *       200:
 *         description: OpenAPI YAML specification
 *         content:
 *           text/yaml:
 *             schema:
 *               type: string
 */
router.get('/yaml', (req, res) => {
  const yaml = require('js-yaml');
  const spec = yaml.dump(swaggerSpec);
  res.setHeader('Content-Type', 'text/yaml');
  res.send(spec);
});

/**
 * @swagger
 * /api-docs/versions:
 *   get:
 *     summary: List all API versions
 *     description: Returns a list of all saved API versions
 *     tags: [Docs]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: List of API versions
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: array
 *                   items:
 *                     $ref: '#/components/schemas/ApiVersionInfo'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
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

/**
 * @swagger
 * /api-docs/versions/{version}:
 *   get:
 *     summary: Get specific API version
 *     description: Returns the OpenAPI specification for a specific version
 *     tags: [Docs]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: version
 *         required: true
 *         schema:
 *           type: string
 *         description: API version (e.g., 1.0.0)
 *     responses:
 *       200:
 *         description: API specification for the requested version
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *       404:
 *         $ref: '#/components/responses/NotFound'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
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

/**
 * @swagger
 * /api-docs/versions/{version}/ui:
 *   get:
 *     summary: Get API version UI
 *     description: Returns the Swagger UI for a specific API version
 *     tags: [Docs]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: version
 *         required: true
 *         schema:
 *           type: string
 *         description: API version (e.g., 1.0.0)
 *     responses:
 *       200:
 *         description: Swagger UI HTML for the requested version
 *         content:
 *           text/html:
 *             schema:
 *               type: string
 *       404:
 *         $ref: '#/components/responses/NotFound'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
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

/**
 * @swagger
 * /api-docs/versions:
 *   post:
 *     summary: Save current API version
 *     description: Save the current API specification as a new version
 *     tags: [Docs]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               description:
 *                 type: string
 *                 description: Version description
 *     responses:
 *       200:
 *         description: Version saved successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   $ref: '#/components/schemas/ApiVersionInfo'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
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

/**
 * @swagger
 * /api-docs/versions/{version}:
 *   delete:
 *     summary: Delete API version
 *     description: Delete a specific API version
 *     tags: [Docs]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: version
 *         required: true
 *         schema:
 *           type: string
 *         description: API version to delete
 *     responses:
 *       200:
 *         description: Version deleted successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 message:
 *                   type: string
 *       404:
 *         $ref: '#/components/responses/NotFound'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
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

/**
 * @swagger
 * /api-docs/compare/{version1}/{version2}:
 *   get:
 *     summary: Compare two API versions
 *     description: Compare two API versions and return the differences
 *     tags: [Docs]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: version1
 *         required: true
 *         schema:
 *           type: string
 *         description: First API version
 *       - in: path
 *         name: version2
 *         required: true
 *         schema:
 *           type: string
 *         description: Second API version
 *     responses:
 *       200:
 *         description: Version comparison result
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: object
 *                   properties:
 *                     added:
 *                       type: array
 *                       items:
 *                         type: string
 *                     removed:
 *                       type: array
 *                       items:
 *                         type: string
 *       404:
 *         $ref: '#/components/responses/NotFound'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
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

/**
 * @swagger
 * /api-docs/stats:
 *   get:
 *     summary: Get API documentation stats
 *     description: Returns statistics about API documentation usage
 *     tags: [Docs]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: API documentation statistics
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: object
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
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

/**
 * @swagger
 * /api-docs/stats/endpoint:
 *   get:
 *     summary: Get endpoint stats
 *     description: Returns statistics for a specific endpoint
 *     tags: [Docs]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: query
 *         name: method
 *         required: true
 *         schema:
 *           type: string
 *         description: HTTP method
 *       - in: query
 *         name: path
 *         required: true
 *         schema:
 *           type: string
 *         description: Endpoint path
 *     responses:
 *       200:
 *         description: Endpoint statistics
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: object
 *       400:
 *         $ref: '#/components/responses/ValidationError'
 *       404:
 *         $ref: '#/components/responses/NotFound'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
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

/**
 * @swagger
 * /api-docs/stats:
 *   delete:
 *     summary: Clear API stats
 *     description: Clear all API documentation statistics
 *     tags: [Docs]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: Stats cleared successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 message:
 *                   type: string
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
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
