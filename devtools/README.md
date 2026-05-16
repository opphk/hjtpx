# HJTPX 开发者工具

HJTPX 验证码系统的开发者工具套件，用于简化开发、测试和集成流程。

## 功能特性

### 1. API 调试控制台
- 在线调试 HJTPX 验证码系统的 API 接口
- 支持 GET、POST、PUT、DELETE 等 HTTP 方法
- 可自定义请求头和请求体
- 显示响应状态、响应头和响应体
- 内置常用 API 预设请求
- 响应体 JSON 高亮显示

访问地址：`/devtools/api-console`

### 2. 验证码在线测试
- 支持滑动验证码、点击验证码、手势验证码和拼图验证码
- 提供完整的测试界面
- 实时显示测试结果
- 响应体格式化显示

访问地址：`/devtools/captcha-test`

### 3. SDK 代码生成器
- 自动生成多语言 SDK 调用代码
- 支持 Python、JavaScript、Go、Java 和 PHP
- 可配置 API 地址、应用 ID 和应用密钥
- 一键复制生成的代码

访问地址：`/devtools/code-generator`

### 4. 集成示例
- 提供完整的多语言集成示例代码
- 包括 Python、JavaScript、Go、Java 和 PHP
- 详细的代码注释
- 一键复制功能

访问地址：`/devtools/examples`

### 5. API 文档
- 完整的 API 接口文档
- 请求/响应示例
- 参数说明

访问地址：`/devtools/docs`

## 技术栈

- 后端：Go + Gin Web 框架
- 前端：HTML5 + Bootstrap 5 + JavaScript
- 图标：Bootstrap Icons
- 样式：BootCDN 提供的 Bootstrap 5 资源

## 使用说明

### 启动服务

```bash
cd /workspace/hjtpx/backend
go run ./cmd/api
```

### 访问开发者工具

启动服务后，可以通过以下地址访问开发者工具：

- 首页（默认跳转到 API 控制台）：`http://localhost:8080/devtools`
- API 调试控制台：`http://localhost:8080/devtools/api-console`
- 验证码在线测试：`http://localhost:8080/devtools/captcha-test`
- SDK 代码生成器：`http://localhost:8080/devtools/code-generator`
- 集成示例：`http://localhost:8080/devtools/examples`
- API 文档：`http://localhost:8080/devtools/docs`

## 文件结构

```
devtools/
├── README.md
├── static/
│   ├── css/    # 自定义样式文件
│   └── js/     # 自定义 JavaScript 文件
└── templates/
    ├── base.html          # 基础模板
    ├── api-console.html   # API 调试控制台
    ├── captcha-test.html  # 验证码在线测试
    ├── code-generator.html # SDK 代码生成器
    ├── examples.html      # 集成示例
    └── docs.html          # API 文档
```

## 路由配置

开发者工具的路由配置在 `/workspace/hjtpx/backend/internal/api/router/router.go` 文件中。

## 注意事项

- 所有资源通过 BootCDN 加载，确保网络连接正常
- API 调试控制台使用相对路径调用接口
- 建议在开发环境使用，生产环境根据需要调整访问权限
