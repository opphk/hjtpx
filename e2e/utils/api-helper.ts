import { APIRequestContext } from '@playwright/test';

export class ApiHelper {
  private request: APIRequestContext;
  private baseURL: string;

  constructor(request: APIRequestContext, baseURL: string = 'http://localhost:8080') {
    this.request = request;
    this.baseURL = baseURL;
  }

  async generateSliderCaptcha(appId?: string) {
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/slider/generate`, {
      data: { appId: appId || 'test-app' }
    });
    return response.json();
  }

  async verifySliderCaptcha(captchaId: string, x: number, y: number) {
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/slider/verify`, {
      data: { captchaId, x, y }
    });
    return response.json();
  }

  async generateClickCaptcha(appId?: string) {
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/click/generate`, {
      data: { appId: appId || 'test-app' }
    });
    return response.json();
  }

  async verifyClickCaptcha(captchaId: string, points: { x: number; y: number }[]) {
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/click/verify`, {
      data: { captchaId, points }
    });
    return response.json();
  }

  async generateRotateCaptcha(appId?: string) {
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/rotate/generate`, {
      data: { appId: appId || 'test-app' }
    });
    return response.json();
  }

  async verifyRotateCaptcha(captchaId: string, angle: number) {
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/rotate/verify`, {
      data: { captchaId, angle }
    });
    return response.json();
  }

  async generateImageCaptcha(appId?: string) {
    const response = await this.request.get(`${this.baseURL}/api/v1/captcha/image`, {
      params: { appId: appId || 'test-app' }
    });
    return response.json();
  }

  async verifyImageCaptcha(captchaId: string, code: string) {
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/image/verify`, {
      data: { captchaId, code }
    });
    return response.json();
  }

  async adminLogin(username: string, password: string) {
    const response = await this.request.post(`${this.baseURL}/api/v1/auth/login`, {
      data: { username, password }
    });
    return response.json();
  }

  async adminLogout(token: string) {
    const response = await this.request.post(`${this.baseURL}/api/v1/admin/logout`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    return response.json();
  }

  async getVerificationStats(token: string) {
    const response = await this.request.get(`${this.baseURL}/api/v1/admin/stats/verification`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    return response.json();
  }

  async getApplications(token: string) {
    const response = await this.request.get(`${this.baseURL}/api/v1/admin/applications`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    return response.json();
  }

  async createApplication(token: string, name: string, description: string) {
    const response = await this.request.post(`${this.baseURL}/api/v1/admin/applications`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { name, description }
    });
    return response.json();
  }

  async getLogs(token: string, params?: any) {
    const response = await this.request.get(`${this.baseURL}/api/v1/admin/logs`, {
      headers: { Authorization: `Bearer ${token}` },
      params
    });
    return response.json();
  }

  async healthCheck() {
    const response = await this.request.get(`${this.baseURL}/health`);
    return response.ok;
  }
}
