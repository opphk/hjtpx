use crate::error::{CaptchaError, Result};
use reqwest::Client;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::RwLock;

pub struct RateLimiter {
    requests_per_second: f64,
    interval_ns: u64,
    last_request_ns: u64,
}

impl RateLimiter {
    pub fn new(requests_per_second: u32) -> Self {
        Self {
            requests_per_second: requests_per_second as f64,
            interval_ns: (1_000_000_000.0 / requests_per_second as f64) as u64,
            last_request_ns: 0,
        }
    }

    pub async fn acquire(&mut self) {
        let now = std::time::Instant::now();
        let elapsed_ns = now.elapsed().as_nanos() as u64;
        
        let time_since_last = elapsed_ns.saturating_sub(self.last_request_ns);
        if time_since_last < self.interval_ns {
            let sleep_ns = self.interval_ns - time_since_last;
            let sleep_duration = Duration::from_nanos(sleep_ns);
            tokio::time::sleep(sleep_duration).await;
        }
        
        self.last_request_ns = now.elapsed().as_nanos() as u64;
    }
}

pub struct RetryConfig {
    pub max_retries: u32,
    pub base_delay_ms: u64,
    pub max_delay_ms: u64,
}

impl Default for RetryConfig {
    fn default() -> Self {
        Self {
            max_retries: 3,
            base_delay_ms: 100,
            max_delay_ms: 5000,
        }
    }
}

pub fn calculate_retry_delay(attempt: u32, config: &RetryConfig) -> Duration {
    let delay = config.base_delay_ms * 2_u64.pow(attempt);
    let delay = delay.min(config.max_delay_ms);
    Duration::from_millis(delay)
}

pub async fn retry_with_backoff<F, T, Fut>(
    mut attempts: u32,
    config: &RetryConfig,
    mut f: F,
) -> Result<T>
where
    F: FnMut() -> Fut,
    Fut: std::future::Future<Output = Result<T>>,
{
    let mut last_error = None;
    
    while attempts > 0 {
        match f().await {
            Ok(result) => return Ok(result),
            Err(e) => {
                if !e.is_retryable() || attempts == 1 {
                    return Err(e);
                }
                last_error = Some(e);
                let delay = calculate_retry_delay(config.max_retries - attempts, config);
                tokio::time::sleep(delay).await;
            }
        }
        attempts -= 1;
    }
    
    last_error.unwrap_or_else(|| CaptchaError::api_error("Max retries exceeded", 0))
}
