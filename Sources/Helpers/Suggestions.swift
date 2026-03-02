import Foundation

struct Suggestions {
    static func generate(from output: String) {
        var suggestions: [String] = []
        var seen = Set<String>()
        let lines = output.components(separatedBy: "\n")

        for line in lines {
            // Broken app source (renamed cask)
            if line.contains("It seems the App source") || line.contains("is not there") {
                let cask = line.hasPrefix("Error: ") ? String(line.dropFirst(7)).components(separatedBy: ":").first ?? "" : ""
                let cmd = "brew reinstall --cask --force \(cask.trimmingCharacters(in: .whitespaces))"
                if !cask.isEmpty && seen.insert(cmd).inserted { suggestions.append(cmd) }
            }

            // Permission denied / sudo
            if line.contains("a terminal is required") || line.contains("Permission denied") {
                let cmd = "sudo brew upgrade"
                if seen.insert(cmd).inserted {
                    suggestions.append("# Run manually with sudo:")
                    suggestions.append(cmd)
                }
            }

            // Version not installed (skipped)
            if line.contains("Warning: Skipping") && line.contains("not installed") {
                let parts = line.components(separatedBy: "Skipping ")
                if let pkg = parts.last?.components(separatedBy: ":").first?.trimmingCharacters(in: .whitespaces) {
                    let cmd = "brew reinstall \(pkg)"
                    if seen.insert(cmd).inserted { suggestions.append(cmd) }
                }
            }

            // Missing dependencies
            if line.contains("missing dependencies") {
                let cmd = "brew missing && brew reinstall $(brew missing | awk -F: '{print $1}')"
                if seen.insert(cmd).inserted { suggestions.append(cmd) }
            }

            // Outdated Xcode CLT
            if line.contains("CLT") || line.contains("command line tools") {
                let cmd = "xcode-select --install"
                if seen.insert(cmd).inserted { suggestions.append(cmd) }
            }

            // Generic brew doctor suggestion
            if line.contains("Error:") && seen.insert("brew doctor").inserted {
                suggestions.append("brew doctor")
            }
        }

        if !suggestions.isEmpty {
            print()
            Terminal.header("Suggested fixes")
            for s in suggestions {
                Terminal.suggestion(s)
            }
        }
    }
}
