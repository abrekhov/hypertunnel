# Installation Guide

This guide provides detailed installation instructions for HyperTunnel across different platforms.

## Table of Contents

- [Linux](#linux)
  - [Debian/Ubuntu (DEB)](#debianubuntu-deb)
  - [Fedora/RHEL/CentOS (RPM)](#fedorarhel centos-rpm)
  - [Alpine Linux (APK)](#alpine-linux-apk)
  - [Arch Linux (AUR)](#arch-linux-aur)
- [macOS](#macos)
- [Windows](#windows)
- [From Source](#from-source)
- [Verification](#verification)

---

## Linux

### One-line Installer (Linux/macOS)

This is the fastest way to install on a fresh host (downloads the latest release binary):

```bash
curl -fsSL https://abrekhov.github.io/hypertunnel/install.sh | sh
```

### Debian/Ubuntu (DEB)

1. **Download the DEB package:**

   Visit the [releases page](https://github.com/abrekhov/hypertunnel/releases) and download the appropriate `.deb` file for your architecture.

   ```bash
   # For AMD64 (most common)
   wget https://github.com/abrekhov/hypertunnel/releases/latest/download/hypertunnel_VERSION_linux_amd64.deb

   # For ARM64 (Raspberry Pi, Apple Silicon via Linux)
   wget https://github.com/abrekhov/hypertunnel/releases/latest/download/hypertunnel_VERSION_linux_arm64.deb
   ```

2. **Install the package:**

   ```bash
   sudo apt install ./hypertunnel_VERSION_linux_amd64.deb
   ```

   Or using `dpkg`:

   ```bash
   sudo dpkg -i hypertunnel_VERSION_linux_amd64.deb
   sudo apt-get install -f  # Install any missing dependencies
   ```

3. **Verify installation:**

   ```bash
   ht --version
   # or
   hypertunnel --version
   ```

### Fedora/RHEL/CentOS (RPM)

1. **Download the RPM package:**

   ```bash
   # For AMD64
   wget https://github.com/abrekhov/hypertunnel/releases/latest/download/hypertunnel-VERSION-amd64.rpm

   # For ARM64
   wget https://github.com/abrekhov/hypertunnel/releases/latest/download/hypertunnel-VERSION-arm64.rpm
   ```

2. **Install the package:**

   **Fedora (dnf):**
   ```bash
   sudo dnf install ./hypertunnel-VERSION-amd64.rpm
   ```

   **RHEL/CentOS (yum):**
   ```bash
   sudo yum install ./hypertunnel-VERSION-amd64.rpm
   ```

   **openSUSE (zypper):**
   ```bash
   sudo zypper install ./hypertunnel-VERSION-amd64.rpm
   ```

3. **Verify installation:**

   ```bash
   ht --version
   ```

### Alpine Linux (APK)

1. **Download the APK package:**

   ```bash
   wget https://github.com/abrekhov/hypertunnel/releases/latest/download/hypertunnel_VERSION_linux_amd64.apk
   ```

2. **Install the package:**

   ```bash
   sudo apk add --allow-untrusted ./hypertunnel_VERSION_linux_amd64.apk
   ```

3. **Verify installation:**

   ```bash
   ht --version
   ```

### Arch Linux (AUR)

Coming soon! PKGBUILD will be published to the AUR.

For now, use the generic Linux binary or build from source.

---

## macOS

### Using Pre-built Binary

1. **Download the archive:**

   ```bash
   # For Intel Macs (AMD64)
   wget https://github.com/abrekhov/hypertunnel/releases/latest/download/hypertunnel_VERSION_darwin_amd64.tar.gz

   # For Apple Silicon (ARM64)
   wget https://github.com/abrekhov/hypertunnel/releases/latest/download/hypertunnel_VERSION_darwin_arm64.tar.gz
   ```

2. **Extract and install:**

   ```bash
   tar -xzf hypertunnel_VERSION_darwin_*.tar.gz
   sudo mv ht /usr/local/bin/
   sudo chmod +x /usr/local/bin/ht
   ```

3. **Verify installation:**

   ```bash
   ht --version
   ```

### Using Homebrew

Coming soon! HyperTunnel will be available via Homebrew tap.

Planned commands:
```bash
brew tap abrekhov/tap
brew install hypertunnel
```

---

## Windows

### Using Pre-built Binary

1. **Download the ZIP archive:**

   Visit the [releases page](https://github.com/abrekhov/hypertunnel/releases) and download:

   - `hypertunnel_VERSION_windows_amd64.zip` (64-bit)
   - `hypertunnel_VERSION_windows_arm64.zip` (ARM64)

2. **Extract the archive:**

   Right-click the ZIP file and select "Extract All..."

3. **Move the binary:**

   Move `ht.exe` to a directory in your PATH, such as:
   - `C:\Program Files\HyperTunnel\`
   - `C:\Windows\System32\` (not recommended)
   - Or add the extracted directory to your PATH

4. **Add to PATH (optional):**

   - Open "System Properties" → "Advanced" → "Environment Variables"
   - Edit the "Path" variable
   - Add the directory containing `ht.exe`

5. **Verify installation:**

   Open Command Prompt or PowerShell:
   ```cmd
   ht --version
   ```

### Using Package Managers

**Scoop** (Coming soon):
```powershell
scoop bucket add abrekhov https://github.com/abrekhov/scoop-bucket
scoop install hypertunnel
```

**Chocolatey** (Coming soon):
```powershell
choco install hypertunnel
```

---

## From Source

### Prerequisites

- Go 1.23 or later
- Git

### Installation Steps

1. **Clone the repository:**

   ```bash
   git clone https://github.com/abrekhov/hypertunnel.git
   cd hypertunnel
   ```

2. **Build the binary:**

   ```bash
   go build -o ht
   ```

3. **Install (optional):**

   **Linux/macOS:**
   ```bash
   sudo mv ht /usr/local/bin/
   ```

   **Or use `go install`:**
   ```bash
   go install github.com/abrekhov/hypertunnel@latest
   ```

   Make sure `$GOBIN` or `$GOPATH/bin` is in your PATH:
   ```bash
   export PATH="$PATH:$(go env GOPATH)/bin"
   ```

4. **Verify installation:**

   ```bash
   ht --version
   ```

---

## Verification

### Checksum Verification

To verify the integrity of downloaded packages:

1. **Download checksums:**

   ```bash
   wget https://github.com/abrekhov/hypertunnel/releases/latest/download/checksums.txt
   ```

2. **Verify the package:**

   **Linux/macOS:**
   ```bash
   sha256sum -c checksums.txt --ignore-missing
   ```

   **Windows (PowerShell):**
   ```powershell
   Get-FileHash hypertunnel_VERSION_windows_amd64.zip -Algorithm SHA256
   # Compare with checksums.txt manually
   ```

### GPG Signature Verification

*Currently, packages are not GPG-signed. This feature will be added in future releases.*

---

## Uninstallation

### DEB (Debian/Ubuntu)
```bash
sudo apt remove hypertunnel
```

### RPM (Fedora/RHEL)
```bash
sudo dnf remove hypertunnel
# or
sudo yum remove hypertunnel
```

### APK (Alpine)
```bash
sudo apk del hypertunnel
```

### Manual Installation
```bash
sudo rm /usr/local/bin/ht
sudo rm /usr/local/bin/hypertunnel  # If symlink exists
```

---

## Troubleshooting

### Command not found after installation

**Linux/macOS:**
- Verify PATH includes installation directory:
  ```bash
  echo $PATH
  ```
- Reload shell configuration:
  ```bash
  source ~/.bashrc  # or ~/.zshrc
  ```

**Windows:**
- Close and reopen Command Prompt/PowerShell
- Verify PATH environment variable

### Permission denied

**Linux/macOS:**
```bash
sudo chmod +x /usr/local/bin/ht
```

### DEB installation fails with dependencies

```bash
sudo apt-get install -f
```

### Connection issues during file transfer

- Ensure both machines have internet access
- Check firewall settings (may block STUN/TURN connections)
- Try enabling verbose mode: `ht -v`

---

## Getting Help

- **Documentation:** [README.md](README.md)
- **Issues:** [GitHub Issues](https://github.com/abrekhov/hypertunnel/issues)
- **Discussions:** [GitHub Discussions](https://github.com/abrekhov/hypertunnel/discussions)

---

## Next Steps

After installation, see the [Usage Guide](README.md#usage) to start transferring files!
