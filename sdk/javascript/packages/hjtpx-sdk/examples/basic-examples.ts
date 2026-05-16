import { CaptchaClient, CaptchaType } from 'hjtpx-sdk';

async function exampleImageCaptcha() {
  console.log('\n=== Image Captcha Example ===');

  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
    debugMode: true,
  });

  const captcha = await client.generateImageCaptcha({
    type: CaptchaType.MIXED,
    count: 4,
    noiseMode: 2,
    lineMode: 1,
  });

  console.log('✓ Challenge ID:', captcha.challenge_id);

  const result = await client.verifyImageCaptcha(captcha.challenge_id, '1234');
  console.log('✓ Verification success:', result.success);

  const stats = client.getStats();
  console.log('\n📊 Statistics:');
  console.log('  Total Requests:', stats.totalRequests);
  console.log('  Success Rate:', stats.successRate.toFixed(2) + '%');
}

async function exampleSliderCaptcha() {
  console.log('\n=== Slider Captcha Example ===');

  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  const slider = await client.generateSliderCaptcha({
    width: 360,
    height: 220,
  });

  console.log('✓ Challenge ID:', slider.challenge_id);
  console.log('  Slider Size:', slider.slider_width + 'x' + slider.slider_height);

  const result = await client.verifySliderCaptcha(slider.challenge_id, '120');
  console.log('✓ Verification success:', result.success);
  console.log('  Score:', result.score);
  console.log('  Risk Level:', result.risk_level);
}

async function exampleClickCaptcha() {
  console.log('\n=== Click Captcha Example ===');

  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  const click = await client.generateClickCaptcha({
    width: 360,
    height: 220,
    iconCount: 4,
  });

  console.log('✓ Challenge ID:', click.challenge_id);
  console.log('  Target Index:', click.target_index);
  console.log('  Icon Positions:', click.icon_positions);

  const clicks = [
    {
      x: click.target_position[0],
      y: click.target_position[1],
      duration: 500,
    },
  ];

  const result = await client.verifyClickCaptcha(click.challenge_id, clicks);
  console.log('✓ Verification success:', result.success);
}

async function exampleErrorHandling() {
  console.log('\n=== Error Handling Example ===');

  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
    maxRetries: 0,
  });

  try {
    await client.verifyImageCaptcha('', '1234');
  } catch (error) {
    console.log('✓ Caught expected error:', error.message);
    console.log('  Error Code:', error.code);
    console.log('  Is Invalid Params:', error.isInvalidParams());
  }

  try {
    await client.verifySliderCaptcha('test-id', '');
  } catch (error) {
    console.log('✓ Caught expected error:', error.message);
  }
}

async function exampleStatistics() {
  console.log('\n=== Statistics Example ===');

  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  await client.generateImageCaptcha();
  await client.generateSliderCaptcha();
  await client.generateClickCaptcha();

  const stats = client.getStats();
  console.log('📊 Current Statistics:');
  console.log('  Total Requests:', stats.totalRequests);
  console.log('  Successful:', stats.successfulRequests);
  console.log('  Failed:', stats.failedRequests);
  console.log('  Retried:', stats.retriedRequests);
  console.log('  Success Rate:', stats.successRate.toFixed(2) + '%');
}

async function main() {
  console.log('='.repeat(50));
  console.log('hjtpx JavaScript/TypeScript SDK Examples');
  console.log('='.repeat(50));

  try {
    await exampleImageCaptcha();
    await exampleSliderCaptcha();
    await exampleClickCaptcha();
    await exampleErrorHandling();
    await exampleStatistics();

    console.log('\n' + '='.repeat(50));
    console.log('All examples completed successfully!');
    console.log('='.repeat(50));
  } catch (error) {
    console.error('\n✗ Error:', error);
  }
}

main();
