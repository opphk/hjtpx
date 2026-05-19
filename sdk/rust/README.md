# HJTPX Captcha SDK for Rust

完整的Rust语言行为验证系统SDK，支持同步和异步操作。

## 功能特性

- ✅ 多种验证码类型：滑块、点击、图形、旋转、手势、拼图、语音、3D、连连看
- ✅ 异步支持：使用tokio实现高性能异步操作
- ✅ 连接池管理：高效的HTTP连接复用
- ✅ 自动重试机制：指数退避算法
- ✅ 完善的错误处理：针对不同错误类型提供专用错误类型
- ✅ 用户认证：注册、登录、令牌刷新
- ✅ 线程安全：支持多线程并发使用

## 安装

在 `Cargo.toml` 中添加依赖：

```toml
[dependencies]
hjtpx-captcha = "1.0"
tokio = { version = "1.0", features = ["full"] }
```

## 快速开始

### 基础使用

```rust
use hjtpx_captcha::{CaptchaClient, CaptchaConfig, TrajectoryPoint};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let config = CaptchaConfig::default();
    let client = CaptchaClient::new("http://localhost:8080", config);

    // 获取滑块验证码
    let slider = client.get_slider_captcha(320, 160, 8).await?;
    println!("Session ID: {}", slider.session_id);

    // 验证验证码
    let result = client.verify_slider_captcha(
        &slider.session_id,
        150,
        slider.secret_y,
        None,
    ).await?;
    
    println!("验证成功: {}", result.success);

    Ok(())
}
```

### 配置选项

```rust
use hjtpx_captcha::CaptchaConfig;
use std::time::Duration;

let config = CaptchaConfig::new()
    .with_api_key("your-api-key")           // API密钥
    .with_timeout(Duration::from_secs(60))  // 超时时间
    .with_max_retries(5)                     // 最大重试次数
    .with_pool_size(20);                    // 连接池大小
```

### 异步并发

```rust
use futures::future::join_all;

let tasks: Vec<_> = (0..10).map(|_| {
    let client = client.clone();
    async move {
        client.get_slider_captcha(320, 160, 8).await
    }
}).collect();

let results = join_all(tasks).await;
```

## API 参考

### 验证码类型

#### 滑块验证码

```rust
// 获取
let slider = client.get_slider_captcha(320, 160, 8).await?;

// 验证
let result = client.verify_slider_captcha(
    &slider.session_id,
    150,                    // X坐标
    slider.secret_y,        // Y坐标
    Some(trajectory),       // 轨迹数据
).await?;
```

#### 点击验证码

```rust
// 获取
let click = client.get_click_captcha("number", 3, true).await?;

// 验证
let result = client.verify_click_captcha(
    &click.session_id,
    vec![vec![100, 100], vec![200, 150], vec![300, 200]],  // 点击坐标
    None,                                                  // 点击顺序
).await?;
```

#### 图形验证码

```rust
// 获取
let image = client.get_image_captcha("mixed", 4).await?;

// 验证
let result = client.verify_image_captcha(
    &image.challenge_id,
    "ABCD",  // 答案
).await?;
```

#### 手势验证码

```rust
// 获取
let gesture = client.get_gesture_captcha().await?;

// 验证
let result = client.verify_gesture_captcha(
    &gesture.session_id,
    vec![1, 2, 3, 6, 9, 8, 7, 4],  // 手势模式
).await?;
```

#### 拼图验证码

```rust
// 获取
let jigsaw = client.get_jigsaw_captcha(300, 300, 3).await?;

// 验证
let result = client.verify_jigsaw_captcha(
    &jigsaw.session_id,
    pieces,  // 碎片列表
).await?;
```

#### 语音验证码

```rust
// 获取
let voice = client.get_voice_captcha().await?;

// 验证
let result = client.verify_voice_captcha(
    &voice.session_id,
    "1234",  // 语音答案
).await?;
```

#### 3D验证码

```rust
// 获取
let captcha_3d = client.get_3d_captcha().await?;

// 验证
let result = client.verify_3d_captcha(
    &captcha_3d.session_id,
    45,  // 旋转角度
).await?;
```

#### 连连看验证码

```rust
// 获取
let lianliankan = client.get_lianliankan_captcha(6).await?;

// 验证
let result = client.verify_lianliankan_captcha(
    &lianliankan.session_id,
    path,  // 连接路径
).await?;
```

### 用户认证

```rust
use hjtpx_captcha::LoginRequest;

// 登录
let login_req = LoginRequest {
    username: "user".to_string(),
    password: "pass".to_string(),
    captcha_token: None,
};

let response = client.login(&login_req).await?;
println!("Token: {}", response.access_token);

// 登出
client.logout().await?;

// 刷新令牌
let new_response = client.refresh_token(&response.refresh_token).await?;
```

## 错误处理

```rust
use hjtpx_captcha::CaptchaError;

match client.get_slider_captcha(320, 160, 8).await {
    Ok(slider) => {
        println!("Success: {}", slider.session_id);
    }
    Err(CaptchaError::NetworkError(msg)) => {
        println!("Network error: {}", msg);
    }
    Err(CaptchaError::TimeoutError) => {
        println!("Request timeout");
    }
    Err(CaptchaError::SessionExpired) => {
        println!("Session expired");
    }
    Err(CaptchaError::ApiError { code, message }) => {
        println!("API error {}: {}", code, message);
    }
    Err(e) => {
        println!("Other error: {}", e);
    }
}
```

## 轨迹数据

轨迹数据对于通过滑块验证非常重要。建议按以下格式构造：

```rust
use hjtpx_captcha::TrajectoryPoint;
use std::time::{SystemTime, UNIX_EPOCH};

let now = SystemTime::now()
    .duration_since(UNIX_EPOCH)?
    .as_millis() as i64;

let trajectory = vec![
    TrajectoryPoint::new(0, 80, now - 1000),        // 起点
    TrajectoryPoint::new(50, 85, now - 800),        // 缓慢移动
    TrajectoryPoint::new(100, 75, now - 500),       // 中间点
    TrajectoryPoint::new(150, 82, now - 200),      // 继续移动
    TrajectoryPoint::new(180, 80, now),             // 终点
];
```

## 运行示例

```bash
# 基础示例
cargo run --example basic_example

# 异步示例
cargo run --example async_example
```

## 测试

```bash
# 运行所有测试
cargo test

# 运行特定测试
cargo test test_trajectory

# 运行文档测试
cargo test --doc
```

## 性能考虑

1. **连接池大小**：根据并发需求调整 `pool_size`
2. **超时设置**：生产环境建议设置合理的超时时间
3. **重试策略**：默认使用指数退避算法
4. **异步并发**：使用 `futures::future::join_all` 或 `tokio::spawn` 进行并发操作

## 依赖

- `reqwest`: HTTP客户端
- `serde`: 序列化/反序列化
- `tokio`: 异步运行时
- `thiserror`: 错误处理
- `async-trait`: 异步 trait 支持

## 许可证

MIT License
