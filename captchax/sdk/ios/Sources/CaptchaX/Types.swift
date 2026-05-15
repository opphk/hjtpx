import Foundation

public enum CaptchaType: String {
    case slider
    case click
    case rotate
    case puzzle
    case text
    case icon

    public var displayName: String {
        switch self {
        case .slider: return "滑块验证"
        case .click: return "点选验证"
        case .rotate: return "旋转验证"
        case .puzzle: return "拼图验证"
        case .text: return "文字验证"
        case .icon: return "图标验证"
        }
    }
}

public enum CaptchaXError: Error, LocalizedError {
    case notInitialized
    case networkError(underlying: Error)
    case serverError(code: Int, message: String)
    case timeout
    case invalidConfig
    case verificationFailed
    case cancelled

    public var errorDescription: String? {
        switch self {
        case .notInitialized:
            return "CaptchaX has not been initialized. Please call initialize() first."
        case .networkError(let underlying):
            return "Network error: \(underlying.localizedDescription)"
        case .serverError(let code, let message):
            return "Server error [\(code)]: \(message)"
        case .timeout:
            return "Request timed out"
        case .invalidConfig:
            return "Invalid configuration"
        case .verificationFailed:
            return "Verification failed"
        case .cancelled:
            return "Verification was cancelled"
        }
    }
}

public struct CaptchaResult: Codable {
    public let token: String
    public let expiresAt: Date?
    public let metadata: [String: String]?

    public init(token: String, expiresAt: Date? = nil, metadata: [String: String]? = nil) {
        self.token = token
        self.expiresAt = expiresAt
        self.metadata = metadata
    }

    public var isValid: Bool {
        guard let expiresAt = expiresAt else { return true }
        return expiresAt > Date()
    }
}

public struct UploadResult: Codable {
    public let imageId: String
    public let url: String?

    public init(imageId: String, url: String? = nil) {
        self.imageId = imageId
        self.url = url
    }
}
