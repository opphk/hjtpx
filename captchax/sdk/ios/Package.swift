// swift-tools-version:5.5
import PackageDescription

let package = Package(
    name: "CaptchaX",
    platforms: [
        .iOS(.v13)
    ],
    products: [
        .library(
            name: "CaptchaX",
            targets: ["CaptchaX"]
        ),
    ],
    dependencies: [
        .package(url: "https://github.com/SnapKit/SnapKit.git", from: "5.6.0")
    ],
    targets: [
        .target(
            name: "CaptchaX",
            dependencies: [
                .product(name: "SnapKit", package: "SnapKit")
            ],
            path: "Sources/CaptchaX",
            publicHeadersPath: "../CaptchaXObjC"
        ),
        .target(
            name: "CaptchaXObjC",
            dependencies: [
                "CaptchaX",
                .product(name: "SnapKit", package: "SnapKit")
            ],
            path: "Sources/CaptchaXObjC"
        )
    ]
)
