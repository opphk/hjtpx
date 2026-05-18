use thiserror::Error;

#[derive(Error, Debug)]
pub enum CaptchaError {
    #[error("API error: {0} (code: {code})")]
    ApiError { message: String, code: i32 },
    
    #[error("Network error: {0}")]
    NetworkError(#[from] reqwest::Error),
    
    #[error("Timeout error: {0}")]
    TimeoutError(String),
    
    #[error("Validation error: {0}")]
    ValidationError(String),
    
    #[error("Rate limit exceeded: {0}")]
    RateLimitError(String),
    
    #[error("Authentication error: {0}")]
    AuthError(String),
    
    #[error("Invalid response: {0}")]
    InvalidResponse(String),
    
    #[error("JSON parse error: {0}")]
    JsonError(#[from] serde_json::Error),
    
    #[error("Configuration error: {0}")]
    ConfigError(String),
}

impl CaptchaError {
    pub fn api_error(message: impl Into<String>, code: i32) -> Self {
        Self::ApiError {
            message: message.into(),
            code,
        }
    }

    pub fn validation_error(message: impl Into<String>) -> Self {
        Self::ValidationError(message.into())
    }

    pub fn timeout_error(message: impl Into<String>) -> Self {
        Self::TimeoutError(message.into())
    }

    pub fn rate_limit_error(message: impl Into<String>) -> Self {
        Self::RateLimitError(message.into())
    }

    pub fn auth_error(message: impl Into<String>) -> Self {
        Self::AuthError(message.into())
    }

    pub fn is_retryable(&self) -> bool {
        matches!(
            self,
            Self::NetworkError(_) | Self::TimeoutError(_) | Self::RateLimitError(_)
        )
    }
}

pub type Result<T> = std::result::Result<T, CaptchaError>;
