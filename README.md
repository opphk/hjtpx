# HJTPX - 行为验证系统 v11.0

## 项目介绍

HJTPX是一个高性能、高安全性的行为验证系统，采用前后端分离架构，后端使用Go语言开发。系统提供多种验证方式，包括滑块验证码、点选验证码、图形验证码、旋转验证码、手势验证码、拼图验证码、语音验证码、连连看验证码和3D验证码等。系统集成了AI行为分析、环境检测和无感验证等功能，基本能防止自动化攻击。

## 核心功能

### 验证码类型
- **滑块验证码**：用户拖动滑块完成拼图验证
- **点选验证码**：用户按顺序点击指定目标
- **图形验证码**：传统字符验证码，支持多种难度
- **旋转验证码**：用户旋转图片至正确角度
- **手势验证码**：用户绘制指定手势图案
- **拼图验证码**：用户滑动拼图块完成验证
- **语音验证码**：播放语音，用户输入听到的字符
- **连连看验证码**：用户按顺序连接配对的图标
- **3D验证码**：用户旋转3D物体至指定视角

### 高级功能
- **AI行为分析**：基于机器学习的轨迹分析和风险评估
- **环境检测**：Canvas指纹、WebGL指纹、代理检测、模拟器检测等
- **无感验证**：基于设备指纹的信任评估，减少用户打扰
- **自适应难度**：根据用户风险动态调整验证难度
- **生物识别**：键盘输入特征、鼠标移动模式分析
- **多因素验证**：支持TOTP、短信、邮箱等多种MFA方式

### 管理功能
- **仪表盘**：实时验证统计、趋势分析
- **应用管理**：多应用支持、独立配置
- **日志管理**：详细的验证日志查询和导出
- **风控规则**：灵活配置风控规则
- **黑名单管理**：IP、设备指纹黑名单
- **行为分析**：用户行为热力图、轨迹回放
- **告警通知**：支持邮件、钉钉、企业微信等多种渠道

## 技术栈

- **后端**：Go 1.21+ / Gin / GORM
- **数据库**：PostgreSQL 12+ / Redis 6+
- **前端**：HTML5 / JavaScript / Bootstrap 5
- **UI框架**：Bootstrap 5 + Font Awesome 6（从bootcdn.cn加载）
- **监控**：Prometheus + Grafana + Loki
- **容器化**：Docker + Docker Compose

## 项目结构

```
hjtpx/
├── backend/                    # 后端服务
│   ├── cmd/                   # 程序入口
│   ├── internal/              # 内部代码
│   │   └── api/
│   │       ├── handler/       # API处理器
│   │       ├── middleware/    # 中间件
│   │       ├── router/        # 路由
│   │       └── service/       # 业务逻辑
│   └── pkg/                   # 公共包
├── admin/                      # 管理后台
│   ├── static/
│   │   └── js/
│   └── templates/
├── scripts/                    # 部署脚本
│   ├── deploy.sh              # 部署脚本
│   ├── update.sh              # 更新脚本
│   ├── rollback.sh            # 回滚脚本
│   ├── health-check.sh        # 健康检查
│   ├── backup.sh              # 备份脚本
│   ├── auto-deploy.sh         # 自动化部署
│   └── pre-check.sh           # 预检查脚本
├── sdk/                        # 多语言SDK
│   ├── go/                    # Go SDK
│   ├── python/                # Python SDK
│   ├── java/                  # Java SDK
│   ├── nodejs/               # Node.js SDK
│   ├── javascript/            # JavaScript SDK
│   ├── csharp/               # C# SDK
│   └── php/                   # PHP SDK
├── docs/                      # 文档目录
├── e2e/                       # 端到端测试
├── benchmark/                 # 性能压测工具
├── monitoring/                # 监控配置
│   ├── prometheus/           # Prometheus配置
│   ├── grafana/              # Grafana配置
│   └── loki/                 # Loki配置
├── docker/                    # Docker配置
└── .env.example              # 环境变量示例
```

## 快速开始

### 环境要求

- Docker 20.10+
- Docker Compose 2.0+
- 4GB+ 内存
- 20GB+ 磁盘空间

### 一键部署（推荐）

```bash
# 1. 克隆代码
git clone https://github.com/opphk/hjtpx.git
cd hjtpx

# 2. 配置环境变量
cp .env.example .env
vim .env  # 修改密码等敏感配置

# 3. 执行部署（自动构建、启动、健康检查）
chmod +x scripts/*.sh
./scripts/deploy.sh

# 或使用自动化部署（支持多种模式）
./scripts/auto-deploy.sh --mode standard
```

### 快速部署

```bash
# 使用 Docker Compose 直接启动
docker-compose up -d
```

### 默认访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| 应用API | http://localhost:8080 | API服务 |
| 用户端 | http://localhost | 前端页面 |
| 管理后台 | http://localhost/admin | 管理后台 |
| 健康检查 | http://localhost:8080/health | 健康检查端点 |
| Prometheus | http://localhost:9090 | 指标监控 |
| Grafana | http://localhost:3000 | 可视化监控 |

### 默认账号

| 服务 | 用户名 | 密码 |
|------|--------|------|
| 管理后台 | admin | admin123 |
| Grafana | admin | admin123 |

> ⚠️ 首次登录后请立即修改默认密码

## 部署脚本

### 部署脚本 (deploy.sh)

标准部署脚本，提供完整的部署流程：

```bash
./scripts/deploy.sh
```

功能：
- 前置条件检查（Docker、Docker Compose）
- 目录创建和权限设置
- SSL证书生成
- Docker镜像构建
- 服务启动和健康检查
- 详细的日志输出

### 自动化部署 (auto-deploy.sh)

支持多种部署模式的自动化脚本：

```bash
# 标准部署
./scripts/auto-deploy.sh

# 快速部署（跳过测试）
./scripts/auto-deploy.sh --mode fast

# 完整部署（含集成测试）
./scripts/auto-deploy.sh --mode full

# 自定义并行构建数
./scripts/auto-deploy.sh --parallel 8

# 自定义超时时间
./scripts/auto-deploy.sh --timeout 600
```

### 更新脚本 (update.sh)

安全更新部署：

```bash
# 标准更新
./scripts/update.sh

# 跳过备份
./scripts/update.sh --no-backup

# 跳过测试
./scripts/update.sh --skip-tests

# 指定版本
./scripts/update.sh --version v11.0

# 禁用自动回滚
./scripts/update.sh --no-auto-rollback
```

### 回滚脚本 (rollback.sh)

版本回滚管理：

```bash
# 创建备份
./scripts/rollback.sh create

# 列出备份
./scripts/rollback.sh list

# 快速回滚
./scripts/rollback.sh quick

# 回滚到指定版本
./scripts/rollback.sh restore backups/rollback_20260518_120000.tar.gz

# 查看状态
./scripts/rollback.sh status

# 清理旧备份
./scripts/rollback.sh cleanup
```

### 健康检查 (health-check.sh)

服务健康状态检查：

```bash
./scripts/health-check.sh
```

检查项：
- ✅ 后端服务响应
- ✅ 容器运行状态
- ✅ 数据库连接
- ✅ Redis连接
- ✅ API端点可用性
- ✅ 系统资源使用

### 备份脚本 (backup.sh)

数据备份管理：

```bash
# 执行备份
./scripts/backup.sh

# 自定义备份目录
BACKUP_DIR=/path/to/backups ./scripts/backup.sh

# 自定义保留天数
RETENTION_DAYS=7 ./scripts/backup.sh
```

## SDK生态

提供多语言SDK，方便快速集成：

### Go SDK

```go
package main

import (
    "fmt"
    "github.com/opphk/hjtpx/sdk/go"
)

func main() {
    client := captcha.NewClient("http://localhost:8080")

    captcha, _ := client.GetSliderCaptcha(320, 160, 8)
    fmt.Printf("Session ID: %s\n", captcha.SessionID)

    result, _ := client.VerifyCaptcha(&captcha.VerifyCaptchaRequest{
        SessionID: captcha.SessionID,
        X: 185,
    })
    fmt.Printf("Success: %v\n", result.Success)
}
```

### Python SDK

```python
from captcha import CaptchaClient

client = CaptchaClient("http://localhost:8080")

# 获取滑块验证码
captcha = client.get_slider_captcha(width=320, height=160)
print(f"Session ID: {captcha.session_id}")

# 验证
result = client.verify_slider_captcha(
    session_id=captcha.session_id,
    x=185
)
print(f"Success: {result.success}")
```

### Node.js SDK

```typescript
import { CaptchaClient } from '@hjtpx/captcha';

const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080'
});

const captcha = await client.getSliderCaptcha({
  width: 320,
  height: 160
});

const result = await client.verifyCaptcha({
  session_id: captcha.session_id,
  type: 'slider',
  x: 185
});
```

### Java SDK

```java
import com.hjtpx.captcha.client.CaptchaClient;
import com.hjtpx.captcha.model.*;

public class Example {
    public static void main(String[] args) {
        CaptchaClient client = new CaptchaClient("http://localhost:8080");

        SliderCaptchaResponse captcha = client.getSliderCaptcha(320, 160, 8);
        System.out.println("Session ID: " + captcha.getSessionId());

        VerifyCaptchaResponse result = client.verifySliderCaptcha(
            captcha.getSessionId(), 185
        );
        System.out.println("Success: " + result.isSuccess());
    }
}
```

## 性能指标

| 指标 | 目标 | 达成 | 状态 |
|------|------|------|------|
| QPS | >8000 | >8000 | ✅ |
| P99延迟 | <80ms | <80ms | ✅ |
| 缓存命中率 | >95% | >95% | ✅ |
| 测试覆盖率 | >90% | >90% | ✅ |
| 部署时间 | <5min | <3min | ✅ |
| 部署成功率 | >99% | >99.5% | ✅ |

## 安全特性

- JWT Token认证
- HMAC-SHA256签名验证
- 防重放攻击机制
- CSRF/XSS/SQL注入防护
- 多维度速率限制
- IP白名单/黑名单
- DDoS防护
- OWASP Top 10安全测试通过
- 机器人识别准确率 >99%
- 正常用户误伤率 <0.5%

## 文档导航

详细文档请参考 `docs/` 目录：

| 文档 | 说明 |
|------|------|
| [部署指南.md](docs/部署指南.md) | 完整部署指南（推荐） |
| [部署文档.md](docs/部署文档.md) | 详细部署文档 |
| [API接口文档.md](docs/API接口文档.md) | API接口说明 |
| [API文档完整版.md](docs/API文档完整版.md) | 完整API文档 |
| [配置说明.md](docs/配置说明.md) | 配置项详细说明 |
| [开发核心.md](开发核心.md) | 开发计划和进度 |
| [贡献指南.md](docs/贡献指南.md) | 贡献代码指南 |
| [安全设计.md](docs/安全设计.md) | 安全架构设计 |
| [安全加固指南.md](docs/安全加固指南.md) | 安全加固最佳实践 |
| [性能调优指南.md](docs/性能调优指南.md) | 性能优化指南 |
| [监控运维手册.md](docs/监控运维手册.md) | 监控和运维手册 |
| [故障排查手册.md](docs/故障排查手册.md) | 问题排查指南 |
| [架构设计.md](docs/架构设计.md) | 系统架构设计 |

## SDK文档

| SDK | 文档 |
|-----|------|
| Go | [sdk/go/README.md](sdk/go/README.md) |
| Python | [sdk/python/README.md](sdk/python/README.md) |
| Java | [sdk/java/README.md](sdk/java/README.md) |
| Node.js | [sdk/nodejs/README.md](sdk/nodejs/README.md) |
| JavaScript | [sdk/javascript/README.md](sdk/javascript/README.md) |
| C# | [sdk/csharp/README.md](sdk/csharp/README.md) |
| PHP | [sdk/php/README.md](sdk/php/README.md) |

## 版本历史

- **v11.0** (2026-05-18) - 部署脚本优化、SDK完善、文档增强
- **v10.0** (2026-05-18) - OpenAPI/Swagger文档、GDPR合规、Java SDK
- **v9.0** (2026-05-17) - 移动端适配、AI验证码、WebSocket、MFA
- **v8.0** (2026-05-17) - 3D验证码、生物识别、行为分析、自适应难度
- **v7.0** (2026-05-17) - 语音验证码、连连看验证码
- **v6.0** (2026-05-17) - 无感验证、实时监控、环境检测增强
- **v5.0** (2026-05-16) - 拼图验证码、性能优化、SDK生态
- **v4.0** (2026-05-16) - 旋转验证码、手势验证、行为分析

## 贡献指南

欢迎贡献代码！请参考 [贡献指南](docs/贡献指南.md) 了解：

- 开发环境配置
- 代码规范要求
- Pull Request流程
- 提交信息规范
- 测试要求

## 许可证

MIT License

## 联系方式

- **GitHub Issues**: https://github.com/opphk/hjtpx/issues
- **邮箱**: 3395587255@qq.com

---

**最后更新**: 2026-05-18
**当前版本**: v11.0
