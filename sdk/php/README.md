# HJTPX 验证码系统 PHP SDK

HJTPX 验证码系统的 PHP SDK，提供完整的验证码类型支持、连接池管理和自动重试机制。

## 功能特性

- **多种验证码类型支持**：
  - 滑块验证码
  - 点击验证码
  - 旋转验证码
  - 手势验证码
  - 拼图验证码
  - 语音验证码
  - 连连看验证码
  - 3D 验证码

- **HMAC-SHA256 API 签名验证**：支持请求签名，确保API调用安全

- **连接池管理**：使用Guzzle HTTP客户端，支持连接复用

- **自动重试机制**：可配置的重试次数和延迟策略

- **PSR 规范兼容**：遵循 PSR-7 HTTP 消息接口规范

- **完善的错误处理**：提供多种异常类型，涵盖不同错误场景

## 安装

### 使用 Composer 安装（推荐）

```bash
composer require hjtpx/captcha-sdk
```

### 手动安装

如果无法使用 Composer，可以手动下载并引入：

```php
require_once 'path/to/src/CaptchaClient.php';
require_once 'path/to/src/Exception/ApiException.php';
require_once 'path/to/src/Exception/CaptchaException.php';
// ... 其他依赖文件
```

## 快速开始

### 基本使用

```php
<?php

require __DIR__ . '/vendor/autoload.php';

use HJTPX\Captcha\CaptchaClient;

// 初始化客户端
$client = new CaptchaClient(
    'http://localhost:8080',      // API基础URL
    'your-api-key',               // API密钥（可选）
    'your-secret-key'            // 密钥（可选，用于签名验证）
);

// 获取滑块验证码
$sliderCaptcha = $client->getSliderCaptcha(320, 160, 8);

// 验证滑块验证码
$result = $client->verifySliderCaptcha(
    $sliderCaptcha->sessionId,    // 会话ID
    150,                          // X坐标
    $sliderCaptcha->secretY,      // 秘密Y坐标
    $trajectory                   // 轨迹点数组（可选）
);

if ($result->success) {
    echo "验证成功！";
}

// 关闭客户端
$client->close();
```

### 使用连接池和重试配置

```php
<?php

use HJTPX\Captcha\CaptchaClient;
use HJTPX\Captcha\Pool\ConnectionPoolConfig;
use HJTPX\Captcha\Retry\RetryConfig;

// 自定义连接池配置
$poolConfig = new ConnectionPoolConfig(
    10,      // 最大连接数
    30,      // 连接超时（秒）
    30       // 请求超时（秒）
);

// 自定义重试配置
$retryConfig = new RetryConfig(
    3,                 // 最大重试次数
    100,               // 初始重试延迟（毫秒）
    2.0,               // 重试延迟倍数
    [429, 500, 502, 503, 504] // 可重试的状态码
);

$client = new CaptchaClient(
    'http://localhost:8080',
    'your-api-key',
    'your-secret-key',
    $poolConfig,
    $retryConfig
);
```

## 验证码类型

### 1. 滑块验证码

```php
<?php

// 获取滑块验证码
$slider = $client->getSliderCaptcha(320, 160, 8);
echo "Session ID: " . $slider->sessionId . "\n";
echo "Image URL: " . $slider->imageUrl . "\n";

// 准备轨迹数据
$trajectory = [
    ['x' => 0, 'y' => 80, 't' => 0],
    ['x' => 50, 'y' => 85, 't' => 200],
    ['x' => 100, 'y' => 77, 't' => 400],
    ['x' => 150, 'y' => 80, 't' => 600],
];

// 验证
$result = $client->verifySliderCaptcha(
    $slider->sessionId,
    150,
    $slider->secretY,
    $trajectory
);

if ($result->success) {
    echo "滑块验证成功！\n";
} else {
    echo "验证失败: " . $result->message . "\n";
}
```

### 2. 点击验证码

```php
<?php

// 获取点击验证码
$click = $client->getClickCaptcha('number', true, 3);
echo "Session ID: " . $click->sessionId . "\n";
echo "Hint: " . $click->hint . "\n";
echo "Required sequence: " . implode(',', $click->hintOrder) . "\n";

// 用户点击的坐标
$points = [
    [100, 100],
    [200, 100],
    [150, 200],
];

// 验证
$result = $client->verifyClickCaptcha(
    $click->sessionId,
    $points,
    $click->hintOrder
);

if ($result->success) {
    echo "点击验证成功！\n";
}
```

### 3. 旋转验证码

```php
<?php

// 获取旋转验证码
$rotation = $client->getRotationCaptcha();

// 验证（用户旋转到的角度）
$result = $client->verifyRotationCaptcha(
    $rotation->challengeId,
    90  // 旋转角度
);
```

### 4. 手势验证码

```php
<?php

// 获取手势验证码
$gesture = $client->getGestureCaptcha();
echo "Session ID: " . $gesture->sessionId . "\n";

// 用户绘制的手势模式
$pattern = [0, 1, 2, 5, 8];  // 手势点索引

// 验证
$result = $client->verifyGestureCaptcha(
    $gesture->sessionId,
    $pattern
);
```

### 5. 拼图验证码

```php
<?php

// 获取拼图验证码
$jigsaw = $client->getJigsawCaptcha(300, 300, 3);

// 准备碎片数据
$pieces = [];
foreach ($jigsaw->pieces as $piece) {
    $pieces[] = [
        'index' => $piece->index,
        'original_x' => $piece->originalX,
        'original_y' => $piece->originalY,
        'current_x' => $piece->currentX,
        'current_y' => $piece->currentY,
        'width' => $piece->width,
        'height' => $piece->height,
    ];
}

// 验证
$result = $client->verifyJigsawCaptcha($jigsaw->sessionId, $pieces);
```

### 6. 语音验证码

```php
<?php

// 获取语音验证码
$voice = $client->getVoiceCaptcha('zh-CN');

// 用户听到的验证码答案
$result = $client->verifyVoiceCaptcha(
    $voice->sessionId,
    '123456'
);
```

### 7. 连连看验证码

```php
<?php

// 获取连连看验证码
$connect = $client->getConnectCaptcha();

// 用户完成的连接
$connections = [
    [0, 1],  // 连接点0和点1
    [2, 3],  // 连接点2和点3
];

$result = $client->verifyConnectCaptcha($connect->sessionId, $connections);
```

### 8. 3D 验证码

```php
<?php

// 获取3D验证码
$threeD = $client->getThreeDCaptcha();

// 用户点击的目标位置
$targetPosition = [100, 50, 0];  // x, y, z

$result = $client->verifyThreeDCaptcha($threeD->sessionId, $targetPosition);
```

## 错误处理

SDK 提供了多种异常类型，用于处理不同的错误场景：

```php
<?php

use HJTPX\Captcha\CaptchaClient;
use HJTPX\Captcha\Exception\CaptchaException;
use HJTPX\Captcha\Exception\ApiException;
use HJTPX\Captcha\Exception\NetworkException;
use HJTPX\Captcha\Exception\ValidationException;
use HJTPX\Captcha\Exception\AuthenticationException;

try {
    $client = new CaptchaClient('http://localhost:8080');
    $result = $client->getSliderCaptcha();

} catch (ValidationException $e) {
    // 参数验证失败
    echo "参数错误: " . $e->getMessage() . "\n";

} catch (AuthenticationException $e) {
    // 认证失败
    echo "认证失败: " . $e->getMessage() . "\n";

} catch (NetworkException $e) {
    // 网络错误
    echo "网络错误: " . $e->getMessage() . "\n";

} catch (ApiException $e) {
    // API错误
    echo "API错误: " . $e->getMessage() . "\n";
    echo "错误码: " . $e->getCode() . "\n";

} catch (CaptchaException $e) {
    // 其他验证码相关错误
    echo "验证码错误: " . $e->getMessage() . "\n";

} catch (Exception $e) {
    // 其他未知错误
    echo "未知错误: " . $e->getMessage() . "\n";
}
```

### 异常类型说明

- `CaptchaException`: 基础异常类，所有SDK异常的父类
- `ApiException`: API返回错误时的异常
- `NetworkException`: 网络连接相关异常
- `ValidationException`: 参数验证失败异常
- `AuthenticationException`: 认证失败异常

## 用户认证

```php
<?php

// 登录
$loginResult = $client->login('username', 'password');

echo "Access Token: " . $loginResult->accessToken . "\n";
echo "User ID: " . $loginResult->user->id . "\n";

// 登出
$client->logout();
```

## 环境检测

```php
<?php

// 获取检测脚本
$script = $client->getDetectionScript('onDetectReady');
echo "Script length: " . strlen($script) . " bytes\n";

// 提交检测数据
$detectionData = [
    'fingerprint' => 'browser-fingerprint-hash',
    'canvas_hash' => 'canvas-fingerprint',
    'webgl_vendor' => 'WebGL Vendor',
    'timezone' => 'Asia/Shanghai',
];

$result = $client->submitDetection($detectionData);

// 检查环境安全
$checkResult = $client->checkEnvironment([
    'fingerprint' => 'browser-fingerprint-hash',
    'risk_score' => 0.1,
]);
```

## API 参考

### CaptchaClient

#### 构造函数

```php
public function __construct(
    string $baseUrl,                    // API基础URL
    string $apiKey = null,              // API密钥
    string $secretKey = null,           // 密钥（用于签名）
    ConnectionPoolConfig $poolConfig = null,  // 连接池配置
    RetryConfig $retryConfig = null     // 重试配置
)
```

#### 验证码方法

| 方法 | 描述 |
|------|------|
| `getSliderCaptcha()` | 获取滑块验证码 |
| `verifySliderCaptcha()` | 验证滑块验证码 |
| `getClickCaptcha()` | 获取点击验证码 |
| `verifyClickCaptcha()` | 验证点击验证码 |
| `getRotationCaptcha()` | 获取旋转验证码 |
| `verifyRotationCaptcha()` | 验证旋转验证码 |
| `getGestureCaptcha()` | 获取手势验证码 |
| `verifyGestureCaptcha()` | 验证手势验证码 |
| `getJigsawCaptcha()` | 获取拼图验证码 |
| `verifyJigsawCaptcha()` | 验证拼图验证码 |
| `getVoiceCaptcha()` | 获取语音验证码 |
| `verifyVoiceCaptcha()` | 验证语音验证码 |
| `getConnectCaptcha()` | 获取连连看验证码 |
| `verifyConnectCaptcha()` | 验证连连看验证码 |
| `getThreeDCaptcha()` | 获取3D验证码 |
| `verifyThreeDCaptcha()` | 验证3D验证码 |
| `verifyCaptcha()` | 通用验证方法 |

#### 认证方法

| 方法 | 描述 |
|------|------|
| `login()` | 用户登录 |
| `logout()` | 用户登出 |

#### 检测方法

| 方法 | 描述 |
|------|------|
| `getDetectionScript()` | 获取检测脚本 |
| `submitDetection()` | 提交检测数据 |
| `checkEnvironment()` | 检查环境安全 |

## 示例代码

更多示例请参考 `examples` 目录：

- `slider_captcha.php` - 滑块验证码使用示例
- `click_captcha.php` - 点击验证码使用示例
- `login_with_captcha.php` - 结合验证码的登录流程示例

## 运行测试

```bash
# 安装依赖
composer install

# 运行单元测试
composer test

# 或直接使用 PHPUnit
./vendor/bin/phpunit
```

## 注意事项

1. 本SDK为基本可用版本，可能存在未发现的问题
2. 请根据实际API接口调整使用方式
3. 生产环境使用前请充分测试
4. 使用签名验证时，请确保密钥安全存储
5. 合理配置重试次数，避免对服务器造成压力

## 依赖要求

- PHP >= 7.4
- GuzzleHTTP >= 6.0
- PSR-7 (HTTP接口)

## 许可证

MIT License
