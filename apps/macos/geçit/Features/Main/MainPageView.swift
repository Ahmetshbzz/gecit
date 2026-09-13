import SwiftUI

struct MainPageView: View {
    @Environment(\.colorScheme) private var colorScheme
    @ObservedObject var model: AppModel
    let theme: AppTheme

    var body: some View {
        VStack(spacing: 0) {
            topBar

            Spacer(minLength: 0)

            PrimaryActionButton(
                title: model.primaryActionTitle,
                symbol: model.primaryActionSymbol,
                enabled: true,
                state: model.status.state,
                theme: theme,
                colorScheme: colorScheme
            ) {
                model.performPrimaryAction()
            }

            StatusBadge(title: model.statusTitle, state: model.status.state, theme: theme)
                .padding(.top, 16)

            Spacer(minLength: 0)

            statusDetails
        }
    }

    private var topBar: some View {
        HStack(spacing: 10) {
            Text("geçit")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(theme.textPrimary)

            Spacer()

            Button {
                model.currentPage = .logs
            } label: {
                HoverCapsuleContent(helpText: "Loglar") {
                    Image(systemName: "doc.text.magnifyingglass")
                        .font(.system(size: 14, weight: .semibold))
                }
            }
            .buttonStyle(ScaleButtonStyle())
            .focusEffectDisabled()

            Button {
                model.currentPage = .settings
            } label: {
                HoverCapsuleContent(helpText: "Ayarlar") {
                    Image(systemName: "gearshape")
                        .font(.system(size: 14, weight: .semibold))
                }
            }
            .buttonStyle(ScaleButtonStyle())
            .focusEffectDisabled()
        }
    }

    private var statusDetails: some View {
        VStack(alignment: .leading, spacing: 10) {
            detailRow("Servis", model.helperInstalled ? "Hazır" : "Yeniden kurulum gerekli")
            divider
            detailRow("PID", model.status.pid.map(String.init) ?? "—")
            divider
            VStack(alignment: .leading, spacing: 4) {
                Text("Mesaj")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(theme.textMuted)
                Text(model.status.message)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(theme.textPrimary)
                    .textSelection(.enabled)
                    .fixedSize(horizontal: false, vertical: true)
            }
            divider
            VStack(alignment: .leading, spacing: 4) {
                Text("Ayarlar")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(theme.textMuted)
                Text(model.currentSettingsSummary)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(theme.textPrimary)
                    .textSelection(.enabled)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .padding(.top, 16)
        .animation(.easeInOut(duration: 0.2), value: model.status.state)
    }

    private var divider: some View {
        Divider().overlay(theme.divider)
    }

    private func detailRow(_ key: String, _ value: String) -> some View {
        HStack(alignment: .firstTextBaseline) {
            Text(key)
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(theme.textMuted)
            Spacer(minLength: 12)
            Text(value)
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(theme.textPrimary)
                .textSelection(.enabled)
        }
    }
}
