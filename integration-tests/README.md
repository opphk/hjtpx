# HJTPX 集成测试 v15.0

## 概述

本目录包含HJTPX v15.0系统的集成测试脚本，用于验证系统各组件之间的集成功能。

## 目录结构

```
integration-tests/
├── README.md                          # 本文件
├── config/
│   └── test-config.yaml               # 测试配置
├── scripts/
│   ├── run-tests.sh                   # 测试运行脚本
│   └── setup-test-env.sh             # 测试环境设置脚本
├── tests/
│   ├── captcha/
│   │   ├── slider_test.go            # 滑块验证码测试
│   │   ├── click_test.go             # 点击验证码测试
│   │   ├── lianliankan_test.go       # 连连看验证码测试
│   │   ├── voice_test.go             # 语音验证码测试
│   │   └── seamless_test.go          # 无感验证测试
│   ├── auth/
│   │   ├── login_test.go             # 登录测试
│   │   ├── register_test.go          # 注册测试
│   │   ├── mfa_test.go               # MFA测试
│   │   └── token_test.go             # Token测试
│   ├── admin/
│   │   ├── application_test.go       # 应用管理测试
│   │   ├── blacklist_test.go         # 黑名单测试
│   │   ├── logs_test.go              # 日志测试
│   │   └── stats_test.go             # 统计测试
│   ├── environment/
│   │   ├── fingerprint_test.go       # 指纹检测测试
│   │   ├── proxy_test.go             # 代理检测测试
│   │   └── bot_detection_test.go     # Bot检测测试
│   └── performance/
│       ├── load_test.go              # 负载测试
│       └── stress_test.go            # 压力测试
├── fixtures/
│   └── test-data.json                # 测试数据
├── reports/
│   └── .gitkeep                      # 测试报告输出目录
└── docker-compose-test.yml           # 测试环境Docker Compose配置

## 测试环境要求

- Go 1.21+
- Docker 和 Docker Compose
- PostgreSQL 14+
- Redis 7+
- 至少4GB可用内存

## 快速开始

### 1. 启动测试环境

```bash
# 进入集成测试目录
cd integration-tests

# 启动测试环境
docker-compose -f docker-compose-test.yml up -d

# 等待服务启动
./scripts/setup-test-env.sh
```

### 2. 运行测试

```bash
# 运行所有测试
./scripts/run-tests.sh

# 运行特定模块测试
go test ./tests/captcha/... -v

# 运行特定测试用例
go test ./tests/captcha/slider_test.go -v -run TestSliderCaptchaGenerate
```

### 3. 查看测试报告

```bash
# 查看测试报告
ls -la reports/

# 生成HTML报告
go test ./tests/... -html=reports/report.html
```

## 测试模块说明

### 验证码模块 (captcha)

测试各种验证码类型的生成和验证功能：

- **滑块验证码**：测试滑块位置验证、容差范围、轨迹分析
- **点击验证码**：测试点击坐标验证、图案匹配
- **连连看验证码**：测试配对验证、图案匹配
- **语音验证码**：测试音频生成、文字识别
- **无感验证**：测试行为分析、信任等级评估

### 认证模块 (auth)

测试用户认证相关功能：

- **登录测试**：用户名密码验证、错误尝试限制
- **注册测试**：用户注册、邮箱验证
- **MFA测试**：多因素认证设置和验证
- **Token测试**：Token生成、刷新、过期处理

### 管理模块 (admin)

测试管理后台功能：

- **应用管理**：应用创建、更新、删除
- **黑名单管理**：IP、指纹、用户ID黑名单
- **日志查询**：验证日志查询、导出
- **统计分析**：统计数据查询、报表生成

### 环境检测模块 (environment)

测试环境检测功能：

- **指纹检测**：Canvas、WebGL、音频指纹
- **代理检测**：VPN、代理服务器检测
- **Bot检测**：自动化工具检测

### 性能模块 (performance)

测试系统性能：

- **负载测试**：正常负载下的响应时间
- **压力测试**：极限负载下的系统表现

## 测试配置

编辑 `config/test-config.yaml` 来自定义测试配置：

```yaml
server:
  host: localhost
  port: 8080
  base_url: http://localhost:8080/api/v1

database:
  host: localhost
  port: 5432
  user: hjtpx_test
  password: hjtpx_test_password
  name: hjtpx_test
  sslmode: disable

redis:
  host: localhost
  port: 6379
  password: hjtpx_test_password
  db: 1

test:
  timeout: 30s
  max_retries: 3
  parallel: 4

admin:
  username: admin
  password: admin123

user:
  username: testuser
  password: Test123456
  email: test@example.com
```

## 编写新测试

### 测试模板

```go
package your_module

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestYourFeature(t *testing.T) {
    // 准备测试数据
    testData := setupTestData()
    
    // 执行测试
    result, err := yourFunction(testData)
    
    // 验证结果
    require.NoError(t, err, "测试函数不应该返回错误")
    assert.NotNil(t, result, "结果不应该为空")
    assert.Equal(t, expectedValue, result.ActualValue, "值不匹配")
}

func TestYourFeatureEdgeCases(t *testing.T) {
    // 测试边界情况
    testCases := []struct {
        name  string
        input interface{}
        expected interface{}
    }{
        {"空输入", nil, nil},
        {"最大输入", maxValue, expectedMax},
        {"最小输入", minValue, expectedMin},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := yourFunction(tc.input)
            assert.NoError(t, err)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

### 测试最佳实践

1. **使用表驱动测试**：对于多个相似测试用例，使用表驱动测试
2. **设置前置条件和清理**：在测试前后正确设置和清理数据
3. **使用清晰的测试名称**：测试名称应清晰表达测试内容
4. **独立的测试**：每个测试应该是独立的，不依赖其他测试
5. **有意义的断言**：使用有意义的断言消息
6. **处理异步操作**：对于异步操作，使用适当的等待机制

## 持续集成

### GitHub Actions 配置

在 `.github/workflows/integration-tests.yml` 中配置：

```yaml
name: Integration Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:14
        env:
          POSTGRES_USER: hjtpx_test
          POSTGRES_PASSWORD: hjtpx_test_password
          POSTGRES_DB: hjtpx_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      
      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run integration tests
        run: |
          cd integration-tests
          ./scripts/run-tests.sh
      
      - name: Upload test results
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: test-results
          path: integration-tests/reports/
```

## 故障排查

### 测试失败

1. **检查测试环境**：确保所有服务正在运行
   ```bash
   docker-compose -f docker-compose-test.yml ps
   ```

2. **查看详细日志**：
   ```bash
   docker-compose -f docker-compose-test.yml logs -f
   ```

3. **重新启动测试环境**：
   ```bash
   docker-compose -f docker-compose-test.yml down
   docker-compose -f docker-compose-test.yml up -d
   ```

### 连接问题

1. **检查端口占用**：
   ```bash
   lsof -i :8080
   lsof -i :5432
   lsof -i :6379
   ```

2. **检查防火墙**：
   ```bash
   sudo iptables -L -n | grep 8080
   ```

## 相关文档

- [API文档](../docs/API接口文档.md)
- [开发者指南](../docs/开发者指南.md)
- [部署文档](../docs/部署文档.md)
- [故障排查手册](../docs/故障排查手册.md)

---

**最后更新**: 2026-05-19
**当前版本**: v15.0
