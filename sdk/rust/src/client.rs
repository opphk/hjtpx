//! 验证码客户端模块
//!
//! 提供完整的验证码操作接口。

use crate::config::CaptchaConfig;
use crate::errors::{CaptchaError, CaptchaResult};
use crate::models::*;
use async_trait::async_trait;
use reqwest::Client;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::RwLock;

/// 验证码客户端
#[derive(Clone)]
pub struct CaptchaClient {
    /// 基础URL
    base_url: String,
    /// HTTP客户端
    client: Client,
    /// 配置
    config: CaptchaConfig,
    /// 访问令牌
    token: Arc<RwLock<Option<String>>>,
}

impl CaptchaClient {
    /// 创建新的验证码客户端
    pub fn new(base_url: impl Into<String>, config: CaptchaConfig) -> Self {
        let client = Client::builder()
            .timeout(config.timeout)
            .pool_max_idle_per_host(config.pool_size as usize)
            .build()
            .expect("Failed to create HTTP client");

        Self {
            base_url: base_url.into().trim_end_matches('/').to_string(),
            client,
            config,
            token: Arc::new(RwLock::new(None)),
        }
    }

    /// 创建带默认配置的客户端
    pub fn with_url(base_url: impl Into<String>) -> Self {
        Self::new(base_url, CaptchaConfig::default())
    }

    /// 获取请求头
    async fn get_headers(&self) -> reqwest::header::HeaderMap {
        let mut headers = reqwest::header::HeaderMap::new();
        headers.insert(
            reqwest::header::CONTENT_TYPE,
            "application/json".parse().unwrap(),
        );
        headers.insert(
            reqwest::header::USER_AGENT,
            "HJTPX-Rust-SDK/1.0".parse().unwrap(),
        );

        if let Some(ref api_key) = self.config.api_key {
            headers.insert(
                "X-API-Key",
                api_key.parse().unwrap(),
            );
        }

        if let Some(token) = self.token.read().await.as_ref() {
            headers.insert(
                reqwest::header::AUTHORIZATION,
                format!("Bearer {}", token).parse().unwrap(),
            );
        }

        headers
    }

    /// 发送请求
    async fn request<T: for<'de> Deserialize<'de>>(
        &self,
        method: reqwest::Method,
        path: &str,
        body: Option<serde_json::Value>,
        params: Option<Vec<(&str, &str)>>,
    ) -> CaptchaResult<T> {
        let url = format!("{}{}", self.base_url, path);
        let headers = self.get_headers().await;

        let mut request = self.client.request(method, &url).headers(headers);

        if let Some(p) = params {
            request = request.query(&p);
        }

        if let Some(b) = body {
            request = request.json(&b);
        }

        let mut retries = 0;
        let max_retries = self.config.max_retries;

        loop {
            match request.send().await {
                Ok(response) => {
                    let status = response.status();
                    let body = response.text().await?;

                    if status.is_success() {
                        let parsed: ApiResponse<T> = serde_json::from_str(&body)?;

                        if parsed.code == 0 {
                            parsed.data.ok_or_else(|| {
                                CaptchaError::Unknown("Empty response data".to_string())
                            })
                        } else {
                            Err(CaptchaError::ApiError {
                                code: parsed.code,
                                message: parsed.message,
                            })
                        }
                    } else {
                        let message = if body.is_empty() {
                            status.canonical_reason().unwrap_or("Unknown").to_string()
                        } else {
                            body.clone()
                        };

                        if status.as_u16() == 404 {
                            return Err(CaptchaError::SessionExpired);
                        }

                        Err(CaptchaError::HttpError {
                            status: status.as_u16(),
                            message,
                        })
                    }
                }
                Err(e) => {
                    if retries >= max_retries {
                        return Err(CaptchaError::from(e));
                    }

                    retries += 1;
                    let delay = Duration::from_secs_f64(
                        self.config.retry_base_delay * 2_f64.powi(retries as i32 - 1),
                    );
                    tokio::time::sleep(delay).await;
                    continue;
                }
            }
        }
    }

    // ==================== 滑块验证码 ====================

    /// 获取滑块验证码
    pub async fn get_slider_captcha(
        &self,
        width: i32,
        height: i32,
        tolerance: i32,
    ) -> CaptchaResult<SliderCaptchaResponse> {
        let response: ApiResponse<SliderCaptchaResponse> = self
            .request(
                reqwest::Method::GET,
                "/api/v1/captcha/slider",
                None,
                Some(vec![
                    ("width", &width.to_string()),
                    ("height", &height.to_string()),
                    ("tolerance", &tolerance.to_string()),
                ]),
            )
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    /// 验证滑块验证码
    pub async fn verify_slider_captcha(
        &self,
        session_id: &str,
        x: i32,
        y: Option<i32>,
        trajectory: Option<Vec<TrajectoryPoint>>,
    ) -> CaptchaResult<VerifyResult> {
        let mut body = serde_json::json!({
            "session_id": session_id,
            "type": "slider",
            "x": x,
        });

        if let Some(y_val) = y {
            body["y"] = serde_json::json!(y_val);
        }

        if let Some(traj) = trajectory {
            body["trajectory"] = serde_json::to_value(traj)?;
        }

        let response: ApiResponse<VerifyResult> = self
            .request(reqwest::Method::POST, "/api/v1/captcha/verify", Some(body), None)
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    // ==================== 点击验证码 ====================

    /// 获取点击验证码
    pub async fn get_click_captcha(
        &self,
        mode: &str,
        max_points: i32,
        allow_shuffle: bool,
    ) -> CaptchaResult<ClickCaptchaResponse> {
        let response: ApiResponse<ClickCaptchaResponse> = self
            .request(
                reqwest::Method::GET,
                "/api/v1/captcha/click",
                None,
                Some(vec![
                    ("mode", mode),
                    ("points", &max_points.to_string()),
                    ("shuffle", &allow_shuffle.to_string()),
                ]),
            )
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    /// 验证点击验证码
    pub async fn verify_click_captcha(
        &self,
        session_id: &str,
        points: Vec<Vec<i32>>,
        click_sequence: Option<Vec<i32>>,
    ) -> CaptchaResult<VerifyResult> {
        let mut body = serde_json::json!({
            "session_id": session_id,
            "type": "click",
            "points": points,
        });

        if let Some(seq) = click_sequence {
            body["click_sequence"] = serde_json::json!(seq);
        }

        let response: ApiResponse<VerifyResult> = self
            .request(reqwest::Method::POST, "/api/v1/captcha/verify", Some(body), None)
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    // ==================== 图形验证码 ====================

    /// 获取图形验证码
    pub async fn get_image_captcha(
        &self,
        type_: &str,
        count: i32,
    ) -> CaptchaResult<ImageCaptchaResponse> {
        let response: ApiResponse<ImageCaptchaResponse> = self
            .request(
                reqwest::Method::GET,
                "/api/v1/captcha/image",
                None,
                Some(vec![
                    ("type", type_),
                    ("count", &count.to_string()),
                ]),
            )
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    /// 验证图形验证码
    pub async fn verify_image_captcha(
        &self,
        challenge_id: &str,
        answer: &str,
    ) -> CaptchaResult<VerifyResult> {
        let body = serde_json::json!({
            "challenge_id": challenge_id,
            "answer": answer,
        });

        let response: ApiResponse<VerifyResult> = self
            .request(
                reqwest::Method::POST,
                "/api/v1/captcha/image/verify",
                Some(body),
                None,
            )
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    // ==================== 旋转验证码 ====================

    /// 获取旋转验证码
    pub async fn get_rotation_captcha(&self) -> CaptchaResult<RotationCaptchaResponse> {
        let response: ApiResponse<RotationCaptchaResponse> = self
            .request(reqwest::Method::GET, "/api/v1/captcha/rotation", None, None)
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    /// 验证旋转验证码
    pub async fn verify_rotation_captcha(
        &self,
        challenge_id: &str,
        angle: i32,
    ) -> CaptchaResult<VerifyResult> {
        let body = serde_json::json!({
            "challenge_id": challenge_id,
            "angle": angle,
        });

        let response: ApiResponse<VerifyResult> = self
            .request(
                reqwest::Method::POST,
                "/api/v1/captcha/rotation/verify",
                Some(body),
                None,
            )
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    // ==================== 手势验证码 ====================

    /// 获取手势验证码
    pub async fn get_gesture_captcha(&self) -> CaptchaResult<GestureCaptchaResponse> {
        let response: ApiResponse<GestureCaptchaResponse> = self
            .request(reqwest::Method::GET, "/api/v1/captcha/gesture", None, None)
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    /// 验证手势验证码
    pub async fn verify_gesture_captcha(
        &self,
        session_id: &str,
        pattern: Vec<i32>,
    ) -> CaptchaResult<VerifyResult> {
        let body = serde_json::json!({
            "session_id": session_id,
            "pattern": pattern,
        });

        let response: ApiResponse<VerifyResult> = self
            .request(
                reqwest::Method::POST,
                "/api/v1/captcha/gesture/verify",
                Some(body),
                None,
            )
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    // ==================== 拼图验证码 ====================

    /// 获取拼图验证码
    pub async fn get_jigsaw_captcha(
        &self,
        width: i32,
        height: i32,
        grid_size: i32,
    ) -> CaptchaResult<JigsawCaptchaResponse> {
        let response: ApiResponse<JigsawCaptchaResponse> = self
            .request(
                reqwest::Method::GET,
                "/api/v1/captcha/jigsaw",
                None,
                Some(vec![
                    ("width", &width.to_string()),
                    ("height", &height.to_string()),
                    ("grid_size", &grid_size.to_string()),
                ]),
            )
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    /// 验证拼图验证码
    pub async fn verify_jigsaw_captcha(
        &self,
        session_id: &str,
        pieces: Vec<JigsawPiece>,
    ) -> CaptchaResult<VerifyResult> {
        let body = serde_json::json!({
            "session_id": session_id,
            "pieces": pieces,
        });

        let response: ApiResponse<VerifyResult> = self
            .request(
                reqwest::Method::POST,
                "/api/v1/captcha/jigsaw/verify",
                Some(body),
                None,
            )
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    // ==================== 语音验证码 ====================

    /// 获取语音验证码
    pub async fn get_voice_captcha(&self) -> CaptchaResult<VoiceCaptchaResponse> {
        let response: ApiResponse<VoiceCaptchaResponse> = self
            .request(reqwest::Method::GET, "/api/v1/captcha/voice", None, None)
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    /// 验证语音验证码
    pub async fn verify_voice_captcha(
        &self,
        session_id: &str,
        answer: &str,
    ) -> CaptchaResult<VerifyResult> {
        let body = serde_json::json!({
            "session_id": session_id,
            "answer": answer,
        });

        let response: ApiResponse<VerifyResult> = self
            .request(
                reqwest::Method::POST,
                "/api/v1/captcha/voice/verify",
                Some(body),
                None,
            )
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    // ==================== 3D验证码 ====================

    /// 获取3D验证码
    pub async fn get_3d_captcha(&self) -> CaptchaResult<ThreeDCaptchaResponse> {
        let response: ApiResponse<ThreeDCaptchaResponse> = self
            .request(reqwest::Method::GET, "/api/v1/captcha/3d", None, None)
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    /// 验证3D验证码
    pub async fn verify_3d_captcha(
        &self,
        session_id: &str,
        angle: i32,
    ) -> CaptchaResult<VerifyResult> {
        let body = serde_json::json!({
            "session_id": session_id,
            "angle": angle,
        });

        let response: ApiResponse<VerifyResult> = self
            .request(
                reqwest::Method::POST,
                "/api/v1/captcha/3d/verify",
                Some(body),
                None,
            )
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    // ==================== 连连看验证码 ====================

    /// 获取连连看验证码
    pub async fn get_lianliankan_captcha(
        &self,
        grid_size: i32,
    ) -> CaptchaResult<LianLianKanCaptchaResponse> {
        let response: ApiResponse<LianLianKanCaptchaResponse> = self
            .request(
                reqwest::Method::GET,
                "/api/v1/captcha/lianliankan",
                None,
                Some(vec![("grid_size", &grid_size.to_string())]),
            )
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    /// 验证连连看验证码
    pub async fn verify_lianliankan_captcha(
        &self,
        session_id: &str,
        path: Vec<Vec<i32>>,
    ) -> CaptchaResult<VerifyResult> {
        let body = serde_json::json!({
            "session_id": session_id,
            "path": path,
        });

        let response: ApiResponse<VerifyResult> = self
            .request(
                reqwest::Method::POST,
                "/api/v1/captcha/lianliankan/verify",
                Some(body),
                None,
            )
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }

    // ==================== 用户认证 ====================

    /// 用户登录
    pub async fn login(&self, req: &LoginRequest) -> CaptchaResult<LoginResponse> {
        let body = serde_json::to_value(req)?;

        let response: ApiResponse<LoginResponse> = self
            .request(reqwest::Method::POST, "/api/v1/auth/login", Some(body), None)
            .await?;

        if let Some(data) = response.data {
            *self.token.write().await = Some(data.access_token.clone());
            Ok(data)
        } else {
            Err(CaptchaError::Unknown("Empty response".to_string()))
        }
    }

    /// 用户登出
    pub async fn logout(&self) -> CaptchaResult<()> {
        let _: ApiResponse<serde_json::Value> = self
            .request(reqwest::Method::POST, "/api/v1/auth/logout", None, None)
            .await?;

        *self.token.write().await = None;
        Ok(())
    }

    /// 刷新令牌
    pub async fn refresh_token(&self, refresh_token: &str) -> CaptchaResult<LoginResponse> {
        let body = serde_json::json!({
            "refresh_token": refresh_token,
        });

        let response: ApiResponse<LoginResponse> = self
            .request(reqwest::Method::POST, "/api/v1/auth/refresh", Some(body), None)
            .await?;

        if let Some(data) = response.data {
            *self.token.write().await = Some(data.access_token.clone());
            Ok(data)
        } else {
            Err(CaptchaError::Unknown("Empty response".to_string()))
        }
    }

    // ==================== 通用验证 ====================

    /// 通用验证方法
    pub async fn verify(
        &self,
        captcha_type: &str,
        session_id: &str,
        params: serde_json::Value,
    ) -> CaptchaResult<VerifyResult> {
        let mut body = serde_json::json!({
            "session_id": session_id,
            "type": captcha_type,
        });

        if let serde_json::Value::Object(mut p) = params {
            body.as_object_mut().unwrap().extend(p);
        }

        let response: ApiResponse<VerifyResult> = self
            .request(reqwest::Method::POST, "/api/v1/captcha/verify", Some(body), None)
            .await?;

        response.data.ok_or_else(|| CaptchaError::Unknown("Empty response".to_string()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_trajectory_point_creation() {
        let point = TrajectoryPoint::new(100, 200, 1234567890);
        assert_eq!(point.x, 100);
        assert_eq!(point.y, 200);
        assert_eq!(point.t, 1234567890);
    }

    #[tokio::test]
    async fn test_client_creation() {
        let client = CaptchaClient::with_url("http://localhost:8080");
        assert_eq!(client.base_url, "http://localhost:8080");
    }

    #[tokio::test]
    async fn test_config_with_api_key() {
        let config = CaptchaConfig::new()
            .with_api_key("test-key")
            .with_timeout(Duration::from_secs(60))
            .with_max_retries(5);

        assert_eq!(config.api_key, Some("test-key".to_string()));
        assert_eq!(config.timeout, Duration::from_secs(60));
        assert_eq!(config.max_retries, 5);
    }
}
