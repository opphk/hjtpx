const crypto = require('crypto');

class APISignature {
  static generate(secret, params) {
    const sortedParams = this.sortParams(params);
    const queryString = this.buildQueryString(sortedParams);
    const timestamp = Date.now();
    const signaturePayload = `${queryString}&timestamp=${timestamp}`;

    const signature = crypto.createHmac('sha256', secret).update(signaturePayload).digest('hex');

    return {
      signature,
      timestamp,
      params: sortedParams
    };
  }

  static verify(secret, signature, timestamp, params) {
    const now = Date.now();
    const fiveMinutes = 5 * 60 * 1000;

    if (Math.abs(now - timestamp) > fiveMinutes) {
      return { valid: false, reason: 'timestamp_expired' };
    }

    const sortedParams = this.sortParams(params);
    const queryString = this.buildQueryString(sortedParams);
    const signaturePayload = `${queryString}&timestamp=${timestamp}`;

    const expectedSignature = crypto
      .createHmac('sha256', secret)
      .update(signaturePayload)
      .digest('hex');

    const isValid = crypto.timingSafeEqual(
      Buffer.from(signature, 'hex'),
      Buffer.from(expectedSignature, 'hex')
    );

    return { valid: isValid, reason: isValid ? 'valid' : 'signature_mismatch' };
  }

  static sortParams(params) {
    if (!params || typeof params !== 'object') {
      return {};
    }

    const sorted = {};
    const keys = Object.keys(params).sort();

    for (const key of keys) {
      const value = params[key];
      if (value !== null && value !== undefined) {
        if (typeof value === 'object') {
          sorted[key] = JSON.stringify(value);
        } else {
          sorted[key] = String(value);
        }
      }
    }

    return sorted;
  }

  static buildQueryString(params) {
    return Object.entries(params)
      .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
      .join('&');
  }
}

const apiSignatureMiddleware = async (req, res, next) => {
  const excludePaths = ['/health', '/public', '/api/health'];

  if (excludePaths.includes(req.path)) {
    return next();
  }

  const signature = req.headers['x-api-signature'];
  const timestamp = req.headers['x-api-timestamp'];

  if (!signature || !timestamp) {
    return res.status(401).json({
      error: 'missing_signature',
      message: 'API signature is required'
    });
  }

  const secret = process.env.API_SECRET;
  if (!secret) {
    console.error('API_SECRET not configured');
    return res.status(500).json({
      error: 'server_config_error',
      message: 'Server configuration error'
    });
  }

  const params = { ...req.query, ...req.body };
  const verification = APISignature.verify(secret, signature, parseInt(timestamp), params);

  if (!verification.valid) {
    return res.status(401).json({
      error: verification.reason,
      message: 'API signature verification failed'
    });
  }

  next();
};

module.exports = { APISignature, apiSignatureMiddleware };
