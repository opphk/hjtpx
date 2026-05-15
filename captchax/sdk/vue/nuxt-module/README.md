# CaptchaX Nuxt 3 Module

CaptchaX Nuxt 3 集成模块

## 安装

```bash
npm install @captchax/nuxt
```

## 快速开始

### 1. 配置模块

在 `nuxt.config.ts` 中添加模块配置：

```typescript
// nuxt.config.ts
export default defineNuxtConfig({
  modules: ['@captchax/nuxt'],
  captcha: {
    apiKey: process.env.CAPTCHA_API_KEY,
    apiSecret: process.env.CAPTCHA_API_SECRET,
    serverUrl: 'https://api.captchax.com',
    enabled: true
  }
});
```

### 2. 使用组件

#### CaptchaButton 验证按钮

```vue
<template>
  <CaptchaButton 
    scene="login" 
    text="点击验证"
    size="medium"
    theme="light"
    @success="handleSuccess"
    @error="handleError"
  >
    自定义按钮文本
  </CaptchaButton>
</template>

<script setup lang="ts">
const handleSuccess = (token: string) => {
  console.log('Verified:', token);
};

const handleError = (error: Error) => {
  console.error('Verification failed:', error);
};
</script>
```

#### CaptchaDialog 验证弹窗

```vue
<template>
  <CaptchaDialog 
    v-model:visible="dialogVisible"
    type="slider"
    title="安全验证"
    @success="handleSuccess"
    @error="handleError"
    @close="handleClose"
  />
</template>

<script setup lang="ts">
const dialogVisible = ref(false);

const openDialog = () => {
  dialogVisible.value = true;
};
</script>
```

#### CaptchaSlider 滑块验证

```vue
<template>
  <CaptchaSlider 
    target-image="/images/target.jpg"
    slider-image="/images/slider.jpg"
    @success="handleSuccess"
    @error="handleError"
  />
</template>

<script setup lang="ts">
const handleSuccess = (token: string) => {
  console.log('Slider verified:', token);
};
</script>
```

### 3. 使用 Composable

```typescript
// pages/verify.vue
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

## 配置选项

### captcha

- **apiKey** (string): CaptchaX API 密钥
- **apiSecret** (string): CaptchaX API 密钥（服务端使用）
- **serverUrl** (string): CaptchaX 服务器地址，默认：`https://api.captchax.com`
- **enabled** (boolean): 是否启用验证，默认：`true`

## 组件列表

### CaptchaButton

验证按钮组件，自动注册为全局组件。

**属性 (Props)**

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| scene | String | 'default' | 验证场景 |
| text | String | '验证' | 按钮文本 |
| size | 'small' \| 'medium' \| 'large' | 'medium' | 按钮尺寸 |
| theme | 'light' \| 'dark' | 'light' | 主题风格 |
| disabled | Boolean | false | 是否禁用 |

**事件 (Events)**

| 事件 | 参数 | 说明 |
|------|------|------|
| success | token: string | 验证成功回调 |
| error | error: Error | 验证失败回调 |

### CaptchaDialog

验证弹窗组件，支持多种验证类型。

**属性 (Props)**

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| visible | Boolean | false | 弹窗可见性 |
| type | 'slider' \| 'click' \| 'rotate' \| 'puzzle' \| 'text' \| 'icon' | 'slider' | 验证类型 |
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

滑块验证组件。

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

验证码核心 composable，自动导入。

```typescript
const { verify, config } = useCaptcha();

// 验证指定场景
const token = await verify('login');
```

### useCaptchaState

验证码状态管理 composable。

```typescript
const { show, hide, isVisible, isLoading, token, error } = useCaptchaState();

// 显示验证弹窗
show();

// 隐藏验证弹窗
hide();

// 重置状态
reset();
```

## TypeScript 支持

该模块完全支持 TypeScript，所有组件和 composables 都提供了完整的类型定义。

## SSR 支持

该模块完全支持服务端渲染 (SSR)，所有组件和 composables 都能在服务端安全使用。

## 示例项目

详细使用示例请参考项目中的 [示例代码](./examples)。

## Vue 3 Plugin

如果你的项目不使用 Nuxt 3，也可以使用独立的 [Vue 3 Plugin](../vue-plugin)。

## 许可证

MIT License
