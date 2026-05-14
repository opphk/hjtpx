export interface CaptchaXConfig {
  baseUrl: string;
  appId?: string;
  timeout?: number;
  retryTimes?: number;
  apiVersion?: 'v1' | 'v2';
}

export interface CaptchaResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

export interface SliderCaptchaResult {
  id: string;
  background_b64: string;
  slider_b64: string;
  target_x: number;
  target_y: number;
}

export interface SliderVerifyRequest {
  captcha_id: string;
  target_x: number;
  target_y?: number;
}

export interface SliderVerifyResult {
  success: boolean;
  message: string;
}

export interface ClickCaptchaResult {
  id: string;
  image: string;
  target_chars: string[];
  char_positions: CharPosition[];
}

export interface CharPosition {
  char: string;
  x: number;
  y: number;
}

export interface ClickVerifyRequest {
  captcha_id: string;
  clicks: CharPosition[];
}

export interface ClickVerifyResult {
  success: boolean;
  score: number;
  message: string;
}

export interface PuzzleCaptchaResult {
  id: string;
  background_b64: string;
  puzzle_b64: string;
  target_x: number;
  target_y: number;
}

export interface PuzzleVerifyRequest {
  captcha_id: string;
  target_x: number;
  target_y?: number;
}

export interface PuzzleVerifyResult {
  success: boolean;
  message: string;
}

export interface Scenario {
  id: string;
  name: string;
  description?: string;
  difficulty?: 'easy' | 'medium' | 'hard';
  config?: Record<string, unknown>;
  created_at?: string;
  updated_at?: string;
}

export interface Webhook {
  id?: string;
  app_id: string;
  url: string;
  secret?: string;
  events: string[];
  headers?: Record<string, string>;
  enabled?: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface BatchVerifyItem {
  captcha_id: string;
  type: 'slider' | 'click' | 'puzzle';
  target_x: number;
  target_y?: number;
  clicks?: CharPosition[];
}

export interface BatchVerifyRequest {
  items: BatchVerifyItem[];
}

export interface BatchVerifyResult {
  captcha_id: string;
  success: boolean;
  message: string;
  score?: number;
}

export interface BatchVerifyResponse {
  results: BatchVerifyResult[];
  summary: {
    total: number;
    success: number;
    failed: number;
    skipped: number;
  };
}

export interface HealthStatus {
  status: string;
  service: string;
  timestamp: string;
  version: string;
}

export class CaptchaXError extends Error {
  public code: number;
  public statusCode: number;
  public details?: unknown;

  constructor(message: string, code: number = 500, statusCode: number = 500, details?: unknown) {
    super(message);
    this.name = 'CaptchaXError';
    this.code = code;
    this.statusCode = statusCode;
    this.details = details;
  }
}

export type CaptchaType = 'slider' | 'click' | 'puzzle';

export interface ClientInfo {
  userAgent?: string;
  ip?: string;
  platform?: string;
  [key: string]: unknown;
}
