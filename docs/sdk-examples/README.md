# HJTPX SDK 示例

本目录包含 HJTPX 行为验证系统的多语言 SDK 示例代码。

## 目录

- [Python SDK 示例](#python-sdk-示例)
- [JavaScript/Node.js SDK 示例](#javascriptnodejs-sdk-示例)
- [Java SDK 示例](#java-sdk-示例)
- [Go SDK 示例](#go-sdk-示例)

## Python SDK 示例

### 安装

```bash
pip install hjtpx-sdk
```

### 基础使用

```python
from hjtpx import HJTPXClient

# 初始化客户端
client = HJTPXClient(
    api_key="your-api-key",
    api_secret="your-api-secret",
    base_url="https://api.hjtpx.com"
)

# 创建滑块验证码
captcha = client.create_slider_captcha(
    width=360,
    height=180,
    difficulty="medium"
)
print(f"Session ID: {captcha.session_id}")

# 验证验证码
result = client.verify_captcha(
    session_id=captcha.session_id,
    captcha_type="slider",
    answer={"x": 150, "y": 50},
    trajectory=[
        {"x": 0, "y": 50, "t": 100},
        {"x": 50, "y": 52, "t": 200},
        {"x": 100, "y": 48, "t": 300},
        {"x": 150, "y": 50, "t": 400}
    ]
)

print(f"Success: {result.success}")
print(f"Risk Score: {result.risk_score}")
print(f"Human Probability: {result.human_probability}")
```

### 安全扫描

```python
from hjtpx import SecurityScanner

scanner = SecurityScanner()

# 检测 SQL 注入
is_safe = not scanner.scan_sql_injection(user_input)

# 检测 XSS
has_xss = scanner.scan_xss(user_comment)

# 扫描所有漏洞
vulns = scanner.scan_all(user_input)
for vuln in vulns:
    print(f"Vulnerability: {vuln.type}, Severity: {vuln.severity}")

# 密码强度检测
strength = scanner.check_password_strength(password)
print(f"Password Strength: {strength.strength}")
```

## JavaScript/Node.js SDK 示例

### 安装

```bash
npm install hjtpx-sdk
# 或
yarn add hjtpx-sdk
```

### 浏览器端使用

```javascript
import { HJTPXCaptcha } from 'hjtpx-sdk/browser';

// 初始化
const captcha = new HJTPXCaptcha({
  container: '#captcha-container',
  apiKey: 'your-api-key',
  captchaType: 'slider',
  theme: 'light'
});

// 监听验证成功
captcha.on('success', (result) => {
  console.log('验证成功', result);
  // 将 token 发送到后端
  fetch('/api/login', {
    method: 'POST',
    body: JSON.stringify({
      username: userInput,
      captchaToken: result.token
    })
  });
});

// 监听验证失败
captcha.on('error', (error) => {
  console.error('验证失败', error);
});

// 渲染验证码
captcha.render();
```

### Node.js 后端使用

```javascript
const { HJTPXClient, SecurityScanner } = require('hjtpx-sdk');

// 初始化客户端
const client = new HJTPXClient({
  apiKey: 'your-api-key',
  apiSecret: 'your-api-secret',
  baseUrl: 'https://api.hjtpx.com'
});

// 验证 token
app.post('/api/login', async (req, res) => {
  const { username, captchaToken } = req.body;
  
  try {
    const result = await client.verifyToken(captchaToken);
    
    if (result.success && result.riskScore < 50) {
      // 登录成功
      res.json({ success: true, userId: result.userId });
    } else {
      res.status(400).json({ success: false, error: '验证失败' });
    }
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// 安全扫描
const scanner = new SecurityScanner();

app.post('/api/comment', (req, res) => {
  const { comment } = req.body;
  
  if (scanner.scanXSS(comment)) {
    res.status(400).json({ error: '评论包含恶意内容' });
    return;
  }
  
  const sanitized = scanner.sanitizeInput(comment);
  // 保存评论...
});
```

## Java SDK 示例

### Maven 依赖

```xml
<dependency>
    <groupId>com.hjtpx</groupId>
    <artifactId>hjtpx-sdk</artifactId>
    <version>19.0.0</version>
</dependency>
```

### Spring Boot 集成

```java
import com.hjtpx.sdk.HJTPXClient;
import com.hjtpx.sdk.SecurityScanner;
import com.hjtpx.sdk.model.VerifyResult;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

@Service
public class CaptchaService {
    
    private final HJTPXClient client;
    private final SecurityScanner scanner;
    
    public CaptchaService(
            @Value("${hjtpx.api-key}") String apiKey,
            @Value("${hjtpx.api-secret}") String apiSecret) {
        this.client = new HJTPXClient(apiKey, apiSecret);
        this.scanner = new SecurityScanner();
    }
    
    public boolean verifyToken(String token) {
        VerifyResult result = client.verifyToken(token);
        return result.isSuccess() && result.getRiskScore() < 50;
    }
    
    public String sanitizeInput(String input) {
        return scanner.sanitizeInput(input);
    }
}
```

### 控制器示例

```java
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api")
public class LoginController {
    
    private final CaptchaService captchaService;
    
    public LoginController(CaptchaService captchaService) {
        this.captchaService = captchaService;
    }
    
    @PostMapping("/login")
    public ApiResponse login(@RequestBody LoginRequest request) {
        if (!captchaService.verifyToken(request.getCaptchaToken())) {
            return ApiResponse.error("验证码验证失败");
        }
        
        // 执行登录逻辑...
        return ApiResponse.success("登录成功");
    }
    
    @PostMapping("/comment")
    public ApiResponse postComment(@RequestBody CommentRequest request) {
        String safeContent = captchaService.sanitizeInput(request.getContent());
        // 保存评论...
        return ApiResponse.success("评论已提交");
    }
}
```

## Go SDK 示例

### 安装

```bash
go get github.com/hjtpx/hjtpx/sdk
```

### 使用示例

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/hjtpx/hjtpx/sdk"
    "github.com/hjtpx/hjtpx/internal/testing/security"
)

func main() {
    // 创建客户端
    client := sdk.NewClient(
        sdk.WithAPIKey("your-api-key"),
        sdk.WithAPISecret("your-api-secret"),
        sdk.WithEndpoint("https://api.hjtpx.com"),
    )
    defer client.Close()

    // 创建验证码
    captcha, err := client.CreateSliderCaptcha(context.Background(), &sdk.CreateSliderRequest{
        Width:      360,
        Height:     180,
        Difficulty: "medium",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Session ID: %s\n", captcha.SessionID)

    // 验证验证码
    result, err := client.VerifyCaptcha(context.Background(), &sdk.VerifyRequest{
        SessionID:   captcha.SessionID,
        CaptchaType: "slider",
        Answer: map[string]interface{}{
            "x": 150,
            "y": 50,
        },
        Trajectory: []sdk.TrajectoryPoint{
            {X: 0, Y: 50, Timestamp: 100},
            {X: 50, Y: 52, Timestamp: 200},
            {X: 100, Y: 48, Timestamp: 300},
            {X: 150, Y: 50, Timestamp: 400},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Success: %v\n", result.Success)
    fmt.Printf("Risk Score: %.2f\n", result.RiskScore)
    fmt.Printf("Human Probability: %.2f\n", result.HumanProbability)

    // 安全扫描
    scanner := security.NewSecurityScanner()
    input := "' OR '1'='1"
    if scanner.ScanSQLInjection(input) {
        fmt.Println("SQL injection detected!")
    }

    // 检查密码强度
    strength := security.CheckPasswordStrength("MyStr0ng!Passw0rd")
    fmt.Printf("Password Strength: %s (Score: %d)\n", strength.Strength, strength.Score)
}
```

## 更多资源

- [API 文档](../API文档完整版.md)
- [开发者指南](../开发者指南.md)
- [测试指南](../v19.0-测试指南.md)
