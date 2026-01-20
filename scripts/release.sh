#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF' >&2
usage: scripts/release.sh vX.Y.Z
EOF
  exit 1
}

if [[ $# -ne 1 ]]; then
  usage
fi

version="$1"
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "version must look like vX.Y.Z (got $version)" >&2
  exit 1
fi

# Preflight checks
if [[ -n "$(git status --porcelain)" ]]; then
  echo "working copy is not clean; commit or stash changes first" >&2
  git status --porcelain >&2
  exit 1
fi

if rg -q "^##[[:space:]]+${version}[[:space:]]*\\(" CHANGELOG.md; then
  echo "$version already exists in CHANGELOG.md" >&2
  exit 1
fi

if ! rg -q "^##[[:space:]]+Unreleased[[:space:]]*$" CHANGELOG.md; then
  echo "CHANGELOG.md is missing a '## Unreleased' section" >&2
  exit 1
fi

# Extract Unreleased section
unreleased_tmp="$(mktemp -t linear-cli-unreleased.XXXXXX)"
notes_tmp="$(mktemp -t linear-cli-notes.XXXXXX)"
out_tmp="$(mktemp -t linear-cli-changelog.XXXXXX)"
cleanup() {
  rm -f "$unreleased_tmp" "$notes_tmp" "$out_tmp"
}
trap cleanup EXIT

awk '
  $0 ~ /^##[[:space:]]+Unreleased[[:space:]]*$/ { flag=1; next }
  /^##[[:space:]]+/ { if (flag) exit }
  { if (flag) print }
' CHANGELOG.md > "$unreleased_tmp"

if ! rg -q "^[[:space:]]*[-*][[:space:]]+\\S" "$unreleased_tmp"; then
  echo "Unreleased section is empty; add notable changes first" >&2
  exit 1
fi

# Promote Unreleased section to a versioned section
date="$(date +%Y-%m-%d)"
awk -v version="$version" -v date="$date" '
  $0 ~ /^##[[:space:]]+Unreleased[[:space:]]*$/ {
    print "## " version " (" date ")"
    flag=1
    next
  }
  /^##[[:space:]]+/ { if (flag) flag=0 }
  { print }
  END { if (!NR) exit 2 }
' CHANGELOG.md > "$out_tmp"

mv "$out_tmp" CHANGELOG.md

# Extract release notes for gh release
awk -v version="$version" '
  $0 ~ "^##[[:space:]]+" version "([[:space:]]|$)" { flag=1; next }
  /^##[[:space:]]+/ { if (flag) exit }
  { if (flag) print }
' CHANGELOG.md > "$notes_tmp"

if ! rg -q "\\S" "$notes_tmp"; then
  echo "release notes are empty after changelog update" >&2
  exit 1
fi

if git rev-parse -q --verify "refs/tags/$version" >/dev/null; then
  echo "tag $version already exists" >&2
  exit 1
fi

# Commit and tag the release
git add CHANGELOG.md
git commit -m "release: $version"
git tag "$version"
git push origin main
git push origin "$version"

# Create or update the GitHub release notes
if gh release view "$version" >/dev/null 2>&1; then
  gh release edit "$version" --notes-file "$notes_tmp"
else
  gh release create "$version" --notes-file "$notes_tmp"
fi

# Start a new Unreleased section
awk '
  /^##[[:space:]]+/ && !inserted {
    print "## Unreleased"
    print ""
    inserted=1
  }
  { print }
  END {
    if (!inserted) {
      print "## Unreleased"
      print ""
    }
  }
' CHANGELOG.md > "$out_tmp"

mv "$out_tmp" CHANGELOG.md
git add CHANGELOG.md
git commit -m "chore: start next cycle"
git push origin main
