//! 错误处理模块

use thiserror::Error;

/// 验证码SDK错误类型
#[derive(Error, Debug)]
pub enum CaptchaError {
    /// 网络请求错误
    #[error("网络请求错误: {0}")]
    NetworkError(String),

    /// HTTP状态码错误
    #[error("HTTP错误: 状态码 {status}, 消息: {message}")]
    HttpError { status: u16, message: String },

    /// API返回错误
    #[error("API错误: 代码={code}, 消息={message}")]
    ApiError { code: i32, message: String },

    /// 超时错误
    #[error("请求超时")]
    TimeoutError,

    /// 验证失败
    #[error("验证失败: {0}")]
    ValidationError(String),

    /// 会话过期
    #[error("会话已过期或不存在")]
    SessionExpired,

    /// 参数错误
    #[error("参数错误: {0}")]
    InvalidParameter(String),

    /// JSON解析错误
    #[error("JSON解析错误: {0}")]
    JsonError(String),

    /// 连接池错误
    #[error("连接池错误: {0}")]
    PoolError(String),

    /// 未知错误
    #[error("未知错误: {0}")]
    Unknown(String),
}

/// 验证码SDK结果类型
pub type CaptchaResult<T> = Result<T, CaptchaError>;

impl From<reqwest::Error> for CaptchaError {
    fn from(err: reqwest::Error) -> Self {
        if err.is_timeout() {
            CaptchaError::TimeoutError
        } else if err.is_connect() {
            CaptchaError::NetworkError(format!("连接失败: {}", err))
        } else {
            CaptchaError::NetworkError(err.to_string())
        }
    }
}

impl From<serde_json::Error> for CaptchaError {
    fn from(err: serde_json::Error) -> Self {
        CaptchaError::JsonError(err.to_string())
    }
}
