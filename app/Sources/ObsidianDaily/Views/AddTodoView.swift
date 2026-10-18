import SwiftUI

struct AddTodoView: View {
    @Bindable var viewModel: DailyViewModel
    @FocusState private var isFocused: Bool

    var body: some View {
        HStack(spacing: 8) {
            TextField(
                "Add a todo…",
                text: $viewModel.newTodoText
            )
            .textFieldStyle(.plain)
            .font(Theme.bodyFont)
            .foregroundStyle(Theme.textPrimary)
            .focused($isFocused)
            .onSubmit { viewModel.addTodo() }

            Button {
                viewModel.addTodo()
            } label: {
                Image(systemName: "plus.circle.fill")
                    .font(.system(size: 16))
                    .foregroundStyle(Theme.accent)
            }
            .buttonStyle(.plain)
            .disabled(
                viewModel.newTodoText
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
