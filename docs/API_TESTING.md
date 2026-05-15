# API自动化测试指南

## 测试策略

### 1. 测试分层

```
┌─────────────────────────────────┐
│     E2E Tests (用户流程)        │
├─────────────────────────────────┤
│   Integration Tests (API)      │
├─────────────────────────────────┤
│     Unit Tests (函数)           │
└─────────────────────────────────┘
```

### 2. 测试覆盖范围

- **单元测试**: 独立函数和类
- **集成测试**: API端点、数据库交互
- **E2E测试**: 完整用户流程

## 测试工具

- Jest: 单元和集成测试
- Supertest: HTTP API测试
- Playwright: E2E测试

## 运行测试

```bash
# 运行所有测试
npm test

# 运行API测试
npm run test:api

# 运行单元测试
npm run test:unit

# 运行集成测试
npm run test:integration

# 运行E2E测试
npm run test:e2e

# 生成覆盖率报告
npm run test:coverage
```

## 测试文件结构

```
tests/
├── unit/
│   └── *.test.js
├── integration/
│   └── *.test.js
└── api/
    ├── basic-tests.spec.js
    ├── extended-tests.spec.js
    ├── boundary-tests.spec.js
    └── error-handling-tests.spec.js
```

## 测试用例设计

### 1. 基本功能测试

```javascript
test('创建用户 - 成功', async () => {
  const response = await request(app)
    .post('/api/users')
    .send({
      username: 'testuser',
      email: 'test@example.com',
      password: 'Password123!'
    });
  
  expect(response.status).toBe(201);
});
```

### 2. 边界条件测试

```javascript
test('用户名最小长度', async () => {
  const response = await request(app)
    .post('/api/users')
    .send({
      username: 'ab', // 太短
      email: 'test@example.com',
      password: 'Password123!'
    });
  
  expect(response.status).toBe(400);
});
```

### 3. 错误处理测试

```javascript
test('用户已存在', async () => {
  // 先创建用户
  await createTestUser();
  
  // 尝试创建相同用户
  const response = await request(app)
    .post('/api/users')
    .send({
      username: 'existinguser',
      email: 'existing@example.com',
      password: 'Password123!'
    });
  
  expect(response.status).toBe(409);
});
```

## 覆盖率目标

| 类型 | 目标覆盖率 |
|------|----------|
| Statements | ≥ 80% |
| Branches | ≥ 75% |
| Functions | ≥ 80% |
| Lines | ≥ 80% |

## 持续集成

测试在CI/CD流程中自动运行：

1. 提交代码触发CI
2. 运行所有测试
3. 生成覆盖率报告
4. 上传到Codecov
5. 失败则阻止合并

## 最佳实践

1. **测试独立性**: 每个测试独立运行
2. **清晰命名**: 测试名称描述清楚测试内容
3. **单一职责**: 每个测试只验证一个功能
4. **Mock依赖**: 隔离外部依赖
5. **全面覆盖**: 测试正常、边界、异常情况
6. **快速反馈**: 测试运行时间 < 10分钟

## 常见问题

### 测试失败怎么办？

1. 检查测试代码
2. 查看错误信息
3. 调试API响应
4. 验证测试数据
5. 修复代码或测试

### 如何添加新测试？

1. 在对应的测试文件中添加
2. 遵循命名约定
3. 提供清晰的测试描述
4. 确保测试可重复
