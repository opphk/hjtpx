import { fetch, Agent } from 'undici';
import {
  CaptchaClientConfig,
  SliderCaptchaResponse,
  ClickCaptchaResponse,
  VerifyCaptchaRequest,
  VerifyCaptchaResponse,
  LoginRequest,
  LoginResponse,
  ApiResponse,
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

  constructor(config: CaptchaClientConfig) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.baseUrl = this.config.baseUrl.replace(/\/+$/, '');
    this.connectionPool = new ConnectionPool({
      maxConnections: this.config.maxConnections,
      timeout: this.config.timeout,
    });
    this.retryManager = new RetryManager(this.config.retryConfig);
  }

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

  async getGestureCaptcha(): Promise<any> {
    const url = `${this.baseUrl}/api/v1/captcha/gesture`;
    return this.request(url, { method: 'GET' });
  }

  async verifyGestureCaptcha(
    session_id: string,
    pattern: number[]
  ): Promise<VerifyCaptchaResponse> {
    const request: VerifyCaptchaRequest = {
      session_id,
      type: 'gesture',
      points: pattern.map((p) => [p, 0] as [number, number]),
    };

    return this.verifyCaptcha(request);
  }

  async verifyCaptcha(
    request: VerifyCaptchaRequest
  ): Promise<VerifyCaptchaResponse> {
    const url = `${this.baseUrl}/api/v1/captcha/verify`;
    return this.request<VerifyCaptchaResponse>(url, {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async authLogin(request: LoginRequest): Promise<LoginResponse> {
    const url = `${this.baseUrl}/api/v1/auth/login`;
    return this.request<LoginResponse>(url, {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async authRegister(request: {
    username: string;
    email: string;
    password: string;
    behavior_data?: string;
  }): Promise<any> {
    const url = `${this.baseUrl}/api/v1/auth/register`;
    return this.request(url, {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async authRefreshToken(refreshToken: string): Promise<any> {
    const url = `${this.baseUrl}/api/v1/auth/refresh`;
    return this.request(url, {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
  }

  async authLogout(): Promise<void> {
    const url = `${this.baseUrl}/api/v1/auth/logout`;
    await this.request(url, { method: 'POST' });
  }

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
