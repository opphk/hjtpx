# HJTPX Captcha Java SDK

HJTPX 验证码系统的 Java SDK，提供多种验证方式，包括滑块、点选、旋转、手势、拼图、语音、连连看和 3D 验证码。

## 功能特性

- **多种验证方式**：支持滑块、点选、旋转、手势、拼图、语音、连连看、3D 等多种验证码类型
- **API 签名验证**：支持 HMAC-SHA256 签名验证
- **连接池管理**：内置连接池，提高性能
- **自动重试机制**：支持失败自动重试
- **类型安全**：完全类型化的 API
- **易于集成**：简单易用的 API 设计

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
try {
    SliderCaptchaResponse captcha = client.getSliderCaptcha();
} catch (NetworkException e) {
    // 网络错误
    System.err.println("Network error: " + e.getMessage());
} catch (ApiException e) {
    // API 错误
    System.err.println("API error: " + e.getMessage() + ", code: " + e.getCode());
} catch (CaptchaException e) {
    // 其他验证错误
    System.err.println("Error: " + e.getMessage());
}
```

## 构建

```bash
mvn clean install
```

## 测试

```bash
mvn test
```

## 许可证

MIT License

## 注意事项

本 SDK 还在开发中，可能存在一些未发现的问题。在生产环境使用前，请充分测试。
