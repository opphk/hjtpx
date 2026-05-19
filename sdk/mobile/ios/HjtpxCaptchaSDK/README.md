# HJTPX iOS Captcha SDK

iOS平台的验证码SDK，提供滑块、点击等验证码类型的集成。

## 功能特性

- 滑块验证码
- 点击验证码
- UIKit/SwiftUI支持
- 自动布局适配
- 触摸反馈
- 图片异步加载
- 性能优化

## 快速开始

### CocoaPods安装

```ruby
pod 'HjtpxCaptchaSDK', :git => 'https://github.com/hjtpx/hjtpx-sdk.git', :tag => '1.0.0'
```

### Swift使用

```swift
import HjtpxCaptchaSDK

let client = HjtpxCaptchaClient(
    baseUrl: "https://your-api-server.com",
    appId: "your-app-id",
    appSecret: "your-app-secret"
)

// 生成滑块验证码
client.generateSliderCaptcha(width: 320, height: 200) { sessionId, backgroundImageUrl, sliderImageUrl, error in
    if let error = error {
        print("Error: \(error.localizedDescription)")
        return
    }

    // 加载图片并显示验证码UI
    loadAndShowCaptcha(backgroundImageUrl: backgroundImageUrl, sliderImageUrl: sliderImageUrl)
}

// 验证滑块位置
client.verifySliderCaptcha(sessionId: sessionId, x: sliderX) { success, score, message, error in
    if success {
        // 验证成功
    } else {
        // 验证失败
    }
}
```

### UIView集成

```swift
let captchaView = CaptchaImageView(frame: CGRect(x: 0, y: 0, width: 320, height: 250))

captchaView.onSliderCompleted = { progress in
    // 验证滑块位置
    client.verifySliderCaptcha(sessionId: sessionId, x: Float(progress)) { success, score, message, error in
        if success {
            captchaView.showSuccess()
        } else {
            captchaView.showFailure()
        }
    }
}

view.addSubview(captchaView)
```

### 配置选项

```swift
CaptchaConfig.shared.captchaWidth = 320
CaptchaConfig.shared.captchaHeight = 200
CaptchaConfig.shared.enableHapticFeedback = true
CaptchaConfig.shared.enableSoundEffect = false
CaptchaConfig.shared.sliderTrackHeight = 4
CaptchaConfig.shared.sliderThumbSize = 50
CaptchaConfig.shared.timeout = 30.0
```

## 注意事项

1. SDK为基本可用版本，可能存在未发现的问题
2. 需要网络权限才能正常工作
3. 请根据实际API接口调整使用方式
4. 生产环境使用前请充分测试
5. 建议合理配置超时时间

## 许可证

MIT License
