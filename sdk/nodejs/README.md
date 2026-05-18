# HJTPX Node.js SDK

Node.js/TypeScript SDK，用于HJTPX验证码验证系统。提供完整的验证码类型支持、连接池管理、自动重试机制和批量请求功能。

## 安装

```bash
npm install hjtpx-sdk
```

或者使用yarn：

```bash
yarn add hjtpx-sdk
```

## 快速开始

### 基础用法

```typescript
import { CaptchaClient } from 'hjtpx-sdk';

async function main() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
    apiKey: 'your-api-key',
    timeout: 30000,
  });

  try {
    const captcha = await client.getSliderCaptcha({
      width: 360,
      height: 220,
    });
    console.log('Session ID:', captcha.session_id);

    const result = await client.verifyCaptcha({
      session_id: captcha.session_id,
      type: 'slider',
      x: 150,
      y: captcha.target_y,
    });

    console.log('Verification result:', result);
  } finally {
    await client.close();
  }
}

main();
```

## 功能特性

- **多种验证码类型**：滑块、点击、手势、旋转、拼图、语音等
- **连接池管理**：使用undici实现高性能HTTP连接池
- **自动重试机制**：指数退避算法，支持可配置的重试策略
- **完善的错误处理**：针对不同错误类型提供专用异常类
- **TypeScript支持**：完整的类型定义，开箱即用
- **批量请求支持**：支持并发批量获取和验证验证码
- **异步/await API**：现代化的异步编程风格

## API参考

### CaptchaClient

#### 构造函数

```typescript
new CaptchaClient(config: CaptchaClientConfig)
```

**参数说明：**
- `baseUrl` (string, 必需): API基础URL
- `apiKey` (string, 可选): API密钥，用于身份验证
- `timeout` (number, 可选): 请求超时时间（毫秒），默认30000
- `maxConnections` (number, 可选): 最大并发连接数，默认100
- `retryConfig` (RetryConfig, 可选): 重试配置

#### 方法

##### 获取验证码

```typescript
// 滑块验证码
async getSliderCaptcha(options?: {
  width?: number;
  height?: number;
  tolerance?: number;
}): Promise<SliderCaptchaResponse>

// 点击验证码
async getClickCaptcha(options?: {
  mode?: 'number' | 'letter' | 'chinese' | 'mixed' | 'icon';
  shuffle?: boolean;
  points?: number;
}): Promise<ClickCaptchaResponse>

// 手势验证码
async getGestureCaptcha(): Promise<GestureCaptchaResponse>
```

##### 验证验证码

```typescript
// 通用验证
async verifyCaptcha(request: VerifyCaptchaRequest): Promise<VerifyCaptchaResponse>

// 手势验证（便捷方法）
async verifyGestureCaptcha(session_id: string, pattern: number[]): Promise<VerifyCaptchaResponse>

// 滑块验证（便捷方法）
async verifySliderCaptcha(
  session_id: string,
  x: number,
  y?: number,
  trajectory?: TrajectoryPoint[]
): Promise<VerifyCaptchaResponse>

// 点击验证（便捷方法）
async verifyClickCaptcha(
  session_id: string,
  points: [number, number][],
  click_sequence?: number[]
): Promise<VerifyCaptchaResponse>
```

##### 用户认证

```typescript
async authLogin(request: LoginRequest): Promise<LoginResponse>
async authRegister(request: RegisterRequest): Promise<any>
async authRefreshToken(refreshToken: string): Promise<any>
async authLogout(): Promise<void>
```

##### 环境检测

```typescript
async getDetectionScript(callback?: string): Promise<string>
async submitDetection(data: Record<string, unknown>): Promise<any>
async checkEnvironment(data: Record<string, unknown>): Promise<any>
```

##### 资源管理

```typescript
async close(): Promise<void>
```

## 错误处理

SDK为不同类型的错误提供了专门的异常类：

- `CaptchaError`: 基础错误类
- `ValidationError`: 请求参数验证失败
- `AuthenticationError`: 身份验证失败
- `NotFoundError`: 资源不存在
- `RateLimitError`: 请求频率超限
- `ServerError`: 服务器错误（5xx）
- `NetworkError`: 网络相关错误

### 错误处理示例

```typescript
import {
  CaptchaClient,
  CaptchaError,
  ValidationError,
  RateLimitError,
  NetworkError,
} from 'hjtpx-sdk';

async function handleCaptcha() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  try {
    const captcha = await client.getSliderCaptcha();
    const result = await client.verifyCaptcha({
      session_id: captcha.session_id,
      type: 'slider',
      x: 150,
    });

    if (result.success) {
      console.log('验证成功！');
    } else {
      console.log('验证失败:', result.message);
      if (result.fail_reason) {
        console.log('失败原因:', result.fail_reason);
      }
    }
  } catch (error) {
    if (error instanceof ValidationError) {
      console.error('参数验证失败:', error.message);
    } else if (error instanceof RateLimitError) {
      console.error('请求过于频繁，请稍后再试');
      console.log('建议等待:', error.retryAfter, '秒');
    } else if (error instanceof NetworkError) {
      console.error('网络错误:', error.message);
    } else if (error instanceof CaptchaError) {
      console.error('验证码错误:', error.message);
      console.error('错误码:', error.code);
    } else {
      console.error('未知错误:', error);
    }
  } finally {
    await client.close();
  }
}
```

## 完整示例

### 滑块验证码示例

```typescript
import { CaptchaClient, TrajectoryPoint } from 'hjtpx-sdk';

async function sliderExample() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
    apiKey: 'your-api-key',
    retryConfig: {
      maxRetries: 3,
      initialDelayMs: 100,
      maxDelayMs: 5000,
    },
  });

  try {
    const captcha = await client.getSliderCaptcha({
      width: 360,
      height: 200,
      tolerance: 8,
    });

    console.log('Session ID:', captcha.session_id);
    console.log('Image URL:', captcha.image_url);
    console.log('Secret Y:', captcha.target_y);

    // 生成用户滑动轨迹
    const trajectory: TrajectoryPoint[] = [
      { x: 0, y: 100, t: 0 },
      { x: 30, y: 102, t: 50 },
      { x: 60, y: 98, t: 100 },
      { x: 90, y: 101, t: 150 },
      { x: 120, y: 99, t: 200 },
      { x: 150, y: 100, t: 250 },
    ];

    // 验证滑块位置和轨迹
    const result = await client.verifyCaptcha({
      session_id: captcha.session_id,
      type: 'slider',
      x: 150,
      y: captcha.target_y,
      trajectory,
    });

    if (result.success) {
      console.log('验证通过！');
      console.log('风险评分:', result.risk_score);
    } else {
      console.log('验证失败:', result.message);
    }
  } finally {
    await client.close();
  }
}
```

### 点击验证码示例

```typescript
import { CaptchaClient } from 'hjtpx-sdk';

async function clickExample() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  try {
    const captcha = await client.getClickCaptcha({
      mode: 'number',
      shuffle: true,
      points: 3,
    });

    console.log('Session ID:', captcha.session_id);
    console.log('Hint:', captcha.hint);
    console.log('Required sequence:', captcha.hint_order);

    // 根据提示顺序点击相应位置
    // 这里简化处理，实际需要根据前端用户点击获取坐标
    const clickSequence = captcha.hint_order;
    const points: [number, number][] = clickSequence.map((idx) => {
      return captcha.icon_positions?.[idx] || captcha.points?.[idx] || [0, 0];
    });

    const result = await client.verifyCaptcha({
      session_id: captcha.session_id,
      type: 'click',
      points,
      click_sequence: clickSequence,
    });

    console.log('Verification result:', result.success);
  } finally {
    await client.close();
  }
}
```

### 手势验证码示例

```typescript
import { CaptchaClient } from 'hjtpx-sdk';

async function gestureExample() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  try {
    const captcha = await client.getGestureCaptcha();
    console.log('Session ID:', captcha.session_id);
    console.log('Grid Size:', captcha.grid_size);

    // 用户绘制的手势模式（示例：Z字形）
    const pattern = [0, 1, 2, 5, 8, 7, 6, 3];
    const result = await client.verifyGestureCaptcha(captcha.session_id, pattern);

    console.log('Verification result:', result.success);
    console.log('Message:', result.message);
  } finally {
    await client.close();
  }
}
```

### 用户认证示例

```typescript
import { CaptchaClient } from 'hjtpx-sdk';

async function authExample() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  try {
    // 注册新用户
    const registerResult = await client.authRegister({
      username: 'newuser',
      email: 'user@example.com',
      password: 'securepassword123',
    });
    console.log('Registration successful');

    // 用户登录
    const loginResult = await client.authLogin({
      username: 'newuser',
      password: 'securepassword123',
    });

    console.log('Access Token:', loginResult.access_token.substring(0, 20) + '...');
    console.log('User ID:', loginResult.user.id);
    console.log('Username:', loginResult.user.username);

    // 刷新令牌
    const refreshResult = await client.authRefreshToken(loginResult.refresh_token);
    console.log('Token refreshed');

    // 登出
    await client.authLogout();
    console.log('Logged out');
  } finally {
    await client.close();
  }
}
```

### 批量请求示例

```typescript
import { CaptchaClient } from 'hjtpx-sdk';

async function batchExample() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  try {
    console.log('Batch request example - getting 10 slider captchas concurrently');

    // 并发获取10个验证码
    const captchaPromises = Array.from({ length: 10 }, () =>
      client.getSliderCaptcha({ width: 320, height: 160 })
    );

    const captchas = await Promise.allSettled(captchaPromises);

    let successCount = 0;
    const results: any[] = [];

    for (let i = 0; i < captchas.length; i++) {
      const result = captchas[i];
      if (result.status === 'fulfilled') {
        successCount++;
        results.push(result.value);
        console.log(`Captcha ${i + 1}: ${result.value.session_id.substring(0, 20)}...`);
      } else {
        console.error(`Captcha ${i + 1} failed:`, result.reason.message);
      }
    }

    console.log(`\nSuccess rate: ${successCount}/${captchas.length}`);
    console.log(`Total captchas generated: ${results.length}`);

    // 批量验证
    console.log('\nBatch verification...');
    const verifyPromises = results.slice(0, 5).map((captcha) =>
      client.verifyCaptcha({
        session_id: captcha.session_id,
        type: 'slider',
        x: Math.floor(Math.random() * 200),
      })
    );

    const verifyResults = await Promise.allSettled(verifyPromises);

    let verifySuccessCount = 0;
    for (const result of verifyResults) {
      if (result.status === 'fulfilled' && result.value.success) {
        verifySuccessCount++;
      }
    }

    console.log(`Verification success rate: ${verifySuccessCount}/${verifyResults.length}`);
  } finally {
    await client.close();
  }
}
```

### 带轨迹的滑块验证示例

```typescript
import { CaptchaClient, TrajectoryPoint } from 'hjtpx-sdk';

async function sliderWithTrajectoryExample() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  try {
    const captcha = await client.getSliderCaptcha();
    console.log('Session ID:', captcha.session_id);

    // 模拟真实的用户滑动轨迹
    // 实际应用中应该在前端收集用户的真实滑动轨迹
    const targetX = 150;
    const trajectory: TrajectoryPoint[] = [];

    // 生成平滑的滑动轨迹
    const startTime = Date.now();
    let currentX = 0;
    const steps = 20;

    for (let i = 0; i <= steps; i++) {
      const progress = i / steps;
      const easeProgress = 1 - Math.pow(1 - progress, 3);

      currentX = Math.floor(targetX * easeProgress);

      const yVariation = Math.sin(progress * Math.PI * 2) * 3;
      const currentY = 100 + (captcha.target_y || 100) - 50 + yVariation;

      trajectory.push({
        x: currentX,
        y: Math.floor(currentY),
        t: startTime + i * 15,
      });
    }

    console.log('Trajectory points:', trajectory.length);

    const result = await client.verifyCaptcha({
      session_id: captcha.session_id,
      type: 'slider',
      x: targetX,
      y: captcha.target_y,
      trajectory,
    });

    console.log('Result:', result.success ? 'Passed' : 'Failed');
    console.log('Risk score:', result.risk_score);

    if (result.trajectory_result) {
      console.log('Trajectory analysis:');
      console.log('  Score:', result.trajectory_result.score);
      console.log('  Passed:', result.trajectory_result.passed);
      if (result.trajectory_result.reasons) {
        console.log('  Reasons:', result.trajectory_result.reasons);
      }
    }
  } finally {
    await client.close();
  }
}
```

## 重试配置

```typescript
import { CaptchaClient, RetryConfig } from 'hjtpx-sdk';

const retryConfig: RetryConfig = {
  maxRetries: 5,
  initialDelayMs: 100,
  maxDelayMs: 5000,
  retryableStatuses: [429, 500, 502, 503, 504],
};

const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
  retryConfig,
});
```

## 连接池配置

```typescript
const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
  maxConnections: 50,
  timeout: 30000,
});
```

## 高级用法

### 自定义HTTP客户端

```typescript
import { CaptchaClient } from 'hjtpx-sdk';
import { Agent } from 'undici';

const dispatcher = new Agent({
  keepAliveTimeout: 30000,
  keepAliveMaxTimeout: 60000,
});

const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
  apiKey: 'your-api-key',
});

// 通过修改内部配置使用自定义的dispatcher
// 这需要访问私有成员，仅在特殊情况下使用
```

### 环境检测集成

```typescript
import { CaptchaClient } from 'hjtpx-sdk';

async function environmentDetectionExample() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  try {
    // 获取检测脚本（在前端执行）
    const script = await client.getDetectionScript('onDetectionComplete');
    console.log('Detection script length:', script.length, 'characters');

    // 从前端收集环境数据后提交
    const detectionData = {
      fingerprint: 'browser-unique-fingerprint',
      canvas_hash: 'canvas-fingerprint-hash',
      webgl_vendor: 'WebGL vendor info',
      webgl_renderer: 'WebGL renderer info',
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      language: navigator.language,
      platform: navigator.platform,
      user_agent: navigator.userAgent,
      screen_width: screen.width,
      screen_height: screen.height,
      color_depth: screen.colorDepth,
      pixel_ratio: window.devicePixelRatio,
      is_webdriver: navigator.webdriver,
    };

    const result = await client.submitDetection(detectionData);
    console.log('Detection result:', result);

    // 检查环境安全状态
    const checkResult = await client.checkEnvironment({
      fingerprint: detectionData.fingerprint,
      risk_score: 0.1,
    });
    console.log('Environment check:', checkResult);
  } finally {
    await client.close();
  }
}
```

## 示例文件

更多示例请参考 `examples` 目录：

- `slider-captcha.ts` - 滑块验证码示例
- `click-captcha.ts` - 点击验证码示例
- `complete-examples.ts` - 完整示例集合

## 运行测试

```bash
npm test
```

## 注意事项

1. 本SDK为基本可用版本，可能存在未发现的问题
2. 请根据实际API接口调整使用方式
3. 生产环境使用前请充分测试
4. 注意合理配置重试次数和超时时间，避免对服务器造成压力

## 许可证

MIT License
