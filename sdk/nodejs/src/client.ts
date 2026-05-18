import { fetch, Agent } from 'undici';
import {
  CaptchaClientConfig,
  SliderCaptchaResponse,
  ClickCaptchaResponse,
  ImageCaptchaResponse,
  RotationCaptchaResponse,
  GestureCaptchaResponse,
  JigsawCaptchaResponse,
  JigsawPiece,
  VerifyCaptchaRequest,
  VerifyCaptchaResponse,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  ApiResponse,
  TrajectoryPoint,
} from './types';
import {
  CaptchaError,
  ValidationError,
  AuthenticationError,
  NotFoundError,
  RateLimitError,
  ServerError,
  NetworkError,
} from './errors';
import { ConnectionPool } from './connection-pool';
import { RetryManager } from './retry';

const DEFAULT_CONFIG: Partial<CaptchaClientConfig> = {
  timeout: 30000,
  maxConnections: 100,
};

export class CaptchaClient {
  private config: CaptchaClientConfig;
  private connectionPool: ConnectionPool;
  private retryManager: RetryManager;
  private baseUrl: string;
  private _token: string | null = null;
  private _refreshToken: string | null = null;

  constructor(config: CaptchaClientConfig) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.baseUrl = this.config.baseUrl.replace(/\/+$/, '');
    this.connectionPool = new ConnectionPool({
      maxConnections: this.config.maxConnections,
      timeout: this.config.timeout,
    });
    this.retryManager = new RetryManager(this.config.retryConfig);
  }

  setToken(token: string | null): void {
    this._token = token;
  }

  // ==================== 滑块验证码 ====================
  async getSliderCaptcha(options?: {
    width?: number;
    height?: number;
    tolerance?: number;
  }): Promise<SliderCaptchaResponse> {
    const params = new URLSearchParams();
    if (options?.width !== undefined) {
      params.append('width', options.width.toString());
    }
    if (options?.height !== undefined) {
      params.append('height', options.height.toString());
    }
    if (options?.tolerance !== undefined) {
      params.append('tolerance', options.tolerance.toString());
    }

    const url = `${this.baseUrl}/api/v1/captcha/slider${
      params.toString() ? `?${params.toString()}` : ''
    }`;

    return this.request<SliderCaptchaResponse>(url, { method: 'GET' });
  }

  async verifySliderCaptcha(
    sessionId: string,
    x: number,
    options?: {
      y?: number;
      trajectory?: TrajectoryPoint[];
      behaviorData?: Record<string, unknown>[];
    }
  ): Promise<VerifyCaptchaResponse> {
    const request: VerifyCaptchaRequest = {
      session_id: sessionId,
      type: 'slider',
      x,
      y: options?.y,
      trajectory: options?.trajectory,
      behavior_data: options?.behaviorData,
    };

    return this.verifyCaptcha(request);
  }

  // ==================== 点击验证码 ====================
  async getClickCaptcha(options?: {
    mode?: 'number' | 'letter' | 'chinese' | 'mixed' | 'icon';
    shuffle?: boolean;
    points?: number;
  }): Promise<ClickCaptchaResponse> {
    const params = new URLSearchParams();
    if (options?.mode) {
      params.append('mode', options.mode);
    }
    if (options?.shuffle !== undefined) {
      params.append('shuffle', options.shuffle.toString());
    }
    if (options?.points !== undefined) {
      params.append('points', options.points.toString());
    }

    const url = `${this.baseUrl}/api/v1/captcha/click${
      params.toString() ? `?${params.toString()}` : ''
    }`;

    return this.request<ClickCaptchaResponse>(url, { method: 'GET' });
  }

  async verifyClickCaptcha(
    sessionId: string,
    points: [number, number][],
    options?: {
      clickSequence?: number[];
      behaviorData?: Record<string, unknown>[];
    }
  ): Promise<VerifyCaptchaResponse> {
    const request: VerifyCaptchaRequest = {
      session_id: sessionId,
      type: 'click',
      points,
      click_sequence: options?.clickSequence,
      behavior_data: options?.behaviorData,
    };

    return this.verifyCaptcha(request);
  }

  // ==================== 图形验证码 ====================
  async getImageCaptcha(options?: {
    type?: 'number' | 'letter' | 'mixed';
    count?: number;
    noiseMode?: number;
    lineMode?: number;
  }): Promise<ImageCaptchaResponse> {
    const params = new URLSearchParams();
    if (options?.type) {
      params.append('type', options.type);
    }
    if (options?.count !== undefined) {
      params.append('count', options.count.toString());
    }
    if (options?.noiseMode !== undefined) {
      params.append('noise_mode', options.noiseMode.toString());
    }
    if (options?.lineMode !== undefined) {
      params.append('line_mode', options.lineMode.toString());
    }

    const url = `${this.baseUrl}/api/v1/captcha/image${
      params.toString() ? `?${params.toString()}` : ''
    }`;

    return this.request<ImageCaptchaResponse>(url, { method: 'GET' });
  }

  async verifyImageCaptcha(
    challengeId: string,
    answer: string
  ): Promise<VerifyCaptchaResponse> {
    const url = `${this.baseUrl}/api/v1/captcha/image/verify`;
    return this.request<VerifyCaptchaResponse>(url, {
      method: 'POST',
      body: JSON.stringify({ challenge_id: challengeId, answer }),
    });
  }

  // ==================== 旋转验证码 ====================
  async getRotationCaptcha(): Promise<RotationCaptchaResponse> {
    const url = `${this.baseUrl}/api/v1/captcha/rotation`;
    return this.request<RotationCaptchaResponse>(url, { method: 'GET' });
  }

  async verifyRotationCaptcha(
    challengeId: string,
    angle: number
  ): Promise<VerifyCaptchaResponse> {
    const url = `${this.baseUrl}/api/v1/captcha/rotation/verify`;
    return this.request<VerifyCaptchaResponse>(url, {
      method: 'POST',
      body: JSON.stringify({ challenge_id: challengeId, angle }),
    });
  }

  // ==================== 手势验证码 ====================
  async getGestureCaptcha(): Promise<GestureCaptchaResponse> {
    const url = `${this.baseUrl}/api/v1/captcha/gesture`;
    return this.request<GestureCaptchaResponse>(url, { method: 'GET' });
  }

  async verifyGestureCaptcha(
    sessionId: string,
    pattern: number[]
  ): Promise<VerifyCaptchaResponse> {
    const request: VerifyCaptchaRequest = {
      session_id: sessionId,
      type: 'gesture',
      points: pattern.map((p) => [p, 0] as [number, number]),
    };

    return this.verifyCaptcha(request);
  }

  // ==================== 拼图验证码 ====================
  async getJigsawCaptcha(options?: {
    width?: number;
    height?: number;
    gridSize?: number;
  }): Promise<JigsawCaptchaResponse> {
    const params = new URLSearchParams();
    if (options?.width !== undefined) {
      params.append('width', options.width.toString());
    }
    if (options?.height !== undefined) {
      params.append('height', options.height.toString());
    }
    if (options?.gridSize !== undefined) {
      params.append('grid_size', options.gridSize.toString());
    }

    const url = `${this.baseUrl}/api/v1/captcha/jigsaw${
      params.toString() ? `?${params.toString()}` : ''
    }`;

    return this.request<JigsawCaptchaResponse>(url, { method: 'GET' });
  }

  async verifyJigsawCaptcha(
    sessionId: string,
    pieces: JigsawPiece[]
  ): Promise<VerifyCaptchaResponse> {
    const request: VerifyCaptchaRequest = {
      session_id: sessionId,
      type: 'jigsaw',
      pieces,
    };

    return this.verifyCaptcha(request);
  }

  // ==================== 通用验证方法 ====================
  async verifyCaptcha(
    request: VerifyCaptchaRequest
  ): Promise<VerifyCaptchaResponse> {
    const url = `${this.baseUrl}/api/v1/captcha/verify`;
    return this.request<VerifyCaptchaResponse>(url, {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  // ==================== 用户认证 ====================
  async authLogin(request: LoginRequest): Promise<LoginResponse> {
    const url = `${this.baseUrl}/api/v1/auth/login`;
    const response = await this.request<LoginResponse>(url, {
      method: 'POST',
      body: JSON.stringify(request),
    });
    this._token = response.access_token;
    this._refreshToken = response.refresh_token;
    return response;
  }

  async authRegister(request: RegisterRequest): Promise<LoginResponse> {
    const url = `${this.baseUrl}/api/v1/auth/register`;
    return this.request<LoginResponse>(url, {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async authRefreshToken(refreshToken?: string): Promise<any> {
    const token = refreshToken || this._refreshToken;
    if (!token) {
      throw new CaptchaError('No refresh token available');
    }

    const url = `${this.baseUrl}/api/v1/auth/refresh`;
    const result = await this.request(url, {
      method: 'POST',
      body: JSON.stringify({ refresh_token: token }),
    });

    if (result.access_token) {
      this._token = result.access_token;
      if (result.refresh_token) {
        this._refreshToken = result.refresh_token;
      }
    }

    return result;
  }

  async authLogout(): Promise<void> {
    const url = `${this.baseUrl}/api/v1/auth/logout`;
    try {
      await this.request(url, { method: 'POST' });
    } finally {
      this._token = null;
      this._refreshToken = null;
    }
  }

  async authVerifyEmail(token: string): Promise<any> {
    const params = new URLSearchParams({ token });
    const url = `${this.baseUrl}/api/v1/auth/verify-email?${params.toString()}`;
    return this.request(url, { method: 'GET' });
  }

  async authRequestPasswordReset(email: string): Promise<any> {
    const url = `${this.baseUrl}/api/v1/auth/request-password-reset`;
    return this.request(url, {
      method: 'POST',
      body: JSON.stringify({ email }),
    });
  }

  async authResetPassword(token: string, newPassword: string): Promise<any> {
    const url = `${this.baseUrl}/api/v1/auth/reset-password`;
    return this.request(url, {
      method: 'POST',
      body: JSON.stringify({ token, new_password: newPassword }),
    });
  }

  // ==================== 环境检测 ====================
  async getDetectionScript(callback?: string): Promise<string> {
    const params = new URLSearchParams();
    if (callback) {
      params.append('callback', callback);
    }

    const url = `${this.baseUrl}/api/v1/detect/script${
      params.toString() ? `?${params.toString()}` : ''
    }`;

    const response = await this.rawRequest(url, { method: 'GET' });
    return response.text();
  }

  async submitDetection(data: Record<string, unknown>): Promise<any> {
    const url = `${this.baseUrl}/api/v1/detect/submit`;
    return this.request(url, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async checkEnvironment(data: Record<string, unknown>): Promise<any> {
    const url = `${this.baseUrl}/api/v1/detect/check`;
    return this.request(url, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // ==================== 内部方法 ====================
  private async request<T>(
    url: string,
    options: {
      method: 'GET' | 'POST' | 'PUT' | 'DELETE';
      body?: string;
    }
  ): Promise<T> {
    return this.retryManager.execute(async () => {
      const response = await this.rawRequest(url, options);
      return this.parseResponse<T>(response);
    });
  }

  private async rawRequest(
    url: string,
    options: {
      method: 'GET' | 'POST' | 'PUT' | 'DELETE';
      body?: string;
    }
  ): Promise<Response> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.config.apiKey) {
      headers['X-API-Key'] = this.config.apiKey;
    }

    if (this._token) {
      headers['Authorization'] = `Bearer ${this._token}`;
    }

    const fetchOptions: RequestInit = {
      method: options.method,
      headers,
      body: options.body,
      signal: AbortSignal.timeout(this.config.timeout ?? 30000),
      dispatcher: this.connectionPool.getUndiciAgent(),
    };

    try {
      const response = await fetch(url, fetchOptions);
      return response;
    } catch (error) {
      if (error instanceof Error && error.name === 'TimeoutError') {
        throw new NetworkError('Request timeout', error);
      }
      throw new NetworkError(
        error instanceof Error ? error.message : 'Network error',
        error instanceof Error ? error : undefined
      );
    }
  }

  private async parseResponse<T>(response: Response): Promise<T> {
    let responseBody: any;

    try {
      responseBody = await response.json();
    } catch (error) {
      if (response.ok) {
        return undefined as T;
      }
      throw this.handleErrorStatus(response);
    }

    if (!response.ok) {
      throw this.handleErrorStatus(response, responseBody);
    }

    const apiResponse = responseBody as ApiResponse<T>;
    if (apiResponse.code !== 0) {
      throw new CaptchaError(
        apiResponse.message || 'Unknown error',
        `API_ERROR_${apiResponse.code}`,
        response.status,
        this.isRetryableStatus(response.status)
      );
    }

    return apiResponse.data;
  }

  private handleErrorStatus(
    response: Response,
    responseBody?: any
  ): CaptchaError {
    const status = response.status;
    const message =
      responseBody?.message ||
      response.statusText ||
      'An error occurred';

    switch (status) {
      case 400:
        return new ValidationError(message);
      case 401:
        return new AuthenticationError(message);
      case 404:
        return new NotFoundError(message);
      case 429:
        const retryAfter = response.headers.get('Retry-After');
        return new RateLimitError(
          message,
          retryAfter ? parseInt(retryAfter, 10) : undefined
        );
      case 500:
      case 502:
      case 503:
      case 504:
        return new ServerError(message, status);
      default:
        return new CaptchaError(
          message,
          `HTTP_ERROR_${status}`,
          status,
          this.isRetryableStatus(status)
        );
    }
  }

  private isRetryableStatus(status: number): boolean {
    return [429, 500, 502, 503, 504].includes(status);
  }

  async close(): Promise<void> {
    await this.connectionPool.destroy();
  }
}