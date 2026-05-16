import { RetryConfig } from './types';
import { CaptchaError } from './errors';

const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxRetries: 3,
  initialDelayMs: 100,
  maxDelayMs: 10000,
  retryableStatuses: [429, 500, 502, 503, 504],
};

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function exponentialBackoff(
  attempt: number,
  initialDelay: number,
  maxDelay: number
): number {
  const delay = initialDelay * Math.pow(2, attempt);
  return Math.min(delay, maxDelay);
}

export class RetryManager {
  private config: RetryConfig;

  constructor(config?: RetryConfig) {
    this.config = { ...DEFAULT_RETRY_CONFIG, ...config };
  }

  async execute<T>(
    fn: () => Promise<T>,
    shouldRetry?: (error: unknown) => boolean
  ): Promise<T> {
    let lastError: unknown;
    const maxRetries = this.config.maxRetries ?? DEFAULT_RETRY_CONFIG.maxRetries!;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        return await fn();
      } catch (error) {
        lastError = error;

        const isRetryable = shouldRetry
          ? shouldRetry(error)
          : this.isRetryableError(error);

        if (!isRetryable || attempt >= maxRetries) {
          break;
        }

        const delayMs = exponentialBackoff(
          attempt,
          this.config.initialDelayMs ?? DEFAULT_RETRY_CONFIG.initialDelayMs!,
          this.config.maxDelayMs ?? DEFAULT_RETRY_CONFIG.maxDelayMs!
        );

        await delay(delayMs);
      }
    }

    throw lastError;
  }

  private isRetryableError(error: unknown): boolean {
    if (error instanceof CaptchaError) {
      return error.retryable;
    }

    if (error instanceof Error) {
      const message = error.message.toLowerCase();
      return (
        message.includes('timeout') ||
        message.includes('network') ||
        message.includes('econnreset') ||
        message.includes('econnrefused') ||
        message.includes('socket hang up')
      );
    }

    return false;
  }
}
