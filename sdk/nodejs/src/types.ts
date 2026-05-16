export interface CaptchaClientConfig {
  baseUrl: string;
  apiKey?: string;
  timeout?: number;
  maxConnections?: number;
  retryConfig?: RetryConfig;
}

export interface RetryConfig {
  maxRetries?: number;
  initialDelayMs?: number;
  maxDelayMs?: number;
  retryableStatuses?: number[];
}

export interface TrajectoryPoint {
  x: number;
  y: number;
  t: number;
}

export interface SliderCaptchaResponse {
  session_id: string;
  image_url: string;
  puzzle_image: string;
  target_x: number;
  target_y: number;
  puzzle_y: number;
  puzzle_style: number;
  tolerance: number;
}

export interface ClickCaptchaResponse {
  session_id: string;
  image_url: string;
  hint: string;
  hint_order: number[];
  max_points: number;
  mode: string;
  allow_shuffle: boolean;
  points: [number, number][];
}

export interface VerifyCaptchaRequest {
  session_id: string;
  type: 'slider' | 'click' | 'gesture';
  x?: number;
  y?: number;
  points?: [number, number][];
  click_sequence?: number[];
  behavior_data?: BehaviorDataPoint[];
  application_id?: number;
  environment_data?: Record<string, unknown>;
}

export interface BehaviorDataPoint {
  x: number;
  y: number;
  timestamp: number;
  event: string;
}

export interface VerifyCaptchaResponse {
  success: boolean;
  message: string;
  risk_score: number;
  captcha_pass: boolean;
  fail_reason?: string;
}

export interface LoginRequest {
  username: string;
  password: string;
  captcha_token?: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: {
    id: number;
    username: string;
    email?: string;
  };
}

export interface ApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
}
