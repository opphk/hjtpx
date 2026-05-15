import { defineNuxtModule, addPlugin, addComponent } from '@nuxt/kit';

export interface CaptchaModuleOptions {
  apiKey?: string;
  apiSecret?: string;
  serverUrl?: string;
  enabled?: boolean;
}

export default defineNuxtModule<CaptchaModuleOptions>({
  name: 'captchax',
  configKey: 'captcha',
  
  defaults: {
    apiKey: '',
    apiSecret: '',
    serverUrl: 'https://api.captchax.com',
    enabled: true
  },
  
  setup(options, nuxt) {
    if (!options.enabled) {
      return;
    }
    
    nuxt.options.runtimeConfig.captcha = {
      apiKey: options.apiKey,
      apiSecret: options.apiSecret,
      serverUrl: options.serverUrl
    };
    
    addPlugin({
      src: './plugin.ts',
      fileName: 'captchax/plugin.ts'
    });
    
    addComponent({
      name: 'CaptchaButton',
      filePath: './runtime/components/CaptchaButton.vue'
    });
    
    addComponent({
      name: 'CaptchaDialog',
      filePath: './runtime/components/CaptchaDialog.vue'
    });
    
    addComponent({
      name: 'CaptchaSlider',
      filePath: './runtime/components/CaptchaSlider.vue'
    });
    
    nuxt.hook('components:dirs', (dirs) => {
      dirs.push({
        path: './runtime/components',
        prefix: 'Captcha'
      });
    });
  }
});

declare module '@nuxt/schema' {
  interface NuxtConfig {
    captcha?: CaptchaModuleOptions;
  }
  
  interface RuntimeConfig {
    captcha: CaptchaModuleOptions;
  }
}
