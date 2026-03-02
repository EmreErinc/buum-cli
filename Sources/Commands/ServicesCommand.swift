import ArgumentParser
import Foundation

struct ServicesCommand: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "services",
        abstract: "Manage Homebrew services",
        subcommands: [ListServices.self, StartService.self, StopService.self, RestartService.self],
        defaultSubcommand: ListServices.self
    )
}

struct ListServices: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "list",
        abstract: "List all Homebrew services"
    )

    func run() throws {
        guard Shell.brewExists else {
            Terminal.error("Homebrew is not installed")
            throw ExitCode.failure
        }

        Terminal.header("Brew Services")

        let services = BrewHelpers.fetchServices()

        if services.isEmpty {
            Terminal.info("No services found")
            return
        }

        Terminal.table(
            headers: ["Status", "Service"],
            rows: services.map { ["\($0.statusIcon) \($0.status)", $0.name] }
        )
    }
}

struct StartService: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "start",
        abstract: "Start a Homebrew service"
    )

    @Argument(help: "Service name to start")
    var service: String

    func run() throws {
        guard Shell.brewExists else {
            Terminal.error("Homebrew is not installed")
            throw ExitCode.failure
        }

        Terminal.header("Starting \(service)")
        Logger.log("--- brew services start \(service) ---")
        let result = Shell.run(Shell.brewPath, ["services", "start", service])
        Logger.log("--- done ---")

        if result.exitCode == 0 {
            Terminal.success("\(service) started")
        } else {
            Terminal.error("Failed to start \(service)")
            throw ExitCode.failure
        }
    }
}

struct StopService: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "stop",
        abstract: "Stop a Homebrew service"
    )

    @Argument(help: "Service name to stop")
    var service: String

    func run() throws {
        guard Shell.brewExists else {
            Terminal.error("Homebrew is not installed")
            throw ExitCode.failure
        }

        Terminal.header("Stopping \(service)")
        Logger.log("--- brew services stop \(service) ---")
        let result = Shell.run(Shell.brewPath, ["services", "stop", service])
        Logger.log("--- done ---")

        if result.exitCode == 0 {
            Terminal.success("\(service) stopped")
        } else {
            Terminal.error("Failed to stop \(service)")
            throw ExitCode.failure
        }
    }
}

struct RestartService: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "restart",
        abstract: "Restart a Homebrew service"
    )

    @Argument(help: "Service name to restart")
    var service: String

    func run() throws {
        guard Shell.brewExists else {
            Terminal.error("Homebrew is not installed")
            throw ExitCode.failure
        }

        Terminal.header("Restarting \(service)")
        Logger.log("--- brew services restart \(service) ---")
        let result = Shell.run(Shell.brewPath, ["services", "restart", service])
        Logger.log("--- done ---")

        if result.exitCode == 0 {
            Terminal.success("\(service) restarted")
        } else {
            Terminal.error("Failed to restart \(service)")
            throw ExitCode.failure
        }
    }
}
