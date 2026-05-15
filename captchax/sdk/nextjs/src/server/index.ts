export { CaptchaXServer, createCaptchaXServer } from './client';
export {
  generateToken,
  generateSignature,
  verifySignature,
  isTimestampValid,
  getClientIp,
  getUserAgent,
  createErrorResponse,
  createSuccessResponse
} from './utils';

export type { CaptchaXServerConfig } from '../types';

export async function verifyCaptchaServer(
  token: string,
  options: {
    apiKey: string;
    apiSecret: string;
    serverUrl?: string;
    scene?: string;
    ip?: string;
    userAgent?: string;
  }
): Promise<{
  success: boolean;
  score?: number;
  riskLevel?: 'low' | 'medium' | 'high';
  error?: string;
}> {
  const { CaptchaXServer } = await import('./client');
  
  const client = new CaptchaXServer({
    apiKey: options.apiKey,
    apiSecret: options.apiSecret,
    serverUrl: options.serverUrl
  });

  return client.verify({
    token,
    scene: options.scene,
    ip: options.ip,
    userAgent: options.userAgent
  });
}
