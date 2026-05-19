import Koa from 'koa';
import Router from '@koa/router';
import bodyParser from 'koa-bodyparser';
import { CaptchaClient, CaptchaError, ValidationError, RateLimitError } from '../src/index';

const app = new Koa();
const router = new Router();

const CAPTCHA_CONFIG = {
  baseUrl: 'http://localhost:8080',
  apiKey: 'your-api-key',
  timeout: 30000,
};

function getCaptchaClient(): CaptchaClient {
  return new CaptchaClient(CAPTCHA_CONFIG);
}

class CaptchaService {
  static async getSlider(width = 320, height = 160, tolerance = 8) {
    const client = getCaptchaClient();
    try {
      return await client.getSliderCaptcha({ width, height, tolerance });
    } finally {
      await client.close();
    }
  }

  static async verifySlider(
    sessionId: string,
    x: number,
    y?: number,
    trajectory?: { x: number; y: number; t: number }[]
  ) {
    const client = getCaptchaClient();
    try {
      return await client.verifyCaptcha({
        session_id: sessionId,
        type: 'slider',
        x,
        y,
        trajectory,
      });
    } finally {
      await client.close();
    }
  }

  static async getClick(mode = 'number', points = 3, shuffle = true) {
    const client = getCaptchaClient();
    try {
      return await client.getClickCaptcha({ mode, points, shuffle });
    } finally {
      await client.close();
    }
  }

  static async verifyClick(
    sessionId: string,
    points: [number, number][],
    clickSequence?: number[]
  ) {
    const client = getCaptchaClient();
    try {
      return await client.verifyCaptcha({
        session_id: sessionId,
        type: 'click',
        points,
        click_sequence: clickSequence,
      });
    } finally {
      await client.close();
    }
  }

  static async userLogin(username: string, password: string, captchaToken?: string) {
    const client = getCaptchaClient();
    try {
      return await client.authLogin({
        username,
        password,
        captcha_token: captchaToken,
      });
    } finally {
      await client.close();
    }
  }
}

router.get('/api/captcha/slider', async (ctx) => {
  try {
    const width = parseInt(ctx.query.width as string) || 320;
    const height = parseInt(ctx.query.height as string) || 160;
    const tolerance = parseInt(ctx.query.tolerance as string) || 8;

    const captcha = await CaptchaService.getSlider(width, height, tolerance);

    ctx.body = {
      success: true,
      session_id: captcha.session_id,
      image_url: captcha.image_url,
      puzzle_url: captcha.puzzle_url,
      secret_y: captcha.target_y,
    };
  } catch (error) {
    handleError(error, ctx);
  }
});

router.post('/api/captcha/slider/verify', async (ctx) => {
  try {
    const { session_id, x, y, trajectory } = ctx.request.body as any;

    if (!session_id || x === undefined) {
      ctx.status = 400;
      ctx.body = {
        success: false,
        error: 'Missing required parameters',
      };
      return;
    }

    const result = await CaptchaService.verifySlider(
      session_id,
      x,
      y,
      trajectory
    );

    ctx.body = {
      success: result.success,
      message: result.message,
      risk_score: result.risk_score,
      captcha_pass: result.captcha_pass,
    };
  } catch (error) {
    handleError(error, ctx);
  }
});

router.get('/api/captcha/click', async (ctx) => {
  try {
    const mode = (ctx.query.mode as string) || 'number';
    const points = parseInt(ctx.query.points as string) || 3;

    const captcha = await CaptchaService.getClick(mode, points);

    ctx.body = {
      success: true,
      session_id: captcha.session_id,
      image_url: captcha.image_url,
      hint: captcha.hint,
      hint_order: captcha.hint_order,
      max_points: captcha.max_points,
      mode: captcha.mode,
    };
  } catch (error) {
    handleError(error, ctx);
  }
});

router.post('/api/captcha/click/verify', async (ctx) => {
  try {
    const { session_id, points, click_sequence } = ctx.request.body as any;

    if (!session_id || !points) {
      ctx.status = 400;
      ctx.body = {
        success: false,
        error: 'Missing required parameters',
      };
      return;
    }

    const result = await CaptchaService.verifyClick(
      session_id,
      points,
      click_sequence
    );

    ctx.body = {
      success: result.success,
      message: result.message,
      risk_score: result.risk_score,
    };
  } catch (error) {
    handleError(error, ctx);
  }
});

router.post('/api/auth/login', async (ctx) => {
  try {
    const { username, password, captcha_token } = ctx.request.body as any;

    if (!username || !password) {
      ctx.status = 400;
      ctx.body = {
        success: false,
        error: 'Missing credentials',
      };
      return;
    }

    const result = await CaptchaService.userLogin(
      username,
      password,
      captcha_token
    );

    ctx.body = {
      success: true,
      access_token: result.access_token,
      refresh_token: result.refresh_token,
      expires_in: result.expires_in,
    };
  } catch (error) {
    handleError(error, ctx);
  }
});

function handleError(error: unknown, ctx: any): void {
  console.error('Error:', error);

  if (error instanceof ValidationError) {
    ctx.status = 400;
    ctx.body = {
      success: false,
      error: 'Validation error: ' + error.message,
    };
  } else if (error instanceof RateLimitError) {
    ctx.status = 429;
    ctx.body = {
      success: false,
      error: 'Rate limit exceeded',
      retryAfter: error.retryAfter,
    };
  } else if (error instanceof CaptchaError) {
    ctx.status = 500;
    ctx.body = {
      success: false,
      error: error.message,
      code: error.code,
    };
  } else {
    ctx.status = 500;
    ctx.body = {
      success: false,
      error: 'Internal server error',
    };
  }
}

app.use(bodyParser());
app.use(router.routes());
app.use(router.allowedMethods());

app.use(async (ctx, next) => {
  try {
    await next();
  } catch (err: any) {
    console.error('Unhandled error:', err);
    ctx.status = err.status || 500;
    ctx.body = {
      success: false,
      error: err.message || 'Internal server error',
    };
  }
});

const PORT = process.env.PORT || 3000;

app.listen(PORT, () => {
  console.log(`Koa server running on port ${PORT}`);
});

export { app, CaptchaService };
