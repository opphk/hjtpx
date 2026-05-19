import Foundation

public class HjtpxSDK {
    public static let shared = HjtpxSDK()
    
    private var apiKey: String
    private var apiSecret: String
    private var serverURL: String
    private var timeout: TimeInterval
    private var language: String
    
    public enum CaptchaType: String {
        case slider = "slider"
        case click = "click"
        case rotate = "rotate"
        case voice = "voice"
        case gesture = "gesture"
    }
    
    public enum Language: String {
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
    
    private init() {
        self.apiKey = ""
        self.apiSecret = ""
        self.serverURL = "https://your-domain.com"
        self.timeout = 30.0
        self.language = Language.english.rawValue
    }
    
    public func configure(apiKey: String, apiSecret: String, serverURL: String) {
        self.apiKey = apiKey
        self.apiSecret = apiSecret
        self.serverURL = serverURL
    }
    
    public func setLanguage(_ language: Language) {
        self.language = language.rawValue
    }
    
    public func setTimeout(_ timeout: TimeInterval) {
        self.timeout = timeout
    }
    
    public func getCaptcha(type: CaptchaType, appId: String, completion: @escaping (Result<CaptchaResponse, Error>) -> Void) {
        let url = "\(serverURL)/api/v1/captcha/get"
        
        guard let requestURL = URL(string: url) else {
            completion(.failure(HjtpxError.invalidURL))
            return
        }
        
        var request = URLRequest(url: requestURL)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue(apiKey, forHTTPHeaderField: "X-API-Key")
        request.timeoutInterval = timeout
        
        let parameters: [String: Any] = [
            "captcha_type": `type`.rawValue,
            "app_id": appId,
            "language": language,
            "timestamp": Int(Date().timeIntervalSince1970 * 1000)
        ]
        
        do {
            request.httpBody = try JSONSerialization.data(withJSONObject: parameters)
        } catch {
            completion(.failure(error))
            return
        }
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(HjtpxError.noData))
                return
            }
            
            do {
                let response = try JSONDecoder().decode(CaptchaResponse.self, from: data)
                completion(.success(response))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
    
    public func verifyCaptcha(captchaId: String, token: String, appId: String, completion: @escaping (Result<VerifyResponse, Error>) -> Void) {
        let url = "\(serverURL)/api/v1/captcha/verify"
        
        guard let requestURL = URL(string: url) else {
            completion(.failure(HjtpxError.invalidURL))
            return
        }
        
        var request = URLRequest(url: requestURL)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue(apiKey, forHTTPHeaderField: "X-API-Key")
        request.timeoutInterval = timeout
        
        let parameters: [String: Any] = [
            "captcha_id": captchaId,
            "token": token,
            "app_id": appId,
            "timestamp": Int(Date().timeIntervalSince1970 * 1000)
        ]
        
        do {
            request.httpBody = try JSONSerialization.data(withJSONObject: parameters)
        } catch {
            completion(.failure(error))
            return
        }
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(HjtpxError.noData))
                return
            }
            
            do {
                let response = try JSONDecoder().decode(VerifyResponse.self, from: data)
                completion(.success(response))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
    
    public func reportResult(captchaId: String, result: Bool, appId: String) {
        let url = "\(serverURL)/api/v1/captcha/report"
        
        guard let requestURL = URL(string: url) else {
            return
        }
        
        var request = URLRequest(url: requestURL)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue(apiKey, forHTTPHeaderField: "X-API-Key")
        
        let parameters: [String: Any] = [
            "captcha_id": captchaId,
            "result": result,
            "app_id": appId,
            "timestamp": Int(Date().timeIntervalSince1970 * 1000)
        ]
        
        do {
            request.httpBody = try JSONSerialization.data(withJSONObject: parameters)
        } catch {
            return
        }
        
        URLSession.shared.dataTask(with: request).resume()
    }
}

public struct CaptchaResponse: Codable {
    public let captchaId: String
    public let captchaType: String
    public let data: CaptchaData?
    public let code: Int
    public let message: String
    
    enum CodingKeys: String, CodingKey {
        case captchaId = "captcha_id"
        case captchaType = "captcha_type"
        case data
        case code
        case message
    }
}

public struct CaptchaData: Codable {
    public let backgroundImage: String?
    public let sliderImage: String?
    public let targetPosition: Int?
    public let hintText: String?
    
    enum CodingKeys: String, CodingKey {
        case backgroundImage = "background_image"
        case sliderImage = "slider_image"
        case targetPosition = "target_position"
        case hintText = "hint_text"
    }
}

public struct VerifyResponse: Codable {
    public let success: Bool
    public let captchaId: String
    public let score: Double?
    public let message: String
    public let verifyId: String?
    
    enum CodingKeys: String, CodingKey {
        case success
        case captchaId = "captcha_id"
        case score
        case message
        case verifyId = "verify_id"
    }
}

public enum HjtpxError: Error, LocalizedError {
    case invalidURL
    case noData
    case invalidResponse
    case networkError(String)
    
    public var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "Invalid URL"
        case .noData:
            return "No data received"
        case .invalidResponse:
            return "Invalid response"
        case .networkError(let message):
            return message
        }
    }
}
