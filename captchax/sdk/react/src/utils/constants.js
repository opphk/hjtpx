export const CAPTCHA_TYPES = {
  SLIDER: 'slider',
  CLICK: 'click',
  ROTATE: 'rotate',
  PUZZLE: 'puzzle',
  TEXT: 'text',
  ICON: 'icon'
};

export const CAPTCHA_SCENES = {
  LOGIN: 'login',
  REGISTER: 'register',
  COMMENT: 'comment',
  PURCHASE: 'purchase',
  FEEDBACK: 'feedback',
  DEFAULT: 'default'
};

export const CAPTCHA_STATUS = {
  IDLE: 'idle',
  LOADING: 'loading',
  READY: 'ready',
  VERIFYING: 'verifying',
  SUCCESS: 'success',
  FAILED: 'failed',
  EXPIRED: 'expired'
};

export const CAPTCHA_ERRORS = {
  NETWORK_ERROR: 'NETWORK_ERROR',
  TIMEOUT: 'TIMEOUT',
  INVALID_TOKEN: 'INVALID_TOKEN',
  SESSION_EXPIRED: 'SESSION_EXPIRED',
  TOO_MANY_ATTEMPTS: 'TOO_MANY_ATTEMPTS',
  UNKNOWN_ERROR: 'UNKNOWN_ERROR'
};

export const DEFAULT_TIMEOUT = 10000;
export const MAX_RETRY_ATTEMPTS = 3;
export const TOKEN_EXPIRY_TIME = 5 * 60 * 1000;

export const API_ENDPOINTS = {
  VERIFY: '/api/captcha/verify',
  REFRESH: '/api/captcha/refresh',
  REPORT: '/api/captcha/report',
  STATS: '/api/captcha/stats'
};

export const SLIDER_CONFIG = {
  DEFAULT_WIDTH: 300,
  DEFAULT_HEIGHT: 150,
  THUMB_SIZE: 40,
  MIN_DRAG_DISTANCE: 20,
  TOLERANCE: 5
};

export const ANIMATION_DURATION = {
  FAST: 150,
  NORMAL: 300,
  SLOW: 500
};

export default {
  CAPTCHA_TYPES,
  CAPTCHA_SCENES,
  CAPTCHA_STATUS,
  CAPTCHA_ERRORS,
  DEFAULT_TIMEOUT,
  MAX_RETRY_ATTEMPTS,
  TOKEN_EXPIRY_TIME,
  API_ENDPOINTS,
  SLIDER_CONFIG,
  ANIMATION_DURATION
};
