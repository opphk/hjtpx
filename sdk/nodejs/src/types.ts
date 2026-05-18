/**
 * 验证码客户端配置
 */
export interface CaptchaClientConfig {
  /** API基础URL */
  baseUrl: string;
  /** API密钥（可选） */
  apiKey?: string;
  /** 请求超时时间（毫秒），默认30000 */
  timeout?: number;
  /** 最大并发连接数，默认100 */
  maxConnections?: number;
  /** 重试配置（可选） */
  retryConfig?: RetryConfig;
}

/**
 * 重试配置
 */
export interface RetryConfig {
  /** 最大重试次数，默认3 */
  maxRetries?: number;
  /** 初始重试延迟（毫秒），默认100 */
  initialDelayMs?: number;
  /** 最大延迟（毫秒），默认5000 */
  maxDelayMs?: number;
  /** 可重试的HTTP状态码 */
  retryableStatuses?: number[];
}

/**
 * 轨迹点
 */
export interface TrajectoryPoint {
  /** X坐标 */
  x: number;
  /** Y坐标 */
  y: number;
  /** 时间戳（毫秒） */
  t: number;
}

/**
 * 滑块验证码响应
 */
export interface SliderCaptchaResponse {
  /** 会话ID */
  session_id: string;
  /** 背景图片URL */
  image_url: string;
  /** 拼图图片 */
  puzzle_image?: string;
  /** 目标X坐标 */
  target_x?: number;
  /** 目标Y坐标 */
  target_y?: number;
  /** 拼图Y坐标 */
  puzzle_y?: number;
  /** 拼图样式 */
  puzzle_style?: number;
  /** 容差值 */
  tolerance?: number;
  /** 提示URL */
  hint_url?: string;
  /** 形状 */
  shape?: number;
  /** 秘密Y坐标 */
  secret_y?: number;
  /** 图片宽度 */
  image_width?: number;
  /** 图片高度 */
  image_height?: number;
}

/**
 * 点击验证码响应
 */
export interface ClickCaptchaResponse {
  /** 会话ID */
  session_id: string;
  /** 图片URL */
  image_url: string;
  /** 提示信息 */
  hint: string;
  /** 提示顺序 */
  hint_order: number[];
  /** 最大点数 */
  max_points: number;
  /** 模式 */
  mode: string;
  /** 是否允许打乱 */
  allow_shuffle: boolean;
  /** 图标位置数组 */
  points?: [number, number][];
  /** 图标位置（扩展字段） */
  icon_positions?: [number, number][];
}

/**
 * 图形验证码响应
 */
export interface ImageCaptchaResponse {
  /** 挑战ID */
  challenge_id: string;
  /** 图片数据（base64） */
  image: string;
}

/**
 * 旋转验证码响应
 */
export interface RotationCaptchaResponse {
  /** 挑战ID */
  challenge_id: string;
  /** 图片数据（base64） */
  image: string;
}

/**
 * 手势验证码响应
 */
export interface GestureCaptchaResponse {
  /** 会话ID */
  session_id: string;
  /** 手势模式 */
  pattern?: string;
  /** 网格大小 */
  grid_size?: number;
  /** 提示信息 */
  hint?: string;
}

/**
 * 拼图碎片
 */
export interface JigsawPiece {
  /** 碎片索引 */
  index: number;
  /** 原始X坐标 */
  original_x: number;
  /** 原始Y坐标 */
  original_y: number;
  /** 当前X坐标 */
  current_x: number;
  /** 当前Y坐标 */
  current_y: number;
  /** 碎片宽度 */
  width: number;
  /** 碎片高度 */
  height: number;
  /** 旋转角度 */
  rotation?: number;
}

/**
 * 拼图验证码响应
 */
export interface JigsawCaptchaResponse {
  /** 会话ID */
  session_id: string;
  /** 图片URL */
  image_url: string;
  /** 碎片列表 */
  pieces: JigsawPiece[];
  /** 碎片图片列表 */
  piece_images: string[];
  /** 网格大小 */
  grid_size: number;
  /** 碎片宽度 */
  piece_width: number;
  /** 碎片高度 */
  piece_height: number;
  /** 图片宽度 */
  image_width: number;
  /** 图片高度 */
  image_height: number;
}

/**
 * 验证码验证请求基础字段
 */
export interface VerifyCaptchaRequestBase {
  /** 会话ID */
  session_id: string;
  /** 验证码类型 */
  type: 'slider' | 'click' | 'gesture' | 'rotation' | 'jigsaw' | 'voice' | 'connect' | '3d';
  /** 行为数据 */
  behavior_data?: BehaviorDataPoint[];
  /** 应用ID */
  application_id?: number;
  /** 环境数据 */
  environment_data?: Record<string, unknown>;
}

/**
 * 滑块验证请求
 */
export interface SliderVerifyRequest extends VerifyCaptchaRequestBase {
  type: 'slider';
  /** X坐标 */
  x?: number;
  /** Y坐标 */
  y?: number;
  /** 轨迹数据 */
  trajectory?: TrajectoryPoint[];
}

/**
 * 点击验证请求
 */
export interface ClickVerifyRequest extends VerifyCaptchaRequestBase {
  type: 'click';
  /** 点击坐标 */
  points?: [number, number][];
  /** 点击顺序 */
  click_sequence?: number[];
}

/**
 * 手势验证请求
 */
export interface GestureVerifyRequest extends VerifyCaptchaRequestBase {
  type: 'gesture';
  /** 手势模式 */
  pattern?: number[];
}

/**
 * 旋转验证请求
 */
export interface RotationVerifyRequest extends VerifyCaptchaRequestBase {
  type: 'rotation';
  /** 旋转角度 */
  angle?: number;
}

/**
 * 拼图验证请求
 */
export interface JigsawVerifyRequest extends VerifyCaptchaRequestBase {
  type: 'jigsaw';
  /** 碎片数据 */
  pieces?: JigsawPieceVerify[];
}

/**
 * 验证请求（联合类型）
 */
export type VerifyCaptchaRequest =
  | SliderVerifyRequest
  | ClickVerifyRequest
  | GestureVerifyRequest
  | RotationVerifyRequest
  | JigsawVerifyRequest;

/**
 * 拼图碎片验证数据
 */
export interface JigsawPieceVerify {
  /** 碎片索引 */
  index: number;
  /** 原始X坐标 */
  original_x: number;
  /** 原始Y坐标 */
  original_y: number;
  /** 当前X坐标 */
  current_x: number;
  /** 当前Y坐标 */
  current_y: number;
  /** 碎片宽度 */
  width: number;
  /** 碎片高度 */
  height: number;
  /** 旋转角度 */
  rotation?: number;
}

/**
 * 行为数据点
 */
export interface BehaviorDataPoint {
  /** X坐标 */
  x: number;
  /** Y坐标 */
  y: number;
  /** 时间戳 */
  timestamp: number;
  /** 事件类型 */
  event: string;
}

/**
 * 验证结果响应
 */
export interface VerifyCaptchaResponse {
  /** 是否成功 */
  success: boolean;
  /** 消息 */
  message: string;
  /** 风险评分 */
  risk_score?: number;
  /** 验证码是否通过 */
  captcha_pass?: boolean;
  /** 失败原因 */
  fail_reason?: string;
  /** 剩余尝试次数 */
  remaining_attempts?: number;
  /** 轨迹分析结果 */
  trajectory_result?: TrajectoryResult;
}

/**
 * 轨迹分析结果
 */
export interface TrajectoryResult {
  /** 评分 */
  score: number;
  /** 是否通过 */
  passed: boolean;
  /** 原因列表 */
  reasons?: string[];
}

/**
 * 登录请求
 */
export interface LoginRequest {
  /** 用户名 */
  username: string;
  /** 密码 */
  password: string;
  /** 验证码令牌（可选） */
  captcha_token?: string;
}

/**
 * 登录响应
 */
export interface LoginResponse {
  /** 访问令牌 */
  access_token: string;
  /** 刷新令牌 */
  refresh_token: string;
  /** 过期时间（秒） */
  expires_in: number;
  /** 用户信息 */
  user: {
    /** 用户ID */
    id: number;
    /** 用户名 */
    username: string;
    /** 邮箱（可选） */
    email?: string;
  };
}

/**
 * 注册请求
 */
export interface RegisterRequest {
  /** 用户名 */
  username: string;
  /** 邮箱 */
  email: string;
  /** 密码 */
  password: string;
  /** 行为数据（可选） */
  behavior_data?: string;
}

/**
 * API响应包装器
 */
export interface ApiResponse<T = unknown> {
  /** 状态码 */
  code: number;
  /** 消息 */
  message: string;
  /** 数据 */
  data: T;
}

/**
 * 批量请求选项
 */
export interface BatchRequestOptions {
  /** 最大并发数 */
  concurrency?: number;
  /** 重试次数 */
  retries?: number;
}

/**
 * 批量请求结果
 */
export interface BatchResult<T> {
  /** 成功的结果 */
  successful: T[];
  /** 失败的结果 */
  failed: Array<{ index: number; error: Error }>;
  /** 成功率 */
  successRate: number;
}
