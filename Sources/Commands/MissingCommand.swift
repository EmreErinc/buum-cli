import ArgumentParser
import Foundation

struct MissingCommand: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "missing",
        abstract: "Find and optionally reinstall missing dependencies"
    )

    @Flag(name: .long, help: "Automatically reinstall formulae with missing deps")
    var fix = false

    func run() throws {
        guard Shell.brewExists else {
            Terminal.error("Homebrew is not installed")
            throw ExitCode.failure
        }

        Terminal.header("Finding missing dependencies")
        Logger.log("--- brew missing started ---")

        let result = Shell.run(Shell.brewPath, ["missing"], stream: true)

        // Parse missing: "formula: dep1 dep2"
        let missing = result.stdout.components(separatedBy: "\n")
            .filter { $0.contains(":") && !$0.isEmpty }

        Logger.log("--- brew missing finished ---")

        Terminal.separator()
        if missing.isEmpty {
            Terminal.success("No missing dependencies found!")
        } else {
            Terminal.warning("\(missing.count) formula(e) have missing dependencies")

            if fix {
                let formulae = missing.compactMap { line -> String? in
                    line.components(separatedBy: ":").first?.trimmingCharacters(in: .whitespaces)
                }
                guard !formulae.isEmpty else { return }

                Terminal.header("Reinstalling \(formulae.count) formula(e)")
                let reinstallResult = Shell.run(Shell.brewPath, ["reinstall"] + formulae)
                if reinstallResult.exitCode == 0 {
                    Terminal.success("Reinstalled successfully")
                } else {
                    Terminal.error("Some reinstalls failed")
                    throw ExitCode.failure
                }
            } else {
                Terminal.info("Run with --fix to reinstall automatically")
            }
        }
    }
}
