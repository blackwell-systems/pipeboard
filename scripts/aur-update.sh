#!/bin/bash
# AUR Update Script for pipeboard
# This script updates the PKGBUILD and pushes to AUR

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check required tools
check_dependencies() {
    local missing=()

    for cmd in git makepkg updpkgsums ssh; do
        if ! command -v $cmd &> /dev/null; then
            missing+=($cmd)
        fi
    done

    if [ ${#missing[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing[*]}"
        log_info "Install with: sudo pacman -S git base-devel openssh"
        exit 1
    fi
}

# Get version from git tag
get_version() {
    git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0"
}

# Update PKGBUILD version
update_pkgbuild() {
    local version=$1

    log_info "Updating PKGBUILD to version $version"

    # Update pkgver
    sed -i "s/^pkgver=.*/pkgver=$version/" aur/PKGBUILD

    # Reset pkgrel to 1 for new versions
    sed -i "s/^pkgrel=.*/pkgrel=1/" aur/PKGBUILD
}

# Update checksums
update_checksums() {
    log_info "Updating checksums..."

    cd aur
    updpkgsums PKGBUILD
    cd ..
}

# Generate .SRCINFO
generate_srcinfo() {
    log_info "Generating .SRCINFO..."

    cd aur
    makepkg --printsrcinfo > .SRCINFO
    cd ..
}

# Test build package
test_build() {
    log_info "Testing package build..."

    cd aur

    # Clean previous builds
    rm -rf pkg/ src/ *.tar.gz

    # Test build (without installing)
    makepkg --syncdeps --noconfirm --skipinteg

    log_info "✓ Test build successful"

    # Clean up
    rm -rf pkg/ src/ *.tar.gz

    cd ..
}

# Clone or update AUR repository
prepare_aur_repo() {
    local aur_repo_dir="$HOME/.cache/pipeboard-aur"

    if [ ! -d "$aur_repo_dir" ]; then
        log_info "Cloning AUR repository..."
        git clone ssh://aur@aur.archlinux.org/pipeboard.git "$aur_repo_dir"
    else
        log_info "Updating AUR repository..."
        cd "$aur_repo_dir"
        git pull
        cd -
    fi

    echo "$aur_repo_dir"
}

# Push to AUR
push_to_aur() {
    local version=$1
    local aur_repo_dir=$(prepare_aur_repo)

    log_info "Copying files to AUR repository..."
    cp aur/PKGBUILD "$aur_repo_dir/"
    cp aur/.SRCINFO "$aur_repo_dir/"

    cd "$aur_repo_dir"

    # Check if there are changes
    if ! git diff --quiet; then
        log_info "Committing changes..."
        git add PKGBUILD .SRCINFO
        git commit -m "Update to version $version"

        log_info "Pushing to AUR..."
        git push

        log_info "✓ Successfully pushed to AUR!"
    else
        log_warn "No changes detected, skipping push"
    fi

    cd -
}

# Main process
main() {
    log_info "Starting AUR update process for pipeboard"

    # Check we're in the right directory
    if [ ! -f "aur/PKGBUILD" ]; then
        log_error "aur/PKGBUILD not found. Run this script from the repository root."
        exit 1
    fi

    # Check dependencies
    check_dependencies

    # Get version
    VERSION=$(get_version)
    log_info "Package version: $VERSION"

    # Check SSH access to AUR
    log_info "Checking AUR SSH access..."
    if ! ssh -T aur@aur.archlinux.org 2>&1 | grep -q "successfully authenticated"; then
        log_error "Cannot connect to AUR. Please set up SSH keys:"
        log_info "1. Generate SSH key: ssh-keygen -t ed25519 -C 'aur'"
        log_info "2. Add to AUR: https://aur.archlinux.org/account/"
        exit 1
    fi

    # Update PKGBUILD
    update_pkgbuild "$VERSION"

    # Update checksums
    update_checksums

    # Generate .SRCINFO
    generate_srcinfo

    # Test build (optional, comment out if you want to skip)
    read -p "Test build package? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        test_build
    fi

    # Push to AUR
    read -p "Push to AUR? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        push_to_aur "$VERSION"

        log_info "========================================"
        log_info "AUR update complete!"
        log_info "========================================"
        log_info "Users can now install with:"
        log_info "  yay -S pipeboard"
        log_info ""
        log_info "Check package at:"
        log_info "  https://aur.archlinux.org/packages/pipeboard"
    else
        log_warn "Skipped push to AUR"
    fi
}

# Run main function
main "$@"
