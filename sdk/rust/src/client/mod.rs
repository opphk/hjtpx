use crate::error::{CaptchaError, Result};
use crate::models::*;
use crate::utils::{RateLimiter, RetryConfig, retry_with_backoff};
use reqwest::Client;
use serde::de::DeserializeOwned;
use serde_json::Value;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::RwLock;

pub struct CaptchaClient {
    base_url: String,
    http_client: Client,
    api_key: Option<String>,
    access_token: Option<String>,
    retry_config: RetryConfig,
    rate_limiter: Option<RateLimiter>,
}

impl CaptchaClient {
    pub fn new(base_url: impl Into<String>) -> Self {
        let http_client = Client::builder()
            .timeout(Duration::from_secs(30))
            .build()
            .expect("Failed to create HTTP client");

        Self {
            base_url: base_url.into().trim_end_matches('/').to_string(),
            http_client,
            api_key: None,
            access_token: None,
            retry_config: RetryConfig::default(),
            rate_limiter: None,
        }
    }

    pub fn with_api_key(mut self, api_key: impl Into<String>) -> Self {
        self.api_key = Some(api_key.into());
        self
    }

    pub fn with_timeout(mut self, timeout: Duration) -> Self {
        self.http_client = Client::builder()
            .timeout(timeout)
            .build()
            .expect("Failed to create HTTP client");
        self
    }

    pub fn with_retry_config(mut self, config: RetryConfig) -> Self {
        self.retry_config = config;
        self
    }

    pub fn with_rate_limit(mut self, requests_per_second: u32) -> Self {
        self.rate_limiter = Some(RateLimiter::new(requests_per_second));
        self
    }

    fn get_headers(&self) -> HashMap<String, String> {
        let mut headers = HashMap::new();
        headers.insert("Content-Type".to_string(), "application/json".to_string());
        headers.insert("User-Agent".to_string(), "Captcha-Rust-SDK/15.0".to_string());
        
        if let Some(ref api_key) = self.api_key {
            headers.insert("X-API-Key".to_string(), api_key.clone());
        }
        
        if let Some(ref token) = self.access_token {
            headers.insert("Authorization".to_string(), format!("Bearer {}", token));
        }
        
        headers
    }

    async fn request<T: DeserializeOwned>(
        &self,
        method: &str,
        path: &str,
        body: Option<Value>,
        params: Option<HashMap<String, String>>,
    ) -> Result<T> {
        let url = format!("{}{}", self.base_url, path);
        
        let mut request = self.http_client.request(
            reqwest::Method::from_bytes(method.as_bytes()).unwrap(),
            &url,
        );

        for (key, value) in self.get_headers() {
            request = request.header(&key, &value);
        }

        if let Some(params) = params {
            request = request.query(&params);
        }

        if let Some(body) = body {
            request = request.json(&body);
        }

        let mut attempts = self.retry_config.max_retries;
        
        loop {
            let response = self.http_client.execute(request.try_clone().unwrap()).await;
            
            match response {
                Ok(resp) => {
                    let status = resp.status();
                    
                    if status.is_success() {
                        let data: T = resp.json().await?;
                        return Ok(data);
                    }
                    
                    if status.as_u16() == 429 {
                        if attempts > 0 {
                            tokio::time::sleep(Duration::from_millis(1000)).await;
                            attempts -= 1;
                            continue;
                        }
                        return Err(CaptchaError::rate_limit_error("Rate limit exceeded"));
                    }
                    
                    if status.as_u16() == 401 {
                        return Err(CaptchaError::auth_error("Unauthorized"));
                    }
                    
                    if status.as_u16() == 400 {
                        return Err(CaptchaError::validation_error("Bad request"));
                    }
                    
                    return Err(CaptchaError::api_error(
                        format!("HTTP error: {}", status),
                        status.as_u16() as i32,
                    ));
                }
                Err(e) => {
                    if attempts > 0 && e.is_timeout() {
                        tokio::time::sleep(Duration::from_millis(100 * (self.retry_config.max_retries - attempts + 1) as u64)).await;
                        attempts -= 1;
                        continue;
                    }
                    return Err(CaptchaError::NetworkError(e));
                }
            }
        }
    }

    pub async fn get_slider_captcha(
        &self,
        width: i32,
        height: i32,
        tolerance: i32,
    ) -> Result<SliderCaptchaResponse> {
        let mut params = HashMap::new();
        params.insert("width".to_string(), width.to_string());
        params.insert("height".to_string(), height.to_string());
        params.insert("tolerance".to_string(), tolerance.to_string());

        let response: ApiResponse<Value> = self
            .request("GET", "/api/v1/captcha/slider", None, Some(params))
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        let data = response.data.ok_or_else(|| {
            CaptchaError::invalid_response("Missing data in response")
        })?;

        Ok(serde_json::from_value(data)?)
    }

    pub async fn verify_slider_captcha(
        &self,
        session_id: &str,
        x: i32,
        y: Option<i32>,
        trajectory: Option<Vec<TrajectoryPoint>>,
    ) -> Result<VerifyCaptchaResponse> {
        let mut body = serde_json::json!({
            "session_id": session_id,
            "type": "slider",
            "x": x,
        });

        if let Some(y) = y {
            body["y"] = serde_json::json!(y);
        }

        if let Some(traj) = trajectory {
            body["trajectory"] = serde_json::to_value(traj)?;
        }

        let response: ApiResponse<VerifyCaptchaResponse> = self
            .request("POST", "/api/v1/captcha/verify", Some(body), None)
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        response.data.ok_or_else(|| CaptchaError::invalid_response("Missing data"))
    }

    pub async fn get_click_captcha(
        &self,
        mode: &str,
        max_points: i32,
        allow_shuffle: bool,
    ) -> Result<ClickCaptchaResponse> {
        let mut params = HashMap::new();
        params.insert("mode".to_string(), mode.to_string());
        params.insert("points".to_string(), max_points.to_string());
        params.insert("shuffle".to_string(), allow_shuffle.to_string());

        let response: ApiResponse<Value> = self
            .request("GET", "/api/v1/captcha/click", None, Some(params))
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        let data = response.data.ok_or_else(|| {
            CaptchaError::invalid_response("Missing data in response")
        })?;

        Ok(serde_json::from_value(data)?)
    }

    pub async fn verify_click_captcha(
        &self,
        session_id: &str,
        points: Vec<Vec<i32>>,
        click_sequence: Option<Vec<i32>>,
    ) -> Result<VerifyCaptchaResponse> {
        let mut body = serde_json::json!({
            "session_id": session_id,
            "type": "click",
            "points": points,
        });

        if let Some(seq) = click_sequence {
            body["click_sequence"] = serde_json::to_value(seq)?;
        }

        let response: ApiResponse<VerifyCaptchaResponse> = self
            .request("POST", "/api/v1/captcha/verify", Some(body), None)
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        response.data.ok_or_else(|| CaptchaError::invalid_response("Missing data"))
    }

    pub async fn get_image_captcha(
        &self,
        type_: &str,
        count: i32,
    ) -> Result<ImageCaptchaResponse> {
        let mut params = HashMap::new();
        params.insert("type".to_string(), type_.to_string());
        params.insert("count".to_string(), count.to_string());

        let response: ApiResponse<ImageCaptchaResponse> = self
            .request("GET", "/api/v1/captcha/image", None, Some(params))
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        response.data.ok_or_else(|| CaptchaError::invalid_response("Missing data"))
    }

    pub async fn verify_image_captcha(
        &self,
        challenge_id: &str,
        answer: &str,
    ) -> Result<VerifyCaptchaResponse> {
        let body = serde_json::json!({
            "challenge_id": challenge_id,
            "answer": answer,
        });

        let response: ApiResponse<VerifyCaptchaResponse> = self
            .request("POST", "/api/v1/captcha/image/verify", Some(body), None)
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        response.data.ok_or_else(|| CaptchaError::invalid_response("Missing data"))
    }

    pub async fn get_rotation_captcha(&self) -> Result<RotationCaptchaResponse> {
        let response: ApiResponse<RotationCaptchaResponse> = self
            .request("GET", "/api/v1/captcha/rotation", None, None)
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        response.data.ok_or_else(|| CaptchaError::invalid_response("Missing data"))
    }

    pub async fn verify_rotation_captcha(
        &self,
        challenge_id: &str,
        angle: i32,
    ) -> Result<VerifyCaptchaResponse> {
        let body = serde_json::json!({
            "challenge_id": challenge_id,
            "angle": angle,
        });

        let response: ApiResponse<VerifyCaptchaResponse> = self
            .request("POST", "/api/v1/captcha/rotation/verify", Some(body), None)
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        response.data.ok_or_else(|| CaptchaError::invalid_response("Missing data"))
    }

    pub async fn get_gesture_captcha(&self) -> Result<GestureCaptchaResponse> {
        let response: ApiResponse<Value> = self
            .request("GET", "/api/v1/captcha/gesture", None, None)
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        let data = response.data.ok_or_else(|| {
            CaptchaError::invalid_response("Missing data in response")
        })?;

        Ok(serde_json::from_value(data)?)
    }

    pub async fn verify_gesture_captcha(
        &self,
        session_id: &str,
        pattern: Vec<i32>,
    ) -> Result<VerifyCaptchaResponse> {
        let body = serde_json::json!({
            "session_id": session_id,
            "pattern": pattern,
        });

        let response: ApiResponse<VerifyCaptchaResponse> = self
            .request("POST", "/api/v1/captcha/gesture/verify", Some(body), None)
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        response.data.ok_or_else(|| CaptchaError::invalid_response("Missing data"))
    }

    pub async fn get_jigsaw_captcha(
        &self,
        width: i32,
        height: i32,
        grid_size: i32,
    ) -> Result<JigsawCaptchaResponse> {
        let mut params = HashMap::new();
        params.insert("width".to_string(), width.to_string());
        params.insert("height".to_string(), height.to_string());
        params.insert("grid_size".to_string(), grid_size.to_string());

        let response: ApiResponse<Value> = self
            .request("GET", "/api/v1/captcha/jigsaw", None, Some(params))
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        let data = response.data.ok_or_else(|| {
            CaptchaError::invalid_response("Missing data in response")
        })?;

        Ok(serde_json::from_value(data)?)
    }

    pub async fn verify_jigsaw_captcha(
        &self,
        session_id: &str,
        pieces: Vec<JigsawPiece>,
    ) -> Result<VerifyCaptchaResponse> {
        let body = serde_json::json!({
            "session_id": session_id,
            "pieces": pieces,
        });

        let response: ApiResponse<VerifyCaptchaResponse> = self
            .request("POST", "/api/v1/captcha/jigsaw/verify", Some(body), None)
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        response.data.ok_or_else(|| CaptchaError::invalid_response("Missing data"))
    }

    pub async fn batch_verify(
        &self,
        requests: Vec<VerifyCaptchaRequest>,
    ) -> Result<BatchVerifyResponse> {
        if requests.is_empty() {
            return Ok(BatchVerifyResponse {
                results: vec![],
                success_count: 0,
                failed_count: 0,
                total_time_ms: 0,
            });
        }

        let start_time = std::time::Instant::now();
        let mut results = Vec::new();
        let mut success_count = 0;
        let mut failed_count = 0;

        for req in requests {
            match self.verify_slider_captcha(
                &req.session_id,
                req.x,
                req.y,
                req.trajectory,
            ).await {
                Ok(resp) => {
                    results.push(VerifyResult {
                        session_id: req.session_id,
                        success: resp.success,
                        message: resp.message,
                        remaining_attempts: resp.remaining_attempts,
                    });
                    if resp.success {
                        success_count += 1;
                    } else {
                        failed_count += 1;
                    }
                }
                Err(e) => {
                    results.push(VerifyResult {
                        session_id: req.session_id,
                        success: false,
                        message: e.to_string(),
                        remaining_attempts: None,
                    });
                    failed_count += 1;
                }
            }
        }

        let total_time_ms = start_time.elapsed().as_millis() as i64;

        Ok(BatchVerifyResponse {
            results,
            success_count,
            failed_count,
            total_time_ms,
        })
    }

    pub async fn async_verify(
        &self,
        session_id: &str,
        x: i32,
        y: Option<i32>,
        trajectory: Option<Vec<TrajectoryPoint>>,
        callback_url: Option<String>,
    ) -> Result<AsyncVerifyResponse> {
        let mut body = serde_json::json!({
            "session_id": session_id,
            "x": x,
        });

        if let Some(y) = y {
            body["y"] = serde_json::json!(y);
        }

        if let Some(traj) = trajectory {
            body["trajectory"] = serde_json::to_value(traj)?;
        }

        if let Some(url) = callback_url {
            body["callback_url"] = serde_json::json!(url);
        }

        let response: ApiResponse<AsyncVerifyResponse> = self
            .request("POST", "/api/v1/captcha/async/verify", Some(body), None)
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        response.data.ok_or_else(|| CaptchaError::invalid_response("Missing data"))
    }

    pub async fn get_async_result(&self, task_id: &str) -> Result<AsyncResultResponse> {
        let response: ApiResponse<AsyncResultResponse> = self
            .request("GET", &format!("/api/v1/captcha/async/result/{}", task_id), None, None)
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        response.data.ok_or_else(|| CaptchaError::invalid_response("Missing data"))
    }

    pub async fn wait_async_result(
        &self,
        task_id: &str,
        timeout_secs: u64,
        poll_interval_ms: u64,
    ) -> Result<AsyncResultResponse> {
        let start_time = std::time::Instant::now();
        let timeout_duration = Duration::from_secs(timeout_secs);
        let poll_interval = Duration::from_millis(poll_interval_ms);

        loop {
            if start_time.elapsed() > timeout_duration {
                return Err(CaptchaError::timeout_error("Timeout waiting for async result"));
            }

            let result = self.get_async_result(task_id).await?;

            if result.status == "completed" || result.status == "failed" {
                return Ok(result);
            }

            tokio::time::sleep(poll_interval).await;
        }
    }

    pub async fn login(&self, username: &str, password: &str, captcha_token: Option<String>) -> Result<LoginResponse> {
        let mut body = serde_json::json!({
            "username": username,
            "password": password,
        });

        if let Some(token) = captcha_token {
            body["captcha_token"] = serde_json::json!(token);
        }

        let response: ApiResponse<LoginResponse> = self
            .request("POST", "/api/v1/auth/login", Some(body), None)
            .await?;

        if response.code != 0 {
            return Err(CaptchaError::api_error(&response.message, response.code));
        }

        let data = response.data.ok_or_else(|| CaptchaError::invalid_response("Missing data"))?;
        
        Ok(data)
    }

    pub async fn logout(&self) -> Result<()> {
        let _: ApiResponse<Value> = self
            .request("POST", "/api/v1/auth/logout", None, None)
            .await?;
        
        Ok(())
    }

    pub async fn get_detection_script(&self, callback: Option<&str>) -> Result<String> {
        let mut params = HashMap::new();
        if let Some(cb) = callback {
            params.insert("callback".to_string(), cb.to_string());
        }

        let url = format!("{}/api/v1/detect/script", self.base_url);
        
        let mut request = self.http_client.get(&url);
        
        for (key, value) in self.get_headers() {
            request = request.header(&key, &value);
        }

        if !params.is_empty() {
            request = request.query(&params);
        }

        let response = request.send().await?;
        let script = response.text().await?;

        Ok(script)
    }
}

impl CaptchaError {
    fn invalid_response(msg: &str) -> Self {
        Self::InvalidResponse(msg.to_string())
    }
}
