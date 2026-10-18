import Foundation

struct DaySummary: Codable, Sendable, Identifiable {
    let date: String
    let openCount: Int
    let doneCount: Int
    let exists: Bool
    let isToday: Bool

    var id: String { date }
}

struct WeekSummary: Codable, Sendable {
    let state: String
    let days: [DaySummary]?
    let errorCode: String?
    let error: String?

    var isOk: Bool { state == "ok" }
}
