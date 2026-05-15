import Foundation

public class CacheManager {
    public static let shared = CacheManager()

    public var cacheEnabled: Bool = true
    private let cacheDirectory: URL
    private let memoryCache = NSCache<NSString, NSData>()
    private let maxMemoryCacheSize = 50
    private let maxDiskCacheSize: UInt64 = 100 * 1024 * 1024

    private init() {
        let paths = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask)
        cacheDirectory = paths[0].appendingPathComponent("CaptchaX", isDirectory: true)

        try? FileManager.default.createDirectory(at: cacheDirectory, withIntermediateDirectories: true)

        memoryCache.countLimit = maxMemoryCacheSize
    }

    public func cacheImage(_ data: Data, forKey key: String) {
        guard cacheEnabled else { return }

        let cacheKey = cacheKeyFor(key)
        memoryCache.setObject(data as NSData, forKey: cacheKey as NSString)

        let fileURL = cacheDirectory.appendingPathComponent(cacheKey)
        try? data.write(to: fileURL)

        cleanupIfNeeded()
    }

    public func getCachedImage(forKey key: String) -> Data? {
        guard cacheEnabled else { return nil }

        let cacheKey = cacheKeyFor(key)

        if let memoryData = memoryCache.object(forKey: cacheKey as NSString) {
            return memoryData as Data
        }

        let fileURL = cacheDirectory.appendingPathComponent(cacheKey)
        if let diskData = try? Data(contentsOf: fileURL) {
            memoryCache.setObject(diskData as NSData, forKey: cacheKey as NSString)
            return diskData
        }

        return nil
    }

    public func clearCache() {
        memoryCache.removeAllObjects()
        try? FileManager.default.removeItem(at: cacheDirectory)
        try? FileManager.default.createDirectory(at: cacheDirectory, withIntermediateDirectories: true)
    }

    public func removeCache(forKey key: String) {
        let cacheKey = cacheKeyFor(key)
        memoryCache.removeObject(forKey: cacheKey as NSString)
        let fileURL = cacheDirectory.appendingPathComponent(cacheKey)
        try? FileManager.default.removeItem(at: fileURL)
    }

    public func cacheSize() -> UInt64 {
        var size: UInt64 = 0
        if let enumerator = FileManager.default.enumerator(at: cacheDirectory, includingPropertiesForKeys: [.fileSizeKey]) {
            while let fileURL = enumerator.nextObject() as? URL {
                if let fileSize = try? fileURL.resourceValues(forKeys: [.fileSizeKey]).fileSize {
                    size += UInt64(fileSize)
                }
            }
        }
        return size
    }

    private func cacheKeyFor(_ key: String) -> String {
        return key.data(using: .utf8)?.sha256Hash() ?? key
    }

    private func cleanupIfNeeded() {
        let currentSize = cacheSize()
        guard currentSize > maxDiskCacheSize else { return }

        guard let enumerator = FileManager.default.enumerator(
            at: cacheDirectory,
            includingPropertiesForKeys: [.contentModificationDateKey],
            options: [.skipsHiddenFiles]
        ) else { return }

        var files: [(url: URL, date: Date)] = []
        while let fileURL = enumerator.nextObject() as? URL {
            if let values = try? fileURL.resourceValues(forKeys: [.contentModificationDateKey]),
               let date = values.contentModificationDate {
                files.append((fileURL, date))
            }
        }

        files.sort { $0.date < $1.date }

        var sizeToRemove = currentSize - (maxDiskCacheSize / 2)
        for file in files {
            guard sizeToRemove > 0 else { break }
            if let fileSize = try? file.url.resourceValues(forKeys: [.fileSizeKey]).fileSize {
                try? FileManager.default.removeItem(at: file.url)
                sizeToRemove -= UInt64(fileSize)
            }
        }
    }
}
