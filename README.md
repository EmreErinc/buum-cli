# buum

macOS updater. Detects installed package managers and runs their update
commands. Built-ins: brew, mas, npm, pip, pipx, gem, cargo, rustup,
softwareupdate, mise, uv, pnpm, gcloud, conda, composer, gh, deno, bun, yarn,
port, tlmgr, flutter, ghcup, opam, nix, tldr. Only installed ones run;
uninstalled managers are skipped by detection.

## Install

### Homebrew (recommended)

```bash
brew tap EmreErinc/tap
brew install buum
brew upgrade buum   # get the latest release
```

### From source

```bash
go build -o /usr/local/bin/buum ./cmd/buum
```

## Release (maintainers)

Push a new semver tag to trigger the release pipeline (build, GitHub Release,
Homebrew tap update):

```bash
./scripts/release.sh v0.1.4
```

Manual order if you prefer raw git:

```bash
git push origin main
git tag v0.1.4
git push origin v0.1.4
```

Each tag must be **new on GitHub** — re-pushing an existing tag does not
trigger the workflow. The repo needs a `TAP_GITHUB_TOKEN` secret with write
access to `EmreErinc/homebrew-tap`.

## Usage

```bash
buum                    # detect + update all detected managers immediately
buum -i                 # pick managers interactively
buum --dry-run          # show what would run
buum --only brew,npm    # subset
buum --except pip       # all except these
buum --json             # machine-readable event stream
buum list               # show detected + known-but-missing managers
buum config path        # print config file path
buum config edit        # edit config in $EDITOR
```

## Interactive mode

`buum -i` opens a checkbox menu to choose which managers to update. Use:

- `↑`/`↓` to move the cursor
- `space` to toggle the selected manager
- `a` to toggle all managers
- `/` to filter by name
- `enter` to run the selected managers
- `q` or `esc` to cancel

All managers start selected. After choosing, buum asks `Save this selection as
default? [y/N]`. Answering yes writes your enable/disable set to the config so
future runs honor it. Note: saving only affects the managers you were shown, so
combining `-i` with `--only`/`--except` limits which managers the save can
enable or disable.

The menu requires a real terminal (stdin and stdout must not be piped). In CI
or when input is redirected, buum falls back to a numbered prompt.

## Prompt handling

buum never auto-answers prompts and never times out waiting for input. If a
manager asks for a sudo password or a `y/N` confirmation, buum surfaces it and
waits. A fully detached run (e.g. cron with no stdin) will block if a manager
prompts — pre-authorize sudo or set non-interactive `steps` in config.

## Config

`~/.config/buum/config.yaml` overrides the built-ins. See `config.example.yaml`.
