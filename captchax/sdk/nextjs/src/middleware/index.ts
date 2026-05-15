import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import { verifySignature, isTimestampValid } from '../server/utils';

interface CaptchaMiddlewareOptions {
  apiKey?: string;
  apiSecret?: string;
  serverUrl?: string;
  protectedPaths?: string[];
  captchaPaths?: string[];
  tokenCookieName?: string;
  tokenHeaderName?: string;
  bypassPaths?: string[];
}

const DEFAULT_OPTIONS: Required<CaptchaMiddlewareOptions> = {
  apiKey: process.env.CAPTCHA_API_KEY || '',
  apiSecret: process.env.CAPTCHA_API_SECRET || '',
  serverUrl: process.env.CAPTCHA_SERVER_URL || 'https://api.captchax.com',
  protectedPaths: ['/api/*'],
  captchaPaths: ['/login', '/register', '/checkout', '/comment'],
  tokenCookieName: 'captcha_token',
  tokenHeaderName: 'x-captcha-token',
  bypassPaths: ['/api/health', '/api/public']
};

export function createCaptchaMiddleware(options: CaptchaMiddlewareOptions = {}) {
  const config = { ...DEFAULT_OPTIONS, ...options };

  return async function captchaMiddleware(
    request: NextRequest
  ): Promise<Response> {
    const { pathname } = request.nextUrl;

    if (shouldBypass(pathname, config.bypassPaths)) {
      return NextResponse.next();
    }

    if (!shouldProtect(pathname, config.protectedPaths)) {
      return NextResponse.next();
    }

    const needsCaptcha = config.captchaPaths.some(captchaPath =>
      pathname.startsWith(captchaPath)
    );

    if (!needsCaptcha) {
      return NextResponse.next();
    }

    const captchaToken = request.cookies.get(config.tokenCookieName)?.value;
    const captchaHeader = request.headers.get(config.tokenHeaderName);

    if (!captchaToken && !captchaHeader) {
      return NextResponse.json(
        {
          error: 'CAPTCHA_REQUIRED',
          message: 'Verification required',
          code: 403
        },
        { status: 403 }
      );
    }

    const token = captchaToken || captchaHeader;

    try {
      const isValid = await verifyCaptchaToken(
        token,
        config.serverUrl,
        config.apiKey,
        config.apiSecret
      );

      if (!isValid) {
        return NextResponse.json(
          {
            error: 'CAPTCHA_INVALID',
            message: 'Invalid or expired verification',
            code: 403
          },
          { status: 403 }
        );
      }

      const response = NextResponse.next();
      
      if (captchaToken) {
        response.cookies.set(config.tokenCookieName, captchaToken, {
          httpOnly: true,
          secure: process.env.NODE_ENV === 'production',
          sameSite: 'lax',
          maxAge: 60 * 60 * 24
        });
      }

      return response;
    } catch (error) {
      console.error('Captcha middleware error:', error);
      return NextResponse.json(
        {
          error: 'CAPTCHA_VERIFICATION_FAILED',
          message: 'Verification service unavailable',
          code: 500
        },
        { status: 500 }
      );
    }
  };
}

async function verifyCaptchaToken(
  token: string,
  serverUrl: string,
  apiKey: string,
  apiSecret: string
): Promise<boolean> {
  try {
    const timestamp = parseInt(token.split(':')[1] || '0', 10);
    
    if (!isTimestampValid(timestamp, 5 * 60 * 1000)) {
      return false;
    }

    const signature = token.split(':')[0];
    const data = `${apiKey}:${timestamp}`;
    
    if (!verifySignature(apiSecret, data, timestamp, signature)) {
      return false;
    }

    const response = await fetch(`${serverUrl}/api/v2/verify`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': apiKey
      },
      body: JSON.stringify({ token })
    });

    const result = await response.json();
    return result.success === true;
  } catch {
    return false;
  }
}

function shouldBypass(pathname: string, bypassPaths: string[]): boolean {
  return bypassPaths.some(bypassPath => {
    if (bypassPath.endsWith('*')) {
      const prefix = bypassPath.slice(0, -1);
      return pathname.startsWith(prefix);
    }
    return pathname === bypassPath || pathname.startsWith(bypassPath + '/');
  });
}

function shouldProtect(pathname: string, protectedPaths: string[]): boolean {
  return protectedPaths.some(protectedPath => {
    if (protectedPath === '*') {
      return true;
    }
    if (protectedPath.endsWith('*')) {
      const prefix = protectedPath.slice(0, -1);
      return pathname.startsWith(prefix);
    }
    return pathname.startsWith(protectedPath);
  });
}

export const captchaMiddleware = createCaptchaMiddleware();

export default captchaMiddleware;
