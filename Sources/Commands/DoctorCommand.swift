import ArgumentParser
import Foundation

struct DoctorCommand: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "doctor",
        abstract: "Run brew doctor and check system health"
    )

    func run() throws {
        guard Shell.brewExists else {
            Terminal.error("Homebrew is not installed")
            throw ExitCode.failure
        }

        Terminal.header("Running brew doctor")
        Logger.log("--- brew doctor started ---")

        let result = Shell.run(Shell.brewPath, ["doctor"], stream: true)

        let healthy = result.exitCode == 0
        Logger.log("--- brew doctor finished (healthy: \(healthy)) ---")

        Terminal.separator()
        if healthy {
            Terminal.success("Your system is ready to brew!")
        } else {
            let issues = (result.stdout + result.stderr)
                .components(separatedBy: "\n")
                .filter { $0.hasPrefix("Warning:") || $0.hasPrefix("Error:") }
            Terminal.warning("\(issues.count) issue(s) found")
            throw ExitCode.failure
        }
    }
}
