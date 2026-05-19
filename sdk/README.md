# HJTPX Mobile SDK v15.0

## 概述

HJTPX Mobile SDK 提供跨平台的验证码集成方案，支持 iOS 和 Android 原生开发，以及 Flutter 和 React Native 跨平台框架。

## iOS SDK

### 快速开始

#### 1. 使用 CocoaPods 安装

```ruby
pod 'HjtpxSDK', :git => 'https://github.com/your-org/hjtpx-ios-sdk.git', :tag => 'v15.0'
```

#### 2. 配置 AppDelegate

```swift
import HjtpxSDK

func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
    
    HjtpxSDK.shared.configure(
        apiKey: "YOUR_API_KEY",
        apiSecret: "YOUR_API_SECRET",
        serverURL: "https://your-server.com"
    )
    
    HjtpxSDK.shared.setLanguage(.english)
    HjtpxSDK.shared.setTimeout(30.0)
    
    return true
}
```

#### 3. 显示验证码

```swift
import UIKit
import HjtpxSDK

class LoginViewController: UIViewController, HjtpxCaptchaViewDelegate {
    
    private var captchaView: HjtpxCaptchaView!
    
    func showCaptcha() {
        let rect = CGRect(x: 20, y: 100, width: view.bounds.width - 40, height: 300)
        captchaView = HjtpxCaptchaView(
            frame: rect,
            captchaType: .slider,
            appId: "YOUR_APP_ID",
            serverURL: "https://your-server.com"
        )
        captchaView.delegate = self
        captchaView.setLanguage(HjtpxSDK.shared.language)
        view.addSubview(captchaView)
        captchaView.loadCaptcha()
    }
    
    // MARK: - HjtpxCaptchaViewDelegate
    
    func captchaViewDidVerify(_ captchaView: HjtpxCaptchaView, verifyId: String) {
        print("Verification successful: \(verifyId)")
        captchaView.removeFromSuperview()
        // 验证成功后继续登录流程
    }
    
    func captchaViewDidFail(_ captchaView: HjtpxCaptchaView, error: Error) {
        print("Verification failed: \(error.localizedDescription)")
        // 处理验证失败
    }
    
    func captchaViewDidClose(_ captchaView: HjtpxCaptchaView) {
        captchaView.removeFromSuperview()
    }
}
```

#### 4. 手动验证

```swift
HjtpxSDK.shared.verifyCaptcha(
    captchaId: "captcha_id_from_view",
    token: "user_token",
    appId: "YOUR_APP_ID"
) { result in
    switch result {
    case .success(let response):
        if response.success {
            print("Verify ID: \(response.verifyId ?? "")")
        }
    case .failure(let error):
        print("Error: \(error)")
    }
}
```

### 支持的验证码类型

- `CaptchaType.slider` - 滑块验证码
- `CaptchaType.click` - 点选验证码
- `CaptchaType.rotate` - 旋转验证码
- `CaptchaType.voice` - 语音验证码
- `CaptchaType.gesture` - 手势验证码

### 支持的语言

```swift
enum Language: String {
    case chineseSimplified = "zh-CN"
    case chineseTraditional = "zh-TW"
    case english = "en-US"
    case japanese = "ja-JP"
    case korean = "ko-KR"
    case french = "fr-FR"
    case german = "de-DE"
    case spanish = "es-ES"
    case portuguese = "pt-BR"
    case russian = "ru-RU"
    case arabic = "ar-SA"
    case hindi = "hi-IN"
    case vietnamese = "vi-VN"
    case thai = "th-TH"
    case indonesian = "id-ID"
}
```

## Android SDK

### 快速开始

#### 1. 添加依赖

```gradle
dependencies {
    implementation 'com.hjtpx:captcha:15.0.0'
}
```

#### 2. 初始化 SDK

```kotlin
class MyApplication : Application() {
    
    override fun onCreate() {
        super.onCreate()
        
        HjtpxClient.getInstance(this).apply {
            configure(
                apiKey = "YOUR_API_KEY",
                apiSecret = "YOUR_API_SECRET",
                serverUrl = "https://your-server.com"
            )
            setLanguage("en-US")
            setTimeout(30)
        }
    }
}
```

#### 3. 显示验证码

```kotlin
class LoginActivity : AppCompatActivity(), CaptchaViewDelegate {
    
    private lateinit var captchaView: CaptchaView
    
    private fun showCaptcha() {
        captchaView = CaptchaView(this).apply {
            layoutParams = LayoutParams(
                LayoutParams.MATCH_PARENT,
                400.dp
            )
            setDelegate(this@LoginActivity)
            setCaptchaType(CaptchaType.SLIDER)
            setAppId("YOUR_APP_ID")
            setServerUrl("https://your-server.com")
            setLanguage("en-US")
        }
        
        container.addView(captchaView)
        captchaView.loadCaptcha()
    }
    
    override fun onCaptchaVerified(view: CaptchaView, verifyId: String) {
        Log.d(TAG, "Verification successful: $verifyId")
        container.removeView(view)
        // 验证成功后继续登录流程
    }
    
    override fun onCaptchaError(view: CaptchaView, error: String) {
        Log.e(TAG, "Verification failed: $error")
        // 处理验证失败
    }
    
    override fun onCaptchaClose(view: CaptchaView) {
        container.removeView(view)
    }
}
```

#### 4. 手动验证

```kotlin
HjtpxClient.getInstance(this).verifyCaptcha(
    captchaId = "captcha_id_from_view",
    token = "user_token",
    appId = "YOUR_APP_ID",
    callback = object : VerifyCallback {
        override fun onSuccess(response: VerifyResponse) {
            if (response.success) {
                Log.d(TAG, "Verify ID: ${response.verifyId}")
            }
        }
        
        override fun onError(error: Exception) {
            Log.e(TAG, "Error: ${error.message}")
        }
    }
)
```

## React Native SDK

### 安装

```bash
npm install @hjtpx/react-native-captcha
```

### 使用

```javascript
import HjtpxCaptcha from '@hjtpx/react-native-captcha';

const App = () => {
  const [captchaKey, setCaptchaKey] = useState(0);
  
  const handleVerify = (verifyId) => {
    console.log('Verification successful:', verifyId);
    setCaptchaKey(prev => prev + 1);
  };
  
  const handleError = (error) => {
    console.error('Verification failed:', error);
  };
  
  return (
    <HjtpxCaptcha
      key={captchaKey}
      apiKey="YOUR_API_KEY"
      appId="YOUR_APP_ID"
      serverUrl="https://your-server.com"
      type="slider"
      language="en-US"
      onVerify={handleVerify}
      onError={handleError}
      onClose={() => console.log('Captcha closed')}
    />
  );
};
```

## Flutter SDK

### 安装

```yaml
dependencies:
  hjtpx_captcha: ^15.0.0
```

### 使用

```dart
import 'package:hjtpx_captcha/hjtpx_captcha.dart';

class LoginPage extends StatefulWidget {
  @override
  _LoginPageState createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  @override
  void initState() {
    super.initState();
    HjtpxCaptcha.configure(
      apiKey: 'YOUR_API_KEY',
      appId: 'YOUR_APP_ID',
      serverUrl: 'https://your-server.com',
    );
  }
  
  void _showCaptcha() async {
    final result = await HjtpxCaptcha.show(
      type: CaptchaType.slider,
      language: 'en-US',
    );
    
    if (result != null) {
      print('Verify ID: ${result.verifyId}');
    }
  }
  
  @override
  Widget build(BuildContext context) {
    return ElevatedButton(
      onPressed: _showCaptcha,
      child: Text('Verify'),
    );
  }
}
```

## 常见问题

### 1. 验证码加载失败

- 检查网络连接
- 确认 API Key 和服务器地址配置正确
- 检查服务器是否正常运行

### 2. 验证回调未触发

- 确保正确实现了代理接口
- 检查 WebView 配置是否允许 JavaScript

### 3. 国际化不生效

- 确保传入正确的语言代码
- 检查服务器是否支持该语言

### 4. WebView 兼容性问题

- iOS: 确保设置 `allowsInlineMediaPlayback = true`
- Android: 确保设置 `mixedContentMode = MIXED_CONTENT_ALWAYS_ALLOW`

## 性能优化

### iOS

- 使用 WKWebView 而非 UIWebView
- 预加载验证码资源
- 复用 WebView 实例

### Android

- 启用 WebView 硬件加速
- 合理设置 WebView 缓存模式
- 在 Activity/Fragment 销毁时清理 WebView

## 安全建议

- 勿在前端存储 API Secret
- 使用 HTTPS 传输所有请求
- 验证服务器返回的签名
- 限制验证码使用频率

## 技术支持

- 邮箱: support@hjtpx.com
- 文档: https://docs.hjtpx.com
- GitHub: https://github.com/your-org/hjtpx-sdk

## 版本历史

### v15.0.0

- 新增滑块、点选、旋转、语音、手势验证码支持
- 新增 15 种语言国际化支持
- 优化移动端性能
- 增强安全机制

### v14.0.0

- 新增 React Native 和 Flutter 支持
- 优化 WebView 兼容性
- 修复已知问题

### v13.0.0

- 重构 SDK 架构
- 提升加载速度 50%
- 新增性能监控

## 许可证

MIT License - 详见 LICENSE 文件
