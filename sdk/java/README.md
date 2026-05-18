# HJTPX Captcha Java SDK

HJTPX 验证码系统的 Java SDK，提供多种验证方式，包括滑块、点选、旋转、手势、拼图、语音、连连看和 3D 验证码。

## 功能特性

- **多种验证方式**：支持滑块、点选、旋转、手势、拼图、语音、连连看、3D 等多种验证码类型
- **API 签名验证**：支持 HMAC-SHA256 签名验证
- **连接池管理**：内置连接池，提高性能
- **自动重试机制**：支持失败自动重试
- **类型安全**：完全类型化的 API
- **易于集成**：简单易用的 API 设计
- **完善的错误处理**：丰富的异常类型

## 安装

### Maven

```xml
<dependency>
    <groupId>com.hjtpx</groupId>
    <artifactId>captcha-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

```groovy
implementation 'com.hjtpx:captcha-sdk:1.0.0'
```

## 快速开始

### 基本使用

```java
import com.hjtpx.captcha.client.CaptchaClient;
import com.hjtpx.captcha.model.*;

public class Example {
    public static void main(String[] args) {
        String baseUrl = "http://your-captcha-server.com";
        String apiKey = "your-api-key";
        
        try (CaptchaClient client = new CaptchaClient(baseUrl, apiKey)) {
            SliderCaptchaResponse captcha = client.getSliderCaptcha(320, 160, 5);
            System.out.println("Session ID: " + captcha.getSessionId());
            System.out.println("Image URL: " + captcha.getImageUrl());
            
            VerifyCaptchaResponse verifyResponse = client.verifySliderCaptcha(
                captcha.getSessionId(),
                180
            );
            
            System.out.println("Success: " + verifyResponse.isSuccess());
            System.out.println("Message: " + verifyResponse.getMessage());
        }
    }
}
```

### 配置客户端

```java
import com.hjtpx.captcha.client.CaptchaClientConfig;
import com.hjtpx.captcha.pool.ConnectionPoolConfig;
import com.hjtpx.captcha.retry.RetryConfig;

CaptchaClientConfig config = new CaptchaClientConfig();
config.setBaseUrl("http://your-captcha-server.com");
config.setApiKey("your-api-key");
config.setSecretKey("your-secret-key");

ConnectionPoolConfig poolConfig = new ConnectionPoolConfig();
poolConfig.setMaxConnections(100);
poolConfig.setConnectionTimeout(5000);
config.setConnectionPoolConfig(poolConfig);

RetryConfig retryConfig = new RetryConfig();
retryConfig.setMaxRetries(3);
retryConfig.setInitialDelayMs(100);
config.setRetryConfig(retryConfig);

CaptchaClient client = new CaptchaClient(config);
```

## 高级配置

### 连接池配置

```java
ConnectionPoolConfig poolConfig = new ConnectionPoolConfig();
poolConfig.setMaxConnections(200);           // 最大连接数
poolConfig.setMaxConnectionsPerRoute(50);     // 每路由最大连接数
poolConfig.setConnectionTimeout(10000);       // 连接超时（毫秒）
poolConfig.setSocketTimeout(30000);          // Socket超时（毫秒）
poolConfig.setTimeToLive(60000);            // 连接生存时间（毫秒）
poolConfig.setValidateAfterInactivity(3000); // 空闲连接验证间隔

CaptchaClientConfig config = new CaptchaClientConfig();
config.setConnectionPoolConfig(poolConfig);
```

### 重试配置

```java
RetryConfig retryConfig = new RetryConfig();
retryConfig.setMaxRetries(5);                 // 最大重试次数
retryConfig.setInitialDelayMs(200);          // 初始延迟
retryConfig.setMaxDelayMs(10000);            // 最大延迟
retryConfig.setBackoffMultiplier(2.0);       // 退避乘数
retryConfig.setRetryableStatusCodes(Arrays.asList(429, 500, 502, 503, 504));

CaptchaClientConfig config = new CaptchaClientConfig();
config.setRetryConfig(retryConfig);
```

## 完整示例

### 滑块验证码完整示例

```java
import com.hjtpx.captcha.client.CaptchaClient;
import com.hjtpx.captcha.model.*;
import java.util.Arrays;
import java.util.List;

public class SliderCaptchaExample {
    public static void main(String[] args) {
        try (CaptchaClient client = new CaptchaClient(
            "http://localhost:8080",
            "your-api-key"
        )) {
            // 1. 获取滑块验证码
            SliderCaptchaResponse captcha = client.getSliderCaptcha(320, 160, 5);
            System.out.println("Session ID: " + captcha.getSessionId());
            System.out.println("Secret Y: " + captcha.getSecretY());
            
            // 2. 模拟用户滑动轨迹
            List<TrajectoryPoint> trajectory = generateTrajectory(captcha.getSecretY());
            
            // 3. 验证
            VerifyCaptchaResponse response = client.verifySliderCaptcha(
                captcha.getSessionId(),
                180,                              // X坐标
                captcha.getSecretY(),             // Y坐标
                trajectory                        // 轨迹数据
            );
            
            System.out.println("Success: " + response.isSuccess());
            System.out.println("Score: " + response.getScore());
        }
    }
    
    private static List<TrajectoryPoint> generateTrajectory(int secretY) {
        long baseTime = System.currentTimeMillis();
        return Arrays.asList(
            new TrajectoryPoint(0, secretY, baseTime - 1000),
            new TrajectoryPoint(30, secretY + 2, baseTime - 800),
            new TrajectoryPoint(60, secretY - 1, baseTime - 600),
            new TrajectoryPoint(100, secretY + 3, baseTime - 400),
            new TrajectoryPoint(140, secretY - 2, baseTime - 200),
            new TrajectoryPoint(180, secretY, baseTime)
        );
    }
}
```

### 点选验证码完整示例

```java
import com.hjtpx.captcha.client.CaptchaClient;
import com.hjtpx.captcha.model.*;
import java.util.Arrays;
import java.util.List;

public class ClickCaptchaExample {
    public static void main(String[] args) {
        try (CaptchaClient client = new CaptchaClient(
            "http://localhost:8080",
            "your-api-key"
        )) {
            // 获取点选验证码
            ClickCaptchaResponse captcha = client.getClickCaptcha("number", true, 3);
            System.out.println("Session ID: " + captcha.getSessionId());
            System.out.println("Hint: " + captcha.getHint());
            
            // 用户按顺序点击
            List<List<Integer>> points = Arrays.asList(
                Arrays.asList(100, 100),  // 第一个点
                Arrays.asList(200, 150),  // 第二个点
                Arrays.asList(150, 200)   // 第三个点
            );
            List<Integer> clickSequence = Arrays.asList(0, 1, 2);
            
            VerifyCaptchaResponse response = client.verifyClickCaptcha(
                captcha.getSessionId(),
                points,
                clickSequence
            );
            
            System.out.println("Success: " + response.isSuccess());
        }
    }
}
```

### 用户登录完整示例

```java
import com.hjtpx.captcha.client.CaptchaClient;
import com.hjtpx.captcha.model.*;

public class LoginExample {
    public static void main(String[] args) {
        try (CaptchaClient client = new CaptchaClient(
            "http://localhost:8080",
            "your-api-key"
        )) {
            // 登录
            LoginResponse loginResponse = client.login("username", "password");
            System.out.println("Login successful!");
            System.out.println("Access Token: " + loginResponse.getAccessToken());
            System.out.println("User: " + loginResponse.getUser().getUsername());
            
            // 使用 AccessToken 进行后续操作
            // ...
            
            // 登出
            client.logout();
            System.out.println("Logged out");
        }
    }
}
```

## 支持的验证码类型

### 1. 滑块验证码

```java
SliderCaptchaResponse captcha = client.getSliderCaptcha(320, 160, 5);
VerifyCaptchaResponse response = client.verifySliderCaptcha(sessionId, x, y, trajectory);
```

### 2. 点选验证码

```java
ClickCaptchaResponse captcha = client.getClickCaptcha("number", true, 3);
VerifyCaptchaResponse response = client.verifyClickCaptcha(sessionId, points, clickSequence);
```

### 3. 旋转验证码

```java
RotationCaptchaResponse captcha = client.getRotationCaptcha();
VerifyCaptchaResponse response = client.verifyRotationCaptcha(challengeId, angle);
```

### 4. 手势验证码

```java
GestureCaptchaResponse captcha = client.getGestureCaptcha();
VerifyCaptchaResponse response = client.verifyGestureCaptcha(sessionId, pattern);
```

### 5. 拼图验证码

```java
JigsawCaptchaResponse captcha = client.getJigsawCaptcha(300, 300, 3);
VerifyCaptchaResponse response = client.verifyJigsawCaptcha(sessionId, pieces);
```

### 6. 语音验证码

```java
VoiceCaptchaResponse captcha = client.getVoiceCaptcha("zh-CN");
VerifyCaptchaResponse response = client.verifyVoiceCaptcha(sessionId, answer);
```

### 7. 连连看验证码

```java
ConnectCaptchaResponse captcha = client.getConnectCaptcha();
VerifyCaptchaResponse response = client.verifyConnectCaptcha(sessionId, connections);
```

### 8. 3D 验证码

```java
ThreeDCaptchaResponse captcha = client.getThreeDCaptcha();
VerifyCaptchaResponse response = client.verifyThreeDCaptcha(sessionId, targetPosition);
```

## 错误处理

```java
import com.hjtpx.captcha.exception.*;

try {
    SliderCaptchaResponse captcha = client.getSliderCaptcha();
} catch (NetworkException e) {
    // 网络错误
    System.err.println("Network error: " + e.getMessage());
} catch (ApiException e) {
    // API 错误
    System.err.println("API error: " + e.getMessage() + ", code: " + e.getCode());
} catch (ValidationException e) {
    // 验证失败
    System.err.println("Validation error: " + e.getMessage());
} catch (AuthenticationException e) {
    // 认证失败
    System.err.println("Auth error: " + e.getMessage());
} catch (CaptchaException e) {
    // 其他验证错误
    System.err.println("Error: " + e.getMessage());
}
```

## 环境检测

```java
// 获取检测脚本
String script = client.getDetectionScript();
System.out.println("Script length: " + script.length());

// 提交检测数据
Map<String, Object> data = new HashMap<>();
data.put("fingerprint", "browser-fingerprint");
data.put("canvas_hash", "canvas-fingerprint");
data.put("webgl_vendor", "WebGL Vendor");

Map<String, Object> result = client.submitDetection(data);
System.out.println("Risk level: " + result.get("risk_level"));
```

## 构建

```bash
mvn clean install
```

## 测试

```bash
mvn test
```

## 项目结构

```
java/
├── src/main/java/com/hjtpx/captcha/
│   ├── client/          # 客户端
│   ├── exception/      # 异常
│   ├── model/          # 数据模型
│   ├── pool/          # 连接池
│   ├── retry/         # 重试
│   └── signer/        # 签名
├── examples/          # 示例代码
└── pom.xml
```

## 兼容性

- Java 11+
- Apache HttpClient 4.5+
- Jackson 2.10+

## 许可证

MIT License

## 注意事项

1. 本 SDK 需要 Java 11 或更高版本
2. 依赖 Apache HttpClient, Jackson, SLF4J 等库
3. 实际使用前需要配置正确的服务器地址和 API Key
4. 建议在生产环境前进行充分测试
