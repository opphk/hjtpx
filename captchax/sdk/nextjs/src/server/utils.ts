import crypto from 'crypto';

export function generateToken(): string {
  return crypto.randomBytes(32).toString('hex');
}

export function generateSignature(
  secret: string,
  token: string,
  timestamp: number
): string {
  const payload = `${token}:${timestamp}`;
  return crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');
}

export function verifySignature(
  secret: string,
  token: string,
  timestamp: number,
  signature: string
): boolean {
  const expectedSignature = generateSignature(secret, token, timestamp);
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expectedSignature)
  );
}

export function isTimestampValid(timestamp: number, windowMs: number = 300000): boolean {
  const now = Date.now();
  const diff = Math.abs(now - timestamp);
  return diff <= windowMs;
}

export function getClientIp(request: Request): string {
  const forwardedFor = request.headers.get('x-forwarded-for');
  if (forwardedFor) {
    return forwardedFor.split(',')[0].trim();
  }
  const realIp = request.headers.get('x-real-ip');
  if (realIp) {
    return realIp;
  }
  return 'unknown';
}

export function getUserAgent(request: Request): string {
  return request.headers.get('user-agent') || 'unknown';
}

export function createErrorResponse(
  error: string,
  status: number = 400
): Response {
  return new Response(
    JSON.stringify({ success: false, error }),
    {
      status,
      headers: { 'Content-Type': 'application/json' }
    }
  );
}

export function createSuccessResponse(data: Record<string, unknown>): Response {
  return new Response(
    JSON.stringify({ success: true, ...data }),
    {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    }
  );
}
