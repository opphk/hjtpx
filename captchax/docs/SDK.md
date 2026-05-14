# CaptchaX 多语言 SDK 使用指南

本文档提供 CaptchaX 各语言 SDK 的详细接入指南。

## 目录

- [JavaScript/TypeScript SDK](#javascripttypescript-sdk)
- [Go SDK](#go-sdk)
- [Python SDK](#python-sdk)
- [Java SDK](#java-sdk)
- [.NET SDK](#net-sdk)
- [PHP SDK](#php-sdk)
- [Ruby SDK](#ruby-sdk)

---

## JavaScript/TypeScript SDK

### 安装

```bash
npm install @captchax/js-sdk
# 或
yarn add @captchax/js-sdk
# 或
pnpm add @captchax/js-sdk
```

### 基础使用

```typescript
import { CaptchaXClient } from '@captchax/js-sdk';

const client = new CaptchaXClient({
  appId: 'your-app-id',
  serverUrl: 'https://captchax.example.com',
  timeout: 10000,
});

async function example() {
  // 生成滑块验证码
  const slider = await client.createSliderCaptcha({
    width: 200,
    height: 80,
  });

  console.log('验证码ID:', slider.id);
  console.log('目标位置:', slider.targetX, slider.targetY);

  // 验证滑块
  const result = await client.verifySlider({
    captchaId: slider.id,
    targetX: 150,
    targetY: 25,
  });

  if (result.success) {
    console.log('验证成功!');
  }
}
```

### React 集成示例

```tsx
import React, { useState } from 'react';
import { CaptchaX } from '@captchax/js-sdk';

interface CaptchaVerifyProps {
  onVerify: (token: string) => void;
}

export const CaptchaVerify: React.FC<CaptchaVerifyProps> = ({ onVerify }) => {
  const [captcha, setCaptcha] = useState<CaptchaX | null>(null);
  const [verified, setVerified] = useState(false);

  const handleReady = () => {
    console.log('验证码已加载');
  };

  const handleSuccess = (result: { token: string }) => {
    console.log('验证成功:', result);
    setVerified(true);
    onVerify(result.token);
  };

  const handleError = (error: Error) => {
    console.error('验证失败:', error);
  };

  return (
    <div>
      {!verified ? (
        <CaptchaX
          appId="your-app-id"
          serverUrl="https://captchax.example.com"
          container="#captcha-container"
          onReady={handleReady}
          onSuccess={handleSuccess}
          onError={handleError}
        />
      ) : (
        <div className="captcha-success">验证成功</div>
      )}
    </div>
  );
};
```

### Vue 3 集成示例

```typescript
import { createApp, ref, onMounted } from 'vue';
import { CaptchaX } from '@captchax/js-sdk';

export const useCaptcha = (options: {
  appId: string;
  serverUrl: string;
  onSuccess: (token: string) => void;
}) => {
  const containerRef = ref<HTMLElement | null>(null);
  const verified = ref(false);
  let captchaInstance: CaptchaX | null = null;

  onMounted(() => {
    if (containerRef.value) {
      captchaInstance = new CaptchaX({
        ...options,
        container: containerRef.value,
        onSuccess: (result) => {
          verified.value = true;
          options.onSuccess(result.token);
        },
      });

      captchaInstance.render();
    }
  });

  const reset = () => {
    verified.value = false;
    captchaInstance?.reset();
  };

  return {
    containerRef,
    verified,
    reset,
  };
};
```

---

## Go SDK

### 安装

```bash
go get github.com/captchax/sdk/go/captchax
```

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/captchax/sdk/go/captchax"
)

func main() {
    client := captchax.NewClient(
        captchax.WithAppID("your-app-id"),
        captchax.WithServerURL("https://captchax.example.com"),
        captchax.WithTimeout(10*time.Second),
    )

    ctx := context.Background()

    // 生成滑块验证码
    slider, err := client.CreateSliderCaptcha(ctx, &captchax.SliderCaptchaRequest{
        Width:  200,
        Height: 80,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("验证码ID: %s\n", slider.ID)
    fmt.Printf("目标位置: (%d, %d)\n", slider.TargetX, slider.TargetY)

    // 验证滑块
    result, err := client.VerifySlider(ctx, &captchax.VerifyRequest{
        CaptchaID: slider.ID,
        TargetX:   150,
        TargetY:   25,
    })
    if err != nil {
        log.Fatal(err)
    }

    if result.Success {
        fmt.Println("验证成功!")
    }
}
```

### 验证中间件 (Gin 框架)

```go
package middleware

import (
    "net/http"
    "strings"

    "github.com/captchax/sdk/go/captchax"
    "github.com/gin-gonic/gin"
)

func CaptchaVerify(client *captchax.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("X-Captcha-Token")
        if token == "" {
            c.JSON(http.StatusBadRequest, gin.H{
                "code":    400,
                "message": "验证码 Token 不能为空",
            })
            c.Abort()
            return
        }

        // 验证 token
        valid, err := client.VerifyToken(c.Request.Context(), token)
        if err != nil || !valid {
            c.JSON(http.StatusBadRequest, gin.H{
                "code":    400,
                "message": "验证码验证失败",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}

// 使用示例
func main() {
    r := gin.Default()

    client := captchax.NewClient(
        captchax.WithAppID("your-app-id"),
        captchax.WithServerURL("https://captchax.example.com"),
    )

    r.POST("/login",
        middleware.CaptchaVerify(client),
        loginHandler,
    )
}
```

### 点选验证

```go
// 生成点选验证码
click, err := client.CreateClickCaptcha(ctx, &captchax.ClickCaptchaRequest{
    CharCount: 4,
})
if err != nil {
    log.Fatal(err)
}

fmt.Println("需要点击的字符:", click.TargetChars)

// 验证点选
result, err := client.VerifyClick(ctx, &captchax.ClickVerifyRequest{
    CaptchaID: click.ID,
    Clicks: []captchax.ClickItem{
        {Char: "中", X: 45, Y: 30},
        {Char: "国", X: 120, Y: 25},
        {Char: "福", X: 200, Y: 40},
        {Char: "田", X: 280, Y: 35},
    },
})
```

---

## Python SDK

### 安装

```bash
pip install captchax
# 或
poetry add captchax
```

### 基础使用

```python
from captchax import CaptchaXClient

client = CaptchaXClient(
    app_id="your-app-id",
    server_url="https://captchax.example.com",
    timeout=10.0,
)

# 生成滑块验证码
slider = client.create_slider_captcha(width=200, height=80)

print(f"验证码ID: {slider.id}")
print(f"目标位置: ({slider.target_x}, {slider.target_y})")

# 验证滑块
result = client.verify_slider(
    captcha_id=slider.id,
    target_x=150,
    target_y=25,
)

if result.success:
    print("验证成功!")
```

### Django 中间件

```python
# middleware.py
from django.http import JsonResponse
from captchax import CaptchaXClient

client = CaptchaXClient(
    app_id="your-app-id",
    server_url="https://captchax.example.com",
)

class CaptchaVerifyMiddleware:
    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        if request.path.startswith('/api/'):
            token = request.headers.get('X-Captcha-Token')

            if not token:
                return JsonResponse({
                    'code': 400,
                    'message': '验证码 Token 不能为空',
                }, status=400)

            valid = client.verify_token(token)
            if not valid:
                return JsonResponse({
                    'code': 400,
                    'message': '验证码验证失败',
                }, status=400)

        response = self.get_response(request)
        return response
```

```python
# settings.py
MIDDLEWARE = [
    # ...
    'yourapp.middleware.CaptchaVerifyMiddleware',
]
```

### Flask 装饰器

```python
from functools import wraps
from flask import request, jsonify
from captchax import CaptchaXClient

client = CaptchaXClient(
    app_id="your-app-id",
    server_url="https://captchax.example.com",
)

def require_captcha(f):
    @wraps(f)
    def decorated_function(*args, **kwargs):
        token = request.headers.get('X-Captcha-Token')

        if not token:
            return jsonify({'code': 400, 'message': '验证码 Token 不能为空'}), 400

        valid = client.verify_token(token)
        if not valid:
            return jsonify({'code': 400, 'message': '验证码验证失败'}), 400

        return f(*args, **kwargs)
    return decorated_function

# 使用
@app.route('/api/login', methods=['POST'])
@require_captcha
def login():
    # 登录逻辑
    return jsonify({'success': True})
```

### 异步使用 (asyncio)

```python
import asyncio
from captchax import CaptchaXClient

client = CaptchaXClient(
    app_id="your-app-id",
    server_url="https://captchax.example.com",
)

async def main():
    slider = await client.create_slider_captcha_async(width=200, height=80)
    print(f"验证码ID: {slider.id}")

    result = await client.verify_slider_async(
        captcha_id=slider.id,
        target_x=150,
        target_y=25,
    )
    print(f"验证结果: {result.success}")

asyncio.run(main())
```

---

## Java SDK

### Maven 依赖

```xml
<dependency>
    <groupId>com.captchax</groupId>
    <artifactId>captchax-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle 依赖

```groovy
implementation 'com.captchax:captchax-sdk:1.0.0'
```

### 基础使用

```java
package com.example;

import com.captchax.sdk.CaptchaXClient;
import com.captchax.sdk.ApiModels.*;

public class Example {
    public static void main(String[] args) {
        CaptchaXClient client = new CaptchaXClient(
            "your-app-id",
            "https://captchax.example.com"
        );

        try {
            // 生成滑块验证码
            SliderCaptchaResponse slider = client.createSliderCaptcha(
                new CaptchaConfig(200, 80)
            );

            System.out.println("验证码ID: " + slider.getId());
            System.out.println("目标位置: " + slider.getTargetX() + ", " + slider.getTargetY());

            // 验证滑块
            VerifyResponse result = client.verifySlider(
                slider.getId(),
                150,  // targetX
                25    // targetY
            );

            if (result.isSuccess()) {
                System.out.println("验证成功!");
            }

        } catch (CaptchaXException e) {
            System.err.println("错误: " + e.getMessage());
        }
    }
}
```

### Spring Boot 集成

```java
package com.example.config;

import com.captchax.sdk.CaptchaXClient;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class CaptchaXConfig {

    @Value("${captchax.app-id}")
    private String appId;

    @Value("${captchax.server-url}")
    private String serverUrl;

    @Bean
    public CaptchaXClient captchaXClient() {
        return new CaptchaXClient(appId, serverUrl);
    }
}
```

```java
package com.example.interceptor;

import com.captchax.sdk.CaptchaXClient;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.HandlerInterceptor;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

@Component
public class CaptchaVerifyInterceptor implements HandlerInterceptor {

    @Autowired
    private CaptchaXClient client;

    @Override
    public boolean preHandle(HttpServletRequest request, HttpServletResponse response, Object handler) throws Exception {
        String token = request.getHeader("X-Captcha-Token");

        if (token == null || token.isEmpty()) {
            response.setStatus(HttpServletResponse.SC_BAD_REQUEST);
            response.getWriter().write("{\"code\":400,\"message\":\"验证码 Token 不能为空\"}");
            return false;
        }

        try {
            boolean valid = client.verifyToken(token);
            if (!valid) {
                response.setStatus(HttpServletResponse.SC_BAD_REQUEST);
                response.getWriter().write("{\"code\":400,\"message\":\"验证码验证失败\"}");
                return false;
            }
        } catch (Exception e) {
            response.setStatus(HttpServletResponse.SC_INTERNAL_SERVER_ERROR);
            response.getWriter().write("{\"code\":500,\"message\":\"验证码服务错误\"}");
            return false;
        }

        return true;
    }
}
```

---

## .NET SDK

### 安装

```bash
dotnet add package CaptchaX.SDK
```

### 基础使用

```csharp
using CaptchaX.SDK;

var client = new CaptchaXClient(
    appId: "your-app-id",
    serverUrl: "https://captchax.example.com"
);

// 生成滑块验证码
var slider = await client.CreateSliderCaptchaAsync(new SliderCaptchaRequest
{
    Width = 200,
    Height = 80
});

Console.WriteLine($"验证码ID: {slider.Id}");
Console.WriteLine($"目标位置: ({slider.TargetX}, {slider.TargetY})");

// 验证滑块
var result = await client.VerifySliderAsync(new VerifySliderRequest
{
    CaptchaId = slider.Id,
    TargetX = 150,
    TargetY = 25
});

if (result.Success)
{
    Console.WriteLine("验证成功!");
}
```

### ASP.NET Core 中间件

```csharp
// CaptchaVerifyMiddleware.cs
using CaptchaX.SDK;

public class CaptchaVerifyMiddleware
{
    private readonly RequestDelegate _next;
    private readonly CaptchaXClient _client;

    public CaptchaVerifyMiddleware(RequestDelegate next, CaptchaXClient client)
    {
        _next = next;
        _client = client;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        if (context.Request.Path.StartsWithSegments("/api"))
        {
            var token = context.Request.Headers["X-Captcha-Token"].FirstOrDefault();

            if (string.IsNullOrEmpty(token))
            {
                context.Response.StatusCode = 400;
                await context.Response.WriteAsJsonAsync(new { code = 400, message = "验证码 Token 不能为空" });
                return;
            }

            try
            {
                var valid = await _client.VerifyTokenAsync(token);
                if (!valid)
                {
                    context.Response.StatusCode = 400;
                    await context.Response.WriteAsJsonAsync(new { code = 400, message = "验证码验证失败" });
                    return;
                }
            }
            catch (Exception)
            {
                context.Response.StatusCode = 500;
                await context.Response.WriteAsJsonAsync(new { code = 500, message = "验证码服务错误" });
                return;
            }
        }

        await _next(context);
    }
}

// Program.cs
builder.Services.AddSingleton(new CaptchaXClient(
    appId: builder.Configuration["CaptchaX:AppId"]!,
    serverUrl: builder.Configuration["CaptchaX:ServerUrl"]!
));

app.UseMiddleware<CaptchaVerifyMiddleware>();
```

---

## PHP SDK

### 安装

```bash
composer require captchax/sdk
```

### 基础使用

```php
<?php

require_once 'vendor/autoload.php';

use CaptchaX\CaptchaXClient;

$client = new CaptchaXClient([
    'app_id' => 'your-app-id',
    'server_url' => 'https://captchax.example.com',
    'timeout' => 10.0,
]);

// 生成滑块验证码
$slider = $client->createSliderCaptcha([
    'width' => 200,
    'height' => 80,
]);

echo "验证码ID: " . $slider['id'] . "\n";
echo "目标位置: " . $slider['target_x'] . ", " . $slider['target_y'] . "\n";

// 验证滑块
$result = $client->verifySlider([
    'captcha_id' => $slider['id'],
    'target_x' => 150,
    'target_y' => 25,
]);

if ($result['success']) {
    echo "验证成功!\n";
}
```

### Laravel 集成

```php
<?php
// config/captchax.php

return [
    'app_id' => env('CAPTCHA_APP_ID'),
    'server_url' => env('CAPTCHA_SERVER_URL'),
    'timeout' => 10.0,
];
```

```php
<?php
// app/Providers/CaptchaXServiceProvider.php

namespace App\Providers;

use Illuminate\Support\ServiceProvider;
use CaptchaX\CaptchaXClient;

class CaptchaXServiceProvider extends ServiceProvider
{
    public function register()
    {
        $this->app->singleton(CaptchaXClient::class, function ($app) {
            return new CaptchaXClient([
                'app_id' => config('captchax.app_id'),
                'server_url' => config('captchax.server_url'),
                'timeout' => config('captchax.timeout'),
            ]);
        });
    }

    public function boot()
    {
        //
    }
}
```

```php
<?php
// app/Http/Middleware/VerifyCaptcha.php

namespace App\Http\Middleware;

use Closure;
use CaptchaX\CaptchaXClient;
use Illuminate\Http\Request;

class VerifyCaptcha
{
    protected CaptchaXClient $client;

    public function __construct(CaptchaXClient $client)
    {
        $this->client = $client;
    }

    public function handle(Request $request, Closure $next)
    {
        $token = $request->header('X-Captcha-Token');

        if (empty($token)) {
            return response()->json([
                'code' => 400,
                'message' => '验证码 Token 不能为空',
            ], 400);
        }

        try {
            $valid = $this->client->verifyToken($token);

            if (!$valid) {
                return response()->json([
                    'code' => 400,
                    'message' => '验证码验证失败',
                ], 400);
            }
        } catch (\Exception $e) {
            return response()->json([
                'code' => 500,
                'message' => '验证码服务错误',
            ], 500);
        }

        return $next($request);
    }
}
```

```php
// app/Http/Kernel.php
protected $routeMiddleware = [
    'captcha' => \App\Http\Middleware\VerifyCaptcha::class,
];

// routes/api.php
Route::post('/login', [AuthController::class, 'login'])
    ->middleware('captcha');
```

---

## Ruby SDK

### 安装

```ruby
# Gemfile
gem 'captchax', '~> 1.0.0'

# 或命令行
gem install captchax
```

### 基础使用

```ruby
require 'captchax'

client = CaptchaX::Client.new(
  app_id: 'your-app-id',
  server_url: 'https://captchax.example.com',
  timeout: 10
)

# 生成滑块验证码
slider = client.create_slider_captcha(width: 200, height: 80)

puts "验证码ID: #{slider.id}"
puts "目标位置: #{slider.target_x}, #{slider.target_y}"

# 验证滑块
result = client.verify_slider(
  captcha_id: slider.id,
  target_x: 150,
  target_y: 25
)

if result.success
  puts "验证成功!"
end
```

### Rails 集成

```ruby
# config/initializers/captchax.rb

CaptchaX.configure do |config|
  config.app_id = ENV['CAPTCHA_APP_ID']
  config.server_url = ENV['CAPTCHA_SERVER_URL']
  config.timeout = 10
end
```

```ruby
# app/middleware/captcha_verify.rb

class CaptchaVerify
  def initialize(app)
    @app = app
  end

  def call(env)
    if env['PATH_INFO'].start_with?('/api/')
      token = env['HTTP_X_CAPTCHA_TOKEN']

      if token.nil? || token.empty?
        return [400, { 'Content-Type' => 'application/json' },
          [{ code: 400, message: '验证码 Token 不能为空' }.to_json]]
      end

      begin
        client = CaptchaX::Client.new(
          app_id: ENV['CAPTCHA_APP_ID'],
          server_url: ENV['CAPTCHA_SERVER_URL']
        )
        valid = client.verify_token(token)

        unless valid
          return [400, { 'Content-Type' => 'application/json' },
            [{ code: 400, message: '验证码验证失败' }.to_json]]
        end
      rescue => e
        Rails.logger.error("CaptchaX error: #{e.message}")
        return [500, { 'Content-Type' => 'application/json' },
          [{ code: 500, message: '验证码服务错误' }.to_json]]
      end
    end

    @app.call(env)
  end
end
```

```ruby
# config/application.rb

module YourApp
  class Application < Rails::Application
    config.middleware.use CaptchaVerify
  end
end
```

---

## 错误处理

所有 SDK 都遵循统一的错误处理机制：

```python
from captchax.exceptions import (
    CaptchaXException,
    CaptchaNotFoundError,
    CaptchaExpiredError,
    CaptchaVerifyFailedError,
    NetworkError,
    APIError,
)

try:
    result = client.verify_slider(...)
except CaptchaNotFoundError:
    print("验证码不存在")
except CaptchaExpiredError:
    print("验证码已过期")
except CaptchaVerifyFailedError:
    print("验证失败")
except NetworkError as e:
    print(f"网络错误: {e}")
except APIError as e:
    print(f"API 错误: {e.code} - {e.message}")
except CaptchaXException as e:
    print(f"未知错误: {e}")
```

---

## 最佳实践

### 1. 客户端初始化

```python
# 推荐：单例模式
_client = None

def get_client():
    global _client
    if _client is None:
        _client = CaptchaXClient(...)
    return _client
```

### 2. 超时配置

```python
client = CaptchaXClient(
    timeout=5.0,      # 请求超时
    connect_timeout=3.0,  # 连接超时
    retry_times=3,        # 重试次数
)
```

### 3. 缓存验证结果

```python
# 验证成功后缓存 token，有效期内无需重复验证
cache_key = f"captcha:verified:{user_id}"
if not redis.exists(cache_key):
    valid = client.verify_token(token)
    if valid:
        redis.setex(cache_key, 1800, "1")  # 30分钟有效期
```

### 4. 日志记录

```python
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger('captchax')

client = CaptchaXClient(
    logger=logger,
    log_level='debug',
)
```

---

## SDK 对照表

| 功能 | JS/TS | Go | Python | Java | .NET | PHP | Ruby |
|------|-------|-----|--------|------|------|-----|------|
| 滑块验证 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 点选验证 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 拼图验证 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 批量验证 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Webhook | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 异步支持 | ✅ | ✅ | ✅ | - | - | - | - |
| 框架中间件 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
