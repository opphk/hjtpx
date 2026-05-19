//! 验证码配置模块

use std::time::Duration;

/// 验证码客户端配置
#[derive(Debug, Clone)]
pub struct CaptchaConfig {
    /// API密钥
    pub api_key: Option<String>,
    /// 请求超时时间
    pub timeout: Duration,
    /// 最大重试次数
    pub max_retries: u32,
    /// 重试间隔基数（秒）
    pub retry_base_delay: f64,
    /// 连接池大小
    pub pool_size: u32,
}

impl Default for CaptchaConfig {
    fn default() -> Self {
        Self {
            api_key: None,
            timeout: Duration::from_secs(30),
            max_retries: 3,
            retry_base_delay: 0.5,
            pool_size: 10,
        }
    }
}

impl CaptchaConfig {
    /// 创建新配置
    pub fn new() -> Self {
        Self::default()
    }

    /// 设置API密钥
    pub fn with_api_key(mut self, api_key: impl Into<String>) -> Self {
        self.api_key = Some(api_key.into());
        self
    }

    /// 设置超时时间
    pub fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// 设置最大重试次数
    pub fn with_max_retries(mut self, max_retries: u32) -> Self {
        self.max_retries = max_retries;
        self
    }

    /// 设置连接池大小
    pub fn with_pool_size(mut self, pool_size: u32) -> Self {
        self.pool_size = pool_size;
        self
    }
}
