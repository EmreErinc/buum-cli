import Foundation

struct Terminal {
    // ANSI color codes
    static let reset   = "\u{001B}[0m"
    static let bold    = "\u{001B}[1m"
    static let dim     = "\u{001B}[2m"
    static let red     = "\u{001B}[31m"
    static let green   = "\u{001B}[32m"
    static let yellow  = "\u{001B}[33m"
    static let blue    = "\u{001B}[34m"
    static let magenta = "\u{001B}[35m"
    static let cyan    = "\u{001B}[36m"
    static let white   = "\u{001B}[37m"

    static let isTerminal = isatty(STDOUT_FILENO) != 0

    static func colored(_ text: String, _ color: String) -> String {
        isTerminal ? "\(color)\(text)\(reset)" : text
    }

    static func header(_ text: String) {
        print(colored("\n==> \(text)", bold + blue))
    }

    static func status(_ text: String) {
        print(colored("  → \(text)", cyan))
    }

    static func success(_ text: String) {
        print(colored("  ✅ \(text)", green))
    }

    static func warning(_ text: String) {
        print(colored("  ⚠️  \(text)", yellow))
    }

    static func error(_ text: String) {
        print(colored("  ❌ \(text)", red))
    }

    static func info(_ text: String) {
        print(colored("  ℹ️  \(text)", dim))
    }

    static func suggestion(_ text: String) {
        print(colored("  💡 \(text)", cyan))
    }

    static func command(_ text: String) {
        print(colored("  $ \(text)", dim))
    }

    static func line(_ text: String, isError: Bool = false) {
        if isError {
            print(colored(text, red))
        } else {
            print(text)
        }
    }

    static func separator() {
        print(colored(String(repeating: "─", count: 60), dim))
    }

    static func table(headers: [String], rows: [[String]]) {
        guard !rows.isEmpty else { return }

        // Calculate column widths
        var widths = headers.map { $0.count }
        for row in rows {
            for (i, cell) in row.enumerated() where i < widths.count {
                widths[i] = max(widths[i], cell.count)
            }
        }

        // Print header
        let headerLine = headers.enumerated().map { i, h in
            h.padding(toLength: widths[i], withPad: " ", startingAt: 0)
        }.joined(separator: "  ")
        print(colored("  \(headerLine)", bold))
        print(colored("  \(widths.map { String(repeating: "─", count: $0) }.joined(separator: "  "))", dim))

        // Print rows
        for row in rows {
            let line = row.enumerated().map { i, cell in
                cell.padding(toLength: i < widths.count ? widths[i] : cell.count, withPad: " ", startingAt: 0)
            }.joined(separator: "  ")
            print("  \(line)")
        }
    }

    static func spinner(_ message: String, work: () throws -> Void) rethrows {
        let frames = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"]

        // Use a class wrapper so the closure captures a reference, not a value
        final class SpinnerState: @unchecked Sendable {
            var running = true
            var frameIndex = 0
        }
        let state = SpinnerState()

        let spinnerQueue = DispatchQueue(label: "spinner")
        spinnerQueue.async {
            while state.running {
                if isTerminal {
                    print("\r\(colored("  \(frames[state.frameIndex]) \(message)", cyan))", terminator: "")
                    fflush(stdout)
                }
                state.frameIndex = (state.frameIndex + 1) % frames.count
                Thread.sleep(forTimeInterval: 0.08)
            }
        }

        defer {
            state.running = false
            spinnerQueue.sync {} // wait for spinner to stop
            if isTerminal {
                print("\r\u{001B}[2K", terminator: "") // clear spinner line
            }
        }

        try work()
    }
}
