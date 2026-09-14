import Foundation

struct GecitHelperInstaller {
    private let plistBuilder: HelperLaunchdPlistBuilder
    private let daemonScriptBuilder: HelperDaemonScriptBuilder
    private let installScriptBuilder: HelperInstallScriptBuilder
    private let administratorScriptRunner: AdministratorScriptRunner

    init(
        plistBuilder: HelperLaunchdPlistBuilder = HelperLaunchdPlistBuilder(),
        daemonScriptBuilder: HelperDaemonScriptBuilder = HelperDaemonScriptBuilder(),
        installScriptBuilder: HelperInstallScriptBuilder = HelperInstallScriptBuilder(),
        administratorScriptRunner: AdministratorScriptRunner = AdministratorScriptRunner()
    ) {
        self.plistBuilder = plistBuilder
        self.daemonScriptBuilder = daemonScriptBuilder
        self.installScriptBuilder = installScriptBuilder
        self.administratorScriptRunner = administratorScriptRunner
    }

    func isInstalled() -> Bool {
        guard FileManager.default.fileExists(atPath: AppPaths.helperScriptPath),
              FileManager.default.fileExists(atPath: AppPaths.helperPlistPath),
              FileManager.default.fileExists(atPath: AppPaths.installedBinaryPath),
              let version = try? String(contentsOfFile: AppPaths.helperVersionPath, encoding: .utf8).trimmingCharacters(in: .whitespacesAndNewlines)
        else {
            return false
        }
        return version == AppPaths.helperVersion
    }

    func install() throws {
        guard let bundledBinaryPath = AppPaths.bundledBinaryPath else {
            throw NSError(
                domain: "GecitHelperInstaller",
                code: 1,
                userInfo: [NSLocalizedDescriptionKey: "Uygulama paketinde gecit-darwin-arm64 bulunamadı."]
            )
        }

        let tempDirectory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString, isDirectory: true)
        try FileManager.default.createDirectory(at: tempDirectory, withIntermediateDirectories: true)

        let daemonURL = tempDirectory.appendingPathComponent("gecit-helper.sh")
        let plistURL = tempDirectory.appendingPathComponent("\(AppPaths.helperIdentifier).plist")
        let installURL = tempDirectory.appendingPathComponent("install-helper.sh")
        let binaryURL = tempDirectory.appendingPathComponent("gecit-darwin-arm64")

        try daemonScriptBuilder.render().write(to: daemonURL, atomically: true, encoding: .utf8)
        try plistBuilder.render().write(to: plistURL, atomically: true, encoding: .utf8)
        try FileManager.default.copyItem(at: URL(fileURLWithPath: bundledBinaryPath), to: binaryURL)
        try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: binaryURL.path)
        try installScriptBuilder.render(daemonURL: daemonURL.path, plistURL: plistURL.path, binaryURL: binaryURL.path).write(to: installURL, atomically: true, encoding: .utf8)
        try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: installURL.path)

        try administratorScriptRunner.runScript(at: installURL.path)
    }
}
