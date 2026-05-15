import Foundation
import UIKit

extension String {
    public var sha256: String {
        guard let data = self.data(using: .utf8) else { return self }
        return data.sha256Hash()
    }

    public func isValidURL() -> Bool {
        guard let url = URL(string: self) else { return false }
        return url.scheme != nil && url.host != nil
    }

    public var base64Encoded: String? {
        return data(using: .utf8)?.base64EncodedString()
    }

    public var base64Decoded: String? {
        guard let data = Data(base64Encoded: self) else { return nil }
        return String(data: data, encoding: .utf8)
    }
}

extension Date {
    public var iso8601String: String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.string(from: self)
    }

    public static func fromISO8601(_ string: String) -> Date? {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.date(from: string)
    }

    public func adding(seconds: TimeInterval) -> Date {
        return addingTimeInterval(seconds)
    }

    public func adding(minutes: Int) -> Date {
        return addingTimeInterval(TimeInterval(minutes * 60))
    }

    public func adding(hours: Int) -> Date {
        return addingTimeInterval(TimeInterval(hours * 3600))
    }

    public func adding(days: Int) -> Date {
        return addingTimeInterval(TimeInterval(days * 86400))
    }
}

extension Dictionary {
    public func toJSONString() -> String? {
        guard let data = try? JSONSerialization.data(withJSONObject: self, options: .sortedKeys) else {
            return nil
        }
        return String(data: data, encoding: .utf8)
    }

    public func toQueryString() -> String {
        return self.map { "\($0.key)=\($0.value)" }.joined(separator: "&")
    }
}

extension Array {
    public func chunked(into size: Int) -> [[Element]] {
        return stride(from: 0, to: count, by: size).map {
            Array(self[$0..<Swift.min($0 + size, count)])
        }
    }
}

extension UIView {
    public func addShadow(
        color: UIColor = .black,
        opacity: Float = 0.2,
        offset: CGSize = CGSize(width: 0, height: 2),
        radius: CGFloat = 4
    ) {
        layer.shadowColor = color.cgColor
        layer.shadowOpacity = opacity
        layer.shadowOffset = offset
        layer.shadowRadius = radius
        layer.masksToBounds = false
    }

    public func roundCorners(_ corners: UIRectCorner, radius: CGFloat) {
        let path = UIBezierPath(
            roundedRect: bounds,
            byRoundingCorners: corners,
            cornerRadii: CGSize(width: radius, height: radius)
        )
        let mask = CAShapeLayer()
        mask.path = path.cgPath
        layer.mask = mask
    }

    public func shake() {
        let animation = CAKeyframeAnimation(keyPath: "transform.translation.x")
        animation.timingFunction = CAMediaTimingFunction(name: .linear)
        animation.duration = 0.6
        animation.values = [-10, 10, -8, 8, -5, 5, -2, 2, 0]
        layer.add(animation, forKey: "shake")
    }
}

extension Data {
    public var prettyPrintedJSON: String? {
        guard let object = try? JSONSerialization.jsonObject(with: self, options: []),
              let data = try? JSONSerialization.data(withJSONObject: object, options: [.prettyPrinted]),
              let string = String(data: data, encoding: .utf8) else {
            return nil
        }
        return string
    }

    public var hexString: String {
        return map { String(format: "%02x", $0) }.joined()
    }

    public init?(hexString: String) {
        let len = hexString.count / 2
        var data = Data(capacity: len)
        var index = hexString.startIndex
        for _ in 0..<len {
            let nextIndex = hexString.index(index, offsetBy: 2)
            guard let byte = UInt8(hexString[index..<nextIndex], radix: 16) else { return nil }
            data.append(byte)
            index = nextIndex
        }
        self = data
    }
}

extension Int {
    public var boolValue: Bool {
        return self != 0
    }
}

extension Bool {
    public var intValue: Int {
        return self ? 1 : 0
    }
}

extension Optional where Wrapped == String {
    public var orEmpty: String {
        return self ?? ""
    }

    public var isNilOrEmpty: Bool {
        return self?.isEmpty ?? true
    }
}

extension Encodable {
    public func toDictionary() -> [String: Any]? {
        guard let data = try? JSONEncoder().encode(self) else { return nil }
        return (try? JSONSerialization.jsonObject(with: data, options: .allowFragments)) as? [String: Any]
    }

    public func toJSONString() -> String? {
        guard let data = try? JSONEncoder().encode(self) else { return nil }
        return String(data: data, encoding: .utf8)
    }
}
