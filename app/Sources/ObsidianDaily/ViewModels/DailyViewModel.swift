import Foundation
import SwiftUI

@Observable
@MainActor
final class DailyViewModel {
    var selectedDate: Date = .now
    var snapshot: Snapshot?
    var weekSummary: WeekSummary?
    var inboxSnapshot: Snapshot? {
        didSet { updateMenuBar() }
    }
    var searchText: String = ""
    var hideDone: Bool = false
    var newTodoText: String = ""
    var newInboxText: String = ""
    var errorMessage: String?
    var showingInbox: Bool = false {
        didSet { updateMenuBar() }
    }
    var vaultPath: String? = VaultSettings.path

    var onMenuBarUpdate: ((String) -> Void)?

    private let cli = CLIService(
        vaultPath: VaultSettings.path
    )
    private var watchTask: Task<Void, Never>?
    private var watchedDate: String?

    // MARK: - Computed

    var dateString: String {
        Self.formatter.string(from: selectedDate)
    }

    var displayDate: String {
        let f = DateFormatter()
        f.dateFormat = "EEEE, d MMM"
        return f.string(from: selectedDate)
    }

    var filteredTodos: [TodoItem] {
        guard let todos = snapshot?.todos else {
            return []
        }
        var result = todos
        if hideDone {
            result = result.filter { !$0.checked }
        }
        if !searchText.isEmpty {
            let query = searchText.lowercased()
            result = result.filter {
                $0.text.lowercased().contains(query)
            }
        }
        return result
    }

    var statusText: String {
        Self.formatStatus(snapshot)
    }

    var inboxStatusText: String {
        Self.formatStatus(inboxSnapshot)
    }

    private static func formatStatus(
        _ snapshot: Snapshot?
    ) -> String {
        let done = snapshot?.doneCount ?? 0
        let open = snapshot?.openCount ?? 0
        return "\(done)/\(done + open) done · \(open) open"
    }

    var menuBarTitle: String {
        let current = showingInbox ? inboxSnapshot : snapshot
        guard let snapshot = current, snapshot.isOk else {
            return ""
        }
        let done = snapshot.doneCount ?? 0
        let total = done + (snapshot.openCount ?? 0)
        return " \(done)/\(total)"
    }

    var carryOverCount: Int {
        guard isToday else { return 0 }
        return snapshot?.carryOverCount ?? 0
    }

    var needsVault: Bool {
        vaultPath == nil
    }

    var isToday: Bool {
        Calendar.current.isDateInToday(selectedDate)
    }

    var inboxTodos: [TodoItem] {
        inboxSnapshot?.todos ?? []
    }

    var inboxOpenCount: Int {
        inboxSnapshot?.openCount ?? 0
    }

    // MARK: - Actions

    func navigateDay(_ offset: Int) {
        guard let newDate = Calendar.current.date(
            byAdding: .day, value: offset,
            to: selectedDate
        ) else { return }
        showingInbox = false
        selectedDate = newDate
        refresh()
    }

    func selectDate(_ date: Date) {
        showingInbox = false
        selectedDate = date
        refresh()
    }

    func selectInbox() {
        showingInbox = true
        Task { await loadInbox() }
    }

    func refresh() {
        guard !needsVault else { return }
        Task {
            await loadStatus()
            await loadWeek()
            await loadInbox()
            updateMenuBar()
            startWatching()
        }
    }

    func chooseVault() {
        guard let path = VaultSettings.choose() else {
            return
        }
        guard VaultSettings.isVault(path) else {
            errorMessage = "Not an Obsidian vault: \(path)"
            return
        }
        VaultSettings.save(path)
        vaultPath = path
        errorMessage = nil
        stopWatching()
        Task {
            await cli.setVault(path)
            refresh()
        }
    }

    func toggleTodo(_ todo: TodoItem) {
        Task {
            do {
                let snap = try await cli.toggle(
                    line: todo.line,
                    date: dateString,
                    expectText: todo.text
                )
                snapshot = snap
                updateMenuBar()
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    func addTodo() {
        let text = newTodoText
            .trimmingCharacters(in: .whitespaces)
        guard !text.isEmpty else { return }
        newTodoText = ""
        Task {
            do {
                let snap = try await cli.add(
                    text: text, date: dateString
                )
                snapshot = snap
                await loadWeek()
                updateMenuBar()
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    func carryOver() {
        Task {
            do {
                snapshot = try await cli.carryOver(
                    date: dateString
                )
                await loadWeek()
                updateMenuBar()
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    func deleteTodo(_ todo: TodoItem) {
        Task {
            do {
                let snap = try await cli.delete(
                    line: todo.line,
                    date: dateString,
                    expectText: todo.text
                )
                snapshot = snap
                updateMenuBar()
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    func openInObsidian() {
        Task {
            do {
                _ = try await cli.open(
                    date: dateString
                )
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    func undo() {
        Task {
            do {
                let snap = try await cli.undo()
                snapshot = snap
                updateMenuBar()
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    func moveInboxTodo(_ todo: TodoItem) {
        Task {
            do {
                let snap = try await cli.inboxMove(
                    line: todo.line,
                    date: dateString,
                    expectText: todo.text
                )
                snapshot = snap
                await loadInbox()
                updateMenuBar()
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    func toggleInboxTodo(_ todo: TodoItem) {
        Task {
            do {
                inboxSnapshot = try await cli.inboxToggle(
                    line: todo.line,
                    expectText: todo.text
                )
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    func deleteInboxTodo(_ todo: TodoItem) {
        Task {
            do {
                inboxSnapshot = try await cli.inboxDelete(
                    line: todo.line,
                    expectText: todo.text
                )
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    func addInboxTodo() {
        let text = newInboxText
            .trimmingCharacters(in: .whitespaces)
        guard !text.isEmpty else { return }
        newInboxText = ""
        Task {
            do {
                inboxSnapshot = try await cli.inboxAdd(
                    text: text
                )
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    // MARK: - Watch

    func startWatching() {
        let date = dateString
        guard watchTask == nil || watchedDate != date
        else { return }
        watchTask?.cancel()
        watchedDate = date
        watchTask = Task {
            let stream = await cli.watch(date: date)
            for await snap in stream {
                guard !Task.isCancelled else { break }
                snapshot = snap
                if let count = snap.inboxCount,
                   count != inboxOpenCount {
                    await loadInbox()
                }
                updateMenuBar()
            }
        }
    }

    func stopWatching() {
        watchTask?.cancel()
        watchTask = nil
        watchedDate = nil
        Task { await cli.stopWatch() }
    }

    // MARK: - Private

    private func loadStatus() async {
        do {
            snapshot = try await cli.status(
                date: dateString
            )
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func loadWeek() async {
        do {
            weekSummary = try await cli.week(
                date: dateString
            )
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func loadInbox() async {
        do {
            inboxSnapshot = try await cli.inbox()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func updateMenuBar() {
        onMenuBarUpdate?(menuBarTitle)
    }

    private static let formatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        return f
    }()
}
