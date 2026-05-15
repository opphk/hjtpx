#!/bin/bash
# 任务17：CI/CD优化
# 优化依赖安装
# 优化测试执行
# 添加并行任务
# 添加缓存机制
# 优化部署流程

echo "=========================================="
echo "任务17：CI/CD优化"
echo "=========================================="

cd /workspace/hjtpx

# 1. 优化CI工作流
echo "[17.1] 优化CI工作流..."

cat > .github/workflows/ci-optimized.yml << 'EOF'
name: CI (优化版)

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

env:
  NODE_VERSION: '18'
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  # 静态检查 - 快速反馈
  lint:
    name: Lint & Type Check
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3
        
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: 'npm'
          
      - name: Cache node_modules
        uses: actions/cache@v3
        id: cache-node
        with:
          path: node_modules
          key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
          restore-keys: |
            ${{ runner.os }}-node-
            
      - name: Install dependencies
        if: steps.cache-node.outputs.cache-hit != 'true'
        run: npm ci --prefer-offline
        
      - name: ESLint
        run: npm run lint
        
      - name: Prettier Check
        run: npm run format:check
        
      - name: Type Check
        run: npm run typecheck

  # 单元测试
  test-unit:
    name: Unit Tests
    runs-on: ubuntu-latest
    needs: lint
    steps:
      - name: Checkout
        uses: actions/checkout@v3
        
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: 'npm'
          
      - name: Get cached node_modules
        uses: actions/cache@v3
        with:
          path: node_modules
          key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
          
      - name: Run unit tests
        run: npm run test:unit -- --coverage --ci
        
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage/lcov.info
          flags: unittests
          
      - name: Generate test report
        if: always()
        run: npm run test:unit -- --reporters=default --reporters=jest-junit

  # 集成测试
  test-integration:
    name: Integration Tests
    runs-on: ubuntu-latest
    needs: test-unit
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
          
      redis:
        image: redis:7
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
          
    steps:
      - name: Checkout
        uses: actions/checkout@v3
        
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: 'npm'
          
      - name: Get cached node_modules
        uses: actions/cache@v3
        with:
          path: node_modules
          key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
          
      - name: Run integration tests
        env:
          DATABASE_URL: postgresql://test:test@localhost:5432/test
          REDIS_URL: redis://localhost:6379
        run: npm run test:integration -- --ci

  # E2E测试
  test-e2e:
    name: E2E Tests
    runs-on: ubuntu-latest
    needs: test-integration
    if: github.event_name == 'pull_request'
    steps:
      - name: Checkout
        uses: actions/checkout@v3
        
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: 'npm'
          
      - name: Get cached node_modules
        uses: actions/cache@v3
        with:
          path: node_modules
          key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
          
      - name: Install Playwright
        run: npx playwright install --with-deps
        
      - name: Run E2E tests
        run: npm run test:e2e -- --ci
        
      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: playwright-report
          path: test-results/
          retention-days: 7

  # 安全扫描
  security:
    name: Security Scan
    runs-on: ubuntu-latest
    needs: lint
    steps:
      - name: Checkout
        uses: actions/checkout@v3
        
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: 'npm'
          
      - name: Get cached node_modules
        uses: actions/cache@v3
        with:
          path: node_modules
          key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
          
      - name: Run npm audit
        run: npm audit --audit-level=high
        
      - name: Run Snyk security scan
        uses: snyk/actions/node@master
        env:
          SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
          
      - name: OWASP dependency check
        run: npm run security:owasp

  # 构建Docker镜像
  build:
    name: Build Docker Image
    runs-on: ubuntu-latest
    needs: [test-unit, test-integration, security]
    if: github.event_name == 'push'
    outputs:
      image-tag: ${{ steps.meta.outputs.tags }}
      image-digest: ${{ steps.build.outputs.digest }}
    steps:
      - name: Checkout
        uses: actions/checkout@v3
        
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2
        
      - name: Log in to Container Registry
        uses: docker/login-action@v2
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
          
      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v4
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=ref,event=branch
            type=sha,prefix={{branch}}-
            type=raw,value=latest,enable={{is_default_branch}}
            
      - name: Build and push with cache
        uses: docker/build-push-action@v4
        with:
          context: .
          push: ${{ github.event_name != 'pull_request' }}
          tags: ${{ steps.meta.outputs.tags }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
          
      - name: Image digest
        run: echo "Digest: ${{ steps.build.outputs.digest }}"

  # 部署到测试环境
  deploy-test:
    name: Deploy to Test
    runs-on: ubuntu-latest
    needs: build
    if: github.ref == 'refs/heads/develop'
    environment:
      name: test
      url: https://test.hjtpx.com
    steps:
      - name: Deploy to test server
        uses: appleboy/ssh-action@master
        with:
          host: ${{ secrets.TEST_SERVER_HOST }}
          username: ${{ secrets.TEST_SERVER_USER }}
          key: ${{ secrets.TEST_SERVER_KEY }}
          script: |
            cd /app
            docker-compose pull
            docker-compose up -d
            docker-compose exec -T app npm run migrate
            docker-compose exec -T app npm run seed
            
  # 部署到生产环境
  deploy-prod:
    name: Deploy to Production
    runs-on: ubuntu-latest
    needs: build
    if: github.ref == 'refs/heads/main'
    environment:
      name: production
      url: https://hjtpx.com
    steps:
      - name: Deploy to production server
        uses: appleboy/ssh-action@master
        with:
          host: ${{ secrets.PROD_SERVER_HOST }}
          username: ${{ secrets.PROD_SERVER_USER }}
          key: ${{ secrets.PROD_SERVER_KEY }}
          script: |
            cd /app
            docker-compose pull
            docker-compose up -d --remove-orphans
            docker-compose exec -T app npm run migrate
            # 等待服务启动
            sleep 30
            # 运行健康检查
            curl -f http://localhost:3000/health || exit 1

  # 生成构建报告
  report:
    name: Build Report
    runs-on: ubuntu-latest
    needs: [build, deploy-test]
    if: always()
    steps:
      - name: Generate report
        run: |
          echo "# 构建报告" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY
          echo "## 工作流状态" >> $GITHUB_STEP_SUMMARY
          echo "\`\`\`" >> $GITHUB_STEP_SUMMARY
          echo "Commit: ${{ github.sha }}" >> $GITHUB_STEP_SUMMARY
          echo "Branch: ${{ github.ref_name }}" >> $GITHUB_STEP_SUMMARY
          echo "Actor: ${{ github.actor }}" >> $GITHUB_STEP_SUMMARY
          echo "\`\`\`" >> $GITHUB_STEP_SUMMARY
EOF

# 2. 优化部署工作流
echo "[17.2] 优化部署工作流..."

cat > .github/workflows/deploy-optimized.yml << 'EOF'
name: Deploy (优化版)

on:
  workflow_dispatch:
    inputs:
      environment:
        description: '部署环境'
        required: true
        type: choice
        options:
          - staging
          - production
      skip_tests:
        description: '跳过测试'
        type: boolean
        default: false

jobs:
  pre-deploy:
    name: Pre-deployment checks
    runs-on: ubuntu-latest
    outputs:
      can_deploy: ${{ steps.check.outputs.can_deploy }}
    steps:
      - name: Check branch protection
        id: check
        run: |
          if [[ "${{ github.ref }}" == "refs/heads/main" ]] || [[ "${{ github.ref }}" == "refs/heads/develop" ]]; then
            echo "can_deploy=true" >> $GITHUB_OUTPUT
          else
            echo "can_deploy=false" >> $GITHUB_OUTPUT
          fi

  deploy:
    name: Deploy to ${{ inputs.environment || 'production' }}
    runs-on: ubuntu-latest
    needs: pre-deploy
    if: needs.pre-deploy.outputs.can_deploy == 'true'
    environment:
      name: ${{ inputs.environment || 'production' }}
    steps:
      - name: Checkout
        uses: actions/checkout@v3
        
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
          cache: 'npm'
          
      - name: Install dependencies
        run: npm ci --prefer-offline
        
      - name: Run tests
        if: ${{ !inputs.skip_tests }}
        run: npm test
        
      - name: Build application
        run: npm run build
        
      - name: Deploy to server
        uses: appleboy/ssh-action@master
        env:
          ENVIRONMENT: ${{ inputs.environment || 'production' }}
        with:
          host: ${{ secrets.SERVER_HOST }}
          username: ${{ secrets.SERVER_USER }}
          key: ${{ secrets.SERVER_KEY }}
          envs: ENVIRONMENT
          script: |
            cd /app/${{ inputs.environment || 'production' }}
            git pull origin main
            docker-compose pull
            docker-compose up -d --build
            docker-compose exec -T app npm run migrate
            
            # 健康检查
            for i in {1..10}; do
              if curl -f http://localhost:3000/health; then
                echo "Health check passed"
                exit 0
              fi
              echo "Waiting for service... ($i/10)"
              sleep 10
            done
            echo "Health check failed"
            exit 1

      - name: Notify deployment
        if: always()
        uses: slackapi/slack-github-action@v1
        with:
          channel-id: ${{ secrets.SLACK_CHANNEL }}
          payload: |
            {
              "text": "Deployment ${{ job.status }} to ${{ inputs.environment || 'production' }}",
              "blocks": [
                {
                  "type": "section",
                  "text": {
                    "type": "mrkdwn",
                    "text": "*Deployment Status*\n${{ job.status }}"
                  }
                }
              ]
            }
        env:
          SLACK_BOT_TOKEN: ${{ secrets.SLACK_BOT_TOKEN }}
EOF

# 3. 创建优化后的package.json脚本
echo "[17.3] 创建优化后的构建脚本..."

cat > scripts/optimized-build.sh << 'EOF'
#!/bin/bash
set -e

echo "优化构建流程"
echo "================================"

# 并行安装依赖
echo "1. 并行安装依赖..."
npm ci --prefer-offline &

# 等待所有后台任务完成
wait

# 并行运行lint和测试
echo "2. 并行运行lint和类型检查..."
npm run lint &
LINT_PID=$!

npm run typecheck &
TYPECHECK_PID=$!

wait $LINT_PID
wait $TYPECHECK_PID

# 并行运行测试
echo "3. 并行运行测试..."
npm run test:unit -- --coverage &
UNIT_TEST_PID=$!

npm run test:integration &
INTEGRATION_TEST_PID=$!

wait $UNIT_TEST_PID
wait $INTEGRATION_TEST_PID

# 构建
echo "4. 构建应用..."
npm run build

echo "================================"
echo "构建完成！"
EOF

chmod +x scripts/optimized-build.sh

# 4. 创建缓存优化配置
echo "[17.4] 创建缓存优化配置..."

cat > .github/cache-config.json << 'EOF'
{
  "cache_strategies": {
    "npm": {
      "path": "node_modules",
      "key_files": ["package-lock.json"],
      "fallback_keys": ["node-"]
    },
    "docker": {
      "type": "gha",
      "mode": "max"
    },
    "webpack": {
      "path": ".next/cache",
      "key_files": ["package-lock.json", "next.config.js"]
    },
    "jest": {
      "path": ".jest-cache"
    }
  },
  "ttl": {
    "default": 86400,
    "production": 604800,
    "development": 3600
  }
}
EOF

# 5. 创建CI优化文档
echo "[17.5] 创建CI优化文档..."

cat > docs/ci-cd/optimization-guide.md << 'EOF'
# CI/CD 优化指南

## 优化策略

### 1. 依赖安装优化

#### 使用离线缓存
```yaml
- name: Cache node_modules
  uses: actions/cache@v3
  with:
    path: node_modules
    key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
    restore-keys: |
      ${{ runner.os }}-node-
```

#### 优化npm ci
- 使用 `npm ci` 而不是 `npm install`
- 添加 `--prefer-offline` 标志
- 使用 `--ignore-scripts` 跳过不必要的脚本

### 2. 测试执行优化

#### 缓存测试依赖
```yaml
- name: Cache Jest
  uses: actions/cache@v3
  with:
    path: .jest-cache
    key: ${{ runner.os }}-jest-${{ hashFiles('package-lock.json') }}
```

#### 并行执行测试
- 将单元测试和集成测试并行运行
- 使用 `test:unit` 和 `test:integration` 分离

#### 增量测试
- 使用 Jest 的 `--watch` 模式
- 只运行修改文件的测试

### 3. Docker构建优化

#### 多阶段构建
```dockerfile
FROM node:18-alpine AS builder
# ... 构建阶段

FROM node:18-alpine AS runner
# ... 运行阶段
```

#### 利用构建缓存
```yaml
- uses: docker/build-push-action@v4
  with:
    cache-from: type=gha
    cache-to: type=gha,mode=max
```

### 4. 部署优化

#### 滚动更新
```yaml
- name: Rolling update
  run: |
    kubectl rollout status deployment/app
    kubectl rollout undo deployment/app  # 如有问题自动回滚
```

#### 健康检查
```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:3000/health"]
  interval: 30s
  timeout: 10s
  retries: 3
```

## 性能指标

### 目标
- CI/CD 总时间 < 10分钟
- 测试覆盖率 > 80%
- 部署时间 < 5分钟

### 监控
- 跟踪每个步骤的时间
- 识别瓶颈
- 持续优化

## 最佳实践

1. **快速失败** - 尽早发现问题
2. **增量构建** - 只构建变化的部分
3. **并行执行** - 同时运行独立任务
4. **智能缓存** - 缓存不变的内容
5. **自动化一切** - 减少人工干预
6. **回滚机制** - 快速恢复
7. **监控告警** - 及时发现失败
EOF

echo "=========================================="
echo "任务17完成：CI/CD优化"
echo "=========================================="
