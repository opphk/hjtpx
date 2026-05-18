# Hjtpx Captcha PHP SDK v15.0

## 特性

- Laravel 扩展包，开箱即用
- 支持 Facade 和依赖注入
- 完整的验证码类型支持
- 批量验证和异步验证
- 自动重试和错误处理
- 连接池管理

## 安装

### 使用 Composer

```bash
composer require hjtpx/captcha-laravel
```

### Laravel 配置

发布配置文件：

```bash
php artisan vendor:publish --tag=captcha-config
```

配置环境变量：

```env
CAPTCHA_BASE_URL=http://localhost:8080
CAPTCHA_API_KEY=your-api-key
CAPTCHA_TIMEOUT=30
CAPTCHA_MAX_RETRIES=3
```

## 快速开始

### 使用 Facade

```php
use Hjtpx\Captcha\Facades\Captcha;

// 获取滑块验证码
$captcha = Captcha::getSliderCaptcha(320, 160, 8);
echo $captcha['session_id'];

// 验证验证码
$result = Captcha::verifySliderCaptcha(
    $captcha['session_id'],
    185,
    $captcha['secret_y'] ?? null,
    [
        ['x' => 0, 'y' => 50, 't' => time() * 1000 - 1000],
        ['x' => 50, 'y' => 55, 't' => time() * 1000 - 800],
        ['x' => 100, 'y' => 47, 't' => time() * 1000 - 500],
        ['x' => 150, 'y' => 52, 't' => time() * 1000 - 200],
        ['x' => 185, 'y' => 50, 't' => time() * 1000],
    ]
);

if ($result['success']) {
    echo "Verification passed!";
}
```

### 使用依赖注入

```php
use Hjtpx\Captcha\Contracts\CaptchaClientInterface;

class CaptchaController extends Controller
{
    public function __construct(
        private CaptchaClientInterface $captchaClient
    ) {}

    public function show()
    {
        $captcha = $this->captchaClient->getSliderCaptcha();
        return response()->json($captcha);
    }

    public function verify(Request $request)
    {
        $result = $this->captchaClient->verifySliderCaptcha(
            $request->session_id,
            $request->x,
            $request->y
        );
        return response()->json($result);
    }
}
```

### 批量验证

```php
$requests = [
    ['session_id' => 'session-1', 'x' => 100, 'y' => 50],
    ['session_id' => 'session-2', 'x' => 150, 'y' => 60],
    ['session_id' => 'session-3', 'x' => 200, 'y' => 70],
];

$result = Captcha::batchVerify($requests);

echo "Success: {$result['success_count']}, Failed: {$result['failed_count']}";
echo "Total time: {$result['total_time_ms']}ms";

foreach ($result['results'] as $r) {
    echo "{$r['session_id']}: " . ($r['success'] ? 'OK' : 'Failed');
}
```

### 异步验证

```php
// 发起异步验证
$asyncResult = Captcha::asyncVerify([
    'session_id' => 'session-async-1',
    'x' => 150,
    'y' => 50,
    'callback_url' => 'https://example.com/callback',
]);

$taskId = $asyncResult['task_id'];

// 轮询获取结果
for ($i = 0; $i < 10; $i++) {
    $result = Captcha::getAsyncResult($taskId);

    if ($result['status'] === 'completed') {
        if ($result['result']['success']) {
            echo "Verification passed!";
        }
        break;
    }

    if ($result['status'] === 'failed') {
        echo "Verification failed: " . $result['error'];
        break;
    }

    usleep(500000); // 500ms
}
```

### 点击验证码

```php
$captcha = Captcha::getClickCaptcha('number', 3, true);
echo $captcha['hint']; // "Click 1, 2, 3 in order"

// 用户点击后验证
$points = [[100, 100], [200, 150], [150, 200]];
$clickSequence = [1, 2, 3];

$result = Captcha::verifyClickCaptcha(
    $captcha['session_id'],
    $points,
    $clickSequence
);
```

### 图形验证码

```php
$captcha = Captcha::getImageCaptcha('mixed', 4);

// 显示图片
echo '<img src="' . $captcha['image'] . '">';

// 验证答案
$result = Captcha::verifyImageCaptcha(
    $captcha['challenge_id'],
    'ABCD'
);
```

## 错误处理

```php
use Hjtpx\Captcha\Exceptions\CaptchaException;
use Hjtpx\Captcha\Exceptions\CaptchaApiException;
use Hjtpx\Captcha\Exceptions\CaptchaNetworkException;
use Hjtpx\Captcha\Exceptions\CaptchaTimeoutException;
use Hjtpx\Captcha\Exceptions\CaptchaValidationException;

try {
    $result = Captcha::verifySliderCaptcha($sessionId, $x, $y);
} catch (CaptchaApiException $e) {
    // API错误
    echo "API Error: " . $e->getMessage();
    echo "Error Code: " . $e->getErrorCode();
} catch (CaptchaNetworkException $e) {
    // 网络错误
    echo "Network Error: " . $e->getMessage();
} catch (CaptchaTimeoutException $e) {
    // 超时错误
    echo "Timeout: " . $e->getMessage();
} catch (CaptchaValidationException $e) {
    // 验证失败
    echo "Validation Error: " . $e->getMessage();
} catch (CaptchaException $e) {
    // 其他错误
    echo "Error: " . $e->getMessage();
}
```

## 服务容器绑定

包自动注册以下绑定：

```php
// 绑定到接口
$this->app->singleton(
    \Hjtpx\Captcha\Contracts\CaptchaClientInterface::class,
    \Hjtpx\Captcha\Client\CaptchaClient::class
);

// 可以通过类型提示注入
public function __construct(\Hjtpx\Captcha\Contracts\CaptchaClientInterface $client)
```

## 自定义客户端

```php
use Hjtpx\Captcha\Client\CaptchaClient;

$client = new CaptchaClient(
    'http://localhost:8080',
    'your-api-key',
    30,      // timeout
    3,       // max retries
    0.5      // retry backoff factor
);

// 使用客户端
$captcha = $client->getSliderCaptcha();
```

## 许可

MIT License
