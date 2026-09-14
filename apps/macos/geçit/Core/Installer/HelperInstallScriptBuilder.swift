import Foundation

struct HelperInstallScriptBuilder {
    func render(daemonURL: String, plistURL: String, binaryURL: String) -> String {
        """
        #!/bin/bash
        set -euo pipefail
        OWNER_USER="$(stat -f %Su /dev/console 2>/dev/null || true)"
        case "$OWNER_USER" in
          ""|root)
            echo "no console user available, aborting install" >&2
            exit 1
            ;;
        esac
        mkdir -p "/Library/Application Support/Gecit"
        install -m 755 "\(daemonURL)" "\(AppPaths.helperScriptPath)"
        install -m 755 "\(binaryURL)" "\(AppPaths.installedBinaryPath)"
        install -m 644 "\(plistURL)" "\(AppPaths.helperPlistPath)"
        echo "\(AppPaths.helperVersion)" > "\(AppPaths.helperVersionPath)"
        mkdir -p "\(AppPaths.sharedDirectory)"
        touch "\(AppPaths.commandFile)" "\(AppPaths.logFile)" "\(AppPaths.statusFile)"
        chmod 700 "\(AppPaths.sharedDirectory)"
        chmod 600 "\(AppPaths.commandFile)" "\(AppPaths.logFile)" "\(AppPaths.statusFile)"
        chown "$OWNER_USER" "\(AppPaths.sharedDirectory)" "\(AppPaths.commandFile)" "\(AppPaths.logFile)" "\(AppPaths.statusFile)"
        # The pid file is kept intact: it is how a restarted daemon still recognises
        # an engine that is already running.
        if [ -f "\(AppPaths.pidFile)" ]; then
          chmod 600 "\(AppPaths.pidFile)"
          chown "$OWNER_USER" "\(AppPaths.pidFile)"
        fi
        launchctl bootout system/\(AppPaths.helperIdentifier) >/dev/null 2>&1 || true
        launchctl bootstrap system "\(AppPaths.helperPlistPath)"
        launchctl kickstart -k system/\(AppPaths.helperIdentifier)
        """
    }
}
