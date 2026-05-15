import SwiftUI
import CaptchaX

struct ContentView: View {
    @State private var isVerified = false
    @State private var verificationStatus: String = "未验证"
    @State private var showingAlert = false
    @State private var alertMessage = ""

    var body: some View {
        NavigationView {
            VStack(spacing: 30) {
                Image(systemName: "lock.shield.fill")
                    .font(.system(size: 80))
                    .foregroundColor(isVerified ? .green : .blue)

                Text("CaptchaX iOS SDK")
                    .font(.largeTitle)
                    .fontWeight(.bold)

                Text("SwiftUI 示例")
                    .font(.subheadline)
                    .foregroundColor(.secondary)

                Divider()
                    .padding(.horizontal, 40)

                VStack(spacing: 15) {
                    HStack {
                        Image(systemName: "checkmark.circle.fill")
                            .foregroundColor(.green)
                        Text("状态: \(verificationStatus)")
                            .font(.headline)
                    }

                    if isVerified {
                        Text("验证成功！")
                            .foregroundColor(.green)
                            .font(.title2)
                    }
                }
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(12)

                Spacer()

                VStack(spacing: 15) {
                    Button(action: performVerification) {
                        HStack {
                            Image(systemName: "checkmark.shield")
                            Text(isVerified ? "重新验证" : "开始验证")
                        }
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(isVerified ? Color.orange : Color.blue)
                        .foregroundColor(.white)
                        .cornerRadius(12)
                    }

                    Button(action: verifyWithSlider) {
                        HStack {
                            Image(systemName: "slider.horizontal.3")
                            Text("滑块验证")
                        }
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color(.systemGray5))
                        .foregroundColor(.primary)
                        .cornerRadius(12)
                    }

                    Button(action: verifyWithClick) {
                        HStack {
                            Image(systemName: "hand.tap")
                            Text("点选验证")
                        }
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color(.systemGray5))
                        .foregroundColor(.primary)
                        .cornerRadius(12)
                    }
                }
                .padding(.horizontal, 30)

                Spacer()
            }
            .padding()
            .navigationTitle("CaptchaX")
            .navigationBarTitleDisplayMode(.inline)
            .alert("验证结果", isPresented: $showingAlert) {
                Button("确定", role: .cancel) { }
            } message: {
                Text(alertMessage)
            }
        }
    }

    private func performVerification() {
        CaptchaX.shared.verify(scene: "login") { result in
            handleResult(result)
        }
    }

    private func verifyWithSlider() {
        CaptchaX.shared.verifyWithView(scene: "login", captchaType: .slider) { result in
            handleResult(result)
        }
    }

    private func verifyWithClick() {
        CaptchaX.shared.verifyWithView(scene: "login", captchaType: .click) { result in
            handleResult(result)
        }
    }

    private func handleResult(_ result: Result<CaptchaResult, Error>) {
        switch result {
        case .success(let captchaResult):
            isVerified = true
            verificationStatus = "已验证"
            alertMessage = "验证成功！Token: \(captchaResult.token.prefix(20))..."
            Logger.info("Verification success: \(captchaResult.token)")
        case .failure(let error):
            alertMessage = "验证失败: \(error.localizedDescription)"
            Logger.error("Verification failed: \(error.localizedDescription)")
        }
        showingAlert = true
    }
}

#if DEBUG
struct ContentView_Previews: PreviewProvider {
    static var previews: some View {
        ContentView()
    }
}
#endif
