# 🍺 Buum CLI

> Terminal version of [Buum](https://github.com/emreerinc/buum-app) — keep your Homebrew packages and Mac App Store apps up to date from the command line.

![macOS](https://img.shields.io/badge/macOS-13%2B-blue)
![Swift](https://img.shields.io/badge/Swift-5.9-orange)
![License](https://img.shields.io/badge/license-GPL--3.0-green)

## What it does

Buum CLI is the terminal companion to the Buum menu bar app. It runs the same update flow:

```bash
brew update → brew upgrade → mas outdated → mas upgrade → brew cleanup
```

…with colored output, error suggestions, and configurable options.

## Installation

### Via Buum.app (bundled)

When you install Buum via Homebrew, `buum-cli` is automatically available:

```bash
brew tap emreerinc/buum
brew install --cask buum
buum-cli --help
```

### Build from source

```bash
git clone https://github.com/emreerinc/buum-cli.git
cd buum-cli
swift build -c release
# Binary is at .build/release/buum-cli
```

## Usage

```bash
# Run full update flow (default command)
buum-cli

# Or explicitly
buum-cli run
buum-cli run --dry-run
buum-cli run --no-mas --no-cleanup

# Check outdated packages
buum-cli outdated

# Run brew doctor
buum-cli doctor

# Find missing dependencies
buum-cli missing
buum-cli missing --fix

# Manage brew services
buum-cli services list
buum-cli services start postgresql
buum-cli services stop postgresql
buum-cli services restart nginx

# macOS software update
buum-cli software-update

# Update npm & pip globals
buum-cli dev-update

# Manage configuration
buum-cli config show
buum-cli config set dryRun true
buum-cli config get greedyUpgrade
buum-cli config reset
buum-cli config path
```

## Configuration

Config is stored at `~/.config/buum-cli/config.json`. Logs are at `~/.local/share/buum-cli/buum.log`.

| Key | Default | Description |
|-----|---------|-------------|
| `runMas` | `true` | Include Mac App Store apps in updates |
| `runCleanup` | `true` | Clean cache after upgrades |
| `runBrokenCaskCheck` | `true` | Detect and disable broken casks |
| `greedyUpgrade` | `true` | Update auto-updating casks too |
| `dryRun` | `false` | Preview changes without installing |
| `backupBeforeUpgrade` | `false` | Backup Brewfile before upgrading |
| `preScript` | `""` | Shell script to run before updates |
| `postScript` | `""` | Shell script to run after updates |

CLI flags (e.g., `--dry-run`, `--no-mas`) override config values for that run.

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).
