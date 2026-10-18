import AppKit
import SwiftUI

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusItem: NSStatusItem!
    private var popover: NSPopover!
    private let viewModel = DailyViewModel()

    func applicationDidFinishLaunching(
        _ notification: Notification
    ) {
        statusItem = NSStatusBar.system.statusItem(
            withLength: NSStatusItem.variableLength
        )

        if let button = statusItem.button {
            button.image = NSImage(
                systemSymbolName: "checklist",
                accessibilityDescription: "Obsidian Daily"
            )
            button.title = ""
            button.imagePosition = .imageLeading
            button.action = #selector(togglePopover)
            button.target = self
        }

        let contentView = PopoverContentView(
            viewModel: viewModel
        )

        popover = NSPopover()
        popover.contentSize = NSSize(width: 360, height: 520)
        popover.behavior = .transient
        popover.contentViewController =
            NSHostingController(rootView: contentView)

        viewModel.onMenuBarUpdate = { [weak self] title in
            self?.statusItem.button?.title = title
        }
    }

    @objc private func togglePopover() {
        guard let button = statusItem.button else { return }

        if popover.isShown {
            popover.performClose(nil)
        } else {
            popover.show(
                relativeTo: button.bounds,
                of: button,
                preferredEdge: .minY
            )
            popover.contentViewController?.view
                .window?
                .makeKey()
        }
    }
}
