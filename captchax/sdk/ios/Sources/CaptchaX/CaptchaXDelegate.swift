import Foundation

@objc public protocol CaptchaXDelegate: AnyObject {
    func captchaX(_ captcha: CaptchaX, didSuccess result: CaptchaResult)
    func captchaX(_ captcha: CaptchaX, didFailed error: Error)
    func captchaXDidClose(_ captcha: CaptchaX)
}
