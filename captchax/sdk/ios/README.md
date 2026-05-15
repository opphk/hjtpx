# CaptchaX iOS SDK

Modern captcha verification SDK for iOS applications with support for Swift and Objective-C.

## Features

- **Multiple Captcha Types**: Slider, click, rotate, puzzle, text, and icon verification
- **Swift & Objective-C Support**: Full interoperability between Swift and Objective-C
- **SwiftUI & UIKit**: Native support for both UI frameworks
- **Device Fingerprinting**: Advanced device identification and tracking
- **Smart Caching**: Efficient image and data caching
- **Preloading**: Fast captcha loading with preloading support
- **Easy Integration**: Simple API with comprehensive documentation

## Requirements

- iOS 13.0+
- Swift 5.0+
- Xcode 12.0+

## Installation

### CocoaPods

Add the following to your `Podfile`:

```ruby
platform :ios, '13.0'
use_frameworks!

target 'YourApp' do
  pod 'CaptchaX', :git => 'https://github.com/opphk/hjtpx.git', :branch => 'main'
end
```

Then run:

```bash
pod install
```

### Swift Package Manager

Add the following to your `Package.swift`:

```swift
dependencies: [
    .package(url: "https://github.com/opphk/hjtpx.git", from: "1.0.0")
]
```

Or add it via Xcode:
1. File > Swift Packages > Add Package Dependency
2. Enter: `https://github.com/opphk/hjtpx.git`
3. Select version 1.0.0+

### Manual Installation

1. Download or clone the SDK
2. Drag the `Sources/CaptchaX` folder into your Xcode project
3. Add SnapKit as a dependency (via SPM or CocoaPods)

## Quick Start

### Swift (UIKit)

```swift
import CaptchaX

class ViewController: UIViewController, CaptchaXDelegate {
    override func viewDidLoad() {
        super.viewDidLoad()

        // Initialize SDK
        CaptchaX.shared.delegate = self
        CaptchaX.shared.initialize(apiKey: "your_api_key", apiSecret: "your_api_secret")
    }

    func verifyCaptcha() {
        CaptchaX.shared.verify(scene: "login") { result in
            switch result {
            case .success(let token):
                print("Verified: \(token.token)")
            case .failure(let error):
                print("Error: \(error)")
            }
        }
    }

    // CaptchaXDelegate
    func captchaX(_ captcha: CaptchaX, didSuccess result: CaptchaResult) {
        print("Success: \(result.token)")
    }

    func captchaX(_ captcha: CaptchaX, didFailed error: Error) {
        print("Failed: \(error)")
    }

    func captchaXDidClose(_ captcha: CaptchaX) {
        print("Captcha closed")
    }
}
```

### Swift (SwiftUI)

```swift
import SwiftUI
import CaptchaX

struct ContentView: View {
    @State private var isVerified = false

    var body: some View {
        Button(action: {
            CaptchaX.shared.verify(scene: "login") { result in
                switch result {
                case .success(let token):
                    isVerified = true
                    print("Verified: \(token.token)")
                case .failure(let error):
                    print("Error: \(error)")
                }
            }
        }) {
            Text(isVerified ? "已验证" : "验证")
        }
    }
}
```

### Objective-C

```objc
#import <CaptchaX/CaptchaX-Swift.h>

@interface ViewController () <CaptchaXDelegate>
@end

@implementation ViewController

- (void)viewDidLoad {
    [super viewDidLoad];

    [[CaptchaXObjC shared] setDelegate:self];
    [[CaptchaXObjC shared] initializeWithApiKey:@"your_api_key"
                                      apiSecret:@"your_api_secret"];
}

- (void)verifyCaptcha {
    [[CaptchaXObjC shared] verifyWithScene:@"login" completion:^(CaptchaResult *result, NSError *error) {
        if (result) {
            NSLog(@"Verified: %@", result.token);
        } else {
            NSLog(@"Error: %@", error.localizedDescription);
        }
    }];
}

#pragma mark - CaptchaXDelegate

- (void)captchaXDidSuccess:(CaptchaResult *)result {
    NSLog(@"Success: %@", result.token);
}

- (void)captchaXDidFailed:(NSError *)error {
    NSLog(@"Failed: %@", error.localizedDescription);
}

- (void)captchaXDidClose {
    NSLog(@"Captcha closed");
}

@end
```

## CaptchaView (Embedded Widget)

For more control, you can use `CaptchaView` directly:

```swift
let captchaView = CaptchaView()
captchaView.frame = CGRect(x: 0, y: 0, width: 350, height: 400)

captchaView.onSuccess = { token in
    print("Success: \(token)")
}

captchaView.onError = { error in
    print("Error: \(error)")
}

captchaView.onClose = {
    print("Closed")
}

view.addSubview(captchaView)
captchaView.load(captchaType: .slider)
```

## API Reference

### CaptchaX (Swift)

#### Properties

- `shared: CaptchaX` - Shared instance
- `config: CaptchaConfig` - SDK configuration
- `delegate: CaptchaXDelegate?` - Delegate for callbacks

#### Methods

```swift
// Initialize the SDK
func initialize(apiKey: String, apiSecret: String)

// Verify without showing UI
func verify(scene: String, completion: @escaping (Result<CaptchaResult, Error>) -> Void)

// Verify with specific captcha type
func verifyWithView(scene: String, captchaType: CaptchaType, completion: @escaping (Result<CaptchaResult, Error>) -> Void)

// Preload captcha for faster display
func preload(scene: String)

// Get device fingerprint
func getDeviceFingerprint() -> String

// Clean up resources
func destroy()
```

### CaptchaXObjC (Objective-C)

```objc
// Initialize
- (void)initializeWithApiKey:(NSString *)apiKey apiSecret:(NSString *)apiSecret;

// Verify
- (void)verifyWithScene:(NSString *)scene completion:(void (^)(CaptchaResult *, NSError *))completion;

// Configuration
- (void)setDebugModeEnabled:(BOOL)enabled;
- (void)setCacheEnabled:(BOOL)enabled;

// Clean up
- (void)destroy;
```

### CaptchaConfig

```swift
public struct CaptchaConfig {
    public var apiKey: String
    public var apiSecret: String
    public var serverURL: String       // Default: "https://api.captchax.com"
    public var timeout: TimeInterval    // Default: 30.0
    public var cacheEnabled: Bool        // Default: true
    public var preloadEnabled: Bool      // Default: true
}
```

### CaptchaType

```swift
public enum CaptchaType: String {
    case slider    // 滑块验证
    case click     // 点选验证
    case rotate    // 旋转验证
    case puzzle    // 拼图验证
    case text      // 文字验证
    case icon      // 图标验证
}
```

### CaptchaResult

```swift
public struct CaptchaResult: Codable {
    public let token: String           // Verification token
    public let expiresAt: Date?         // Token expiration time
    public let metadata: [String: String]? // Additional metadata
}
```

### Error Types

```swift
public enum CaptchaXError: Error {
    case notInitialized
    case networkError(underlying: Error)
    case serverError(code: Int, message: String)
    case timeout
    case invalidConfig
    case verificationFailed
    case cancelled
}
```

## Configuration

### CaptchaConfig Example

```swift
let config = CaptchaConfig(
    apiKey: "your_api_key",
    apiSecret: "your_api_secret",
    serverURL: "https://api.captchax.com",
    timeout: 30.0,
    cacheEnabled: true,
    preloadEnabled: true
)

CaptchaX.shared.initialize(apiKey: config.apiKey, apiSecret: config.apiSecret)
```

## Device Fingerprinting

The SDK automatically collects device information for security:

```swift
// Get fingerprint string
let fingerprint = CaptchaX.shared.getDeviceFingerprint()

// Collect device info
let deviceInfo = DeviceFingerprint.collect()
// Returns: deviceId, deviceName, systemName, systemVersion,
//          model, screenBounds, processorCount, memoryTotal, etc.
```

## Caching

The SDK includes automatic caching for better performance:

```swift
// Clear cache
CacheManager.shared.clearCache()

// Check cache size
let cacheSize = CacheManager.shared.cacheSize()

// Disable caching
CacheManager.shared.cacheEnabled = false
```

## Logging

Enable debug logging during development:

```swift
// Enable debug mode
Logger.shared.isEnabled = true
Logger.shared.minimumLevel = .debug

// Log messages
Logger.debug("Debug message")
Logger.info("Info message")
Logger.warning("Warning message")
Logger.error("Error message")
```

## Examples

See the `Examples/` folder for complete implementations:

- `SwiftUIExample/` - SwiftUI app example
- `UIKitExample/` - UIKit app example

## Troubleshooting

### Common Issues

1. **SDK not initialized**
   - Ensure `initialize()` is called before any verification

2. **Verification always fails**
   - Check your API key and secret
   - Verify network connectivity
   - Check server URL configuration

3. **CaptchaView not displaying**
   - Ensure the view is added to a visible window
   - Check frame bounds are set correctly

4. **Memory issues**
   - Call `destroy()` when done with the SDK
   - Clear cache periodically with `clearCache()`

## License

MIT License - see LICENSE file for details

## Support

- Email: captchax@example.com
- GitHub Issues: https://github.com/opphk/hjtpx/issues
- Documentation: https://docs.captchax.com

## Version History

### 1.0.0 (2026-05-15)
- Initial release
- Support for multiple captcha types
- Swift and Objective-C support
- SwiftUI and UIKit integration
- Device fingerprinting
- Smart caching
