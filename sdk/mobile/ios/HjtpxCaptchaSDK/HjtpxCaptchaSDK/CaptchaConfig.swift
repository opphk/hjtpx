import Foundation

public class CaptchaConfig {
    public static var shared = CaptchaConfig()

    public var captchaWidth: Int = 320
    public var captchaHeight: Int = 200
    public var enableHapticFeedback: Bool = true
    public var enableSoundEffect: Bool = false
    public var sliderTrackHeight: Int = 4
    public var sliderThumbSize: Int = 50
    public var timeout: TimeInterval = 30.0

    private init() {}

    public func reset() {
        captchaWidth = 320
        captchaHeight = 200
        enableHapticFeedback = true
        enableSoundEffect = false
        sliderTrackHeight = 4
        sliderThumbSize = 50
        timeout = 30.0
    }
}
