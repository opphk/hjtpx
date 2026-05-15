class XSSProtection {
  static get escapeMap() {
    return {
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#x27;'
    };
  }

  static escapeHtml(str) {
    if (str === null || str === undefined) {
      return '';
    }
    return String(str).replace(/[&<>"']/g, char => this.escapeMap[char]);
  }

  static sanitize(input) {
    if (input === null || input === undefined) {
      return '';
    }

    let str = String(input);

    str = str.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '');
    str = str.replace(/\s*on\w+\s*=\s*["'][^"']*["']/gi, '');
    str = str.replace(/\s*on\w+\s*=\s*[^\s>]+/gi, '');
    str = str.replace(/javascript\s*:/gi, '');
    str = str.replace(/data\s*:\s*text\/html/gi, '');
    str = str.replace(/<iframe/gi, '&lt;iframe');
    str = str.replace(/<object/gi, '&lt;object');
    str = str.replace(/<embed/gi, '&lt;embed');
    str = str.replace(/<link/gi, '&lt;link');
    str = str.replace(/<import/gi, '&lt;import');
    str = str.replace(/expression\s*\(/gi, '');
    str = str.replace(/url\s*\(/gi, '');

    return this.escapeHtml(str);
  }

  static sanitizeObject(obj) {
    if (obj === null || obj === undefined) {
      return obj;
    }

    if (Array.isArray(obj)) {
      return obj.map(item => this.sanitizeObject(item));
    }

    if (typeof obj === 'object') {
      const sanitized = {};
      for (const key in obj) {
        if (Object.prototype.hasOwnProperty.call(obj, key)) {
          sanitized[key] = this.sanitizeObject(obj[key]);
        }
      }
      return sanitized;
    }

    return this.sanitize(obj);
  }

  static validateNoXSS(input) {
    const patterns = [
      /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi,
      /javascript\s*:/gi,
      /on\w+\s*=/gi,
      /data\s*:\s*text\/html/gi,
      /<iframe/gi,
      /expression\s*\(/gi,
      /\$\{[^}]+\}/g
    ];

    return !patterns.some(pattern => pattern.test(input));
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = XSSProtection;
}
