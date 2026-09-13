import SwiftUI

struct StatusBadge: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String
    let state: GecitRuntimeState
    let theme: AppTheme

    var body: some View {
        Text(title)
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(foreground)
            .padding(.horizontal, 12)
            .padding(.vertical, 6)
            .background(background, in: Capsule())
            .animation(.easeInOut(duration: 0.22), value: state)
    }

    private var foreground: Color {
        switch state {
        case .running:
            return Color.green
        case .starting, .stopping:
            return Color.orange
        default:
            return theme.textPrimary
        }
    }

    private var background: Color {
        switch state {
        case .running:
            return Color.green.opacity(colorScheme == .dark ? 0.16 : 0.14)
        case .starting, .stopping:
            return Color.orange.opacity(colorScheme == .dark ? 0.18 : 0.14)
        default:
            return theme.badgeBackground
        }
    }
}
