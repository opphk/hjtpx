const SENSITIVE_FIELDS = [
  'password',
  'token',
  'authorization',
  'cookie',
  'secret',
  'apiKey',
  'creditCard',
  'ssn',
  'phone',
  'idCard',
  'bankAccount',
  'passport',
  'privateKey',
  'accessToken',
  'refreshToken',
  'cvv',
  'securityCode',
  'pin'
];

class LogSanitizer {
  static sanitize(obj, options = {}) {
    if (obj === null || obj === undefined) {
      return obj;
    }

    const {
      maskChar = '*',
      maskLength = 6,
      preserveFirst = 2,
      preserveLast = 4,
      customFields = []
    } = options;

    const sensitiveFields = [...SENSITIVE_FIELDS, ...customFields];

    if (typeof obj === 'string') {
      return this.sanitizeValue(obj, sensitiveFields, maskChar, maskLength);
    }

    if (Array.isArray(obj)) {
      return obj.map(item => this.sanitize(item, options));
    }

    if (typeof obj === 'object') {
      const sanitized = {};

      for (const key in obj) {
        if (Object.prototype.hasOwnProperty.call(obj, key)) {
          const value = obj[key];

          if (this.isSensitiveField(key, sensitiveFields)) {
            sanitized[key] = this.maskValue(value, maskChar);
          } else if (value && typeof value === 'object') {
            sanitized[key] = this.sanitize(value, options);
          } else if (typeof value === 'string' && this.containsSensitivePattern(value)) {
            sanitized[key] = this.maskSensitiveContent(value, maskChar);
          } else {
            sanitized[key] = value;
          }
        }
      }

      return sanitized;
    }

    return obj;
  }

  static isSensitiveField(key, sensitiveFields) {
    const lowerKey = key.toLowerCase();
    return sensitiveFields.some(field => lowerKey.includes(field.toLowerCase()));
  }

  static maskValue(value, maskChar = '*') {
    if (value === null || value === undefined) {
      return null;
    }

    const str = String(value);
    if (str.length <= 4) {
      return maskChar.repeat(str.length);
    }

    return str.substring(0, 2) + maskChar.repeat(str.length - 4) + str.substring(str.length - 2);
  }

  static maskSensitiveContent(value, maskChar = '*') {
    const emailPattern = /[\w.-]+@[\w.-]+\.\w+/g;
    const phonePattern = /1[3-9]\d{9}/g;
    const idCardPattern = /\d{17}[\dXx]/g;
    const creditCardPattern = /\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}/g;

    let result = value;

    result = result.replace(emailPattern, match => this.maskEmail(match));
    result = result.replace(phonePattern, match => this.maskPhone(match));
    result = result.replace(idCardPattern, match => this.maskIdCard(match));
    result = result.replace(creditCardPattern, match => this.maskCreditCard(match));

    return result;
  }

  static maskEmail(email) {
    if (!email || !email.includes('@')) {
      return email;
    }

    const [local, domain] = email.split('@');
    if (local.length <= 3) {
      return local[0] + '***@' + domain;
    }

    return local.substring(0, 2) + '***@' + domain;
  }

  static maskPhone(phone) {
    if (!phone || phone.length < 7) {
      return phone;
    }

    const visibleStart = phone.substring(0, 3);
    const visibleEnd = phone.substring(phone.length - 4);
    return visibleStart + '****' + visibleEnd;
  }

  static maskIdCard(idCard) {
    if (!idCard || idCard.length < 10) {
      return idCard;
    }

    const visibleStart = idCard.substring(0, 3);
    const visibleEnd = idCard.substring(idCard.length - 4);
    return visibleStart + '***********' + visibleEnd;
  }

  static maskCreditCard(card) {
    if (!card) {
      return card;
    }

    const digitsOnly = card.replace(/\D/g, '');
    if (digitsOnly.length < 13) {
      return card;
    }

    const visibleEnd = digitsOnly.substring(digitsOnly.length - 4);
    return '****-****-****-' + visibleEnd;
  }

  static sanitizeValue(value, sensitiveFields, maskChar, maskLength) {
    const str = String(value);

    if (this.containsSensitivePattern(str)) {
      return this.maskSensitiveContent(str, maskChar);
    }

    return str;
  }

  static containsSensitivePattern(value) {
    const patterns = [
      /[\w.-]+@[\w.-]+\.\w+/,
      /1[3-9]\d{9}/,
      /\d{17}[\dXx]/,
      /\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}/
    ];

    return patterns.some(pattern => pattern.test(value));
  }

  static createSanitizer(options = {}) {
    return obj => this.sanitize(obj, options);
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = { LogSanitizer, SENSITIVE_FIELDS };
}
