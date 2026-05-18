use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SliderCaptchaResponse {
    pub session_id: String,
    pub image_url: String,
    pub puzzle_url: String,
    #[serde(default)]
    pub hint_url: Option<String>,
    #[serde(default)]
    pub shape: Option<i32>,
    #[serde(default)]
    pub secret_y: Option<i32>,
    #[serde(default)]
    pub image_width: Option<i32>,
    #[serde(default)]
    pub image_height: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrajectoryPoint {
    pub x: i32,
    pub y: i32,
    pub t: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyCaptchaRequest {
    pub session_id: String,
    pub x: i32,
    #[serde(default)]
    pub y: Option<i32>,
    #[serde(default)]
    pub trajectory: Option<Vec<TrajectoryPoint>>,
    #[serde(default)]
    pub r#type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyCaptchaResponse {
    pub success: bool,
    pub message: String,
    #[serde(default)]
    pub remaining_attempts: Option<i32>,
    #[serde(default)]
    pub trajectory_result: Option<TrajectoryResult>,
    #[serde(default)]
    pub risk_score: Option<f64>,
    #[serde(default)]
    pub captcha_pass: Option<bool>,
    #[serde(default)]
    pub fail_reason: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrajectoryResult {
    pub score: f64,
    pub passed: bool,
    #[serde(default)]
    pub reasons: Option<Vec<String>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClickCaptchaResponse {
    pub session_id: String,
    pub image_url: String,
    pub hint: String,
    pub hint_order: Vec<i32>,
    pub max_points: i32,
    pub mode: String,
    pub allow_shuffle: bool,
    #[serde(default)]
    pub points: Option<Vec<Vec<i32>>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImageCaptchaResponse {
    pub challenge_id: String,
    pub image: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RotationCaptchaResponse {
    pub challenge_id: String,
    pub image: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GestureCaptchaResponse {
    pub session_id: String,
    #[serde(default)]
    pub pattern: Option<String>,
    #[serde(default)]
    pub grid_size: Option<i32>,
    #[serde(default)]
    pub hint: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JigsawPiece {
    pub index: i32,
    pub original_x: i32,
    pub original_y: i32,
    pub current_x: i32,
    pub current_y: i32,
    pub width: i32,
    pub height: i32,
    #[serde(default)]
    pub rotation: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JigsawCaptchaResponse {
    pub session_id: String,
    pub image_url: String,
    pub pieces: Vec<JigsawPiece>,
    #[serde(default)]
    pub piece_images: Option<Vec<String>>,
    pub grid_size: i32,
    pub piece_width: i32,
    pub piece_height: i32,
    pub image_width: i32,
    pub image_height: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BatchVerifyRequest {
    pub requests: Vec<VerifyCaptchaRequest>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BatchVerifyResponse {
    pub results: Vec<VerifyResult>,
    pub success_count: i32,
    pub failed_count: i32,
    pub total_time_ms: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyResult {
    pub session_id: String,
    pub success: bool,
    pub message: String,
    #[serde(default)]
    pub remaining_attempts: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AsyncVerifyRequest {
    pub session_id: String,
    pub x: i32,
    #[serde(default)]
    pub y: Option<i32>,
    #[serde(default)]
    pub trajectory: Option<Vec<TrajectoryPoint>>,
    #[serde(default)]
    pub callback_url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AsyncVerifyResponse {
    pub task_id: String,
    pub status: String,
    #[serde(default)]
    pub result_url: Option<String>,
    pub created_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AsyncResultResponse {
    pub task_id: String,
    pub status: String,
    #[serde(default)]
    pub result: Option<VerifyCaptchaResponse>,
    #[serde(default)]
    pub error: Option<String>,
    #[serde(default)]
    pub completed_at: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoginRequest {
    pub username: String,
    pub password: String,
    #[serde(default)]
    pub captcha_token: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoginResponse {
    pub access_token: String,
    pub refresh_token: String,
    pub expires_in: i64,
    pub user: User,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    pub id: u32,
    pub username: String,
    pub email: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiResponse<T> {
    pub code: i32,
    pub message: String,
    #[serde(default)]
    pub data: Option<T>,
}
