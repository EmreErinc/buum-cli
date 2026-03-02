import Foundation

struct Connectivity {
    static func isConnected() -> Bool {
        let result = Shell.run("/sbin/ping", ["-c", "1", "-W", "2000", "8.8.8.8"], showCommand: false, stream: false, silent: true)
        return result.exitCode == 0
    }
}
