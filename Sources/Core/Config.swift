import Foundation

struct Config: Codable {
    var runMas: Bool
    var runCleanup: Bool
    var runBrokenCaskCheck: Bool
    var greedyUpgrade: Bool
    var dryRun: Bool
    var backupBeforeUpgrade: Bool
    var preScript: String
    var postScript: String

    static let defaultConfig = Config(
        runMas: true,
        runCleanup: true,
        runBrokenCaskCheck: true,
        greedyUpgrade: true,
        dryRun: false,
        backupBeforeUpgrade: false,
        preScript: "",
        postScript: ""
    )

    static let configDir: URL = {
        let dir = FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent(".config/buum-cli")
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        return dir
    }()

    static let configPath: URL = configDir.appendingPathComponent("config.json")

    static func load() -> Config {
        guard FileManager.default.fileExists(atPath: configPath.path),
              let data = try? Data(contentsOf: configPath),
              let config = try? JSONDecoder().decode(Config.self, from: data) else {
            return defaultConfig
        }
        return config
    }

    func save() {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        guard let data = try? encoder.encode(self) else { return }
        try? data.write(to: Config.configPath)
    }

    static func reset() {
        defaultConfig.save()
    }

    /// All config keys and their descriptions
    static let keys: [(key: String, description: String)] = [
        ("runMas", "Include Mac App Store apps in updates"),
        ("runCleanup", "Clean cache after upgrades"),
        ("runBrokenCaskCheck", "Detect and disable broken casks"),
        ("greedyUpgrade", "Update auto-updating casks too"),
        ("dryRun", "Preview changes without installing"),
        ("backupBeforeUpgrade", "Backup Brewfile before upgrading"),
        ("preScript", "Shell script to run before updates"),
        ("postScript", "Shell script to run after updates"),
    ]

    mutating func set(key: String, value: String) -> Bool {
        switch key {
        case "runMas":              runMas = value.boolValue
        case "runCleanup":         runCleanup = value.boolValue
        case "runBrokenCaskCheck": runBrokenCaskCheck = value.boolValue
        case "greedyUpgrade":      greedyUpgrade = value.boolValue
        case "dryRun":             dryRun = value.boolValue
        case "backupBeforeUpgrade": backupBeforeUpgrade = value.boolValue
        case "preScript":          preScript = value
        case "postScript":         postScript = value
        default: return false
        }
        return true
    }

    func get(key: String) -> String? {
        switch key {
        case "runMas":              return String(runMas)
        case "runCleanup":         return String(runCleanup)
        case "runBrokenCaskCheck": return String(runBrokenCaskCheck)
        case "greedyUpgrade":      return String(greedyUpgrade)
        case "dryRun":             return String(dryRun)
        case "backupBeforeUpgrade": return String(backupBeforeUpgrade)
        case "preScript":          return preScript.isEmpty ? "(empty)" : preScript
        case "postScript":         return postScript.isEmpty ? "(empty)" : postScript
        default: return nil
        }
    }
}

private extension String {
    var boolValue: Bool {
        ["true", "1", "yes", "on"].contains(self.lowercased())
    }
}
