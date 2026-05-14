const apiVersionManager = require('./apiVersionManager');

const versionNegotiationMiddleware = (req, res, next) => {
  const acceptHeader = req.headers['accept'];
  const queryVersion = req.query.api_version || req.query.version;

  const negotiation = apiVersionManager.negotiateVersion(acceptHeader, queryVersion);

  req.apiVersion = negotiation.version;
  req.versionInfo = negotiation;

  res.setHeader('X-API-Version', negotiation.version);

  if (negotiation.isDeprecated) {
    const warnings = apiVersionManager.getDeprecationWarnings(negotiation.version);
    if (warnings && warnings.length > 0) {
      res.setHeader('X-API-Deprecation-Warning', warnings[0].message);
      res.setHeader('X-API-Sunset-Date', warnings[0].sunsetDate || negotiation.sunsetDate);
    }
  }

  if (!negotiation.negotiated && negotiation.reason) {
    res.setHeader('X-API-Version-Note', negotiation.reason);
  }

  next();
};

const versionEnforcementMiddleware = (requiredVersion) => {
  return (req, res, next) => {
    const requestedVersion = req.apiVersion || req.query.api_version;

    if (!requestedVersion) {
      return res.status(400).json({
        success: false,
        error: {
          code: 'VERSION_REQUIRED',
          message: `API version is required. Please specify version ${requiredVersion} or higher.`,
          supportedVersions: ['v1', 'v2']
        }
      });
    }

    const requestedNum = parseInt(requestedVersion.replace(/\D/g, ''));
    const requiredNum = parseInt(requiredVersion.replace(/\D/g, ''));

    if (requestedNum < requiredNum) {
      const migrationStrategy = apiVersionManager.getMigrationStrategy(requestedVersion, requiredVersion);
      return res.status(400).json({
        success: false,
        error: {
          code: 'VERSION_NOT_SUPPORTED',
          message: `API version ${requestedVersion} is no longer supported. Please upgrade to ${requiredVersion} or higher.`,
          migration: migrationStrategy ? {
            from: requestedVersion,
            to: requiredVersion,
            available: true
          } : {
            from: requestedVersion,
            to: requiredVersion,
            available: false,
            contactSupport: 'Contact support for migration assistance'
          }
        }
      });
    }

    next();
  };
};

const deprecatedVersionMiddleware = (req, res, next) => {
  if (req.versionInfo && req.versionInfo.isDeprecated) {
    const warnings = apiVersionManager.getDeprecationWarnings(req.apiVersion);

    if (warnings && warnings.length > 0) {
      res.setHeader('Warning', `299 - "${warnings[0].message}"`);

      if (warnings[0].daysRemaining && warnings[0].daysRemaining <= 30) {
        console.warn(`[DEPRECATION WARNING] Version ${req.apiVersion}: ${warnings[0].message}`);
      }
    }
  }
  next();
};

module.exports = {
  versionNegotiationMiddleware,
  versionEnforcementMiddleware,
  deprecatedVersionMiddleware,
  apiVersionManager
};
