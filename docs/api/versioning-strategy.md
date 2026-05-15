# API Versioning Strategy

## Overview

The HJTPX API supports version control to ensure backward compatibility while allowing continuous improvement. This document outlines our versioning strategy, migration process, and best practices.

## Version Lifecycle

### Version States

| State | Description | Support Level |
|-------|-------------|---------------|
| **Stable** | Production-ready, fully supported | Full support |
| **Deprecated** | Still functional but will be removed | Limited support |
| **Sunset** | No longer available | No support |

### Current API Versions

| Version | Status | Sunset Date | Deprecation Date |
|---------|--------|-------------|------------------|
| v1 | Deprecated | 2026-07-01 | 2026-01-01 |
| v2 | Stable | - | - |

## Version Negotiation

### URL Path Versioning

The primary method for specifying API version is through the URL path:

```bash
# v1 API
GET /api/v1/users
GET /api/v1/health

# v2 API
GET /api/v2/users
GET /api/v2/health
```

### Header-Based Versioning

You can also specify version using HTTP headers:

#### Accept-Version Header

```bash
GET /api/health
Accept-Version: v1

GET /api/health
Accept-Version: v2
```

#### Accept Header

```bash
GET /api/users
Accept: application/vnd.hjtpx.v1+json

GET /api/users
Accept: application/vnd.hjtpx.v2+json
```

#### Custom Header

```bash
GET /api/users
X-API-Version: v1
```

#### Prefer Header

```bash
GET /api/users
Prefer: version=v1
```

## Version Downgrade Strategy

When a requested version is not available or deprecated, the API automatically negotiates to the next available version:

1. **Requested version exists** → Use requested version
2. **Requested version deprecated** → Use deprecated version with warnings
3. **Requested version unavailable** → Fallback to default version (v2)

### Example: Requesting Unavailable Version

```bash
# Request v3 (doesn't exist)
GET /api/health
X-API-Version: v3

# Response headers
X-API-Version: v2
X-API-Version-Negotiated: true
X-API-Version-Upgrade: Version v3 not available. Using v2.
```

## Response Headers

All API responses include version-related headers:

| Header | Description | Example |
|--------|-------------|---------|
| `X-API-Version` | Current API version | `v2` |
| `X-API-Version-Status` | Version status | `stable` |
| `X-API-Supported-Versions` | All supported versions | `v1, v2` |
| `X-API-Latest-Version` | Latest stable version | `v2` |
| `X-API-Version-Negotiated` | Version was negotiated | `true` |

### Deprecation Headers

For deprecated versions (v1):

| Header | Description |
|--------|-------------|
| `Deprecation` | Indicates version is deprecated |
| `X-API-Deprecation-Date` | Date when deprecation started |
| `X-API-Sunset-Date` | Date when version will be removed |
| `X-API-Migration-Guide` | Link to migration guide |
| `X-API-Days-Until-Sunset` | Days remaining before sunset |
| `Warning` | Deprecation warning message |

## v1 to v2 Migration Guide

### Breaking Changes in v2

#### 1. Authentication Changes

**v1:**
```javascript
// Basic authentication
Authorization: Basic base64(username:password)
```

**v2:**
```javascript
// JWT token authentication
Authorization: Bearer <jwt_token>
```

#### 2. Response Format Changes

**v1 Response:**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "John Doe"
  }
}
```

**v2 Response:**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "John Doe",
    "profile": {
      "avatar": "url",
      "bio": "string"
    }
  },
  "meta": {
    "timestamp": "2026-05-15T10:00:00Z",
    "version": "v2"
  }
}
```

#### 3. Pagination

**v1:** No pagination (all results returned)

**v2:**
```bash
GET /api/v2/users?page=1&limit=20
```

Response includes:
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "totalPages": 5
  }
}
```

#### 4. Error Format

**v1:**
```json
{
  "success": false,
  "error": "User not found"
}
```

**v2:**
```json
{
  "success": false,
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User not found",
    "details": {
      "userId": 123
    }
  }
}
```

### Migration Steps

#### Step 1: Update API Base URL

```javascript
// Before (v1)
const API_BASE = '/api/v1';

// After (v2)
const API_BASE = '/api/v2';
```

#### Step 2: Update Authentication

```javascript
// Implement JWT authentication
const getAuthToken = async (credentials) => {
  const response = await fetch('/api/v2/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(credentials)
  });
  
  const data = await response.json();
  return data.data.token;
};

// Use token in requests
const fetchWithAuth = async (url, token) => {
  return fetch(url, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Accept': 'application/vnd.hjtpx.v2+json'
    }
  });
};
```

#### Step 3: Update Response Parsing

```javascript
// v2 response handling
const handleUserResponse = (response) => {
  const { data, pagination, meta } = response;
  
  // Access user data
  const user = data;
  
  // Access profile (new in v2)
  const profile = data.profile;
  
  // Access metadata
  const timestamp = meta.timestamp;
  
  return { user, profile, pagination };
};
```

#### Step 4: Implement Pagination

```javascript
const fetchPaginatedUsers = async (page = 1, limit = 20) => {
  const response = await fetch(
    `/api/v2/users?page=${page}&limit=${limit}`,
    {
      headers: {
        'Accept': 'application/vnd.hjtpx.v2+json'
      }
    }
  );
  
  const data = await response.json();
  
  return {
    users: data.data,
    pagination: data.pagination,
    hasMore: data.pagination.page < data.pagination.totalPages
  };
};
```

#### Step 5: Update Error Handling

```javascript
const handleApiError = (error) => {
  const { code, message, details } = error.error;
  
  switch (code) {
    case 'USER_NOT_FOUND':
      showUserNotFoundError(details.userId);
      break;
    case 'UNAUTHORIZED':
      redirectToLogin();
      break;
    default:
      showGenericError(message);
  }
};
```

### Testing Your Migration

```javascript
// Test script to verify v2 compatibility
const testV2Migration = async () => {
  const tests = [
    {
      name: 'Authentication',
      test: async () => {
        const token = await getAuthToken(testCredentials);
        return token !== null;
      }
    },
    {
      name: 'User Fetch',
      test: async () => {
        const response = await fetchWithAuth('/api/v2/users/1', token);
        const data = await response.json();
        return data.data.profile !== undefined;
      }
    },
    {
      name: 'Pagination',
      test: async () => {
        const response = await fetchWithAuth('/api/v2/users?page=1', token);
        const data = await response.json();
        return data.pagination !== undefined;
      }
    },
    {
      name: 'Error Format',
      test: async () => {
        const response = await fetchWithAuth('/api/v2/users/99999', token);
        const data = await response.json();
        return data.error && data.error.code !== undefined;
      }
    }
  ];
  
  for (const test of tests) {
    const result = await test.test();
    console.log(`${test.name}: ${result ? 'PASS' : 'FAIL'}`);
  }
};
```

## Feature Comparison

| Feature | v1 | v2 |
|---------|----|----|
| Basic Authentication | ✅ | ❌ |
| JWT Authentication | ❌ | ✅ |
| Legacy Response Format | ✅ | ❌ |
| Enhanced Response Format | ❌ | ✅ |
| Pagination | ❌ | ✅ |
| Rate Limiting | ❌ | ✅ |
| Advanced Filtering | ❌ | ✅ |
| Deprecation Warnings | ✅ | ✅ |
| Enhanced Error Handling | ❌ | ✅ |

## Best Practices

### 1. Always Specify Version

```bash
# Good: Explicitly specify version
curl -H "Accept: application/vnd.hjtpx.v2+json" /api/users

# Avoid: Relying on default
curl /api/users
```

### 2. Handle Deprecation Warnings

```javascript
const checkDeprecationHeaders = (response) => {
  const warning = response.headers.get('Warning');
  if (warning) {
    console.warn('API Deprecation Warning:', warning);
    // Trigger migration alert
    sendMigrationAlert(warning);
  }
};
```

### 3. Implement Retry Logic

```javascript
const fetchWithRetry = async (url, options, retries = 3) => {
  for (let i = 0; i < retries; i++) {
    try {
      const response = await fetch(url, options);
      checkDeprecationHeaders(response);
      return response;
    } catch (error) {
      if (i === retries - 1) throw error;
      await delay(1000 * Math.pow(2, i));
    }
  }
};
```

### 4. Monitor Version Usage

Track version usage in your application:

```javascript
const trackVersionUsage = (version) => {
  analytics.track('api_version_used', {
    version: version,
    timestamp: new Date().toISOString()
  });
};
```

## Resources

- [v1 Migration Guide](./v1-migration-guide.md)
- [API Documentation](./API_VERSIONING.md)
- [Changelog](./versions/changes-1.0.0.json)

## Support

For migration assistance:
- Email: support@hjtpx.com
- Documentation: /docs
- Support Portal: /support

## Timeline

```
2026-01-01  v1 Deprecated
2026-05-15  v2 Released (Current)
2026-07-01  v1 Sunset Date
```

**Important:** v1 will be discontinued on 2026-07-01. Please migrate to v2 before this date.
