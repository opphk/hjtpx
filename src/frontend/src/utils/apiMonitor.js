import { recordApiCall, recordError } from './sentry';

const DEFAULT_TIMEOUT = 30000;

export class ApiMonitor {
  constructor() {
    this.pendingRequests = new Map();
    this.requestCounter = 0;
  }

  generateRequestId() {
    this.requestCounter += 1;
    return `req_${this.requestCounter}_${Date.now()}`;
  }

  wrapFetch(originalFetch) {
    return async (input, init = {}) => {
      const requestId = this.generateRequestId();
      const startTime = performance.now();
      
      const endpoint = typeof input === 'string' ? input : input.url;
      const method = init.method || 'GET';
      
      this.pendingRequests.set(requestId, {
        endpoint,
        method,
        startTime,
      });

      try {
        const response = await Promise.race([
          originalFetch(input, {
            ...init,
            headers: {
              ...init.headers,
              'X-Request-ID': requestId,
            },
          }),
          new Promise((_, reject) =>
            setTimeout(() => reject(new Error('Request timeout')), DEFAULT_TIMEOUT)
          ),
        ]);

        const endTime = performance.now();
        const duration = endTime - startTime;
        const status = response.status;

        recordApiCall(endpoint, method, status, duration);

        this.pendingRequests.delete(requestId);

        return response;
      } catch (error) {
        const endTime = performance.now();
        const duration = endTime - startTime;
        
        recordApiCall(endpoint, method, 0, duration);
        recordError(error, {
          endpoint,
          method,
          requestId,
        });

        this.pendingRequests.delete(requestId);
        throw error;
      }
    };
  }

  getPendingRequests() {
    return Array.from(this.pendingRequests.entries()).map(([id, data]) => ({
      requestId: id,
      ...data,
      duration: performance.now() - data.startTime,
    }));
  }

  getPendingCount() {
    return this.pendingRequests.size;
  }
}

export const apiMonitor = new ApiMonitor();

export function setupApiMonitoring() {
  if (typeof window === 'undefined') return;

  const originalFetch = window.fetch;
  window.fetch = apiMonitor.wrapFetch(originalFetch);
}

export function trackApiPerformance(endpoint, startTime, endTime, status) {
  const duration = endTime - startTime;
  recordApiCall(endpoint, 'API', status, duration);
}
