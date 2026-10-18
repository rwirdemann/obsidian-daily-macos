import SwiftUI

struct InboxListView: View {
    @Bindable var viewModel: DailyViewModel

    var body: some View {
        ScrollView {
            LazyVStack(spacing: 0) {
                ForEach(viewModel.inboxTodos) { todo in
                    InboxRowView(
                        todo: todo,
                        onMove: {
                            viewModel.moveInboxTodo(todo)
                        },
                        onDelete: {
                            viewModel.deleteInboxTodo(
                                todo
                            )
                        }
                    )
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 4)
        }
        .frame(maxHeight: .infinity)
    }
}

struct InboxRowView: View {
    let todo: TodoItem
    let onMove: () -> Void
    let onDelete: () -> Void

    @State private var isHovered = false

    var body: some View {
        HStack(spacing: 8) {
            Image(systemName: "tray")
                .font(.system(size: 11))
                .foregroundStyle(Theme.textSecondary)

            Text(todo.text)
                .font(Theme.bodyFont)
                .foregroundStyle(Theme.textPrimary)
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

            Button(action: onMove) {
                Image(
                    systemName: "arrow.right.circle"
                )
                .font(.system(size: 14))
                .foregroundStyle(Theme.accent)
            }
            .buttonStyle(.plain)
            .help("Move to daily note")
        }
        .padding(.vertical, 4)
        .contentShape(Rectangle())
        .onHover { isHovered = $0 }
    }
}

struct AddInboxView: View {
    @Bindable var viewModel: DailyViewModel
    @FocusState private var isFocused: Bool

    var body: some View {
        HStack(spacing: 8) {
            TextField(
                "Add to inbox…",
                text: $viewModel.newInboxText
            )
            .textFieldStyle(.plain)
            .font(Theme.bodyFont)
            .foregroundStyle(Theme.textPrimary)
            .focused($isFocused)
            .onSubmit { viewModel.addInboxTodo() }

            Button {
                viewModel.addInboxTodo()
            } label: {
                Image(systemName: "plus.circle.fill")
                    .font(.system(size: 16))
                    .foregroundStyle(Theme.accent)
            }
            .buttonStyle(.plain)
            .disabled(
                viewModel.newInboxText
                    .trimmingCharacters(
                        in: .whitespaces
                    )
                    .isEmpty
            )
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 5)
        .background(Theme.surface)
        .clipShape(RoundedRectangle(cornerRadius: 6))
    }
}
