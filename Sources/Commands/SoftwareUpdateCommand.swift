import ArgumentParser
import Foundation

struct SoftwareUpdateCommand: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "software-update",
        abstract: "Run macOS Software Update"
    )

    func run() throws {
        Terminal.header("Running macOS Software Update")
        Logger.log("--- softwareupdate started ---")

        Terminal.info("This may require admin privileges")

        let result = Shell.runWithAuth(
            "/usr/sbin/softwareupdate --install --all",
            label: "softwareupdate --install --all"
        )

        Logger.log("--- softwareupdate finished ---")

        Terminal.separator()
        if result.exitCode == 0 {
            Terminal.success("macOS software update complete!")
        } else {
            Terminal.error("Software update finished with errors")
            throw ExitCode.failure
        }
    }
}
