# HJTPX 验证码系统 PHP SDK

HJTPX 验证码系统的 PHP SDK，提供完整的验证码类型支持、连接池管理和自动重试机制。

## 功能特性

- 支持多种验证码类型：
  - 滑块验证码
  - 点击验证码
  - 旋转验证码
  - 手势验证码
  - 拼图验证码
  - 语音验证码
  - 连连看验证码
  - 3D 验证码
- HMAC-SHA256 API 签名验证
- 连接池管理
- 自动重试机制
- PSR 规范兼容

## 安装

使用 Composer 安装：

```bash
composer require hjtpx/captcha-sdk
```

## 快速开始

### 基本使用

```php
<?php

require __DIR__ . '/vendor/autoload.php';

use HJTPX\Captcha\CaptchaClient;

// 初始化客户端
$client = new CaptchaClient(
    'https://your-captcha-api.com',
    'your-api-key',
    'your-secret-key' // 可选，用于签名验证
);
```

### 滑块验证码

```php
// 获取滑块验证码
$sliderCaptcha = $client->getSliderCaptcha(320, 160, 8);

// 验证滑块验证码
$result = $client->verifySliderCaptcha(
    $sliderCaptcha->sessionId,
    150, // x 坐标
    $sliderCaptcha->secretY,
    $trajectory // 轨迹点数组
);

if ($result->success) {
    echo "验证成功！";
}
```

### 点击验证码

```php
// 获取点击验证码
$clickCaptcha = $client->getClickCaptcha('icon', true, 3);

// 验证点击验证码
$result = $client->verifyClickCaptcha(
    $clickCaptcha->sessionId,
    [[50, 50], [150, 50], [250, 50]] // 点击坐标数组
);
```

### 其他验证码类型

```php
// 旋转验证码
$rotationCaptcha = $client->getRotationCaptcha();
$result = $client->verifyRotationCaptcha($rotationCaptcha->challengeId, 90);

// 手势验证码
$gestureCaptcha = $client->getGestureCaptcha();
$result = $client->verifyGestureCaptcha($gestureCaptcha->sessionId, [0, 1, 2, 5, 8]);

// 拼图验证码
$jigsawCaptcha = $client->getJigsawCaptcha(300, 300, 3);
$result = $client->verifyJigsawCaptcha($jigsawCaptcha->sessionId, $pieces);

// 语音验证码
$voiceCaptcha = $client->getVoiceCaptcha('zh-CN');
$result = $client->verifyVoiceCaptcha($voiceCaptcha->sessionId, '123456');

// 连连看验证码
$connectCaptcha = $client->getConnectCaptcha();
$result = $client->verifyConnectCaptcha($connectCaptcha->sessionId, $connections);

// 3D 验证码
$threeDCaptcha = $client->getThreeDCaptcha();
$result = $client->verifyThreeDCaptcha($threeDCaptcha->sessionId, [100, 50, 0]);
```

## 高级配置

### 自定义连接池配置

```php
use HJTPX\Captcha\Pool\ConnectionPoolConfig;

$poolConfig = new ConnectionPoolConfig(
    10,      // 最大连接数
    30,      // 连接超时（秒）
    30       // 请求超时（秒）
);

$client = new CaptchaClient(
    'https://your-captcha-api.com',
    'your-api-key',
    'your-secret-key',
    $poolConfig
);
```

### 自定义重试配置

```php
use HJTPX\Captcha\Retry\RetryConfig;

$retryConfig = new RetryConfig(
    3,                 // 最大重试次数
    100,               // 初始重试延迟（毫秒）
    2.0,               // 重试延迟倍数
    [429, 500, 502, 503, 504] // 可重试的状态码
);

$client = new CaptchaClient(
    'https://your-captcha-api.com',
    'your-api-key',
    'your-secret-key',
    null,
    $retryConfig
);
```

## 运行测试

```bash
composer test
```

## 示例

更多示例请参考 `examples` 目录：

- `slider_captcha.php` - 滑块验证码使用示例
- `click_captcha.php` - 点击验证码使用示例
- `login_with_captcha.php` - 结合验证码的登录流程示例

## 注意事项

- 本 SDK 为基本可用版本，可能存在未发现的问题
- 请根据实际 API 接口调整使用方式
- 生产环境使用前请充分测试

## 许可证

MIT License
