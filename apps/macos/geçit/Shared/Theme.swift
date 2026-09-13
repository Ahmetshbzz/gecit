import SwiftUI

struct AppTheme {
    let backgroundBase: Color
    let backgroundAccent: Color
    let card: Color
    let cardStrong: Color
    let textPrimary: Color
    let textSecondary: Color
    let textMuted: Color
    let logBackground: Color
    let badgeBackground: Color
    let divider: Color
    let primaryButtonStart: Color
    let primaryButtonStop: Color
    let primaryButtonText: Color
    let disabledButton: Color
    let disabledButtonText: Color

    static func current(for scheme: ColorScheme) -> AppTheme {
        if scheme == .light {
            return AppTheme(
                backgroundBase: Color.white,
                backgroundAccent: Color.black.opacity(0.03),
                card: Color.black.opacity(0.05),
                cardStrong: Color.black.opacity(0.035),
                textPrimary: Color.black.opacity(0.92),
                textSecondary: Color.black.opacity(0.66),
                textMuted: Color.black.opacity(0.5),
                logBackground: Color.black.opacity(0.06),
                badgeBackground: Color.black.opacity(0.06),
                divider: Color.black.opacity(0.08),
                primaryButtonStart: Color(red: 0.10, green: 0.58, blue: 0.32),
                primaryButtonStop: Color(red: 0.82, green: 0.22, blue: 0.18),
                primaryButtonText: .white,
                disabledButton: Color.black.opacity(0.08),
                disabledButtonText: Color.black.opacity(0.35)
            )
        }

        return AppTheme(
            backgroundBase: Color.black,
            backgroundAccent: Color.white.opacity(0.03),
            card: Color.white.opacity(0.06),
            cardStrong: Color.white.opacity(0.045),
            textPrimary: .white,
            textSecondary: Color.white.opacity(0.80),
            textMuted: Color.white.opacity(0.58),
            logBackground: Color.white.opacity(0.06),
            badgeBackground: Color.white.opacity(0.08),
            divider: Color.white.opacity(0.10),
            primaryButtonStart: Color(red: 0.20, green: 0.78, blue: 0.42),
            primaryButtonStop: Color(red: 0.95, green: 0.32, blue: 0.28),
            primaryButtonText: .white,
            disabledButton: Color.white.opacity(0.12),
            disabledButtonText: Color.white.opacity(0.4)
        )
    }
}

