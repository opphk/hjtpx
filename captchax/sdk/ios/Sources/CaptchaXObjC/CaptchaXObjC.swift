import Foundation

@objc public class CaptchaXObjC: NSObject {
    @objc public static let shared = CaptchaXObjC()

    private var config: CaptchaConfig
    @objc public weak var delegate: CaptchaXDelegate?

    private override init() {
        self.config = CaptchaConfig()
        super.init()
    }

    @objc public func initialize(apiKey: String, apiSecret: String) {
        config.apiKey = apiKey
        config.apiSecret = apiSecret
        CaptchaX.shared.initialize(apiKey: apiKey, apiSecret: apiSecret)
    }

    @objc public func initializeWithServerURL(apiKey: String, apiSecret: String, serverURL: String) {
        config.apiKey = apiKey
        config.apiSecret = apiSecret
        config.serverURL = serverURL
        CaptchaX.shared.initialize(apiKey: apiKey, apiSecret: apiSecret)
    }

    @objc public func verify(scene: String, completion: @escaping (CaptchaResult?, Error?) -> Void) {
        CaptchaX.shared.verify(scene: scene) { result in
            switch result {
            case .success(let captchaResult):
                self.delegate?.captchaX(CaptchaX.shared, didSuccess: captchaResult)
                completion(captchaResult, nil)
            case .failure(let error):
                self.delegate?.captchaX(CaptchaX.shared, didFailed: error)
                completion(nil, error)
            }
        }
    }

    @objc public func verifyWithType(scene: String, type: String, completion: @escaping (CaptchaResult?, Error?) -> Void) {
        CaptchaX.shared.verify(scene: scene) { result in
            switch result {
            case .success(let captchaResult):
                completion(captchaResult, nil)
            case .failure(let error):
                completion(nil, error)
            }
        }
    }

    @objc public func setDebugMode(enabled: Bool) {
        Logger.shared.isEnabled = enabled
        Logger.shared.minimumLevel = enabled ? .debug : .warning
    }

    @objc public func setCacheEnabled(enabled: Bool) {
        config.cacheEnabled = enabled
        CacheManager.shared.cacheEnabled = enabled
    }

    @objc public func clearCache() {
        CacheManager.shared.clearCache()
    }

    @objc public func destroy() {
        CaptchaX.shared.destroy()
    }
}
