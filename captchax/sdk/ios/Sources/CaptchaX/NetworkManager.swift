import Foundation

public enum HTTPMethod: String {
    case GET
    case POST
    case PUT
    case DELETE
    case PATCH
}

public class NetworkManager {
    public static let shared = NetworkManager()
    
    private let session: URLSession
    private let cacheManager = CacheManager.shared
    private var signingKey: String?

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30.0
        config.timeoutIntervalForResource = 60.0
        config.requestCachePolicy = .reloadIgnoringLocalCacheData
        self.session = URLSession(configuration: config)
    }

    public func setSigningKey(_ key: String) {
        self.signingKey = key
    }

    public func request<T: Decodable>(
        _ endpoint: String,
        method: HTTPMethod = .GET,
        params: [String: Any]? = nil
    ) async throws -> T {
        guard let url = URL(string: endpoint) else {
            throw CaptchaXError.invalidConfig
        }

        var request = URLRequest(url: url)
        request.httpMethod = method.rawValue
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("CaptchaX-iOS-SDK/1.0.0", forHTTPHeaderField: "User-Agent")

        if let params = params, method != .GET {
            request.httpBody = try? JSONSerialization.data(withJSONObject: params)
        }

        request = signRequest(request)

        do {
            let (data, response) = try await session.data(for: request)

            guard let httpResponse = response as? HTTPURLResponse else {
                throw CaptchaXError.networkError(underlying: NSError(domain: "NetworkManager", code: -1))
            }

            guard (200...299).contains(httpResponse.statusCode) else {
                let message = String(data: data, encoding: .utf8) ?? "Unknown error"
                throw CaptchaXError.serverError(code: httpResponse.statusCode, message: message)
            }

            let decoder = JSONDecoder()
            decoder.dateDecodingStrategy = .iso8601
            return try decoder.decode(T.self, from: data)
        } catch let error as CaptchaXError {
            throw error
        } catch is DecodingError {
            throw CaptchaXError.verificationFailed
        } catch {
            throw CaptchaXError.networkError(underlying: error)
        }
    }

    public func upload(data: Data, to endpoint: String) async throws -> UploadResult {
        guard let url = URL(string: endpoint) else {
            throw CaptchaXError.invalidConfig
        }

        var request = URLRequest(url: url)
        request.httpMethod = HTTPMethod.POST.rawValue

        let boundary = UUID().uuidString
        request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")

        var body = Data()
        body.append("--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Disposition: form-data; name=\"file\"; filename=\"captcha.jpg\"\r\n".data(using: .utf8)!)
        body.append("Content-Type: image/jpeg\r\n\r\n".data(using: .utf8)!)
        body.append(data)
        body.append("\r\n--\(boundary)--\r\n".data(using: .utf8)!)

        request.httpBody = body
        request = signRequest(request)

        let (responseData, response) = try await session.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse,
              (200...299).contains(httpResponse.statusCode) else {
            throw CaptchaXError.serverError(code: (response as? HTTPURLResponse)?.statusCode ?? -1, message: "Upload failed")
        }

        let decoder = JSONDecoder()
        return try decoder.decode(UploadResult.self, from: responseData)
    }

    private func signRequest(_ request: URLRequest) -> URLRequest {
        var signedRequest = request

        guard let signingKey = signingKey else {
            return signedRequest
        }

        let timestamp = String(Int(Date().timeIntervalSince1970 * 1000))
        let signature = generateSignature(for: request, timestamp: timestamp, secret: signingKey)

        signedRequest.setValue(timestamp, forHTTPHeaderField: "X-CaptchaX-Timestamp")
        signedRequest.setValue(signature, forHTTPHeaderField: "X-CaptchaX-Signature")

        return signedRequest
    }

    private func generateSignature(for request: URLRequest, timestamp: String, secret: String) -> String {
        let method = request.httpMethod ?? "GET"
        let path = request.url?.path ?? ""
        let body = request.httpBody.flatMap { String(data: $0, encoding: .utf8) } ?? ""

        let stringToSign = "\(method)\(path)\(timestamp)\(body)"
        let key = secret.data(using: .utf8)!

        let signature = stringToSign.hmacSHA256(key: key)
        return signature.base64EncodedString()
    }

    public func downloadImage(from url: String) async throws -> Data {
        if cacheManager.cacheEnabled, let cached = cacheManager.getCachedImage(forKey: url) {
            return cached
        }

        guard let imageURL = URL(string: url) else {
            throw CaptchaXError.invalidConfig
        }

        let (data, response) = try await session.data(from: imageURL)

        guard let httpResponse = response as? HTTPURLResponse,
              (200...299).contains(httpResponse.statusCode) else {
            throw CaptchaXError.networkError(underlying: NSError(domain: "NetworkManager", code: -1))
        }

        if cacheManager.cacheEnabled {
            cacheManager.cacheImage(data, forKey: url)
        }

        return data
    }
}
