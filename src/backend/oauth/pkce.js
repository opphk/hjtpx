const crypto = require('crypto');

class PKCE {
  static generateCodeVerifier(length = 128) {
    const charset = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~';
    let verifier = '';
    const randomValues = crypto.randomBytes(length);
    for (let i = 0; i < length; i++) {
      verifier += charset[randomValues[i] % charset.length];
    }
    return verifier;
  }

  static async generateCodeChallenge(verifier) {
    const hash = crypto.createHash('sha256').update(verifier).digest('base64');
    return hash.replace(/\+/g, '-').replace(/\//g, '_').replace(/[=]/g, '');
  }

  static async verifyCodeChallenge(verifier, challenge) {
    const expectedChallenge = await this.generateCodeChallenge(verifier);
    return crypto.timingSafeEqual(Buffer.from(expectedChallenge), Buffer.from(challenge));
  }

  static validateCodeVerifier(verifier) {
    if (!verifier || typeof verifier !== 'string') {
      return false;
    }
    if (verifier.length < 43 || verifier.length > 128) {
      return false;
    }
    const validChars = /^[A-Za-z0-9-._~]+$/;
    return validChars.test(verifier);
  }

  static validateCodeChallenge(challenge) {
    if (!challenge || typeof challenge !== 'string') {
      return false;
    }
    const base64urlRegex = /^[A-Za-z0-9_-]+$/;
    return base64urlRegex.test(challenge);
  }
}

module.exports = PKCE;
