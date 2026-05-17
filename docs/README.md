# 行为验证系统 API 文档

完整的行为验证系统API文档集合，包含详细的接口说明、使用示例、SDK和工具链。

## 📚 文档目录

### 1. API 文档
- [API接口文档.md](API接口文档.md) - 原始API文档
- [API文档完整版.md](API文档完整版.md) - **完整详细的API文档**（推荐阅读）
- [openapi.yaml](openapi.yaml) - Swagger/OpenAPI 3.0 规范

### 2. 在线文档
- [api-docs.html](api-docs.html) - 美观的在线API文档页面（可直接在浏览器打开）

### 3. 使用指南
- [使用指南.md](使用指南.md) - **用户快速上手指南**（推荐新用户阅读）
- [最佳实践.md](最佳实践.md) - **最佳实践指南**（推荐开发者阅读）

### 4. 开发工具
- [postman-collection.json](postman-collection.json) - Postman API集合，可直接导入使用

### 5. SDK
- [Go SDK](../sdk/go/) - Go语言SDK，包含示例代码
- [JavaScript SDK](../sdk/javascript/) - JavaScript/Node.js SDK
- [Python SDK](../sdk/python/) - Python SDK

### 6. 其他文档
- [架构设计.md](架构设计.md) - 系统架构设计文档
- [部署文档.md](部署文档.md) - 部署指南
- [配置说明.md](配置说明.md) - 配置项详细说明
- [安全设计.md](安全设计.md) - 安全设计说明
- [安全加固指南.md](安全加固指南.md) - 安全加固最佳实践
- [性能调优指南.md](性能调优指南.md) - 性能优化指南
- [监控运维手册.md](监控运维手册.md) - 监控和运维手册
- [故障排查手册.md](故障排查手册.md) - 问题排查指南
- [运维手册.md](运维手册.md) - 日常运维手册
- [贡献指南.md](贡献指南.md) - 贡献代码指南

---

## 🚀 快速开始

### 方式一：在线文档（推荐）
直接在浏览器中打开 `api-docs.html` 查看交互式API文档。

### 方式二：Postman
1. 打开Postman
2. 导入 `postman-collection.json`
3. 配置环境变量（baseUrl、apiKey等）
4. 开始测试API

### 方式三：Swagger UI
使用Swagger UI加载 `openapi.yaml` 文件：
```bash
# 使用 Docker 运行 Swagger UI
docker run -p 8081:8080 -e SWAGGER_JSON=/openapi.yaml -v $(pwd)/openapi.yaml:/openapi.yaml swaggerapi/swagger-ui
```
然后访问 http://localhost:8081

### 方式四：使用SDK

#### Go SDK
```go
package main

import (
    "fmt"
    "github.com/hjtpx/captcha-sdk-go"
)

func main() {
    client := captcha.NewClient("http://localhost:8080", "your-api-key")
    
    // 获取滑块验证码
    resp, err := client.GetSliderCaptcha(320, 160, 8)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Session ID: %s\n", resp.SessionID)
}
```

#### JavaScript SDK
```javascript
import CaptchaClient from './captcha-sdk-js';

const client = new CaptchaClient('http://localhost:8080', {
    apiKey: 'your-api-key'
});

// 获取滑块验证码
const resp = await client.getSliderCaptcha({ width: 320, height: 160 });
console.log('Session ID:', resp.session_id);
```

#### Python SDK
```python
from captcha_sdk import CaptchaClient

client = CaptchaClient('http://localhost:8080', api_key='your-api-key')

# 获取滑块验证码
resp = client.get_slider_captcha(width=320, height=160, tolerance=8)
print(f'Session ID: {resp.session_id}')
```

---

## 📋 API 概览

### 用户端 API
| 分类 | 说明 |
|------|------|
| 验证码API | 滑块、图形、点选、旋转、手势验证码 |
| 认证API | 注册、登录、Token刷新、登出、密码重置 |
| 用户资料API | 获取/更新用户信息、修改密码 |
| 环境检测API | 设备指纹、风险检测 |

### 管理端 API
| 分类 | 说明 |
|------|------|
| 统计API | 仪表盘、趋势、风险分布 |
| 应用管理API | 应用CRUD、API密钥管理 |
| 日志API | 日志查询、详情、导出 |
| 黑名单API | 黑名单管理 |
| 风控规则API | 规则配置和管理 |
| 高级分析API | 行为分析、攻击趋势、热力图 |
| 实时监控API | 实时指标、WebSocket监控 |

---

## 🔧 错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 10001 | 验证失败 |
| 10002 | Session过期 |
| 10003 | 参数错误 |
| 20001 | 认证失败 |
| 20002 | Token无效 |
| 20003 | Token过期 |
| 30001 | 权限不足 |
| 40001 | 限流触发 |
| 50001 | 服务器内部错误 |

详细错误码说明请参考 [API文档完整版.md](API文档完整版.md)

---

## 💡 使用场景

### 典型验证流程
1. 前端调用获取验证码接口
2. 用户完成验证操作
3. 前端提交验证结果
4. 后端返回验证token
5. 将token用于后续业务接口

### 集成建议
- 使用SDK简化接入
- 前端实现良好的用户体验
- 后端验证token有效性
- 记录验证日志便于分析

---

## 📞 支持

如有问题，请参考：
- [故障排查手册.md](故障排查手册.md)
- [贡献指南.md](贡献指南.md)

---

## 📄 许可证

详见项目根目录 LICENSE 文件。
