import { parseErrorResponse, AppError, NetworkError, TimeoutError } from './errorClasses';
import { getErrorMessage } from './errorCodes';

const MAX_RETRIES = 3;
const RETRY_DELAY = 1000;

class ErrorHandlerService {
  constructor() {
    this.errorListeners = new Set();
    this.errorLog = [];
    this.maxLogSize = 100;
  }

  addListener(callback) {
    this.errorListeners.add(callback);
    return () => this.errorListeners.delete(callback);
  }

  notifyListeners(error) {
    this.errorListeners.forEach(callback => {
      try {
        callback(error);
      } catch (e) {
        console.error('Error in error listener:', e);
      }
    });
  }

  logError(error) {
    const logEntry = {
      message: error.message,
      code: error.code,
      stack: error.stack,
      timestamp: new Date().toISOString()
    };

    this.errorLog.push(logEntry);
    if (this.errorLog.length > this.maxLogSize) {
      this.errorLog.shift();
    }

    console.error('Error logged:', logEntry);
  }

  handleResponseError(error) {
    const parsedError = parseErrorResponse(error.response);
    this.logError(parsedError);
    this.notifyListeners(parsedError);
    return parsedError;
  }

  handleNetworkError(error) {
    const networkError = new NetworkError(error.message || 'Network error');
    this.logError(networkError);
    this.notifyListeners(networkError);
    return networkError;
  }

  handleTimeoutError(error) {
    const timeoutError = new TimeoutError(error.message || 'Request timed out');
    this.logError(timeoutError);
    this.notifyListeners(timeoutError);
    return timeoutError;
  }

  async withRetry(fn, options = {}) {
    const { retries = MAX_RETRIES, delay = RETRY_DELAY } = options;
    let lastError;

    for (let i = 0; i <= retries; i++) {
      try {
        return await fn();
      } catch (error) {
        lastError = error;

        if (i < retries && this.shouldRetry(error)) {
          await this.delay(delay * (i + 1));
          continue;
        }
      }
    }

    throw lastError;
  }

  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  shouldRetry(error) {
    if (error instanceof TimeoutError) return true;
    if (error instanceof NetworkError) return true;
    if (error.statusCode === 503) return true;
    if (error.statusCode === 429) return true;
    return false;
  }

  getErrorLog() {
    return [...this.errorLog];
  }

  clearErrorLog() {
    this.errorLog = [];
  }

  getErrorSummary() {
    const summary = {
      total: this.errorLog.length,
      byCategory: {},
      byCode: {},
      recent: this.errorLog.slice(-10)
    };

    this.errorLog.forEach(entry => {
      if (entry.code) {
        const category = entry.code.substring(0, 3);
        summary.byCategory[category] = (summary.byCategory[category] || 0) + 1;
        summary.byCode[entry.code] = (summary.byCode[entry.code] || 0) + 1;
      }
    });

    return summary;
  }
}

export const errorHandler = new ErrorHandlerService();

export function handleApiError(error) {
  if (error.response) {
    return errorHandler.handleResponseError(error);
  } else if (error.code === 'ECONNABORTED' || error.message?.includes('timeout')) {
    return errorHandler.handleTimeoutError(error);
  } else {
    return errorHandler.handleNetworkError(error);
  }
}

export default errorHandler;
