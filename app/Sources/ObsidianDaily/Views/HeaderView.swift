import SwiftUI

struct HeaderView: View {
    @Bindable var viewModel: DailyViewModel

    var body: some View {
        HStack(spacing: 8) {
            Image(systemName: "doc.text")
                .font(.system(size: 16))
                .foregroundStyle(Theme.accent)

            Text("Obsidian Daily")
                .font(Theme.headerFont)
                .foregroundStyle(Theme.textPrimary)

            Spacer()

            Text(viewModel.displayDate)
                .font(Theme.captionFont)
                .foregroundStyle(Theme.textSecondary)

            Button {
                viewModel.navigateDay(-1)
            } label: {
                Image(systemName: "chevron.left")
                    .font(.system(size: 11))
            }
            .buttonStyle(.plain)
            .foregroundStyle(Theme.textSecondary)

            Button {
                viewModel.navigateDay(1)
            } label: {
                Image(systemName: "chevron.right")
                    .font(.system(size: 11))
            }
            .buttonStyle(.plain)
            .foregroundStyle(Theme.textSecondary)

            Button {
                viewModel.chooseVault()
            } label: {
                Image(systemName: "folder")
                    .font(.system(size: 12))
            }
            .buttonStyle(.plain)
            .foregroundStyle(Theme.textSecondary)
            .help("Change vault")

            Button {
                viewModel.openInObsidian()
            } label: {
                Image(
                    systemName: "arrow.up.right.square"
                )
                .font(.system(size: 12))
            }
            .buttonStyle(.plain)
            .foregroundStyle(Theme.textSecondary)
            .help("Open in Obsidian")
        }
    }
}
