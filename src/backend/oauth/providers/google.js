const crypto = require('crypto');

const axios = require('axios');

class GoogleProvider {
  constructor() {
    this.clientId = process.env.GOOGLE_CLIENT_ID;
    this.clientSecret = process.env.GOOGLE_CLIENT_SECRET;
    this.callbackURL = process.env.GOOGLE_CALLBACK_URL || '/auth/google/callback';
    this.authorizationURL = 'https://accounts.google.com/o/oauth2/v2/auth';
    this.tokenURL = 'https://oauth2.googleapis.com/token';
    this.apiURL = 'https://www.googleapis.com/oauth2/v2';
    this.userInfoURL = 'https://www.googleapis.com/oauth2/v3/userinfo';
  }

  getAuthorizationURL(state, scope = 'openid email profile') {
    const params = new URLSearchParams({
      client_id: this.clientId,
      redirect_uri: this.callbackURL,
      response_type: 'code',
      scope,
      state,
      access_type: 'offline',
      prompt: 'consent'
    });
    return `${this.authorizationURL}?${params.toString()}`;
  }

  async getAccessToken(code) {
    try {
      const response = await axios.post(
        this.tokenURL,
        {
          client_id: this.clientId,
          client_secret: this.clientSecret,
          code,
          grant_type: 'authorization_code',
          redirect_uri: this.callbackURL
        },
        {
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
          }
        }
      );
      return response.data;
    } catch (error) {
      console.error('Google token exchange error:', error);
      throw new Error('Failed to exchange code for access token');
    }
  }

  async getUserProfile(accessToken, idToken) {
    try {
      let profile;

      if (idToken) {
        const payload = this.decodeIdToken(idToken);
        profile = {
          id: payload.sub,
          email: payload.email,
          emailVerified: payload.email_verified,
          name: payload.name,
          picture: payload.picture
        };
      }

      if (!profile || !profile.email) {
        const response = await axios.get(this.userInfoURL, {
          headers: {
            Authorization: `Bearer ${accessToken}`
          }
        });
        profile = response.data;
      }

      return {
        provider: 'google',
        providerId: profile.id || profile.sub,
        email: profile.email,
        emailVerified: profile.email_verified || false,
        name: profile.name,
        picture: profile.picture,
        accessToken
      };
    } catch (error) {
      console.error('Google profile fetch error:', error);
      throw new Error('Failed to fetch user profile from Google');
    }
  }

  decodeIdToken(idToken) {
    try {
      const parts = idToken.split('.');
      if (parts.length !== 3) {
        throw new Error('Invalid ID token format');
      }
      const payload = Buffer.from(parts[1], 'base64').toString('utf-8');
      return JSON.parse(payload);
    } catch (error) {
      throw new Error('Failed to decode ID token');
    }
  }

  async refreshAccessToken(refreshToken) {
    try {
      const response = await axios.post(
        this.tokenURL,
        {
          client_id: this.clientId,
          client_secret: this.clientSecret,
          grant_type: 'refresh_token',
          refresh_token: refreshToken
        },
        {
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
          }
        }
      );
      return response.data;
    } catch (error) {
      console.error('Google token refresh error:', error);
      throw new Error('Failed to refresh access token');
    }
  }

  async revokeToken(accessToken) {
    try {
      await axios.post(
        'https://oauth2.googleapis.com/revoke',
        new URLSearchParams({
          token: accessToken
        }),
        {
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
          }
        }
      );
      return true;
    } catch (error) {
      console.error('Google token revocation error:', error);
      return false;
    }
  }

  generateState() {
    return crypto.randomBytes(32).toString('hex');
  }
}

module.exports = new GoogleProvider();
