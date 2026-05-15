import UIKit
import CaptchaX

class ViewController: UIViewController, CaptchaXDelegate {
    private var captchaView: CaptchaView?
    private let statusLabel = UILabel()
    private let verifyButton = UIButton(type: .system)
    private let sliderButton = UIButton(type: .system)
    private let clickButton = UIButton(type: .system)

    override func viewDidLoad() {
        super.viewDidLoad()
        setupUI()
        setupCaptchaX()
    }

    private func setupUI() {
        view.backgroundColor = .systemBackground

        let titleLabel = UILabel()
        titleLabel.text = "CaptchaX iOS SDK"
        titleLabel.font = .systemFont(ofSize: 28, weight: .bold)
        titleLabel.textAlignment = .center
        titleLabel.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(titleLabel)

        let subtitleLabel = UILabel()
        subtitleLabel.text = "UIKit 示例"
        subtitleLabel.font = .systemFont(ofSize: 16)
        subtitleLabel.textColor = .secondaryLabel
        subtitleLabel.textAlignment = .center
        subtitleLabel.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(subtitleLabel)

        statusLabel.text = "状态: 未验证"
        statusLabel.font = .systemFont(ofSize: 18)
        statusLabel.textAlignment = .center
        statusLabel.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(statusLabel)

        verifyButton.setTitle("开始验证", for: .normal)
        verifyButton.titleLabel?.font = .systemFont(ofSize: 18, weight: .medium)
        verifyButton.backgroundColor = .systemBlue
        verifyButton.setTitleColor(.white, for: .normal)
        verifyButton.layer.cornerRadius = 12
        verifyButton.translatesAutoresizingMaskIntoConstraints = false
        verifyButton.addTarget(self, action: #selector(verifyTapped), for: .touchUpInside)
        view.addSubview(verifyButton)

        sliderButton.setTitle("滑块验证", for: .normal)
        sliderButton.titleLabel?.font = .systemFont(ofSize: 16)
        sliderButton.backgroundColor = .systemGray5
        sliderButton.setTitleColor(.label, for: .normal)
        sliderButton.layer.cornerRadius = 10
        sliderButton.translatesAutoresizingMaskIntoConstraints = false
        sliderButton.addTarget(self, action: #selector(sliderTapped), for: .touchUpInside)
        view.addSubview(sliderButton)

        clickButton.setTitle("点选验证", for: .normal)
        clickButton.titleLabel?.font = .systemFont(ofSize: 16)
        clickButton.backgroundColor = .systemGray5
        clickButton.setTitleColor(.label, for: .normal)
        clickButton.layer.cornerRadius = 10
        clickButton.translatesAutoresizingMaskIntoConstraints = false
        clickButton.addTarget(self, action: #selector(clickTapped), for: .touchUpInside)
        view.addSubview(clickButton)

        NSLayoutConstraint.activate([
            titleLabel.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 40),
            titleLabel.centerXAnchor.constraint(equalTo: view.centerXAnchor),

            subtitleLabel.topAnchor.constraint(equalTo: titleLabel.bottomAnchor, constant: 8),
            subtitleLabel.centerXAnchor.constraint(equalTo: view.centerXAnchor),

            statusLabel.topAnchor.constraint(equalTo: subtitleLabel.bottomAnchor, constant: 40),
            statusLabel.centerXAnchor.constraint(equalTo: view.centerXAnchor),

            verifyButton.topAnchor.constraint(equalTo: statusLabel.bottomAnchor, constant: 40),
            verifyButton.centerXAnchor.constraint(equalTo: view.centerXAnchor),
            verifyButton.widthAnchor.constraint(equalToConstant: 200),
            verifyButton.heightAnchor.constraint(equalToConstant: 50),

            sliderButton.topAnchor.constraint(equalTo: verifyButton.bottomAnchor, constant: 20),
            sliderButton.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 40),
            sliderButton.widthAnchor.constraint(equalToConstant: 150),
            sliderButton.heightAnchor.constraint(equalToConstant: 44),

            clickButton.topAnchor.constraint(equalTo: verifyButton.bottomAnchor, constant: 20),
            clickButton.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -40),
            clickButton.widthAnchor.constraint(equalToConstant: 150),
            clickButton.heightAnchor.constraint(equalToConstant: 44)
        ])
    }

    private func setupCaptchaX() {
        CaptchaX.shared.delegate = self
        CaptchaX.shared.initialize(apiKey: "your_api_key", apiSecret: "your_api_secret")
    }

    @objc private func verifyTapped() {
        CaptchaX.shared.verify(scene: "login") { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success(let token):
                    self?.showAlert(title: "验证成功", message: "Token: \(token.token.prefix(20))...")
                case .failure(let error):
                    self?.showAlert(title: "验证失败", message: error.localizedDescription)
                }
            }
        }
    }

    @objc private func sliderTapped() {
        showCaptcha(type: .slider)
    }

    @objc private func clickTapped() {
        showCaptcha(type: .click)
    }

    private func showCaptcha(type: CaptchaType) {
        captchaView?.removeFromSuperview()

        let captcha = CaptchaView()
        captcha.frame = CGRect(x: (view.bounds.width - 350) / 2, y: (view.bounds.height - 400) / 2, width: 350, height: 400)

        captcha.onSuccess = { [weak self] token in
            self?.statusLabel.text = "状态: 已验证"
            self?.statusLabel.textColor = .systemGreen
            self?.showAlert(title: "验证成功", message: "Token: \(token.prefix(20))...")
            captcha.destroy()
        }

        captcha.onError = { [weak self] error in
            self?.showAlert(title: "验证失败", message: error.localizedDescription)
            captcha.destroy()
        }

        captcha.onClose = { [weak self] in
            self?.showAlert(title: "提示", message: "用户关闭了验证")
        }

        view.addSubview(captcha)
        captcha.load(captchaType: type)
        captchaView = captcha
    }

    private func showAlert(title: String, message: String) {
        let alert = UIAlertController(title: title, message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "确定", style: .default))
        present(alert, animated: true)
    }

    func captchaX(_ captcha: CaptchaX, didSuccess result: CaptchaResult) {
        DispatchQueue.main.async { [weak self] in
            self?.statusLabel.text = "状态: 已验证"
            self?.statusLabel.textColor = .systemGreen
        }
    }

    func captchaX(_ captcha: CaptchaX, didFailed error: Error) {
        DispatchQueue.main.async { [weak self] in
            self?.showAlert(title: "验证失败", message: error.localizedDescription)
        }
    }

    func captchaXDidClose(_ captcha: CaptchaX) {
        DispatchQueue.main.async { [weak self] in
            self?.showAlert(title: "提示", message: "验证已关闭")
        }
    }
}
