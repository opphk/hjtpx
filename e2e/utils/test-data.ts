export const testUsers = {
  admin: {
    username: 'admin',
    password: 'admin123',
    email: 'admin@example.com'
  },
  user: {
    username: 'testuser',
    password: 'TestPass123!',
    email: 'testuser@example.com'
  },
  test: {
    username: 'testuser',
    password: 'TestPass123!',
    email: 'testuser@example.com'
  }
};

export const testApplications = [
  {
    name: 'Test App 1',
    description: 'Test application 1',
    domain: 'test1.example.com'
  },
  {
    name: 'Test App 2',
    description: 'Test application 2',
    domain: 'test2.example.com'
  }
];

export const captchaTypes = ['slider', 'click', 'rotate', 'gesture', 'image'] as const;

export const testBlacklistEntries = [
  {
    type: 'ip',
    value: '192.168.1.100',
    reason: 'bot_attack',
    action: 'block'
  },
  {
    type: 'ip',
    value: '10.0.0.50',
    reason: 'suspicious_activity',
    action: 'captcha'
  },
  {
    type: 'user_id',
    value: 'suspicious_user',
    reason: 'abuse',
    action: 'block'
  }
];

export const testRiskLevels = {
  low: {
    min: 0,
    max: 30,
    expectedAction: 'allow'
  },
  medium: {
    min: 31,
    max: 60,
    expectedAction: 'captcha'
  },
  high: {
    min: 61,
    max: 100,
    expectedAction: 'block'
  }
};

export const testApiEndpoints = {
  health: '/health',
  admin: {
    login: '/admin/login',
    dashboard: '/admin/dashboard',
    stats: '/admin/stats',
    logs: '/admin/logs',
    applications: '/admin/applications',
    blacklist: '/admin/blacklist',
    monitoring: '/admin/monitoring',
    settings: '/admin/settings',
    security: '/admin/security'
  },
  api: {
    captcha: {
      slider: '/api/v1/captcha/slider',
      click: '/api/v1/captcha/click',
      rotate: '/api/v1/captcha/rotate',
      image: '/api/v1/captcha/image',
      verify: '/api/v1/captcha/verify'
    },
    user: {
      register: '/api/v1/user/register',
      login: '/api/v1/user/login',
      logout: '/api/v1/user/logout',
      profile: '/api/v1/user/profile'
    },
    detection: {
      script: '/api/v1/detect/script',
      submit: '/api/v1/detect/submit',
      check: '/api/v1/detect/check'
    }
  }
};

export const testBrowserConfigs = {
  desktop: {
    viewport: { width: 1280, height: 720 },
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
  },
  tablet: {
    viewport: { width: 768, height: 1024 },
    userAgent: 'Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X) AppleWebKit/605.1.15'
  },
  mobile: {
    viewport: { width: 375, height: 667 },
    userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15'
  }
};

export const testTimeout = {
  default: 30000,
  navigation: 60000,
  short: 5000,
  long: 120000
};

export const testRetryConfig = {
  maxRetries: 3,
  retryDelay: 1000
};

export function generateRandomString(length: number = 8): string {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

export function generateRandomEmail(): string {
  return `${generateRandomString(8)}@example.com`;
}

export function generateRandomUsername(): string {
  return `user_${generateRandomString(10)}`;
}

export function generateRandomIP(): string {
  return `${Math.floor(Math.random() * 256)}.${Math.floor(Math.random() * 256)}.${Math.floor(Math.random() * 256)}.${Math.floor(Math.random() * 256)}`;
}

export function generateTestCaptchaToken(): string {
  return `captcha_${generateRandomString(32)}`;
}

export function generateTestUserToken(): string {
  return `token_${generateRandomString(64)}`;
}
