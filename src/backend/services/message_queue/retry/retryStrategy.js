const config = require('../../config/messageQueue');

class RetryStrategy {
  constructor(options = {}) {
    this.maxAttempts = options.maxAttempts || config.retry.maxAttempts;
    this.initialDelay = options.initialDelay || config.retry.initialDelay;
    this.maxDelay = options.maxDelay || config.retry.maxDelay;
    this.backoffMultiplier = options.backoffMultiplier || config.retry.backoffMultiplier;
    this.jitter = options.jitter !== undefined ? options.jitter : config.retry.jitter;
  }

  calculateDelay(attempt) {
    if (attempt <= 0) {
      return this.initialDelay;
    }

    let delay = this.initialDelay * Math.pow(this.backoffMultiplier, attempt - 1);
    delay = Math.min(delay, this.maxDelay);

    if (this.jitter) {
      const jitterFactor = 0.5 + Math.random() * 0.5;
      delay = Math.floor(delay * jitterFactor);
    }

    return delay;
  }

  shouldRetry(attempt, error) {
    if (attempt >= this.maxAttempts) {
      return false;
    }

    if (error && error.noRetry) {
      return false;
    }

    return true;
  }

  getNextDelay(attempt) {
    return this.calculateDelay(attempt);
  }

  reset() {
    return new RetryStrategy({
      maxAttempts: this.maxAttempts,
      initialDelay: this.initialDelay,
      maxDelay: this.maxDelay,
      backoffMultiplier: this.backoffMultiplier,
      jitter: this.jitter
    });
  }
}

class RetryManager {
  constructor() {
    this.strategies = new Map();
    this.retryHistory = new Map();
    this.defaultStrategy = new RetryStrategy();
  }

  registerStrategy(name, strategy) {
    this.strategies.set(name, strategy);
  }

  getStrategy(name) {
    return this.strategies.get(name) || this.defaultStrategy;
  }

  async executeWithRetry(operation, options = {}) {
    const strategy = options.strategy ? this.getStrategy(options.strategy) : this.defaultStrategy;

    const context = {
      operationName: options.operationName || 'unknown',
      attempt: 0,
      maxAttempts: options.maxAttempts || strategy.maxAttempts,
      startTime: Date.now(),
      errors: []
    };

    while (context.attempt < context.maxAttempts) {
      try {
        const result = await operation(context);
        this.recordSuccess(context);
        return result;
      } catch (error) {
        context.attempt++;
        context.errors.push({
          attempt: context.attempt,
          error: error.message,
          timestamp: Date.now()
        });

        if (!strategy.shouldRetry(context.attempt, error)) {
          this.recordFailure(context);
          throw error;
        }

        if (context.attempt < context.maxAttempts) {
          const delay = strategy.calculateDelay(context.attempt);
          console.log(
            `[RetryManager] ${context.operationName} failed, retrying in ${delay}ms (attempt ${context.attempt}/${context.maxAttempts})`
          );
          await this.sleep(delay);
        }
      }
    }

    this.recordFailure(context);
    const finalError = new Error(
      `Max retry attempts (${context.maxAttempts}) exceeded for ${context.operationName}`
    );
    finalError.context = context;
    throw finalError;
  }

  recordSuccess(context) {
    const key = context.operationName;
    if (!this.retryHistory.has(key)) {
      this.retryHistory.set(key, { successes: 0, failures: 0, attempts: 0 });
    }
    const stats = this.retryHistory.get(key);
    stats.successes++;
    stats.attempts++;
    stats.lastSuccess = Date.now();
  }

  recordFailure(context) {
    const key = context.operationName;
    if (!this.retryHistory.has(key)) {
      this.retryHistory.set(key, { successes: 0, failures: 0, attempts: 0 });
    }
    const stats = this.retryHistory.get(key);
    stats.failures++;
    stats.attempts++;
    stats.lastFailure = Date.now();
  }

  getStats(operationName) {
    if (operationName) {
      return this.retryHistory.get(operationName) || null;
    }
    return Object.fromEntries(this.retryHistory);
  }

  resetStats(operationName) {
    if (operationName) {
      this.retryHistory.delete(operationName);
    } else {
      this.retryHistory.clear();
    }
  }

  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

const retryManager = new RetryManager();

retryManager.registerStrategy(
  'aggressive',
  new RetryStrategy({
    maxAttempts: 3,
    initialDelay: 500,
    maxDelay: 10000,
    backoffMultiplier: 2
  })
);

retryManager.registerStrategy(
  'conservative',
  new RetryStrategy({
    maxAttempts: 5,
    initialDelay: 1000,
    maxDelay: 60000,
    backoffMultiplier: 2,
    jitter: true
  })
);

retryManager.registerStrategy(
  'noRetry',
  new RetryStrategy({
    maxAttempts: 1,
    initialDelay: 0,
    maxDelay: 0,
    backoffMultiplier: 1,
    jitter: false
  })
);

module.exports = {
  RetryStrategy,
  RetryManager,
  retryManager
};
