import AppKit

enum VaultSettings {
    private static let key = "vaultPath"

    /// The stored vault path, else OBSIDIAN_VAULT_ROOT.
    static var path: String? {
        if let stored = UserDefaults.standard
            .string(forKey: key), !stored.isEmpty
        {
            return stored
        }
        let env = ProcessInfo.processInfo
            .environment["OBSIDIAN_VAULT_ROOT"]
        return env?.isEmpty == false ? env : nil
    }

    static func save(_ path: String) {
        UserDefaults.standard.set(path, forKey: key)
    }

    static func isVault(_ path: String) -> Bool {
        var isDir: ObjCBool = false
        let marker = (path as NSString)
            .appendingPathComponent(".obsidian")
        return FileManager.default.fileExists(
            atPath: marker, isDirectory: &isDir
        ) && isDir.boolValue
    }

    @MainActor
    static func choose() -> String? {
        NSApp.activate()
        let panel = NSOpenPanel()
        panel.canChooseDirectories = true
        panel.canChooseFiles = false
        panel.allowsMultipleSelection = false
        panel.prompt = "Choose Vault"
        panel.message = "Select your Obsidian vault folder"
        if let current = path {
            panel.directoryURL = URL(fileURLWithPath: current)
        }
        guard panel.runModal() == .OK,
            let url = panel.url
        else { return nil }
        return url.path
    }
}
