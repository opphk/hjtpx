const crypto = require('crypto');

class APIVersionManager {
  constructor() {
    this.versions = new Map();
    this.deprecationSchedule = new Map();
    this.migrationStrategies = new Map();
    this.registerDefaultVersions();
  }

  registerDefaultVersions() {
    this.registerVersion('v1', {
      status: 'stable',
      releaseDate: '2024-01-01',
      sunsetDate: null,
      isDeprecated: false
    });

    this.registerVersion('v2', {
      status: 'beta',
      releaseDate: '2025-06-01',
      sunsetDate: null,
      isDeprecated: false
    });
  }

  registerVersion(version, config) {
    this.versions.set(version, {
      ...config,
      registeredAt: new Date().toISOString()
    });
  }

  getVersion(version) {
    return this.versions.get(version);
  }

  getAllVersions() {
    return Array.from(this.versions.entries()).map(([version, config]) => ({
      version,
      ...config
    }));
  }

  getSupportedVersions() {
    return this.getAllVersions().filter(v => !v.isDeprecated);
  }

  deprecateVersion(version, sunsetDate) {
    const v = this.versions.get(version);
    if (v) {
      v.isDeprecated = true;
      v.sunsetDate = sunsetDate;
      this.deprecationSchedule.set(version, {
        version,
        sunsetDate,
        deprecatedAt: new Date().toISOString()
      });
    }
  }

  registerMigrationStrategy(fromVersion, toVersion, strategy) {
    const key = `${fromVersion}->${toVersion}`;
    this.migrationStrategies.set(key, {
      from: fromVersion,
      to: toVersion,
      strategy,
      registeredAt: new Date().toISOString()
    });
  }

  getMigrationStrategy(fromVersion, toVersion) {
    const key = `${fromVersion}->${toVersion}`;
    return this.migrationStrategies.get(key);
  }

  negotiateVersion(acceptHeader, queryVersion) {
    const requestedVersion = queryVersion || this.parseAcceptHeader(acceptHeader);

    if (!requestedVersion) {
      return {
        version: 'v1',
        negotiated: false,
        reason: 'No version specified, defaulting to v1'
      };
    }

    if (this.versions.has(requestedVersion)) {
      const versionInfo = this.versions.get(requestedVersion);
      return {
        version: requestedVersion,
        negotiated: true,
        status: versionInfo.status,
        isDeprecated: versionInfo.isDeprecated,
        sunsetDate: versionInfo.sunsetDate
      };
    }

    const compatibleVersion = this.findCompatibleVersion(requestedVersion);
    return {
      version: compatibleVersion,
      negotiated: true,
      reason: `Version ${requestedVersion} not found, using compatible version ${compatibleVersion}`
    };
  }

  parseAcceptHeader(acceptHeader) {
    if (!acceptHeader) return null;

    const versionPattern = /api-version\s*=\s*"?v?\d+"?"/i;
    const match = acceptHeader.match(versionPattern);

    if (match) {
      let version = match[0].split('=')[1].replace(/"/g, '').toLowerCase();
      if (!version.startsWith('v') && /^\d+$/.test(version)) {
        version = 'v' + version;
      }
      return version;
    }

    return null;
  }

  findCompatibleVersion(requestedVersion) {
    const requestedNum = parseInt(requestedVersion.replace(/\D/g, ''));
    const availableVersions = Array.from(this.versions.keys())
      .map(v => parseInt(v.replace(/\D/g, '')))
      .filter(n => !isNaN(n))
      .sort((a, b) => b - a);

    for (const versionNum of availableVersions) {
      if (versionNum <= requestedNum) {
        return `v${versionNum}`;
      }
    }

    return 'v1';
  }

  getDeprecationWarnings(version) {
    const versionInfo = this.versions.get(version);
    if (!versionInfo || !versionInfo.isDeprecated) {
      return null;
    }

    const warnings = [];
    const sunsetDate = new Date(versionInfo.sunsetDate);
    const now = new Date();
    const daysUntilSunset = Math.ceil((sunsetDate - now) / (1000 * 60 * 60 * 24));

    if (daysUntilSunset > 0) {
      warnings.push({
        type: 'deprecation',
        message: `API version ${version} is deprecated and will be sunset on ${versionInfo.sunsetDate}`,
        daysRemaining: daysUntilSunset,
        migrationUrl: `/api-docs/migration/${version}`
      });
    } else {
      warnings.push({
        type: 'sunset',
        message: `API version ${version} has reached its sunset date and may not be available`,
        actionRequired: 'Please migrate to a supported version immediately'
      });
    }

    return warnings;
  }

  generateVersionReport() {
    return {
      versions: this.getAllVersions(),
      supportedVersions: this.getSupportedVersions(),
      deprecatedVersions: this.getAllVersions().filter(v => v.isDeprecated),
      migrationStrategies: Array.from(this.migrationStrategies.entries()).map(([key, value]) => ({
        migrationPath: key,
        ...value
      })),
      generatedAt: new Date().toISOString()
    };
  }
}

const apiVersionManager = new APIVersionManager();

module.exports = apiVersionManager;
