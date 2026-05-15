import Foundation

public class CaptchaX {
    public static let shared = CaptchaX()

    public var config: CaptchaConfig
    @objc public weak var delegate: CaptchaXDelegate?

    private var isInitialized = false
    private let fingerprint = DeviceFingerprint.generate()
    private var sessionToken: String?

    private init() {
        self.config = CaptchaConfig.default
    }

    public func initialize(apiKey: String, apiSecret: String) {
        config.apiKey = apiKey
        config.apiSecret = apiSecret
        isInitialized = true

        if config.cacheEnabled {
            CacheManager.shared.cacheEnabled = true
        }

        NetworkManager.shared.setSigningKey(apiSecret)

        Logger.info("CaptchaX initialized with API Key: \(apiKey.prefix(8))...")
    }

    public func verify(scene: String, completion: @escaping (Result<CaptchaResult, Error>) -> Void) {
        guard isInitialized else {
            let error = CaptchaXError.notInitialized
            Logger.error(error.localizedDescription)
            completion(.failure(error))
            return
        }

        guard !config.apiKey.isEmpty else {
            completion(.failure(CaptchaXError.invalidConfig))
            return
        }

        Logger.debug("Starting verification for scene: \(scene)")

        Task {
            do {
                let result = try await performVerification(scene: scene)
                await MainActor.run {
                    self.delegate?.captchaX(self, didSuccess: result)
                    completion(.success(result))
                }
            } catch {
                await MainActor.run {
                    self.delegate?.captchaX(self, didFailed: error)
                    completion(.failure(error))
                }
            }
        }
    }

    public func verifyWithView(scene: String, captchaType: CaptchaType, completion: @escaping (Result<CaptchaResult, Error>) -> Void) {
        verify(scene: scene, completion: completion)
    }

    public func preload(scene: String) {
        guard config.preloadEnabled else { return }
        Logger.debug("Preloading captcha for scene: \(scene)")

        Task {
            do {
                let endpoint = "\(config.serverURL)/api/v1/captcha/preload"
                let params: [String: Any] = [
                    "appId": config.apiKey,
                    "scene": scene,
                    "fingerprint": fingerprint
                ]

                let _: PreloadResponse = try await NetworkManager.shared.request(endpoint, method: .POST, params: params)
                Logger.debug("Preload completed for scene: \(scene)")
            } catch {
                Logger.warning("Preload failed: \(error.localizedDescription)")
            }
        }
    }

    public func destroy() {
        Logger.info("Destroying CaptchaX instance")
        isInitialized = false
        sessionToken = nil
        delegate = nil
        CacheManager.shared.clearCache()
    }

    public func getDeviceFingerprint() -> String {
        return fingerprint
    }

    private func performVerification(scene: String) async throws -> CaptchaResult {
        let endpoint = "\(config.serverURL)/api/v1/captcha/verify"

        let deviceInfo = DeviceFingerprint.collect()
        let params: [String: Any] = [
            "appId": config.apiKey,
            "scene": scene,
            "fingerprint": fingerprint,
            "deviceInfo": deviceInfo,
            "timestamp": Int(Date().timeIntervalSince1970 * 1000)
        ]

        let response: VerifyResponse = try await NetworkManager.shared.request(endpoint, method: .POST, params: params)

        guard response.success, let token = response.token else {
            throw CaptchaXError.verificationFailed
        }

        sessionToken = token
        return CaptchaResult(
            token: token,
            expiresAt: response.expiresAt,
            metadata: response.metadata
        )
    }
}

private struct PreloadResponse: Codable {
    let success: Bool
    let preloadId: String?
}

private struct VerifyResponse: Codable {
    let success: Bool
    let token: String?
    let expiresAt: Date?
    let metadata: [String: String]?
}
