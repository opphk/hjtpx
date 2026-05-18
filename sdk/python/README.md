# 行为验证系统 Python SDK

功能比较完整的验证码 SDK，支持多种验证码类型，提供完整的错误处理、连接池管理和自动重试机制。

## 功能特性

- **多种验证码类型支持**：
  - 滑块验证码
  - 点击验证码（数字、字母、中文、图标模式）
  - 图形验证码
  - 旋转验证码
  - 手势验证码
  - 拼图验证码

- **完善的错误处理**：
  - 清晰的异常类层次结构
  - 详细的错误信息和代码
  - 优雅的异常处理机制

- **性能优化**：
  - 连接池管理
  - 自动重试机制
  - 可配置的超时和重试策略
  - 支持高并发场景的异步客户端

- **异步支持**：
  - 完整的异步/await兼容版本
  - 支持高并发批量请求
  - 内置重试和错误处理

- **用户认证**：
  - 用户注册/登录
  - 令牌刷新
  - 邮箱验证
  - 密码重置

- **环境检测**：
  - 获取检测脚本
  - 提交检测数据
  - 环境安全检查

## 安装要求

- Python 3.7+
- requests >= 2.25.0
- urllib3 >= 1.26.0
- aiohttp >= 3.8.0 (仅异步版本需要)

```bash
# 同步版本
pip install requests

# 异步版本
pip install aiohttp
```

## 快速开始

### 同步客户端

```python
from captcha import CaptchaClient

# 创建客户端
client = CaptchaClient(
    base_url="http://localhost:8080",
    api_key="your-api-key",  # 可选
    timeout=30,
)

# 获取滑块验证码
captcha = client.get_slider_captcha()
print(f"会话ID: {captcha.session_id}")

# 验证
result = client.verify_slider_captcha(
    session_id=captcha.session_id,
    x=150,
    y=captcha.secret_y,
)
print(f"验证结果: {result.success}")

# 关闭连接
client.close()
```

### 异步客户端

```python
import asyncio
from async_captcha import AsyncCaptchaClient

async def main():
    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        # 获取验证码
        captcha = await client.get_slider_captcha()
        print(f"会话ID: {captcha.session_id}")

        # 验证
        result = await client.verify_slider_captcha(
            session_id=captcha.session_id,
            x=150,
        )
        print(f"验证结果: {result.success}")

asyncio.run(main())
```

### 批量并发请求示例

```python
import asyncio
from async_captcha import AsyncCaptchaClient

async def batch_example():
    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        # 同时获取10个验证码
        tasks = [
            client.get_slider_captcha(width=320, height=160)
            for _ in range(10)
        ]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        success_count = sum(
            1 for r in results
            if not isinstance(r, Exception)
        )
        print(f"成功率: {success_count}/{len(results)}")

asyncio.run(batch_example())
```

### 使用上下文管理器

```python
from captcha import CaptchaClient

with CaptchaClient(base_url="http://localhost:8080") as client:
    captcha = client.get_slider_captcha()
    result = client.verify_slider_captcha(
        session_id=captcha.session_id,
        x=150,
    )
    print(result.success)
```

## 验证码类型

### 1. 滑块验证码

```python
from captcha import CaptchaClient, TrajectoryPoint

client = CaptchaClient(base_url="http://localhost:8080")

# 获取验证码
captcha = client.get_slider_captcha(
    width=320,
    height=160,
    tolerance=8,
)

# 准备轨迹
trajectory = [
    TrajectoryPoint(x=0, y=80, t=0),
    TrajectoryPoint(x=50, y=85, t=200),
    TrajectoryPoint(x=100, y=77, t=400),
    TrajectoryPoint(x=150, y=80, t=800),
]

# 验证
result = client.verify_slider_captcha(
    session_id=captcha.session_id,
    x=150,
    y=captcha.secret_y,
    trajectory=trajectory,
)
```

### 2. 点击验证码

```python
from captcha import CaptchaClient, ClickMode

client = CaptchaClient(base_url="http://localhost:8080")

# 获取验证码
captcha = client.get_click_captcha(
    mode=ClickMode.NUMBER,  # 或 LETTER, CHINESE, MIXED, ICON
    max_points=3,
    allow_shuffle=True,
)

# 验证
result = client.verify_click_captcha(
    session_id=captcha.session_id,
    points=[[100, 100], [200, 100], [150, 200]],
    click_sequence=[0, 1, 2],
)
```

### 3. 图形验证码

```python
client = CaptchaClient(base_url="http://localhost:8080")

# 获取验证码
captcha = client.get_image_captcha(
    type_="mixed",  # number, letter, mixed, chinese
    count=4,
)

# 验证
result = client.verify_image_captcha(
    challenge_id=captcha.challenge_id,
    answer="ABCD",
)
```

### 4. 手势验证码

```python
client = CaptchaClient(base_url="http://localhost:8080")

# 获取验证码
captcha = client.get_gesture_captcha()

# 验证
result = client.verify_gesture_captcha(
    session_id=captcha.session_id,
    pattern=[1, 2, 3, 5, 7],  # 手势点的顺序
)
```

### 5. 拼图验证码

```python
from captcha import CaptchaClient, JigsawPiece

client = CaptchaClient(base_url="http://localhost:8080")

# 获取验证码
captcha = client.get_jigsaw_captcha(
    width=300,
    height=300,
    grid_size=3,
)

# 准备正确位置的碎片
pieces = []
for piece in captcha.pieces:
    correct_piece = JigsawPiece(
        index=piece.index,
        original_x=piece.original_x,
        original_y=piece.original_y,
        current_x=piece.original_x,
        current_y=piece.original_y,
        width=piece.width,
        height=piece.height,
        rotation=0,
    )
    pieces.append(correct_piece)

# 验证
result = client.verify_jigsaw_captcha(
    session_id=captcha.session_id,
    pieces=pieces,
)
```

### 6. 旋转验证码

```python
client = CaptchaClient(base_url="http://localhost:8080")

# 获取验证码
captcha = client.get_rotation_captcha()

# 验证（旋转角度）
result = client.verify_rotation_captcha(
    challenge_id=captcha.challenge_id,
    angle=90,
)
```

## 高级配置

### 连接池和重试配置

```python
client = CaptchaClient(
    base_url="http://localhost:8080",
    timeout=30,
    max_retries=5,  # 最大重试次数
    retry_backoff_factor=0.3,  # 重试延迟因子
    pool_connections=20,  # 连接池大小
    pool_maxsize=20,  # 最大连接数
)
```

### 用户认证

```python
client = CaptchaClient(base_url="http://localhost:8080")
auth = client.auth()

# 登录
login_result = auth.login(
    username="user",
    password="pass",
    captcha_token="token",  # 可选
)

# 刷新令牌
auth.refresh_token()

# 登出
auth.logout()
```

### 环境检测

```python
client = CaptchaClient(base_url="http://localhost:8080")
env = client.env()

# 获取检测脚本
script = env.get_detection_script(callback="onReady")

# 提交检测数据
result = env.submit_detection({
    "detection_id": "123",
    "risk_score": 0.1,
    "fingerprint": "hash",
})

# 检查环境
check_result = env.check_environment({
    "fingerprint": "hash",
    "canvas_hash": "hash",
    "webgl_vendor": "vendor",
})
```

## 错误处理

```python
from captcha import (
    CaptchaClient,
    CaptchaError,
    CaptchaAPIError,
    CaptchaNetworkError,
    CaptchaTimeoutError,
    CaptchaValidationError,
    CaptchaSessionExpiredError,
)

try:
    client = CaptchaClient(base_url="http://localhost:8080")
    captcha = client.get_slider_captcha()
    # ...
except CaptchaTimeoutError as e:
    print(f"请求超时: {e}")
except CaptchaNetworkError as e:
    print(f"网络错误: {e}")
except CaptchaSessionExpiredError as e:
    print(f"会话过期: {e}")
except CaptchaValidationError as e:
    print(f"验证失败: {e}")
except CaptchaAPIError as e:
    print(f"API错误: {e}, 代码: {e.code}")
except CaptchaError as e:
    print(f"验证码错误: {e}")
```

## API 参考

### CaptchaClient

主要的客户端类，用于与验证码服务交互。

#### 初始化参数

- `base_url`: API 基础 URL (必需)
- `api_key`: API 密钥 (可选)
- `timeout`: 请求超时时间，默认 30 秒
- `max_retries`: 最大重试次数，默认 3
- `retry_backoff_factor`: 重试延迟因子，默认 0.5
- `pool_connections`: 连接池大小，默认 10
- `pool_maxsize`: 最大连接数，默认 10

#### 验证码方法

| 方法 | 描述 |
|------|------|
| `get_slider_captcha()` | 获取滑块验证码 |
| `verify_slider_captcha()` | 验证滑块验证码 |
| `get_click_captcha()` | 获取点击验证码 |
| `verify_click_captcha()` | 验证点击验证码 |
| `get_image_captcha()` | 获取图形验证码 |
| `verify_image_captcha()` | 验证图形验证码 |
| `get_rotation_captcha()` | 获取旋转验证码 |
| `verify_rotation_captcha()` | 验证旋转验证码 |
| `get_gesture_captcha()` | 获取手势验证码 |
| `verify_gesture_captcha()` | 验证手势验证码 |
| `get_jigsaw_captcha()` | 获取拼图验证码 |
| `verify_jigsaw_captcha()` | 验证拼图验证码 |
| `verify_captcha()` | 通用验证方法 |

#### 其他方法

| 方法 | 描述 |
|------|------|
| `auth()` | 获取用户认证 API |
| `env()` | 获取环境检测 API |
| `close()` | 关闭客户端连接 |

## 示例代码

更多示例请参考 [examples.py](examples.py) 文件。

运行示例：

```bash
python examples.py
```

## 许可证

MIT License
