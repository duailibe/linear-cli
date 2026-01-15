#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: scripts/update_homebrew_tap.sh vX.Y.Z" >&2
  exit 1
fi

tag="$1"
version="${tag#v}"
checksums="$(curl -sSL "https://github.com/duailibe/linear-cli/releases/download/${tag}/checksums.txt")"
darwin_arm64="$(printf '%s\n' "$checksums" | grep " linear_${version}_darwin_arm64.tar.gz$" | cut -d' ' -f1)"
darwin_amd64="$(printf '%s\n' "$checksums" | grep " linear_${version}_darwin_amd64.tar.gz$" | cut -d' ' -f1)"
linux_arm64="$(printf '%s\n' "$checksums" | grep " linear_${version}_linux_arm64.tar.gz$" | cut -d' ' -f1)"
linux_amd64="$(printf '%s\n' "$checksums" | grep " linear_${version}_linux_amd64.tar.gz$" | cut -d' ' -f1)"

cat > homebrew-tap/Formula/linear-cli.rb <<EOF
class LinearCli < Formula
  desc "A fast, no-nonsense CLI for Linear"
  homepage "https://github.com/duailibe/linear-cli"
  version "${version}"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/duailibe/linear-cli/releases/download/v#{version}/linear_#{version}_darwin_arm64.tar.gz"
      sha256 "${darwin_arm64}"
    else
      url "https://github.com/duailibe/linear-cli/releases/download/v#{version}/linear_#{version}_darwin_amd64.tar.gz"
      sha256 "${darwin_amd64}"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/duailibe/linear-cli/releases/download/v#{version}/linear_#{version}_linux_arm64.tar.gz"
      sha256 "${linux_arm64}"
    else
      url "https://github.com/duailibe/linear-cli/releases/download/v#{version}/linear_#{version}_linux_amd64.tar.gz"
      sha256 "${linux_amd64}"
    end
  end

  def install
    bin.install "linear"
  end

  test do
    assert_match "linear version", shell_output("#{bin}/linear --version")
  end
end
EOF

git -C homebrew-tap config user.name "github-actions[bot]"
git -C homebrew-tap config user.email "github-actions[bot]@users.noreply.github.com"
git -C homebrew-tap add Formula/linear-cli.rb
git -C homebrew-tap commit -m "linear-cli v${version}"
git -C homebrew-tap push
