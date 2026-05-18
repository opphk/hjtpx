import { CaptchaClient } from './client';
import { CaptchaClientConfig } from './types';

const MOCK_BASE_URL = 'http://localhost:8080';

describe('CaptchaClient Workflow Tests', () => {
  let client: CaptchaClient;

  beforeEach(() => {
    client = new CaptchaClient({
      baseUrl: MOCK_BASE_URL,
      apiKey: 'test-api-key',
      timeout: 5000,
    });
  });

  afterEach(async () => {
    await client.close();
  });

  describe('Complete Slider Workflow', () => {
    it('should complete full slider captcha workflow', async () => {
      const mockSliderResponse = {
        session_id: 'slider-workflow-123',
        image_url: 'http://example.com/slider.jpg',
        puzzle_url: 'http://example.com/puzzle.jpg',
        secret_y: 80,
        image_width: 320,
        image_height: 160,
      };

      const mockVerifyResponse = {
        success: true,
        message: 'Verification passed',
        score: 0.95,
        risk_level: 'low',
        trajectory_result: {
          score: 0.92,
          passed: true,
          reasons: [],
        },
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockSliderResponse,
        }),
      } as Response);

      const sliderCaptcha = await client.getSliderCaptcha({
        width: 320,
        height: 160,
        tolerance: 8,
      });

      expect(sliderCaptcha).toEqual(mockSliderResponse);
      expect(sliderCaptcha.session_id).toBe('slider-workflow-123');

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockVerifyResponse,
        }),
      } as Response);

      const verifyResult = await client.verifyCaptcha({
        session_id: sliderCaptcha.session_id,
        type: 'slider',
        x: 150,
        y: 80,
        trajectory: [
          { x: 0, y: 80, t: 1000 },
          { x: 50, y: 82, t: 1200 },
          { x: 100, y: 78, t: 1400 },
          { x: 150, y: 80, t: 1600 },
        ],
      });

      expect(verifyResult).toEqual(mockVerifyResponse);
      expect(verifyResult.success).toBe(true);
      expect(verifyResult.score).toBe(0.95);
    });

    it('should handle slider workflow with failed verification', async () => {
      const mockSliderResponse = {
        session_id: 'slider-fail-123',
        image_url: 'http://example.com/slider.jpg',
        puzzle_url: 'http://example.com/puzzle.jpg',
        secret_y: 80,
      };

      const mockVerifyResponse = {
        success: false,
        message: 'Verification failed',
        score: 0.3,
        risk_level: 'high',
        fail_reason: 'Invalid trajectory',
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockSliderResponse,
        }),
      } as Response);

      const sliderCaptcha = await client.getSliderCaptcha();
      expect(sliderCaptcha.session_id).toBe('slider-fail-123');

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockVerifyResponse,
        }),
      } as Response);

      const verifyResult = await client.verifyCaptcha({
        session_id: sliderCaptcha.session_id,
        type: 'slider',
        x: 50,
      });

      expect(verifyResult.success).toBe(false);
      expect(verifyResult.score).toBe(0.3);
    });
  });

  describe('Complete Click Workflow', () => {
    it('should complete full click captcha workflow', async () => {
      const mockClickResponse = {
        session_id: 'click-workflow-123',
        image_url: 'http://example.com/click.jpg',
        hint: 'Click 1, 2, 3',
        hint_order: [0, 1, 2],
        mode: 'number',
        points: [
          [100, 100],
          [200, 200],
          [300, 300],
        ],
        max_points: 3,
        allow_shuffle: true,
      };

      const mockVerifyResponse = {
        success: true,
        message: 'Verification passed',
        score: 0.88,
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockClickResponse,
        }),
      } as Response);

      const clickCaptcha = await client.getClickCaptcha({
        mode: 'number',
        shuffle: true,
        points: 3,
      });

      expect(clickCaptcha).toEqual(mockClickResponse);
      expect(clickCaptcha.session_id).toBe('click-workflow-123');

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockVerifyResponse,
        }),
      } as Response);

      const verifyResult = await client.verifyCaptcha({
        session_id: clickCaptcha.session_id,
        type: 'click',
        points: [
          [100, 100],
          [200, 200],
          [300, 300],
        ],
        click_sequence: [0, 1, 2],
      });

      expect(verifyResult).toEqual(mockVerifyResponse);
      expect(verifyResult.success).toBe(true);
    });

    it('should handle click captcha with icon mode', async () => {
      const mockClickResponse = {
        session_id: 'click-icon-123',
        image_url: 'http://example.com/click-icon.jpg',
        hint: 'Click icons',
        mode: 'icon',
        points: [
          [150, 150],
          [250, 150],
          [350, 150],
        ],
        max_points: 3,
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockClickResponse,
        }),
      } as Response);

      const clickCaptcha = await client.getClickCaptcha({
        mode: 'icon',
        shuffle: false,
        points: 3,
      });

      expect(clickCaptcha.mode).toBe('icon');
      expect(clickCaptcha.max_points).toBe(3);
    });
  });

  describe('Gesture Captcha Workflow', () => {
    it('should complete gesture captcha workflow', async () => {
      const mockGestureResponse = {
        session_id: 'gesture-workflow-123',
        pattern: '1-2-3-4',
        grid_size: 3,
        hint: 'Draw the pattern',
      };

      const mockVerifyResponse = {
        success: true,
        message: 'Verification passed',
        score: 0.91,
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockGestureResponse,
        }),
      } as Response);

      const gestureCaptcha = await client.getGestureCaptcha();
      expect(gestureCaptcha.session_id).toBe('gesture-workflow-123');
      expect(gestureCaptcha.pattern).toBe('1-2-3-4');

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockVerifyResponse,
        }),
      } as Response);

      const verifyResult = await client.verifyGestureCaptcha(
        gestureCaptcha.session_id,
        [1, 2, 3, 4]
      );

      expect(verifyResult.success).toBe(true);
    });
  });

  describe('Authentication Workflow', () => {
    it('should complete login workflow', async () => {
      const mockLoginResponse = {
        access_token: 'test-access-token-123',
        refresh_token: 'test-refresh-token-456',
        expires_in: 3600,
        user: {
          id: 1,
          username: 'testuser',
          email: 'test@example.com',
        },
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockLoginResponse,
        }),
      } as Response);

      const loginResult = await client.authLogin({
        username: 'testuser',
        password: 'password123',
      });

      expect(loginResult).toEqual(mockLoginResponse);
      expect(loginResult.access_token).toBe('test-access-token-123');
      expect(loginResult.user.username).toBe('testuser');
    });

    it('should complete registration workflow', async () => {
      const mockRegisterResponse = {
        user_id: 123,
        username: 'newuser',
        email: 'newuser@example.com',
        created_at: '2024-01-01T00:00:00Z',
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockRegisterResponse,
        }),
      } as Response);

      const registerResult = await client.authRegister({
        username: 'newuser',
        email: 'newuser@example.com',
        password: 'password123',
      });

      expect(registerResult).toEqual(mockRegisterResponse);
      expect(registerResult.username).toBe('newuser');
    });

    it('should handle token refresh workflow', async () => {
      const mockRefreshResponse = {
        access_token: 'new-access-token-789',
        refresh_token: 'new-refresh-token-012',
        expires_in: 3600,
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockRefreshResponse,
        }),
      } as Response);

      const refreshResult = await client.authRefreshToken('old-refresh-token');
      expect(refreshResult.access_token).toBe('new-access-token-789');
    });
  });

  describe('Environment Detection Workflow', () => {
    it('should complete environment detection workflow', async () => {
      const mockScriptResponse = 'function detect() { return true; }';

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        text: async () => mockScriptResponse,
      } as Response);

      const script = await client.getDetectionScript('onDetectReady');
      expect(script).toBe(mockScriptResponse);

      const mockSubmitResponse = {
        success: true,
        risk_level: 'low',
        risk_score: 0.1,
        checks: {
          browser: 'passed',
          device: 'passed',
          network: 'passed',
        },
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockSubmitResponse,
        }),
      } as Response);

      const submitResult = await client.submitDetection({
        fingerprint: 'test-fingerprint',
        canvas_hash: 'test-canvas',
        webgl_vendor: 'test-vendor',
      });

      expect(submitResult).toEqual(mockSubmitResponse);
    });

    it('should complete environment check workflow', async () => {
      const mockCheckResponse = {
        success: true,
        risk_level: 'low',
        risk_score: 0.15,
        recommendations: ['allow', 'allow'],
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockCheckResponse,
        }),
      } as Response);

      const checkResult = await client.checkEnvironment({
        fingerprint: 'test-fingerprint',
        canvas_hash: 'test-canvas',
        webgl_renderer: 'test-renderer',
      });

      expect(checkResult).toEqual(mockCheckResponse);
      expect(checkResult.risk_level).toBe('low');
    });
  });

  describe('Error Recovery Workflows', () => {
    it('should handle rate limit and retry', async () => {
      let attemptCount = 0;

      global.fetch = jest.fn().mockImplementation(async () => {
        attemptCount++;
        if (attemptCount === 1) {
          return {
            ok: false,
            status: 429,
            statusText: 'Too Many Requests',
            headers: new Map([['retry-after', '1']]),
            json: async () => ({ message: 'Rate limited' }),
          };
        }
        return {
          ok: true,
          json: async () => ({
            code: 0,
            message: 'success',
            data: { session_id: 'after-retry' },
          }),
        };
      });

      const result = await client.getSliderCaptcha();
      expect(attemptCount).toBeGreaterThanOrEqual(1);
    });

    it('should handle server error and retry', async () => {
      let attemptCount = 0;

      global.fetch = jest.fn().mockImplementation(async () => {
        attemptCount++;
        if (attemptCount < 3) {
          return {
            ok: false,
            status: 500,
            statusText: 'Internal Server Error',
            json: async () => ({ message: 'Server error' }),
          };
        }
        return {
          ok: true,
          json: async () => ({
            code: 0,
            message: 'success',
            data: { session_id: 'after-server-retry' },
          }),
        };
      });

      const result = await client.getSliderCaptcha();
      expect(attemptCount).toBeGreaterThanOrEqual(2);
    });
  });

  describe('Multiple Captcha Types Workflow', () => {
    it('should handle different captcha types sequentially', async () => {
      const mockResponses = {
        slider: {
          session_id: 'slider-seq-1',
          image_url: 'http://example.com/slider.jpg',
          puzzle_url: 'http://example.com/puzzle.jpg',
        },
        click: {
          session_id: 'click-seq-1',
          image_url: 'http://example.com/click.jpg',
          hint: 'Click 1, 2',
          hint_order: [0, 1],
          mode: 'number',
          points: [
            [100, 100],
            [200, 200],
          ],
          max_points: 2,
        },
        gesture: {
          session_id: 'gesture-seq-1',
          pattern: '1-2-3',
          grid_size: 3,
        },
      };

      let requestCount = 0;

      global.fetch = jest.fn().mockImplementation(async () => {
        requestCount++;
        if (requestCount === 1) {
          return {
            ok: true,
            json: async () => ({
              code: 0,
              message: 'success',
              data: mockResponses.slider,
            }),
          };
        } else if (requestCount === 2) {
          return {
            ok: true,
            json: async () => ({
              code: 0,
              message: 'success',
              data: mockResponses.click,
            }),
          };
        } else {
          return {
            ok: true,
            json: async () => ({
              code: 0,
              message: 'success',
              data: mockResponses.gesture,
            }),
          };
        }
      });

      const slider = await client.getSliderCaptcha();
      expect(slider.session_id).toBe('slider-seq-1');

      const click = await client.getClickCaptcha({ points: 2 });
      expect(click.session_id).toBe('click-seq-1');

      const gesture = await client.getGestureCaptcha();
      expect(gesture.session_id).toBe('gesture-seq-1');
    });
  });
});

describe('CaptchaClient Configuration Tests', () => {
  it('should handle custom configuration', () => {
    const config: CaptchaClientConfig = {
      baseUrl: 'http://custom-endpoint.com',
      apiKey: 'custom-api-key',
      timeout: 10000,
      maxConnections: 200,
    };

    const client = new CaptchaClient(config);
    expect(client).toBeDefined();
  });

  it('should handle minimal configuration', () => {
    const config: CaptchaClientConfig = {
      baseUrl: 'http://minimal.com',
    };

    const client = new CaptchaClient(config);
    expect(client).toBeDefined();
  });

  it('should trim trailing slashes from baseUrl', () => {
    const client = new CaptchaClient({
      baseUrl: 'http://test.com///',
    });
    expect(client).toBeDefined();
  });
});

describe('CaptchaClient Error Handling Tests', () => {
  let client: CaptchaClient;

  beforeEach(() => {
    client = new CaptchaClient({
      baseUrl: MOCK_BASE_URL,
    });
  });

  afterEach(async () => {
    await client.close();
  });

  it('should handle network errors gracefully', async () => {
    global.fetch = jest.fn().mockRejectedValue(new Error('Network error'));

    await expect(client.getSliderCaptcha()).rejects.toThrow();
  });

  it('should handle timeout errors', async () => {
    global.fetch = jest.fn().mockRejectedValue(
      Object.assign(new Error('TimeoutError'), { name: 'TimeoutError' })
    );

    await expect(client.getSliderCaptcha()).rejects.toThrow();
  });

  it('should handle invalid JSON responses', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      json: async () => {
        throw new Error('Invalid JSON');
      },
    } as Response);

    await expect(client.getSliderCaptcha()).rejects.toThrow();
  });
});
