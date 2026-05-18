/**
 * JavaScript Browser SDK 类型定义
 *
 * 为浏览器端验证码SDK提供完整的TypeScript类型支持
 */

/**
 * 验证码客户端配置选项
 */
export interface CaptchaClientOptions {
  /** API基础URL */
  baseURL: string;
  /** API密钥（可选） */
  apiKey?: string;
  /** 请求超时时间（毫秒），默认30000 */
  timeout?: number;
  /** 最大重试次数，默认3 */
  retryCount?: number;
  /** 重试延迟（毫秒），默认1000 */
  retryDelay?: number;
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
  /** 拼图图片URL */
  puzzle_url?: string;
  /** 提示URL */
  hint_url?: string;
  /** 形状类型 */
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
  hint?: string;
  /** 提示顺序 */
  hint_order?: number[];
  /** 最大点击点数 */
  max_points?: number;
  /** 模式 */
  mode?: string;
  /** 是否允许打乱顺序 */
  allow_shuffle?: boolean;
  /** 点击坐标点 */
  points?: [number, number][];
}

/**
 * 图形验证码响应
 */
export interface ImageCaptchaResponse {
  /** 挑战ID */
  challenge_id: string;
  /** 图片数据（base64或URL） */
  image: string;
}

/**
 * 旋转验证码响应
 */
export interface RotationCaptchaResponse {
  /** 挑战ID */
  challenge_id: string;
  /** 图片数据 */
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
 * 拼图碎片数据
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
  pieces?: JigsawPiece[];
  /** 碎片图片列表 */
  piece_images?: string[];
  /** 网格大小 */
  grid_size?: number;
  /** 碎片宽度 */
  piece_width?: number;
  /** 碎片高度 */
  piece_height?: number;
  /** 图片宽度 */
  image_width?: number;
  /** 图片高度 */
  image_height?: number;
}

/**
 * 验证码验证请求基础类型
 */
export interface VerifyCaptchaRequest {
  /** 会话ID */
  session_id: string;
  /** 验证码类型 */
  type: 'slider' | 'click' | 'gesture' | 'rotation' | 'jigsaw' | 'image' | 'voice';
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
export interface SliderVerifyRequest extends VerifyCaptchaRequest {
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
export interface ClickVerifyRequest extends VerifyCaptchaRequest {
  type: 'click';
  /** 点击坐标列表 */
  points?: [number, number][];
  /** 点击顺序 */
  click_sequence?: number[];
}

/**
 * 手势验证请求
 */
export interface GestureVerifyRequest extends VerifyCaptchaRequest {
  type: 'gesture';
  /** 手势模式 */
  pattern?: number[];
}

/**
 * 旋转验证请求
 */
export interface RotationVerifyRequest extends VerifyCaptchaRequest {
  type: 'rotation';
  /** 旋转角度 */
  angle?: number;
}

/**
 * 拼图验证请求
 */
export interface JigsawVerifyRequest extends VerifyCaptchaRequest {
  type: 'jigsaw';
  /** 碎片数据 */
  pieces?: JigsawPiece[];
}

/**
 * 图形验证请求
 */
export interface ImageVerifyRequest extends VerifyCaptchaRequest {
  type: 'image';
  /** 答案 */
  answer?: string;
}

/**
 * 验证请求（联合类型）
 */
export type CaptchaVerifyRequest =
  | SliderVerifyRequest
  | ClickVerifyRequest
  | GestureVerifyRequest
  | RotationVerifyRequest
  | JigsawVerifyRequest
  | ImageVerifyRequest;

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
  event: 'mousedown' | 'mousemove' | 'mouseup' | 'touchstart' | 'touchmove' | 'touchend';
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
  /** 评分（0-100） */
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
  user: UserInfo;
}

/**
 * 用户信息
 */
export interface UserInfo {
  /** 用户ID */
  id: number;
  /** 用户名 */
  username: string;
  /** 邮箱（可选） */
  email?: string;
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
  /** 状态码，0表示成功 */
  code: number;
  /** 消息 */
  message: string;
  /** 数据 */
  data: T;
}

/**
 * 轨迹记录器接口
 */
export interface TrajectoryRecorder {
  /** 获取当前轨迹点列表 */
  getPoints: () => TrajectoryPoint[];
  /** 开始记录 */
  start: () => void;
  /** 停止记录 */
  stop: () => TrajectoryPoint[];
  /** 重置轨迹 */
  reset: () => void;
  /** 是否正在记录 */
  isRecording: () => boolean;
  /** 销毁记录器 */
  destroy: () => void;
}

/**
 * 滑块验证码组件选项
 */
export interface SliderCaptchaWidgetOptions {
  /** 图片宽度 */
  width?: number;
  /** 图片高度 */
  height?: number;
  /** 容差值 */
  tolerance?: number;
  /** 验证成功回调 */
  onSuccess?: (result: VerifyCaptchaResponse) => void;
  /** 验证失败回调 */
  onFail?: (message: string) => void;
  /** 加载错误回调 */
  onError?: (error: Error) => void;
}

/**
 * 点击验证码组件选项
 */
export interface ClickCaptchaWidgetOptions {
  /** 模式 */
  mode?: 'number' | 'letter' | 'chinese' | 'mixed' | 'icon';
  /** 是否允许打乱顺序 */
  shuffle?: boolean;
  /** 最大点击点数 */
  points?: number;
  /** 验证成功回调 */
  onSuccess?: (result: VerifyCaptchaResponse) => void;
  /** 验证失败回调 */
  onFail?: (message: string) => void;
  /** 加载错误回调 */
  onError?: (error: Error) => void;
}

/**
 * 浏览器指纹数据
 */
export interface BrowserFingerprint {
  /** 用户代理 */
  user_agent: string;
  /** 语言 */
  language: string;
  /** 平台 */
  platform: string;
  /** 屏幕宽度 */
  screen_width: number;
  /** 屏幕高度 */
  screen_height: number;
  /** 颜色深度 */
  color_depth: number;
  /** 像素比 */
  pixel_ratio: number;
  /** 时区 */
  timezone: string;
  /** 时区偏移 */
  timezone_offset: number;
  /** WebGL厂商 */
  webgl_vendor?: string;
  /** WebGL渲染器 */
  webgl_renderer?: string;
  /** 插件列表 */
  plugins?: string;
  /** 检测到的字体 */
  fonts?: string[];
  /** Canvas指纹 */
  canvas_hash?: string;
  /** 音频指纹 */
  audio_fingerprint?: string;
  /** 是否为自动化工具 */
  is_webdriver?: boolean;
}

/**
 * 错误类型
 */
export interface CaptchaError extends Error {
  /** HTTP状态码 */
  status?: number;
  /** 错误码 */
  code?: number;
  /** 错误消息 */
  message: string;
}

/**
 * SDK配置接口
 */
export interface SDKConfig {
  /** API基础URL */
  baseURL: string;
  /** API密钥 */
  apiKey?: string;
  /** 超时时间（毫秒） */
  timeout: number;
  /** 重试次数 */
  retryCount: number;
  /** 重试延迟（毫秒） */
  retryDelay: number;
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
export interface BatchRequestResult<T> {
  /** 成功的请求 */
  successful: Array<{ index: number; data: T }>;
  /** 失败的请求 */
  failed: Array<{ index: number; error: Error }>;
  /** 成功率 */
  successRate: number;
}

/**
 * 导出类型
 */
export type {
  CaptchaClient as CaptchaClientClass,
  UserAuth as UserAuthClass,
  Environment as EnvironmentClass,
  SliderCaptchaWidget as SliderCaptchaWidgetClass,
  ClickCaptchaWidget as ClickCaptchaWidgetClass,
} from './captcha';
