const VERSIONS = {
  v1: {
    version: 'v1',
    status: 'stable',
    deprecated: true,
    deprecationDate: '2026-01-01',
    sunsetDate: '2026-07-01',
    migrationGuide: '/docs/v1-migration-guide.md',
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

const apiVersionNegotiator = (req, res, next) => {
  let version = null;
  let negotiationMethod = null;

  const urlMatch = req.path.match(/^\/api\/(v\d+)/);
  if (urlMatch && SUPPORTED_VERSIONS.includes(urlMatch[1])) {
    version = urlMatch[1];
    negotiationMethod = 'url';
  }

  if (!version) {
    const acceptVersion = req.headers['accept-version'];
    if (acceptVersion && SUPPORTED_VERSIONS.includes(acceptVersion)) {
      version = acceptVersion;
      negotiationMethod = 'accept-version-header';
    }
  }

  if (!version) {
    const acceptHeader = req.headers.accept;
    if (acceptHeader) {
      const acceptMatch = acceptHeader.match(/application\/vnd\.hjtpx\.(v\d+)\+json/);
      if (acceptMatch && SUPPORTED_VERSIONS.includes(acceptMatch[1])) {
        version = acceptMatch[1];
        negotiationMethod = 'accept-header';
      }
    }
  }

  if (!version) {
    const customHeader = req.headers['x-api-version'];
    if (customHeader && SUPPORTED_VERSIONS.includes(customHeader)) {
      version = customHeader;
      negotiationMethod = 'custom-header';
    }
  }

  if (!version) {
    const preferHeader = req.headers.prefer;
    if (preferHeader) {
      const preferMatch = preferHeader.match(/version=(v\d+)/);
      if (preferMatch && SUPPORTED_VERSIONS.includes(preferMatch[1])) {
        version = preferMatch[1];
        negotiationMethod = 'prefer-header';
      }
    }
  }

  if (!version) {
    version = DEFAULT_VERSION;
    negotiationMethod = 'default';
  }

  const versionInfo = VERSIONS[version] || VERSIONS[DEFAULT_VERSION];
  const isNegotiated =
    negotiationMethod && negotiationMethod !== 'url' && negotiationMethod !== 'default';
  const isUpgrade = negotiationMethod === 'default' && req.headers['accept-version'];

  req.apiVersion = version;
  req.apiVersionInfo = versionInfo;
  req.versionNegotiation = {
    requestedVersion: version,
    resolvedVersion: version,
    negotiationMethod: negotiationMethod,
    isLatest: version === LATEST_STABLE_VERSION,
    isDeprecated: versionInfo.deprecated || false
  };

  res.setHeader('X-API-Version', version);
  res.setHeader('X-API-Version-Status', versionInfo.status || 'unknown');
  res.setHeader('X-API-Supported-Versions', SUPPORTED_VERSIONS.join(', '));
  res.setHeader('X-API-Latest-Version', LATEST_STABLE_VERSION);

  if (isNegotiated) {
    res.setHeader('X-API-Version-Negotiated', 'true');
  }

  if (isUpgrade) {
    res.setHeader('X-API-Version-Negotiated', 'true');
    res.setHeader(
      'X-API-Version-Upgrade',
      `Version ${req.headers['accept-version']} not available. Using ${version}.`
    );
  }

  next();
};

const isVersionSupported = version => {
  return SUPPORTED_VERSIONS.includes(version);
};

const getVersionInfo = version => {
  return VERSIONS[version] || null;
};

const getSupportedVersions = () => {
  return [...SUPPORTED_VERSIONS];
};

const getDefaultVersion = () => {
  return DEFAULT_VERSION;
};

const getLatestVersion = () => {
  return LATEST_STABLE_VERSION;
};

module.exports = {
  apiVersionNegotiator,
  VERSIONS,
  DEFAULT_VERSION,
  SUPPORTED_VERSIONS,
  LATEST_STABLE_VERSION,
  isVersionSupported,
  getVersionInfo,
  getSupportedVersions,
  getDefaultVersion,
  getLatestVersion
};
