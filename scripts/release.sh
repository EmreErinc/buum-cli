#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-}"

usage() {
  cat <<'EOF'
Usage: ./scripts/release.sh v0.1.4

Creates a git tag and pushes it to origin.
GitHub Actions picks up the tag push and runs GoReleaser:
  - builds binaries for macOS/Linux
  - publishes a GitHub Release
  - updates the Homebrew tap (EmreErinc/tap)

Requirements:
  - clean working tree on main
  - main pushed to origin before the tag
  - TAP_GITHUB_TOKEN secret configured in the repo
EOF
}

if [[ -z "$VERSION" ]]; then
  usage
  exit 1
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
  echo "error: version must look like v0.1.4 or v0.1.4-rc1" >&2
  exit 1
fi

if [[ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]]; then
  echo "error: switch to main before releasing" >&2
  exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
  echo "error: working tree is not clean; commit or stash changes first" >&2
  exit 1
fi

echo "→ pushing main"
git push origin main

if git show-ref --verify --quiet "refs/tags/${VERSION}"; then
  if git ls-remote --exit-code --tags origin "${VERSION}" >/dev/null 2>&1; then
    echo "error: tag ${VERSION} already exists on origin" >&2
    echo "hint: bump the version or delete the remote tag first:" >&2
    echo "  git push origin :refs/tags/${VERSION}" >&2
    exit 1
  fi
  echo "→ tag ${VERSION} exists locally, pushing to origin"
else
  echo "→ creating tag ${VERSION}"
  git tag "${VERSION}"
fi

echo "→ pushing tag ${VERSION}"
git push origin "${VERSION}"

cat <<EOF

Release pipeline triggered for ${VERSION}.

Actions:  https://github.com/EmreErinc/buum-cli/actions
Release:  https://github.com/EmreErinc/buum-cli/releases/tag/${VERSION}

After the workflow finishes, users can upgrade with:
  brew update && brew upgrade buum
EOF
