const express = require('express');
const swaggerUi = require('swagger-ui-express');
const swaggerSpec = require('../config/swagger');
const router = express.Router();

router.use('/', swaggerUi.serve, swaggerUi.setup(swaggerSpec, {
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
}));

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

module.exports = router;
