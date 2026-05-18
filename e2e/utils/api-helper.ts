import { APIRequestContext } from '@playwright/test';

export interface PerformanceMetrics {
  latency: number;
  qps: number;
  timestamp: number;
}

export class ApiHelper {
  private request: APIRequestContext;
  private baseURL: string;
  private performanceHistory: PerformanceMetrics[] = [];

  constructor(request: APIRequestContext, baseURL: string = 'http://localhost:8080') {
    this.request = request;
    this.baseURL = baseURL;
  }

  async generateSliderCaptcha(appId?: string) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/slider/generate`, {
      data: { appId: appId || 'test-app' }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('slider_generate', latency);
    return response.json();
  }

  async verifySliderCaptcha(captchaId: string, x: number, y: number) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/slider/verify`, {
      data: { captchaId, x, y }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('slider_verify', latency);
    return response.json();
  }

  async generateClickCaptcha(appId?: string) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/click/generate`, {
      data: { appId: appId || 'test-app' }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('click_generate', latency);
    return response.json();
  }

  async verifyClickCaptcha(captchaId: string, points: { x: number; y: number }[]) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/click/verify`, {
      data: { captchaId, points }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('click_verify', latency);
    return response.json();
  }

  async generateRotateCaptcha(appId?: string) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/rotate/generate`, {
      data: { appId: appId || 'test-app' }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('rotate_generate', latency);
    return response.json();
  }

  async verifyRotateCaptcha(captchaId: string, angle: number) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/rotate/verify`, {
      data: { captchaId, angle }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('rotate_verify', latency);
    return response.json();
  }

  async generateImageCaptcha(appId?: string) {
    const startTime = Date.now();
    const response = await this.request.get(`${this.baseURL}/api/v1/captcha/image`, {
      params: { appId: appId || 'test-app' }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('image_generate', latency);
    return response.json();
  }

  async verifyImageCaptcha(captchaId: string, code: string) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/image/verify`, {
      data: { captchaId, code }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('image_verify', latency);
    return response.json();
  }

  async generateVoiceCaptcha(language: string = 'zh-CN', length: number = 4) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/voice/create`, {
      data: { language, length }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('voice_generate', latency);
    return response.json();
  }

  async verifyVoiceCaptcha(sessionId: string, code: string) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/voice/verify`, {
      data: { session_id: sessionId, code }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('voice_verify', latency);
    return response.json();
  }

  async generateGestureCaptcha(appId?: string) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/gesture/generate`, {
      data: { appId: appId || 'test-app' }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('gesture_generate', latency);
    return response.json();
  }

  async verifyGestureCaptcha(captchaId: string, gesture: string) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/gesture/verify`, {
      data: { captchaId, gesture }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('gesture_verify', latency);
    return response.json();
  }

  async generate3DCaptcha(appId?: string) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/3d/generate`, {
      data: { appId: appId || 'test-app' }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('3d_generate', latency);
    return response.json();
  }

  async verify3DCaptcha(captchaId: string, rotations: number[]) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/3d/verify`, {
      data: { captchaId, rotations }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('3d_verify', latency);
    return response.json();
  }

  async createLianliankanCaptcha(width: number = 6, height: number = 6, tileTypes: number = 8) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/lianliankan/create`, {
      data: { width, height, tile_types: tileTypes }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('lianliankan_create', latency);
    return response.json();
  }

  async verifyLianliankanCaptcha(sessionId: string, pairs: any[]) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/lianliankan/verify`, {
      data: { session_id: sessionId, pairs }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('lianliankan_verify', latency);
    return response.json();
  }

  async generateSeamlessCaptcha() {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/seamless/generate`, {
      data: {}
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('seamless_generate', latency);
    return response.json();
  }

  async verifySeamlessCaptcha(token: string) {
    const startTime = Date.now();
    const response = await this.request.post(`${this.baseURL}/api/v1/captcha/seamless/verify`, {
      data: { token }
    });
    const latency = Date.now() - startTime;
    this.recordPerformance('seamless_verify', latency);
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
    const startTime = Date.now();
    const response = await this.request.get(`${this.baseURL}/health`);
    const latency = Date.now() - startTime;
    this.recordPerformance('health', latency);
    return response.ok;
  }

  private recordPerformance(endpoint: string, latency: number) {
    const qps = latency > 0 ? 1000 / latency : 0;
    this.performanceHistory.push({
      latency,
      qps,
      timestamp: Date.now()
    });
  }

  getPerformanceMetrics() {
    return this.performanceHistory;
  }

  getAverageLatency() {
    if (this.performanceHistory.length === 0) return 0;
    const total = this.performanceHistory.reduce((sum, m) => sum + m.latency, 0);
    return total / this.performanceHistory.length;
  }

  getMaxLatency() {
    if (this.performanceHistory.length === 0) return 0;
    return Math.max(...this.performanceHistory.map(m => m.latency));
  }

  getMinLatency() {
    if (this.performanceHistory.length === 0) return 0;
    return Math.min(...this.performanceHistory.map(m => m.latency));
  }

  clearPerformanceHistory() {
    this.performanceHistory = [];
  }
}
