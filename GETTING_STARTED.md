# buum — Getting Started

Setup commands, in order.

## 1. Prerequisite — Go toolchain

```bash
brew install go
go version   # expect go1.22 or newer
```

## 2. Build

From repo root (`/Users/emreerinc/projects/buum-cli`):

```bash
go build -o ./buum ./cmd/buum
./buum version   # 0.1.0-dev
```

## 3. Install on PATH (pick one)

```bash
# System-wide (prompts for sudo password)
sudo mv ./buum /usr/local/bin/buum

# Or via Go (lands in $(go env GOPATH)/bin)
go install ./cmd/buum
# add to ~/.zshrc if GOPATH/bin not on PATH:
export PATH="$PATH:$(go env GOPATH)/bin"
```

After install, run `buum` from anywhere (drop the `./`).

## 4. Verify

```bash
buum list        # detected + known-but-missing managers
buum --dry-run   # preview steps, runs nothing
```

## 5. Usage

```bash
buum                    # update ALL detected managers immediately (NO confirm)
buum -i                 # interactive checklist, pick managers
buum --dry-run          # preview only
buum --only brew,npm    # subset
buum --except pip       # all detected except these
buum --json             # machine-readable event stream (for scripts / future GUI)
buum config path        # print config file path
buum config edit        # edit config in $EDITOR
buum version
```

**Warning:** bare `buum` runs every detected manager with no `y/N`. The
`softwareupdate` step is `sudo softwareupdate -ia` — it prompts for your password
inline and waits. Use `--dry-run` / `--only` / `-i` to stay in control.

## 6. Config (optional)

Override built-ins at `~/.config/buum/config.yaml`. Example:

```bash
mkdir -p ~/.config/buum
cp config.example.yaml ~/.config/buum/config.yaml
buum config edit
```

See `config.example.yaml` for the format (disable, override steps, add custom
managers, set run order).

## 7. Run tests (for development)

```bash
go test ./...
go vet ./...
gofmt -l internal/ pkg/ cmd/   # empty = clean
```
