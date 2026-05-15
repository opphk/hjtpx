# API Version Migration Guide

## Overview
This guide helps you migrate from API v1 to v2, covering all breaking changes and best practices.

## Versioning Strategy

### Supported Versions
- **v1 (Stable)**: Current production version, will receive security updates until sunset date
- **v2 (Beta)**: New version with enhanced features, recommended for new development

### Version Lifecycle
1. **Beta**: Initial release, may have breaking changes with notice
2. **Stable**: Production-ready, security updates provided
3. **Deprecated**: Security updates only, migration strongly recommended
4. **Sunset**: No longer available, requests will fail

## Breaking Changes from v1 to v2

### 1. User Response Format

#### v1 Format
```json
{
  "user": {
    "id": "123",
    "name": "John Doe",
    "email": "john@example.com",
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-06-01T00:00:00Z"
  }
}
```

#### v2 Format
```json
{
  "user": {
    "id": "123",
    "name": "John Doe",
    "email": "john@example.com",
    "metadata": {
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-06-01T00:00:00Z"
    }
  }
}
```

**Migration Steps:**
1. Update your JSON parsing logic to handle the new `metadata` object
2. Access user timestamps via `user.metadata.createdAt` instead of `user.createdAt`
3. Test the changes in a non-production environment first

### 2. Pagination Parameters

#### v1 Parameters
- `page`: Page number (1-based)
- `limit`: Items per page

#### v2 Parameters
- `offset`: Number of items to skip
- `limit`: Items per page

**Conversion Formula:**
```javascript
// v1 to v2
const offset = (page - 1) * limit;
```

**Migration Steps:**
1. Replace `page` parameter with `offset`
2. Calculate offset: `offset = (page - 1) * limit`
3. Update pagination UI components if applicable

### 3. Error Response Format

#### v1 Error
```json
{
  "error": "Validation failed",
  "message": "Email is required"
}
```

#### v2 Error
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Email is required",
    "details": {
      "field": "email",
      "reason": "missing"
    }
  }
}
```

**Migration Steps:**
1. Update error handling to check `response.success` first
2. Access error details via `response.error.code` and `response.error.details`
3. Update logging and monitoring systems

## Migration Timeline

| Milestone | Date | Description |
|-----------|------|-------------|
| v2 Beta Release | 2025-06-01 | New version available for testing |
| v2 Stable | 2025-09-01 | v2 becomes production-ready |
| v1 Deprecation Notice | 2025-12-01 | v1 marked as deprecated |
| v1 Sunset | 2026-06-01 | v1 no longer available |

## Testing Your Migration

### Test Checklist
- [ ] User response parsing handles new metadata format
- [ ] Pagination works correctly with offset parameter
- [ ] Error handling works with new error structure
- [ ] Authentication still functions properly
- [ ] Rate limiting works as expected
- [ ] WebSocket connections stable
- [ ] File uploads/downloads working

### Test Endpoints
```bash
# Test v2 with query parameter
GET /api/v2/users?api_version=v2

# Test with Accept header
GET /api/v2/users
Accept: application/json; api-version=v2

# Test version info
GET /api/v2/versions

# Test migration guide
GET /api/v2/versions/migration/v1/v2
```

## Rollback Plan

If issues occur after migration:

1. **Immediate**: Switch back to v1 by changing the version parameter
2. **Short-term**: Use version negotiation to support both versions
3. **Long-term**: Fix issues in v2 and redeploy

## Support

- **Email**: api-support@hjtpx.com
- **Slack**: #api-migration
- **Documentation**: /api-docs/v2
- **Status Page**: status.hjtpx.com
