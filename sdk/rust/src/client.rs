use crate::errors::{ApiResponse, CaptchaError};
use crate::models::*;
use hmac::{Hmac, Mac};
use reqwest::{Client, ClientBuilder, RequestBuilder};
use serde::Serialize;
use sha2::Sha256;
use std::collections::HashMap;
use std::time::Duration;

pub struct CaptchaClient {
    base_url: String,
    api_key: Option<String>,
    app_id: Option<String>,
    app_secret: Option<String>,
    timeout: Duration,
    max_retries: usize,
    retry_delay: Duration,
    http_client: Client,
    access_token: Option<String>,
}

impl CaptchaClient {
    pub fn new(base_url: &str) -> Self {
        let http_client = ClientBuilder::new()
            .timeout(Duration::from_secs(30))
            .build()
            .expect("Failed to create HTTP client");

        Self {
            base_url: base_url.to_string(),
            api_key: None,
            app_id: None,
            app_secret: None,
            timeout: Duration::from_secs(30),
            max_retries: 3,
            retry_delay: Duration::from_millis(100),
            http_client,
            access_token: None,
        }
    }

    pub fn with_api_key(mut self, api_key: &str) -> Self {
        self.api_key = Some(api_key.to_string());
        self
    }

    pub fn with_app_credentials(mut self, app_id: &str, app_secret: &str) -> Self {
        self.app_id = Some(app_id.to_string());
        self.app_secret = Some(app_secret.to_string());
        self
    }

    pub fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self.http_client = ClientBuilder::new()
            .timeout(timeout)
            .build()
            .expect("Failed to create HTTP client");
        self
    }

    pub fn with_retries(mut self, max_retries: usize, retry_delay: Duration) -> Self {
        self.max_retries = max_retries;
        self.retry_delay = retry_delay;
        self
    }

    pub fn base_url(&self) -> &str {
        &self.base_url
    }

    fn build_request(&self, method: reqwest::Method, path: &str) -> RequestBuilder {
        let url = format!("{}{}", self.base_url, path);
        let mut request = self.http_client.request(method, &url);
        
        request = request.header("User-Agent", "HJTPX-Captcha-Rust-SDK/1.0.0");
        
        if let Some(api_key) = &self.api_key {
            request = request.header("X-API-Key", api_key);
        }
        
        if let Some(token) = &self.access_token {
            request = request.header("Authorization", format!("Bearer {}", token));
        }
        
        if let (Some(app_id), Some(app_secret)) = (&self.app_id, &self.app_secret) {
            let timestamp = chrono::Utc::now().timestamp();
            let sign_str = format!("{}:{}:{}", app_id, timestamp, app_secret);
            let mut hmac = Hmac::<Sha256>::new_from_slice(app_secret.as_bytes()).unwrap();
            hmac.update(sign_str.as_bytes());
            let signature = hex::encode(hmac.finalize().into_bytes());
            
            request = request
                .header("X-App-ID", app_id)
                .header("X-Timestamp", timestamp.to_string())
                .header("X-Signature", signature);
        }
        
        request
    }

    async fn execute_request<T: serde::de::DeserializeOwned>(
        &self,
        path: &str,
        method: reqwest::Method,
        query_params: Option<&HashMap<&str, &str>>,
        body: Option<&impl Serialize>,
    ) -> Result<T, CaptchaError> {
        for attempt in 0..=self.max_retries {
            let mut request = self.build_request(method, path);
            
            if let Some(params) = query_params {
                request = request.query(params);
            }
            
            if let Some(data) = body {
                request = request.json(data);
            }
            
            let response = match request.send().await {
                Ok(r) => r,
                Err(e) => {
                    if attempt < self.max_retries && self.is_retryable_error(&e) {
                        tokio::time::sleep(self.calculate_retry_delay(attempt)).await;
                        continue;
                    }
                    return Err(CaptchaError::Network(e));
                }
            };
            
            let status = response.status();
            
            let content = match response.text().await {
                Ok(c) => c,
                Err(e) => {
                    if attempt < self.max_retries {
                        tokio::time::sleep(self.calculate_retry_delay(attempt)).await;
                        continue;
                    }
                    return Err(CaptchaError::Network(e));
                }
            };
            
            if !status.is_success() {
                let error = CaptchaError::from_status_code(status, &content);
                if attempt < self.max_retries && error.is_retryable() {
                    tokio::time::sleep(self.calculate_retry_delay(attempt)).await;
                    continue;
                }
                return Err(error);
            }
            
            let api_response = ApiResponse::from_json(&content)?;
            return api_response.into_data();
        }
        
        Err(CaptchaError::Timeout)
    }

    fn is_retryable_error(&self, err: &reqwest::Error) -> bool {
        err.is_timeout() || err.is_connect()
    }

    fn calculate_retry_delay(&self, attempt: usize) -> Duration {
        self.retry_delay * (2u32.pow(attempt as u32) as u32)
    }

    pub async fn get_slider_captcha(
        &self,
        width: Option<i32>,
        height: Option<i32>,
        tolerance: Option<i32>,
    ) -> Result<SliderCaptchaResponse, CaptchaError> {
        let mut params = HashMap::new();
        if let Some(w) = width {
            params.insert("width", &w.to_string());
        }
        if let Some(h) = height {
            params.insert("height", &h.to_string());
        }
        if let Some(t) = tolerance {
            params.insert("tolerance", &t.to_string());
        }

        self.execute_request(
            "/api/v1/captcha/slider",
            reqwest::Method::GET,
            Some(&params),
            None,
        )
        .await
    }

    pub async fn get_click_captcha(
        &self,
        mode: Option<&str>,
        max_points: Option<i32>,
        allow_shuffle: Option<bool>,
    ) -> Result<ClickCaptchaResponse, CaptchaError> {
        let mut params = HashMap::new();
        if let Some(m) = mode {
            params.insert("mode", m);
        }
        if let Some(p) = max_points {
            params.insert("points", &p.to_string());
        }
        if let Some(s) = allow_shuffle {
            params.insert("shuffle", &s.to_string().to_lowercase());
        }

        self.execute_request(
            "/api/v1/captcha/click",
            reqwest::Method::GET,
            Some(&params),
            None,
        )
        .await
    }

    pub async fn get_image_captcha(
        &self,
        r#type: Option<&str>,
        count: Option<i32>,
    ) -> Result<ImageCaptchaResponse, CaptchaError> {
        let mut params = HashMap::new();
        if let Some(t) = r#type {
            params.insert("type", t);
        }
        if let Some(c) = count {
            params.insert("count", &c.to_string());
        }

        self.execute_request(
            "/api/v1/captcha/image",
            reqwest::Method::GET,
            Some(&params),
            None,
        )
        .await
    }

    pub async fn get_rotation_captcha(&self) -> Result<RotationCaptchaResponse, CaptchaError> {
        self.execute_request(
            "/api/v1/captcha/rotation",
            reqwest::Method::GET,
            None,
            None,
        )
        .await
    }

    pub async fn get_gesture_captcha(&self) -> Result<GestureCaptchaResponse, CaptchaError> {
        self.execute_request(
            "/api/v1/captcha/gesture",
            reqwest::Method::GET,
            None,
            None,
        )
        .await
    }

    pub async fn get_jigsaw_captcha(
        &self,
        width: Option<i32>,
        height: Option<i32>,
        grid_size: Option<i32>,
    ) -> Result<JigsawCaptchaResponse, CaptchaError> {
        let mut params = HashMap::new();
        if let Some(w) = width {
            params.insert("width", &w.to_string());
        }
        if let Some(h) = height {
            params.insert("height", &h.to_string());
        }
        if let Some(g) = grid_size {
            params.insert("grid_size", &g.to_string());
        }

        self.execute_request(
            "/api/v1/captcha/jigsaw",
            reqwest::Method::GET,
            Some(&params),
            None,
        )
        .await
    }

    pub async fn get_voice_captcha(&self, language: Option<&str>) -> Result<VoiceCaptchaResponse, CaptchaError> {
        let mut params = HashMap::new();
        if let Some(l) = language {
            params.insert("language", l);
        }

        self.execute_request(
            "/api/v1/captcha/voice",
            reqwest::Method::GET,
            Some(&params),
            None,
        )
        .await
    }

    pub async fn verify_slider_captcha(
        &self,
        session_id: &str,
        x: i32,
        y: Option<i32>,
        trajectory: Option<&[TrajectoryPoint]>,
    ) -> Result<VerifyResult, CaptchaError> {
        let request = VerifyCaptchaRequest {
            session_id: session_id.to_string(),
            r#type: "slider".to_string(),
            x: Some(x),
            y,
            trajectory: trajectory.map(|t| t.to_vec()),
            ..Default::default()
        };

        self.execute_request(
            "/api/v1/captcha/verify",
            reqwest::Method::POST,
            None,
            Some(&request),
        )
        .await
    }

    pub async fn verify_click_captcha(
        &self,
        session_id: &str,
        points: &[Vec<i32>],
        click_sequence: Option<&[i32]>,
    ) -> Result<VerifyResult, CaptchaError> {
        let request = VerifyCaptchaRequest {
            session_id: session_id.to_string(),
            r#type: "click".to_string(),
            points: Some(points.to_vec()),
            click_sequence: click_sequence.map(|s| s.to_vec()),
            ..Default::default()
        };

        self.execute_request(
            "/api/v1/captcha/verify",
            reqwest::Method::POST,
            None,
            Some(&request),
        )
        .await
    }

    pub async fn verify_image_captcha(
        &self,
        challenge_id: &str,
        answer: &str,
    ) -> Result<VerifyResult, CaptchaError> {
        let request = VerifyCaptchaRequest {
            session_id: challenge_id.to_string(),
            r#type: "image".to_string(),
            answer: Some(answer.to_string()),
            ..Default::default()
        };

        self.execute_request(
            "/api/v1/captcha/image/verify",
            reqwest::Method::POST,
            None,
            Some(&request),
        )
        .await
    }

    pub async fn verify_rotation_captcha(
        &self,
        challenge_id: &str,
        angle: i32,
    ) -> Result<VerifyResult, CaptchaError> {
        let request = VerifyCaptchaRequest {
            session_id: challenge_id.to_string(),
            r#type: "rotation".to_string(),
            angle: Some(angle),
            ..Default::default()
        };

        self.execute_request(
            "/api/v1/captcha/rotation/verify",
            reqwest::Method::POST,
            None,
            Some(&request),
        )
        .await
    }

    pub async fn verify_gesture_captcha(
        &self,
        session_id: &str,
        pattern: &[i32],
    ) -> Result<VerifyResult, CaptchaError> {
        let request = VerifyCaptchaRequest {
            session_id: session_id.to_string(),
            r#type: "gesture".to_string(),
            pattern: Some(pattern.to_vec()),
            ..Default::default()
        };

        self.execute_request(
            "/api/v1/captcha/gesture/verify",
            reqwest::Method::POST,
            None,
            Some(&request),
        )
        .await
    }

    pub async fn verify_jigsaw_captcha(
        &self,
        session_id: &str,
        pieces: &[JigsawPiece],
    ) -> Result<VerifyResult, CaptchaError> {
        let request = VerifyCaptchaRequest {
            session_id: session_id.to_string(),
            r#type: "jigsaw".to_string(),
            pieces: Some(pieces.to_vec()),
            ..Default::default()
        };

        self.execute_request(
            "/api/v1/captcha/jigsaw/verify",
            reqwest::Method::POST,
            None,
            Some(&request),
        )
        .await
    }

    pub async fn login(
        &mut self,
        username: &str,
        password: &str,
        captcha_token: Option<&str>,
    ) -> Result<LoginResponse, CaptchaError> {
        let request = LoginRequest {
            username: username.to_string(),
            password: password.to_string(),
            captcha_token: captcha_token.map(|t| t.to_string()),
        };

        let response = self.execute_request(
            "/api/v1/auth/login",
            reqwest::Method::POST,
            None,
            Some(&request),
        )
        .await?;

        self.access_token = Some(response.access_token.clone());
        Ok(response)
    }

    pub async fn logout(&mut self) -> Result<(), CaptchaError> {
        match self.execute_request::<()>(
            "/api/v1/auth/logout",
            reqwest::Method::POST,
            None,
            None,
        ).await {
            Ok(_) => {
                self.access_token = None;
                Ok(())
            }
            Err(e) => {
                self.access_token = None;
                Err(e)
            }
        }
    }

    pub async fn register(
        &self,
        username: &str,
        email: &str,
        password: &str,
        behavior_data: Option<&str>,
    ) -> Result<UserInfo, CaptchaError> {
        let request = RegisterRequest {
            username: username.to_string(),
            email: email.to_string(),
            password: password.to_string(),
            behavior_data: behavior_data.map(|d| d.to_string()),
        };

        self.execute_request(
            "/api/v1/auth/register",
            reqwest::Method::POST,
            None,
            Some(&request),
        )
        .await
    }

    pub async fn refresh_token(&mut self, refresh_token: &str) -> Result<LoginResponse, CaptchaError> {
        let body = serde_json::json!({
            "refresh_token": refresh_token
        });

        let response = self.execute_request(
            "/api/v1/auth/refresh",
            reqwest::Method::POST,
            None,
            Some(&body),
        )
        .await?;

        self.access_token = Some(response.access_token.clone());
        Ok(response)
    }

    pub async fn get_detection_script(&self, callback: Option<&str>) -> Result<String, CaptchaError> {
        let mut params = HashMap::new();
        if let Some(c) = callback {
            params.insert("callback", c);
        }

        let response = self.http_client
            .get(format!("{}/api/v1/detect/script", self.base_url))
            .query(&params)
            .send()
            .await?;

        Ok(response.text().await?)
    }

    pub async fn submit_detection(
        &self,
        data: &EnvironmentData,
    ) -> Result<DetectionResult, CaptchaError> {
        self.execute_request(
            "/api/v1/detect/submit",
            reqwest::Method::POST,
            None,
            Some(data),
        )
        .await
    }

    pub async fn check_environment(
        &self,
        data: &EnvironmentData,
    ) -> Result<DetectionResult, CaptchaError> {
        self.execute_request(
            "/api/v1/detect/check",
            reqwest::Method::POST,
            None,
            Some(data),
        )
        .await
    }
}

impl Default for VerifyCaptchaRequest {
    fn default() -> Self {
        Self {
            session_id: String::new(),
            r#type: String::new(),
            x: None,
            y: None,
            angle: None,
            answer: None,
            pattern: None,
            points: None,
            pieces: None,
            connections: None,
            target_position: None,
            trajectory: None,
            click_sequence: None,
        }
    }
}
