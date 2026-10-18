import Foundation

struct TodoItem: Codable, Sendable, Identifiable {
    let line: Int
    let checked: Bool
    let text: String
    let depth: Int
    let parentLine: Int?

    var id: Int { line }
}

struct Snapshot: Codable, Sendable {
    let state: String
    let date: String?
    let path: String?
    let exists: Bool?
    let openCount: Int?
    let doneCount: Int?
    let todos: [TodoItem]?
    let obsidianUri: String?
    let carryOverCount: Int?
    let isToday: Bool?
    let templateName: String?
    let createdFromTemplate: Bool?
    let inboxCount: Int?
    let errorCode: String?
    let error: String?

    var isOk: Bool { state == "ok" }
}
