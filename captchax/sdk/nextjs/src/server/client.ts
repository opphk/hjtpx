import crypto from 'crypto';
import type { CaptchaXServerConfig, VerifyOptions, VerifyResponse } from '../types';

export class CaptchaXServer {
  private apiKey: string;
  private apiSecret: string;
  private serverUrl: string;

  constructor(config: CaptchaXServerConfig) {
    this.apiKey = config.apiKey;
    this.apiSecret = config.apiSecret;
    this.serverUrl = config.serverUrl || 'https://api.captchax.com';
  }

  async verify(options: VerifyOptions): Promise<VerifyResponse> {
    const { token, scene, ip, userAgent } = options;

    if (!token) {
      return {
        success: false,
        error: 'Token is required'
      };
    }

    const timestamp = Date.now();
    const signature = this.generateSignature(token, timestamp);

    try {
      const response = await fetch(`${this.serverUrl}/api/v2/verify`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': this.apiKey,
          'X-Timestamp': timestamp.toString(),
          'X-Signature': signature
        },
        body: JSON.stringify({
          token,
          scene,
          ip,
          userAgent
        })
      });

      if (!response.ok) {
        return {
          success: false,
          error: `HTTP error: ${response.status}`
        };
      }

      const data = await response.json();
      return {
        success: data.success || false,
        score: data.score,
        riskLevel: data.riskLevel,
        error: data.error
      };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Verification failed'
      };
    }
  }

  async getChallenge(scene: string = 'default'): Promise<{
    success: boolean;
    challenge?: unknown;
    error?: string;
  }> {
    const timestamp = Date.now();
    const signature = this.generateSignature(scene, timestamp);

    try {
      const response = await fetch(`${this.serverUrl}/api/v1/captcha/challenge`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': this.apiKey,
          'X-Timestamp': timestamp.toString(),
          'X-Signature': signature
        },
        body: JSON.stringify({ scene })
      });

      const data = await response.json();
      return {
        success: response.ok,
        challenge: data,
        error: data.error
      };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to get challenge'
      };
    }
  }

  private generateSignature(data: string, timestamp: number): string {
    const payload = `${data}:${timestamp}`;
    return crypto
      .createHmac('sha256', this.apiSecret)
      .update(payload)
      .digest('hex');
  }
}

export function createCaptchaXServer(config: CaptchaXServerConfig): CaptchaXServer {
  return new CaptchaXServer(config);
}
