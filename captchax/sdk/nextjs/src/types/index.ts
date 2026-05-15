export interface VerifyOptions {
  token: string;
  scene?: string;
  ip?: string;
  userAgent?: string;
}

export interface VerifyResponse {
  success: boolean;
  score?: number;
  riskLevel?: 'low' | 'medium' | 'high';
  error?: string;
}

export interface CaptchaConfig {
  apiKey: string;
  apiSecret?: string;
  serverUrl?: string;
}

export interface CaptchaProviderProps {
  children: React.ReactNode;
  apiKey: string;
  serverUrl?: string;
}

export interface CaptchaButtonProps {
  children?: React.ReactNode;
  scene?: string;
  onSuccess?: (token: string) => void;
  onError?: (error: Error) => void;
  text?: string;
  disabled?: boolean;
  className?: string;
}

export interface CaptchaDialogProps {
  open: boolean;
  onClose: () => void;
  onSuccess: (token: string) => void;
  onError?: (error: Error) => void;
  scene?: string;
  type?: 'slider' | 'click' | 'puzzle' | 'rotate' | 'text' | 'icon';
}

export interface CaptchaSliderProps {
  onSuccess: (token: string) => void;
  onError?: (error: Error) => void;
  onClose?: () => void;
  scene?: string;
}

export interface UseCaptchaVerifyOptions {
  scene?: string;
  onSuccess?: (token: string) => void;
  onError?: (error: Error) => void;
}

export interface UseCaptchaVerifyReturn {
  token: string | null;
  loading: boolean;
  error: Error | null;
  verify: () => Promise<string | null>;
  reset: () => void;
}

export interface MiddlewareOptions {
  apiKey?: string;
  protectedPaths?: string[];
  captchaPaths?: string[];
}

export interface CaptchaXServerConfig {
  apiKey: string;
  apiSecret: string;
  serverUrl?: string;
}

export type CaptchaType = 'slider' | 'click' | 'puzzle' | 'rotate' | 'text' | 'icon';

export interface CaptchaChallenge {
  id: string;
  type: CaptchaType;
  data: {
    backgroundImage?: string;
    sliderImage?: string;
    targetPosition?: { x: number; y: number };
    clickPositions?: Array<{ x: number; y: number }>;
    rotationAngle?: number;
    text?: string;
    icons?: string[];
    targetIcon?: string;
  };
  expiresAt: number;
}

export interface CaptchaResult {
  success: boolean;
  token?: string;
  score?: number;
  riskLevel?: 'low' | 'medium' | 'high';
  error?: string;
}
