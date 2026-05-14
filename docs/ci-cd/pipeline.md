# CI/CD Pipeline Guide

## Overview

This document describes the CI/CD pipeline implemented for the HJTPX project.

## Pipeline Structure

### Continuous Integration (CI)

The CI pipeline runs on every push and pull request to `main` and `develop` branches.

#### Stages

1. **Lint Stage**
   - ESLint code quality checks
   - Prettier formatting validation
   - Status: Required to pass

2. **Test Stage**
   - Unit tests with coverage
   - Integration tests
   - Full test suite
   - Database migrations
   - Services: PostgreSQL 14, Redis 7

3. **Security Stage**
   - npm audit
   - Snyk vulnerability scan
   - Dependency checks

4. **Build Stage**
   - Application build
   - Docker image creation
   - Artifact generation

5. **E2E Test Stage** (Pull Requests only)
   - Playwright E2E tests
   - Browser automation tests

### Continuous Deployment (CD)

The CD pipeline runs on pushes to `main` branch and manual triggers.

#### Environments

- **Staging**: Automatic deployment for testing
- **Production**: Manual approval required

#### Deployment Flow

1. **Prepare**
   - Set environment variables
   - Generate image tags

2. **Build**
   - Docker image build
   - Push to container registry
   - Cache optimization

3. **Test Deployment**
   - Run smoke tests
   - Execute migrations

4. **Deploy**
   - AWS ECS deployment
   - Health check verification
   - Rolling update

5. **Rollback** (on failure)
   - Automatic rollback
   - Slack notification

## GitHub Actions Workflows

### CI Workflow (ci.yml)

Triggers:
- Push to `main`, `develop` branches
- Pull requests

Jobs:
- `lint`: Code quality checks
- `test`: Unit and integration tests
- `security`: Security audits
- `build`: Application build
- `e2e-test`: End-to-end tests
- `notify`: Status notifications

### Deploy Workflow (deploy.yml)

Triggers:
- Push to `main` branch
- Manual workflow dispatch

Inputs:
- `environment`: staging or production

Jobs:
- `prepare`: Environment setup
- `build`: Docker image build
- `test-deployment`: Pre-deployment tests
- `deploy-staging`: Staging deployment
- `deploy-production`: Production deployment
- `rollback`: Failure recovery
- `notify-success`: Success notifications

## Environment Variables

### Required Secrets

| Variable | Description |
|----------|-------------|
| `AWS_ACCESS_KEY_ID` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key |
| `AWS_REGION` | AWS region |
| `SLACK_WEBHOOK_URL` | Slack webhook for notifications |
| `SNYK_TOKEN` | Snyk security scan token |
| `CODECOV_TOKEN` | Codecov coverage token |

### Environment-specific Variables

| Variable | Staging | Production |
|----------|---------|------------|
| `DATABASE_URL` | staging-db-url | production-db-url |
| `REDIS_URL` | staging-redis-url | production-redis-url |
| `JWT_SECRET` | staging-secret | production-secret |

## Deployment Configuration

### Docker

```dockerfile
# Multi-stage build
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:18-alpine AS runner
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package*.json ./
RUN npm ci --production
EXPOSE 3000
CMD ["node", "dist/index.js"]
```

### AWS ECS Configuration

```yaml
# Task definition
taskDefinition:
  family: hjtpx-api
  containerDefinitions:
    - name: api
      image: ghcr.io/hjtpx/hjtpx:latest
      portMappings:
        - containerPort: 3000
      environment:
        - name: NODE_ENV
          value: production
      secrets:
        - name: DATABASE_URL
          valueFrom: arn:aws:secretsmanager:...
      healthCheck:
        command: ["CMD", "curl", "-f", "http://localhost:3000/health"]
```

## Monitoring and Alerts

### Health Checks

- Endpoint: `/health`
- Interval: 30 seconds
- Timeout: 5 seconds
- Healthy threshold: 3 consecutive successes
- Unhealthy threshold: 3 consecutive failures

### Slack Notifications

Notifications sent for:
- Build success/failure
- Deployment success/failure
- Rollback triggered

## Rollback Procedures

### Automatic Rollback

Triggered when:
- Health check fails
- Deployment timeout exceeded
- Critical error in logs

### Manual Rollback

```bash
# Rollback to previous task definition
aws ecs update-service \
  --cluster hjtpx-production \
  --service hjtpx-api \
  --task-definition hjtpx-api:previous_version

# Verify rollback
aws ecs describe-services \
  --cluster hjtpx-production \
  --services hjtpx-api
```

## Best Practices

1. **Always use versioned tags**
2. **Run tests before deployment**
3. **Implement health checks**
4. **Use infrastructure as code**
5. **Monitor deployment metrics**
6. **Have rollback plan ready**
7. **Notify on failures**
8. **Test in staging first**

## Troubleshooting

### Common Issues

1. **Build fails**
   - Check Node.js version compatibility
   - Verify npm dependencies
   - Check Docker build context

2. **Deployment timeout**
   - Increase timeout in ECS service
   - Check health check configuration
   - Verify network connectivity

3. **Rollback not working**
   - Ensure previous task definition exists
   - Check IAM permissions
   - Verify service stability

### Debug Commands

```bash
# Check workflow status
gh run list

# View workflow logs
gh run view <run-id> --log

# Download artifacts
gh run download <run-id>

# Re-run failed workflow
gh run rerun <run-id>
```

## Security Considerations

1. **Secrets Management**
   - Use GitHub Secrets for sensitive data
   - Rotate secrets regularly
   - Never commit secrets to repository

2. **Access Control**
   - Limit GitHub Actions permissions
   - Use environment protection rules
   - Require approval for production

3. **Container Security**
   - Use minimal base images
   - Scan for vulnerabilities
   - Keep dependencies updated

## Maintenance

### Regular Tasks

- Review and update dependencies
- Clean up old Docker images
- Update documentation
- Review security alerts
- Optimize CI/CD performance
