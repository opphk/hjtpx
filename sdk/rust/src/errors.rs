use reqwest::StatusCode;
use std::fmt;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum CaptchaError {
    #[error("网络错误: {0}")]
    Network(#[from] reqwest::Error),
    
    #[error("请求超时")]
    Timeout,
    
    #[error("无效的响应数据")]
    InvalidResponse,
    
    #[error("API错误: {0} (代码: {1})")]
    ApiError(String, i32),
    
    #[error("认证失败: {0}")]
    Authentication(String),
    
    #[error("请求被限流")]
    RateLimited,
    
    #[error("验证码已过期")]
    CaptchaExpired,
    
    #[error("验证失败")]
    VerificationFailed,
    
    #[error("无效的参数: {0}")]
    InvalidParameter(String),
    
    #[error("序列化错误: {0}")]
    Serialization(#[from] serde_json::Error),
    
    #[error("内部错误: {0}")]
    Internal(String),
    
    #[error("会话过期")]
    SessionExpired,
    
    #[error("服务器错误")]
    ServerError,
}

impl CaptchaError {
    pub fn from_status_code(status: StatusCode, message: &str) -> Self {
        match status {
            StatusCode::UNAUTHORIZED => CaptchaError::Authentication(message.to_string()),
            StatusCode::FORBIDDEN => CaptchaError::Authentication(message.to_string()),
            StatusCode::TOO_MANY_REQUESTS => CaptchaError::RateLimited,
            StatusCode::REQUEST_TIMEOUT => CaptchaError::Timeout,
            StatusCode::GATEWAY_TIMEOUT => CaptchaError::Timeout,
            s if s.is_server_error() => CaptchaError::ServerError,
            _ => CaptchaError::ApiError(message.to_string(), status.as_u16() as i32),
        }
    }

    pub fn is_retryable(&self) -> bool {
        matches!(
            self,
            CaptchaError::Timeout | 
            CaptchaError::ServerError | 
            CaptchaError::RateLimited |
            CaptchaError::Network(_)
        )
    }
}

#[derive(Debug)]
pub struct ApiResponse<T> {
    code: i32,
    message: String,
    data: Option<T>,
}

impl<T: serde::de::DeserializeOwned> ApiResponse<T> {
    pub fn from_json(json_str: &str) -> Result<Self, CaptchaError> {
        let response: serde_json::Value = serde_json::from_str(json_str)?;
        
        Ok(Self {
            code: response.get("code").and_then(|v| v.as_i64()).unwrap_or(-1) as i32,
            message: response.get("message").and_then(|v| v.as_str()).unwrap_or("").to_string(),
            data: if let Some(data_val) = response.get("data") {
                Some(serde_json::from_value(data_val.clone())?)
            } else {
                None
            },
        })
    }

    pub fn is_success(&self) -> bool {
        self.code == 0
    }

    pub fn code(&self) -> i32 {
        self.code
    }

    pub fn message(&self) -> &str {
        &self.message
    }

    pub fn data(self) -> Option<T> {
        self.data
    }

    pub fn into_data(self) -> Result<T, CaptchaError> {
        if !self.is_success() {
            let error = match self.code {
                401 | 403 => CaptchaError::Authentication(self.message),
                408 | 504 => CaptchaError::Timeout,
                429 => CaptchaError::RateLimited,
                404 => CaptchaError::SessionExpired,
                s if s >= 500 => CaptchaError::ServerError,
                _ => CaptchaError::ApiError(self.message, self.code),
            };
            return Err(error);
        }
        self.data.ok_or(CaptchaError::InvalidResponse)
    }
}
