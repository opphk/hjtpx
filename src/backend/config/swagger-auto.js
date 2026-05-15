const swaggerJsdoc = require('swagger-jsdoc');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const packageJson = require('../../../package.json');

const CURRENT_VERSION = packageJson.version;

const getGitCommitHash = () => {
  try {
    return execSync('git rev-parse --short HEAD', { encoding: 'utf-8' }).trim();
  } catch {
    return 'unknown';
  }
};

const swaggerOptions = {
  definition: {
    openapi: '3.0.0',
    info: {
      title: 'HJTPX API Documentation',
      version: CURRENT_VERSION,
      description: 'Complete API documentation for HJTPX application',
      contact: {
        name: 'API Support',
        email: 'support@hjtpx.com'
      },
      license: {
        name: 'MIT',
        url: 'https://opensource.org/licenses/MIT'
      },
      "x-build": {
        commitHash: getGitCommitHash(),
        buildTime: new Date().toISOString()
      }
    },
    servers: [
      {
        url: 'http://localhost:3000',
        description: 'Development server'
      },
      {
        url: 'https://api.hjtpx.com',
        description: 'Production server'
      }
    ],
    tags: [
      { name: 'Authentication', description: 'User authentication endpoints' },
      { name: 'Users', description: 'User management endpoints' },
      { name: 'Analytics', description: 'Analytics and statistics endpoints' },
      { name: 'Notifications', description: 'Notification management endpoints' },
      { name: 'Search', description: 'Search and filtering endpoints' },
      { name: 'Export', description: 'Data export endpoints' },
      { name: 'Health', description: 'Health check endpoints' },
      { name: 'Docs', description: 'API documentation and version management' }
    ],
    components: {
      securitySchemes: {
        bearerAuth: {
          type: 'http',
          scheme: 'bearer',
          bearerFormat: 'JWT',
          description: 'Enter JWT token'
        },
        apiKeyAuth: {
          type: 'apiKey',
          in: 'header',
          name: 'X-API-Key',
          description: 'API key for external access'
        }
      },
      schemas: {
        User: {
          type: 'object',
          properties: {
            id: { type: 'string', format: 'uuid' },
            email: { type: 'string', format: 'email' },
            username: { type: 'string' },
            role: { type: 'string', enum: ['user', 'admin', 'moderator'] },
            createdAt: { type: 'string', format: 'date-time' },
            updatedAt: { type: 'string', format: 'date-time' }
          }
        },
        Notification: {
          type: 'object',
          properties: {
            id: { type: 'string', format: 'uuid' },
            userId: { type: 'string', format: 'uuid' },
            title: { type: 'string' },
            message: { type: 'string' },
            type: { type: 'string', enum: ['system', 'user', 'security', 'promotion'] },
            status: { type: 'string', enum: ['unread', 'read', 'archived'] },
            channels: {
              type: 'array',
              items: { type: 'string', enum: ['email', 'sms', 'push', 'in_app'] }
            },
            createdAt: { type: 'string', format: 'date-time' }
          }
        },
        AnalyticsData: {
          type: 'object',
          properties: {
            activeUsers: { type: 'integer' },
            eventsToday: { type: 'integer' },
            topFeatures: {
              type: 'array',
              items: {
                type: 'object',
                properties: {
                  feature: { type: 'string' },
                  usageCount: { type: 'integer' }
                }
              }
            },
            recentActivity: {
              type: 'array',
              items: { $ref: '#/components/schemas/ActivityEvent' }
            }
          }
        },
        ActivityEvent: {
          type: 'object',
          properties: {
            userId: { type: 'string' },
            eventType: { type: 'string' },
            createdAt: { type: 'string', format: 'date-time' }
          }
        },
        SearchResult: {
          type: 'object',
          properties: {
            data: { type: 'array', items: {} },
            pagination: {
              type: 'object',
              properties: {
                page: { type: 'integer' },
                limit: { type: 'integer' },
                total: { type: 'integer' },
                pages: { type: 'integer' }
              }
            },
            meta: {
              type: 'object',
              properties: {
                queryTime: { type: 'integer' },
                searchedFields: { type: 'array', items: { type: 'string' } }
              }
            }
          }
        },
        Error: {
          type: 'object',
          properties: {
            success: { type: 'boolean', default: false },
            error: {
              type: 'object',
              properties: {
                message: { type: 'string' },
                code: { type: 'string' },
                details: { type: 'string' }
              }
            }
          }
        },
        SuccessResponse: {
          type: 'object',
          properties: {
            success: { type: 'boolean', default: true },
            data: { type: 'object' },
            message: { type: 'string' }
          }
        },
        ApiUsageStats: {
          type: 'object',
          properties: {
            endpoint: { type: 'string' },
            method: { type: 'string' },
            callCount: { type: 'integer' },
            averageResponseTime: { type: 'number' },
            errorRate: { type: 'number' },
            lastCalledAt: { type: 'string', format: 'date-time' }
          }
        },
        ApiVersionInfo: {
          type: 'object',
          properties: {
            version: { type: 'string' },
            createdAt: { type: 'string', format: 'date-time' },
            commitHash: { type: 'string' },
            description: { type: 'string' },
            downloadUrl: { type: 'string' }
          }
        }
      },
      responses: {
        Unauthorized: {
          description: 'Unauthorized - Invalid or missing authentication',
          content: {
            'application/json': {
              schema: { $ref: '#/components/schemas/Error' }
            }
          }
        },
        Forbidden: {
          description: 'Forbidden - Insufficient permissions',
          content: {
            'application/json': {
              schema: { $ref: '#/components/schemas/Error' }
            }
          }
        },
        NotFound: {
          description: 'Not Found - Resource does not exist',
          content: {
            'application/json': {
              schema: { $ref: '#/components/schemas/Error' }
            }
          }
        },
        ValidationError: {
          description: 'Validation Error - Invalid input data',
          content: {
            'application/json': {
              schema: { $ref: '#/components/schemas/Error' }
            }
          }
        },
        InternalServerError: {
          description: 'Internal Server Error',
          content: {
            'application/json': {
              schema: { $ref: '#/components/schemas/Error' }
            }
          }
        }
      }
    },
    security: [
      { bearerAuth: [] }
    ]
  },
  apis: [
    './src/backend/routes/*.js',
    './src/backend/routes/v1/*.js',
    './src/backend/routes/v2/*.js'
  ]
};

const generateSwaggerSpec = () => {
  return swaggerJsdoc(swaggerOptions);
};

const saveSwaggerSpec = (outputDir = './docs') => {
  const spec = generateSwaggerSpec();
  
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }
  
  const jsonPath = path.join(outputDir, 'openapi.json');
  fs.writeFileSync(jsonPath, JSON.stringify(spec, null, 2));
  console.log(`✅ Swagger JSON saved to: ${jsonPath}`);
  
  try {
    const yaml = require('js-yaml');
    const yamlPath = path.join(outputDir, 'openapi.yaml');
    fs.writeFileSync(yamlPath, yaml.dump(spec));
    console.log(`✅ Swagger YAML saved to: ${yamlPath}`);
  } catch (e) {
    console.log('⚠️ YAML generation skipped (js-yaml not available)');
  }
  
  return spec;
};

const getSwaggerOptions = () => swaggerOptions;

module.exports = {
  generateSwaggerSpec,
  saveSwaggerSpec,
  swaggerOptions,
  getSwaggerOptions
};
