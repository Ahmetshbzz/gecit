import SwiftUI

struct PageHeader: View {
    let title: String
    let theme: AppTheme
    let onBack: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 10) {
                Button(action: onBack) {
                    HoverCapsuleContent(helpText: "Geri") {
                        Image(systemName: "chevron.left")
                            .font(.system(size: 14, weight: .semibold))
                    }
                }
                .buttonStyle(ScaleButtonStyle())
                .focusEffectDisabled()

                Spacer()

                Text(title)
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(theme.textPrimary)
            }
        }
    }
}
