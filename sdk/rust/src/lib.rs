//! HJTPX Captcha SDK for Rust
//!
//! 提供完整的行为验证系统SDK，支持同步和异步操作。
//!
//! # 快速开始
//!
//! ```rust,no_run
//! use hjtpx_captcha::{CaptchaClient, CaptchaConfig};
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     let config = CaptchaConfig::default();
//!     let client = CaptchaClient::new("http://localhost:8080", config);
//!
//!     // 获取滑块验证码
//!     let slider = client.get_slider_captcha(320, 160, 8).await?;
//!     println!("Session ID: {}", slider.session_id);
//!
//!     // 验证验证码
//!     let result = client.verify_slider_captcha(&slider.session_id, 150, None, None).await?;
//!     println!("Verification success: {}", result.success);
//!
//!     Ok(())
//! }
//! ```

pub mod client;
pub mod errors;
pub mod models;

pub use client::CaptchaClient;
pub use config::CaptchaConfig;
pub use errors::{CaptchaError, CaptchaResult};
pub use models::*;

mod config;
