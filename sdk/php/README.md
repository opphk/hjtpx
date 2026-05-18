# HJTPX Captcha System PHP SDK

PHP SDK for HJTPX Captcha System, providing comprehensive support for multiple captcha types with advanced features like connection pooling, automatic retry, and HMAC signature verification.

## Features

- **Multiple Captcha Types**:
  - Slider Captcha (滑块验证码)
  - Click Captcha (点击验证码)
  - Rotation Captcha (旋转验证码)
  - Gesture Captcha (手势验证码)
  - Jigsaw Captcha (拼图验证码)
  - Voice Captcha (语音验证码)
  - Connect Captcha (连连看验证码)
  - 3D Captcha (3D验证码)

- **Advanced Features**:
  - Connection Pool Management (连接池管理)
  - Automatic Retry with Exponential Backoff (自动重试与指数退避)
  - HMAC-SHA256 API Signature Verification (HMAC-SHA256签名验证)
  - Comprehensive Error Handling (完整错误处理)
  - Request/Response Logging (请求响应日志)
  - Debug Mode Support (调试模式支持)

- **Standards Compliance**:
  - PSR-4 Autoloading (PSR-4自动加载)
  - PSR-7 HTTP Messages (PSR-7 HTTP消息)
  - PHP 7.4+ Compatible (PHP 7.4+兼容)

## Requirements

- PHP 7.4 or higher
- Guzzle HTTP Client 7.0+
- Composer

## Installation

```bash
composer require hjtpx/captcha-sdk
```

## Quick Start

```php
<?php

require __DIR__ . '/vendor/autoload.php';

use HJTPX\Captcha\CaptchaClient;

$client = new CaptchaClient(
    'https://your-captcha-api.com',
    'your-api-key',
    'your-secret-key'
);

try {
    $sliderCaptcha = $client->getSliderCaptcha(320, 160, 8);

    $result = $client->verifySliderCaptcha(
        $sliderCaptcha->sessionId,
        150,
        $sliderCaptcha->secretY,
        $trajectory
    );

    if ($result->success) {
        echo "Verification successful!";
    }
} catch (Exception $e) {
    echo "Error: " . $e->getMessage();
}

$client->close();
```

## Captcha Types

### Slider Captcha

```php
$captcha = $client->getSliderCaptcha(320, 160, 8);

$result = $client->verifySliderCaptcha(
    $captcha->sessionId,
    150,
    $captcha->secretY,
    $trajectory
);
```

### Click Captcha

```php
$captcha = $client->getClickCaptcha('icon', true, 3);

$result = $client->verifyClickCaptcha(
    $captcha->sessionId,
    [[50, 50], [150, 50], [250, 50]]
);
```

### Rotation Captcha

```php
$captcha = $client->getRotationCaptcha();

$result = $client->verifyRotationCaptcha(
    $captcha->challengeId,
    90
);
```

### Gesture Captcha

```php
$captcha = $client->getGestureCaptcha();

$result = $client->verifyGestureCaptcha(
    $captcha->sessionId,
    [0, 1, 2, 5, 8]
);
```

### Jigsaw Captcha

```php
$captcha = $client->getJigsawCaptcha(300, 300, 3);

$result = $client->verifyJigsawCaptcha(
    $captcha->sessionId,
    $pieces
);
```

### Voice Captcha

```php
$captcha = $client->getVoiceCaptcha('zh-CN');

$result = $client->verifyVoiceCaptcha(
    $captcha->sessionId,
    '123456'
);
```

### Connect Captcha (Lianliankan)

```php
$captcha = $client->getLianliankanCaptcha(4, 4, 60);

$result = $client->verifyLianliankanCaptcha(
    $captcha->sessionId,
    $connections,
    30
);
```

### 3D Captcha

```php
$captcha = $client->getThreeDCaptcha();

$result = $client->verifyThreeDCaptcha(
    $captcha->sessionId,
    [100, 50, 0]
);
```

## Advanced Configuration

### Connection Pool Configuration

```php
use HJTPX\Captcha\Pool\ConnectionPoolConfig;

$poolConfig = new ConnectionPoolConfig(
    10,          // maxConnections
    10,          // connectionTimeout
    30,          // requestTimeout
    null,        // proxy
    true,        // sslVerify
    5,           // maxConcurrentRequests
    true         // keepAlive
);

$client = new CaptchaClient(
    'https://your-captcha-api.com',
    'your-api-key',
    'your-secret-key',
    $poolConfig
);
```

### Retry Configuration

```php
use HJTPX\Captcha\Retry\RetryConfig;

$retryConfig = new RetryConfig(
    3,                          // maxRetries
    100,                        // retryDelay (ms)
    2.0,                        // retryMultiplier
    [429, 500, 502, 503, 504],  // retryableStatusCodes
    30000,                      // maxRetryDelay (ms)
    true                        // enableExponentialBackoff
);

$client = new CaptchaClient(
    'https://your-captcha-api.com',
    'your-api-key',
    'your-secret-key',
    null,
    $retryConfig
);
```

### Debug Mode and Logging

```php
$client->setDebugMode(true);

$client->setLogger(function ($level, $message) {
    echo "[$level] $message\n";
});
```

### Connection Health Check

```php
if ($client->isConnected()) {
    echo "Connection is healthy";
}
```

### Get Connection Pool Statistics

```php
$stats = $client->poolManager->getStats();
print_r($stats);

$successRate = $client->poolManager->getSuccessRate();
echo "Success rate: $successRate%";
```

## Exception Handling

The SDK provides comprehensive exception types for different error scenarios:

```php
use HJTPX\Captcha\CaptchaClient;
use HJTPX\Captcha\Exception\CaptchaException;
use HJTPX\Captcha\Exception\ApiException;
use HJTPX\Captcha\Exception\NetworkException;
use HJTPX\Captcha\Exception\ValidationException;
use HJTPX\Captcha\Exception\AuthenticationException;

try {
    $result = $client->verifySliderCaptcha($sessionId, $x, $y);
} catch (ApiException $e) {
    echo "API Error: " . $e->getMessage();
    echo "Status Code: " . $e->getHttpStatusCode();
    echo "Error Code: " . $e->getErrorCode();
} catch (NetworkException $e) {
    echo "Network Error: " . $e->getMessage();
    echo "Host: " . $e->getHost();
} catch (ValidationException $e) {
    echo "Validation Error: " . $e->getMessage();
    echo "Field: " . $e->getField();
} catch (AuthenticationException $e) {
    echo "Auth Error: " . $e->getMessage();
    if ($e->isExpiredToken()) {
        // Handle token refresh
    }
} catch (CaptchaException $e) {
    echo "Captcha Error: " . $e->getMessage();
    echo "Context: " . json_encode($e->getContext());
}
```

## Authentication

```php
$loginResult = $client->login('username', 'password', $captchaToken);

if ($loginResult->accessToken) {
    $client->setAccessToken($loginResult->accessToken);
}

$client->logout();
```

## Environment Detection

```php
$script = $client->getDetectionScript('myCallback');

$checkResult = $client->checkEnvironment($environmentData);

$submitResult = $client->submitDetection($detectionData);
```

## CAPTCHA Type Constants

```php
use HJTPX\Captcha\CaptchaClient;

CaptchaClient::CAPTCHA_TYPE_SLIDER;
CaptchaClient::CAPTCHA_TYPE_CLICK;
CaptchaClient::CAPTCHA_TYPE_ROTATION;
CaptchaClient::CAPTCHA_TYPE_GESTURE;
CaptchaClient::CAPTCHA_TYPE_JIGSAW;
CaptchaClient::CAPTCHA_TYPE_VOICE;
CaptchaClient::CAPTCHA_TYPE_CONNECT;
CaptchaClient::CAPTCHA_TYPE_3D;
CaptchaClient::CAPTCHA_TYPE_LIANLIANKAN;
```

## Examples

More examples are available in the `examples/` directory:

- `slider_captcha.php` - Slider Captcha usage
- `click_captcha.php` - Click Captcha usage
- `login_with_captcha.php` - Login flow with Captcha

## Testing

```bash
composer test
```

## Code Style

```bash
composer cs-check
composer cs-fix
```

## License

MIT License
