import Foundation

@objc public protocol CaptchaXDelegate: AnyObject {
    func captchaX(_ captcha: CaptchaX, didSuccess result: CaptchaResult)
    func captchaX(_ captcha: CaptchaX, didFailed error: Error)
    func captchaXDidClose(_ captcha: CaptchaX)
}

public struct CaptchaConfig {
    public var apiKey: String
    public var apiSecret: String
    public var serverURL: String
    public var timeout: TimeInterval
    public var cacheEnabled: Bool
    public var preloadEnabled: Bool

    public init(
        apiKey: String = "",
        apiSecret: String = "",
        serverURL: String = "https://api.captchax.com",
        timeout: TimeInterval = 30.0,
        cacheEnabled: Bool = true,
        preloadEnabled: Bool = true
    ) {
        self.apiKey = apiKey
        self.apiSecret = apiSecret
        self.serverURL = serverURL
        self.timeout = timeout
        self.cacheEnabled = cacheEnabled
        self.preloadEnabled = preloadEnabled
    }

    public static var `default`: CaptchaConfig {
        return CaptchaConfig()
    }
}
