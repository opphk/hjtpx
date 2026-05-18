/**
 * Node.js TypeScript SDK 完整示例
 *
 * 展示所有功能的使用方法
 */

import {
  CaptchaClient,
  CaptchaClientConfig,
  SliderCaptchaResponse,
  ClickCaptchaResponse,
  ImageCaptchaResponse,
  GestureCaptchaResponse,
  VerifyCaptchaResponse,
  TrajectoryPoint,
  LoginRequest,
} from './src/index';

const config: CaptchaClientConfig = {
  baseUrl: 'http://localhost:8080',
  apiKey: 'your-api-key',
  timeout: 30000,
  maxConnections: 100,
  retryConfig: {
    maxRetries: 3,
    initialDelayMs: 100,
    maxDelayMs: 5000,
  },
};

async function basicSliderExample() {
  console.log('='.repeat(60));
  console.log('滑块验证码完整示例');
  console.log('='.repeat(60));

  const client = new CaptchaClient(config);

  try {
    console.log('\n步骤1: 获取滑块验证码');
    const slider: SliderCaptchaResponse = await client.getSliderCaptcha({
      width: 320,
      height: 160,
      tolerance: 8,
    });

    console.log(`✓ Session ID: ${slider.session_id}`);
    console.log(`✓ 图片宽度: ${slider.image_width || 'N/A'}`);
    console.log(`✓ 图片高度: ${slider.image_height || 'N/A'}`);
    console.log(`✓ Secret Y: ${slider.secret_y || 'N/A'}`);

    console.log('\n步骤2: 模拟用户滑动');
    const targetX = slider.target_x || 150;
    const trajectory: TrajectoryPoint[] = generateRealisticTrajectory(
      slider.secret_y || 80,
      targetX,
      800
    );

    console.log(`✓ 生成 ${trajectory.length} 个轨迹点`);

    console.log('\n步骤3: 提交验证');
    const result: VerifyCaptchaResponse = await client.verifyCaptcha({
      session_id: slider.session_id,
      type: 'slider',
      x: targetX,
      y: slider.secret_y,
      trajectory,
    });

    console.log(`\n验证结果:`);
    console.log(`  成功: ${result.success}`);
    console.log(`  消息: ${result.message}`);

    if (result.trajectory_result) {
      console.log(`  轨迹得分: ${result.trajectory_result.score}`);
      console.log(`  轨迹通过: ${result.trajectory_result.passed}`);
    }

    if (result.risk_score !== undefined) {
      console.log(`  风险评分: ${result.risk_score}`);
    }

    if (result.remaining_attempts !== undefined) {
      console.log(`  剩余尝试: ${result.remaining_attempts}`);
    }

  } catch (error) {
    console.error(`✗ 错误: ${error}`);
    throw error;
  } finally {
    await client.close();
  }
}

async function clickCaptchaExample() {
  console.log('='.repeat(60));
  console.log('点击验证码完整示例');
  console.log('='.repeat(60));

  const client = new CaptchaClient(config);

  try {
    console.log('\n步骤1: 获取点击验证码');
    const click: ClickCaptchaResponse = await client.getClickCaptcha({
      mode: 'number',
      shuffle: true,
      points: 4,
    });

    console.log(`✓ Session ID: ${click.session_id}`);
    console.log(`✓ 提示: ${click.hint}`);
    console.log(`✓ 模式: ${click.mode}`);
    console.log(`✓ 最大点数: ${click.max_points}`);

    if (click.icon_positions) {
      console.log(`✓ 图标位置数量: ${click.icon_positions.length}`);
    }

    console.log('\n步骤2: 模拟用户点击');
    const mockClicks: [number, number][] = [
      [120, 150],
      [200, 150],
      [160, 220],
      [280, 220],
    ];

    console.log(`✓ 点击坐标: ${JSON.stringify(mockClicks)}`);

    console.log('\n步骤3: 提交验证');
    const result: VerifyCaptchaResponse = await client.verifyCaptcha({
      session_id: click.session_id,
      type: 'click',
      points: mockClicks,
      click_sequence: click.hint_order,
    });

    console.log(`\n验证结果:`);
    console.log(`  成功: ${result.success}`);
    console.log(`  消息: ${result.message}`);

    if (result.fail_reason) {
      console.log(`  失败原因: ${result.fail_reason}`);
    }

  } catch (error) {
    console.error(`✗ 错误: ${error}`);
    throw error;
  } finally {
    await client.close();
  }
}

async function imageCaptchaExample() {
  console.log('='.repeat(60));
  console.log('图形验证码完整示例');
  console.log('='.repeat(60));

  const client = new CaptchaClient(config);

  try {
    const testCases = [
      { name: '数字验证码', params: { type: 'number' as const, count: 4 } },
      { name: '字母验证码', params: { type: 'letter' as const, count: 5 } },
      { name: '混合验证码', params: { type: 'mixed' as const, count: 6 } },
      { name: '中文验证码', params: { type: 'chinese' as const, count: 3 } },
    ];

    for (const testCase of testCases) {
      console.log(`\n测试: ${testCase.name}`);

      const image: ImageCaptchaResponse = await client.getSliderCaptcha(testCase.params as any);

      console.log(`  ✓ Challenge ID: ${(image as any).challenge_id || image.session_id}`);

      const result = await client.verifyCaptcha({
        session_id: image.session_id,
        type: 'slider',
        x: 0,
      });

      console.log(`  ✓ 验证成功: ${result.success}`);
    }

  } catch (error) {
    console.error(`✗ 错误: ${error}`);
    throw error;
  } finally {
    await client.close();
  }
}

async function gestureCaptchaExample() {
  console.log('='.repeat(60));
  console.log('手势验证码完整示例');
  console.log('='.repeat(60));

  const client = new CaptchaClient(config);

  try {
    console.log('\n步骤1: 获取手势验证码');
    const gesture: any = await client.getGestureCaptcha();

    console.log(`✓ Session ID: ${gesture.session_id}`);
    if (gesture.hint) {
      console.log(`✓ 提示: ${gesture.hint}`);
    }
    if (gesture.grid_size) {
      console.log(`✓ 网格大小: ${gesture.grid_size}x${gesture.grid_size}`);
    }

    console.log('\n步骤2: 模拟手势模式');
    const pattern = [0, 1, 2, 4, 8];
    console.log(`✓ 手势模式: ${pattern.join(' -> ')}`);

    console.log('\n步骤3: 提交验证');
    const result = await client.verifyGestureCaptcha(gesture.session_id, pattern);

    console.log(`\n验证结果:`);
    console.log(`  成功: ${result.success}`);
    console.log(`  消息: ${result.message}`);

  } catch (error) {
    console.error(`✗ 错误: ${error}`);
    throw error;
  } finally {
    await client.close();
  }
}

async function batchProcessingExample() {
  console.log('='.repeat(60));
  console.log('批量处理示例');
  console.log('='.repeat(60));

  const client = new CaptchaClient(config);
  const numRequests = 10;

  try {
    console.log(`\n启动 ${numRequests} 个并发请求...`);

    const startTime = Date.now();

    const promises = Array.from({ length: numRequests }, async (_, index) => {
      try {
        const slider = await client.getSliderCaptcha();
        const result = await client.verifyCaptcha({
          session_id: slider.session_id,
          type: 'slider',
          x: 150,
        });
        return { success: result.success, duration: Date.now() - startTime };
      } catch (error) {
        return { success: false, error };
      }
    });

    const results = await Promise.all(promises);

    const totalDuration = Date.now() - startTime;
    const successCount = results.filter((r) => r.success).length;

    console.log(`\n批量处理统计:`);
    console.log(`  总请求数: ${numRequests}`);
    console.log(`  成功数: ${successCount}`);
    console.log(`  成功率: ${(successCount / numRequests * 100).toFixed(1)}%`);
    console.log(`  总耗时: ${(totalDuration / 1000).toFixed(2)}秒`);
    console.log(`  平均每个请求: ${(totalDuration / numRequests).toFixed(0)}ms`);

  } catch (error) {
    console.error(`✗ 错误: ${error}`);
    throw error;
  } finally {
    await client.close();
  }
}

async function errorHandlingExample() {
  console.log('='.repeat(60));
  console.log('错误处理示例');
  console.log('='.repeat(60));

  const testConfigs = [
    {
      name: '无效Session验证',
      config: config,
      testFn: async (client: CaptchaClient) => {
        return client.verifyCaptcha({
          session_id: 'invalid-session-id',
          type: 'slider',
          x: 150,
        });
      },
    },
  ];

  for (const testCase of testConfigs) {
    console.log(`\n测试场景: ${testCase.name}`);

    const client = new CaptchaClient(testCase.config);

    try {
      await testCase.testFn(client);
      console.log('  ✓ 通过');
    } catch (error: any) {
      console.log(`  ✗ 捕获错误: ${error.message || error}`);
      console.log(`    错误类型: ${error.name || error.constructor.name}`);
      if (error.statusCode) {
        console.log(`    HTTP状态码: ${error.statusCode}`);
      }
      if (error.code) {
        console.log(`    错误码: ${error.code}`);
      }
    } finally {
      await client.close();
    }
  }
}

async function authenticationExample() {
  console.log('='.repeat(60));
  console.log('用户认证示例');
  console.log('='.repeat(60));

  const client = new CaptchaClient(config);

  try {
    console.log('\n步骤1: 用户登录');
    const loginRequest: LoginRequest = {
      username: 'testuser',
      password: 'password123',
    };

    const loginResult = await client.authLogin(loginRequest);

    console.log(`✓ 登录成功`);
    console.log(`  Access Token: ${loginResult.access_token.substring(0, 20)}...`);
    console.log(`  过期时间: ${loginResult.expires_in}秒`);
    console.log(`  用户ID: ${loginResult.user.id}`);
    console.log(`  用户名: ${loginResult.user.username}`);

    console.log('\n步骤2: 获取检测脚本');
    const script = await client.getDetectionScript();
    console.log(`✓ 脚本长度: ${script.length} 字符`);

    console.log('\n步骤3: 提交环境检测数据');
    const detectionResult = await client.submitDetection({
      fingerprint: 'browser-fingerprint',
      canvas_hash: 'canvas-hash',
      webgl_vendor: 'WebGL Vendor',
      timezone: 'Asia/Shanghai',
    });
    console.log(`✓ 检测提交成功`);

  } catch (error) {
    console.error(`✗ 错误: ${error}`);
    throw error;
  } finally {
    await client.close();
  }
}

function generateRealisticTrajectory(
  secretY: number,
  targetX: number,
  durationMs: number = 800
): TrajectoryPoint[] {
  const numPoints = 20;
  const points: TrajectoryPoint[] = [];

  for (let i = 0; i < numPoints; i++) {
    const progress = i / (numPoints - 1);
    const easedProgress = 0.5 - 0.5 * Math.pow(1 - progress, 3);

    const x = Math.round(targetX * easedProgress);
    const yOffset = Math.round(
      5 * (i % 2 === 0 ? 1 : -1) * (1 - progress) * (1 - Math.abs(progress - 0.5) * 2)
    );
    const y = secretY + yOffset;
    const t = Math.round(durationMs * progress);

    points.push({ x, y, t });
  }

  points[points.length - 1] = { x: targetX, y: secretY, t: durationMs };

  return points;
}

async function runAllExamples() {
  console.log('\n' + '='.repeat(60));
  console.log('  Node.js TypeScript SDK 完整示例');
  console.log('='.repeat(60) + '\n');

  const examples = [
    { name: '滑块验证码', fn: basicSliderExample },
    { name: '点击验证码', fn: clickCaptchaExample },
    { name: '图形验证码', fn: imageCaptchaExample },
    { name: '手势验证码', fn: gestureCaptchaExample },
    { name: '批量处理', fn: batchProcessingExample },
    { name: '错误处理', fn: errorHandlingExample },
    { name: '用户认证', fn: authenticationExample },
  ];

  for (const example of examples) {
    try {
      await example.fn();
    } catch (error) {
      console.error(`\n✗ 示例 '${example.name}' 执行失败`);
      console.error(error);
    }
    console.log('\n');
  }

  console.log('='.repeat(60));
  console.log('  所有示例运行完成');
  console.log('='.repeat(60));
}

runAllExamples().catch(console.error);
