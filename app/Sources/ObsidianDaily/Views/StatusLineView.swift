import SwiftUI

struct StatusLineView: View {
    @Bindable var viewModel: DailyViewModel

    var body: some View {
        HStack {
            Text(viewModel.statusText)
                .font(Theme.captionFont)
                .foregroundStyle(Theme.textSecondary)

            Spacer()

            if viewModel.carryOverCount > 0 {
                Button {
                    viewModel.carryOver()
                } label: {
                    Text(
                        "Carry over (\(viewModel.carryOverCount))"
                    )
                    .font(Theme.captionFont)
                    .foregroundStyle(Color.white)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(Theme.accent.opacity(0.6))
                    .clipShape(
                        RoundedRectangle(cornerRadius: 5)
                    )
                    .fixedSize()
                }
                .buttonStyle(.plain)
                .help("Move open todos from the previous day")
            }

            Toggle("Hide done", isOn: $viewModel.hideDone)
                .toggleStyle(.switch)
                .controlSize(.mini)
                .font(Theme.captionFont)
                .foregroundStyle(Theme.textSecondary)
                .fixedSize()
        }
    }
}
