import UIKit
import WebKit

public protocol HjtpxCaptchaViewDelegate: AnyObject {
    func captchaViewDidVerify(_ captchaView: HjtpxCaptchaView, verifyId: String)
    func captchaViewDidFail(_ captchaView: HjtpxCaptchaView, error: Error)
    func captchaViewDidClose(_ captchaView: HjtpxCaptchaView)
}

public class HjtpxCaptchaView: UIView {
    public weak var delegate: HjtpxCaptchaViewDelegate?
    
    private var webView: WKWebView!
    private var captchaId: String?
    private var appId: String?
    private var captchaType: HjtpxSDK.CaptchaType
    private var serverURL: String
    private var language: String
    
    private let loadingIndicator: UIActivityIndicatorView = {
        let indicator = UIActivityIndicatorView(style: .large)
        indicator.hidesWhenStopped = true
        indicator.color = .systemGray
        return indicator
    }()
    
    private let closeButton: UIButton = {
        let button = UIButton(type: .system)
        button.setImage(UIImage(systemName: "xmark.circle.fill"), for: .normal)
        button.tintColor = .systemGray
        return button
    }()
    
    public init(frame: CGRect, captchaType: HjtpxSDK.CaptchaType, appId: String, serverURL: String) {
        self.captchaType = captchaType
        self.appId = appId
        self.serverURL = serverURL
        self.language = "en-US"
        super.init(frame: frame)
        setupUI()
    }
    
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }
    
    private func setupUI() {
        backgroundColor = .white
        layer.cornerRadius = 12
        layer.shadowColor = UIColor.black.cgColor
        layer.shadowOffset = CGSize(width: 0, height: 4)
        layer.shadowRadius = 8
        layer.shadowOpacity = 0.15
        
        let config = WKWebViewConfiguration()
        config.preferences.javaScriptEnabled = true
        config.allowsInlineMediaPlayback = true
        
        webView = WKWebView(frame: bounds, configuration: config)
        webView.autoresizingMask = [.flexibleWidth, .flexibleHeight]
        webView.navigationDelegate = self
        addSubview(webView)
        
        addSubview(loadingIndicator)
        addSubview(closeButton)
        
        closeButton.addTarget(self, action: #selector(closeTapped), for: .touchUpInside)
        
        loadingIndicator.startAnimating()
    }
    
    public override func layoutSubviews() {
        super.layoutSubviews()
        
        closeButton.frame = CGRect(x: bounds.width - 44, y: 8, width: 32, height: 32)
        webView.frame = bounds
        loadingIndicator.center = CGPoint(x: bounds.width / 2, y: bounds.height / 2)
    }
    
    public func setLanguage(_ language: String) {
        self.language = language
    }
    
    public func loadCaptcha() {
        let captchaURL = buildCaptchaURL()
        guard let url = URL(string: captchaURL) else { return }
        webView.load(URLRequest(url: url))
    }
    
    private func buildCaptchaURL() -> String {
        let encodedAppId = appId?.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? ""
        let encodedType = captchaType.rawValue.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? ""
        let encodedLang = language.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? ""
        
        return "\(serverURL)/captcha?app_id=\(encodedAppId)&type=\(encodedType)&lang=\(encodedLang)"
    }
    
    @objc private func closeTapped() {
        delegate?.captchaViewDidClose(self)
    }
    
    public func setCaptchaId(_ captchaId: String) {
        self.captchaId = captchaId
    }
}

extension HjtpxCaptchaView: WKNavigationDelegate {
    public func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        loadingIndicator.stopAnimating()
    }
    
    public func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        loadingIndicator.stopAnimating()
        delegate?.captchaViewDidFail(self, error: error)
    }
    
    public func webView(_ webView: WKWebView, decidePolicyFor navigationResponse: WKNavigationResponse, decisionHandler: @escaping (WKNavigationResponsePolicy) -> Void) {
        
        guard let url = navigationResponse.response.url else {
            decisionHandler(.allow)
            return
        }
        
        if url.scheme == "hjtpx" {
            handleCustomURL(url)
            decisionHandler(.cancel)
            return
        }
        
        decisionHandler(.allow)
    }
    
    private func handleCustomURL(_ url: URL) {
        guard let host = url.host else { return }
        
        switch host {
        case "verify":
            if let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
               let queryItems = components.queryItems,
               let verifyId = queryItems.first(where: { $0.name == "verify_id" })?.value {
                delegate?.captchaViewDidVerify(self, verifyId: verifyId)
            }
            
        case "close":
            delegate?.captchaViewDidClose(self)
            
        case "error":
            if let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
               let queryItems = components.queryItems,
               let errorMessage = queryItems.first(where: { $0.name == "message" })?.value {
                let error = NSError(domain: "HjtpxCaptcha", code: -1, userInfo: [NSLocalizedDescriptionKey: errorMessage])
                delegate?.captchaViewDidFail(self, error: error)
            }
            
        default:
            break
        }
    }
}
