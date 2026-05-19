import express, { Request, Response, NextFunction } from 'express';
import { CaptchaClient, CaptchaError, ValidationError, RateLimitError } from '../src/index';

const app = express();
app.use(express.json());

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

  static async getImage(type = 'mixed', count = 4) {
    const client = getCaptchaClient();
    try {
      return await client.getClickCaptcha({ mode: type as any });
    } finally {
      await client.close();
    }
  }

  static async verifyImage(challengeId: string, answer: string) {
    const client = getCaptchaClient();
    try {
      return await client.verifyCaptcha({
        session_id: challengeId,
        type: 'image',
        answer,
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

app.get('/api/captcha/slider', async (req: Request, res: Response) => {
  try {
    const width = parseInt(req.query.width as string) || 320;
    const height = parseInt(req.query.height as string) || 160;
    const tolerance = parseInt(req.query.tolerance as string) || 8;

    const captcha = await CaptchaService.getSlider(width, height, tolerance);

    res.json({
      success: true,
      session_id: captcha.session_id,
      image_url: captcha.image_url,
      puzzle_url: captcha.puzzle_url,
      secret_y: captcha.target_y,
    });
  } catch (error) {
    handleError(error, res);
  }
});

app.post('/api/captcha/slider/verify', async (req: Request, res: Response) => {
  try {
    const { session_id, x, y, trajectory } = req.body;

    if (!session_id || x === undefined) {
      return res.status(400).json({
        success: false,
        error: 'Missing required parameters',
      });
    }

    const result = await CaptchaService.verifySlider(
      session_id,
      x,
      y,
      trajectory
    );

    res.json({
      success: result.success,
      message: result.message,
      risk_score: result.risk_score,
      captcha_pass: result.captcha_pass,
    });
  } catch (error) {
    handleError(error, res);
  }
});

app.get('/api/captcha/click', async (req: Request, res: Response) => {
  try {
    const mode = (req.query.mode as string) || 'number';
    const points = parseInt(req.query.points as string) || 3;

    const captcha = await CaptchaService.getClick(mode, points);

    res.json({
      success: true,
      session_id: captcha.session_id,
      image_url: captcha.image_url,
      hint: captcha.hint,
      hint_order: captcha.hint_order,
      max_points: captcha.max_points,
      mode: captcha.mode,
    });
  } catch (error) {
    handleError(error, res);
  }
});

app.post('/api/captcha/click/verify', async (req: Request, res: Response) => {
  try {
    const { session_id, points, click_sequence } = req.body;

    if (!session_id || !points) {
      return res.status(400).json({
        success: false,
        error: 'Missing required parameters',
      });
    }

    const result = await CaptchaService.verifyClick(
      session_id,
      points,
      click_sequence
    );

    res.json({
      success: result.success,
      message: result.message,
      risk_score: result.risk_score,
    });
  } catch (error) {
    handleError(error, res);
  }
});

app.get('/api/captcha/image', async (req: Request, res: Response) => {
  try {
    const type = (req.query.type as string) || 'mixed';
    const count = parseInt(req.query.count as string) || 4;

    const captcha = await CaptchaService.getClick(type, count);

    res.json({
      success: true,
      session_id: captcha.session_id,
      image_url: captcha.image_url,
    });
  } catch (error) {
    handleError(error, res);
  }
});

app.post('/api/captcha/image/verify', async (req: Request, res: Response) => {
  try {
    const { session_id, answer } = req.body;

    if (!session_id || !answer) {
      return res.status(400).json({
        success: false,
        error: 'Missing required parameters',
      });
    }

    const result = await CaptchaService.verifyImage(session_id, answer);

    res.json({
      success: result.success,
      message: result.message,
    });
  } catch (error) {
    handleError(error, res);
  }
});

app.post('/api/auth/login', async (req: Request, res: Response) => {
  try {
    const { username, password, captcha_token } = req.body;

    if (!username || !password) {
      return res.status(400).json({
        success: false,
        error: 'Missing credentials',
      });
    }

    const result = await CaptchaService.userLogin(
      username,
      password,
      captcha_token
    );

    res.json({
      success: true,
      access_token: result.access_token,
      refresh_token: result.refresh_token,
      expires_in: result.expires_in,
    });
  } catch (error) {
    handleError(error, res);
  }
});

function handleError(error: unknown, res: Response): void {
  console.error('Error:', error);

  if (error instanceof ValidationError) {
    res.status(400).json({
      success: false,
      error: 'Validation error: ' + error.message,
    });
  } else if (error instanceof RateLimitError) {
    res.status(429).json({
      success: false,
      error: 'Rate limit exceeded',
      retryAfter: error.retryAfter,
    });
  } else if (error instanceof CaptchaError) {
    res.status(500).json({
      success: false,
      error: error.message,
      code: error.code,
    });
  } else {
    res.status(500).json({
      success: false,
      error: 'Internal server error',
    });
  }
}

app.use((err: Error, req: Request, res: Response, next: NextFunction) => {
  console.error('Unhandled error:', err);
  res.status(500).json({
    success: false,
    error: 'Internal server error',
  });
});

const PORT = process.env.PORT || 3000;

app.listen(PORT, () => {
  console.log(`Express server running on port ${PORT}`);
});

export { app, CaptchaService };
