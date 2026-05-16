export interface Config {
  baseUrl?: string;
  appId?: string;
  appSecret?: string;
  timeout?: number;
  maxRetries?: number;
  retryDelay?: number;
  maxIdleConns?: number;
  maxOpenConns?: number;
  debugMode?: boolean;
}

export interface SDKError {
  code: number;
  message: string;
  retryAfter?: number;
}

export interface SDKResponse<T> {
  code: number;
  message: string;
  data?: T;
}

export enum CaptchaType {
  NUMBER = 'number',
  LETTER = 'letter',
  MIXED = 'mixed'
}

export interface ImageCaptchaRequest {
  type?: CaptchaType;
  count?: number;
  customSet?: string;
  noiseMode?: number;
  lineMode?: number;
}

export interface ImageCaptchaResponse {
  challenge_id: string;
  image: string;
}

export interface SliderCaptchaRequest {
  width?: number;
  height?: number;
}

export interface SliderCaptchaResponse {
  challenge_id: string;
  background_image: string;
  slider_image: string;
  slider_width: number;
  slider_height: number;
}

export interface ClickCaptchaRequest {
  width?: number;
  height?: number;
  iconCount?: number;
}

export interface ClickCaptchaResponse {
  challenge_id: string;
  background_image: string;
  target_position: number[];
  target_index: number;
  icon_positions: number[][];
}

export interface ClickData {
  x: number;
  y: number;
  duration?: number;
}

export interface VerifyCaptchaRequest {
  challenge_id: string;
  action: string;
  data?: Record<string, any>;
}

export interface VerifyCaptchaResponse {
  success: boolean;
  score?: number;
  message?: string;
  risk_level?: string;
}

export interface VerifyImageCaptchaRequest {
  challenge_id: string;
  answer: string;
}

export interface VerifyImageCaptchaResponse {
  success: boolean;
}

export interface PoolStats {
  activeConnections: number;
  idleConnections: number;
  totalRequests: number;
  failedRequests: number;
  successfulRequests: number;
  retriedRequests: number;
  successRate: number;
  lastError?: string;
  lastErrorTime?: string;
}

export interface SDKOptions extends Config {
  fetch?: typeof fetch;
}
