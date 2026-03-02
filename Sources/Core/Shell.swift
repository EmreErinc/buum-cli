import Foundation

struct Shell {
    static let brewPath: String = {
        // Check common Homebrew paths
        for path in ["/opt/homebrew/bin/brew", "/usr/local/bin/brew"] {
            if FileManager.default.fileExists(atPath: path) { return path }
        }
        return "/opt/homebrew/bin/brew"
    }()

    static let masPath: String = {
        for path in ["/opt/homebrew/bin/mas", "/usr/local/bin/mas"] {
            if FileManager.default.fileExists(atPath: path) { return path }
        }
        return "/opt/homebrew/bin/mas"
    }()

    static let defaultEnv: [String: String] = {
        var env = ProcessInfo.processInfo.environment
        env["PATH"] = "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
        return env
    }()

    static var brewExists: Bool {
        FileManager.default.fileExists(atPath: brewPath)
    }

    static var masExists: Bool {
        FileManager.default.fileExists(atPath: masPath)
    }

    /// Run a command with real-time streaming output. Returns the exit code.
    @discardableResult
    static func run(
        _ path: String,
        _ args: [String] = [],
        env: [String: String]? = nil,
        showCommand: Bool = true,
        stream: Bool = true,
        silent: Bool = false
    ) -> (exitCode: Int32, stdout: String, stderr: String) {
        let environment = env ?? defaultEnv
        let cmd = ([path.components(separatedBy: "/").last ?? path] + args).joined(separator: " ")

        if showCommand && !silent {
            Terminal.command(cmd)
        }

        Logger.log("$ \(([path] + args).joined(separator: " "))")

        let task = Process()
        task.launchPath = path
        task.arguments = args
        task.environment = environment

        let outPipe = Pipe()
        let errPipe = Pipe()
        task.standardOutput = outPipe
        task.standardError = errPipe

        var stdoutData = Data()
        var stderrData = Data()

        if stream && !silent {
            outPipe.fileHandleForReading.readabilityHandler = { handle in
                let data = handle.availableData
                guard !data.isEmpty else { return }
                stdoutData.append(data)
                if let text = String(data: data, encoding: .utf8) {
                    Logger.log("stdout: \(text.trimmingCharacters(in: .whitespacesAndNewlines))")
                    for line in text.components(separatedBy: "\n") where !line.isEmpty {
                        Terminal.line(line)
                    }
                }
            }
            errPipe.fileHandleForReading.readabilityHandler = { handle in
                let data = handle.availableData
                guard !data.isEmpty else { return }
                stderrData.append(data)
                if let text = String(data: data, encoding: .utf8) {
                    Logger.log("stderr: \(text.trimmingCharacters(in: .whitespacesAndNewlines))")
                    for line in text.components(separatedBy: "\n") where !line.isEmpty {
                        Terminal.line(line, isError: true)
                    }
                }
            }
        }

        do {
            try task.run()
        } catch {
            Logger.log("Failed to launch: \(error)")
            if !silent { Terminal.error("Failed to run: \(cmd)") }
            return (-1, "", "")
        }
        task.waitUntilExit()

        outPipe.fileHandleForReading.readabilityHandler = nil
        errPipe.fileHandleForReading.readabilityHandler = nil

        if !stream || silent {
            stdoutData = outPipe.fileHandleForReading.readDataToEndOfFile()
            stderrData = errPipe.fileHandleForReading.readDataToEndOfFile()
        }

        let exitCode = task.terminationStatus
        Logger.log("exit: \(exitCode)")

        let stdout = String(data: stdoutData, encoding: .utf8) ?? ""
        let stderr = String(data: stderrData, encoding: .utf8) ?? ""

        return (exitCode, stdout, stderr)
    }

    /// Run a shell command string via /bin/bash -c
    @discardableResult
    static func bash(
        _ command: String,
        env: [String: String]? = nil,
        showCommand: Bool = true,
        stream: Bool = true,
        silent: Bool = false
    ) -> (exitCode: Int32, stdout: String, stderr: String) {
        return run("/bin/bash", ["-c", command], env: env, showCommand: showCommand, stream: stream, silent: silent)
    }

    /// Run a command interactively with a timeout.
    /// Inherits stdin/stdout/stderr so password prompts and progress bars work natively.
    /// Terminates the process if it exceeds the timeout.
    @discardableResult
    static func runInteractive(
        _ path: String,
        _ args: [String] = [],
        env: [String: String]? = nil,
        timeout: TimeInterval = 300,
        showCommand: Bool = true
    ) -> (exitCode: Int32, timedOut: Bool) {
        let environment = env ?? defaultEnv
        let cmd = ([path.components(separatedBy: "/").last ?? path] + args).joined(separator: " ")

        if showCommand {
            Terminal.command(cmd)
        }

        Logger.log("$ \(([path] + args).joined(separator: " ")) [interactive, timeout: \(Int(timeout))s]")

        let task = Process()
        task.launchPath = path
        task.arguments = args
        task.environment = environment
        task.standardInput = FileHandle.standardInput
        task.standardOutput = FileHandle.standardOutput
        task.standardError = FileHandle.standardError

        do {
            try task.run()
        } catch {
            Logger.log("Failed to launch: \(error)")
            Terminal.error("Failed to run: \(cmd)")
            return (-1, false)
        }

        var timedOut = false
        let deadline = DispatchTime.now() + timeout
        let group = DispatchGroup()
        group.enter()
        DispatchQueue.global().async {
            task.waitUntilExit()
            group.leave()
        }

        if group.wait(timeout: deadline) == .timedOut {
            timedOut = true
            Logger.log("TIMEOUT: \(cmd) exceeded \(Int(timeout))s — terminating")
            task.terminate()
            _ = group.wait(timeout: .now() + 5)
        }

        let exitCode = task.terminationStatus
        Logger.log("exit: \(exitCode) (timedOut: \(timedOut))")

        return (exitCode, timedOut)
    }

    /// Run a command with sudo, inheriting the terminal for secure password input
    @discardableResult
    static func runWithAuth(
        _ command: String,
        label: String? = nil,
        env: [String: String]? = nil
    ) -> (exitCode: Int32, stdout: String, stderr: String) {
        let environment = env ?? defaultEnv
        let displayLabel = label ?? command

        Terminal.command("\(displayLabel) (admin)")
        Logger.log("$ \(command) [via sudo]")

        let path = environment["PATH"] ?? "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
        let fullCommand = "export PATH=\"\(path)\"; \(command)"

        let task = Process()
        task.launchPath = "/usr/bin/sudo"
        task.arguments = ["/bin/bash", "-c", fullCommand]
        task.environment = environment

        // Inherit the terminal directly so sudo can securely prompt for password
        task.standardInput = FileHandle.standardInput
        task.standardOutput = FileHandle.standardOutput
        task.standardError = FileHandle.standardError

        do {
            try task.run()
        } catch {
            Logger.log("Failed to launch with auth: \(error)")
            Terminal.error("Failed to run with auth: \(displayLabel)")
            return (-1, "", "")
        }
        task.waitUntilExit()

        let exitCode = task.terminationStatus
        Logger.log("exit: \(exitCode)")

        return (exitCode, "", "")
    }
}
