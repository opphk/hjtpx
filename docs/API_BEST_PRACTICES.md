# HJTPX API 最佳实践指南

本文档提供了使用 HJTPX API 的最佳实践建议，帮助开发者构建安全、高效、可靠的应用程序。

## 目录

- [认证与授权](#认证与授权)
- [请求格式](#请求格式)
- [错误处理](#错误处理)
- [性能优化](#性能优化)
- [安全性建议](#安全性建议)
- [调试与监控](#调试与监控)
- [代码组织](#代码组织)

---

## 认证与授权

### 安全地存储 Token

**❌ 不推荐的做法**
```javascript
// 错误：将 Token 存储在 localStorage
localStorage.setItem('token', response.data.token);
```

**✅ 推荐的做法**
```javascript
// 方式 1: 使用 HttpOnly Cookie（服务器端设置）
// 服务器端设置：
// res.cookie('token', token, { httpOnly: true, secure: true });

// 方式 2: 使用 sessionStorage（仅当前会话）
sessionStorage.setItem('token', token);

// 方式 3: 使用内存变量 + 刷新机制
class TokenManager {
  #token = null;
  #refreshTimeout = null;

  setToken(token) {
    this.#token = token;
    this.#scheduleRefresh(token);
  }

  getToken() {
    return this.#token;
  }

  #scheduleRefresh(token) {
    // 解析 JWT 获取过期时间
    const payload = JSON.parse(atob(token.split('.')[1]));
    const expiresIn = payload.exp * 1000 - Date.now();
    
    // 在过期前 1 小时刷新
    const refreshTime = expiresIn - 3600000;
    
    if (refreshTime > 0) {
      this.#refreshTimeout = setTimeout(
        () => this.refresh(),
        refreshTime
      );
    }
  }
}
```

### Token 刷新策略

```javascript
class AuthClient {
  constructor() {
    this.tokenManager = new TokenManager();
  }

  async request(url, options = {}) {
    const token = this.tokenManager.getToken();
    
    const response = await fetch(url, {
      ...options,
      headers: {
        ...options.headers,
        'Authorization': token ? `Bearer ${token}` : undefined
      }
    });

    if (response.status === 401) {
      // Token 过期，尝试刷新
      const refreshed = await this.tokenManager.refresh();
      
      if (refreshed) {
        // 重试原请求
        return this.request(url, options);
      }
      
      // 刷新失败，跳转登录
      window.location.href = '/login';
    }

    return response;
  }
}
```

### 权限检查

```javascript
class PermissionChecker {
  static canAccess(userRole, resource, action) {
    const permissions = {
      admin: { users: ['read', 'create', 'update', 'delete'] },
      moderator: { users: ['read', 'update'], notifications: ['read', 'send'] },
      user: { users: ['read'], notifications: ['read'] }
    };

    const rolePermissions = permissions[userRole];
    if (!rolePermissions) return false;

    const resourcePermissions = rolePermissions[resource];
    if (!resourcePermissions) return false;

    return resourcePermissions.includes(action);
  }

  static requirePermission(userRole, resource, action) {
    if (!this.canAccess(userRole, resource, action)) {
      throw new Error(`Permission denied: ${action} on ${resource}`);
    }
  }
}
```

---

## 请求格式

### 请求头设置

```javascript
class ApiClient {
  constructor(baseUrl) {
    this.baseUrl = baseUrl;
  }

  getDefaultHeaders() {
    return {
      'Content-Type': 'application/json',
      'Accept': 'application/json'
    };
  }

  async request(endpoint, options = {}) {
    const url = `${this.baseUrl}${endpoint}`;
    
    const response = await fetch(url, {
      ...options,
      headers: {
        ...this.getDefaultHeaders(),
        ...options.headers
      }
    });

    return response;
  }
}
```

### 请求体格式化

```javascript
// 序列化请求数据
function serializeRequestBody(data) {
  if (data instanceof FormData) {
    return data; // 不设置 Content-Type，让浏览器处理
  }
  
  if (data instanceof URLSearchParams) {
    return data;
  }
  
  return JSON.stringify(data);
}

// 清理对象中的空值
function cleanRequestData(data) {
  return Object.fromEntries(
    Object.entries(data)
      .filter(([_, value]) => value !== null && value !== undefined && value !== '')
  );
}

// 使用示例
async function updateUser(userId, data) {
  const cleanedData = cleanRequestData(data);
  
  return api.request(`/users/${userId}`, {
    method: 'PUT',
    body: JSON.stringify(cleanedData)
  });
}
```

### 查询参数构建

```javascript
// 构建查询字符串
function buildQueryString(params) {
  const searchParams = new URLSearchParams();
  
  Object.entries(params).forEach(([key, value]) => {
    if (Array.isArray(value)) {
      value.forEach(v => searchParams.append(key, v));
    } else if (value !== undefined && value !== null) {
      searchParams.append(key, value);
    }
  });
  
  return searchParams.toString();
}

// 使用示例
function getUsers(filters) {
  const queryString = buildQueryString({
    page: filters.page || 1,
    limit: filters.limit || 20,
    status: filters.status,
    role: filters.roles // 数组
  });
  
  return api.request(`/users?${queryString}`);
}
```

---

## 错误处理

### 分层错误处理

```javascript
// 1. HTTP 状态码处理
function handleHttpError(response) {
  switch (response.status) {
    case 400:
      return handleBadRequest(response);
    case 401:
      return handleUnauthorized();
    case 403:
      return handleForbidden();
    case 404:
      return handleNotFound();
    case 429:
      return handleRateLimit(response);
    case 500:
    case 502:
    case 503:
      return handleServerError();
    default:
      throw new Error(`Unhandled HTTP status: ${response.status}`);
  }
}

// 2. 业务错误码处理
function handleBusinessError(error) {
  const errorHandlers = {
    'VALIDATION_ERROR': handleValidationError,
    'UNAUTHORIZED': handleAuthError,
    'FORBIDDEN': handlePermissionError,
    'NOT_FOUND': handleNotFoundError
  };

  const handler = errorHandlers[error.code];
  if (handler) {
    return handler(error);
  }

  // 默认错误处理
  showErrorNotification(error.message);
}

// 3. 网络错误处理
function handleNetworkError(error) {
  if (error.name === 'TypeError' && error.message.includes('fetch')) {
    return {
      code: 'NETWORK_ERROR',
      message: '网络连接失败，请检查您的网络设置'
    };
  }
  
  return {
    code: 'UNKNOWN_ERROR',
    message: '发生未知错误，请稍后重试'
  };
}
```

### 统一的错误响应

```javascript
class ApiError extends Error {
  constructor(code, message, details = null, statusCode = 500) {
    super(message);
    this.code = code;
    this.details = details;
    this.statusCode = statusCode;
  }

  static fromResponse(response) {
    return new ApiError(
      response.error?.code || 'UNKNOWN_ERROR',
      response.error?.message || 'An error occurred',
      response.error?.details,
      response.status
    );
  }
}

// 使用示例
async function safeRequest(requestFn) {
  try {
    const response = await requestFn();
    const data = await response.json();
    
    if (!data.success) {
      throw ApiError.fromResponse(data);
    }
    
    return data;
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    
    // 转换其他错误
    throw new ApiError('NETWORK_ERROR', error.message);
  }
}
```

### 用户友好的错误提示

```javascript
const errorMessages = {
  'VALIDATION_ERROR': {
    'email': {
      'string.email': '请输入有效的邮箱地址',
      'any.required': '邮箱地址不能为空'
    },
    'password': {
      'string.min': '密码至少需要 8 个字符',
      'string.pattern.base': '密码必须包含大小写字母和数字'
    }
  },
  'UNAUTHORIZED': '登录已过期，请重新登录',
  'FORBIDDEN': '您没有权限执行此操作',
  'TOO_MANY_REQUESTS': '请求过于频繁，请稍后再试',
  'NOT_FOUND': '请求的资源不存在',
  'NETWORK_ERROR': '网络连接失败，请检查网络后重试',
  'SERVER_ERROR': '服务器繁忙，请稍后重试'
};

function getUserFriendlyMessage(error) {
  if (error.details && error.details.length > 0) {
    return error.details
      .map(d => {
        const fieldMessages = errorMessages.VALIDATION_ERROR[error.field];
        if (fieldMessages && fieldMessages[d.type]) {
          return fieldMessages[d.type];
        }
        return d.message;
      })
      .join('\n');
  }

  return errorMessages[error.code] || error.message;
}
```

---

## 性能优化

### 请求缓存策略

```javascript
class CacheManager {
  constructor(defaultTTL = 5 * 60 * 1000) {
    this.cache = new Map();
    this.defaultTTL = defaultTTL;
  }

  set(key, data, ttl = this.defaultTTL) {
    this.cache.set(key, {
      data,
      expiresAt: Date.now() + ttl
    });
  }

  get(key) {
    const cached = this.cache.get(key);
    
    if (!cached) return null;
    
    if (Date.now() > cached.expiresAt) {
      this.cache.delete(key);
      return null;
    }
    
    return cached.data;
  }

  invalidate(pattern) {
    for (const key of this.cache.keys()) {
      if (key.includes(pattern)) {
        this.cache.delete(key);
      }
    }
  }

  clear() {
    this.cache.clear();
  }
}

// 缓存使用示例
const cacheManager = new CacheManager();

async function getCachedUsers() {
  const cacheKey = 'users_list';
  let users = cacheManager.get(cacheKey);
  
  if (!users) {
    const response = await api.getUsers();
    users = response.data;
    cacheManager.set(cacheKey, users, 2 * 60 * 1000); // 2 分钟缓存
  }
  
  return users;
}
```

### 防抖和节流

```javascript
// 防抖：等待用户停止输入后再发送请求
function debounce(fn, delay = 300) {
  let timeoutId;
  
  return function (...args) {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => fn.apply(this, args), delay);
  };
}

// 节流：限制请求频率
function throttle(fn, limit = 1000) {
  let lastCall = 0;
  
  return function (...args) {
    const now = Date.now();
    
    if (now - lastCall >= limit) {
      lastCall = now;
      return fn.apply(this, args);
    }
  };
}

// 使用示例
class SearchComponent {
  constructor() {
    this.search = debounce(this.performSearch.bind(this), 300);
    this.submit = throttle(this.submitForm.bind(this), 1000);
  }

  onSearchInput(event) {
    this.search(event.target.value);
  }

  async performSearch(query) {
    const results = await api.search(query);
    this.displayResults(results);
  }
}
```

### 并发请求优化

```javascript
// 并发请求限制
class RequestPool {
  constructor(maxConcurrent = 5) {
    this.maxConcurrent = maxConcurrent;
    this.running = 0;
    this.queue = [];
  }

  async add(requestFn) {
    if (this.running >= this.maxConcurrent) {
      return new Promise((resolve, reject) => {
        this.queue.push({ requestFn, resolve, reject });
      });
    }

    return this.execute(requestFn);
  }

  async execute(requestFn) {
    this.running++;
    
    try {
      const result = await requestFn();
      return result;
    } finally {
      this.running--;
      this.processQueue();
    }
  }

  async processQueue() {
    if (this.queue.length > 0 && this.running < this.maxConcurrent) {
      const { requestFn, resolve, reject } = this.queue.shift();
      this.execute(requestFn).then(resolve).catch(reject);
    }
  }
}

// 批量请求优化
async function batchGetUsers(userIds) {
  const pool = new RequestPool(5);
  
  const promises = userIds.map(id => 
    pool.add(() => api.getUser(id))
  );
  
  return Promise.all(promises);
}
```

### 响应数据处理

```javascript
// 响应数据规范化
function normalizeResponse(response) {
  return {
    success: response.success,
    data: response.data,
    message: response.message,
    meta: {
      timestamp: response.timestamp,
      version: response.version
    }
  };
}

// 数据转换管道
function createDataPipeline(...transforms) {
  return (data) => transforms.reduce((acc, fn) => fn(acc), data);
}

const userPipeline = createDataPipeline(
  // 移除敏感字段
  (user) => {
    const { password, ...safeUser } = user;
    return safeUser;
  },
  // 格式化日期
  (user) => ({
    ...user,
    createdAt: new Date(user.createdAt).toLocaleDateString(),
    updatedAt: new Date(user.updatedAt).toLocaleDateString()
  }),
  // 排序字段
  (user) => ({
    id: user.id,
    email: user.email,
    name: user.name,
    role: user.role,
    createdAt: user.createdAt,
    updatedAt: user.updatedAt
  })
);

// 使用示例
async function getFormattedUser(userId) {
  const response = await api.getUser(userId);
  return userPipeline(response.data);
}
```

---

## 安全性建议

### 输入验证

```javascript
// 前端输入验证（不仅用于展示，还用于防止 XSS）
function sanitizeInput(input) {
  if (typeof input !== 'string') return input;
  
  return input
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#x27;')
    .replace(/\//g, '&#x2F;');
}

// 验证邮箱格式
function isValidEmail(email) {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
}

// 验证密码强度
function validatePasswordStrength(password) {
  const checks = {
    length: password.length >= 8,
    uppercase: /[A-Z]/.test(password),
    lowercase: /[a-z]/.test(password),
    number: /[0-9]/.test(password)
  };
  
  const passedChecks = Object.values(checks).filter(Boolean).length;
  
  return {
    isValid: passedChecks >= 3,
    strength: passedChecks === 4 ? 'strong' : passedChecks >= 2 ? 'medium' : 'weak',
    checks
  };
}
```

### CSRF 防护

```javascript
// 获取 CSRF Token
function getCsrfToken() {
  const metaTag = document.querySelector('meta[name="csrf-token"]');
  return metaTag ? metaTag.content : null;
}

// 在请求中包含 CSRF Token
async function securedRequest(url, options = {}) {
  const csrfToken = getCsrfToken();
  
  if (csrfToken) {
    options.headers = {
      ...options.headers,
      'X-CSRF-Token': csrfToken
    };
  }
  
  return fetch(url, options);
}
```

### 敏感数据处理

```javascript
// 脱敏显示
function maskSensitiveData(data, fields = ['email', 'phone']) {
  const masked = { ...data };
  
  fields.forEach(field => {
    if (masked[field]) {
      if (field === 'email') {
        const [local, domain] = masked[field].split('@');
        masked[field] = `${local[0]}***@${domain}`;
      } else if (field === 'phone') {
        masked[field] = masked[field].replace(/(\d{3})\d{4}(\d{4})/, '$1****$2');
      }
    }
  });
  
  return masked;
}

// 使用示例
const safeUser = maskSensitiveData(user, ['email']);
console.log(safeUser.email); // j***@example.com
```

---

## 调试与监控

### 请求日志

```javascript
class DebugLogger {
  constructor(enabled = process.env.NODE_ENV === 'development') {
    this.enabled = enabled;
  }

  logRequest(url, options, startTime) {
    if (!this.enabled) return;

    console.group(`📤 REQUEST: ${options.method || 'GET'} ${url}`);
    console.log('Time:', new Date().toISOString());
    console.log('Headers:', options.headers);
    if (options.body) {
      console.log('Body:', JSON.parse(options.body));
    }
    console.log('Duration:', Date.now() - startTime, 'ms');
    console.groupEnd();
  }

  logResponse(response, data, startTime) {
    if (!this.enabled) return;

    console.group(`📥 RESPONSE: ${response.url}`);
    console.log('Status:', response.status);
    console.log('Duration:', Date.now() - startTime, 'ms');
    console.log('Data:', data);
    console.groupEnd();
  }

  logError(error, context) {
    if (!this.enabled) return;

    console.error(`❌ ERROR${context ? ` (${context})` : ''}:`, {
      message: error.message,
      code: error.code,
      stack: error.stack
    });
  }
}

const logger = new DebugLogger();

// 中间件使用
async function loggedRequest(url, options) {
  const startTime = Date.now();
  logger.logRequest(url, options, startTime);

  try {
    const response = await fetch(url, options);
    const data = await response.json();
    logger.logResponse(response, data, startTime);
    return data;
  } catch (error) {
    logger.logError(error, `${options.method} ${url}`);
    throw error;
  }
}
```

### 性能监控

```javascript
class PerformanceMonitor {
  constructor() {
    this.metrics = {
      requests: 0,
      errors: 0,
      totalDuration: 0,
      byEndpoint: {}
    };
  }

  recordRequest(endpoint, duration, success = true) {
    this.metrics.requests++;
    this.metrics.totalDuration += duration;
    
    if (!success) {
      this.metrics.errors++;
    }

    if (!this.metrics.byEndpoint[endpoint]) {
      this.metrics.byEndpoint[endpoint] = {
        count: 0,
        totalDuration: 0,
        errors: 0
      };
    }

    const endpointMetrics = this.metrics.byEndpoint[endpoint];
    endpointMetrics.count++;
    endpointMetrics.totalDuration += duration;
    if (!success) {
      endpointMetrics.errors++;
    }
  }

  getStats() {
    const avgDuration = this.metrics.requests > 0
      ? this.metrics.totalDuration / this.metrics.requests
      : 0;

    return {
      ...this.metrics,
      averageDuration: Math.round(avgDuration),
      errorRate: this.metrics.requests > 0
        ? (this.metrics.errors / this.metrics.requests * 100).toFixed(2) + '%'
        : '0%'
    };
  }

  getEndpointStats(endpoint) {
    const stats = this.metrics.byEndpoint[endpoint];
    if (!stats) return null;

    return {
      ...stats,
      averageDuration: Math.round(stats.totalDuration / stats.count),
      errorRate: ((stats.errors / stats.count) * 100).toFixed(2) + '%'
    };
  }
}

const monitor = new PerformanceMonitor();
```

---

## 代码组织

### API 客户端组织

```javascript
// 1. 配置文件
const config = {
  baseUrl: process.env.API_BASE_URL || 'http://localhost:3000/api/v1',
  timeout: 30000,
  retryAttempts: 3
};

// 2. 错误类定义
class ApiException extends Error {
  constructor(message, code, statusCode, details = null) {
    super(message);
    this.name = 'ApiException';
    this.code = code;
    this.statusCode = statusCode;
    this.details = details;
  }
}

// 3. HTTP 客户端
class HttpClient {
  // ... 实现
}

// 4. API 服务模块
const authService = {
  login: (credentials) => httpClient.post('/auth/login', credentials),
  register: (userData) => httpClient.post('/auth/register', userData),
  refresh: (token) => httpClient.post('/auth/refresh', { token })
};

const userService = {
  getCurrent: () => httpClient.get('/users/me'),
  getById: (id) => httpClient.get(`/users/${id}`),
  update: (id, data) => httpClient.put(`/users/${id}`, data)
};

const notificationService = {
  list: (params) => httpClient.get('/notifications', params),
  markAsRead: (id) => httpClient.put(`/notifications/${id}/read`),
  markAllAsRead: () => httpClient.put('/notifications/read-all')
};

// 5. 导出
export {
  config,
  ApiException,
  HttpClient,
  authService,
  userService,
  notificationService
};
```

### React Hooks 组织

```javascript
// hooks/useAuth.js
import { useState, useEffect, useCallback } from 'react';
import { authService } from '../services';

export function useAuth() {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    checkAuth();
  }, []);

  const checkAuth = async () => {
    try {
      const token = localStorage.getItem('token');
      if (token) {
        const response = await authService.verify(token);
        if (response.data.valid) {
          setUser(response.data.user);
        }
      }
    } catch (error) {
      console.error('Auth check failed:', error);
    } finally {
      setLoading(false);
    }
  };

  const login = useCallback(async (email, password) => {
    const response = await authService.login({ email, password });
    localStorage.setItem('token', response.data.token);
    setUser(response.data.user);
    return response;
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('token');
    setUser(null);
  }, []);

  return { user, loading, login, logout };
}

// hooks/useNotifications.js
import { useState, useEffect } from 'react';
import { notificationService } from '../services';

export function useNotifications(options = {}) {
  const [notifications, setNotifications] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetchNotifications();
  }, [options.page, options.status]);

  const fetchNotifications = async () => {
    try {
      setLoading(true);
      const response = await notificationService.list(options);
      setNotifications(response.data);
      setError(null);
    } catch (err) {
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  const markAsRead = async (id) => {
    await notificationService.markAsRead(id);
    setNotifications(prev =>
      prev.map(n => n.id === id ? { ...n, status: 'read' } : n)
    );
  };

  return { notifications, loading, error, markAsRead, refetch: fetchNotifications };
}
```

---

## 测试建议

### API Mock

```javascript
// mocks/apiMocks.js
export const mockApiResponses = {
  '/api/v1/auth/login': {
    success: true,
    data: {
      user: {
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        role: 'user'
      },
      token: 'mock-jwt-token',
      expiresIn: '7d'
    }
  },
  '/api/v1/users/me': {
    success: true,
    data: {
      id: 1,
      email: 'test@example.com',
      name: 'Test User',
      role: 'user'
    }
  }
};

// Mock 服务器设置
export function setupMockServer() {
  beforeEach(() => {
    jest.mock('node-fetch', () => async (url) => {
      const mockResponse = mockApiResponses[url];
      if (mockResponse) {
        return {
          ok: true,
          json: async () => mockResponse,
          status: 200
        };
      }
      return {
        ok: false,
        status: 404,
        json: async () => ({ success: false, error: { code: 'NOT_FOUND' } })
      };
    });
  });
}
```

---

## 部署注意事项

### 环境变量

```bash
# .env.production
NODE_ENV=production
API_BASE_URL=https://api.hjtpx.com
API_VERSION=v1

# 安全相关
JWT_SECRET=<从密钥管理服务获取>
SESSION_SECRET=<从密钥管理服务获取>

# 性能相关
API_TIMEOUT=30000
MAX_RETRIES=3
```

### 生产环境检查清单

```javascript
const productionChecklist = {
  security: [
    '✓ HTTPS 已启用',
    '✓ HTTP Strict Transport Security 已配置',
    '✓ CORS 策略已正确设置',
    '✓ CSRF 防护已启用',
    '✓ 敏感数据已加密存储',
    '✓ API 密钥已从环境变量加载'
  ],
  performance: [
    '✓ 响应压缩已启用',
    '✓ 静态资源已CDN分发',
    '✓ 数据库连接池已配置',
    '✓ 缓存策略已实施'
  ],
  monitoring: [
    '✓ 日志记录已配置',
    '✓ 性能监控已启用',
    '✓ 错误追踪已集成',
    '✓ 健康检查端点已配置'
  ]
};
```

---

## 更多资源

- [API 文档](./API_REFERENCE.md)
- [错误码对照表](./API_ERROR_CODES.md)
- [使用示例](./API_EXAMPLES.md)

---

**文档版本**: 1.0.0  
**最后更新**: 2026-05-15  
**维护团队**: HJTPX Development Team
