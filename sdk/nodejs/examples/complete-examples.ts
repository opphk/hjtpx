import {
  CaptchaClient,
  CaptchaClientConfig,
  SliderCaptchaResponse,
  ClickCaptchaResponse,
  VerifyCaptchaResponse,
  TrajectoryPoint,
} from '../src/index';

async function runAllExamples(): Promise<void> {
  console.log('='.repeat(50));
  console.log('  HJTpx Captcha SDK - Complete Examples');
  console.log('='.repeat(50));
  console.log();

  const config: CaptchaClientConfig = {
    baseUrl: 'http://localhost:8080',
    apiKey: 'demo-api-key',
    timeout: 30000,
    maxConnections: 100,
    retryConfig: {
      maxRetries: 3,
      initialDelayMs: 100,
      maxDelayMs: 5000,
    },
  };

  const client = new CaptchaClient(config);

  try {
    await sliderExample(client);
    console.log();
    await clickExample(client);
    console.log();
    await gestureExample(client);
    console.log();
    await batchExample(client);
    console.log();
    console.log('All examples completed successfully!');
  } catch (error) {
    console.error('Error running examples:', error);
  } finally {
    await client.close();
  }
}

async function sliderExample(client: CaptchaClient): Promise<void> {
  console.log('[1] Slider Captcha Example');

  const slider: SliderCaptchaResponse = await client.getSliderCaptcha({
    width: 360,
    height: 200,
    tolerance: 8,
  });

  console.log(`  Session ID: ${slider.session_id}`);

  const trajectory: TrajectoryPoint[] = [
    { x: 0, y: 100, t: 0 },
    { x: 30, y: 102, t: 50 },
    { x: 60, y: 98, t: 100 },
    { x: 90, y: 101, t: 150 },
    { x: 120, y: 99, t: 200 },
    { x: 150, y: 100, t: 250 },
  ];

  const result = await client.verifyCaptcha({
    session_id: slider.session_id,
    type: 'slider',
    x: 150,
    y: slider.secret_y,
    trajectory,
  });

  if (result.success) {
    console.log('  ✓ Verification successful!');
  } else {
    console.log(`  ✗ Verification failed: ${result.message}`);
  }
}

async function clickExample(client: CaptchaClient): Promise<void> {
  console.log('[2] Click Captcha Example');

  const click: ClickCaptchaResponse = await client.getClickCaptcha({
    mode: 'number',
    shuffle: true,
    points: 3,
  });

  console.log(`  Session ID: ${click.session_id}`);
  console.log(`  Hint: ${click.hint}`);

  const points: number[][] = [];
  if (click.icon_positions && click.hint_order) {
    for (const idx of click.hint_order.slice(0, click.max_points)) {
      if (click.icon_positions[idx]) {
        points.push(click.icon_positions[idx]);
      }
    }
  }

  const result = await client.verifyCaptcha({
    session_id: click.session_id,
    type: 'click',
    points,
    click_sequence: click.hint_order?.slice(0, click.max_points),
  });

  if (result.success) {
    console.log('  ✓ Verification successful!');
  } else {
    console.log(`  ✗ Verification failed: ${result.message}`);
  }
}

async function gestureExample(client: CaptchaClient): Promise<void> {
  console.log('[3] Gesture Captcha Example');

  const gesture = await client.getGestureCaptcha();
  console.log(`  Session ID: ${gesture.session_id}`);

  const result = await client.verifyGestureCaptcha(gesture.session_id, [0, 1, 2, 5, 8]);

  if (result.success) {
    console.log('  ✓ Verification successful!');
  } else {
    console.log(`  ✗ Verification failed: ${result.message}`);
  }
}

async function batchExample(client: CaptchaClient): Promise<void> {
  console.log('[4] Batch Captcha Example');

  const promises = Array.from({ length: 5 }, () =>
    client.getSliderCaptcha({ width: 320, height: 160 })
  );

  const results = await Promise.allSettled(promises);

  let successCount = 0;
  for (const result of results) {
    if (result.status === 'fulfilled') {
      successCount++;
      console.log(`  ✓ ${result.value.session_id.substring(0, 8)}...`);
    }
  }

  console.log(`  Success: ${successCount}/${results.length}`);
}

if (require.main === module) {
  runAllExamples().catch(console.error);
}

export { runAllExamples };
