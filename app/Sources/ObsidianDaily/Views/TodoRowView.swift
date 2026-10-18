import SwiftUI

struct TodoRowView: View {
    let todo: TodoItem
    let onToggle: () -> Void
    let onDelete: () -> Void

    @State private var isHovered = false

    var body: some View {
        HStack(spacing: 8) {
            Button(action: onToggle) {
                Image(
                    systemName: todo.checked
                        ? "checkmark.circle.fill"
                        : "circle"
                )
                .font(.system(size: 14))
                .foregroundStyle(
                    todo.checked
                        ? Theme.todoDone
                        : Theme.accent
                )
            }
            .buttonStyle(.plain)

            Text(todo.text)
                .font(Theme.bodyFont)
                .foregroundStyle(
                    todo.checked
                        ? Theme.todoDone
                        : Theme.textPrimary
                )
                .strikethrough(todo.checked)
                .lineLimit(2)

            Spacer()

            if isHovered {
                Button(action: onDelete) {
                    Image(
                        systemName: "xmark.circle"
                    )
                    .font(.system(size: 12))
                    .foregroundStyle(
                        Theme.textSecondary
                    )
                }
                .buttonStyle(.plain)
                .help("Delete")
            }
        }
        .padding(.leading, CGFloat(todo.depth) * 16)
        .padding(.vertical, 4)
        .contentShape(Rectangle())
        .onHover { isHovered = $0 }
    }
}
