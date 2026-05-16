import {
  Config,
  SDKOptions,
  SDKResponse,
  SDKError,
  ImageCaptchaRequest,
  ImageCaptchaResponse,
  SliderCaptchaRequest,
  SliderCaptchaResponse,
  ClickCaptchaRequest,
  ClickCaptchaResponse,
  ClickData,
  VerifyCaptchaRequest,
  VerifyCaptchaResponse,
  VerifyImageCaptchaRequest,
  VerifyImageCaptchaResponse,
  PoolStats,
  CaptchaType,
} from './types';

import {
  SDKError as SDKErrorClass,
  NetworkError,
  TimeoutError,
  InvalidParamsError,
  RateLimitedError,
  UnauthorizedError,
  ServerError,
} from './errors';

const DEFAULT_API_ENDPOINT = 'http://localhost:8080';
const IMAGE_CAPTCHA_PATH = '/api/v1/captcha/image';
const IMAGE_VERIFY_PATH = '/api/v1/captcha/image/verify';
const SLIDER_CAPTCHA_PATH = '/api/v1/captcha/slider';
const CLICK_CAPTCHA_PATH = '/api/v1/captcha/click';
const VERIFY_PATH = '/api/v1/captcha/verify';

interface Stats {
  totalRequests: number;
  failedRequests: number;
  successfulRequests: number;
  retriedRequests: number;
  lastError: string | null;
  lastErrorTime: string | null;
}

export class CaptchaClient {
  private config: Required<Config>;
  private fetch: typeof fetch;
  private stats: Stats;

  constructor(options: SDKOptions = {}) {
    this.config = {
      baseUrl: options.baseUrl || DEFAULT_API_ENDPOINT,
      appId: options.appId || '',
      appSecret: options.appSecret || '',
      timeout: options.timeout || 30000,
      maxRetries: options.maxRetries ?? 3,
      retryDelay: options.retryDelay || 100,
      maxIdleConns: options.maxIdleConns || 10,
      maxOpenConns: options.maxOpenConns || 100,
      debugMode: options.debugMode || false,
    };

    this.fetch = options.fetch || globalThis.fetch.bind(globalThis);
    this.stats = {
      totalRequests: 0,
      failedRequests: 0,
      successfulRequests: 0,
      retriedRequests: 0,
      lastError: null,
      lastErrorTime: null,
    };
  }

  private buildUrl(path: string, params?: Record<string, string | number | undefined>): string {
    let url = `${this.config.baseUrl.replace(/\/$/, '')}${path}`;

    if (params) {
      const queryParams = Object.entries(params)
        .filter(([, value]) => value !== undefined)
        .map(([key, value]) => {
          return `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`;
        })
        .join('&');

      if (queryParams) {
        url += `?${queryParams}`;
      }
    }

    return url;
  }

  private async doRequest<T>(
    method: string,
    path: string,
    body?: any,
    params?: Record<string, string | number | undefined>
  ): Promise<SDKResponse<T>> {
    const url = this.buildUrl(path, params);
    let lastError: SDKError | null = null;

    for (let attempt = 0; attempt <= this.config.maxRetries; attempt++) {
      if (attempt > 0) {
        this.stats.retriedRequests++;
        await this.sleep(this.config.retryDelay * attempt);
      }

      try {
        const headers: Record<string, string> = {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        };

        if (this.config.appId) {
          headers['X-App-ID'] = this.config.appId;
        }
        if (this.config.appSecret) {
          headers['X-App-Secret'] = this.config.appSecret;
        }

        const requestInit: RequestInit = {
          method,
          headers,
          signal: AbortSignal.timeout(this.config.timeout),
        };

        if (body && method !== 'GET') {
          requestInit.body = JSON.stringify(body);
        }

        if (this.config.debugMode) {
          console.log(`[DEBUG] ${method} ${url}`);
          if (body) {
            console.log(`[DEBUG] Body:`, body);
          }
        }

        const response = await this.fetch(url, requestInit);

        if (response.status === 429) {
          const retryAfterHeader = response.headers.get('Retry-After');
          const retryAfter = retryAfterHeader ? parseInt(retryAfterHeader, 10) : undefined;
          throw new RateLimitedError('Rate limited', retryAfter);
        }

        if (response.status === 401) {
          throw new UnauthorizedError();
        }

        if (response.status >= 500) {
          throw new ServerError(response.status);
        }

        const responseData = await response.json() as SDKResponse<T>;

        if (responseData.code !== 0) {
          throw new SDKErrorClass(responseData.code, responseData.message);
        }

        this.stats.successfulRequests++;
        return responseData;

      } catch (error) {
        if (error instanceof RateLimitedError || error instanceof UnauthorizedError) {
          if (error instanceof RateLimitedError && error.retryAfter) {
            await this.sleep(error.retryAfter * 1000);
          }
          if (attempt < this.config.maxRetries && !(error instanceof UnauthorizedError)) {
            lastError = error;
            continue;
          }
          throw error;
        }

        if (error instanceof SDKErrorClass) {
          if (attempt < this.config.maxRetries) {
            lastError = error;
            continue;
          }
          throw error;
        }

        if (error instanceof Error) {
          if (error.name === 'TimeoutError' || error.message.includes('timeout')) {
            lastError = new TimeoutError(error.message);
            if (attempt < this.config.maxRetries) {
              continue;
            }
            throw lastError;
          }

          if (error.message.includes('fetch') || error.message.includes('network') || error.message.includes('Failed to fetch')) {
            lastError = new NetworkError(error.message);
            if (attempt < this.config.maxRetries) {
              continue;
            }
            throw lastError;
          }

          lastError = new NetworkError(error.message);
          if (attempt < this.config.maxRetries) {
            continue;
          }
          throw lastError;
        }

        lastError = new NetworkError('Unknown error');
        if (attempt < this.config.maxRetries) {
          continue;
        }
        throw lastError;
      }
    }

    this.stats.failedRequests++;
    this.stats.lastError = lastError?.message || 'Unknown error';
    this.stats.lastErrorTime = new Date().toISOString();

    if (lastError) {
      throw lastError;
    }
    throw new SDKErrorClass(500, 'Unknown error');
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  public async generateImageCaptcha(
    request?: ImageCaptchaRequest
  ): Promise<ImageCaptchaResponse> {
    const params: Record<string, string | number | undefined> = {};

    if (request) {
      if (request.type) {
        params.type = request.type;
      }
      if (request.count) {
        params.count = request.count;
      }
      if (request.customSet) {
        params.custom_set = request.customSet;
      }
      if (request.noiseMode) {
        params.noise_mode = request.noiseMode;
      }
      if (request.lineMode) {
        params.line_mode = request.lineMode;
      }
    }

    const response = await this.doRequest<ImageCaptchaResponse>(
      'GET',
      IMAGE_CAPTCHA_PATH,
      undefined,
      Object.keys(params).length > 0 ? params : undefined
    );

    if (!response.data) {
      throw new SDKErrorClass(500, 'Empty response data');
    }

    return response.data;
  }

  public async verifyImageCaptcha(
    challengeId: string,
    answer: string
  ): Promise<VerifyImageCaptchaResponse> {
    if (!challengeId) {
      throw new InvalidParamsError('challenge_id is required');
    }
    if (!answer) {
      throw new InvalidParamsError('answer is required');
    }

    const request: VerifyImageCaptchaRequest = {
      challenge_id: challengeId,
      answer,
    };

    const response = await this.doRequest<VerifyImageCaptchaResponse>(
      'POST',
      IMAGE_VERIFY_PATH,
      request
    );

    if (!response.data) {
      throw new SDKErrorClass(500, 'Empty response data');
    }

    return response.data;
  }

  public async generateSliderCaptcha(
    request?: SliderCaptchaRequest
  ): Promise<SliderCaptchaResponse> {
    const params: Record<string, string | number | undefined> = {};

    if (request) {
      if (request.width) {
        params.width = request.width;
      }
      if (request.height) {
        params.height = request.height;
      }
    }

    const response = await this.doRequest<SliderCaptchaResponse>(
      'GET',
      SLIDER_CAPTCHA_PATH,
      undefined,
      Object.keys(params).length > 0 ? params : undefined
    );

    if (!response.data) {
      throw new SDKErrorClass(500, 'Empty response data');
    }

    return response.data;
  }

  public async verifySliderCaptcha(
    challengeId: string,
    offset: string
  ): Promise<VerifyCaptchaResponse> {
    if (!challengeId) {
      throw new InvalidParamsError('challenge_id is required');
    }
    if (!offset) {
      throw new InvalidParamsError('offset is required');
    }

    const request: VerifyCaptchaRequest = {
      challenge_id: challengeId,
      action: 'slide',
      data: {
        offset,
      },
    };

    const response = await this.doRequest<VerifyCaptchaResponse>(
      'POST',
      VERIFY_PATH,
      request
    );

    if (!response.data) {
      throw new SDKErrorClass(500, 'Empty response data');
    }

    return response.data;
  }

  public async generateClickCaptcha(
    request?: ClickCaptchaRequest
  ): Promise<ClickCaptchaResponse> {
    const params: Record<string, string | number | undefined> = {};

    if (request) {
      if (request.width) {
        params.width = request.width;
      }
      if (request.height) {
        params.height = request.height;
      }
      if (request.iconCount) {
        params.icon_count = request.iconCount;
      }
    }

    const response = await this.doRequest<ClickCaptchaResponse>(
      'GET',
      CLICK_CAPTCHA_PATH,
      undefined,
      Object.keys(params).length > 0 ? params : undefined
    );

    if (!response.data) {
      throw new SDKErrorClass(500, 'Empty response data');
    }

    return response.data;
  }

  public async verifyClickCaptcha(
    challengeId: string,
    clicks: ClickData[]
  ): Promise<VerifyCaptchaResponse> {
    if (!challengeId) {
      throw new InvalidParamsError('challenge_id is required');
    }
    if (!clicks || clicks.length === 0) {
      throw new InvalidParamsError('clicks is required');
    }

    const request: VerifyCaptchaRequest = {
      challenge_id: challengeId,
      action: 'click',
      data: {
        clicks,
      },
    };

    const response = await this.doRequest<VerifyCaptchaResponse>(
      'POST',
      VERIFY_PATH,
      request
    );

    if (!response.data) {
      throw new SDKErrorClass(500, 'Empty response data');
    }

    return response.data;
  }

  public extractBase64Image(dataUri: string): Buffer {
    if (!dataUri) {
      throw new InvalidParamsError('data_uri is required');
    }

    let prefix: string;
    if (dataUri.startsWith('data:image/png;base64,')) {
      prefix = 'data:image/png;base64,';
    } else if (dataUri.startsWith('data:image/jpeg;base64,')) {
      prefix = 'data:image/jpeg;base64,';
    } else {
      throw new InvalidParamsError('Unsupported image format');
    }

    const base64Data = dataUri.substring(prefix.length);

    if (typeof globalThis.Buffer !== 'undefined') {
      return globalThis.Buffer.from(base64Data, 'base64');
    } else if (typeof globalThis.atob !== 'undefined') {
      const binaryString = globalThis.atob(base64Data);
      const bytes = new Uint8Array(binaryString.length);
      for (let i = 0; i < binaryString.length; i++) {
        bytes[i] = binaryString.charCodeAt(i);
      }
      return bytes;
    } else {
      throw new InvalidParamsError('Base64 decoding not available');
    }
  }

  public getStats(): PoolStats {
    const total = this.stats.totalRequests;
    const success = this.stats.successfulRequests;
    const successRate = total > 0 ? (success / total) * 100 : 0;

    return {
      activeConnections: 0,
      idleConnections: this.config.maxIdleConns,
      totalRequests: total,
      failedRequests: this.stats.failedRequests,
      successfulRequests: success,
      retriedRequests: this.stats.retriedRequests,
      successRate,
      lastError: this.stats.lastError || undefined,
      lastErrorTime: this.stats.lastErrorTime || undefined,
    };
  }

  public setDebugMode(enabled: boolean): void {
    this.config.debugMode = enabled;
  }

  public setTimeout(timeout: number): void {
    this.config.timeout = timeout;
  }

  public setMaxRetries(maxRetries: number): void {
    this.config.maxRetries = maxRetries;
  }

  public setRetryDelay(delay: number): void {
    this.config.retryDelay = delay;
  }

  public getConfig(): Required<Config> {
    return { ...this.config };
  }
}

export function createClient(options?: SDKOptions): CaptchaClient {
  return new CaptchaClient(options);
}

export {
  Config,
  SDKOptions,
  SDKResponse,
  SDKError,
  ImageCaptchaRequest,
  ImageCaptchaResponse,
  SliderCaptchaRequest,
  SliderCaptchaResponse,
  ClickCaptchaRequest,
  ClickCaptchaResponse,
  ClickData,
  VerifyCaptchaRequest,
  VerifyCaptchaResponse,
  VerifyImageCaptchaRequest,
  VerifyImageCaptchaResponse,
  PoolStats,
  CaptchaType,
};
