import { CaptchaXError, CaptchaResponse } from './types';

const DEFAULT_TIMEOUT = 10000;
const DEFAULT_RETRY_TIMES = 3;

interface RequestOptions extends RequestInit {
  timeout?: number;
  retryTimes?: number;
  deduplicationId?: string;
}

export class HttpClient {
  private baseUrl: string;
  private defaultHeaders: Record<string, string>;
  private defaultTimeout: number;
  private defaultRetryTimes: number;

  constructor(baseUrl: string, timeout: number = DEFAULT_TIMEOUT, retryTimes: number = DEFAULT_RETRY_TIMES) {
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.defaultHeaders = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
    this.defaultTimeout = timeout;
    this.defaultRetryTimes = retryTimes;
  }

  public setHeader(key: string, value: string): void {
    this.defaultHeaders[key] = value;
  }

  public setHeaders(headers: Record<string, string>): void {
    this.defaultHeaders = { ...this.defaultHeaders, ...headers };
  }

  private async fetchWithTimeout(
    url: string,
    options: RequestInit,
    timeout: number
  ): Promise<Response> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);

    try {
      const response = await fetch(url, {
        ...options,
        signal: controller.signal,
      });
      return response;
    } finally {
      clearTimeout(timeoutId);
    }
  }

  private async request<T>(
    method: string,
    endpoint: string,
    body?: unknown,
    options: Partial<RequestOptions> = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const timeout = options.timeout ?? this.defaultTimeout;
    const retryTimes = options.retryTimes ?? this.defaultRetryTimes;
    const deduplicationId = options.deduplicationId;

    const headers: Record<string, string> = { ...this.defaultHeaders };
    if (options.headers) {
      Object.assign(headers, options.headers);
    }

    const requestInit: RequestInit = {
      method,
      headers,
      ...options,
    };

    if (body && method !== 'GET') {
      requestInit.body = JSON.stringify(body);
    }

    let lastError: Error | null = null;

    for (let attempt = 0; attempt <= retryTimes; attempt++) {
      try {
        const requestHeaders = { ...headers };
        if (deduplicationId) {
          requestHeaders['X-Deduplication-ID'] = deduplicationId;
        }

        const response = await this.fetchWithTimeout(
          url,
          { ...requestInit, headers: requestHeaders },
          timeout
        );

        const data = await response.json() as CaptchaResponse<T>;

        if (!response.ok) {
          const errorMessage = data.message || `HTTP error: ${response.status}`;
          throw new CaptchaXError(
            errorMessage,
            data.code || response.status,
            response.status,
            data
          );
        }

        return data.data;
      } catch (error) {
        if (error instanceof CaptchaXError) {
          if (error.statusCode >= 500 && attempt < retryTimes) {
            lastError = error;
            await this.delay(Math.pow(2, attempt) * 100);
            continue;
          }
          throw error;
        }

        if (error instanceof Error) {
          if (error.name === 'AbortError') {
            throw new CaptchaXError('Request timeout', 408, 408);
          }

          if (attempt < retryTimes) {
            lastError = error;
            await this.delay(Math.pow(2, attempt) * 100);
            continue;
          }

          throw new CaptchaXError(
            `Network error: ${error.message}`,
            0,
            0,
            error
          );
        }

        lastError = new Error('Unknown error');
      }
    }

    throw new CaptchaXError(
      `Request failed after ${retryTimes + 1} attempts: ${lastError?.message}`,
      0,
      0,
      lastError
    );
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  public async get<T>(endpoint: string, options?: Partial<RequestOptions>): Promise<T> {
    return this.request<T>('GET', endpoint, undefined, options);
  }

  public async post<T>(
    endpoint: string,
    body: unknown,
    options?: Partial<RequestOptions>
  ): Promise<T> {
    return this.request<T>('POST', endpoint, body, options);
  }

  public async put<T>(
    endpoint: string,
    body: unknown,
    options?: Partial<RequestOptions>
  ): Promise<T> {
    return this.request<T>('PUT', endpoint, body, options);
  }

  public async delete<T>(
    endpoint: string,
    options?: Partial<RequestOptions>
  ): Promise<T> {
    return this.request<T>('DELETE', endpoint, undefined, options);
  }
}
