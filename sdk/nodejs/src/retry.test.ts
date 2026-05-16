import { RetryManager } from './retry';
import { CaptchaError, NetworkError, ServerError, ValidationError } from './errors';

describe('RetryManager', () => {
  describe('constructor', () => {
    it('should create a retry manager with default config', () => {
      const manager = new RetryManager();
      expect(manager).toBeDefined();
    });

    it('should create a retry manager with custom config', () => {
      const manager = new RetryManager({
        maxRetries: 5,
        initialDelayMs: 200,
        maxDelayMs: 5000,
      });
      expect(manager).toBeDefined();
    });
  });

  describe('execute', () => {
    it('should execute a successful function without retries', async () => {
      const manager = new RetryManager();
      const fn = jest.fn().mockResolvedValue('success');

      const result = await manager.execute(fn);

      expect(result).toBe('success');
      expect(fn).toHaveBeenCalledTimes(1);
    });

    it('should retry on retryable errors', async () => {
      const manager = new RetryManager({ maxRetries: 2, initialDelayMs: 1 });
      let attempts = 0;

      const fn = jest.fn().mockImplementation(() => {
        attempts++;
        if (attempts < 3) {
          throw new NetworkError('Network error');
        }
        return 'success';
      });

      const result = await manager.execute(fn);

      expect(result).toBe('success');
      expect(fn).toHaveBeenCalledTimes(3);
    });

    it('should not retry on non-retryable errors', async () => {
      const manager = new RetryManager({ maxRetries: 2, initialDelayMs: 1 });
      const fn = jest.fn().mockRejectedValue(new ValidationError('Invalid input'));

      await expect(manager.execute(fn)).rejects.toThrow(ValidationError);
      expect(fn).toHaveBeenCalledTimes(1);
    });

    it('should throw the last error after max retries', async () => {
      const manager = new RetryManager({ maxRetries: 2, initialDelayMs: 1 });
      const fn = jest.fn().mockRejectedValue(new NetworkError('Network error'));

      await expect(manager.execute(fn)).rejects.toThrow(NetworkError);
      expect(fn).toHaveBeenCalledTimes(3);
    });

    it('should use custom retry checker', async () => {
      const manager = new RetryManager({ maxRetries: 2, initialDelayMs: 1 });
      let attempts = 0;

      const fn = jest.fn().mockImplementation(() => {
        attempts++;
        if (attempts < 3) {
          throw new Error('Custom error');
        }
        return 'success';
      });

      const result = await manager.execute(
        fn,
        (error) => error instanceof Error && error.message === 'Custom error'
      );

      expect(result).toBe('success');
      expect(fn).toHaveBeenCalledTimes(3);
    });
  });
});
