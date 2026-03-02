import Foundation

struct BrewHelpers {
    struct OutdatedPackage {
        let name: String
        let current: String
        let latest: String
    }

    struct BrewService {
        let name: String
        let status: String

        var statusIcon: String {
            switch status {
            case "started": return "🟢"
            case "error":   return "🔴"
            case "stopped": return "⚫"
            default:        return "⚪"
            }
        }
    }

    static func fetchOutdated() -> [OutdatedPackage] {
        guard Shell.brewExists else { return [] }
        let result = Shell.run(Shell.brewPath, ["outdated", "--verbose"], showCommand: false, stream: false, silent: true)
        guard result.exitCode == 0 else { return [] }

        return result.stdout.components(separatedBy: "\n").compactMap { line in
            let parts = line.trimmingCharacters(in: .whitespaces).components(separatedBy: " ")
            guard parts.count >= 4 else { return nil }
            let name = parts[0]
            let current = parts[1].trimmingCharacters(in: CharacterSet(charactersIn: "()"))
            let latest = parts.last ?? ""
            return OutdatedPackage(name: name, current: current, latest: latest)
        }
    }

    static func fetchServices() -> [BrewService] {
        guard Shell.brewExists else { return [] }
        let result = Shell.run(Shell.brewPath, ["services", "list"], showCommand: false, stream: false, silent: true)
        guard result.exitCode == 0 else { return [] }

        return result.stdout.components(separatedBy: "\n")
            .dropFirst() // skip header
            .compactMap { line in
                let parts = line.split(separator: " ", omittingEmptySubsequences: true).map(String.init)
                guard parts.count >= 2 else { return nil }
                return BrewService(name: parts[0], status: parts[1])
            }
    }

    /// Parse output lines for "Warning: Skipping X: most recent version Y not installed"
    static func skippedPackages(from output: String) -> [String] {
        output.components(separatedBy: "\n")
            .filter { $0.contains("Warning: Skipping") && $0.contains("not installed") }
            .compactMap { line -> String? in
                let parts = line.components(separatedBy: "Skipping ")
                guard parts.count >= 2 else { return nil }
                return parts[1].components(separatedBy: ":").first?.trimmingCharacters(in: .whitespaces)
            }
    }

    /// Parse "Error: cask-name: It seems the App source ... is not there." lines
    static func brokenCaskUpgrades(from output: String) -> [String] {
        output.components(separatedBy: "\n")
            .filter { $0.contains("It seems the App source") || $0.contains("is not there") }
            .compactMap { line -> String? in
                let text = line.hasPrefix("Error: ") ? String(line.dropFirst(7)) : line
                return text.components(separatedBy: ":").first?.trimmingCharacters(in: .whitespaces)
            }
    }

    /// Find installed casks where `brew info --cask` fails
    static func findBrokenCasks() -> [String] {
        guard Shell.brewExists else { return [] }
        let result = Shell.run(Shell.brewPath, ["list", "--cask"], showCommand: false, stream: false, silent: true)
        guard result.exitCode == 0 else { return [] }

        let casks = result.stdout.split(separator: "\n").map(String.init).filter { !$0.isEmpty }
        return casks.filter { cask in
            let check = Shell.run(Shell.brewPath, ["info", "--cask", cask], showCommand: false, stream: false, silent: true)
            return check.exitCode != 0
        }
    }

    /// Disable broken casks by writing to ~/.config/homebrew/ignored-casks.rb
    static func ignoreBrokenCasks(_ casks: [String]) {
        let configDir = FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent(".config/homebrew")
        try? FileManager.default.createDirectory(at: configDir, withIntermediateDirectories: true)

        let ignoredFile = configDir.appendingPathComponent("ignored-casks.rb")
        let lines = casks.map { "cask '\($0)' do\n  disable!\nend" }.joined(separator: "\n")
        let data = (lines + "\n").data(using: .utf8)

        if FileManager.default.fileExists(atPath: ignoredFile.path) {
            if let handle = try? FileHandle(forWritingTo: ignoredFile) {
                handle.seekToEndOfFile()
                handle.write(data ?? Data())
                handle.closeFile()
            }
        } else {
            try? data?.write(to: ignoredFile)
        }
    }

    static func diskFreeBytes() -> Int64 {
        let attrs = try? FileManager.default.attributesOfFileSystem(forPath: "/")
        return (attrs?[.systemFreeSize] as? NSNumber)?.int64Value ?? 0
    }
}
