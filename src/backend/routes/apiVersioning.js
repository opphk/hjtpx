const express = require('express');
const router = express.Router();

const API_VERSIONS = {
  'v1': {
    version: 'v1',
    status: 'current',
    deprecationDate: null,
    sunsetDate: null,
    sunsetDuration: null,
    migrations: [],
    breakingChanges: []
  },
  'v2': {
    version: 'v2',
    status: 'current',
    deprecationDate: null,
    sunsetDate: null,
    sunsetDuration: null,
    migrations: [
      {
        from: 'v1',
        description: 'Migrating from v1 to v2',
        steps: [
          'Authentication mechanism updated to JWT with refresh tokens',
          'Response format standardized to include metadata field',
          'Pagination parameters changed from page/pageSize to offset/limit',
          'Date format standardized to ISO 8601'
        ]
      }
    ],
    breakingChanges: [
      {
        endpoint: '/api/v1/users',
        change: 'Response structure modified',
        migration: 'Use new response structure with data and meta fields'
      }
    ]
  }
};

const DEPRECATION_WARNINGS = {
  'v1': {
    deprecationDate: new Date('2024-06-01'),
    sunsetDate: new Date('2025-12-31'),
    warning: 'API v1 is deprecated and will be sunset on 2025-12-31. Please migrate to v2.',
    migrationGuide: '/api-docs/v1-to-v2-migration'
  }
};

router.get('/versions', (req, res) => {
  const requestedVersion = req.headers['accept-version'] || req.query.version;
  const currentVersion = getCurrentVersion(requestedVersion);

  const versions = Object.entries(API_VERSIONS).map(([version, config]) => ({
    version,
    status: config.status,
    deprecationDate: config.deprecationDate,
    sunsetDate: config.sunsetDate,
    isCurrent: version === currentVersion,
    isDeprecated: config.status === 'deprecated',
    breakingChanges: config.breakingChanges?.length || 0
  }));

  const response = {
    versions,
    current: currentVersion,
    supported: Object.keys(API_VERSIONS),
    default: 'v2',
    documentation: '/api-docs'
  };

  if (DEPRECATION_WARNINGS[currentVersion]) {
    response.deprecationWarning = DEPRECATION_WARNINGS[currentVersion];
  }

  res.json({
    success: true,
    data: response
  });
});

router.get('/negotiate', (req, res) => {
  const clientVersions = req.headers['accept-version']?.split(',').map(v => v.trim()) || ['v1'];
  const serverVersions = Object.keys(API_VERSIONS);

  const negotiatedVersion = negotiateVersion(clientVersions, serverVersions);

  if (!negotiatedVersion) {
    return res.status(406).json({
      success: false,
      error: {
        code: 'VERSION_NOT_SUPPORTED',
        message: 'No compatible API version found',
        supportedVersions: serverVersions,
        requestedVersions: clientVersions
      }
    });
  }

  const config = API_VERSIONS[negotiatedVersion];
  const isDeprecated = config.status === 'deprecated';

  const response = {
    negotiatedVersion,
    status: config.status,
    isDeprecated,
    migration: isDeprecated ? DEPRECATION_WARNINGS[negotiatedVersion] : null,
    responseHeaders: {
      'API-Version': negotiatedVersion,
      'API-Status': config.status
    }
  };

  if (isDeprecated) {
    res.setHeader('Deprecation', 'true');
    res.setHeader('Sunset', config.sunsetDate?.toUTCString() || '');
    res.setHeader('Link', `<${DEPRECATION_WARNINGS[negotiatedVersion].migrationGuide}>; rel="deprecation"`);
  }

  res.json({
    success: true,
    data: response
  });
});

router.get('/v1-to-v2-migration', (req, res) => {
  const migrationGuide = {
    title: 'API v1 to v2 Migration Guide',
    version: {
      from: 'v1',
      to: 'v2'
    },
    overview: 'This guide will help you migrate from API v1 to v2 with minimal disruption.',
    timeline: {
      v1Release: '2023-01-01',
      v2Release: '2024-01-01',
      v1Deprecation: '2024-06-01',
      v1Sunset: '2025-12-31'
    },
    breakingChanges: [
      {
        category: 'Authentication',
        changes: [
          {
            endpoint: '/api/v1/auth/login',
            v1Format: {
              method: 'POST',
              body: { email: 'string', password: 'string' },
              response: { token: 'string', user: 'object' }
            },
            v2Format: {
              method: 'POST',
              body: { email: 'string', password: 'string' },
              response: {
                accessToken: 'string',
                refreshToken: 'string',
                expiresIn: 'number',
                user: 'object'
              }
            },
            migration: 'Update your authentication logic to handle accessToken and refreshToken separately. Implement token refresh mechanism.'
          },
          {
            endpoint: '/api/v1/auth/refresh',
            change: 'New endpoint in v2',
            v2Format: {
              method: 'POST',
              body: { refreshToken: 'string' },
              response: { accessToken: 'string', expiresIn: 'number' }
            },
            migration: 'Implement token refresh using the new /auth/refresh endpoint before token expiration.'
          }
        ]
      },
      {
        category: 'Response Format',
        changes: [
          {
            endpoint: 'All endpoints',
            v1Format: {
              response: 'Direct object or array'
            },
            v2Format: {
              response: {
                success: 'boolean',
                data: 'object | array',
                meta: {
                  timestamp: 'string',
                  version: 'string'
                }
              }
            },
            migration: 'Wrap all responses in the new response structure. Check success field before processing data.'
          }
        ]
      },
      {
        category: 'Pagination',
        changes: [
          {
            endpoint: '/api/v1/users',
            v1Format: {
              queryParams: { page: 'number', pageSize: 'number' },
              response: {
                users: 'array',
                totalCount: 'number',
                currentPage: 'number',
                totalPages: 'number'
              }
            },
            v2Format: {
              queryParams: { offset: 'number', limit: 'number' },
              response: {
                data: 'array',
                meta: {
                  total: 'number',
                  offset: 'number',
                  limit: 'number',
                  hasMore: 'boolean'
                }
              }
            },
            migration: 'Replace page/pageSize with offset/limit. Update response parsing logic to access data.data and data.meta.'
          }
        ]
      },
      {
        category: 'Date Format',
        changes: [
          {
            v1Format: 'YYYY-MM-DD HH:mm:ss',
            v2Format: 'ISO 8601 (YYYY-MM-DDTHH:mm:ss.sssZ)',
            migration: 'Update date parsing logic to handle ISO 8601 format. Use standard Date parsing libraries.'
          }
        ]
      },
      {
        category: 'Error Handling',
        changes: [
          {
            endpoint: 'All error responses',
            v1Format: {
              error: 'string',
              message: 'string',
              code: 'string'
            },
            v2Format: {
              success: false,
              error: {
                code: 'string',
                message: 'string',
                details: 'object | null',
                requestId: 'string',
                timestamp: 'string'
              }
            },
            migration: 'Update error handling to check success field. Access error details from error object.'
          }
        ]
      }
    ],
    codeExamples: {
      authentication: {
        v1: `
const response = await fetch('/api/v1/auth/login', {
  method: 'POST',
  body: JSON.stringify({ email, password })
});
const { token, user } = await response.json();
localStorage.setItem('token', token);
        `,
        v2: `
const response = await fetch('/api/v2/auth/login', {
  method: 'POST',
  body: JSON.stringify({ email, password })
});
const { data } = await response.json();
const { accessToken, refreshToken, user } = data;
localStorage.setItem('accessToken', accessToken);
localStorage.setItem('refreshToken', refreshToken);

// Implement token refresh
async function refreshAccessToken() {
  const refreshToken = localStorage.getItem('refreshToken');
  const response = await fetch('/api/v2/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refreshToken })
  });
  const { data } = await response.json();
  localStorage.setItem('accessToken', data.accessToken);
  return data.accessToken;
}
        `
      },
      pagination: {
        v1: `
const response = await fetch('/api/v1/users?page=1&pageSize=10');
const { users, totalCount, totalPages } = await response.json();
        `,
        v2: `
const response = await fetch('/api/v2/users?offset=0&limit=10');
const { data, meta } = await response.json();
const { data: users, meta: pagination } = data;
        `
      }
    },
    testingChecklist: [
      'Test authentication flow end-to-end',
      'Verify token refresh mechanism',
      'Check all response parsing logic',
      'Test pagination with new parameters',
      'Verify date parsing across all features',
      'Test error handling with new format',
      'Test all CRUD operations',
      'Verify WebSocket connections if applicable'
    ],
    rollbackPlan: {
      description: 'If migration issues arise, you can temporarily revert to v1 by specifying version in request headers.',
      header: 'Accept-Version: v1',
      note: 'v1 will remain available until 2025-12-31. Please resolve issues before this date.'
    },
    support: {
      documentation: '/api-docs/v2',
      migrationGuide: '/api-docs/v1-to-v2-migration',
      contact: 'support@hjtpx.com'
    }
  };

  res.json({
    success: true,
    data: migrationGuide
  });
});

function getCurrentVersion(requestedVersion) {
  if (!requestedVersion) {
    return 'v2';
  }

  if (API_VERSIONS[requestedVersion]) {
    return requestedVersion;
  }

  const versionMatch = requestedVersion.match(/^v(\d+)/);
  if (versionMatch) {
    const requestedNum = parseInt(versionMatch[1]);
    const availableVersions = Object.keys(API_VERSIONS).map(v => parseInt(v.substring(1)));
    const closestLower = availableVersions.filter(v => v <= requestedNum).sort((a, b) => b - a)[0];

    if (closestLower) {
      return `v${closestLower}`;
    }
  }

  return 'v2';
}

function negotiateVersion(clientVersions, serverVersions) {
  const normalizedClientVersions = clientVersions.map(v => {
    const match = v.match(/^v?(\d+)/);
    return match ? `v${parseInt(match[1])}` : v;
  });

  for (const clientVersion of normalizedClientVersions) {
    if (clientVersion.startsWith('~')) {
      const baseVersion = clientVersion.substring(1);
      if (serverVersions.includes(baseVersion)) {
        return baseVersion;
      }
    }

    if (clientVersion.startsWith('^')) {
      const majorVersion = parseInt(clientVersion.substring(1).substring(1));
      const compatible = serverVersions.find(v => parseInt(v.substring(1)) === majorVersion);
      if (compatible) {
        return compatible;
      }
    }

    if (serverVersions.includes(clientVersion)) {
      return clientVersion;
    }
  }

  return serverVersions.find(v => API_VERSIONS[v].status === 'current') || serverVersions[0];
}

function checkDeprecation(req, res, next) {
  const version = extractVersion(req.path);

  if (DEPRECATION_WARNINGS[version]) {
    const warning = DEPRECATION_WARNINGS[version];
    const now = new Date();

    if (now >= warning.deprecationDate) {
      res.setHeader('Deprecation', 'true');
      res.setHeader('Sunset', warning.sunsetDate.toUTCString());
      res.setHeader('Warning', `299 "${warning.warning}"`);
      res.setHeader('Link', `<${warning.migrationGuide}>; rel="deprecation"; type="text/html"`);

      if (now >= warning.sunsetDate) {
        return res.status(410).json({
          success: false,
          error: {
            code: 'API_VERSION_SUNSET',
            message: `API ${version} has been sunset as of ${warning.sunsetDate.toISOString()}. Please migrate to a supported version.`,
            migrationGuide: warning.migrationGuide,
            supportedVersions: Object.keys(API_VERSIONS).filter(v => API_VERSIONS[v].status !== 'deprecated')
          }
        });
      }
    }
  }

  next();
}

function extractVersion(path) {
  const versionMatch = path.match(/^\/api\/(v\d+)/);
  return versionMatch ? versionMatch[1] : 'v1';
}

function versionHeader(req, res, next) {
  const version = extractVersion(req.path);
  res.setHeader('API-Version', version);
  res.setHeader('API-Status', API_VERSIONS[version]?.status || 'unknown');
  next();
}

router.use(checkDeprecation);
router.use(versionHeader);

module.exports = router;
module.exports.API_VERSIONS = API_VERSIONS;
module.exports.DEPRECATION_WARNINGS = DEPRECATION_WARNINGS;
module.exports.checkDeprecation = checkDeprecation;
module.exports.versionHeader = versionHeader;
module.exports.getCurrentVersion = getCurrentVersion;
module.exports.negotiateVersion = negotiateVersion;
