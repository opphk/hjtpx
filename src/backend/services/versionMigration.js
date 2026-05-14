const apiVersionManager = require('../middleware/apiVersionManager');

const migrationStrategies = {
  v1Tov2: {
    from: 'v1',
    to: 'v2',
    breakingChanges: [
      {
        type: 'endpoint',
        description: 'User endpoint response format changed',
        v1Format: {
          user: {
            id: 'string',
            name: 'string',
            email: 'string'
          }
        },
        v2Format: {
          user: {
            id: 'string',
            name: 'string',
            email: 'string',
            metadata: {
              createdAt: 'timestamp',
              updatedAt: 'timestamp'
            }
          }
        },
        migration: 'Add metadata object wrapper to user response'
      },
      {
        type: 'parameter',
        description: 'Pagination parameters renamed',
        v1Params: {
          page: 'number',
          limit: 'number'
        },
        v2Params: {
          offset: 'number',
          limit: 'number'
        },
        migration: 'Replace "page" parameter with "offset" (offset = (page - 1) * limit)'
      },
      {
        type: 'status_code',
        description: 'Error response format updated',
        v1Error: {
          error: 'string',
          message: 'string'
        },
        v2Error: {
          success: false,
          error: {
            code: 'string',
            message: 'string',
            details: 'object'
          }
        },
        migration: 'Update error handling to use new error structure'
      }
    ],
    steps: [
      'Review breaking changes list',
      'Update response handling to support new formats',
      'Replace "page" pagination with "offset"',
      'Add error code handling',
      'Test with v2 API version',
      'Deploy and monitor'
    ],
    estimatedTime: '2-4 hours',
    riskLevel: 'medium'
  }
};

for (const [key, strategy] of Object.entries(migrationStrategies)) {
  const [from, to] = key.split('To').map(v => v.toLowerCase());
  apiVersionManager.registerMigrationStrategy(from, to, strategy);
}

const generateMigrationGuide = (fromVersion, toVersion) => {
  const strategy = apiVersionManager.getMigrationStrategy(fromVersion, toVersion);

  if (!strategy) {
    return {
      success: false,
      error: {
        code: 'MIGRATION_NOT_AVAILABLE',
        message: `Migration from ${fromVersion} to ${toVersion} is not available`
      }
    };
  }

  return {
    success: true,
    migration: {
      from: strategy.from,
      to: strategy.to,
      breakingChanges: strategy.strategy.breakingChanges,
      steps: strategy.strategy.steps,
      estimatedTime: strategy.strategy.estimatedTime,
      riskLevel: strategy.strategy.riskLevel,
      resources: [
        {
          title: 'API Documentation',
          url: `/api-docs/${toVersion}`
        },
        {
          title: 'Migration Tutorial',
          url: `/api-docs/migration/${fromVersion}-to-${toVersion}`
        },
        {
          title: 'Changelog',
          url: '/api-docs/changelog'
        }
      ],
      support: {
        email: 'api-support@hjtpx.com',
        slack: '#api-migration'
      }
    }
  };
};

const applyMigrationTransform = (data, fromVersion, toVersion, endpoint) => {
  const strategy = apiVersionManager.getMigrationStrategy(fromVersion, toVersion);

  if (!strategy) {
    return data;
  }

  try {
    if (endpoint === 'users' || endpoint === 'user') {
      if (data.user && !data.user.metadata) {
        data.user.metadata = {
          createdAt: data.user.createdAt || new Date().toISOString(),
          updatedAt: data.user.updatedAt || new Date().toISOString()
        };
      }
    }

    if (data.pagination && data.pagination.page !== undefined) {
      data.pagination.offset = (data.pagination.page - 1) * (data.pagination.limit || 10);
      delete data.pagination.page;
    }

    return data;
  } catch (error) {
    console.error('Migration transform error:', error);
    return data;
  }
};

module.exports = {
  migrationStrategies,
  generateMigrationGuide,
  applyMigrationTransform
};
