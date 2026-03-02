import ArgumentParser
import Foundation

struct OutdatedCommand: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "outdated",
        abstract: "List outdated Homebrew packages"
    )

    func run() throws {
        guard Shell.brewExists else {
            Terminal.error("Homebrew is not installed")
            throw ExitCode.failure
        }

        Terminal.header("Checking outdated packages")

        let packages = BrewHelpers.fetchOutdated()

        Terminal.separator()
        if packages.isEmpty {
            Terminal.success("All packages are up to date!")
        } else {
            print(Terminal.colored("  📦 \(packages.count) outdated package(s):\n", Terminal.bold))
            Terminal.table(
                headers: ["Package", "Current", "→", "Latest"],
                rows: packages.map { [$0.name, $0.current, "→", $0.latest] }
            )
            print()
            Terminal.info("Run `buum-cli run` to upgrade all packages")
        }
    }
}
