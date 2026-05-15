# CaptchaX Vue 3 Plugin

Vue 3 验证码组件库

## 安装

```bash
npm install @captchax/vue
```

## 快速开始

### 1. 引入插件

```javascript
import { createApp } from 'vue';
import CaptchaX from '@captchax/vue';
import App from './App.vue';

const app = createApp(App);

app.use(CaptchaX, {
  apiKey: 'YOUR_API_KEY',
  apiSecret: 'YOUR_API_SECRET',
  serverUrl: 'https://api.captchax.com'
});

app.mount('#app');
```

### 2. 使用组件

#### CaptchaButton 验证按钮

```vue
<template>
  <captcha-button 
    scene="login" 
    text="点击验证"
    size="medium"
    theme="light"
    @success="handleSuccess"
    @error="handleError"
  >
    自定义按钮文本
  </captcha-button>
</template>

<script setup>
const handleSuccess = (token) => {
  console.log('Verified:', token);
};

const handleError = (error) => {
  console.error('Verification failed:', error);
};
</script>
```

#### CaptchaDialog 验证弹窗

```vue
<template>
  <captcha-dialog 
    v-model:visible="dialogVisible"
    type="slider"
    title="安全验证"
    @success="handleSuccess"
    @error="handleError"
    @close="handleClose"
  />
</template>

<script setup>
import { ref } from 'vue';

const dialogVisible = ref(false);

const openDialog = () => {
  dialogVisible.value = true;
};
</script>
```

#### CaptchaSlider 滑块验证

```vue
<template>
  <captcha-slider 
    target-image="/images/target.jpg"
    slider-image="/images/slider.jpg"
    @success="handleSuccess"
    @error="handleError"
  />
</template>

<script setup>
const handleSuccess = (token) => {
  console.log('Slider verified:', token);
};
</script>
```

### 3. 使用 Composable

```javascript
import { useCaptcha } from '@captchax/vue';

export default {
  setup() {
    const { verify, config } = useCaptcha();
    
    const performVerification = async () => {
      try {
        const token = await verify('login');
        console.log('Verification token:', token);
      } catch (error) {
        console.error('Verification failed:', error);
      }
    };
    
    return {
      verify,
      config,
      performVerification
    };
  }
};
```

## 组件列表

### CaptchaButton

验证按钮组件

**属性 (Props)**

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| scene | String | 'default' | 验证场景 |
| text | String | '验证' | 按钮文本 |
| size | String | 'medium' | 按钮尺寸，可选值：small, medium, large |
| theme | String | 'light' | 主题风格，可选值：light, dark |
| disabled | Boolean | false | 是否禁用 |

**事件 (Events)**

| 事件 | 参数 | 说明 |
|------|------|------|
| success | token: string | 验证成功回调 |
| error | error: Error | 验证失败回调 |

### CaptchaDialog

验证弹窗组件

**属性 (Props)**

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| visible (v-model) | Boolean | false | 弹窗可见性 |
| type | String | 'slider' | 验证类型，可选值：slider, click, rotate, puzzle, text, icon |
| title | String | '安全验证' | 弹窗标题 |
| targetImage | String | '' | 目标图片 URL |
| sliderImage | String | '' | 滑块图片 URL |

**事件 (Events)**

| 事件 | 参数 | 说明 |
|------|------|------|
| success | token: string | 验证成功回调 |
| error | error: Error | 验证失败回调 |
| close | - | 弹窗关闭回调 |

### CaptchaSlider

滑块验证组件

**属性 (Props)**

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| targetImage | String | '' | 目标图片 URL |
| sliderImage | String | '' | 滑块图片 URL |

**事件 (Events)**

| 事件 | 参数 | 说明 |
|------|------|------|
| success | token: string | 验证成功回调 |
| error | error: Error | 验证失败回调 |

## Composables

### useCaptcha

验证码核心 composable

**返回值**

```typescript
{
  verify: (scene?: string) => Promise<string>;
  config: Readonly<CaptchaConfig>;
}
```

### useCaptchaState

验证码状态管理 composable

**返回值**

```typescript
{
  show: () => void;
  hide: () => void;
  setLoading: (loading: boolean) => void;
  setToken: (token: string) => void;
  setError: (error: Error) => void;
  reset: () => void;
  isVisible: Readonly<Ref<boolean>>;
  isLoading: Readonly<Ref<boolean>>;
  token: Readonly<Ref<string | null>>;
  error: Readonly<Ref<Error | null>>;
}
```

## Nuxt 3 使用

对于 Nuxt 3 项目，推荐使用专用的 [@captchax/nuxt](./nuxt-module) 模块：

```bash
npm install @captchax/nuxt
```

```typescript
// nuxt.config.ts
export default defineNuxtConfig({
  modules: ['@captchax/nuxt'],
  captcha: {
    apiKey: process.env.CAPTCHA_API_KEY,
    apiSecret: process.env.CAPTCHA_API_SECRET
  }
});
```

## TypeScript 支持

该插件完全支持 TypeScript，所有组件和 composables 都提供了完整的类型定义。

## 示例项目

详细使用示例请参考项目中的 [示例代码](./examples)。

## 许可证

MIT License
