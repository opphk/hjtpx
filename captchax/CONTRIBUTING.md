# CaptchaX 贡献指南

感谢您对 CaptchaX 项目的关注！我们欢迎各种形式的贡献，包括但不限于代码提交、文档改进、问题反馈等。

## 目录

- [行为准则](#行为准则)
- [开始贡献](#开始贡献)
- [开发环境](#开发环境)
- [代码规范](#代码规范)
- [提交规范](#提交规范)
- [测试要求](#测试要求)
- [文档要求](#文档要求)
- [Pull Request 流程](#pull-request-流程)
- [版本发布](#版本发布)

---

## 行为准则

我们期望所有贡献者遵守以下行为准则：

1. **尊重**：尊重他人的观点和贡献
2. **包容**：欢迎不同背景和经验水平的贡献者
3. **专业**：保持专业和建设性的交流
4. **诚实**：坦诚面对错误和问题

---

## 开始贡献

### 通过 Issue 反馈

- 搜索现有 Issue，避免重复
- 使用清晰的问题描述模板
- 提供复现步骤和环境信息
- 附上相关日志和截图

### 通过 Pull Request 贡献

1. Fork 仓库
2. 创建特性分支
3. 进行开发和测试
4. 提交 Pull Request
5. 等待代码审查
6. 根据反馈进行修改
7. 合并代码

---

## 开发环境

### 环境要求

| 软件 | 版本要求 |
|------|----------|
| Go | 1.21+ |
| Node.js | 18+ |
| PostgreSQL | 13+ |
| Redis | 6.0+ |
| Docker | 20.10+ |

### 本地开发

```bash
# 1. Fork 并克隆仓库
git clone https://github.com/your-username/captchax.git
cd captchax

# 2. 添加上游仓库
git remote add upstream https://github.com/captchax/captchax.git

# 3. 创建特性分支
git checkout -b feat/your-feature-name

# 4. 安装依赖
go mod download

# 5. 配置本地环境
cp .env.example .env.local

# 6. 启动开发服务
docker-compose up -d postgres redis

# 7. 运行迁移
psql -h localhost -U postgres -d captcha_db -f migrations/001_initial_schema.sql

# 8. 启动服务
go run ./cmd/server/main.go
go run ./cmd/admin/main.go
```

### 前端开发

```bash
cd web

# 安装依赖
npm install

# 开发模式
npm run dev

# 运行测试
npm test

# 构建
npm run build
```

---

## 代码规范

### Go 代码规范

1. **格式化**：使用 `gofmt` 或 `goimports`
2. **命名规范**：
   - 包名：小写，无下划线
   - 函数名：PascalCase
   - 变量名：camelCase
   - 常量名：全大写下划线分隔
3. **注释规范**：使用完整的句子，以被命名的元素开头
4. **错误处理**：始终检查错误，不要忽略 `_`

```go
// Good
func GenerateSliderCaptcha(ctx context.Context, req *SliderRequest) (*SliderResponse, error) {
    if req.Width <= 0 {
        return nil, ErrInvalidWidth
    }
    // ...
}

// Bad
func gen_slider(w int) (string, error) {
    // ...
}
```

### 前端代码规范

1. **ESLint**：遵循项目 ESLint 配置
2. **Prettier**：使用 Prettier 格式化
3. **TypeScript**：优先使用 TypeScript
4. **组件规范**：
   - 组件文件使用 PascalCase
   - 样式文件使用 kebab-case
   - 测试文件使用 `.test.ts`

```typescript
// Good
interface CaptchaProps {
  appId: string;
  onSuccess: (token: string) => void;
  onError?: (error: Error) => void;
}

// Bad
const Captcha = (props: any) => {
  // ...
};
```

### SQL 代码规范

1. **关键字**：大写
2. **表名/列名**：小写下划线分隔
3. **缩进**：4 空格
4. **注释**：`--` 单行注释

```sql
-- Good
SELECT
    id,
    username,
    created_at
FROM users
WHERE status = 'active';

-- Bad
SELECT id,username,created_at FROM users WHERE status='active';
```

---

## 提交规范

### Commit Message 格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type 类型

| 类型 | 说明 |
|------|------|
| feat | 新功能 |
| fix | Bug 修复 |
| docs | 文档更新 |
| style | 代码格式（不影响功能） |
| refactor | 重构（不是新功能或修复） |
| perf | 性能优化 |
| test | 测试相关 |
| chore | 构建/工具相关 |

### Scope 范围

| 范围 | 说明 |
|------|------|
| api | API 相关 |
| web | 前端相关 |
| db | 数据库相关 |
| cache | 缓存相关 |
| security | 安全相关 |
| ci | CI/CD 相关 |

### 示例

```
feat(api): add batch verification endpoint

Add new /api/v2/captcha/batch/verify endpoint to support
batch verification of multiple captchas in a single request.

Closes #123
```

```
fix(cache): resolve Redis connection pool exhaustion

Fixed issue where high concurrency could exhaust Redis connection
pool, causing subsequent requests to fail. Added connection pool
monitoring and automatic cleanup.

Fixes #456
```

---

## 测试要求

### 测试覆盖率

| 模块 | 最低覆盖率 |
|------|-----------|
| API handlers | 80% |
| Service layer | 85% |
| Core logic | 90% |
| Security | 95% |

### 单元测试

```go
// Good
func TestSliderVerify(t *testing.T) {
    tests := []struct {
        name     string
        req      *VerifyRequest
        wantErr  bool
        wantCode int
    }{
        {
            name:     "valid verification",
            req:      &VerifyRequest{CaptchaID: "valid", TargetX: 100},
            wantErr:  false,
            wantCode: 200,
        },
        {
            name:     "expired captcha",
            req:      &VerifyRequest{CaptchaID: "expired", TargetX: 100},
            wantErr:  true,
            wantCode: 400,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

### 集成测试

```go
func TestSliderCaptchaE2E(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping e2e test in short mode")
    }

    // Setup test database
    // ...

    // Test flow
    client := NewTestClient()
    captcha, err := client.CreateSliderCaptcha(ctx, req)
    require.NoError(t, err)

    result, err := client.VerifySlider(ctx, &VerifyRequest{
        CaptchaID: captcha.ID,
        TargetX:   captcha.TargetX,
    })
    require.NoError(t, err)
    assert.True(t, result.Success)
}
```

### 前端测试

```typescript
describe('CaptchaX Component', () => {
  it('should render captcha container', () => {
    render(<CaptchaX appId="test" />);
    expect(screen.getByTestId('captcha-container')).toBeInTheDocument();
  });

  it('should call onSuccess when verification succeeds', async () => {
    const onSuccess = jest.fn();
    render(<CaptchaX appId="test" onSuccess={onSuccess} />);

    // Simulate user interaction
    // ...

    expect(onSuccess).toHaveBeenCalled();
  });
});
```

---

## 文档要求

### 新增功能文档

每个新功能必须包含以下文档：

1. **API 文档**：更新 `docs/API.md`，包含接口说明、参数、返回值
2. **SDK 更新**：如涉及 SDK，更新对应语言的 SDK
3. **示例代码**：在 `examples/` 目录添加使用示例
4. **更新日志**：在 `CHANGELOG.md` 添加条目

### 文档格式

```markdown
## 功能名称

### 概述
简要说明功能用途。

### API
- **端点**：`POST /api/v1/xxx`
- **参数**：
  | 参数 | 类型 | 必填 | 说明 |
  |------|------|------|------|
  | xxx | string | 是 | xxx |
- **响应**：
  ```json
  {
    "code": 200,
    "data": {}
  }
  ```

### 示例
```bash
curl -X POST http://localhost:8080/api/v1/xxx \
  -H "Content-Type: application/json" \
  -d '{}'
```
```

---

## Pull Request 流程

### 创建 PR

1. 确保分支基于 `main` 最新代码
2. 填写 PR 模板所有必填项
3. 关联相关 Issue
4. 添加必要的标签

### PR 模板

```markdown
## 变更内容

<!-- 简要描述本次变更 -->

## 变更类型

- [ ] 新功能 (feat)
- [ ] Bug 修复 (fix)
- [ ] 文档更新 (docs)
- [ ] 代码重构 (refactor)
- [ ] 性能优化 (perf)
- [ ] 测试相关 (test)
- [ ] 构建/工具 (chore)

## 影响范围

<!-- 描述本次变更对哪些模块有影响 -->

## 测试计划

- [ ] 添加了新的单元测试
- [ ] 现有测试通过
- [ ] 手动测试验证

## 相关 Issue

Closes #xxx
Fixes #xxx
```

### 代码审查清单

- [ ] 代码风格符合规范
- [ ] 有适当的测试覆盖
- [ ] 更新了相关文档
- [ ] 没有引入安全问题
- [ ] 没有引入性能问题
- [ ] 提交信息清晰准确

### 审查反馈

1. 礼貌、具体、建设性
2. 解释原因，不仅仅是指出问题
3. 区分必须修改和建议修改
4. 提供修改建议

---

## 版本发布

### 版本号规则

采用语义化版本 `MAJOR.MINOR.PATCH`：

- `MAJOR`：不兼容的 API 变更
- `MINOR`：向后兼容的功能新增
- `PATCH`：向后兼容的问题修复

### 发布流程

1. **准备发布**
   ```bash
   git checkout main
   git pull upstream main
   ```

2. **更新版本号**
   ```bash
   # 更新 VERSION 文件
   echo "2.0.0" > VERSION

   # 更新 go.mod
   # 更新 package.json
   ```

3. **更新 CHANGELOG**
   ```bash
   git changelog --all --version 2.0.0
   ```

4. **创建发布标签**
   ```bash
   git tag -a v2.0.0 -m "Release version 2.0.0"
   git push upstream v2.0.0
   ```

5. **构建发布**
   ```bash
   make release
   ```

6. **创建 GitHub Release**
   - 使用标签作为版本号
   - 复制 CHANGELOG 内容
   - 上传构建产物

### 发布检查清单

- [ ] 所有测试通过
- [ ] 文档已更新
- [ ] CHANGELOG 已更新
- [ ] 版本号已更新
- [ ] 构建成功
- [ ] Docker 镜像已推送
- [ ] GitHub Release 已创建

---

## 获取帮助

- **文档**：[https://captchax.example.com/docs](https://captchax.example.com/docs)
- **Issue**：[GitHub Issues](https://github.com/captchax/captchax/issues)
- **讨论**：[GitHub Discussions](https://github.com/captchax/captchax/discussions)
- **邮件**：support@example.com

---

再次感谢您的贡献！
