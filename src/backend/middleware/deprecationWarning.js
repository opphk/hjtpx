const { VERSIONS, LATEST_STABLE_VERSION } = require('./apiVersionNegotiation');

const deprecationWarning = (req, res, next) => {
  const versionInfo = req.apiVersionInfo;
  const version = req.apiVersion;

  if (!versionInfo) {
    return next();
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
      estimatedMigrationTime: estimateMigrationTime(versionInfo.breakingChanges?.length || 0)
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

const getDeprecationInfo = version => {
  const versionInfo = VERSIONS[version];

  if (!versionInfo) {
    return null;
  }

  if (!versionInfo.deprecated) {
    return null;
  }

  return {
    deprecated: true,
    currentVersion: version,
    latestVersion: LATEST_STABLE_VERSION,
    deprecationDate: versionInfo.deprecationDate,
    sunsetDate: versionInfo.sunsetDate,
    migrationGuide: versionInfo.migrationGuide,
    breakingChanges: versionInfo.breakingChanges || [],
    featuresEnabled: versionInfo.features || [],
    migrationSteps: generateMigrationSteps(version),
    estimatedMigrationTime: estimateMigrationTime(versionInfo.breakingChanges?.length || 0)
  };
};

const isVersionDeprecated = version => {
  const versionInfo = VERSIONS[version];
  return versionInfo ? versionInfo.deprecated : false;
};

const getSunsetDate = version => {
  const versionInfo = VERSIONS[version];
  return versionInfo ? versionInfo.sunsetDate : null;
};

const getDaysUntilSunset = version => {
  const sunsetDate = getSunsetDate(version);
  if (!sunsetDate) return null;

  const sunsetDateObj = new Date(sunsetDate);
  const currentDate = new Date();
  return Math.ceil((sunsetDateObj - currentDate) / (1000 * 60 * 60 * 24));
};

module.exports = {
  deprecationWarning,
  getDeprecationInfo,
  isVersionDeprecated,
  getSunsetDate,
  getDaysUntilSunset,
  generateMigrationSteps,
  estimateMigrationTime
};
