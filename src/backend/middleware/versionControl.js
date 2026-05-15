const VERSIONS = {
  v1: {
    version: 'v1',
    status: 'stable',
    deprecated: true,
    deprecationDate: '2026-01-01',
    sunsetDate: '2026-07-01',
    migrationGuide: '/docs/v1-migration-guide.md',
    routes: require('../routes/v1'),
    breakingChanges: [
      'Removed legacy authentication endpoints',
      'Changed response format for user endpoints',
      'Removed deprecated fields'
    ],
    features: ['basic_auth', 'legacy_response_format', 'no_pagination']
  },
  v2: {
    version: 'v2',
    status: 'stable',
    deprecated: false,
    deprecationDate: null,
    sunsetDate: null,
    migrationGuide: null,
    routes: require('../routes/v2'),
    breakingChanges: [],
    features: [
      'jwt_auth',
      'enhanced_response_format',
      'pagination',
      'rate_limiting',
      'advanced_filtering'
    ]
  }
};

const DEFAULT_VERSION = 'v2';
const SUPPORTED_VERSIONS = Object.keys(VERSIONS);
const LATEST_STABLE_VERSION = 'v2';

const versionNegotiator = (req, res, next) => {
  let version = null;
  const negotiationDetails = {
    requestedVersion: null,
    resolvedVersion: null,
    negotiationMethod: null,
    isLatest: false,
    isDeprecated: false,
    alternatives: []
  };

  const urlMatch = req.path.match(/^\/api\/(v\d+)/);
  if (urlMatch && SUPPORTED_VERSIONS.includes(urlMatch[1])) {
    version = urlMatch[1];
    negotiationDetails.negotiationMethod = 'url';
    negotiationDetails.requestedVersion = version;
  }

  if (!version) {
    const acceptVersion = req.headers['accept-version'];
    if (acceptVersion && SUPPORTED_VERSIONS.includes(acceptVersion)) {
      version = acceptVersion;
      negotiationDetails.negotiationMethod = 'accept-version-header';
      negotiationDetails.requestedVersion = version;
    }
  }

  if (!version) {
    const acceptHeader = req.headers.accept;
    if (acceptHeader) {
      const acceptMatch = acceptHeader.match(/application\/vnd\.hjtpx\.(v\d+)\+json/);
      if (acceptMatch && SUPPORTED_VERSIONS.includes(acceptMatch[1])) {
        version = acceptMatch[1];
        negotiationDetails.negotiationMethod = 'accept-header';
        negotiationDetails.requestedVersion = version;
      }
    }
  }

  if (!version) {
    const customHeader = req.headers['x-api-version'];
    if (customHeader && SUPPORTED_VERSIONS.includes(customHeader)) {
      version = customHeader;
      negotiationDetails.negotiationMethod = 'custom-header';
      negotiationDetails.requestedVersion = version;
    }
  }

  if (!version) {
    const preferHeader = req.headers.prefer;
    if (preferHeader) {
      const preferMatch = preferHeader.match(/version=(v\d+)/);
      if (preferMatch && SUPPORTED_VERSIONS.includes(preferMatch[1])) {
        version = preferMatch[1];
        negotiationDetails.negotiationMethod = 'prefer-header';
        negotiationDetails.requestedVersion = version;
      }
    }
  }

  if (!version) {
    version = DEFAULT_VERSION;
    negotiationDetails.negotiationMethod = 'default';
    negotiationDetails.requestedVersion = null;
  }

  negotiationDetails.resolvedVersion = version;
  negotiationDetails.isLatest = version === LATEST_STABLE_VERSION;
  negotiationDetails.isDeprecated = VERSIONS[version]?.deprecated || false;

  negotiationDetails.alternatives = SUPPORTED_VERSIONS.filter(v => v !== version);

  req.apiVersion = version;
  req.apiVersionInfo = VERSIONS[version];
  req.versionNegotiation = negotiationDetails;

  res.setHeader('X-API-Version', req.apiVersion);
  res.setHeader('X-API-Version-Status', req.apiVersionInfo?.status || 'unknown');
  res.setHeader('X-API-Supported-Versions', SUPPORTED_VERSIONS.join(', '));
  res.setHeader('X-API-Latest-Version', LATEST_STABLE_VERSION);
  if (req.apiVersionInfo?.breakingChanges && req.apiVersionInfo.breakingChanges.length > 0) {
    res.setHeader('X-API-Breaking-Changes', req.apiVersionInfo.breakingChanges.length.toString());
  }

  if (negotiationDetails.negotiationMethod && negotiationDetails.negotiationMethod !== 'url') {
    res.setHeader('X-API-Version-Negotiated', 'true');
    if (negotiationDetails.requestedVersion) {
      res.setHeader('X-API-Original-Version', negotiationDetails.requestedVersion);
    }
  }

  if (
    !negotiationDetails.requestedVersion &&
    (req.headers['accept-version'] || req.headers['x-api-version'] || req.headers.prefer)
  ) {
    const requested =
      req.headers['accept-version'] ||
      req.headers['x-api-version'] ||
      (req.headers.prefer ? req.headers.prefer.match(/version=(v\d+)/)?.[1] : null);
    if (requested && !SUPPORTED_VERSIONS.includes(requested)) {
      res.setHeader('X-API-Version-Negotiated', 'true');
      res.setHeader(
        'X-API-Version-Upgrade',
        `Version ${requested} not available. Using ${version}.`
      );
    }
  }

  next();
};

const generateMigrationSteps = version => {
  const steps = {
    v1: [
      {
        step: 1,
        title: 'Update API Base URL',
        description: 'Change API base URL from /api/v1 to /api/v2',
        action: 'Replace /api/v1/ with /api/v2/ in all API calls'
      },
      {
        step: 2,
        title: 'Update Authentication',
        description: 'Migrate from basic auth to JWT tokens',
        action: 'Implement JWT-based authentication flow'
      },
      {
        step: 3,
        title: 'Update Response Format',
        description: 'Adapt to new enhanced response structure',
        action: 'Update response parsing logic to handle new format'
      },
      {
        step: 4,
        title: 'Implement Pagination',
        description: 'Add pagination support for list endpoints',
        action: 'Add page and limit query parameters'
      },
      {
        step: 5,
        title: 'Update Error Handling',
        description: 'Adapt to new error response format',
        action: 'Update error handling to use new error codes'
      }
    ]
  };

  return steps[version] || [];
};

const estimateMigrationTime = breakingChangesCount => {
  const baseTime = 2;
  const perChangeTime = 1;
  return `${baseTime + breakingChangesCount * perChangeTime} hours`;
};

const deprecationWarningMiddleware = (req, res, next) => {
  const versionInfo = req.apiVersionInfo;
  const version = req.apiVersion;

  if (!versionInfo) {
    next();
    return;
  }

  if (versionInfo.deprecated) {
    const deprecationDate = versionInfo.deprecationDate || 'unknown';
    const sunsetDate = versionInfo.sunsetDate || 'unknown';
    const migrationGuide = versionInfo.migrationGuide || '/docs/v1-migration-guide.md';

    res.setHeader('Deprecation', `API version ${version} is deprecated since ${deprecationDate}`);
    res.setHeader('X-API-Deprecation-Date', deprecationDate);
    res.setHeader('X-API-Sunset-Date', sunsetDate);
    res.setHeader('X-API-Migration-Guide', migrationGuide);

    const linkHeader = [
      `<${migrationGuide}>; rel="deprecation"`,
      `<${LATEST_STABLE_VERSION}>; rel="successor-version"`
    ];
    res.setHeader('Link', linkHeader.join(', '));

    if (versionInfo.sunsetDate) {
      const sunsetDateObj = new Date(versionInfo.sunsetDate);
      const currentDate = new Date();
      const daysUntilSunset = Math.ceil((sunsetDateObj - currentDate) / (1000 * 60 * 60 * 24));

      if (daysUntilSunset > 0) {
        res.setHeader('X-API-Days-Until-Sunset', daysUntilSunset.toString());

        if (daysUntilSunset <= 30) {
          res.setHeader(
            'Warning',
            `299 - "API ${version} will be sunset in ${daysUntilSunset} days. Urgent upgrade required."`
          );
        } else {
          res.setHeader(
            'Warning',
            `299 - "API ${version} is deprecated. Please upgrade to ${LATEST_STABLE_VERSION}."`
          );
        }
      } else {
        res.setHeader('Warning', `299 - "API ${version} has been sunset. Requests will fail."`);
      }
    } else {
      res.setHeader(
        'Warning',
        `299 - "API ${version} is deprecated. Please upgrade to ${LATEST_STABLE_VERSION}."`
      );
    }

    const deprecationInfo = {
      deprecated: true,
      currentVersion: version,
      latestVersion: LATEST_STABLE_VERSION,
      deprecationDate: deprecationDate,
      sunsetDate: sunsetDate,
      migrationGuide: migrationGuide,
      breakingChanges: versionInfo.breakingChanges || [],
      featuresEnabled: versionInfo.features || [],
      migrationSteps: generateMigrationSteps(version),
      estimatedMigrationTime: estimateMigrationTime(versionInfo.breakingChanges?.length || 0),
      message: `API version ${version} is deprecated. Please upgrade to ${LATEST_STABLE_VERSION}.`
    };

    req.deprecationInfo = deprecationInfo;

    const originalJson = res.json.bind(res);
    res.json = body => {
      if (body && typeof body === 'object') {
        body.deprecation = deprecationInfo;
      }
      return originalJson(body);
    };
  } else {
    const originalJson = res.json.bind(res);
    res.json = body => {
      if (body && typeof body === 'object') {
        body.features = {
          allAvailable: true,
          version: version,
          availableFeatures: versionInfo.features || []
        };
      }
      return originalJson(body);
    };
  }

  next();
};

const versionRouter = (req, res, next) => {
  const versionInfo = req.apiVersionInfo;
  if (versionInfo && versionInfo.routes) {
    return versionInfo.routes(req, res, next);
  }
  return res.notFound(`API version ${req.apiVersion} not found`);
};

module.exports = {
  versionNegotiator,
  deprecationWarning: deprecationWarningMiddleware,
  versionRouter,
  VERSIONS,
  DEFAULT_VERSION,
  SUPPORTED_VERSIONS,
  LATEST_STABLE_VERSION,
  getVersionInfo: version => VERSIONS[version] || null,
  isVersionSupported: version => SUPPORTED_VERSIONS.includes(version),
  getDeprecationStatus: version => {
    const info = VERSIONS[version];
    if (!info) return { supported: false, deprecated: null };
    return {
      supported: true,
      deprecated: info.deprecated,
      status: info.status,
      sunsetDate: info.sunsetDate,
      daysUntilSunset: info.sunsetDate
        ? Math.ceil((new Date(info.sunsetDate) - new Date()) / (1000 * 60 * 60 * 24))
        : null
    };
  },
  getMigrationInfo: version => {
    const info = VERSIONS[version];
    if (!info) return null;
    return {
      currentVersion: version,
      latestVersion: LATEST_STABLE_VERSION,
      isLatest: version === LATEST_STABLE_VERSION,
      breakingChanges: info.breakingChanges || [],
      migrationGuide: info.migrationGuide,
      estimatedMigrationTime: info.breakingChanges?.length
        ? `${info.breakingChanges.length * 2} hours`
        : null
    };
  }
};
