# HyperTunnel

## Installation

```bash
git clone https://github.com/abrekhov/hypertunnel
cd hypertunnel
go build -o ht
```

Or using go install (GOBIN must be set)

```bash
export GOPATH=$HOME/go
export GOBIN="${GOPATH}/bin"
export PATH="$PATH:${GOPATH}/bin:${GOROOT}/bin"
go install github.com/abrekhov/hypertunnel
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
- ✅ **Flexible Startup**: Peers can start in any order (symmetric connection setup)
- ✅ **Auto-accept Mode**: Skip confirmation prompts for automation
- ✅ **File Integrity**: SHA-256 checksum verification
- ✅ **Cross-platform**: Linux, macOS, Windows support

## RoadMap

- [X] Encrypt file with key as stream
- [X] Decrypt file with key as stream
- [X] TCP/IP Connection through stun/turn/ice
- [X] ORTC connection behind NAT
- [X] Move one file between candidates behind NAT
- [X] Directory transfer with automatic archiving
- [X] Progress tracking and checksums
- [X] Tests infrastructure
- [X] Start candidates in any order (symmetric connection setup)
- [ ] Terminal UI with progress bars (Phase 2)
- [ ] Resumable transfers
- [ ] SSH server behind NAT
- [ ] Benchmarks
