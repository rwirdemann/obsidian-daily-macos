import SwiftUI

struct PopoverContentView: View {
    @Bindable var viewModel: DailyViewModel

    var body: some View {
        VStack(spacing: 0) {
            HeaderView(viewModel: viewModel)
                .padding(.horizontal, 16)
                .padding(.top, 12)
                .padding(.bottom, 8)

            WeekStripView(viewModel: viewModel)
                .padding(.horizontal, 16)
                .padding(.bottom, 8)

            if viewModel.needsVault {
                VaultSetupView(viewModel: viewModel)
            } else if viewModel.showingInbox {
                inboxContent
            } else {
                dailyContent
            }

            if let error = viewModel.errorMessage {
                Text(error)
                    .font(Theme.captionFont)
                    .foregroundStyle(.red)
                    .padding(.horizontal, 16)
                    .padding(.bottom, 8)
                    .onTapGesture {
                        viewModel.errorMessage = nil
                    }
            }
        }
        .frame(width: 360, height: 520)
        .background(Theme.background)
        .onAppear { viewModel.refresh() }
    }

    private var dailyContent: some View {
        VStack(spacing: 0) {
            SearchBarView(
                searchText: $viewModel.searchText
            )
            .padding(.horizontal, 16)
            .padding(.bottom, 6)

            StatusLineView(viewModel: viewModel)
                .padding(.horizontal, 16)
                .padding(.bottom, 6)

            Divider()
                .background(Theme.surface)

            TodoListView(viewModel: viewModel)

            Divider()
                .background(Theme.surface)

            AddTodoView(viewModel: viewModel)
                .padding(.horizontal, 16)
                .padding(.vertical, 10)
        }
    }

    private var inboxContent: some View {
        VStack(spacing: 0) {
            HStack {
                Text(
                    "\(viewModel.inboxOpenCount) open"
                )
                .font(Theme.captionFont)
                .foregroundStyle(Theme.textSecondary)

                Spacer()
            }
            .padding(.horizontal, 16)
            .padding(.bottom, 6)

            Divider()
                .background(Theme.surface)

            InboxListView(viewModel: viewModel)

            Divider()
                .background(Theme.surface)

            AddInboxView(viewModel: viewModel)
                .padding(.horizontal, 16)
                .padding(.vertical, 10)
        }
    }
}
