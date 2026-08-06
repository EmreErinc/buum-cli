#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-}"
REPO="${GITHUB_REPOSITORY:-EmreErinc/buum-cli}"

usage() {
  cat <<'EOF'
Usage: ./scripts/release.sh v0.1.5

Release flow:
  1. push main
  2. create + push tag (separate push — not --follow-tags)
  3. dispatch Release workflow explicitly (reliable fallback)

GoReleaser then builds binaries, publishes GitHub Release, updates Homebrew tap.
EOF
}

if [[ -z "$VERSION" ]]; then
  usage
  exit 1
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
  echo "error: version must look like v0.1.5 or v0.1.5-rc1" >&2
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

if git ls-remote --exit-code --tags origin "${VERSION}" >/dev/null 2>&1; then
  echo "error: tag ${VERSION} already exists on origin" >&2
  echo "hint: use a new version, or delete the remote tag:" >&2
  echo "  git push origin :refs/tags/${VERSION}" >&2
  exit 1
fi

echo "→ pushing main"
git push origin main

# Tag push and workflow file must not land in the same atomic push.
echo "→ waiting for main to settle on GitHub"
sleep 3

if git show-ref --verify --quiet "refs/tags/${VERSION}"; then
  echo "→ removing stale local tag ${VERSION}"
  git tag -d "${VERSION}"
fi

echo "→ creating tag ${VERSION}"
git tag "${VERSION}"

echo "→ pushing tag ${VERSION} (never use --follow-tags here)"
git push origin "${VERSION}"

echo "→ dispatching Release workflow"
if command -v gh >/dev/null 2>&1; then
  gh workflow run "Release" --repo "${REPO}" --ref "${VERSION}"
else
  echo "warning: gh CLI not found; rely on tag push trigger or run manually:" >&2
  echo "  gh workflow run Release --repo ${REPO} --ref ${VERSION}" >&2
fi

cat <<EOF

Release started for ${VERSION}.

Actions:  https://github.com/${REPO}/actions
Release:  https://github.com/${REPO}/releases/tag/${VERSION}

After the workflow finishes:
  brew update && brew upgrade buum
EOF
