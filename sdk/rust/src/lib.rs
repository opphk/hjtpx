//! HJTPX Captcha SDK for Rust
//! 
//! 提供完整的验证码功能，包括滑块、点击、旋转、手势、拼图等多种验证码类型。
//! 支持连接池管理、自动重试和错误处理。

pub mod client;
pub mod errors;
pub mod models;

pub use client::CaptchaClient;
pub use errors::CaptchaError;
pub use models::*;

#[cfg(test)]
mod tests {
    use super::*;
    use tokio;

    #[tokio::test]
    async fn test_client_creation() {
        let client = CaptchaClient::new("http://localhost:8080");
        assert_eq!(client.base_url(), "http://localhost:8080");
    }
}
