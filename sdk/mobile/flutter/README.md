# HJTPX Flutter Captcha SDK

Flutter平台的验证码SDK，提供滑块、点击等验证码类型的集成。

## 功能特性

- 滑块验证码
- 点击验证码
- 图片预加载
- 触摸反馈
- 响应式布局
- 自动缓存管理
- 跨平台支持（iOS/Android）

## 快速开始

### 添加依赖

```yaml
dependencies:
  hjtpx_captcha: ^1.0.0
```

### 基本使用

```dart
import 'package:hjtpx_captcha/hjtpx_captcha.dart';

final client = CaptchaClient(
  baseUrl: 'https://your-api-server.com',
  appId: 'your-app-id',
  appSecret: 'your-app-secret',
);

// 生成滑块验证码
final result = await client.generateSliderCaptcha(
  width: 320,
  height: 200,
);

// 使用滑块组件
SliderCaptchaWidget(
  backgroundImageUrl: result.backgroundImage,
  sliderImageUrl: result.sliderImage,
  onSliderMoved: (progress) {
    // 滑块移动回调
  },
  onSliderCompleted: (progress) async {
    // 验证滑块位置
    final verifyResult = await client.verifySliderCaptcha(
      sessionId: result.sessionId,
      x: progress,
    );

    if (verifyResult.success) {
      // 验证成功
    }
  },
)
```

### 图片预加载

```dart
final loader = ImageLoader();

// 预加载验证码图片
await loader.preloadImages([
  result.backgroundImage,
  result.sliderImage,
]);

// 加载图片
final bytes = await loader.loadImage(result.backgroundImage);
```

### 配置选项

```dart
final config = CaptchaConfig(
  captchaWidth: 320,
  captchaHeight: 200,
  enableHapticFeedback: true,
  enableSoundEffect: false,
  sliderTrackHeight: 4,
  sliderThumbSize: 50,
  timeout: 30,
);

final client = CaptchaClient(
  baseUrl: 'https://your-api-server.com',
  appId: 'your-app-id',
  appSecret: 'your-app-secret',
  config: config,
);
```

## 注意事项

1. SDK为基本可用版本，可能存在未发现的问题
2. 需要网络权限才能正常工作
3. 请根据实际API接口调整使用方式
4. 生产环境使用前请充分测试
5. 建议合理配置超时时间

## 许可证

MIT License
