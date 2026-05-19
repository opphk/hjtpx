# HJTPX iOS Captcha SDK

iOS平台的验证码SDK，提供滑块、点击等验证码类型的集成。

## 功能特性

- 滑块验证码
- 点击验证码
- 图片缓存优化
- 手势识别
- 自动适配屏幕尺寸
- NSURLSession连接管理
- Swift 5.0+ 完整支持

## 快速开始

### 添加依赖

#### Swift Package Manager

```swift
dependencies: [
    .package(url: "https://github.com/hjtpx/hjtpx-ios-sdk.git", from: "1.0.0")
]
```

#### CocoaPods

```ruby
pod 'HjtpxCaptchaSDK', '~> 1.0.0'
```

### 基本使用

```swift
import HjtpxCaptchaSDK

// 初始化客户端
let config = CaptchaConfig(
    baseURL: "https://your-api-server.com",
    appId: "your-app-id",
    appSecret: "your-app-secret"
)

let client = CaptchaClient(config: config)

// 生成滑块验证码
client.getSliderCaptcha(width: 320, height: 160) { result in
    switch result {
    case .success(let captcha):
        // 加载图片并显示验证码UI
        loadAndShowCaptcha(captcha)
    case .failure(let error):
        // 处理错误
        print("Error: \(error)")
    }
}

// 验证滑块位置
client.verifySliderCaptcha(
    sessionId: sessionId,
    x: sliderX,
    y: sliderY
) { result in
    switch result {
    case .success(let verifyResult):
        if verifyResult.success {
            // 验证成功
        } else {
            // 验证失败
        }
    case .failure(let error):
        // 处理错误
        print("Error: \(error)")
    }
}
```

### UIKit 集成

```swift
import UIKit
import HjtpxCaptchaSDK

class CaptchaViewController: UIViewController {
    private let captchaClient = CaptchaClient(config: config)
    private var sessionId: String?

    @IBOutlet weak var backgroundImageView: UIImageView!
    @IBOutlet weak var sliderView: UIImageView!
    @IBOutlet weak var sliderTrackView: UIView!

    private var sliderStartX: CGFloat = 0

    override func viewDidLoad() {
        super.viewDidLoad()
        setupGestures()
        loadCaptcha()
    }

    private func loadCaptcha() {
        captchaClient.getSliderCaptcha(width: 320, height: 160) { [weak self] result in
            switch result {
            case .success(let captcha):
                self?.sessionId = captcha.sessionId
                self?.displayCaptcha(captcha)
            case .failure(let error):
                print("Failed to load captcha: \(error)")
            }
        }
    }

    private func displayCaptcha(_ captcha: SliderCaptchaResponse) {
        // 加载并显示背景图和滑块图
        loadImage(from: captcha.imageUrl) { [weak self] image in
            self?.backgroundImageView.image = image
        }

        loadImage(from: captcha.puzzleUrl) { [weak self] image in
            self?.sliderView.image = image
        }
    }

    private func setupGestures() {
        let panGesture = UIPanGestureRecognizer(target: self, action: #selector(handlePan(_:)))
        sliderView.addGestureRecognizer(panGesture)
        sliderView.isUserInteractionEnabled = true
    }

    @objc private func handlePan(_ gesture: UIPanGestureRecognizer) {
        let translation = gesture.translation(in: sliderTrackView)

        switch gesture.state {
        case .began:
            sliderStartX = sliderView.frame.origin.x
        case .changed:
            var newX = sliderStartX + translation.x
            newX = max(0, min(newX, sliderTrackView.frame.width - sliderView.frame.width))
            sliderView.frame.origin.x = newX
        case .ended:
            let finalX = Int(sliderView.frame.origin.x)
            verifySlider(x: finalX)
        default:
            break
        }
    }

    private func verifySlider(x: Int) {
        guard let sessionId = sessionId else { return }

        captchaClient.verifySliderCaptcha(
            sessionId: sessionId,
            x: x,
            y: nil
        ) { [weak self] result in
            switch result {
            case .success(let verifyResult):
                if verifyResult.success {
                    self?.showSuccess()
                } else {
                    self?.showFailure()
                }
            case .failure(let error):
                print("Verification failed: \(error)")
            }
        }
    }
}
```

### SwiftUI 集成

```swift
import SwiftUI
import HjtpxCaptchaSDK

struct CaptchaView: View {
    @State private var backgroundImage: UIImage?
    @State private var sliderImage: UIImage?
    @State private var sliderOffset: CGFloat = 0
    @State private var sessionId: String?

    private let captchaClient = CaptchaClient(config: config)

    var body: some View {
        VStack {
            Image(uiImage: backgroundImage ?? UIImage())
                .resizable()
                .frame(width: 320, height: 160)

            GeometryReader { geometry in
                ZStack(alignment: .leading) {
                    Rectangle()
                        .fill(Color.gray.opacity(0.3))
                        .frame(height: 40)

                    Image(uiImage: sliderImage ?? UIImage())
                        .resizable()
                        .frame(width: 50, height: 40)
                        .offset(x: sliderOffset)
                        .gesture(
                            DragGesture()
                                .onChanged { value in
                                    sliderOffset = max(0, min(value.translation.width, geometry.size.width - 50))
                                }
                                .onEnded { value in
                                    verifySlider(x: Int(sliderOffset))
                                }
                        )
                }
            }
            .frame(height: 40)
            .padding(.horizontal, 20)
        }
        .onAppear {
            loadCaptcha()
        }
    }

    private func loadCaptcha() {
        captchaClient.getSliderCaptcha(width: 320, height: 160) { result in
            if case .success(let captcha) = result {
                sessionId = captcha.sessionId
                loadImage(from: captcha.imageUrl) { image in
                    backgroundImage = image
                }
                loadImage(from: captcha.puzzleUrl) { image in
                    sliderImage = image
                }
            }
        }
    }

    private func loadImage(from url: String, completion: @escaping (UIImage?) -> Void) {
        guard let imageURL = URL(string: url) else {
            completion(nil)
            return
        }

        URLSession.shared.dataTask(with: imageURL) { data, _, _ in
            if let data = data, let image = UIImage(data: data) {
                DispatchQueue.main.async {
                    completion(image)
                }
            } else {
                DispatchQueue.main.async {
                    completion(nil)
                }
            }
        }.resume()
    }

    private func verifySlider(x: Int) {
        guard let sessionId = sessionId else { return }

        captchaClient.verifySliderCaptcha(
            sessionId: sessionId,
            x: x,
            y: nil
        ) { result in
            if case .success(let verifyResult) = result {
                print("Verification: \(verifyResult.success)")
            }
        }
    }
}
```

### 配置选项

```swift
let config = CaptchaConfig(
    baseURL: "https://your-api-server.com",
    appId: "your-app-id",
    appSecret: "your-app-secret"
)

config.connectionTimeout = 10.0  // 秒
config.readTimeout = 30.0        // 秒
config.maxRetries = 3
config.retryDelay = 1.0         // 秒
```

## 注意事项

1. SDK需要网络权限才能正常工作
2. 需要在Info.plist中添加App Transport Security配置
3. 请根据实际API接口调整使用方式
4. 生产环境使用前请充分测试

## 许可证

MIT License
