# Hjtpx Captcha Rust SDK v15.0

## 特性

- 同步和异步客户端
- 高性能 async/await 支持
- 完整的类型安全
- 自动重试和退避
- 速率限制
- 支持所有验证码类型

## 安装

添加以下内容到 `Cargo.toml`：

```toml
[dependencies]
hjtpx-captcha = "15.0"
tokio = { version = "1", features = ["full"] }
```

## 快速开始

### 基本使用

```rust
use hjtpx_captcha::{CaptchaClient, Result};
use std::time::Duration;

#[tokio::main]
async fn main() -> Result<()> {
    let client = CaptchaClient::new("http://localhost:8080")
        .with_api_key("your-api-key")
        .with_timeout(Duration::from_secs(30));

    // 获取滑块验证码
    let captcha = client.get_slider_captcha(320, 160, 8).await?;
    println!("Session ID: {}", captcha.session_id);

    // 创建轨迹
    let trajectory = vec![
        hjtpx_captcha::TrajectoryPoint {
            x: 0,
            y: captcha.secret_y.unwrap_or(50),
            t: chrono::Utc::now().timestamp_millis() - 1000,
        },
        hjtpx_captcha::TrajectoryPoint {
            x: 50,
            y: captcha.secret_y.unwrap_or(50) + 5,
            t: chrono::Utc::now().timestamp_millis() - 800,
        },
        // ... 更多轨迹点
    ];

    // 验证验证码
    let result = client
        .verify_slider_captcha(
            &captcha.session_id,
            185,
            captcha.secret_y,
            Some(trajectory),
        )
        .await?;

    println!("Success: {}", result.success);
    Ok(())
}
```

### 批量验证

```rust
use hjtpx_captcha::{CaptchaClient, VerifyCaptchaRequest, Result};

#[tokio::main]
async fn main() -> Result<()> {
    let client = CaptchaClient::new("http://localhost:8080");

    let requests = vec![
        VerifyCaptchaRequest {
            session_id: "session-1".to_string(),
            x: 100,
            y: Some(50),
            trajectory: None,
            r#type: "slider".to_string(),
        },
        VerifyCaptchaRequest {
            session_id: "session-2".to_string(),
            x: 150,
            y: Some(60),
            trajectory: None,
            r#type: "slider".to_string(),
        },
    ];

    let result = client.batch_verify(requests).await?;

    println!("Success: {}, Failed: {}", result.success_count, result.failed_count);
    println!("Total time: {}ms", result.total_time_ms);

    for r in &result.results {
        println!("Session {}: {}", r.session_id, if r.success { "OK" } else { "FAILED" });
    }

    Ok(())
}
```

### 异步验证

```rust
use hjtpx_captcha::{CaptchaClient, Result};

#[tokio::main]
async fn main() -> Result<()> {
    let client = CaptchaClient::new("http://localhost:8080");

    // 发起异步验证
    let async_result = client
        .async_verify(
            "session-1",
            150,
            Some(50),
            None,
            Some("https://example.com/callback".to_string()),
        )
        .await?;

    println!("Task ID: {}", async_result.task_id);
    println!("Status: {}", async_result.status);

    // 等待结果
    let result = client
        .wait_async_result(&async_result.task_id, 10, 500)
        .await?;

    println!("Final Status: {}", result.status);
    if let Some(result_data) = result.result {
        println!("Success: {}", result_data.success);
    }

    Ok(())
}
```

### 多种验证码类型

```rust
use hjtpx_captcha::{CaptchaClient, Result};

#[tokio::main]
async fn main() -> Result<()> {
    let client = CaptchaClient::new("http://localhost:8080");

    // 点击验证码
    let captcha = client.get_click_captcha("number", 3, true).await?;
    let points = vec![vec![100, 100], vec![200, 150], vec![150, 200]];
    let result = client
        .verify_click_captcha(&captcha.session_id, points, Some(vec![1, 2, 3]))
        .await?;

    // 图形验证码
    let captcha = client.get_image_captcha("mixed", 4).await?;
    let result = client
        .verify_image_captcha(&captcha.challenge_id, "ABCD")
        .await?;

    // 旋转验证码
    let captcha = client.get_rotation_captcha().await?;
    let result = client
        .verify_rotation_captcha(&captcha.challenge_id, 45)
        .await?;

    // 手势验证码
    let captcha = client.get_gesture_captcha().await?;
    let result = client
        .verify_gesture_captcha(&captcha.session_id, vec![0, 1, 2, 3])
        .await?;

    // 拼图验证码
    let captcha = client.get_jigsaw_captcha(300, 300, 3).await?;
    let result = client
        .verify_jigsaw_captcha(&captcha.session_id, captcha.pieces)
        .await?;

    Ok(())
}
```

## 错误处理

```rust
use hjtpx_captcha::{CaptchaClient, CaptchaError, Result};

#[tokio::main]
async fn main() -> Result<()> {
    let client = CaptchaClient::new("http://localhost:8080");

    match client.verify_slider_captcha("invalid", 100, None, None).await {
        Ok(result) => println!("Success: {}", result.success),
        Err(e) => {
            match e {
                CaptchaError::ApiError { message, code } => {
                    eprintln!("API Error {}: {}", code, message);
                }
                CaptchaError::NetworkError(e) => {
                    eprintln!("Network Error: {}", e);
                }
                CaptchaError::TimeoutError(msg) => {
                    eprintln!("Timeout: {}", msg);
                }
                CaptchaError::ValidationError(msg) => {
                    eprintln!("Validation Error: {}", msg);
                }
                CaptchaError::RateLimitError(msg) => {
                    eprintln!("Rate Limited: {}", msg);
                }
                _ => {
                    eprintln!("Error: {}", e);
                }
            }
        }
    }

    Ok(())
}
```

## 配置选项

### 基础配置

```rust
let client = CaptchaClient::new("http://localhost:8080")
    .with_api_key("your-api-key")
    .with_timeout(Duration::from_secs(60));
```

### 重试配置

```rust
use hjtpx_captcha::utils::RetryConfig;

let client = CaptchaClient::new("http://localhost:8080")
    .with_retry_config(RetryConfig {
        max_retries: 5,
        base_delay_ms: 100,
        max_delay_ms: 5000,
    });
```

### 速率限制

```rust
let client = CaptchaClient::new("http://localhost:8080")
    .with_rate_limit(100); // 每秒100请求
```

## API 参考

### CaptchaClient

#### 创建客户端

```rust
CaptchaClient::new(base_url)
    .with_api_key(api_key)
    .with_timeout(timeout)
    .with_retry_config(config)
    .with_rate_limit(requests_per_second)
```

#### 滑块验证码

```rust
get_slider_captcha(width, height, tolerance) -> Result<SliderCaptchaResponse>
verify_slider_captcha(session_id, x, y, trajectory) -> Result<VerifyCaptchaResponse>
```

#### 点击验证码

```rust
get_click_captcha(mode, max_points, allow_shuffle) -> Result<ClickCaptchaResponse>
verify_click_captcha(session_id, points, click_sequence) -> Result<VerifyCaptchaResponse>
```

#### 图形验证码

```rust
get_image_captcha(type, count) -> Result<ImageCaptchaResponse>
verify_image_captcha(challenge_id, answer) -> Result<VerifyCaptchaResponse>
```

#### 旋转验证码

```rust
get_rotation_captcha() -> Result<RotationCaptchaResponse>
verify_rotation_captcha(challenge_id, angle) -> Result<VerifyCaptchaResponse>
```

#### 手势验证码

```rust
get_gesture_captcha() -> Result<GestureCaptchaResponse>
verify_gesture_captcha(session_id, pattern) -> Result<VerifyCaptchaResponse>
```

#### 拼图验证码

```rust
get_jigsaw_captcha(width, height, grid_size) -> Result<JigsawCaptchaResponse>
verify_jigsaw_captcha(session_id, pieces) -> Result<VerifyCaptchaResponse>
```

#### 批量和异步验证

```rust
batch_verify(requests) -> Result<BatchVerifyResponse>
async_verify(session_id, x, y, trajectory, callback_url) -> Result<AsyncVerifyResponse>
get_async_result(task_id) -> Result<AsyncResultResponse>
wait_async_result(task_id, timeout_secs, poll_interval_ms) -> Result<AsyncResultResponse>
```

#### 认证

```rust
login(username, password, captcha_token) -> Result<LoginResponse>
logout() -> Result<()>
get_detection_script(callback) -> Result<String>
```

## 性能优化

### 连接复用

```rust
// 客户端应被复用，而不是每次请求创建新客户端
let client = CaptchaClient::new("http://localhost:8080");

// 多次请求使用同一个客户端
for _ in 0..100 {
    let captcha = client.get_slider_captcha(320, 160, 8).await?;
    // ...
}
```

### 并发请求

```rust
use futures::future::join_all;

#[tokio::main]
async fn main() -> Result<()> {
    let client = CaptchaClient::new("http://localhost:8080");

    let futures: Vec<_> = (0..10)
        .map(|_| client.get_slider_captcha(320, 160, 8))
        .collect();

    let results = join_all(futures).await;

    for (i, result) in results.into_iter().enumerate() {
        match result {
            Ok(captcha) => println!("Request {}: {}", i, captcha.session_id),
            Err(e) => println!("Request {} failed: {}", i, e),
        }
    }

    Ok(())
}
```

## 许可

MIT License
