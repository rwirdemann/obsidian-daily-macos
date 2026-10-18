import SwiftUI

struct TodoListView: View {
    @Bindable var viewModel: DailyViewModel

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                ForEach(viewModel.filteredTodos) { todo in
                    TodoRowView(
                        todo: todo,
                        onToggle: {
                            viewModel.toggleTodo(todo)
                        },
                        onDelete: {
                            viewModel.deleteTodo(todo)
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
