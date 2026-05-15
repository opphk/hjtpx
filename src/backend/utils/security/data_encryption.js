const crypto = require('crypto');

const SENSITIVE_FIELDS = [
  'phone',
  'idCard',
  'bankAccount',
  'password',
  'apiSecret',
  'creditCard',
  'ssn',
  'passport',
  'secretKey',
  'privateKey',
  'accessToken',
  'refreshToken'
];

class SensitiveDataEncryption {
  constructor(masterKey = null) {
    this.masterKey = masterKey || process.env.ENCRYPTION_MASTER_KEY;
    if (!this.masterKey) {
      console.warn('WARNING: ENCRYPTION_MASTER_KEY not set. Using temporary key.');
      this.masterKey = crypto.randomBytes(32).toString('hex');
    }
    this.algorithm = 'aes-256-gcm';
    this.keyRotationDays = 30;
    this.lastRotation = Date.now();
  }

  encrypt(plaintext) {
    if (!plaintext) {
      return null;
    }

    const iv = crypto.randomBytes(16);
    const cipher = crypto.createCipheriv(
      this.algorithm,
      Buffer.from(this.masterKey, 'hex'),
      iv
    );

    let encrypted = cipher.update(plaintext, 'utf8', 'hex');
    encrypted += cipher.final('hex');

    const authTag = cipher.getAuthTag();

    return {
      iv: iv.toString('hex'),
      ciphertext: encrypted,
      tag: authTag.toString('hex'),
      algorithm: this.algorithm,
      timestamp: Date.now()
    };
  }

  decrypt(encrypted) {
    if (!encrypted || !encrypted.ciphertext) {
      return null;
    }

    try {
      const iv = Buffer.from(encrypted.iv, 'hex');
      const authTag = Buffer.from(encrypted.tag, 'hex');
      
      const decipher = crypto.createDecipheriv(
        encrypted.algorithm || this.algorithm,
        Buffer.from(this.masterKey, 'hex'),
        iv
      );

      decipher.setAuthTag(authTag);

      let decrypted = decipher.update(encrypted.ciphertext, 'hex', 'utf8');
      decrypted += decipher.final('utf8');

      return decrypted;
    } catch (error) {
      console.error('Decryption failed:', error.message);
      return null;
    }
  }

  shouldEncrypt(key) {
    return SENSITIVE_FIELDS.some(field => 
      key.toLowerCase().includes(field.toLowerCase())
    );
  }

  encryptObject(obj) {
    if (!obj || typeof obj !== 'object') {
      return obj;
    }

    const encrypted = Array.isArray(obj) ? [] : {};

    for (const key in obj) {
      if (Object.prototype.hasOwnProperty.call(obj, key)) {
        const value = obj[key];

        if (value && typeof value === 'object') {
          encrypted[key] = this.encryptObject(value);
        } else if (this.shouldEncrypt(key) && value) {
          const encryptedValue = this.encrypt(String(value));
          encrypted[key] = JSON.stringify(encryptedValue);
        } else {
          encrypted[key] = value;
        }
      }
    }

    return encrypted;
  }

  decryptObject(obj, encryptedFields = SENSITIVE_FIELDS) {
    if (!obj || typeof obj !== 'object') {
      return obj;
    }

    const decrypted = Array.isArray(obj) ? [] : {};

    for (const key in obj) {
      if (Object.prototype.hasOwnProperty.call(obj, key)) {
        const value = obj[key];

        if (value && typeof value === 'object') {
          decrypted[key] = this.decryptObject(value, encryptedFields);
        } else if (this.shouldEncrypt(key) && typeof value === 'string') {
          try {
            const encryptedData = JSON.parse(value);
            decrypted[key] = this.decrypt(encryptedData);
          } catch {
            decrypted[key] = value;
          }
        } else {
          decrypted[key] = value;
        }
      }
    }

    return decrypted;
  }

  rotateKey(newKey) {
    console.log('Key rotation initiated');
    this.masterKey = newKey;
    this.lastRotation = Date.now();
  }

  needsRotation() {
    const daysSinceRotation = (Date.now() - this.lastRotation) / (1000 * 60 * 60 * 24);
    return daysSinceRotation >= this.keyRotationDays;
  }
}

module.exports = { SensitiveDataEncryption, SENSITIVE_FIELDS };
