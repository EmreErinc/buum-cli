import ArgumentParser
import Foundation

struct RunCommand: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "run",
        abstract: "Run full update flow (brew update → upgrade → mas → cleanup)"
    )

    @Flag(name: .long, help: "Preview changes without installing")
    var dryRun = false

    @Flag(name: .long, help: "Skip Mac App Store updates")
    var noMas = false

    @Flag(name: .long, help: "Skip cleanup after upgrade")
    var noCleanup = false

    @Flag(name: .long, help: "Disable greedy cask upgrades")
    var noGreedy = false

    func run() throws {
        var config = Config.load()

        // CLI flags override config
        if dryRun { config.dryRun = true }
        if noMas { config.runMas = false }
        if noCleanup { config.runCleanup = false }
        if noGreedy { config.greedyUpgrade = false }

        print(Terminal.colored("🍺 Buum CLI — Running updates", Terminal.bold + Terminal.green))
        Terminal.separator()

        Logger.log("--- buum-cli run started ---")

        // Connectivity check
        Terminal.header("Checking connectivity")
        if !Connectivity.isConnected() {
            Terminal.warning("No internet connection. Aborting.")
            Logger.log("No internet connection — aborting")
            throw ExitCode.failure
        }
        Terminal.success("Connected")

        var failed = false
        var allOutput = ""

        // Pre-update script
        let preScript = config.preScript.trimmingCharacters(in: .whitespacesAndNewlines)
        if !preScript.isEmpty {
            Terminal.header("Running pre-update script")
            let result = Shell.bash(preScript)
            allOutput += result.stdout + result.stderr
            if result.exitCode != 0 { failed = true }
        }

        // Install Homebrew if missing
        if !Shell.brewExists {
            Terminal.header("Installing Homebrew")
            let result = Shell.bash(#"/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)""#)
            allOutput += result.stdout + result.stderr
            if result.exitCode != 0 {
                Terminal.error("Failed to install Homebrew")
                failed = true
            }
        }

        // Install mas if missing
        if config.runMas && !Shell.masExists {
            Terminal.header("Installing mas")
            let result = Shell.run(Shell.brewPath, ["install", "mas"])
            allOutput += result.stdout + result.stderr
            if result.exitCode != 0 { failed = true }
        }

        // brew update
        Terminal.header("Updating Homebrew")
        let updateResult = Shell.run(Shell.brewPath, ["update"])
        allOutput += updateResult.stdout + updateResult.stderr
        if updateResult.exitCode != 0 { failed = true }

        // Backup Brewfile
        if config.backupBeforeUpgrade {
            Terminal.header("Backing up package list")
            let backupPath = FileManager.default.homeDirectoryForCurrentUser
                .appendingPathComponent(".config/homebrew/Brewfile.bak").path
            Shell.run(Shell.brewPath, ["bundle", "dump", "--force", "--file=\(backupPath)"])
        }

        // brew upgrade
        Terminal.header("Upgrading packages")
        var upgradeArgs = ["upgrade"]
        if config.greedyUpgrade { upgradeArgs.append("--greedy") }
        if config.dryRun { upgradeArgs.append("--dry-run") }
        let upgradeResult = Shell.run(Shell.brewPath, upgradeArgs)
        allOutput += upgradeResult.stdout + upgradeResult.stderr
        if upgradeResult.exitCode != 0 { failed = true }

        // Handle skipped packages
        let skipped = BrewHelpers.skippedPackages(from: upgradeResult.stdout + upgradeResult.stderr)
        if !skipped.isEmpty && !config.dryRun {
            Terminal.header("Reinstalling \(skipped.count) skipped package(s)")
            Terminal.info("Skipped: \(skipped.joined(separator: ", "))")
            let result = Shell.run(Shell.brewPath, ["reinstall"] + skipped)
            allOutput += result.stdout + result.stderr
        }

        // Handle broken cask upgrades
        let brokenUpgrades = BrewHelpers.brokenCaskUpgrades(from: upgradeResult.stdout + upgradeResult.stderr)
        if !brokenUpgrades.isEmpty && !config.dryRun {
            Terminal.header("Reinstalling \(brokenUpgrades.count) broken cask(s)")
            for cask in brokenUpgrades {
                Terminal.info("Reinstalling: \(cask)")
                let result = Shell.run(Shell.brewPath, ["reinstall", "--cask", "--force", cask])
                allOutput += result.stdout + result.stderr
            }
        }

        // Mac App Store
        if config.runMas {
            Terminal.header("Checking App Store updates")
            let masOutdated = Shell.run(Shell.masPath, ["outdated"])
            allOutput += masOutdated.stdout + masOutdated.stderr

            let outdatedApps = masOutdated.stdout.trimmingCharacters(in: .whitespacesAndNewlines)
            if outdatedApps.isEmpty {
                Terminal.success("No App Store updates available")
            } else {
                Terminal.info("Outdated: \(outdatedApps.components(separatedBy: "\n").count) app(s)")
                Terminal.header("Upgrading App Store apps")
                let masUpgrade = Shell.runInteractive(Shell.masPath, ["upgrade"], timeout: 600)
                if masUpgrade.timedOut {
                    Terminal.warning("mas upgrade timed out — App Store may be unresponsive")
                    Terminal.info("Try running manually: mas upgrade")
                    failed = true
                } else if masUpgrade.exitCode != 0 {
                    Terminal.warning("mas upgrade failed — make sure you're signed in to the App Store")
                    failed = true
                }
            }
        }

        // Cleanup
        if config.runCleanup {
            Terminal.header("Cleaning up")
            let beforeCleanup = BrewHelpers.diskFreeBytes()
            let cleanupResult = Shell.run(Shell.brewPath, ["cleanup", "--prune=all"])
            allOutput += cleanupResult.stdout + cleanupResult.stderr

            // Handle skipped in cleanup
            let cleanupSkipped = BrewHelpers.skippedPackages(from: cleanupResult.stdout + cleanupResult.stderr)
            if !cleanupSkipped.isEmpty && !config.dryRun {
                Terminal.info("Reinstalling \(cleanupSkipped.count) package(s) skipped in cleanup…")
                Shell.run(Shell.brewPath, ["reinstall"] + cleanupSkipped)
                Shell.run(Shell.brewPath, ["cleanup", "--prune=all"])
            }

            let freed = BrewHelpers.diskFreeBytes() - beforeCleanup
            if freed > 0 {
                Terminal.success("Freed \(freed / 1_000_000) MB")
            }
        }

        // Broken cask check
        if config.runBrokenCaskCheck {
            Terminal.header("Checking for broken casks")
            let brokenCasks = BrewHelpers.findBrokenCasks()
            if !brokenCasks.isEmpty {
                Terminal.warning("Disabling \(brokenCasks.count) broken cask(s): \(brokenCasks.joined(separator: ", "))")
                BrewHelpers.ignoreBrokenCasks(brokenCasks)
                Logger.log("Disabled broken casks: \(brokenCasks.joined(separator: ", "))")
            } else {
                Terminal.success("No broken casks")
            }
        }

        // Post-update script
        let postScript = config.postScript.trimmingCharacters(in: .whitespacesAndNewlines)
        if !postScript.isEmpty {
            Terminal.header("Running post-update script")
            let result = Shell.bash(postScript)
            allOutput += result.stdout + result.stderr
            if result.exitCode != 0 { failed = true }
        }

        Logger.log("--- buum-cli run finished (success: \(!failed)) ---")

        // Summary
        Terminal.separator()
        if failed {
            Terminal.error("Updates finished with errors")
            Suggestions.generate(from: allOutput)
            throw ExitCode.failure
        } else {
            Terminal.success("All updates completed & cache cleaned!")
        }
    }
}
