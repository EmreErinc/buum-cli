import ArgumentParser
import Foundation

struct ConfigCommand: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "config",
        abstract: "Manage buum-cli preferences",
        subcommands: [ShowConfig.self, SetConfig.self, GetConfig.self, ResetConfig.self, PathConfig.self],
        defaultSubcommand: ShowConfig.self
    )
}

struct ShowConfig: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "show",
        abstract: "Show all configuration values"
    )

    func run() throws {
        let config = Config.load()
        Terminal.header("Configuration")
        Terminal.info("Config file: \(Config.configPath.path)")
        print()

        Terminal.table(
            headers: ["Key", "Value", "Description"],
            rows: Config.keys.map { entry in
                [entry.key, config.get(key: entry.key) ?? "?", entry.description]
            }
        )
    }
}

struct SetConfig: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "set",
        abstract: "Set a configuration value"
    )

    @Argument(help: "Config key (e.g., runMas, dryRun)")
    var key: String

    @Argument(help: "Value to set (true/false for booleans, text for scripts)")
    var value: String

    func run() throws {
        var config = Config.load()
        if config.set(key: key, value: value) {
            config.save()
            Terminal.success("\(key) = \(config.get(key: key) ?? value)")
        } else {
            Terminal.error("Unknown config key: \(key)")
            Terminal.info("Valid keys: \(Config.keys.map { $0.key }.joined(separator: ", "))")
            throw ExitCode.failure
        }
    }
}

struct GetConfig: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "get",
        abstract: "Get a configuration value"
    )

    @Argument(help: "Config key")
    var key: String

    func run() throws {
        let config = Config.load()
        if let value = config.get(key: key) {
            print(value)
        } else {
            Terminal.error("Unknown config key: \(key)")
            throw ExitCode.failure
        }
    }
}

struct ResetConfig: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "reset",
        abstract: "Reset all configuration to defaults"
    )

    func run() throws {
        Config.reset()
        Terminal.success("Configuration reset to defaults")
    }
}

struct PathConfig: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "path",
        abstract: "Show config and log file paths"
    )

    func run() throws {
        print("  Config: \(Config.configPath.path)")
        print("  Log:    \(Logger.logPath.path)")
    }
}
