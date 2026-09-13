import Foundation

enum AppPaths {
    static let helperVersion = "4"
    static let onboardingVersion = "1"
    static let helperIdentifier = "com.ahmetshbz.gecit.helper"
    static let helperScriptPath = "/Library/Application Support/Gecit/gecit-helper.sh"
    static let helperPlistPath = "/Library/LaunchDaemons/\(helperIdentifier).plist"
    static let installedBinaryPath = "/Library/Application Support/Gecit/gecit-darwin-arm64"
    static let helperVersionPath = "/Library/Application Support/Gecit/helper-version"
    static let sharedDirectory = "/Users/Shared/GecitHelper"
    static let commandFile = sharedDirectory + "/command"
    static let statusFile = sharedDirectory + "/status.json"
    static let logFile = sharedDirectory + "/gecit.log"

    static var bundledBinaryPath: String {
        if let bundled = Bundle.main.path(forResource: "gecit-darwin-arm64", ofType: nil) {
            return bundled
        }
        return "/Users/ahmet/Desktop/gecit/apps/macos/geçit/Resources/gecit-darwin-arm64"
    }
}
