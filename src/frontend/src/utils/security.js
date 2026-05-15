class FrontendSecurityService {
  constructor() {
    this.csrfToken = null;
    this.securityHeaders = {
      'Content-Security-Policy': "default-src 'self'",
      'X-Content-Type-Options': 'nosniff',
      'X-Frame-Options': 'DENY',
      'X-XSS-Protection': '1; mode=block',
      'Strict-Transport-Security': 'max-age=31536000; includeSubDomains'
    };
    this.sanitizedInputs = new Set();
    this.xssPatterns = [
      /<script[^>]*>/i,
      /<iframe[^>]*>/i,
      /javascript:/i,
      /on\w+\s*=/i,
      /<img[^>]+onerror/i
    ];
  }

  generateCSRFToken() {
    const array = new Uint8Array(32);
    crypto.getRandomValues(array);
    this.csrfToken = Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
    return this.csrfToken;
  }

  getCSRFToken() {
    return this.csrfToken;
  }

  validateCSRFToken(token) {
    if (!this.csrfToken || !token) {
      return false;
    }
    return token === this.csrfToken;
  }

  sanitizeInput(input) {
    if (typeof input !== 'string') {
      return input;
    }

    let sanitized = input;
    sanitized = sanitized.replace(/</g, '&lt;');
    sanitized = sanitized.replace(/>/g, '&gt;');
    sanitized = sanitized.replace(/"/g, '&quot;');
    sanitized = sanitized.replace(/'/g, '&#x27;');
    sanitized = sanitized.replace(/\//g, '&#x2F;');

    return sanitized;
  }

  detectXSS(input) {
    if (typeof input !== 'string') {
      return false;
    }

    for (const pattern of this.xssPatterns) {
      if (pattern.test(input)) {
        return true;
      }
    }

    return false;
  }

  sanitizeObject(obj) {
    if (typeof obj === 'string') {
      return this.sanitizeInput(obj);
    }

    if (Array.isArray(obj)) {
      return obj.map(item => this.sanitizeObject(item));
    }

    if (typeof obj === 'object' && obj !== null) {
      const sanitized = {};
      for (const [key, value] of Object.entries(obj)) {
        sanitized[key] = this.sanitizeObject(value);
      }
      return sanitized;
    }

    return obj;
  }

  validatePassword(password) {
    const errors = [];

    if (password.length < 8) {
      errors.push('Password must be at least 8 characters long');
    }

    if (!/[A-Z]/.test(password)) {
      errors.push('Password must contain at least one uppercase letter');
    }

    if (!/[a-z]/.test(password)) {
      errors.push('Password must contain at least one lowercase letter');
    }

    if (!/[0-9]/.test(password)) {
      errors.push('Password must contain at least one number');
    }

    if (!/[!@#$%^&*(),.?":{}|<>]/.test(password)) {
      errors.push('Password must contain at least one special character');
    }

    return {
      isValid: errors.length === 0,
      errors
    };
  }

  validateEmail(email) {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
  }

  getSecurityHeaders() {
    return { ...this.securityHeaders };
  }

  encryptData(data, key) {
    const encoder = new TextEncoder();
    const dataBuffer = encoder.encode(JSON.stringify(data));
    const keyBuffer = encoder.encode(key);

    let result = [];
    for (let i = 0; i < dataBuffer.length; i++) {
      result.push(dataBuffer[i] ^ keyBuffer[i % keyBuffer.length]);
    }

    return btoa(String.fromCharCode(...result));
  }

  decryptData(encryptedData, key) {
    try {
      const dataBuffer = Uint8Array.from(atob(encryptedData), c => c.charCodeAt(0));
      const keyBuffer = new TextEncoder().encode(key);

      let result = [];
      for (let i = 0; i < dataBuffer.length; i++) {
        result.push(dataBuffer[i] ^ keyBuffer[i % keyBuffer.length]);
      }

      const decoder = new TextDecoder();
      return JSON.parse(decoder.decode(new Uint8Array(result)));
    } catch (error) {
      console.error('Decryption failed:', error);
      return null;
    }
  }

  generateSecureRandom(length = 32) {
    const array = new Uint8Array(length);
    crypto.getRandomValues(array);
    return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
  }

  hashPassword(password) {
    let hash = 0;
    for (let i = 0; i < password.length; i++) {
      const char = password.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return hash.toString(16);
  }
}

const frontendSecurityService = new FrontendSecurityService();

export default frontendSecurityService;
export { FrontendSecurityService };
