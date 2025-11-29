# typed: false
# frozen_string_literal: true

# Homebrew formula for pipeboard
# Install: brew install blackwell-systems/homebrew-tap/pipeboard
# Or tap first: brew tap blackwell-systems/homebrew-tap && brew install pipeboard

class Pipeboard < Formula
  desc "Cross-platform clipboard CLI with sync, transforms, and peer sharing"
  homepage "https://github.com/blackwell-systems/pipeboard"
  version "0.5.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/blackwell-systems/pipeboard/releases/download/v#{version}/pipeboard_#{version}_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER_DARWIN_ARM64_SHA256"
    else
      url "https://github.com/blackwell-systems/pipeboard/releases/download/v#{version}/pipeboard_#{version}_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER_DARWIN_AMD64_SHA256"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/blackwell-systems/pipeboard/releases/download/v#{version}/pipeboard_#{version}_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_ARM64_SHA256"
    else
      url "https://github.com/blackwell-systems/pipeboard/releases/download/v#{version}/pipeboard_#{version}_linux_amd64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_AMD64_SHA256"
    end
  end

  def install
    bin.install "pipeboard"

    # Install shell completions
    output = Utils.safe_popen_read("#{bin}/pipeboard", "completion", "bash")
    (bash_completion/"pipeboard").write output

    output = Utils.safe_popen_read("#{bin}/pipeboard", "completion", "zsh")
    (zsh_completion/"_pipeboard").write output

    output = Utils.safe_popen_read("#{bin}/pipeboard", "completion", "fish")
    (fish_completion/"pipeboard.fish").write output
  end

  def caveats
    <<~EOS
      To get started, run:
        pipeboard init

      Shell completions have been installed.
      You may need to restart your shell or source the completions.

      For more information:
        pipeboard --help
        pipeboard doctor
    EOS
  end

  test do
    assert_match "pipeboard v#{version}", shell_output("#{bin}/pipeboard version")
    assert_match "Backend:", shell_output("#{bin}/pipeboard backend")
  end
end
