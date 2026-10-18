// Renders the app icon as a 1024x1024 PNG.
// Usage: swift scripts/make-icon.swift <output.png>
import AppKit

let size: CGFloat = 1024
let out = CommandLine.arguments[1]

let rep = NSBitmapImageRep(
    bitmapDataPlanes: nil, pixelsWide: Int(size), pixelsHigh: Int(size),
    bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false,
    colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: rep)

// Body follows the macOS icon grid: 824pt tile, centered, with shadow.
let body = NSRect(x: 100, y: 100, width: 824, height: 824)
let tile = NSBezierPath(roundedRect: body, xRadius: 185, yRadius: 185)

NSGraphicsContext.saveGraphicsState()
let shadow = NSShadow()
shadow.shadowColor = NSColor.black.withAlphaComponent(0.3)
shadow.shadowBlurRadius = 20
shadow.shadowOffset = NSSize(width: 0, height: -10)
shadow.set()
NSColor.black.setFill()
tile.fill()
NSGraphicsContext.restoreGraphicsState()

NSGradient(
    starting: NSColor(srgbRed: 0.62, green: 0.42, blue: 1.0, alpha: 1),
    ending: NSColor(srgbRed: 0.33, green: 0.16, blue: 0.75, alpha: 1)
)!.draw(in: tile, angle: -90)

let config = NSImage.SymbolConfiguration(pointSize: 440, weight: .semibold)
    .applying(.init(paletteColors: [.white]))
let symbol = NSImage(
    systemSymbolName: "checklist", accessibilityDescription: nil
)!.withSymbolConfiguration(config)!
let s = symbol.size
symbol.draw(in: NSRect(
    x: (size - s.width) / 2, y: (size - s.height) / 2,
    width: s.width, height: s.height))

NSGraphicsContext.current = nil
try! rep.representation(using: .png, properties: [:])!
    .write(to: URL(fileURLWithPath: out))
