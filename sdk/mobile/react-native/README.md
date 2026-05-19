# HJTPX React Native Captcha SDK

React Native平台的验证码SDK，提供滑块、点击等验证码类型的集成。

## 功能特性

- 滑块验证码组件
- 点击验证码组件
- 触摸反馈支持
- 响应式布局
- TypeScript支持
- 跨平台支持（iOS/Android）

## 快速开始

### 安装

```bash
npm install hjtpx-captcha-react-native
# 或
yarn add hjtpx-captcha-react-native
```

### 基本使用

```tsx
import React, { useState } from 'react';
import { View } from 'react-native';
import { CaptchaClient, SliderCaptcha, CaptchaButton } from 'hjtpx-captcha-react-native';

const client = new CaptchaClient({
  baseUrl: 'https://your-api-server.com',
  appId: 'your-app-id',
  appSecret: 'your-app-secret',
});

export default function CaptchaScreen() {
  const [captchaData, setCaptchaData] = useState(null);

  const handleGenerateCaptcha = async () => {
    try {
      const result = await client.generateSliderCaptcha(320, 200);
      setCaptchaData(result);
    } catch (error) {
      console.error('Failed to generate captcha:', error);
    }
  };

  const handleVerify = async (progress: number) => {
    try {
      const result = await client.verifySliderCaptcha(
        captchaData.sessionId,
        progress
      );

      if (result.success) {
        // 验证成功
      }
    } catch (error) {
      console.error('Failed to verify captcha:', error);
    }
  };

  return (
    <View>
      {captchaData ? (
        <SliderCaptcha
          backgroundImageUrl={captchaData.backgroundImage}
          sliderImageUrl={captchaData.sliderImage}
          onSliderCompleted={handleVerify}
        />
      ) : (
        <CaptchaButton onPress={handleGenerateCaptcha} title="加载验证码" />
      )}
    </View>
  );
}
```

### 滑块验证码组件

```tsx
import { SliderCaptcha } from 'hjtpx-captcha-react-native';

<SliderCaptcha
  backgroundImageUrl="https://example.com/bg.jpg"
  sliderImageUrl="https://example.com/slider.jpg"
  width={320}
  height={200}
  onSliderMoved={(progress) => {
    console.log('Slider progress:', progress);
  }}
  onSliderCompleted={(progress) => {
    console.log('Slider completed at:', progress);
  }}
/>
```

### 配置选项

```tsx
import { ConfigManager, defaultConfig } from 'hjtpx-captcha-react-native';

const configManager = new ConfigManager({
  width: 320,
  height: 200,
  enableHapticFeedback: true,
  enableSoundEffect: false,
  sliderTrackHeight: 4,
  sliderThumbSize: 50,
  timeout: 30000,
});
```

## 注意事项

1. SDK为基本可用版本，可能存在未发现的问题
2. 需要网络权限才能正常工作
3. 请根据实际API接口调整使用方式
4. 生产环境使用前请充分测试
5. 建议合理配置超时时间

## 许可证

MIT License
