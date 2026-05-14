# Monitoring and Logging Guide

## Overview

This document describes the monitoring and logging system implemented in the HJTPX project.

## Logging System

### Logger Configuration

The project uses Winston for structured logging with multiple transport options:

- **Console Transport**: For development and debugging
- **File Transport**: For production error and combined logs
- **HTTP Transport**: Optional remote logging support

### Log Levels

| Level | Description | Use Case |
|-------|-------------|----------|
| error | Error messages and exceptions | Failures, crashes |
| warn | Warning messages | Deprecated usage, unusual patterns |
| info | Informational messages | Request/response logging |
| http | HTTP request logs | API monitoring |
| debug | Debug information | Development troubleshooting |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| LOG_LEVEL | info | Minimum log level to output |
| LOG_FORMAT | json | Log format (json or simple) |
| LOG_CONSOLE | true | Enable console logging |
| LOG_FILE | false | Enable file logging |
| LOG_MAX_SIZE | 5242880 | Max log file size (bytes) |
| LOG_MAX_FILES | 5 | Number of log files to keep |

### Usage

```javascript
const { Logger, logInfo, logError } = require('./utils/logger');

logInfo('User logged in', { userId: 123, email: 'user@example.com' });
logError(new Error('Database error'), req, { operation: 'fetch_users' });
```

### Request Logging

The `requestLogger` middleware automatically logs all HTTP requests:

```javascript
const { requestLogger } = require('./middleware/requestLogger');
app.use(requestLogger);
```

Features:
- Automatic request ID generation
- Request/response duration tracking
- Sensitive data masking (passwords, tokens)
- Slow request detection

## Monitoring System

### Metrics Collection

The monitoring system collects:

- **HTTP Metrics**: Request count, response time, status codes
- **System Metrics**: CPU usage, memory usage, process uptime
- **Database Metrics**: Connection pool status, query performance
- **Custom Metrics**: Application-specific business metrics

### Health Checks

Two health check endpoints are available:

- `GET /api/v1/health` - Basic health check
- `GET /api/v1/health/detailed` - Detailed status with dependencies

### Slow Request Detection

Configure slow request threshold:

```bash
SLOW_REQUEST_THRESHOLD=3000
```

Requests exceeding this threshold are logged as warnings.

## Alerting

### Alert Conditions

- Slow requests (>3s default)
- High error rate (>5% default)
- High CPU usage (>80% default)
- High memory usage (>85% default)

### Alert Configuration

```javascript
// In src/backend/config/logging.js
module.exports = {
  alerts: {
    enabled: true,
    slowRequestThreshold: 3000,
    errorRateThreshold: 0.05,
    cpuThreshold: 0.8,
    memoryThreshold: 0.85
  }
};
```

## Production Setup

### Log Rotation

Logs are automatically rotated based on size (5MB default) with 5 backup files kept.

### Log Aggregation

For production environments, configure remote logging:

```bash
LOG_REMOTE=true
LOG_REMOTE_ENDPOINT=https://logs.example.com/api/logs
LOG_REMOTE_TOKEN=your-api-token
```

### Recommended External Services

- ELK Stack (Elasticsearch, Logstash, Kibana)
- Datadog
- New Relic
- CloudWatch Logs
- Papertrail

## Troubleshooting

### Common Issues

1. **Logs not appearing in console**
   - Check `LOG_CONSOLE=true`
   - Verify `LOG_LEVEL` is not set too high

2. **Log files not created**
   - Ensure `LOG_FILE=true`
   - Check write permissions for logs directory

3. **High memory usage from logging**
   - Reduce `LOG_MAX_FILES`
   - Lower `LOG_LEVEL` to reduce verbosity

### Debug Mode

Enable debug logging:

```bash
NODE_ENV=development LOG_LEVEL=debug
```

## Best Practices

1. Always include request ID in logs for traceability
2. Mask sensitive data before logging
3. Use appropriate log levels consistently
4. Set up log rotation in production
5. Configure alerts for critical conditions
6. Centralize logs in production environments
7. Regularly review slow request patterns
8. Monitor error rates and set up alerts
