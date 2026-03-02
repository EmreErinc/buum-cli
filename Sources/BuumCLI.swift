// Buum CLI - Terminal version of Buum
// Copyright (C) 2026 Emre Erinç
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

import ArgumentParser

@main
struct BuumCLI: ParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "buum-cli",
        abstract: "Keep your Homebrew packages and Mac App Store apps up to date — from the terminal.",
        version: "1.0.0",
        subcommands: [
            RunCommand.self,
            DoctorCommand.self,
            MissingCommand.self,
            OutdatedCommand.self,
            ServicesCommand.self,
            SoftwareUpdateCommand.self,
            DevUpdateCommand.self,
            ConfigCommand.self,
        ],
        defaultSubcommand: RunCommand.self
    )
}
