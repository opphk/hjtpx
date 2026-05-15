class GeoRestriction {
  constructor(options = {}) {
    this.allowedCountries = new Set(
      options.allowedCountries || ['CN', 'US', 'HK', 'TW', 'SG', 'JP', 'KR']
    );
    this.blockedCountries = new Set(options.blockedCountries || []);
    this.countryCache = new Map();
    this.enabled = options.enabled !== false;
  }

  isCountryAllowed(countryCode) {
    if (!this.enabled) {
      return true;
    }

    const code = countryCode?.toUpperCase();

    if (!code) {
      return false;
    }

    if (this.blockedCountries.has(code)) {
      return false;
    }

    if (this.allowedCountries.size === 0) {
      return true;
    }

    return this.allowedCountries.has(code);
  }

  async getCountryByIP(ip) {
    if (this.countryCache.has(ip)) {
      return this.countryCache.get(ip);
    }

    try {
      const countryCode = await this.lookupCountry(ip);
      this.countryCache.set(ip, countryCode);
      return countryCode;
    } catch (error) {
      console.error('GeoIP lookup failed:', error);
      return null;
    }
  }

  async lookupCountry(ip) {
    if (ip === '127.0.0.1' || ip === '::1' || ip === '::') {
      return 'LOCAL';
    }

    if (ip.startsWith('192.168.') || ip.startsWith('10.') || ip.startsWith('172.')) {
      return 'PRIVATE';
    }

    if (process.env.GEOIP_DATABASE) {
      const MaxMind = require('maxmind');
      const lookup = MaxMind.open(process.env.GEOIP_DATABASE);
      const result = lookup.get(ip);
      return result?.country?.iso_code || 'UNKNOWN';
    }

    const geoip = require('geoip-lite');
    const result = geoip.lookup(ip);
    return result?.country || 'UNKNOWN';
  }

  addAllowedCountry(countryCode) {
    this.allowedCountries.add(countryCode.toUpperCase());
  }

  removeAllowedCountry(countryCode) {
    this.allowedCountries.delete(countryCode.toUpperCase());
  }

  addBlockedCountry(countryCode) {
    this.blockedCountries.add(countryCode.toUpperCase());
  }

  removeBlockedCountry(countryCode) {
    this.blockedCountries.delete(countryCode.toUpperCase());
  }

  enable() {
    this.enabled = true;
  }

  disable() {
    this.enabled = false;
  }

  clearCache() {
    this.countryCache.clear();
  }

  getConfig() {
    return {
      enabled: this.enabled,
      allowedCountries: Array.from(this.allowedCountries),
      blockedCountries: Array.from(this.blockedCountries)
    };
  }
}

const geoRestrictionMiddleware = async (req, res, next) => {
  const geoRestriction = global.geoRestriction || new GeoRestriction();

  if (!geoRestriction.enabled) {
    return next();
  }

  const ip =
    req.ip || req.connection.remoteAddress || req.headers['x-forwarded-for']?.split(',')[0]?.trim();

  try {
    const countryCode = await geoRestriction.getCountryByIP(ip);

    if (!countryCode || !geoRestriction.isCountryAllowed(countryCode)) {
      return res.status(403).json({
        error: 'geo_restricted',
        message: 'Access from your region is not allowed'
      });
    }

    req.countryCode = countryCode;
    next();
  } catch (error) {
    console.error('Geo restriction error:', error);
    next();
  }
};

module.exports = { GeoRestriction, geoRestrictionMiddleware };
