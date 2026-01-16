# HyperTunnel

**P2P secure file and directory transfer tool** - Transfer files directly between machines behind NAT without a central server, using WebRTC technology.

Inspired by [magic-wormhole](https://github.com/magic-wormhole/magic-wormhole), [gfile](https://github.com/Antonito/gfile), and [croc](https://github.com/schollz/croc).

## Installation

### Debian/Ubuntu (APT)

Download the latest `.deb` package from the [releases page](https://github.com/abrekhov/hypertunnel/releases):

```bash
# Download the latest .deb package
wget https://github.com/abrekhov/hypertunnel/releases/latest/download/hypertunnel_VERSION_linux_amd64.deb

# Install with apt
sudo apt install ./hypertunnel_VERSION_linux_amd64.deb

# Or use dpkg directly
sudo dpkg -i hypertunnel_VERSION_linux_amd64.deb
```

### RPM-based Linux (Fedora, RHEL, CentOS)

```bash
# Download the latest .rpm package
wget https://github.com/abrekhov/hypertunnel/releases/latest/download/hypertunnel-VERSION-amd64.rpm

# Install with dnf/yum
sudo dnf install ./hypertunnel-VERSION-amd64.rpm
# or
sudo yum install ./hypertunnel-VERSION-amd64.rpm
```

### Alpine Linux

```bash
# Download the latest .apk package
wget https://github.com/abrekhov/hypertunnel/releases/latest/download/hypertunnel_VERSION_linux_amd64.apk

# Install with apk
sudo apk add --allow-untrusted ./hypertunnel_VERSION_linux_amd64.apk
```

### macOS / Linux (Archives)

```bash
# Download and extract the archive for your platform
wget https://github.com/abrekhov/hypertunnel/releases/latest/download/hypertunnel_VERSION_OS_ARCH.tar.gz
tar -xzf hypertunnel_VERSION_OS_ARCH.tar.gz
sudo mv ht /usr/local/bin/
```

### Windows

Download the `.zip` archive from the [releases page](https://github.com/abrekhov/hypertunnel/releases) and extract `ht.exe` to your desired location. Add it to your PATH for easy access.

### From Source

```bash
# Clone the repository
git clone https://github.com/abrekhov/hypertunnel
cd hypertunnel
go build -o ht
```

Or using go install (GOBIN must be set):

```bash
export GOPATH=$HOME/go
export GOBIN="${GOPATH}/bin"
export PATH="$PATH:${GOPATH}/bin:${GOROOT}/bin"
go install github.com/abrekhov/hypertunnel@latest
```

## Usage

Both computers must have access to the Internet!

### Transfer a File

```bash
# First machine (sender)
./ht -f <file>

# Second machine (receiver)
./ht

# Cross insert signals (copy-paste the encoded signal from each side)
```

### Transfer a Directory

```bash
# First machine (sender)
./ht -f <directory>

# Second machine (receiver)
./ht

# Cross insert signals (copy-paste the encoded signal from each side)
# The directory will be automatically archived, transferred, and extracted
```

### Additional Options

```bash
# Auto-accept incoming files/directories (skip confirmation prompts)
./ht --auto-accept

# Enable verbose logging
./ht -v -f <file-or-directory>

# Encrypt a file before transfer
./ht encrypt -k "your-passphrase" <file>

# Decrypt a received encrypted file
./ht decrypt -k "your-passphrase" <file.enc>
```

## Features

- ✅ **File Transfer**: Secure P2P file transfer between machines behind NAT
- ✅ **Directory Transfer**: Recursive directory transfer with automatic archiving (tar.gz)
- ✅ **Encryption/Decryption**: AES-256-CTR file encryption with keyphrase
- ✅ **Progress Tracking**: Real-time progress monitoring with checksums
- ✅ **NAT Traversal**: WebRTC-based connection through STUN/TURN/ICE
- ✅ **Auto-accept Mode**: Skip confirmation prompts for automation
- ✅ **File Integrity**: SHA-256 checksum verification
- ✅ **Cross-platform**: Linux, macOS, Windows support

## Roadmap

### Completed ✅
- [x] AES-256-CTR file encryption/decryption
- [x] WebRTC P2P connection with NAT traversal
- [x] File transfer between peers behind NAT
- [x] Directory transfer with automatic tar.gz archiving
- [x] Progress tracking and SHA-256 checksums
- [x] Comprehensive test suite (70%+ coverage)
- [x] CI/CD with cross-platform testing
- [x] Linux packages (DEB/RPM/APK)
- [x] Basic TUI framework (Bubble Tea)
- [x] Auto-accept mode and overwrite protection

### In Progress 🚧
- [ ] Full TUI with real-time progress updates
- [ ] Bi-directional candidate exchange (start in any order)
- [ ] Multiple STUN/TURN server support
- [ ] Homebrew tap for macOS

### Planned 📋
- [ ] Resumable transfers
- [ ] SSH tunnel mode
- [ ] Port forwarding
- [ ] Performance benchmarks

See [CLAUDE.md](CLAUDE.md) for detailed development roadmap.
