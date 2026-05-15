import Foundation
import UIKit

public class DeviceFingerprint {
    public static func generate() -> String {
        let components = collect()
        let data = try? JSONSerialization.data(withJSONObject: components, options: .sortedKeys)
        guard let jsonData = data else {
            return UUID().uuidString
        }
        return jsonData.sha256Hash()
    }

    public static func collect() -> [String: Any] {
        var fingerprint: [String: Any] = [:]

        fingerprint["deviceId"] = UIDevice.current.identifierForVendor?.uuidString ?? UUID().uuidString
        fingerprint["deviceName"] = UIDevice.current.name
        fingerprint["systemName"] = UIDevice.current.systemName
        fingerprint["systemVersion"] = UIDevice.current.systemVersion
        fingerprint["model"] = UIDevice.current.model
        fingerprint["localizedModel"] = UIDevice.current.localizedModel

        if let screenScale = UIScreen.main.currentMode?.scale {
            fingerprint["screenScale"] = screenScale
        }
        fingerprint["screenBounds"] = "\(UIScreen.main.bounds.width)x\(UIScreen.main.bounds.height)"
        fingerprint["screenBrightness"] = UIScreen.main.brightness

        fingerprint["processorCount"] = ProcessInfo.processInfo.activeProcessorCount
        fingerprint["memoryTotal"] = ProcessInfo.processInfo.physicalMemory

        let locales = Locale.preferredLanguages.prefix(3)
        fingerprint["preferredLanguages"] = Array(locales)
        fingerprint["locale"] = Locale.current.identifier

        fingerprint["timezone"] = TimeZone.current.identifier
        fingerprint["timeOffset"] = TimeZone.current.secondsFromGMT()

        var cpuBrand: String = "unknown"
        #if arch(arm64)
        cpuBrand = "Apple Silicon"
        #endif
        fingerprint["cpuBrand"] = cpuBrand

        fingerprint["batteryLevel"] = UIDevice.current.isBatteryMonitoringEnabled ? UIDevice.current.batteryLevel : -1
        fingerprint["batteryState"] = batteryStateString()

        fingerprint["isSimulator"] = TARGET_OS_SIMULATOR != 0

        return fingerprint
    }

    private static func batteryStateString() -> String {
        switch UIDevice.current.batteryState {
        case .unknown:
            return "unknown"
        case .unplugged:
            return "unplugged"
        case .charging:
            return "charging"
        case .full:
            return "full"
        @unknown default:
            return "unknown"
        }
    }
}

extension Data {
    func sha256Hash() -> String {
        var hash = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
        self.withUnsafeBytes { buffer in
            _ = CC_SHA256(buffer.baseAddress, CC_LONG(self.count), &hash)
        }
        return hash.map { String(format: "%02x", $0) }.joined()
    }
}

import CommonCrypto
