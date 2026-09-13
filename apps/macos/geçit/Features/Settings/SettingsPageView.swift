import SwiftUI

struct SettingsPageView: View {
    @Environment(\.colorScheme) private var colorScheme
    @ObservedObject var model: AppModel
    let theme: AppTheme

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            PageHeader(title: "Ayarlar", theme: theme) {
                model.currentPage = .main
            }

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Button {
                        model.resetSettingsToDefault()
                    } label: {
                        HoverCapsuleContent(helpText: "Varsayılan ayarlara dön") {
                            Text("Varsayılana dön")
                                .font(.system(size: 12, weight: .semibold))
                        }
                    }
                    .buttonStyle(ScaleButtonStyle())
                    .focusEffectDisabled()

                    section("Bağlantı") {
                        SettingsField(title: "Fake TTL", theme: theme) {
                            NativeStepperField(value: $model.settingsFakeTTL, minValue: 1, maxValue: 64)
                        }

                        divider

                        SettingsField(title: "DoH", theme: theme) {
                            Toggle("Etkin", isOn: $model.settingsDoHEnabled)
                                .toggleStyle(.switch)
                        }

                        divider

                        SettingsField(title: "Upstream", theme: theme) {
                            Picker("Upstream", selection: $model.settingsDoHUpstream) {
                                ForEach(AppModel.dohPresets, id: \.self) { preset in
                                    Text(preset.capitalized).tag(preset)
                                }
                            }
                            .pickerStyle(.menu)
                            .labelsHidden()
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }

                        divider

                        SettingsField(title: "Interface", theme: theme) {
                            TextField("en0", text: $model.settingsInterface)
                                .textFieldStyle(.roundedBorder)
                        }

                        divider

                        SettingsField(title: "Ports", theme: theme) {
                            TextField("443 veya 443,8443", text: $model.settingsPorts)
                                .textFieldStyle(.roundedBorder)
                        }
                    }

                    section("Tanılama") {
                        infoRow("Servis", model.helperInstalled ? "Hazır" : "Yeniden kurulum gerekli")
                        divider
                        HStack {
                            Text("Runtime")
                                .font(.system(size: 13, weight: .medium))
                                .foregroundStyle(theme.textMuted)
                                .frame(width: 84, alignment: .leading)
                            StatusBadge(title: model.statusTitle, state: model.status.state, theme: theme)
                            Spacer(minLength: 0)
                        }
                        divider
                        infoRow("PID", model.status.pid.map(String.init) ?? "—")
                        divider
                        infoRow("Mesaj", model.status.message)
                        divider
                        infoRow("Ayarlar", model.currentSettingsSummary)
                        divider
                        infoRow("Log dosyası", AppPaths.logFile)
                    }
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
    }

    private var divider: some View {
        Divider().overlay(theme.divider)
    }

    private func section<Content: View>(_ title: String, @ViewBuilder content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(title)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(theme.textMuted)

            content()
        }
    }

    private func infoRow(_ key: String, _ value: String) -> some View {
        HStack(alignment: .top) {
            Text(key)
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(theme.textMuted)
                .frame(width: 84, alignment: .leading)
            Text(value)
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(theme.textPrimary)
                .textSelection(.enabled)
            Spacer(minLength: 0)
        }
    }
}
