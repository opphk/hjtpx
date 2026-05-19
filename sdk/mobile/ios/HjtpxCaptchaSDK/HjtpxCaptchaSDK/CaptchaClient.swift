import Foundation

@objc(HjtpxCaptchaClient)
public class HjtpxCaptchaClient: NSObject {
    private let baseUrl: String
    private let appId: String
    private let appSecret: String
    private let session: URLSession
    private let decoder = JSONDecoder()
    private let encoder = JSONEncoder()

    @objc public var onCaptchaLoaded: ((SliderCaptchaResult) -> Void)?
    @objc public var onCaptchaVerified: ((VerifyResult) -> Void)?
    @objc public var onError: ((String) -> Void)?

    @objc public init(baseUrl: String, appId: String, appSecret: String) {
        self.baseUrl = baseUrl
        self.appId = appId
        self.appSecret = appSecret

        let configuration = URLSessionConfiguration.default
        configuration.timeoutIntervalForRequest = 30
        configuration.timeoutIntervalForResource = 30
        configuration.httpAdditionalHeaders = [
            "User-Agent": "HjtpxCaptcha-iOS/1.0",
            "Content-Type": "application/json"
        ]

        self.session = URLSession(configuration: configuration)
        super.init()
    }

    @objc public func generateSliderCaptcha(width: Int, height: Int, completion: @escaping (Result<SliderCaptchaResult, Error>) -> Void) {
        let url = URL(string: "\(baseUrl)/api/captcha/slider")!

        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let body: [String: Any] = [
            "app_id": appId,
            "captcha_type": "slider",
            "width": width,
            "height": height
        ]

        do {
            request.httpBody = try JSONSerialization.data(withJSONObject: body)
        } catch {
            completion(.failure(error))
            return
        }

        let task = session.dataTask(with: request) { [weak self] data, response, error in
            guard let self = self else { return }

            if let error = error {
                DispatchQueue.main.async {
                    completion(.failure(error))
                }
                return
            }

            guard let data = data else {
                DispatchQueue.main.async {
                    completion(.failure(CaptchaError.noData))
                }
                return
            }

            do {
                let result = try self.decoder.decode(SliderCaptchaResponse.self, from: data)

                let captchaResult = SliderCaptchaResult()
                captchaResult.sessionId = result.sessionId
                captchaResult.backgroundImageUrl = "\(self.baseUrl)\(result.backgroundImage)"
                captchaResult.sliderImageUrl = "\(self.baseUrl)\(result.sliderImage)"

                DispatchQueue.main.async {
                    completion(.success(captchaResult))
                    self.onCaptchaLoaded?(captchaResult)
                }
            } catch {
                DispatchQueue.main.async {
                    completion(.failure(error))
                }
            }
        }

        task.resume()
    }

    @objc public func verifySliderCaptcha(sessionId: String, x: Float, completion: @escaping (Result<VerifyResult, Error>) -> Void) {
        let url = URL(string: "\(baseUrl)/api/captcha/verify/slider")!

        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let body: [String: Any] = [
            "session_id": sessionId,
            "app_id": appId,
            "x": x
        ]

        do {
            request.httpBody = try JSONSerialization.data(withJSONObject: body)
        } catch {
            completion(.failure(error))
            return
        }

        let task = session.dataTask(with: request) { [weak self] data, response, error in
            guard let self = self else { return }

            if let error = error {
                DispatchQueue.main.async {
                    completion(.failure(error))
                }
                return
            }

            guard let data = data else {
                DispatchQueue.main.async {
                    completion(.failure(CaptchaError.noData))
                }
                return
            }

            do {
                let response = try self.decoder.decode(VerifyResponse.self, from: data)

                let result = VerifyResult()
                result.success = response.success
                result.score = response.score ?? 0.0
                result.message = response.message ?? ""

                DispatchQueue.main.async {
                    completion(.success(result))
                    self.onCaptchaVerified?(result)
                }
            } catch {
                DispatchQueue.main.async {
                    completion(.failure(error))
                }
            }
        }

        task.resume()
    }

    @objc public func generateClickCaptcha(count: Int, completion: @escaping (Result<ClickCaptchaResult, Error>) -> Void) {
        let url = URL(string: "\(baseUrl)/api/captcha/click")!

        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let body: [String: Any] = [
            "app_id": appId,
            "captcha_type": "click",
            "count": count
        ]

        do {
            request.httpBody = try JSONSerialization.data(withJSONObject: body)
        } catch {
            completion(.failure(error))
            return
        }

        let task = session.dataTask(with: request) { [weak self] data, response, error in
            guard let self = self else { return }

            if let error = error {
                DispatchQueue.main.async {
                    completion(.failure(error))
                }
                return
            }

            guard let data = data else {
                DispatchQueue.main.async {
                    completion(.failure(CaptchaError.noData))
                }
                return
            }

            do {
                let response = try self.decoder.decode(ClickCaptchaResponse.self, from: data)

                let captchaResult = ClickCaptchaResult()
                captchaResult.sessionId = response.sessionId
                captchaResult.backgroundImageUrl = "\(self.baseUrl)\(response.backgroundImage)"
                captchaResult.targetCount = response.targetCount

                DispatchQueue.main.async {
                    completion(.success(captchaResult))
                }
            } catch {
                DispatchQueue.main.async {
                    completion(.failure(error))
                }
            }
        }

        task.resume()
    }

    @objc public func loadImage(from urlString: String, completion: @escaping (Result<Data, Error>) -> Void) {
        guard let url = URL(string: urlString) else {
            completion(.failure(CaptchaError.invalidUrl))
            return
        }

        let task = session.dataTask(with: url) { data, response, error in
            if let error = error {
                DispatchQueue.main.async {
                    completion(.failure(error))
                }
                return
            }

            guard let data = data else {
                DispatchQueue.main.async {
                    completion(.failure(CaptchaError.noData))
                }
                return
            }

            DispatchQueue.main.async {
                completion(.success(data))
            }
        }

        task.resume()
    }

    @objc public func cancelAllRequests() {
        session.getAllTasks { tasks in
            tasks.forEach { $0.cancel() }
        }
    }
}

public enum CaptchaError: Error {
    case noData
    case invalidUrl
    case parseError
}

@objc public class SliderCaptchaResult: NSObject {
    @objc public var sessionId: String = ""
    @objc public var backgroundImageUrl: String = ""
    @objc public var sliderImageUrl: String = ""
}

@objc public class ClickCaptchaResult: NSObject {
    @objc public var sessionId: String = ""
    @objc public var backgroundImageUrl: String = ""
    @objc public var targetCount: Int = 0
}

@objc public class VerifyResult: NSObject {
    @objc public var success: Bool = false
    @objc public var score: Double = 0.0
    @objc public var message: String = ""
}

struct SliderCaptchaResponse: Codable {
    let sessionId: String
    let backgroundImage: String
    let sliderImage: String

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case backgroundImage = "background_image"
        case sliderImage = "slider_image"
    }
}

struct ClickCaptchaResponse: Codable {
    let sessionId: String
    let backgroundImage: String
    let targetCount: Int

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case backgroundImage = "background_image"
        case targetCount = "target_count"
    }
}

struct VerifyResponse: Codable {
    let success: Bool
    let score: Double?
    let message: String?
}
