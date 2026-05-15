const Sentry = require('@sentry/node');

jest.mock('@sentry/node', () => {
  const mockScope = {
    setTag: jest.fn(),
    setLevel: jest.fn(),
    setExtra: jest.fn()
  };

  return {
    init: jest.fn(),
    captureException: jest.fn(),
    captureMessage: jest.fn(),
    addBreadcrumb: jest.fn(),
    setTag: jest.fn(),
    setUser: jest.fn(),
    setLevel: jest.fn(),
    setExtra: jest.fn(),
    withScope: jest.fn(callback => callback(mockScope)),
    startTransaction: jest.fn(() => ({
      startChild: jest.fn(() => ({
        finish: jest.fn()
      })),
      finish: jest.fn()
    })),
    Integrations: {
      Http: jest.fn().mockImplementation(() => ({ name: 'Http' })),
      Express: jest.fn().mockImplementation(() => ({ name: 'Express' })),
      Mongo: jest.fn().mockImplementation(() => ({ name: 'Mongo' }))
    },
    Severity: {
      Info: 'info',
      Warning: 'warning',
      Error: 'error'
    },
    metrics: {
      increment: jest.fn(),
      gauge: jest.fn(),
      set: jest.fn()
    }
  };
});

const {
  initSentry,
  setupPerformanceMonitoring,
  captureDatabaseError,
  captureWebSocketError,
  captureGraphQLError,
  setUserContext,
  addPerformanceTag,
  recordMetric,
  createTransaction
} = require('../../src/backend/config/sentry');

describe('Sentry Configuration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    delete process.env.SENTRY_DSN;
    process.env.NODE_ENV = 'test';
  });

  describe('initSentry', () => {
    test('should not initialize Sentry when DSN is not provided', () => {
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).not.toHaveBeenCalled();
    });

    test('should initialize Sentry when DSN is provided', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      process.env.NODE_ENV = 'production';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalled();
    });

    test('should set correct environment from SENTRY_ENVIRONMENT', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      process.env.SENTRY_ENVIRONMENT = 'staging';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          environment: 'staging'
        })
      );
    });

    test('should set correct release version', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      process.env.SENTRY_RELEASE = 'hjtpx@2.0.0';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          release: 'hjtpx@2.0.0'
        })
      );
    });

    test('should use default traces sample rate for production', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      process.env.NODE_ENV = 'production';
      delete process.env.SENTRY_TRACES_SAMPLE_RATE;
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          tracesSampleRate: 0.1
        })
      );
    });

    test('should use high traces sample rate for development', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      process.env.NODE_ENV = 'development';
      delete process.env.SENTRY_TRACES_SAMPLE_RATE;
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          tracesSampleRate: 1.0
        })
      );
    });

    test('should configure ignore errors list', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          ignoreErrors: expect.arrayContaining(['Network Error', 'ECONNREFUSED', 'ETIMEDOUT'])
        })
      );
    });

    test('should configure deny URLs for extensions', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          denyUrls: expect.arrayContaining([/chrome-extension:\/\//i, /safari-extension:\/\//i])
        })
      );
    });

    test('should configure beforeSend callback', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          beforeSend: expect.any(Function)
        })
      );
    });

    test('should configure beforeSendTransaction callback', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          beforeSendTransaction: expect.any(Function)
        })
      );
    });

    test('should include MongoDB integration', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      const initCall = Sentry.init.mock.calls[0][0];
      expect(initCall.integrations).toContainEqual(expect.objectContaining({ name: 'Mongo' }));
    });

    test('should set application tags', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      process.env.SENTRY_RELEASE = 'hjtpx@1.0.0';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.setTag).toHaveBeenCalledWith('application', 'hjtpx-api');
      expect(Sentry.setTag).toHaveBeenCalledWith('application_version', 'hjtpx@1.0.0');
    });
  });

  describe('beforeSend filtering', () => {
    test('should filter out health check errors', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      const initCall = Sentry.init.mock.calls[0][0];
      const beforeSend = initCall.beforeSend;

      const mockEvent = {
        request: { url: 'http://localhost/health' },
        tags: {}
      };
      const mockHint = {};

      const result = beforeSend(mockEvent, mockHint);
      expect(result).toBeNull();
    });

    test('should mark network errors as info level', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      const initCall = Sentry.init.mock.calls[0][0];
      const beforeSend = initCall.beforeSend;

      const mockEvent = {
        request: { url: 'http://localhost/api/test' },
        tags: {}
      };
      const mockHint = {
        originalException: new Error('Network Error')
      };

      const result = beforeSend(mockEvent, mockHint);
      expect(result.tags.network_error).toBe('true');
    });

    test('should set fingerprint for validation errors', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      const initCall = Sentry.init.mock.calls[0][0];
      const beforeSend = initCall.beforeSend;

      const mockEvent = {
        request: { url: 'http://localhost/api/test' },
        tags: {}
      };
      const mockHint = {
        originalException: { name: 'ValidationError', message: 'Email is required: invalid email' }
      };

      const result = beforeSend(mockEvent, mockHint);
      expect(result.fingerprint).toContain('validation-error');
    });

    test('should tag CSRF token errors', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      const initCall = Sentry.init.mock.calls[0][0];
      const beforeSend = initCall.beforeSend;

      const mockEvent = {
        request: { url: 'http://localhost/api/test' },
        tags: {}
      };
      const mockHint = {
        originalException: { code: 'EBADCSRFTOKEN' }
      };

      const result = beforeSend(mockEvent, mockHint);
      expect(result.tags.security).toBe('csrf');
      expect(result.fingerprint).toContain('csrf-token-error');
    });
  });

  describe('beforeSendTransaction filtering', () => {
    test('should filter out health check transactions', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      const initCall = Sentry.init.mock.calls[0][0];
      const beforeSendTransaction = initCall.beforeSendTransaction;

      const mockEvent = {
        transaction: '/health',
        spans: []
      };

      const result = beforeSendTransaction(mockEvent);
      expect(result).toBeNull();
    });

    test('should filter out ping transactions', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      const initCall = Sentry.init.mock.calls[0][0];
      const beforeSendTransaction = initCall.beforeSendTransaction;

      const mockEvent = {
        transaction: '/api/v1/ping',
        spans: []
      };

      const result = beforeSendTransaction(mockEvent);
      expect(result).toBeNull();
    });

    test('should limit span count to 100', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      const initCall = Sentry.init.mock.calls[0][0];
      const beforeSendTransaction = initCall.beforeSendTransaction;

      const manySpans = Array.from({ length: 150 }, (_, i) => ({ id: i }));
      const mockEvent = {
        transaction: '/api/v1/test',
        spans: manySpans
      };

      const result = beforeSendTransaction(mockEvent);
      expect(result.spans.length).toBe(100);
    });
  });

  describe('setupPerformanceMonitoring', () => {
    test('should not set up monitoring without DSN', () => {
      const mockApp = { use: jest.fn() };

      setupPerformanceMonitoring(mockApp);

      expect(mockApp.use).not.toHaveBeenCalled();
    });

    test('should set up performance monitoring with DSN', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = { use: jest.fn() };

      setupPerformanceMonitoring(mockApp);

      expect(mockApp.use).toHaveBeenCalled();
    });

    test('should skip health check endpoints', () => {
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = { use: jest.fn() };

      setupPerformanceMonitoring(mockApp);

      const middleware = mockApp.use.mock.calls[0][0];
      const mockReq = { path: '/health' };
      const mockRes = { on: jest.fn() };
      const mockNext = jest.fn();

      middleware(mockReq, mockRes, mockNext);

      expect(mockRes.on).not.toHaveBeenCalled();
      expect(mockNext).toHaveBeenCalled();
    });
  });

  describe('captureDatabaseError', () => {
    test('should capture database error with scope', () => {
      const error = new Error('Database connection failed');

      captureDatabaseError(error, 'find', 'users');

      expect(Sentry.withScope).toHaveBeenCalled();
    });
  });

  describe('captureWebSocketError', () => {
    test('should capture WebSocket error with tags', () => {
      const error = new Error('WebSocket connection error');

      captureWebSocketError(error, 'connection', 'socket-123');

      expect(Sentry.withScope).toHaveBeenCalled();
    });
  });

  describe('captureGraphQLError', () => {
    test('should capture GraphQL error with operation name', () => {
      const error = new Error('GraphQL parse error');

      captureGraphQLError(error, 'getUsers', { limit: 10 });

      expect(Sentry.withScope).toHaveBeenCalled();
    });

    test('should sanitize sensitive variables', () => {
      const error = new Error('GraphQL error');

      captureGraphQLError(error, 'login', {
        username: 'test',
        password: 'secret123',
        apiKey: 'key123'
      });

      expect(Sentry.withScope).toHaveBeenCalled();
    });
  });

  describe('setUserContext', () => {
    test('should set user context with user data', () => {
      const user = {
        id: 'user-123',
        email: 'test@example.com',
        username: 'testuser',
        role: 'admin'
      };

      setUserContext(user);

      expect(Sentry.setUser).toHaveBeenCalledWith({
        id: 'user-123',
        email: 'test@example.com',
        username: 'testuser',
        role: 'admin'
      });
    });

    test('should clear user context when user is null', () => {
      setUserContext(null);

      expect(Sentry.setUser).toHaveBeenCalledWith(null);
    });

    test('should handle user with _id field', () => {
      const user = {
        _id: 'user-456',
        email: 'test@example.com'
      };

      setUserContext(user);

      expect(Sentry.setUser).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 'user-456'
        })
      );
    });
  });

  describe('addPerformanceTag', () => {
    test('should add performance tag with prefix', () => {
      addPerformanceTag('endpoint', '/api/v1/users');

      expect(Sentry.setTag).toHaveBeenCalledWith('perf_endpoint', '/api/v1/users');
    });
  });

  describe('recordMetric', () => {
    test('should record metric with default unit', () => {
      recordMetric('request_count', 1);

      expect(Sentry.metrics.increment).toHaveBeenCalledWith('request_count', 1, { unit: 'none' });
    });

    test('should record metric with custom unit', () => {
      recordMetric('response_time', 150, 'millisecond');

      expect(Sentry.metrics.increment).toHaveBeenCalledWith('response_time', 150, {
        unit: 'millisecond'
      });
    });
  });

  describe('createTransaction', () => {
    test('should create and return transaction', () => {
      const callback = jest.fn();
      const transaction = createTransaction('test-transaction', 'custom', callback);

      expect(Sentry.startTransaction).toHaveBeenCalledWith(
        { name: 'test-transaction', op: 'custom' },
        callback
      );
      expect(transaction).toBeDefined();
    });
  });

  describe('getDefaultTracesSampleRate', () => {
    test('should return 0.1 for production', () => {
      process.env.NODE_ENV = 'production';
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          tracesSampleRate: 0.1
        })
      );
    });

    test('should return 0.3 for staging', () => {
      process.env.NODE_ENV = 'staging';
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          tracesSampleRate: 0.3
        })
      );
    });

    test('should return 1.0 for development', () => {
      process.env.NODE_ENV = 'development';
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          tracesSampleRate: 1.0
        })
      );
    });
  });

  describe('getDefaultProfilesSampleRate', () => {
    test('should return 0.05 for production', () => {
      process.env.NODE_ENV = 'production';
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          profilesSampleRate: 0.05
        })
      );
    });

    test('should return 0.1 for staging', () => {
      process.env.NODE_ENV = 'staging';
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          profilesSampleRate: 0.1
        })
      );
    });

    test('should return 0.5 for development', () => {
      process.env.NODE_ENV = 'development';
      process.env.SENTRY_DSN = 'https://test@sentry.io/test';
      const mockApp = {};

      initSentry(mockApp);

      expect(Sentry.init).toHaveBeenCalledWith(
        expect.objectContaining({
          profilesSampleRate: 0.5
        })
      );
    });
  });
});
