const crypto = require('crypto');

const axios = require('axios');

class GitHubProvider {
  constructor() {
    this.clientId = process.env.GITHUB_CLIENT_ID;
    this.clientSecret = process.env.GITHUB_CLIENT_SECRET;
    this.callbackURL = process.env.GITHUB_CALLBACK_URL || '/auth/github/callback';
    this.authorizationURL = 'https://github.com/login/oauth/authorize';
    this.tokenURL = 'https://github.com/login/oauth/access_token';
    this.apiURL = 'https://api.github.com';
  }

  getAuthorizationURL(state, scope = 'user:email') {
    const params = new URLSearchParams({
      client_id: this.clientId,
      redirect_uri: this.callbackURL,
      scope,
      state
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
          redirect_uri: this.callbackURL
        },
        {
          headers: {
            Accept: 'application/json'
          }
        }
      );
      return response.data;
    } catch (error) {
      console.error('GitHub token exchange error:', error);
      throw new Error('Failed to exchange code for access token');
    }
  }

  async getUserProfile(accessToken) {
    try {
      const [userResponse, emailsResponse] = await Promise.all([
        axios.get(`${this.apiURL}/user`, {
          headers: {
            Authorization: `Bearer ${accessToken}`,
            Accept: 'application/json'
          }
        }),
        axios.get(`${this.apiURL}/user/emails`, {
          headers: {
            Authorization: `Bearer ${accessToken}`,
            Accept: 'application/json'
          }
        })
      ]);

      const user = userResponse.data;
      const emails = emailsResponse.data;
      const primaryEmail = emails.find(e => e.primary) || emails[0];

      return {
        provider: 'github',
        providerId: user.id.toString(),
        email: primaryEmail?.email,
        emailVerified: primaryEmail?.verified || false,
        name: user.name || user.login,
        username: user.login,
        picture: user.avatar_url,
        bio: user.bio,
        blog: user.blog,
        location: user.location,
        accessToken
      };
    } catch (error) {
      console.error('GitHub profile fetch error:', error);
      throw new Error('Failed to fetch user profile from GitHub');
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
            Accept: 'application/json'
          }
        }
      );
      return response.data;
    } catch (error) {
      console.error('GitHub token refresh error:', error);
      throw new Error('Failed to refresh access token');
    }
  }

  async revokeToken(accessToken) {
    try {
      await axios.delete(`${this.apiURL}/applications/${this.clientId}/tokens/${accessToken}`, {
        headers: {
          Authorization: `Bearer ${accessToken}`,
          Accept: 'application/json'
        },
        auth: {
          username: this.clientId,
          password: this.clientSecret
        }
      });
      return true;
    } catch (error) {
      console.error('GitHub token revocation error:', error);
      return false;
    }
  }

  generateState() {
    return crypto.randomBytes(32).toString('hex');
  }
}

module.exports = new GitHubProvider();
