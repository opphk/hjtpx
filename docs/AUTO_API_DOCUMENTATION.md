# API文档自动更新系统

## 概述

本文档说明了HJTPX项目中实现的API文档自动更新系统，包括Swagger配置、变更检测、版本管理和使用统计功能。

## 已实现功能

### 1. Swagger自动生成配置

**文件位置**: `src/backend/config/swagger-auto.js`

- 自动从路由文件生成OpenAPI 3.0规范
- 包含版本信息和构建元数据
- 支持JSON和YAML格式输出
- 可配置的输出目录

**主要导出**:
- `generateSwaggerSpec()` - 生成当前API规范
- `saveSwaggerSpec(outputDir)` - 保存规范到指定目录
- `getSwaggerOptions()` - 获取Swagger配置选项

### 2. API变更检测

**文件位置**: `scripts/check-api-changes.js`

功能:
- 自动比较当前API与上次保存的版本
- 检测新增、修改、删除的端点
- 识别破坏性变更(breaking changes)
- 生成详细变更报告

**命令行选项**:
```bash
node scripts/check-api-changes.js [options]

选项:
  --no-save          不要自动保存新版本
  --verbose, -v      显示详细变更信息
  --format <type>    输出格式: console, json, markdown
  --output <dir>     版本文件输出目录
  --help, -h        显示帮助信息
```

**示例**:
```bash
# 基本检查
npm run swagger:check-changes

# 详细输出
node scripts/check-api-changes.js --verbose

# JSON格式报告
node scripts/check-api-changes.js --format json --no-save
```

### 3. 文档版本管理

**文件位置**: `scripts/manage-api-versions.js`

功能:
- 列出所有保存的API版本
- 保存当前版本快照
- 查看版本详情
- 比较两个版本
- 导出版本为JSON/YAML
- 删除旧版本

**命令**:
```bash
# 列出所有版本
node scripts/manage-api-versions.js list

# 保存当前版本
node scripts/manage-api-versions.js save "添加了新搜索端点"

# 查看版本详情
node scripts/manage-api-versions.js show 1.0.0

# 比较两个版本
node scripts/manage-api-versions.js compare 1.0.0 1.1.0

# 详细差异分析
node scripts/manage-api-versions.js diff 1.0.0 1.1.0

# 导出版本
node scripts/manage-api-versions.js export 1.0.0 json
```

### 4. API使用统计

**文件位置**: 
- 中间件: `src/backend/middleware/apiUsage.js`
- 报告脚本: `scripts/generate-api-usage-report.js`

功能:
- 记录所有API调用
- 统计端点使用情况
- 追踪响应时间和错误率
- 生成使用报告
- 识别慢端点

**统计维度**:
- 总请求数
- 按端点统计
- 按HTTP方法统计
- 按状态码统计
- 平均响应时间
- 错误率

**命令**:
```bash
# 查看统计摘要
node scripts/generate-api-usage-report.js stats

# 显示使用最多的端点
node scripts/generate-api-usage-report.js top-endpoints --limit 20

# 显示慢端点
node scripts/generate-api-usage-report.js slow-endpoints

# 生成Markdown报告
node scripts/generate-api-usage-report.js markdown

# 导出统计数据
node scripts/generate-api-usage-report.js export json
```

### 5. CI/CD集成

**文件位置**: `.github/workflows/check-docs.yml`

工作流程:
1. **文档验证** - 验证Swagger规范有效性
2. **变更检测** - 检测API变更并生成报告
3. **质量检查** - 检查文档完整性
4. **PR评论** - 自动在PR中添加文档状态评论

**触发条件**:
- PR到main或develop分支
- 推送涉及路由文件时
- 手动触发

## NPM脚本

项目提供了便捷的NPM命令:

```bash
# 生成文档
npm run swagger:generate

# 验证文档
npm run swagger:validate

# 检查变更
npm run swagger:check-changes

# 版本管理
npm run swagger:versions      # 显示帮助
npm run swagger:save-version  # 保存当前版本
npm run swagger:list-versions # 列出所有版本
npm run swagger:compare       # 比较版本

# 文档CI检查
npm run docs:ci               # 完整CI检查
npm run docs:check           # 仅验证

# 使用报告
npm run swagger:usage         # 使用统计
npm run docs:report          # Markdown报告
```

## 文件结构

```
docs/
├── openapi.json              # 当前API规范
├── openapi.yaml              # YAML格式规范
├── api-usage/                 # 使用统计
│   ├── usage.json
│   ├── daily/
│   └── hourly/
└── versions/                 # API版本历史
    ├── versions.json          # 版本元数据
    ├── openapi-1.0.0-*.json   # 版本快照
    └── changes-*.json        # 变更报告
```

## 使用流程

### 开发流程

1. **开发新API**
   ```bash
   # 在 routes 目录添加新路由
   ```

2. **生成文档**
   ```bash
   npm run swagger:generate
   ```

3. **检查变更**
   ```bash
   npm run swagger:check-changes --verbose
   ```

4. **保存版本** (如有需要)
   ```bash
   npm run swagger:save-version "添加了用户资料API"
   ```

### CI/CD流程

```bash
# 本地测试CI检查
npm run docs:ci

# GitHub Actions会自动:
# 1. 验证Swagger规范
# 2. 检测API变更
# 3. 检查文档质量
# 4. 在PR中添加评论
```

## 最佳实践

1. **定期保存版本** - 在重要版本发布时保存快照
2. **检查破坏性变更** - 确保没有意外的破坏性变更
3. **补充文档注释** - 为新端点添加完整的Swagger注释
4. **监控API使用** - 定期查看使用报告优化性能
5. **维护变更日志** - 记录重要变更的详细信息

## 技术栈

- **swagger-jsdoc**: OpenAPI规范生成
- **swagger-ui-express**: API文档UI
- **js-yaml**: YAML格式支持
- **文件系统**: 版本存储

## 扩展功能

可以进一步添加的功能:
1. 自动生成SDK代码
2. API使用配额管理
3. 自动生成变更日志
4. Slack/Teams通知集成
5. API文档版本对比UI
