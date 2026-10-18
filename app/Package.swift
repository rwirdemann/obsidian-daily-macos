// swift-tools-version: 6.0

import PackageDescription

let package = Package(
    name: "ObsidianDaily",
    platforms: [.macOS(.v14)],
    targets: [
        .executableTarget(
            name: "ObsidianDaily",
            path: "Sources/ObsidianDaily",
            exclude: ["Resources/Info.plist", "Resources/AppIcon.icns"]
        ),
    ]
)
