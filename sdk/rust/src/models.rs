//! 数据模型模块
//!
//! 定义所有验证码相关的请求和响应数据结构。

use serde::{Deserialize, Serialize};

/// 轨迹点
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrajectoryPoint {
    /// X坐标
    pub x: i32,
    /// Y坐标
    pub y: i32,
    /// 时间戳（毫秒）
    pub t: i64,
}

impl TrajectoryPoint {
    /// 创建新的轨迹点
    pub fn new(x: i32, y: i32, t: i64) -> Self {
        Self { x, y, t }
    }
}

/// 滑块验证码响应
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SliderCaptchaResponse {
    /// 会话ID
    pub session_id: String,
    /// 背景图片URL
    pub image_url: String,
    /// 拼图URL
    pub puzzle_url: String,
    /// 提示URL
    pub hint_url: Option<String>,
    /// 形状类型
    pub shape: Option<i32>,
    /// 秘密Y坐标
    pub secret_y: Option<i32>,
    /// 图片宽度
    pub image_width: Option<i32>,
    /// 图片高度
    pub image_height: Option<i32>,
}

/// 点击验证码响应
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClickCaptchaResponse {
    /// 会话ID
    pub session_id: String,
    /// 图片URL
    pub image_url: String,
    /// 提示文本
    pub hint: String,
    /// 提示顺序
    pub hint_order: Vec<i32>,
    /// 最大点击点数
    pub max_points: i32,
    /// 模式
    pub mode: String,
    /// 是否允许打乱
    pub allow_shuffle: bool,
    /// 点击坐标点
    pub points: Option<Vec<Vec<i32>>>,
}

/// 图形验证码响应
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImageCaptchaResponse {
    /// 挑战ID
    pub challenge_id: String,
    /// 图片数据（base64）
    pub image: String,
}

/// 旋转验证码响应
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RotationCaptchaResponse {
    /// 挑战ID
    pub challenge_id: String,
    /// 图片数据（base64）
    pub image: String,
}

/// 手势验证码响应
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GestureCaptchaResponse {
    /// 会话ID
    pub session_id: String,
    /// 手势模式
    pub pattern: Option<String>,
    /// 网格大小
    pub grid_size: Option<i32>,
    /// 提示
    pub hint: Option<String>,
}

/// 拼图碎片
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JigsawPiece {
    /// 索引
    pub index: i32,
    /// 原始X坐标
    pub original_x: i32,
    /// 原始Y坐标
    pub original_y: i32,
    /// 当前X坐标
    pub current_x: i32,
    /// 当前Y坐标
    pub current_y: i32,
    /// 宽度
    pub width: i32,
    /// 高度
    pub height: i32,
    /// 旋转角度
    pub rotation: i32,
}

/// 拼图验证码响应
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JigsawCaptchaResponse {
    /// 会话ID
    pub session_id: String,
    /// 图片URL
    pub image_url: String,
    /// 碎片列表
    pub pieces: Vec<JigsawPiece>,
    /// 碎片图片列表
    pub piece_images: Vec<String>,
    /// 网格大小
    pub grid_size: i32,
    /// 碎片宽度
    pub piece_width: i32,
    /// 碎片高度
    pub piece_height: i32,
    /// 图片宽度
    pub image_width: i32,
    /// 图片高度
    pub image_height: i32,
}

/// 语音验证码响应
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VoiceCaptchaResponse {
    /// 会话ID
    pub session_id: String,
    /// 音频URL
    pub audio_url: String,
    /// 音频内容（base64）
    pub audio_data: Option<String>,
    /// 提示文本
    pub hint: String,
}

/// 3D验证码响应
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ThreeDCaptchaResponse {
    /// 会话ID
    pub session_id: String,
    /// 图片URL
    pub image_url: String,
    /// 旋转角度
    pub angle: i32,
    /// 提示
    pub hint: Option<String>,
}

/// 连连看验证码响应
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LianLianKanCaptchaResponse {
    /// 会话ID
    pub session_id: String,
    /// 图片URL
    pub image_url: String,
    /// 网格大小
    pub grid_size: i32,
    /// 连接路径
    pub paths: Option<Vec<Vec<Vec<i32>>>>,
}

/// 验证结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyResult {
    /// 是否成功
    pub success: bool,
    /// 消息
    pub message: String,
    /// 剩余尝试次数
    pub remaining_attempts: Option<i32>,
    /// 轨迹分析结果
    pub trajectory_result: Option<TrajectoryResult>,
    /// 风险分数
    pub risk_score: Option<f64>,
    /// 验证码是否通过
    pub captcha_pass: Option<bool>,
    /// 失败原因
    pub fail_reason: Option<String>,
}

/// 轨迹分析结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrajectoryResult {
    /// 分数
    pub score: f64,
    /// 是否通过
    pub passed: bool,
    /// 原因列表
    pub reasons: Option<Vec<String>>,
}

/// 登录请求
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoginRequest {
    /// 用户名
    pub username: String,
    /// 密码
    pub password: String,
    /// 验证码令牌
    pub captcha_token: Option<String>,
}

/// 用户信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    /// 用户ID
    pub id: u64,
    /// 用户名
    pub username: String,
    /// 邮箱
    pub email: String,
}

/// 登录响应
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoginResponse {
    /// 访问令牌
    pub access_token: String,
    /// 刷新令牌
    pub refresh_token: String,
    /// 过期时间（秒）
    pub expires_in: i64,
    /// 用户信息
    pub user: User,
}

/// API通用响应
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiResponse<T> {
    /// 状态码
    pub code: i32,
    /// 消息
    pub message: String,
    /// 数据
    pub data: Option<T>,
}

impl<T> ApiResponse<T> {
    /// 检查是否成功
    pub fn is_success(&self) -> bool {
        self.code == 0
    }
}
