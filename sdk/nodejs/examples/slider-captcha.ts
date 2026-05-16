import { CaptchaClient } from '../src';

async function main() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
    timeout: 30000,
  });

  try {
    console.log('Getting slider captcha...');
    const captcha = await client.getSliderCaptcha({
      width: 360,
      height: 220,
    });

    console.log('Captcha received:', captcha.session_id);
    console.log('Target X:', captcha.target_x);
    console.log('Target Y:', captcha.target_y);

    // In a real app, you would collect user behavior data
    const verifyResponse = await client.verifyCaptcha({
      session_id: captcha.session_id,
      type: 'slider',
      x: captcha.target_x,
      y: captcha.target_y,
      behavior_data: [
        { x: 0, y: captcha.target_y, timestamp: Date.now() - 1000, event: 'mousemove' },
        { x: 50, y: captcha.target_y + 5, timestamp: Date.now() - 800, event: 'mousemove' },
        { x: 100, y: captcha.target_y - 3, timestamp: Date.now() - 500, event: 'mousemove' },
        { x: captcha.target_x, y: captcha.target_y, timestamp: Date.now(), event: 'mouseup' },
      ],
    });

    console.log('Verification result:', verifyResponse);

    if (verifyResponse.success) {
      console.log('Verification passed!');
    } else {
      console.log('Verification failed:', verifyResponse.message);
    }
  } catch (error) {
    console.error('Error:', error);
  } finally {
    await client.close();
  }
}

main().catch(console.error);
