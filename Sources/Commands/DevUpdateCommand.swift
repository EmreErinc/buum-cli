import ArgumentParser
import Foundation

struct DevUpdateCommand: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "dev-update",
        abstract: "Update npm globals and pip3"
    )

    func run() throws {
        Logger.log("--- dev tools update started ---")
        var failed = false

        // npm
        Terminal.header("Updating npm globals")
        let npmCheck = Shell.bash("which npm", showCommand: false, stream: false, silent: true)
        if npmCheck.exitCode == 0 {
            let result = Shell.bash("npm update -g")
            if result.exitCode != 0 { failed = true }
        } else {
            Terminal.info("npm not found, skipping")
        }

        // pip3
        Terminal.header("Updating pip3")
        let pipCheck = Shell.bash("which pip3", showCommand: false, stream: false, silent: true)
        if pipCheck.exitCode == 0 {
            let result = Shell.bash("pip3 install --upgrade pip")
            if result.exitCode != 0 { failed = true }
        } else {
            Terminal.info("pip3 not found, skipping")
        }

        Logger.log("--- dev tools update finished ---")

        Terminal.separator()
        if failed {
            Terminal.error("Dev tools update finished with errors")
            throw ExitCode.failure
        } else {
            Terminal.success("Dev tools updated!")
        }
    }
}
