use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Deserialize)]
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
    #[serde(default)]
    pub tolerance: Option<i32>,
}

#[derive(Debug, Deserialize)]
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

#[derive(Debug, Deserialize)]
pub struct ImageCaptchaResponse {
    pub challenge_id: String,
    pub image: String,
}

#[derive(Debug, Deserialize)]
pub struct RotationCaptchaResponse {
    pub challenge_id: String,
    pub image: String,
}

#[derive(Debug, Deserialize)]
pub struct GestureCaptchaResponse {
    pub session_id: String,
    #[serde(default)]
    pub pattern: Option<String>,
    #[serde(default)]
    pub grid_size: Option<i32>,
    #[serde(default)]
    pub hint: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct JigsawPiece {
    pub index: i32,
    pub original_x: i32,
    pub original_y: i32,
    pub current_x: i32,
    pub current_y: i32,
    pub width: i32,
    pub height: i32,
    #[serde(default)]
    pub rotation: i32,
}

#[derive(Debug, Deserialize)]
pub struct JigsawCaptchaResponse {
    pub session_id: String,
    pub image_url: String,
    pub pieces: Vec<JigsawPiece>,
    pub piece_images: Vec<String>,
    pub grid_size: i32,
    pub piece_width: i32,
    pub piece_height: i32,
    pub image_width: i32,
    pub image_height: i32,
}

#[derive(Debug, Deserialize)]
pub struct VoiceCaptchaResponse {
    pub session_id: String,
    pub audio_url: String,
    pub length: i32,
    #[serde(default)]
    pub hint: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ConnectCaptchaResponse {
    pub session_id: String,
    pub image_url: String,
    pub pairs: Vec<Vec<i32>>,
    pub lines: Vec<Vec<i32>>,
}

#[derive(Debug, Deserialize)]
pub struct ThreeDCaptchaResponse {
    pub session_id: String,
    pub model_url: String,
    pub target_position: Vec<f64>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct TrajectoryPoint {
    pub x: i32,
    pub y: i32,
    pub t: i64,
}

#[derive(Debug, Deserialize)]
pub struct VerifyResult {
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

#[derive(Debug, Deserialize)]
pub struct TrajectoryResult {
    pub score: f64,
    pub passed: bool,
    #[serde(default)]
    pub reasons: Option<Vec<String>>,
}

#[derive(Debug, Serialize)]
pub struct VerifyCaptchaRequest {
    pub session_id: String,
    pub r#type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub x: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub y: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub angle: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub answer: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub pattern: Option<Vec<i32>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub points: Option<Vec<Vec<i32>>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub pieces: Option<Vec<JigsawPiece>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub connections: Option<Vec<Vec<i32>>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_position: Option<Vec<f64>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub trajectory: Option<Vec<TrajectoryPoint>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub click_sequence: Option<Vec<i32>>,
}

#[derive(Debug, Deserialize)]
pub struct LoginResponse {
    pub access_token: String,
    pub refresh_token: String,
    pub expires_in: i64,
    pub user: UserInfo,
}

#[derive(Debug, Deserialize)]
pub struct UserInfo {
    pub id: u64,
    pub username: String,
    pub email: String,
}

#[derive(Debug, Serialize)]
pub struct LoginRequest {
    pub username: String,
    pub password: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub captcha_token: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct RegisterRequest {
    pub username: String,
    pub email: String,
    pub password: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub behavior_data: Option<String>,
}

pub type EnvironmentData = HashMap<String, serde_json::Value>;
pub type DetectionResult = HashMap<String, serde_json::Value>;

#[derive(Debug, Deserialize)]
pub struct BatchVerifyRequest {
    pub items: Vec<BatchVerifyItem>,
}

#[derive(Debug, Serialize)]
pub struct BatchVerifyItem {
    pub session_id: String,
    pub r#type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub x: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub y: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub points: Option<Vec<Vec<i32>>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub trajectory: Option<Vec<TrajectoryPoint>>,
}

#[derive(Debug, Deserialize)]
pub struct BatchVerifyResponse {
    pub results: Vec<BatchVerifyResultItem>,
}

#[derive(Debug, Deserialize)]
pub struct BatchVerifyResultItem {
    pub index: i32,
    pub success: bool,
    pub message: String,
    #[serde(default)]
    pub remaining_attempts: Option<i32>,
}
