# HJTPX API 使用示例

本文档提供 HJTPX API 的详细使用示例，包括 curl 命令和 JavaScript/fetch 示例。

## 目录

- [认证 API](#认证-api)
- [用户管理 API](#用户管理-api)
- [通知 API](#通知-api)
- [健康检查 API](#健康检查-api)

---

## 认证 API

### 用户登录

#### curl 示例

```bash
# 基本登录
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123"
  }'

# 完整示例（包含错误处理）
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123"
  }' \
  -w "\nHTTP Status: %{http_code}\n"
```

#### JavaScript/fetch 示例

```javascript
// 基础登录请求
async function login(email, password) {
  try {
    const response = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json'
      },
      body: JSON.stringify({ email, password }),
      credentials: 'include'
    });

    const data = await response.json();

    if (!response.ok) {
      throw new Error(data.error?.message || 'Login failed');
    }

    // 保存 token
    localStorage.setItem('token', data.data.token);

    return data;
  } catch (error) {
    console.error('Login error:', error);
    throw error;
  }
}

// 使用示例
login('user@example.com', 'SecurePass123')
  .then(result => {
    console.log('Login successful:', result);
    console.log('User:', result.data.user);
    console.log('Token:', result.data.token);
  })
  .catch(err => {
    console.error('Failed to login:', err.message);
  });
```

```typescript
// TypeScript 版本
interface LoginRequest {
  email: string;
  password: string;
}

interface LoginResponse {
  success: boolean;
  data: {
    user: {
      id: number;
      email: string;
      name: string;
      role: 'user' | 'admin' | 'moderator';
    };
    token: string;
    expiresIn: string;
  };
  message: string;
  timestamp: string;
}

async function login(email: string, password: string): Promise<LoginResponse> {
  const response = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ email, password })
  });

  if (!response.ok) {
    throw new Error(`Login failed: ${response.status}`);
  }

  return response.json();
}
```

### 用户注册

#### curl 示例

```bash
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "name": "New User",
    "password": "SecurePass123"
  }'
```

#### JavaScript/fetch 示例

```javascript
async function register(email, name, password) {
  const response = await fetch('/api/v1/auth/register', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ email, name, password })
  });

  const data = await response.json();

  if (!data.success) {
    throw new Error(data.error?.message || 'Registration failed');
  }

  // 自动登录并保存 token
  localStorage.setItem('token', data.data.token);

  return data;
}

// 使用示例
register('newuser@example.com', 'New User', 'SecurePass123')
  .then(result => {
    console.log('Registered successfully:', result);
  });
```

### 验证 Token

#### curl 示例

```bash
curl -X POST http://localhost:3000/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

#### JavaScript/fetch 示例

```javascript
async function verifyToken(token) {
  const response = await fetch('/api/v1/auth/verify', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({ token })
  });

  return response.json();
}

// 检查本地存储的 token 是否有效
async function checkAuthStatus() {
  const token = localStorage.getItem('token');

  if (!token) {
    return { valid: false, user: null };
  }

  try {
    const result = await verifyToken(token);
    return result.data;
  } catch (error) {
    localStorage.removeItem('token');
    return { valid: false, user: null };
  }
}
```

### 刷新 Token

#### curl 示例

```bash
curl -X POST http://localhost:3000/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

#### JavaScript/fetch 示例

```javascript
async function refreshToken(currentToken) {
  const response = await fetch('/api/v1/auth/refresh', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ token: currentToken })
  });

  const data = await response.json();

  if (data.success) {
    localStorage.setItem('token', data.data.token);
  }

  return data;
}

// 自动刷新 token（建议在过期前 1 小时调用）
async function autoRefreshToken() {
  const token = localStorage.getItem('token');
  if (token) {
    const result = await refreshToken(token);
    if (result.success) {
      console.log('Token refreshed successfully');
    }
  }
}
```

### 用户登出

#### curl 示例

```bash
curl -X POST http://localhost:3000/api/v1/auth/logout \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

#### JavaScript/fetch 示例

```javascript
async function logout() {
  const token = localStorage.getItem('token');

  await fetch('/api/v1/auth/logout', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  // 清除本地存储
  localStorage.removeItem('token');
  sessionStorage.clear();
}
```

### 密码重置

#### 请求密码重置邮件

```bash
# curl
curl -X POST http://localhost:3000/api/v1/password/forgot \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

```javascript
// JavaScript
async function requestPasswordReset(email) {
  const response = await fetch('/api/v1/password/forgot', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ email })
  });

  return response.json();
}
```

#### 重置密码

```bash
# curl（使用邮箱中的重置链接获取的 token）
curl -X POST http://localhost:3000/api/v1/password/reset \
  -H "Content-Type: application/json" \
  -d '{
    "token": "abc123def456...",
    "newPassword": "NewSecurePass123!"
  }'
```

```javascript
// JavaScript
async function resetPassword(token, newPassword) {
  const response = await fetch('/api/v1/password/reset', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ token, newPassword })
  });

  const data = await response.json();

  if (!data.success) {
    throw new Error(data.error?.message);
  }

  return data;
}
```

---

## 用户管理 API

### 获取当前用户信息

#### curl 示例

```bash
curl -X GET http://localhost:3000/api/v1/users/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

#### JavaScript/fetch 示例

```javascript
async function getCurrentUser() {
  const token = localStorage.getItem('token');

  const response = await fetch('/api/v1/users/me', {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  const data = await response.json();

  if (!data.success) {
    throw new Error(data.error?.message);
  }

  return data.data;
}

// 使用示例
getCurrentUser()
  .then(user => {
    console.log('Current user:', user);
  });
```

### 获取用户列表（仅管理员）

#### curl 示例

```bash
curl -X GET "http://localhost:3000/api/v1/users?page=1&limit=20" \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE"
```

#### JavaScript/fetch 示例

```javascript
async function getUsers(page = 1, limit = 20) {
  const token = localStorage.getItem('token');

  const response = await fetch(`/api/v1/users?page=${page}&limit=${limit}`, {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  return response.json();
}

// 获取所有用户（分页）
async function getAllUsers() {
  let page = 1;
  const limit = 100;
  let allUsers = [];
  let hasMore = true;

  while (hasMore) {
    const result = await getUsers(page, limit);
    if (result.success) {
      allUsers = [...allUsers, ...result.data];
      hasMore = result.data.length === limit;
      page++;
    } else {
      throw new Error(result.error?.message);
    }
  }

  return allUsers;
}
```

### 获取指定用户

#### curl 示例

```bash
curl -X GET http://localhost:3000/api/v1/users/1 \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

#### JavaScript/fetch 示例

```javascript
async function getUserById(userId) {
  const token = localStorage.getItem('token');

  const response = await fetch(`/api/v1/users/${userId}`, {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  return response.json();
}
```

### 更新当前用户信息

#### curl 示例

```bash
curl -X PUT http://localhost:3000/api/v1/users/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Name",
    "email": "newemail@example.com"
  }'
```

#### JavaScript/fetch 示例

```javascript
async function updateCurrentUser(updateData) {
  const token = localStorage.getItem('token');

  const response = await fetch('/api/v1/users/me', {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(updateData)
  });

  return response.json();
}

// 使用示例
updateCurrentUser({ name: 'Updated Name' })
  .then(result => {
    if (result.success) {
      console.log('Profile updated:', result.data);
    }
  });
```

### 创建用户（仅管理员）

#### curl 示例

```bash
curl -X POST http://localhost:3000/api/v1/users \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "name": "New User",
    "password": "SecurePass123"
  }'
```

#### JavaScript/fetch 示例

```javascript
async function createUser(userData) {
  const token = localStorage.getItem('token');

  const response = await fetch('/api/v1/users', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(userData)
  });

  return response.json();
}

// 使用示例
createUser({
  email: 'newuser@example.com',
  name: 'New User',
  password: 'SecurePass123'
})
  .then(result => {
    if (result.success) {
      console.log('User created:', result.data);
    }
  });
```

### 删除用户（仅管理员）

#### curl 示例

```bash
curl -X DELETE http://localhost:3000/api/v1/users/123 \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE"
```

#### JavaScript/fetch 示例

```javascript
async function deleteUser(userId) {
  const token = localStorage.getItem('token');

  const response = await fetch(`/api/v1/users/${userId}`, {
    method: 'DELETE',
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  if (response.status === 204) {
    return { success: true };
  }

  return response.json();
}
```

---

## 通知 API

### 获取通知列表

#### curl 示例

```bash
# 获取所有通知
curl -X GET http://localhost:3000/api/v1/notifications \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"

# 获取未读通知
curl -X GET "http://localhost:3000/api/v1/notifications?status=unread" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"

# 分页获取通知
curl -X GET "http://localhost:3000/api/v1/notifications?page=2&limit=10" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

#### JavaScript/fetch 示例

```javascript
async function getNotifications(options = {}) {
  const { page = 1, limit = 20, status } = options;
  const token = localStorage.getItem('token');

  let url = `/api/v1/notifications?page=${page}&limit=${limit}`;
  if (status) {
    url += `&status=${status}`;
  }

  const response = await fetch(url, {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  return response.json();
}

// 使用示例
getNotifications({ status: 'unread', page: 1, limit: 10 })
  .then(result => {
    if (result.success) {
      console.log('Notifications:', result.data);
    }
  });
```

### 获取未读通知数量

#### curl 示例

```bash
curl -X GET http://localhost:3000/api/v1/notifications/unread/count \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

#### JavaScript/fetch 示例

```javascript
async function getUnreadCount() {
  const token = localStorage.getItem('token');

  const response = await fetch('/api/v1/notifications/unread/count', {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  const result = await response.json();
  return result.data.count;
}

// 在 UI 中显示未读数量
async function updateNotificationBadge() {
  const count = await getUnreadCount();
  const badge = document.querySelector('.notification-badge');

  if (badge) {
    badge.textContent = count;
    badge.style.display = count > 0 ? 'block' : 'none';
  }
}
```

### 获取通知详情

#### curl 示例

```bash
curl -X GET http://localhost:3000/api/v1/notifications/123 \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

#### JavaScript/fetch 示例

```javascript
async function getNotificationById(notificationId) {
  const token = localStorage.getItem('token');

  const response = await fetch(`/api/v1/notifications/${notificationId}`, {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  return response.json();
}
```

### 标记通知为已读

#### curl 示例

```bash
curl -X PUT http://localhost:3000/api/v1/notifications/123/read \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

#### JavaScript/fetch 示例

```javascript
async function markAsRead(notificationId) {
  const token = localStorage.getItem('token');

  const response = await fetch(`/api/v1/notifications/${notificationId}/read`, {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  return response.json();
}
```

### 标记所有通知为已读

#### curl 示例

```bash
curl -X PUT http://localhost:3000/api/v1/notifications/read-all \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

#### JavaScript/fetch 示例

```javascript
async function markAllAsRead() {
  const token = localStorage.getItem('token');

  const response = await fetch('/api/v1/notifications/read-all', {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  return response.json();
}

// 使用示例
markAllAsRead()
  .then(result => {
    if (result.success) {
      console.log('All notifications marked as read');
      updateNotificationBadge();
    }
  });
```

### 删除通知

#### curl 示例

```bash
curl -X DELETE http://localhost:3000/api/v1/notifications/123 \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

#### JavaScript/fetch 示例

```javascript
async function deleteNotification(notificationId) {
  const token = localStorage.getItem('token');

  const response = await fetch(`/api/v1/notifications/${notificationId}`, {
    method: 'DELETE',
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  return response.json();
}
```

### 发送通知

#### curl 示例

```bash
curl -X POST http://localhost:3000/api/v1/notifications/send \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "123",
    "title": "系统通知",
    "message": "您的账户信息已更新",
    "type": "system",
    "channels": ["in_app", "email"]
  }'
```

#### JavaScript/fetch 示例

```javascript
async function sendNotification(data) {
  const token = localStorage.getItem('token');

  const response = await fetch('/api/v1/notifications/send', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(data)
  });

  return response.json();
}

// 使用示例
sendNotification({
  userId: '123',
  title: '系统通知',
  message: '您的账户信息已更新',
  type: 'system',
  channels: ['in_app', 'email']
})
  .then(result => {
    if (result.success) {
      console.log('Notification sent:', result.data);
    }
  });
```

---

## 健康检查 API

### 基础健康检查

#### curl 示例

```bash
curl -X GET http://localhost:3000/api/v1/health
```

#### JavaScript/fetch 示例

```javascript
async function healthCheck() {
  const response = await fetch('/api/v1/health');
  return response.json();
}

// 使用示例
healthCheck()
  .then(result => {
    if (result.success) {
      console.log('Service status:', result.data.status);
      console.log('Version:', result.data.version);
      console.log('Uptime:', result.data.uptime, 'seconds');
    }
  });
```

### 详细健康检查

#### curl 示例

```bash
curl -X GET http://localhost:3000/api/v1/health/detailed
```

#### JavaScript/fetch 示例

```javascript
async function detailedHealthCheck() {
  const response = await fetch('/api/v1/health/detailed');
  return response.json();
}

// 使用示例
detailedHealthCheck()
  .then(result => {
    if (result.success) {
      const { status, checks } = result.data;
      console.log('Overall status:', status);
      console.log('Database:', checks.database.status);
      console.log('Redis:', checks.redis.status);
      console.log('Cache:', checks.cache.status);
    }
  });
```

---

## 完整的 React Hook 示例

```typescript
import { useState, useEffect, useCallback } from 'react';

interface User {
  id: number;
  email: string;
  name: string;
  role: 'user' | 'admin' | 'moderator';
}

interface UseApiOptions {
  requireAuth?: boolean;
}

interface ApiState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

export function useApi<T>(
  fetchFn: () => Promise<T>,
  options: UseApiOptions = {}
) {
  const [state, setState] = useState<ApiState<T>>({
    data: null,
    loading: true,
    error: null
  });

  const fetchData = useCallback(async () => {
    setState(prev => ({ ...prev, loading: true, error: null }));

    try {
      const token = localStorage.getItem('token');
      if (options.requireAuth && !token) {
        throw new Error('Authentication required');
      }

      const result = await fetchFn();
      setState({ data: result, loading: false, error: null });
    } catch (error) {
      setState({
        data: null,
        loading: false,
        error: error instanceof Error ? error.message : 'Unknown error'
      });
    }
  }, [fetchFn, options.requireAuth]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { ...state, refetch: fetchData };
}

// 使用示例
function UserProfile() {
  const { data: user, loading, error, refetch } = useApi<User>(
    () => getCurrentUser(),
    { requireAuth: true }
  );

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;

  return (
    <div>
      <h1>{user?.name}</h1>
      <p>{user?.email}</p>
      <button onClick={refetch}>Refresh</button>
    </div>
  );
}
```

---

## API 客户端封装示例

```javascript
class ApiClient {
  constructor(baseUrl = '/api/v1') {
    this.baseUrl = baseUrl;
  }

  getToken() {
    return localStorage.getItem('token');
  }

  setToken(token) {
    localStorage.setItem('token', token);
  }

  removeToken() {
    localStorage.removeItem('token');
  }

  async request(endpoint, options = {}) {
    const url = `${this.baseUrl}${endpoint}`;
    const token = this.getToken();

    const defaultHeaders = {
      'Content-Type': 'application/json',
      'Accept': 'application/json'
    };

    if (token) {
      defaultHeaders['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(url, {
      ...options,
      headers: {
        ...defaultHeaders,
        ...options.headers
      }
    });

    const data = await response.json();

    if (!response.ok) {
      const error = new Error(data.error?.message || 'Request failed');
      error.status = response.status;
      error.code = data.error?.code;
      throw error;
    }

    return data;
  }

  // Auth API
  async login(email, password) {
    return this.request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password })
    });
  }

  async register(email, name, password) {
    return this.request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, name, password })
    });
  }

  async logout() {
    return this.request('/auth/logout', { method: 'POST' });
  }

  // Users API
  async getCurrentUser() {
    return this.request('/users/me');
  }

  async getUsers(page = 1, limit = 20) {
    return this.request(`/users?page=${page}&limit=${limit}`);
  }

  async getUserById(id) {
    return this.request(`/users/${id}`);
  }

  async updateCurrentUser(data) {
    return this.request('/users/me', {
      method: 'PUT',
      body: JSON.stringify(data)
    });
  }

  async deleteUser(id) {
    return this.request(`/users/${id}`, { method: 'DELETE' });
  }

  // Notifications API
  async getNotifications(options = {}) {
    const params = new URLSearchParams(options);
    return this.request(`/notifications?${params}`);
  }

  async markNotificationAsRead(id) {
    return this.request(`/notifications/${id}/read`, { method: 'PUT' });
  }

  async markAllNotificationsAsRead() {
    return this.request('/notifications/read-all', { method: 'PUT' });
  }

  async deleteNotification(id) {
    return this.request(`/notifications/${id}`, { method: 'DELETE' });
  }

  async sendNotification(data) {
    return this.request('/notifications/send', {
      method: 'POST',
      body: JSON.stringify(data)
    });
  }

  // Health API
  async healthCheck() {
    return this.request('/health');
  }

  async detailedHealthCheck() {
    return this.request('/health/detailed');
  }
}

// 创建全局实例
const api = new ApiClient();

// 使用示例
async function example() {
  try {
    // 登录
    const loginResult = await api.login('user@example.com', 'password');
    console.log('Logged in:', loginResult);

    // 获取当前用户
    const user = await api.getCurrentUser();
    console.log('User:', user);

    // 获取通知
    const notifications = await api.getNotifications({ status: 'unread' });
    console.log('Unread notifications:', notifications);

    // 健康检查
    const health = await api.healthCheck();
    console.log('Health:', health);

  } catch (error) {
    console.error('API Error:', error.message);
    if (error.status === 401) {
      // Token 过期，重新登录
      api.removeToken();
      window.location.href = '/login';
    }
  }
}
```

---

## 错误处理示例

```javascript
// 统一的错误处理函数
function handleApiError(error, context = '') {
  console.error(`API Error${context ? ` in ${context}` : ''}:`, error);

  switch (error.code) {
    case 'UNAUTHORIZED':
      alert('Please login to continue');
      window.location.href = '/login';
      break;

    case 'FORBIDDEN':
      alert('You do not have permission to perform this action');
      break;

    case 'VALIDATION_ERROR':
      const details = error.details || [];
      const message = details.map(d => d.message).join('\n');
      alert(`Validation Error:\n${message}`);
      break;

    case 'TOO_MANY_REQUESTS':
      alert(`Too many requests. Please try again in ${error.retryAfter} seconds`);
      break;

    case 'NOT_FOUND':
      alert('The requested resource was not found');
      break;

    default:
      alert('An unexpected error occurred. Please try again later.');
  }
}

// 在 API 调用中使用
async function safeApiCall(apiFunction, context) {
  try {
    return await apiFunction();
  } catch (error) {
    handleApiError(error, context);
    return null;
  }
}

// 使用示例
const user = await safeApiCall(
  () => api.getCurrentUser(),
  'fetching user profile'
);
```

---

## 分页和筛选示例

```javascript
class PaginationHelper {
  constructor(apiFunction) {
    this.apiFunction = apiFunction;
    this.page = 1;
    this.limit = 20;
    this.hasMore = true;
  }

  async loadMore() {
    if (!this.hasMore) return [];

    const result = await this.apiFunction(this.page, this.limit);

    if (result.success) {
      const data = result.data;
      this.hasMore = data.length === this.limit;
      this.page++;
      return data;
    }

    throw new Error(result.error?.message);
  }

  async refresh() {
    this.page = 1;
    this.hasMore = true;
    return this.loadMore();
  }

  reset() {
    this.page = 1;
    this.hasMore = true;
  }
}

// 使用示例 - 无限滚动加载通知
class NotificationLoader {
  constructor() {
    this.helper = new PaginationHelper(
      (page, limit) => api.getNotifications({ page, limit })
    );
    this.notifications = [];
  }

  async loadMore() {
    const newNotifications = await this.helper.loadMore();
    this.notifications = [...this.notifications, ...newNotifications];
    return this.notifications;
  }

  async refresh() {
    this.helper.reset();
    this.notifications = await this.helper.refresh();
    return this.notifications;
  }
}

// 在 React 组件中使用
function NotificationList() {
  const [notifications, setNotifications] = useState([]);
  const [loading, setLoading] = useState(false);
  const loader = new NotificationLoader();

  const loadMore = async () => {
    if (loading) return;
    setLoading(true);

    try {
      const all = await loader.loadMore();
      setNotifications(all);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      {notifications.map(n => (
        <NotificationItem key={n.id} notification={n} />
      ))}
      <button onClick={loadMore} disabled={loading}>
        {loading ? 'Loading...' : 'Load More'}
      </button>
    </div>
  );
}
```

---

## Rate Limiting 处理

```javascript
class RateLimitedApiClient extends ApiClient {
  constructor(...args) {
    super(...args);
    this.retryAfter = 0;
    this.queue = [];
    this.processing = false;
  }

  async request(endpoint, options = {}) {
    // 检查是否需要等待
    if (this.retryAfter > 0) {
      await this.wait(this.retryAfter);
      this.retryAfter = 0;
    }

    try {
      const result = await super.request(endpoint, options);
      return result;
    } catch (error) {
      if (error.code === 'TOO_MANY_REQUESTS') {
        this.retryAfter = error.retryAfter || 60;
        console.warn(`Rate limited. Retrying in ${this.retryAfter} seconds`);
        return this.request(endpoint, options);
      }
      throw error;
    }
  }

  wait(seconds) {
    return new Promise(resolve => setTimeout(resolve, seconds * 1000));
  }
}
```

---

## 缓存策略示例

```javascript
class CachedApiClient extends ApiClient {
  constructor(...args) {
    super(...args);
    this.cache = new Map();
    this.cacheTimeout = 5 * 60 * 1000; // 5 minutes
  }

  getCacheKey(endpoint, options) {
    return `${endpoint}:${JSON.stringify(options)}`;
  }

  getCached(key) {
    const cached = this.cache.get(key);
    if (cached && Date.now() - cached.timestamp < this.cacheTimeout) {
      return cached.data;
    }
    this.cache.delete(key);
    return null;
  }

  setCache(key, data) {
    this.cache.set(key, {
      data,
      timestamp: Date.now()
    });
  }

  clearCache() {
    this.cache.clear();
  }

  async request(endpoint, options = {}) {
    const cacheKey = this.getCacheKey(endpoint, options);

    // GET 请求使用缓存
    if (options.method === undefined || options.method === 'GET') {
      const cached = this.getCached(cacheKey);
      if (cached) {
        console.log('Returning cached data');
        return cached;
      }
    }

    const result = await super.request(endpoint, options);

    // 缓存成功的 GET 响应
    if (result.success && (!options.method || options.method === 'GET')) {
      this.setCache(cacheKey, result);
    }

    return result;
  }
}

// 使用示例
const cachedApi = new CachedApiClient();

// 获取用户信息（会被缓存）
const user1 = await cachedApi.getCurrentUser(); // 首次请求
const user2 = await cachedApi.getCurrentUser(); // 使用缓存

// 清除缓存
cachedApi.clearCache();
```
