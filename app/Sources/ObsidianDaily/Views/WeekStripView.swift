import SwiftUI

struct WeekStripView: View {
    @Bindable var viewModel: DailyViewModel

    private static let dayLabels = [
        "Mon", "Tue", "Wed", "Thu",
        "Fri", "Sat", "Sun",
    ]

    var body: some View {
        HStack(spacing: 4) {
            inboxCell
                .onTapGesture {
                    viewModel.selectInbox()
                }

            if let days = viewModel.weekSummary?.days {
                ForEach(days) { day in
                    dayCellView(day: day)
                        .onTapGesture {
                            if let date = Self.parse(
                                day.date
                            ) {
                                viewModel.selectDate(
                                    date
                                )
                            }
                        }
                }
            } else {
                ForEach(Self.dayLabels, id: \.self) {
                    label in
                    placeholderCell(label: label)
                }
            }
        }
    }

    private var inboxCell: some View {
        let count = viewModel.inboxOpenCount
        let isSelected = viewModel.showingInbox

        return VStack(spacing: 2) {
            Text("Inbox")
                .font(.system(size: 10))
                .foregroundStyle(Theme.textSecondary)

            Text("\(count)")
                .font(.system(
                    size: 12, weight: .medium
                ))
                .foregroundStyle(
                    isSelected
                        ? Color.white
                        : Theme.textPrimary
                )

            if count > 0 {
                Circle()
                    .fill(Color.orange)
                    .frame(width: 4, height: 4)
            } else {
                Circle()
                    .fill(Color.clear)
                    .frame(width: 4, height: 4)
            }
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 4)
        .background(
            isSelected
                ? Theme.accent.opacity(0.3)
                : Color.clear
        )
        .clipShape(RoundedRectangle(cornerRadius: 6))
    }

    @ViewBuilder
    private func dayCellView(
        day: DaySummary
    ) -> some View {
        let isSelected = !viewModel.showingInbox
            && day.date == viewModel.dateString

        VStack(spacing: 2) {
            Text(Self.shortDay(from: day.date))
                .font(.system(size: 10))
                .foregroundStyle(Theme.textSecondary)

            Text(Self.dayNum(from: day.date))
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(
                    isSelected
                        ? Color.white
                        : Theme.textPrimary
                )

            dotIndicator(day: day)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 4)
        .background(
            isSelected
                ? Theme.accent.opacity(0.3)
                : Color.clear
        )
        .clipShape(RoundedRectangle(cornerRadius: 6))
    }

    @ViewBuilder
    private func dotIndicator(
        day: DaySummary
    ) -> some View {
        if day.openCount > 0 {
            Circle()
                .fill(Theme.accent)
                .frame(width: 4, height: 4)
        } else if day.doneCount > 0 {
            Circle()
                .strokeBorder(
                    Theme.textSecondary,
                    lineWidth: 1
                )
                .frame(width: 4, height: 4)
        } else {
            Circle()
                .fill(Color.clear)
                .frame(width: 4, height: 4)
        }
    }

    private func placeholderCell(
        label: String
    ) -> some View {
        VStack(spacing: 2) {
            Text(label)
                .font(.system(size: 10))
                .foregroundStyle(Theme.textSecondary)
            Text("–")
                .font(.system(size: 12))
                .foregroundStyle(Theme.textSecondary)
            Circle()
                .fill(Color.clear)
                .frame(width: 4, height: 4)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 4)
    }

    private static func parse(
        _ dateStr: String
    ) -> Date? {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        return f.date(from: dateStr)
    }

    private static func shortDay(
        from dateStr: String
    ) -> String {
        guard let date = parse(dateStr) else {
            return "?"
        }
        let f = DateFormatter()
        f.dateFormat = "EEE"
        return f.string(from: date)
    }

    private static func dayNum(
        from dateStr: String
    ) -> String {
        guard let date = parse(dateStr) else {
            return "?"
        }
        let f = DateFormatter()
        f.dateFormat = "d"
        return f.string(from: date)
    }
}
