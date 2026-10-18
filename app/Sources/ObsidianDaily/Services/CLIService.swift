import Foundation

actor CLIService {
    enum CLIError: Error, LocalizedError {
        case binaryNotFound
        case processFailure(String)
        case decodingFailure(String)

        var errorDescription: String? {
            switch self {
            case .binaryNotFound:
                "CLI binary not found"
            case .processFailure(let msg):
                "CLI error: \(msg)"
            case .decodingFailure(let msg):
                "Decode error: \(msg)"
            }
        }
    }

    private let binaryPath: String
    private var vaultPath: String?
    private var watchProcess: Process?

    init(vaultPath: String?) {
        let fm = FileManager.default
        let name = "obsidian-daily-macos"

        // 1. Bundled in .app/Contents/Resources
        if let bundled = Bundle.main.url(
            forResource: name, withExtension: nil
        ), fm.fileExists(atPath: bundled.path) {
            binaryPath = bundled.path
        }
        // 2. Sibling of working directory (swift run
        //    from app/ → repo root is parent)
        else if let parent = URL(
            fileURLWithPath: fm.currentDirectoryPath
        ).deletingLastPathComponent()
            .appendingPathComponent(name) as URL?,
            fm.fileExists(atPath: parent.path)
        {
            binaryPath = parent.path
        }
        // 3. Current working directory
        else {
            let cwd = fm.currentDirectoryPath
            let local = (cwd as NSString)
                .appendingPathComponent(name)
            if fm.fileExists(atPath: local) {
                binaryPath = local
            } else {
                binaryPath = name
            }
        }

        self.vaultPath = vaultPath
    }

    func setVault(_ path: String) {
        vaultPath = path
    }

    private func run(
        _ args: [String]
    ) async throws -> Data {
        var fullArgs = args
        if let vault = vaultPath, !vault.isEmpty {
            fullArgs = ["--vault", vault] + fullArgs
        }

        let process = Process()
        process.executableURL = URL(
            fileURLWithPath: binaryPath
        )
        process.arguments = fullArgs

        let stdout = Pipe()
        let stderr = Pipe()
        process.standardOutput = stdout
        process.standardError = stderr

        try process.run()
        process.waitUntilExit()

        let data = stdout.fileHandleForReading
            .readDataToEndOfFile()

        if process.terminationStatus != 0 {
            if !data.isEmpty { return data }
            let errData = stderr.fileHandleForReading
                .readDataToEndOfFile()
            let errMsg = String(
                data: errData, encoding: .utf8
            ) ?? "unknown error"
            throw CLIError.processFailure(errMsg)
        }

        return data
    }

    func status(
        date: String? = nil
    ) async throws -> Snapshot {
        var args = ["status"]
        if let d = date { args += ["--date", d] }
        let data = try await run(args)
        return try JSONDecoder().decode(
            Snapshot.self, from: data
        )
    }

    func week(
        date: String? = nil
    ) async throws -> WeekSummary {
        var args = ["week"]
        if let d = date { args += ["--date", d] }
        let data = try await run(args)
        return try JSONDecoder().decode(
            WeekSummary.self, from: data
        )
    }

    func add(
        text: String, date: String? = nil
    ) async throws -> Snapshot {
        var args = ["add", "--text", text]
        if let d = date { args += ["--date", d] }
        let data = try await run(args)
        return try JSONDecoder().decode(
            Snapshot.self, from: data
        )
    }

    func toggle(
        line: Int,
        date: String? = nil,
        expectText: String? = nil
    ) async throws -> Snapshot {
        var args = ["toggle", "--line", "\(line)"]
        if let d = date { args += ["--date", d] }
        if let et = expectText {
            args += ["--expect-text", et]
        }
        let data = try await run(args)
        return try JSONDecoder().decode(
            Snapshot.self, from: data
        )
    }

    func open(
        date: String? = nil
    ) async throws -> Snapshot {
        var args = ["open"]
        if let d = date { args += ["--date", d] }
        let data = try await run(args)
        return try JSONDecoder().decode(
            Snapshot.self, from: data
        )
    }

    func undo() async throws -> Snapshot {
        let data = try await run(["undo"])
        return try JSONDecoder().decode(
            Snapshot.self, from: data
        )
    }

    func delete(
        line: Int,
        date: String? = nil,
        expectText: String? = nil
    ) async throws -> Snapshot {
        var args = ["delete", "--line", "\(line)"]
        if let d = date { args += ["--date", d] }
        if let et = expectText {
            args += ["--expect-text", et]
        }
        let data = try await run(args)
        return try JSONDecoder().decode(
            Snapshot.self, from: data
        )
    }

    func inbox() async throws -> Snapshot {
        let data = try await run(["inbox"])
        return try JSONDecoder().decode(
            Snapshot.self, from: data
        )
    }

    func inboxAdd(
        text: String
    ) async throws -> Snapshot {
        let data = try await run(
            ["inbox-add", "--text", text]
        )
        return try JSONDecoder().decode(
            Snapshot.self, from: data
        )
    }

    func inboxDelete(
        line: Int,
        expectText: String? = nil
    ) async throws -> Snapshot {
        var args = [
            "inbox-delete", "--line", "\(line)",
        ]
        if let et = expectText {
            args += ["--expect-text", et]
        }
        let data = try await run(args)
        return try JSONDecoder().decode(
            Snapshot.self, from: data
        )
    }

    func inboxMove(
        line: Int,
        date: String? = nil,
        expectText: String? = nil
    ) async throws -> Snapshot {
        var args = [
            "inbox-move", "--line", "\(line)",
        ]
        if let d = date { args += ["--date", d] }
        if let et = expectText {
            args += ["--expect-text", et]
        }
        let data = try await run(args)
        return try JSONDecoder().decode(
            Snapshot.self, from: data
        )
    }

    func watch() -> AsyncStream<Snapshot> {
        AsyncStream { continuation in
            let process = Process()
            process.executableURL = URL(
                fileURLWithPath: binaryPath
            )
            var args = ["watch"]
            if let vault = vaultPath, !vault.isEmpty {
                args = ["--vault", vault] + args
            }
            process.arguments = args

            let pipe = Pipe()
            process.standardOutput = pipe

            continuation.onTermination = { _ in
                if process.isRunning {
                    process.terminate()
                }
            }

            let handle = pipe.fileHandleForReading

            do {
                try process.run()
            } catch {
                continuation.finish()
                return
            }

            self.setWatchProcess(process)

            Task.detached {
                let decoder = JSONDecoder()
                var buffer = Data()

                while true {
                    let chunk = handle
                        .availableData
                    if chunk.isEmpty { break }

                    buffer.append(chunk)

                    while let range = buffer.range(
                        of: Data("\n".utf8)
                    ) {
                        let line = buffer[
                            buffer.startIndex..<range
                                .lowerBound
                        ]
                        buffer.removeSubrange(
                            buffer.startIndex...range
                                .lowerBound
                        )

                        if line.isEmpty { continue }

                        if let snap = try? decoder
                            .decode(
                                Snapshot.self,
                                from: line
                            )
                        {
                            continuation.yield(snap)
                        }
                    }
                }
                continuation.finish()
            }
        }
    }

    private func setWatchProcess(
        _ process: Process
    ) {
        watchProcess?.terminate()
        watchProcess = process
    }

    func stopWatch() {
        watchProcess?.terminate()
        watchProcess = nil
    }
}
