import SwiftUI

struct VaultSetupView: View {
    @Bindable var viewModel: DailyViewModel

    var body: some View {
        VStack(spacing: 12) {
            Spacer()

            Image(systemName: "folder.badge.questionmark")
                .font(.system(size: 40))
                .foregroundStyle(Theme.accent)

            Text("No vault selected")
                .font(Theme.headerFont)
                .foregroundStyle(Theme.textPrimary)

            Text(
                "Choose the folder of your Obsidian vault "
                    + "to show its daily notes."
            )
            .font(Theme.captionFont)
            .foregroundStyle(Theme.textSecondary)
            .multilineTextAlignment(.center)

            Button("Choose Vault…") {
                viewModel.chooseVault()
            }
            .buttonStyle(.borderedProminent)

            Spacer()
        }
        .padding(.horizontal, 32)
    }
}
