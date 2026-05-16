import { CaptchaClient } from '../src';

async function main() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
    timeout: 30000,
  });

  try {
    console.log('Getting click captcha...');
    const captcha = await client.getClickCaptcha({
      mode: 'number',
      shuffle: true,
      points: 3,
    });

    console.log('Captcha received:', captcha.session_id);
    console.log('Hint:', captcha.hint);
    console.log('Hint order:', captcha.hint_order);

    // In a real app, you would collect user click points
    const points = captcha.points;

    const verifyResponse = await client.verifyCaptcha({
      session_id: captcha.session_id,
      type: 'click',
      points,
      click_sequence: captcha.hint_order,
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
