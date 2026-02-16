# Packaging and Distribution Guide

This document explains how HyperTunnel is packaged and distributed across different platforms.

## Overview

HyperTunnel uses [GoReleaser](https://goreleaser.com/) with [nFPM](https://nfpm.goreleaser.com/) to create packages for multiple platforms and formats:

- **DEB** packages (Debian, Ubuntu)
- **RPM** packages (Fedora, RHEL, CentOS, openSUSE)
- **APK** packages (Alpine Linux)
- **Archives** (tar.gz for Linux/macOS, zip for Windows)

## Build Process

### Automated Releases

Releases are automatically built when a new version tag is pushed:

```bash
# Create and push a new version tag
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

This triggers the GitHub Actions workflow (`.github/workflows/release.yaml`) which:

1. Checks out the repository
2. Sets up Go environment
3. Runs GoReleaser with the configuration from `.goreleaser.yaml`
4. Creates packages for all platforms
5. Uploads packages to GitHub Releases

### Local Testing

To test the packaging locally without creating a release:

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Build packages without releasing
goreleaser release --snapshot --clean

# Output will be in the ./dist directory
```

## Package Details

### Binary Name

The main binary is named `ht` and is installed to:
- **Linux/macOS**: `/usr/bin/ht`
- **Windows**: User-defined location

A symbolic link `hypertunnel` → `ht` is also created for convenience.

### Package Contents

All packages include:
- `ht` binary
- LICENSE file → `/usr/share/doc/hypertunnel/copyright`
- README.md → `/usr/share/doc/hypertunnel/README.md`

### Post-Installation Script

The `scripts/postinstall.sh` script runs after installation and:
- Creates a symbolic link `/usr/bin/hypertunnel` → `/usr/bin/ht`
- Displays installation success message

### Dependencies

Packages depend on:
- `ca-certificates` (for HTTPS connections to STUN/TURN servers)

## Distribution Channels

### GitHub Releases (Primary)

All packages are published to GitHub Releases:
- https://github.com/abrekhov/hypertunnel/releases

Users can download packages directly from there.

### Future Distribution Plans

#### APT Repository (Launchpad PPA)

To set up an official Ubuntu PPA:

1. Create a Launchpad account
2. Generate GPG key for package signing
3. Create debian packaging files
4. Upload to Launchpad

Instructions: https://help.launchpad.net/Packaging/PPA

#### Homebrew

To add HyperTunnel to Homebrew:

**Option 1: Personal Tap (Quick)**
```bash
# Create homebrew-tap repository
# Add Formula/hypertunnel.rb

# Users install with:
brew tap abrekhov/tap
brew install hypertunnel
```

**Option 2: Official Homebrew Core (Requires Review)**
```bash
# Fork homebrew-core
# Add Formula/h/hypertunnel.rb
# Submit PR
```

Update `.goreleaser.yaml` to automate Homebrew tap updates.

#### Arch Linux AUR

Create and publish PKGBUILD to AUR:
1. Create PKGBUILD file
2. Test with `makepkg -si`
3. Push to AUR repository

See: https://wiki.archlinux.org/title/AUR_submission_guidelines

#### Windows Package Managers

**Scoop:**
- Create manifest in scoop bucket
- Submit to scoop-extras

**Chocolatey:**
- Create .nuspec file
- Submit to Chocolatey community feed

## Version Management

### Version Tagging Convention

Use semantic versioning: `vMAJOR.MINOR.PATCH`

Examples:
- `v1.0.0` - First stable release
- `v1.1.0` - New features added
- `v1.1.1` - Bug fixes
- `v2.0.0` - Breaking changes

### Pre-release Versions

For alpha/beta releases:
- `v1.0.0-alpha.1`
- `v1.0.0-beta.1`
- `v1.0.0-rc.1`

These will be marked as pre-releases on GitHub.

## Package Naming

### DEB/APK/RPM
Stable, versionless filenames are used so `releases/latest/download/...` links work:

```
hypertunnel_linux_amd64.deb
hypertunnel_linux_arm64.deb
hypertunnel_linux_amd64.rpm
hypertunnel_linux_arm64.rpm
hypertunnel_linux_amd64.apk
hypertunnel_linux_arm64.apk
```

### Raw Binaries
```
ht_linux_amd64
ht_linux_arm64
ht_darwin_amd64
ht_darwin_arm64
ht_windows_amd64.exe
```

### Archives
```
hypertunnel_linux_amd64.tar.gz
hypertunnel_linux_arm64.tar.gz
hypertunnel_darwin_amd64.tar.gz
hypertunnel_darwin_arm64.tar.gz
hypertunnel_windows_amd64.zip
```

## Checksums

A `checksums.txt` file is generated for all packages and binaries, containing SHA-256 hashes.

Users can verify downloads:
```bash
sha256sum -c checksums.txt
```

## Signing Packages

### Current State
Packages are **not signed** yet.

### Future: GPG Signing

To sign DEB/RPM packages:

1. Generate GPG key:
   ```bash
   gpg --gen-key
   ```

2. Export public key:
   ```bash
   gpg --armor --export your@email.com > public.key
   ```

3. Add to `.goreleaser.yaml`:
   ```yaml
   signs:
     - artifacts: checksum
       args:
         - "--batch"
         - "--local-user"
         - "{{ .Env.GPG_FINGERPRINT }}"
         - "--output"
         - "${signature}"
         - "--detach-sign"
         - "${artifact}"
   ```

4. Set GPG_FINGERPRINT in GitHub secrets

## Troubleshooting

### Build Failures

**Issue:** GoReleaser fails to build
- Check Go version compatibility (requires Go 1.23+)
- Verify `.goreleaser.yaml` syntax with `goreleaser check`

**Issue:** Missing dependencies in package
- Update `nfpms.dependencies` in `.goreleaser.yaml`

### Installation Issues

**DEB: Dependency errors**
```bash
sudo apt-get install -f
```

**RPM: Conflicts with existing files**
```bash
sudo rpm -e hypertunnel  # Remove old version first
```

## References

- [GoReleaser Documentation](https://goreleaser.com/intro/)
- [nFPM Documentation](https://nfpm.goreleaser.com/)
- [Debian Package Guide](https://www.debian.org/doc/manuals/maint-guide/)
- [RPM Packaging Guide](https://rpm-packaging-guide.github.io/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
