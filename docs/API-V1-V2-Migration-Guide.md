# API v1 to v2 Migration Guide

## Overview

This guide helps you migrate from API v1 to v2 with minimal disruption to your application.

## Timeline

| Milestone | Date |
|-----------|------|
| v1 Release | 2023-01-01 |
| v2 Release | 2024-01-01 |
| v1 Deprecation Notice | 2024-06-01 |
| v1 Sunset Date | 2025-12-31 |

## Breaking Changes Summary

### 1. Authentication

#### Login Endpoint

**v1:**
```json
POST /api/v1/auth/login
Request: { "email": "string", "password": "string" }
Response: { "token": "string", "user": "object" }
```

**v2:**
```json
POST /api/v2/auth/login
Request: { "email": "string", "password": "string" }
Response: {
  "success": true,
  "data": {
    "accessToken": "string",
    "refreshToken": "string",
    "expiresIn": 3600,
    "user": "object"
  }
}
```

**Migration Steps:**
1. Update authentication logic to handle separate access and refresh tokens
2. Implement token refresh mechanism before expiration
3. Update token storage to use separate keys

#### Token Refresh Endpoint

**v2 New Endpoint:**
```json
POST /api/v2/auth/refresh
Request: { "refreshToken": "string" }
Response: { "accessToken": "string", "expiresIn": 3600 }
```

**Migration Steps:**
1. Implement automatic token refresh before expiration
2. Handle token refresh failures gracefully
3. Redirect to login on refresh token expiration

### 2. Response Format

#### All Endpoints

**v1:**
```json
{ "data": ... }
```

**v2:**
```json
{
  "success": true,
  "data": ...,
  "meta": {
    "timestamp": "ISO8601",
    "version": "v2",
    "requestId": "string"
  }
}
```

**Migration Steps:**
1. Check `success` field before processing data
2. Access actual data via `response.data`
3. Use `response.meta` for additional information

### 3. Pagination

**v1:**
```json
GET /api/v1/users?page=1&pageSize=10
Response: {
  "users": [...],
  "totalCount": 100,
  "currentPage": 1,
  "totalPages": 10
}
```

**v2:**
```json
GET /api/v2/users?offset=0&limit=10
Response: {
  "success": true,
  "data": [...],
  "meta": {
    "total": 100,
    "offset": 0,
    "limit": 10,
    "hasMore": true
  }
}
```

**Migration Steps:**
1. Replace `page/pageSize` with `offset/limit`
2. Calculate offset: `offset = (page - 1) * pageSize`
3. Access data via `response.data`
4. Check `response.meta.hasMore` instead of pagination

### 4. Date Format

**v1:** `YYYY-MM-DD HH:mm:ss`

**v2:** `ISO 8601` (e.g., `2024-01-15T10:30:00.000Z`)

**Migration Steps:**
1. Update date parsing logic
2. Use standard Date parsing libraries
3. Handle timezone conversions if needed

### 5. Error Handling

**v1:**
```json
{
  "error": "Error message",
  "message": "Details",
  "code": "ERROR_CODE"
}
```

**v2:**
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Error message",
    "details": {},
    "requestId": "req_xxx",
    "timestamp": "ISO8601"
  }
}
```

**Migration Steps:**
1. Check `success` field
2. Access error details from `error` object
3. Use `requestId` for debugging

## Code Examples

### JavaScript/TypeScript

```javascript
// v1
const response = await fetch('/api/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email, password })
});
const { token, user } = await response.json();
localStorage.setItem('token', token);

// v2
const response = await fetch('/api/v2/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email, password })
});
const json = await response.json();
if (!json.success) {
  throw new Error(json.error.message);
}
const { accessToken, refreshToken, user } = json.data;
localStorage.setItem('accessToken', accessToken);
localStorage.setItem('refreshToken', refreshToken);
```

### Token Refresh

```javascript
class ApiClient {
  constructor() {
    this.baseUrl = '/api/v2';
    this.accessToken = localStorage.getItem('accessToken');
    this.refreshToken = localStorage.getItem('refreshToken');
  }

  async fetch(endpoint, options = {}) {
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers: {
        ...options.headers,
        'Authorization': `Bearer ${this.accessToken}`,
        'Content-Type': 'application/json'
      }
    });

    if (response.status === 401) {
      await this.refreshAccessToken();
      return this.fetch(endpoint, options);
    }

    return response.json();
  }

  async refreshAccessToken() {
    const response = await fetch(`${this.baseUrl}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken: this.refreshToken })
    });

    const json = await response.json();
    if (json.success) {
      this.accessToken = json.data.accessToken;
      localStorage.setItem('accessToken', this.accessToken);
    } else {
      localStorage.removeItem('accessToken');
      localStorage.removeItem('refreshToken');
      window.location.href = '/login';
    }
  }
}
```

### Pagination

```javascript
// v1
const loadUsers = async (page) => {
  const response = await api.fetch(`/users?page=${page}&pageSize=10`);
  const { users, totalPages } = response;
  return { users, totalPages };
};

// v2
const loadUsers = async (offset, limit) => {
  const response = await api.fetch(`/users?offset=${offset}&limit=${limit}`);
  const { data: users, meta } = response;
  return {
    users,
    total: meta.total,
    hasMore: meta.hasMore
  };
};

// With helper
const getPaginationParams = (page, pageSize) => {
  return {
    offset: (page - 1) * pageSize,
    limit: pageSize
  };
};
```

## Testing Checklist

- [ ] Authentication flow works end-to-end
- [ ] Token refresh mechanism functions correctly
- [ ] All response parsing works with new format
- [ ] Pagination works with offset/limit
- [ ] Date parsing handles ISO 8601
- [ ] Error handling catches all error cases
- [ ] All CRUD operations work
- [ ] WebSocket connections (if applicable)

## Rollback Plan

If issues arise, temporarily revert to v1:

```javascript
const response = await fetch('/api/v1/users', {
  headers: {
    'Accept-Version': 'v1'
  }
});
```

**Note:** v1 will remain available until 2025-12-31. Resolve all issues before this date.

## Support

- Documentation: `/api-docs/v2`
- Migration Guide: `/api-docs/v1-to-v2-migration`
- Support Email: support@hjtpx.com

## Changelog

### v2.1.0 (2024-01-01)
- Initial v2 release
- JWT with refresh tokens
- Standardized response format
- New pagination parameters
- ISO 8601 date format

### v2.0.0 (2024-01-01)
- Migration from v1
- Breaking changes documented
