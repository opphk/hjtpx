import UIKit
import WebKit
import SnapKit

public class CaptchaView: UIView {
    public var onSuccess: ((String) -> Void)?
    public var onError: ((Error) -> Void)?
    public var onClose: (() -> Void)?

    private var webView: WKWebView?
    private var loadingView: UIActivityIndicatorView?
    private var closeButton: UIButton?
    private var containerView: UIView?
    private var currentType: CaptchaType = .slider
    private var isLoading = false

    private let containerCornerRadius: CGFloat = 12
    private let shadowRadius: CGFloat = 10
    private let shadowOpacity: Float = 0.3

    public override init(frame: CGRect) {
        super.init(frame: frame)
        setupBaseUI()
    }

    public required init?(coder: NSCoder) {
        super.init(coder: coder)
        setupBaseUI()
    }

    private func setupBaseUI() {
        backgroundColor = UIColor.black.withAlphaComponent(0.5)
        alpha = 0

        containerView = UIView()
        containerView?.backgroundColor = .white
        containerView?.layer.cornerRadius = containerCornerRadius
        containerView?.layer.shadowColor = UIColor.black.cgColor
        containerView?.layer.shadowOffset = CGSize(width: 0, height: 4)
        containerView?.layer.shadowRadius = shadowRadius
        containerView?.layer.shadowOpacity = shadowOpacity
        containerView?.clipsToBounds = false

        if let container = containerView {
            addSubview(container)
            container.snp.makeConstraints { make in
                make.center.equalToSuperview()
                make.width.equalTo(350)
                make.height.equalTo(400)
            }
        }

        setupCloseButton()
        setupLoadingView()
    }

    private func setupCloseButton() {
        closeButton = UIButton(type: .system)
        closeButton?.setImage(UIImage(systemName: "xmark.circle.fill"), for: .normal)
        closeButton?.tintColor = .gray
        closeButton?.addTarget(self, action: #selector(closeTapped), for: .touchUpInside)

        if let container = containerView, let button = closeButton {
            container.addSubview(button)
            button.snp.makeConstraints { make in
                make.top.equalToSuperview().offset(8)
                make.trailing.equalToSuperview().offset(-8)
                make.width.height.equalTo(30)
            }
        }
    }

    private func setupLoadingView() {
        loadingView = UIActivityIndicatorView(style: .large)
        loadingView?.color = .systemBlue
        loadingView?.hidesWhenStopped = true

        if let container = containerView, let loader = loadingView {
            container.addSubview(loader)
            loader.snp.makeConstraints { make in
                make.center.equalToSuperview()
            }
        }
    }

    public func load(captchaType: CaptchaType) {
        currentType = captchaType
        showWithAnimation()

        if CaptchaX.shared.config.cacheEnabled {
            preloadResources(for: captchaType)
        }

        loadCaptchaContent()
    }

    private func loadCaptchaContent() {
        isLoading = true
        loadingView?.startAnimating()

        let config = WKWebViewConfiguration()
        config.preferences.javaScriptEnabled = true
        config.allowsInlineMediaPlayback = true

        webView?.removeFromSuperview()
        webView = WKWebView(frame: containerView?.bounds ?? .zero, configuration: config)
        webView?.navigationDelegate = self
        webView?.scrollView.isScrollEnabled = false
        webView?.backgroundColor = .clear
        webView?.isOpaque = false

        if let container = containerView, let wv = webView {
            container.insertSubview(wv, at: 0)
            wv.snp.makeConstraints { make in
                make.edges.equalToSuperview()
            }
        }

        let baseURL = CaptchaX.shared.config.serverURL
        let widgetURL = "\(baseURL)/widget?type=\(captchaType.rawValue)&appId=\(CaptchaX.shared.config.apiKey)"
        
        if let url = URL(string: widgetURL) {
            let request = URLRequest(url: url)
            webView?.load(request)
        } else {
            loadDemoContent()
        }
    }

    private func loadDemoContent() {
        let demoHTML = generateDemoHTML(for: currentType)
        webView?.loadHTMLString(demoHTML, baseURL: URL(string: CaptchaX.shared.config.serverURL))
    }

    private func generateDemoHTML(for type: CaptchaType) -> String {
        switch type {
        case .slider:
            return """
            <!DOCTYPE html>
            <html>
            <head>
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <style>
                    body { margin: 0; padding: 20px; font-family: -apple-system, sans-serif; text-align: center; background: #f5f5f5; }
                    .container { background: white; border-radius: 8px; padding: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
                    .captcha-area { width: 300px; height: 200px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); margin: 20px auto; border-radius: 8px; position: relative; overflow: hidden; }
                    .slider { width: 50px; height: 50px; background: white; border-radius: 50%; position: absolute; top: 75px; left: 10px; cursor: pointer; box-shadow: 0 2px 5px rgba(0,0,0,0.3); display: flex; align-items: center; justify-content: center; }
                    .slider::after { content: '→'; font-size: 20px; color: #667eea; }
                    .target { width: 40px; height: 40px; background: rgba(255,255,255,0.8); position: absolute; top: 80px; right: 50px; border-radius: 4px; }
                    .track { width: 280px; height: 40px; background: #e0e0e0; margin: 20px auto; border-radius: 20px; position: relative; }
                    .success { display: none; color: #4caf50; font-size: 24px; }
                </style>
            </head>
            <body>
                <div class="container">
                    <h3>滑块验证</h3>
                    <p>请将滑块拖动到最右侧完成验证</p>
                    <div class="captcha-area">
                        <div class="target"></div>
                        <div class="slider" id="slider"></div>
                    </div>
                    <div class="track" id="track"></div>
                    <div class="success" id="success">✓ 验证成功</div>
                </div>
                <script>
                    const slider = document.getElementById('slider');
                    const track = document.getElementById('track');
                    const success = document.getElementById('success');
                    let isDragging = false;
                    let startX = 0;

                    slider.addEventListener('mousedown', (e) => { isDragging = true; startX = e.clientX - slider.offsetLeft; });
                    document.addEventListener('mousemove', (e) => {
                        if (!isDragging) return;
                        let newX = e.clientX - startX;
                        newX = Math.max(0, Math.min(newX, 240));
                        slider.style.left = newX + 'px';
                    });
                    document.addEventListener('mouseup', () => {
                        if (!isDragging) return;
                        isDragging = false;
                        const pos = parseInt(slider.style.left || '0');
                        if (pos > 200) {
                            slider.style.left = '240px';
                            success.style.display = 'block';
                            setTimeout(() => { window.webkit.messageHandlers.captchaSuccess.postMessage({token: 'demo_token_\(UUID().uuidString)'}); }, 500);
                        } else {
                            slider.style.left = '0px';
                        }
                    });
                </script>
            </body>
            </html>
            """
        case .click:
            return """
            <!DOCTYPE html>
            <html>
            <head>
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <style>
                    body { margin: 0; padding: 20px; font-family: -apple-system, sans-serif; text-align: center; background: #f5f5f5; }
                    .container { background: white; border-radius: 8px; padding: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
                    .captcha-area { width: 300px; height: 200px; background: #e8f5e9; margin: 20px auto; border-radius: 8px; position: relative; display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 10px; }
                    .click-item { width: 50px; height: 50px; background: #4caf50; border-radius: 8px; cursor: pointer; display: flex; align-items: center; justify-content: center; color: white; font-size: 24px; }
                    .instruction { color: #666; margin: 10px 0; }
                </style>
            </head>
            <body>
                <div class="container">
                    <h3>点选验证</h3>
                    <p class="instruction">请依次点击: 太阳</p>
                    <div class="captcha-area" id="area">
                        <div class="click-item" data-value="sun">☀️</div>
                        <div class="click-item" data-value="moon">🌙</div>
                        <div class="click-item" data-value="star">⭐</div>
                        <div class="click-item" data-value="cloud">☁️</div>
                        <div class="click-item" data-value="sun">☀️</div>
                        <div class="click-item" data-value="rain">🌧️</div>
                    </div>
                </div>
                <script>
                    const items = document.querySelectorAll('.click-item');
                    let selected = [];
                    items.forEach(item => {
                        item.addEventListener('click', () => {
                            item.style.background = '#81c784';
                            selected.push(item.dataset.value);
                            if (selected.filter(v => v === 'sun').length >= 2) {
                                setTimeout(() => { window.webkit.messageHandlers.captchaSuccess.postMessage({token: 'demo_token_\(UUID().uuidString)'}); }, 300);
                            }
                        });
                    });
                </script>
            </body>
            </html>
            """
        case .rotate:
            return """
            <!DOCTYPE html>
            <html>
            <head>
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <style>
                    body { margin: 0; padding: 20px; font-family: -apple-system, sans-serif; text-align: center; background: #f5f5f5; }
                    .container { background: white; border-radius: 8px; padding: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
                    .captcha-area { width: 200px; height: 200px; margin: 20px auto; position: relative; }
                    .puzzle-image { width: 100%; height: 100%; background: linear-gradient(45deg, #ff6b6b 25%, #4ecdc4 50%, #45b7d1 75%); border-radius: 8px; transition: transform 0.1s; }
                    .slider-container { width: 280px; margin: 20px auto; }
                    input[type="range"] { width: 100%; }
                </style>
            </head>
            <body>
                <div class="container">
                    <h3>旋转验证</h3>
                    <p>将图片旋转至正确方向</p>
                    <div class="captcha-area">
                        <div class="puzzle-image" id="puzzle"></div>
                    </div>
                    <div class="slider-container">
                        <input type="range" id="rotateSlider" min="0" max="360" value="0">
                        <p id="angle">0°</p>
                    </div>
                    <button onclick="checkRotation()" style="padding: 10px 30px; background: #4ecdc4; color: white; border: none; border-radius: 5px; cursor: pointer;">确认</button>
                </div>
                <script>
                    const slider = document.getElementById('rotateSlider');
                    const puzzle = document.getElementById('puzzle');
                    const angleText = document.getElementById('angle');
                    slider.addEventListener('input', () => {
                        puzzle.style.transform = 'rotate(' + slider.value + 'deg)';
                        angleText.textContent = slider.value + '°';
                    });
                    function checkRotation() {
                        const angle = parseInt(slider.value) % 360;
                        if (angle < 10 || angle > 350) {
                            window.webkit.messageHandlers.captchaSuccess.postMessage({token: 'demo_token_\(UUID().uuidString)'});
                        }
                    }
                </script>
            </body>
            </html>
            """
        case .puzzle:
            return """
            <!DOCTYPE html>
            <html>
            <head>
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <style>
                    body { margin: 0; padding: 20px; font-family: -apple-system, sans-serif; text-align: center; background: #f5f5f5; }
                    .container { background: white; border-radius: 8px; padding: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
                    .captcha-area { width: 300px; height: 200px; background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); margin: 20px auto; border-radius: 8px; position: relative; overflow: hidden; }
                    .puzzle-piece { width: 60px; height: 60px; background: rgba(255,255,255,0.9); position: absolute; border-radius: 8px; top: 70px; left: 10px; cursor: grab; }
                    .target-slot { width: 60px; height: 60px; border: 3px dashed white; position: absolute; border-radius: 8px; top: 70px; right: 60px; }
                </style>
            </head>
            <body>
                <div class="container">
                    <h3>拼图验证</h3>
                    <p>将拼图拖动到正确位置</p>
                    <div class="captcha-area">
                        <div class="target-slot"></div>
                        <div class="puzzle-piece" id="piece"></div>
                    </div>
                </div>
                <script>
                    const piece = document.getElementById('piece');
                    let isDragging = false;
                    let startX, startY;
                    piece.addEventListener('mousedown', (e) => { isDragging = true; startX = e.clientX - piece.offsetLeft; startY = e.clientY - piece.offsetTop; });
                    document.addEventListener('mousemove', (e) => {
                        if (!isDragging) return;
                        piece.style.left = (e.clientX - startX) + 'px';
                        piece.style.top = (e.clientY - startY) + 'px';
                    });
                    document.addEventListener('mouseup', () => {
                        if (!isDragging) return;
                        isDragging = false;
                        const left = parseInt(piece.style.left || '0');
                        if (left > 150 && left < 220) {
                            piece.style.left = '190px';
                            setTimeout(() => { window.webkit.messageHandlers.captchaSuccess.postMessage({token: 'demo_token_\(UUID().uuidString)'}); }, 300);
                        }
                    });
                </script>
            </body>
            </html>
            """
        case .text:
            return """
            <!DOCTYPE html>
            <html>
            <head>
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <style>
                    body { margin: 0; padding: 20px; font-family: -apple-system, sans-serif; text-align: center; background: #f5f5f5; }
                    .container { background: white; border-radius: 8px; padding: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
                    .captcha-code { font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #333; background: #f0f0f0; padding: 20px; border-radius: 8px; margin: 20px auto; width: 200px; }
                    input { width: 200px; padding: 10px; font-size: 18px; border: 1px solid #ddd; border-radius: 5px; margin: 10px; }
                    button { padding: 10px 30px; background: #667eea; color: white; border: none; border-radius: 5px; cursor: pointer; font-size: 16px; }
                </style>
            </head>
            <body>
                <div class="container">
                    <h3>文字验证</h3>
                    <p>请输入下方验证码</p>
                    <div class="captcha-code" id="code">X7K9</div>
                    <input type="text" id="input" maxlength="4" placeholder="请输入验证码">
                    <br>
                    <button onclick="verify()">验证</button>
                </div>
                <script>
                    function verify() {
                        const code = document.getElementById('code').textContent;
                        const input = document.getElementById('input').value.toUpperCase();
                        if (code === input) {
                            window.webkit.messageHandlers.captchaSuccess.postMessage({token: 'demo_token_\(UUID().uuidString)'});
                        } else {
                            document.getElementById('input').value = '';
                            document.getElementById('code').textContent = String.fromCharCode(65+Math.random()*26) + Math.floor(Math.random()*10) + String.fromCharCode(65+Math.random()*26) + Math.floor(Math.random()*10);
                        }
                    }
                </script>
            </body>
            </html>
            """
        case .icon:
            return """
            <!DOCTYPE html>
            <html>
            <head>
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <style>
                    body { margin: 0; padding: 20px; font-family: -apple-system, sans-serif; text-align: center; background: #f5f5f5; }
                    .container { background: white; border-radius: 8px; padding: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
                    .icon-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; width: 280px; margin: 20px auto; }
                    .icon-item { width: 60px; height: 60px; background: #f5f5f5; border-radius: 8px; display: flex; align-items: center; justify-content: center; font-size: 30px; cursor: pointer; border: 2px solid transparent; }
                    .icon-item.selected { border-color: #667eea; background: #e8eaf6; }
                </style>
            </head>
            <body>
                <div class="container">
                    <h3>图标验证</h3>
                    <p>请选择所有包含「🌙」的图片</p>
                    <div class="icon-grid" id="grid"></div>
                </div>
                <script>
                    const icons = ['☀️', '🌙', '⭐', '☁️', '🌧️', '❄️', '🌈', '⚡'];
                    const grid = document.getElementById('grid');
                    let selected = [];
                    icons.forEach((icon, i) => {
                        const div = document.createElement('div');
                        div.className = 'icon-item';
                        div.textContent = icon;
                        div.onclick = () => {
                            div.classList.toggle('selected');
                            if (div.classList.contains('selected')) selected.push(i);
                            else selected = selected.filter(s => s !== i);
                            if (selected.length === 2 && selected.includes(1)) {
                                setTimeout(() => { window.webkit.messageHandlers.captchaSuccess.postMessage({token: 'demo_token_\(UUID().uuidString)'}); }, 300);
                            }
                        };
                        grid.appendChild(div);
                    });
                </script>
            </body>
            </html>
            """
        }
    }

    private func preloadResources(for type: CaptchaType) {
        Logger.debug("Preloading resources for \(type.rawValue) captcha")
    }

    @objc private func closeTapped() {
        destroy()
        onClose?()
    }

    public func reset() {
        webView?.removeFromSuperview()
        webView = nil
        loadCaptchaContent()
    }

    public func destroy() {
        hideWithAnimation()
        webView?.removeFromSuperview()
        webView = nil
        isLoading = false
        loadingView?.stopAnimating()
    }

    private func showWithAnimation() {
        UIView.animate(withDuration: 0.3) {
            self.alpha = 1
        }
    }

    private func hideWithAnimation() {
        UIView.animate(withDuration: 0.3, animations: {
            self.alpha = 0
        }) { _ in
            self.removeFromSuperview()
        }
    }
}

extension CaptchaView: WKNavigationDelegate {
    public func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        isLoading = false
        loadingView?.stopAnimating()
        
        webView.evaluateJavaScript("window.CaptchaXBridge = { success: function(token) { window.webkit.messageHandlers.captchaSuccess.postMessage({token: token}); }, error: function(msg) { window.webkit.messageHandlers.captchaError.postMessage({error: msg}); } };") { _, _ in }
    }

    public func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        isLoading = false
        loadingView?.stopAnimating()
        onError?(CaptchaXError.networkError(underlying: error))
    }

    public func webView(_ webView: WKWebView, decidePolicyFor navigationResponse: WKNavigationResponse, decisionHandler: @escaping (WKNavigationResponsePolicy) -> Void) {
        decisionHandler(.allow)
    }
}
