import axios from 'axios';

const DEFAULT_TIMEOUT = 10000;
const MAX_RETRIES = 3;

class CaptchaAPI {
  constructor(config = {}) {
    this.baseURL = config.apiServer || 'http://localhost:8080';
    this.timeout = config.timeout || DEFAULT_TIMEOUT;
    this.client = axios.create({
      baseURL: this.baseURL,
      timeout: this.timeout,
      headers: {
        'Content-Type': 'application/json'
      }
    });

    this.client.interceptors.response.use(
      response => response.data,
      error => {
        const message = error.response?.data?.message || error.message;
        const customError = new Error(message);
        customError.code = error.response?.status;
        customError.details = error.response?.data;
        return Promise.reject(customError);
      }
    );
  }

  setToken(token) {
    this.client.defaults.headers.common['Authorization'] = `Bearer ${token}`;
  }

  clearToken() {
    delete this.client.defaults.headers.common['Authorization'];
  }

  async verify(scene, params = {}, options = {}) {
    const { fingerprint, token } = params;
    const retries = options.retries ?? MAX_RETRIES;

    for (let attempt = 0; attempt < retries; attempt++) {
      try {
        const response = await this.client.post('/api/captcha/verify', {
          scene,
          fingerprint,
          token,
          timestamp: Date.now()
        });

        if (response.success) {
          return {
            success: true,
            token: response.token,
            expiresAt: response.expiresAt
          };
        }

        throw new Error(response.message || 'Verification failed');
      } catch (error) {
        if (attempt === retries - 1) throw error;
        await this.delay(1000 * Math.pow(2, attempt));
      }
    }
  }

  async refresh(scene, params = {}) {
    const { fingerprint } = params;
    const response = await this.client.post('/api/captcha/refresh', {
      scene,
      fingerprint,
      timestamp: Date.now()
    });

    if (response.success) {
      return {
        success: true,
        data: response.data,
        expiresAt: response.expiresAt
      };
    }

    throw new Error(response.message || 'Refresh failed');
  }

  async report(result, metadata = {}) {
    return this.client.post('/api/captcha/report', {
      result,
      ...metadata,
      timestamp: Date.now()
    });
  }

  async getStats(scene) {
    return this.client.get('/api/captcha/stats', {
      params: { scene }
    });
  }

  async validateToken(token) {
    return this.client.post('/api/captcha/validate', {
      token,
      timestamp: Date.now()
    });
  }

  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

let apiInstance = null;

export const createCaptchaAPI = (config) => {
  apiInstance = new CaptchaAPI(config);
  return apiInstance;
};

export const getCaptchaAPI = () => {
  if (!apiInstance) {
    apiInstance = new CaptchaAPI();
  }
  return apiInstance;
};

export default CaptchaAPI;
