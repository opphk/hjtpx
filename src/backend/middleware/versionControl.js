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
    ]
  },
  v2: {
    version: 'v2',
    status: 'stable',
    deprecated: false,
    deprecationDate: null,
    sunsetDate: null,
    migrationGuide: null,
    routes: require('../routes/v2'),
    breakingChanges: []
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

  if (negotiationDetails.requestedVersion && negotiationDetails.requestedVersion !== version) {
    res.setHeader('X-API-Version-Negotiated', 'true');
    res.setHeader('X-API-Original-Version', negotiationDetails.requestedVersion);
  }

  next();
};

const deprecationWarning = (req, res, next) => {
  const versionInfo = req.apiVersionInfo;

  if (versionInfo.deprecated) {
    const warningMessage = `API ${versionInfo.version} is deprecated. Please upgrade to the latest version.`;
    res.setHeader('Warning', `299 - "${warningMessage}"`);

    if (versionInfo.deprecationDate) {
      res.setHeader('X-API-Deprecation-Date', versionInfo.deprecationDate);
    }

    if (versionInfo.sunsetDate) {
      res.setHeader('X-API-Sunset-Date', versionInfo.sunsetDate);
    }

    if (versionInfo.migrationGuide) {
      res.setHeader('X-API-Migration-Guide', versionInfo.migrationGuide);
    }

    if (versionInfo.breakingChanges && versionInfo.breakingChanges.length > 0) {
      res.setHeader('X-API-Breaking-Changes', versionInfo.breakingChanges.length);
    }

    if (req.apiVersionInfo.sunsetDate) {
      const sunsetDate = new Date(req.apiVersionInfo.sunsetDate);
      const currentDate = new Date();
      const daysUntilSunset = Math.ceil((sunsetDate - currentDate) / (1000 * 60 * 60 * 24));

      if (daysUntilSunset > 0) {
        res.setHeader('X-API-Days-Until-Sunset', daysUntilSunset);

        if (daysUntilSunset <= 30) {
          res.setHeader(
            'Warning',
            `299 - "API ${versionInfo.version} will be sunset in ${daysUntilSunset} days. Urgent upgrade required."`
          );
        }
      } else {
        res.setHeader(
          'Warning',
          `299 - "API ${versionInfo.version} has been sunset. Requests will fail."`
        );
      }
    }

    const deprecationInfo = {
      deprecated: true,
      message: warningMessage,
      currentVersion: versionInfo.version,
      latestVersion: LATEST_STABLE_VERSION,
      deprecationDate: versionInfo.deprecationDate,
      sunsetDate: versionInfo.sunsetDate,
      migrationGuide: versionInfo.migrationGuide,
      breakingChanges: versionInfo.breakingChanges
    };

    req.deprecationInfo = deprecationInfo;

    const originalJson = res.json.bind(res);
    res.json = body => {
      if (body && typeof body === 'object') {
        body.deprecation = deprecationInfo;
      }
      return originalJson(body);
    };
  } else if (
    req.versionNegotiation?.requestedVersion &&
    req.versionNegotiation?.requestedVersion !== req.apiVersion
  ) {
    const upgradeMessage = `Version ${req.versionNegotiation.requestedVersion} not available. Using ${req.apiVersion}.`;
    res.setHeader('X-API-Version-Upgrade', upgradeMessage);
  }

  next();
};

const versionRouter = (req, res, next) => {
  const versionInfo = req.apiVersionInfo;
  if (versionInfo && versionInfo.routes) {
    return versionInfo.routes(req, res, next);
  }
  res.notFound(`API version ${req.apiVersion} not found`);
};

module.exports = {
  versionNegotiator,
  deprecationWarning,
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
