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
