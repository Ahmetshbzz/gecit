# gecit macOS app

This directory contains the native Swift macOS client for `gecit`.

The app is a menu bar control plane for the bundled runtime binary. It does not implement DPI bypass itself; it installs and manages the packaged `gecit-darwin-arm64` executable.

## Responsibilities

- present onboarding UI
- request privileged helper installation
- install launchd helper assets
- start / stop / cleanup the runtime
- persist user settings
- display status and tail logs in a native popover UI

## Runtime model

The app uses a split model:

1. **Swift UI layer** — menu bar app, onboarding, settings, logs, status
2. **Privileged helper/runtime layer** — installed binary and helper assets under `/Library/Application Support/Gecit`

Control and state move through shared files:

- directory: `/Users/Shared/GecitHelper` (mode `700`, owned by the console user)
- command file: `/Users/Shared/GecitHelper/command` (mode `600`)
- status file: `/Users/Shared/GecitHelper/status.json` (mode `600`)
- log file: `/Users/Shared/GecitHelper/gecit.log` (mode `600`, capped at 5 MiB: rotated to `gecit.log.1` on start, truncated in place while running)
- pid file: `/Users/Shared/GecitHelper/gecit.pid` (mode `600`)

The helper runs as root and re-applies owner and mode on every loop, so files it
recreates never stay readable or writable by other accounts. The log records every
visited host name, so it must not be world readable.

Only the `start` arguments the app itself emits are forwarded to the runtime binary:
`--fake-ttl` (1-255), `--doh` (`true`/`false`), `--doh-upstream` (preset name only),
`--interface` (interface name characters only) and `--ports` (comma separated, 1-65535).
Anything else is rejected without spawning the runtime.

## Main components

- `geçit/App/AppDelegate.swift` — app lifecycle, status bar item, popover, onboarding flow
- `geçit/Core/AppModel.swift` — UI-facing application model
- `geçit/Core/RuntimeStore.swift` — runtime state, polling, primary actions
- `geçit/Core/GecitHelperInstaller.swift` — helper installation pipeline
- `geçit/Core/GecitControlService.swift` — command/status/log file IO
- `geçit/Core/SettingsStore.swift` — persisted settings
- `geçit/Features/*` — onboarding, main page, logs page, settings page

## Installation flow

On first launch, the app shows onboarding and installs the helper with administrator privileges.

The installer:

- writes helper scripts and launchd plist to a temporary directory
- copies the bundled `gecit-darwin-arm64` binary
- marks assets executable
- runs the installation script with elevated privileges

After installation, the app can start and stop the runtime from the menu bar.

## Settings exposed by the UI

- fake TTL
- DoH enabled/disabled
- DoH upstream preset
- network interface override
- destination ports

These settings are converted into CLI arguments before the runtime starts. The helper
accepts only the presets listed in `SettingsStore.dohPresets`; custom upstream URLs can
be passed to the binary directly from the terminal, but never through the command file.

## Build

Open the Xcode project:

```text
geçit.xcodeproj
```

The app expects the runtime binary to be bundled as an app resource.

## Positioning

This app is the main product differentiation of the fork.

The upstream repository provides the underlying network engine; this directory turns that engine into a macOS-native user experience with installation, lifecycle management, and UI.