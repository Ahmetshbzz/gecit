import SwiftUI

struct LogsPageView: View {
    @Environment(\.colorScheme) private var colorScheme
    @ObservedObject var model: AppModel
    let theme: AppTheme

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            PageHeader(title: "Loglar", theme: theme) {
                model.currentPage = .main
            }

            ScrollViewReader { proxy in
                ScrollView {
                    VStack(alignment: .leading, spacing: 0) {
                        Text(attributedLogs)
                            .font(.system(size: 11, weight: .medium, design: .monospaced))
                            .lineSpacing(3)
                            .textSelection(.enabled)
                            .frame(maxWidth: .infinity, alignment: .leading)
                        Color.clear
                            .frame(height: 1)
                            .id("logs-bottom")
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .onAppear {
                    scrollToBottom(proxy)
                }
                .onChange(of: model.logs) { _, _ in
                    scrollToBottom(proxy)
                }
            }
        }
    }

    private var attributedLogs: AttributedString {
        let lines = model.logs
            .split(separator: "\n", omittingEmptySubsequences: false)
            .map(String.init)

        let source = lines.isEmpty ? ["Henüz log yok."] : lines
        var output = AttributedString()

        for (index, entry) in source.enumerated() {
            var line = AttributedString(entry.isEmpty ? " " : entry)
            line.foregroundColor = logTextColor(for: entry)
            output.append(line)
            if index < source.count - 1 {
                output.append(AttributedString("\n"))
            }
        }

        return output
    }

    private func logTextColor(for entry: String) -> Color {
        if entry.contains("level=error") || entry.contains("error") {
            return colorScheme == .dark ? Color.red.opacity(0.95) : Color.red.opacity(0.85)
        }
        if entry.contains("level=info") || entry.contains("injected") || entry.contains("running") {
            return colorScheme == .dark ? Color.green.opacity(0.95) : Color.green.opacity(0.78)
        }
        if entry.contains("level=debug") || entry.contains("resolved") {
            return colorScheme == .dark ? Color.blue.opacity(0.92) : Color.blue.opacity(0.78)
        }
        if entry.contains("warning") || entry.contains("stopping") || entry.contains("starting") {
            return colorScheme == .dark ? Color.orange.opacity(0.95) : Color.orange.opacity(0.82)
        }
        return theme.textSecondary
    }

    private func scrollToBottom(_ proxy: ScrollViewProxy) {
        DispatchQueue.main.async {
            withAnimation(.easeOut(duration: 0.24)) {
                proxy.scrollTo("logs-bottom", anchor: .bottom)
            }
        }
    }
}
