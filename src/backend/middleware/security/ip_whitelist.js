class IPControl {
  constructor(options = {}) {
    this.whitelist = new Set();
    this.blacklist = new Set();
    this.cidrRanges = { whitelist: [], blacklist: [] };
    this.defaultAction = options.defaultAction || 'allow';
  }

  addWhitelist(ip) {
    if (this.isCIDR(ip)) {
      this.cidrRanges.whitelist.push(this.parseCIDR(ip));
    } else {
      this.whitelist.add(ip);
    }
  }

  addBlacklist(ip) {
    if (this.isCIDR(ip)) {
      this.cidrRanges.blacklist.push(this.parseCIDR(ip));
    } else {
      this.blacklist.add(ip);
    }
  }

  removeWhitelist(ip) {
    this.whitelist.delete(ip);
    this.cidrRanges.whitelist = this.cidrRanges.whitelist.filter(range => range.ip !== ip);
  }

  removeBlacklist(ip) {
    this.blacklist.delete(ip);
    this.cidrRanges.blacklist = this.cidrRanges.blacklist.filter(range => range.ip !== ip);
  }

  isAllowed(ip) {
    if (!this.isValidIP(ip)) {
      return false;
    }

    if (this.blacklist.has(ip) || this.isInCIDRRange(ip, this.cidrRanges.blacklist)) {
      return false;
    }

    if (this.whitelist.has(ip) || this.isInCIDRRange(ip, this.cidrRanges.whitelist)) {
      return true;
    }

    return this.defaultAction === 'allow';
  }

  setDefaultAction(action) {
    if (['allow', 'deny'].includes(action)) {
      this.defaultAction = action;
    }
  }

  isCIDR(ip) {
    return ip.includes('/');
  }

  parseCIDR(cidr) {
    const [ip, bits] = cidr.split('/');
    return {
      ip,
      bits: parseInt(bits),
      mask: parseInt('1'.repeat(bits) + '0'.repeat(32 - bits), 2)
    };
  }

  ipToNumber(ip) {
    return ip.split('.').reduce((acc, octet) => (acc << 8) + parseInt(octet), 0);
  }

  isInCIDRRange(ip, ranges) {
    const ipNum = this.ipToNumber(ip);
    return ranges.some(range => {
      const rangeNum = this.ipToNumber(range.ip);
      return (ipNum & range.mask) === (rangeNum & range.mask);
    });
  }

  isValidIP(ip) {
    const ipv4Regex = /^(\d{1,3}\.){3}\d{1,3}$/;
    const ipv6Regex = /^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}$/;

    if (ipv4Regex.test(ip)) {
      return ip.split('.').every(octet => parseInt(octet) <= 255);
    }

    return ipv6Regex.test(ip) || ip === '::1' || ip === '::';
  }

  clearAll() {
    this.whitelist.clear();
    this.blacklist.clear();
    this.cidrRanges = { whitelist: [], blacklist: [] };
  }

  getStatus(ip) {
    return {
      ip,
      allowed: this.isAllowed(ip),
      inWhitelist: this.whitelist.has(ip) || this.isInCIDRRange(ip, this.cidrRanges.whitelist),
      inBlacklist: this.blacklist.has(ip) || this.isInCIDRRange(ip, this.cidrRanges.blacklist),
      defaultAction: this.defaultAction
    };
  }
}

const ipControlMiddleware = async (req, res, next) => {
  const ip =
    req.ip || req.connection.remoteAddress || req.headers['x-forwarded-for']?.split(',')[0]?.trim();

  const ipControl = global.ipControl || new IPControl();

  if (!ipControl.isAllowed(ip)) {
    return res.status(403).json({
      error: 'access_denied',
      message: 'IP address not allowed'
    });
  }

  req.clientIP = ip;
  next();
};

module.exports = { IPControl, ipControlMiddleware };
