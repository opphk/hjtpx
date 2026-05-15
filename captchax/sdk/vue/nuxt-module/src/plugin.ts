import { useCaptcha } from './runtime/composables/useCaptcha';

export default defineNuxtPlugin(() => {
  const captcha = useCaptcha();
  
  return {
    provide: {
      captcha
    }
  };
});
