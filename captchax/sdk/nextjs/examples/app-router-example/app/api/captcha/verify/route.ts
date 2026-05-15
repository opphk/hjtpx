import { NextRequest, NextResponse } from 'next/server';
import { verifyCaptchaServer } from '@captchax/nextjs/server';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { token, scene = 'default' } = body;

    if (!token) {
      return NextResponse.json(
        { success: false, error: 'Token is required' },
        { status: 400 }
      );
    }

    const apiKey = process.env.CAPTCHA_API_KEY;
    const apiSecret = process.env.CAPTCHA_API_SECRET;
    const serverUrl = process.env.CAPTCHA_SERVER_URL || 'https://api.captchax.com';

    if (!apiKey || !apiSecret) {
      return NextResponse.json(
        { success: false, error: 'Captcha configuration missing' },
        { status: 500 }
      );
    }

    const clientIp = request.headers.get('x-forwarded-for') || 'unknown';
    const userAgent = request.headers.get('user-agent') || 'unknown';

    const result = await verifyCaptchaServer(token, {
      apiKey,
      apiSecret,
      serverUrl,
      scene,
      ip: clientIp,
      userAgent
    });

    if (result.success) {
      return NextResponse.json({
        success: true,
        token,
        score: result.score,
        riskLevel: result.riskLevel
      });
    } else {
      return NextResponse.json(
        {
          success: false,
          error: result.error || 'Verification failed',
          score: result.score,
          riskLevel: result.riskLevel
        },
        { status: 400 }
      );
    }
  } catch (error) {
    console.error('Captcha verification error:', error);
    return NextResponse.json(
      { success: false, error: 'Verification service unavailable' },
      { status: 500 }
    );
  }
}
