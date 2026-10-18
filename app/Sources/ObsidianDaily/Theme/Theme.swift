import SwiftUI

enum Theme {
    static let background = Color(
        red: 0x1C / 255.0,
        green: 0x1C / 255.0,
        blue: 0x1E / 255.0
    )
    static let surface = Color(
        red: 0x2C / 255.0,
        green: 0x2C / 255.0,
        blue: 0x2E / 255.0
    )
    static let accent = Color.accentColor

    static let textPrimary = Color.white
    static let textSecondary = Color.gray.opacity(0.6)
    static let todoDone = Color.gray.opacity(0.4)

    static let bodyFont = Font.system(size: 13)
    static let captionFont = Font.system(size: 11)
    static let headerFont = Font.system(
        size: 14, weight: .semibold
    )
}
