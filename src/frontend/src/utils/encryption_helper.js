const crypto = require('crypto');

class EncryptionHelper {
  static encryptAes256(data, key) {
    const iv = crypto.randomBytes(16);
    const cipher = crypto.createCipheriv('aes-256-gcm', Buffer.from(key, 'hex'), iv);
    
    let encrypted = cipher.update(data, 'utf8', 'hex');
    encrypted += cipher.final('hex');
    
    const authTag = cipher.getAuthTag();
    
    return {
      iv: iv.toString('hex'),
      ciphertext: encrypted,
      authTag: authTag.toString('hex')
    };
  }

  static decryptAes256(encrypted, key) {
    const iv = Buffer.from(encrypted.iv, 'hex');
    const authTag = Buffer.from(encrypted.authTag, 'hex');
    const decipher = crypto.createDecipheriv('aes-256-gcm', Buffer.from(key, 'hex'), iv);
    
    decipher.setAuthTag(authTag);
    
    let decrypted = decipher.update(encrypted.ciphertext, 'hex', 'utf8');
    decrypted += decipher.final('utf8');
    
    return decrypted;
  }

  static hashSha256(data) {
    return crypto.createHash('sha256').update(data).digest('hex');
  }

  static generateSecureRandom(length = 32) {
    return crypto.randomBytes(length).toString('hex');
  }

  static deriveKey(password, salt, iterations = 100000) {
    return crypto.pbkdf2Sync(password, salt, iterations, 32, 'sha512').toString('hex');
  }

  static generateSalt() {
    return crypto.randomBytes(16).toString('hex');
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = EncryptionHelper;
}
