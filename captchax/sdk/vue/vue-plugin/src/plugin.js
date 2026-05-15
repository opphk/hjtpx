import CaptchaButton from './components/CaptchaButton.vue';
import CaptchaDialog from './components/CaptchaDialog.vue';
import CaptchaSlider from './components/CaptchaSlider.vue';

export default {
  install(app, options = {}) {
    const config = {
      apiKey: options.apiKey,
      apiSecret: options.apiSecret,
      serverUrl: options.serverUrl || 'https://api.captchax.com',
      ...options
    };
    
    app.provide('captchaConfig', config);
    
    app.component('CaptchaButton', CaptchaButton);
    app.component('CaptchaDialog', CaptchaDialog);
    app.component('CaptchaSlider', CaptchaSlider);
  }
};
