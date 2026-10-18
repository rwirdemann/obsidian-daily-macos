import SwiftUI

struct StatusLineView: View {
    @Bindable var viewModel: DailyViewModel

    var body: some View {
        HStack {
            Text(viewModel.statusText)
                .font(Theme.captionFont)
                .foregroundStyle(Theme.textSecondary)

            Spacer()

            Toggle("Hide done", isOn: $viewModel.hideDone)
                .toggleStyle(.switch)
                .controlSize(.mini)
                .font(Theme.captionFont)
                .foregroundStyle(Theme.textSecondary)
        }
    }
}
