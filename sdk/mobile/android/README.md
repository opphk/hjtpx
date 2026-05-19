# HJTPX Android Captcha SDK

Android平台的验证码SDK，提供滑块、点击等验证码类型的集成。

## 功能特性

- 滑块验证码
- 点击验证码
- 图片预加载优化
- 触摸反馈
- 自动适配屏幕尺寸
- OkHttp连接池管理

## 快速开始

### 添加依赖

```groovy
dependencies {
    implementation 'com.hjtpx:captcha-sdk:1.0.0'
}
```

### 权限配置

```xml
<uses-permission android:name="android.permission.INTERNET" />
<uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />
```

### 基本使用

```java
// 初始化客户端
CaptchaClient client = new CaptchaClient(
    context,
    "https://your-api-server.com",
    "your-app-id",
    "your-app-secret"
);

// 生成滑块验证码
client.generateSliderCaptcha(320, 200, new CaptchaClient.CaptchaCallback<CaptchaClient.SliderCaptchaResult>() {
    @Override
    public void onSuccess(CaptchaClient.SliderCaptchaResult result) {
        // 加载图片并显示验证码UI
        loadAndShowCaptcha(result);
    }

    @Override
    public void onError(String error) {
        // 处理错误
    }
});

// 验证滑块位置
client.verifySliderCaptcha(sessionId, sliderX, new CaptchaClient.CaptchaCallback<CaptchaClient.VerifyResult>() {
    @Override
    public void onSuccess(CaptchaClient.VerifyResult result) {
        if (result.success) {
            // 验证成功
        } else {
            // 验证失败
        }
    }

    @Override
    public void onError(String error) {
        // 处理错误
    }
});
```

### 图片预加载

```java
CaptchaImageLoader loader = new CaptchaImageLoader(client);

// 预加载验证码图片
loader.preloadImage(captchaResult.backgroundImage);
loader.preloadImage(captchaResult.sliderImage);

// 加载图片
loader.loadImage(captchaResult.backgroundImage, new CaptchaImageLoader.ImageCallback() {
    @Override
    public void onSuccess(Bitmap bitmap) {
        imageView.setImageBitmap(bitmap);
    }

    @Override
    public void onError(String error) {
        // 处理错误
    }
});
```

### 配置选项

```java
CaptchaConfig config = new CaptchaConfig(context);
config.setCaptchaWidth(320);
config.setCaptchaHeight(200);
config.setEnableHapticFeedback(true);
config.setEnableSoundEffect(false);
config.setSliderTrackHeight(40);
config.setSliderThumbSize(50);
```

## 注意事项

1. SDK为基本可用版本，可能存在未发现的问题
2. 需要网络权限才能正常工作
3. 请根据实际API接口调整使用方式
4. 生产环境使用前请充分测试
5. 建议合理配置超时时间

## 许可证

MIT License
