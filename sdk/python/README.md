# Hjtpx Captcha Python SDK v15.0

## 特性

- 同步和异步客户端
- 完整的类型提示
- 支持所有验证码类型
- 批量验证和异步验证
- 连接池优化
- 自动重试机制
- 速率限制
- 完整的错误处理

## 安装

```bash
pip install hjtpx-captcha
```

或使用 asyncio 版本（推荐）：

```bash
pip install hjtpx-captcha[async]
```

## 快速开始

### 同步客户端

```python
from captcha import CaptchaClient, TrajectoryPoint

client = CaptchaClient(
    base_url="http://localhost:8080",
    api_key="your-api-key",
    timeout=30,
)

# 获取滑块验证码
captcha = client.get_slider_captcha(width=320, height=160, tolerance=8)
print(f"Session ID: {captcha.session_id}")

# 验证验证码
result = client.verify_slider_captcha(
    session_id=captcha.session_id,
    x=185,
    y=captcha.secret_y,
    trajectory=[
        TrajectoryPoint(x=0, y=50, t=1700000000000),
        TrajectoryPoint(x=50, y=55, t=1700000000100),
        TrajectoryPoint(x=100, y=47, t=1700000000200),
        TrajectoryPoint(x=150, y=52, t=1700000000300),
        TrajectoryPoint(x=185, y=50, t=1700000000400),
    ]
)

if result.success:
    print("Verification passed!")
```

### 异步客户端

```python
import asyncio
from async_captcha import AsyncCaptchaClient

async def main():
    async with AsyncCaptchaClient(
        base_url="http://localhost:8080",
        api_key="your-api-key",
    ) as client:
        # 获取验证码
        captcha = await client.get_slider_captcha()
        print(f"Session ID: {captcha.session_id}")

        # 验证验证码
        result = await client.verify_slider_captcha(
            session_id=captcha.session_id,
            x=150,
            y=captcha.secret_y,
        )
        print(f"Success: {result.success}")

asyncio.run(main())
```

### 使用上下文管理器

```python
from captcha import CaptchaClient

with CaptchaClient("http://localhost:8080") as client:
    captcha = client.get_slider_captcha()
    result = client.verify_slider_captcha(captcha.session_id, 150)
    print(result.success)
```

### 批量验证

```python
# 同步批量验证
requests = [
    {"session_id": "session-1", "x": 100, "y": 50},
    {"session_id": "session-2", "x": 150, "y": 60},
    {"session_id": "session-3", "x": 200, "y": 70},
]

result = client.batch_verify(requests)
print(f"Success: {result.success_count}, Failed: {result.failed_count}")
print(f"Total time: {result.total_time_ms}ms")

# 异步批量验证
import asyncio
from async_captcha import AsyncCaptchaClient, create_async_client

async def async_batch():
    async with create_async_client("http://localhost:8080") as client:
        result = await client.batch_verify(requests, max_concurrent=10)
        print(f"Success: {result.success_count}")
```

### 异步验证（服务端异步）

```python
import asyncio
from async_captcha import AsyncCaptchaClient

async def async_verify():
    async with AsyncCaptchaClient("http://localhost:8080") as client:
        # 发起异步验证
        async_result = await client.async_verify(
            session_id="session-1",
            x=150,
            y=50,
            callback_url="https://example.com/callback",
        )
        print(f"Task ID: {async_result.task_id}")

        # 等待结果
        result = await client.wait_async_result(
            async_result.task_id,
            timeout=10.0,
            poll_interval=0.5,
        )
        print(f"Status: {result.status}")
        if result.result:
            print(f"Success: {result.result.success}")
```

### 多种验证码类型

```python
# 点击验证码
captcha = client.get_click_captcha(mode="number", max_points=3)
print(f"Hint: {captcha.hint}")
result = client.verify_click_captcha(
    session_id=captcha.session_id,
    points=[[100, 100], [200, 150], [150, 200]],
    click_sequence=[1, 2, 3],
)

# 图形验证码
captcha = client.get_image_captcha(type="mixed", count=4)
print(f"Challenge ID: {captcha.challenge_id}")
result = client.verify_image_captcha(captcha.challenge_id, "ABCD")

# 旋转验证码
captcha = client.get_rotation_captcha()
result = client.verify_rotation_captcha(captcha.challenge_id, angle=45)

# 手势验证码
captcha = client.get_gesture_captcha()
result = client.verify_gesture_captcha(captcha.session_id, pattern=[0, 1, 2, 3])

# 拼图验证码
captcha = client.get_jigsaw_captcha(width=300, height=300, grid_size=3)
result = client.verify_jigsaw_captcha(captcha.session_id, captcha.pieces)
```

## 错误处理

```python
from captcha import (
    CaptchaError,
    CaptchaAPIError,
    CaptchaNetworkError,
    CaptchaTimeoutError,
    CaptchaValidationError,
)

try:
    result = client.verify_slider_captcha("invalid-session", 100)
except CaptchaAPIError as e:
    print(f"API Error: {e.message}, Code: {e.code}")
except CaptchaNetworkError as e:
    print(f"Network Error: {e}")
except CaptchaTimeoutError as e:
    print(f"Timeout: {e}")
except CaptchaValidationError as e:
    print(f"Validation Error: {e}")
except CaptchaError as e:
    print(f"Error: {e}")

# 异步客户端错误
from async_captcha import (
    AsyncCaptchaError,
    AsyncCaptchaAPIError,
    AsyncCaptchaRateLimitError,
)

async def async_error_handling():
    try:
        result = await client.verify_slider_captcha("invalid", 100)
    except AsyncCaptchaRateLimitError as e:
        print(f"Rate limited: {e}")
    except AsyncCaptchaAPIError as e:
        print(f"API Error: {e.message}")
```

## 配置选项

### 同步客户端

```python
from captcha import CaptchaClient

client = CaptchaClient(
    base_url="http://localhost:8080",
    api_key="your-api-key",
    timeout=30,                    # 请求超时（秒）
    max_retries=3,                 # 最大重试次数
    retry_backoff_factor=0.5,      # 重试退避因子
    pool_connections=10,           # 连接池大小
    pool_maxsize=10,               # 最大连接数
)
```

### 异步客户端

```python
from async_captcha import AsyncCaptchaClient

client = AsyncCaptchaClient(
    base_url="http://localhost:8080",
    api_key="your-api-key",
    timeout=30,                     # 请求超时（秒）
    max_retries=3,                  # 最大重试次数
    retry_backoff_factor=0.5,       # 重试退避因子
    max_connections=100,            # 最大并发连接数
    requests_per_second=50,         # 每秒请求数限制（可选）
)
```

## 速率限制

```python
from async_captcha import AsyncCaptchaClient

# 启用速率限制
client = AsyncCaptchaClient(
    "http://localhost:8080",
    requests_per_second=50,  # 每秒最多50个请求
)
```

## API 参考

### CaptchaClient

| 方法 | 描述 |
|------|------|
| `get_slider_captcha(width, height, tolerance)` | 获取滑块验证码 |
| `verify_slider_captcha(session_id, x, y, trajectory)` | 验证滑块验证码 |
| `get_click_captcha(mode, max_points, allow_shuffle)` | 获取点击验证码 |
| `verify_click_captcha(session_id, points, click_sequence)` | 验证点击验证码 |
| `get_image_captcha(type, count)` | 获取图形验证码 |
| `verify_image_captcha(challenge_id, answer)` | 验证图形验证码 |
| `batch_verify(requests)` | 批量验证 |

### AsyncCaptchaClient

| 方法 | 描述 |
|------|------|
| `get_slider_captcha(...)` | 异步获取滑块验证码 |
| `verify_slider_captcha(...)` | 异步验证滑块验证码 |
| `batch_verify(requests, max_concurrent)` | 异步批量验证 |
| `async_verify(...)` | 发起异步验证 |
| `get_async_result(task_id)` | 获取异步验证结果 |
| `wait_async_result(task_id, timeout, poll_interval)` | 等待异步验证结果 |

## 许可

MIT License
