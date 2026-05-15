class SQLInjectionProtection {
  static get dangerousPatterns() {
    return [
      /\b(SELECT|INSERT|UPDATE|DELETE|DROP|CREATE|ALTER|TRUNCATE|VALUES)\b/gi,
      /\b(UNION|EXEC|EXECUTE|XP_|SP_)\b/gi,
      /(--|#|\/\*|\*\/)/g,
      /;\s*(DROP|INSERT|UPDATE|DELETE|EXEC)/gi,
      /\b(OR|AND)\s+\d+\s*=\s*\d+/gi,
      /\b(OR|AND)\s+['"][^'"]*['"]\s*=\s*['"][^'"]*['"]/gi,
      /(WAITFOR|DELAY|SLEEP|BENCHMARK)\s*\(/gi,
      /\bINTO\s+(OUTFILE|DUMPFILE)\s+/gi,
      /LOAD_FILE\s*\(/gi,
      /CONCAT\s*\(/gi,
      /CHAR\s*\(/gi,
      /\bUNION\s+(ALL\s+)?SELECT/gi
    ];
  }

  static sanitize(input) {
    if (input === null || input === undefined) {
      return '';
    }

    if (typeof input === 'number') {
      return input;
    }

    let str = String(input);

    str = str.replace(/'/g, "''");
    
    for (const pattern of this.dangerousPatterns) {
      str = str.replace(pattern, '');
    }

    str = str.replace(/\x00/g, '');
    str = str.replace(/\x1a/g, '');

    return str.trim();
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

  static validate(input) {
    if (input === null || input === undefined) {
      return { valid: true };
    }

    const str = String(input);

    for (const pattern of this.dangerousPatterns) {
      if (pattern.test(str)) {
        return {
          valid: false,
          reason: 'dangerous_pattern_detected',
          pattern: pattern.source
        };
      }
    }

    return { valid: true };
  }

  static isNumeric(value) {
    return !isNaN(parseFloat(value)) && isFinite(value);
  }

  static validateNumericInput(value) {
    if (this.isNumeric(value)) {
      return { valid: true, value: parseFloat(value) };
    }
    return { valid: false, reason: 'non_numeric_input' };
  }
}

module.exports = SQLInjectionProtection;
