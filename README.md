# HyperTunnel

P2P secure file transfer tool using WebRTC. Transfer files directly between machines behind NAT without a server.

## Quick Start

**From Source (recommended for testing):**

```bash
# Clone and build
git clone https://github.com/abrekhov/hypertunnel
cd hypertunnel
go build -o ht

# Sender (machine with the file)
./ht -f myfile.txt

# Receiver (other machine)
./ht
```

**Example: Transfer between macOS and cloud VM:**

```bash
# On your Mac (sender):
./ht -f important-data.tar.gz
# Copy the printed base64 signal

# On the cloud VM (receiver):
./ht
# Paste the Mac's signal, press Enter twice
# Copy the VM's signal back to Mac

# On Mac: paste VM's signal, press Enter twice
# Transfer starts automatically
```

**Tips:**
- Both machines need internet access
- Press Enter twice after pasting a signal
- Use `--auto-accept` to skip confirmation prompts
- Use `-v` for verbose debug output

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

## RoadMap

**Completed:**
- [X] Encrypt/decrypt file with key as stream (AES-256-CTR)
- [X] WebRTC P2P connection through STUN/TURN/ICE
- [X] ORTC connection behind NAT
- [X] Single file transfer between peers
- [X] Directory transfer with automatic archiving (tar.gz)
- [X] Progress tracking and SHA-256 checksums
- [X] Auto-accept mode (`--auto-accept`)
- [X] File overwrite protection with prompts
- [X] Tests infrastructure (unit + integration)
- [X] Multi-platform packaging (DEB/RPM/APK)
- [X] Basic TUI framework (Bubble Tea)

**In Progress:**
- [ ] Start candidates in any order (symmetric connection)
- [ ] Terminal UI polish (keep signals copyable!)

**Planned:**
- [ ] Resumable transfers
- [ ] SSH tunnel mode
- [ ] Benchmarks
- [ ] Homebrew/Scoop packaging
