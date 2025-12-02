#!/bin/bash
# PPA Upload Script for pipeboard
# This script builds source packages and uploads them to Ubuntu PPA

set -e

# Configuration
PPA="${PPA:-ppa:blackwell-systems/pipeboard}"
GPG_KEY="${GPG_KEY:-}"  # Set to your GPG key ID, or leave empty to use default

# Ubuntu releases to build for (latest LTS and current releases)
RELEASES=("noble" "mantic" "jammy" "focal")

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

    for cmd in debuild dput gpg git; do
        if ! command -v $cmd &> /dev/null; then
            missing+=($cmd)
        fi
    done

    if [ ${#missing[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing[*]}"
        log_info "Install with: sudo apt install devscripts debhelper dput gnupg git"
        exit 1
    fi
}

# Get version from git tag
get_version() {
    git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0"
}

# Update debian/changelog for specific Ubuntu release
update_changelog() {
    local version=$1
    local release=$2
    local iteration=$3

    log_info "Updating changelog for $release (iteration $iteration)"

    DEBEMAIL="blackwellsystems@protonmail.com" \
    DEBFULLNAME="Blackwell Systems" \
    dch --force-distribution \
        --distribution "$release" \
        --newversion "${version}-${iteration}ubuntu1~${release}1" \
        "Build for $release"
}

# Build source package
build_source() {
    log_info "Building source package..."

    if [ -n "$GPG_KEY" ]; then
        debuild -S -sa -k"$GPG_KEY"
    else
        debuild -S -sa
    fi
}

# Upload to PPA
upload_to_ppa() {
    local changes_file=$1

    log_info "Uploading $changes_file to $PPA..."
    dput "$PPA" "../$changes_file"
}

# Main process
main() {
    log_info "Starting PPA upload process for pipeboard"

    # Check we're in the right directory
    if [ ! -f "debian/control" ]; then
        log_error "debian/control not found. Run this script from the repository root."
        exit 1
    fi

    # Check dependencies
    check_dependencies

    # Get version
    VERSION=$(get_version)
    log_info "Package version: $VERSION"

    # Check GPG key
    if [ -z "$GPG_KEY" ]; then
        log_warn "GPG_KEY not set, using default key"
        log_info "Set GPG_KEY environment variable to specify a key"
    fi

    # Save original changelog
    cp debian/changelog debian/changelog.orig

    # Build for each Ubuntu release
    ITERATION=1
    for release in "${RELEASES[@]}"; do
        log_info "========================================"
        log_info "Building for Ubuntu $release"
        log_info "========================================"

        # Restore original changelog
        cp debian/changelog.orig debian/changelog

        # Update changelog for this release
        update_changelog "$VERSION" "$release" "$ITERATION"

        # Build source package
        build_source

        # Upload to PPA
        CHANGES_FILE="pipeboard_${VERSION}-${ITERATION}ubuntu1~${release}1_source.changes"
        upload_to_ppa "$CHANGES_FILE"

        log_info "✓ Successfully uploaded for $release"

        # Clean up build artifacts
        git checkout debian/changelog
    done

    # Restore original changelog
    mv debian/changelog.orig debian/changelog

    # Clean up
    log_info "Cleaning up build artifacts..."
    rm -f ../*.build ../*.buildinfo ../*.changes ../*.dsc ../*.tar.* ../*.upload

    log_info "========================================"
    log_info "All uploads complete!"
    log_info "========================================"
    log_info "Monitor build status at:"
    log_info "https://launchpad.net/~blackwell-systems/+archive/ubuntu/pipeboard"
}

# Run main function
main "$@"
