# AGENTS.md - HyperTunnel Codebase Guide for AI Assistants

This document provides a comprehensive overview of the HyperTunnel codebase for AI assistants working on this project.

**Note:** Always keep this file in sync with `CLAUDE.md`.

## Project Overview

**HyperTunnel** is a peer-to-peer (P2P) secure file transfer tool written in Go that enables direct file transfers between two machines behind NAT without requiring a central server. It uses WebRTC technology for NAT traversal and DTLS for encryption.

**Key Features:**
- Direct P2P file transfer using WebRTC data channels
- Directory transfer with automatic archiving (tar.gz)
- NAT traversal via ICE/STUN/TURN protocols
- Built-in file encryption/decryption with AES-256-CTR
- Manual signal exchange (copy-paste) for security
- Progress tracking with SHA-256 checksums
- Auto-accept mode for automation (`--auto-accept`)
- File overwrite protection with prompts
- Cross-platform support (Linux, macOS, Windows)
- Multi-platform packaging (DEB/RPM/APK)
- Basic TUI framework (Bubble Tea)

**Project Stats:**
- Language: Go 1.23
- Total Go Files: 26
- Lines of Code: ~2000+
- License: Apache 2.0
- Repository: https://github.com/abrekhov/hypertunnel

---

## Repository Structure

```
/home/user/hypertunnel/
├── .git/                          # Git repository
├── .github/
│   └── workflows/
│       └── release.yaml           # GitHub Actions CI/CD for releases
├── cmd/                           # CLI command implementations (Cobra)
│   ├── root.go                   # Main command, connection logic
│   ├── encrypt.go                # File encryption command
│   └── decrypt.go                # File decryption command
├── pkg/                           # Internal reusable packages
│   ├── archive/                  # Directory archiving (tar.gz)
│   │   ├── archive.go            # Create/extract tar.gz archives
│   │   └── archive_test.go       # Archive tests
│   ├── datachannel/              # WebRTC data channel utilities
│   │   ├── datachannel.go        # SDP encoding/decoding
│   │   ├── signal.go             # Signal struct for WebRTC handshake
│   │   ├── handlers.go           # File/directory transfer handlers
│   │   ├── datachannel_test.go   # Encoding tests
│   │   └── handlers_test.go      # Handler tests
│   ├── hashutils/                # Cryptographic utilities
│   │   ├── hashutils.go          # Key hashing
│   │   └── hashutils_test.go     # Hash tests
│   ├── transfer/                 # Transfer utilities
│   │   ├── progress.go           # Progress tracking
│   │   ├── checksum.go           # SHA-256 checksum verification
│   │   ├── metadata.go           # Transfer metadata
│   │   └── *_test.go             # Tests
│   └── tui/                      # Terminal UI (Bubble Tea)
│       ├── tui.go                # Main TUI model
│       ├── connection.go         # Connection screen
│       ├── transfer.go           # Transfer progress screen
│       └── tui_test.go           # TUI tests
├── integration_test.go            # End-to-end tests
├── main.go                        # Application entry point
├── go.mod                         # Go module dependencies
├── go.sum                         # Dependency checksums
├── .goreleaser.yaml              # GoReleaser configuration for builds
├── .gitignore                     # Git ignore patterns
├── LICENSE                        # Apache License 2.0
├── README.md                      # User-facing documentation
├── CLAUDE.md                      # AI assistant guide (this file)
└── AGENTS.md                      # AI assistant guide (sync with CLAUDE.md)
```

---

## Core Components and Architecture

### 1. Main Entry Point (`main.go`)

**Location:** `/home/user/hypertunnel/main.go`

Simple entry point that delegates to the Cobra CLI framework:

```go
package main
import "github.com/abrekhov/hypertunnel/cmd"
func main() {
    cmd.Execute()
}
```

### 2. Root Command (`cmd/root.go` - 246 lines)

**Purpose:** Main CLI logic and WebRTC P2P connection orchestration

**Key Responsibilities:**
- Determines sender vs. receiver mode based on `-f` flag
- Sets up WebRTC stack (ICE/DTLS/SCTP)
- Manages ICE candidate gathering via STUN
- Handles signal exchange (manual copy-paste)
- Initiates file transfer via data channels

**Important Functions:**
- `Connection(cmd, args)` - Main connection logic (line 98-246)
- `Execute()` - Cobra command executor (line 59-61)
- `initConfig()` - Viper config initialization (line 75-96)

**WebRTC Flow:**
1. Create ICE gatherer with Google STUN server (`stun:stun.l.google.com:19302`)
2. Gather local ICE candidates
3. Construct Signal struct with ICE/DTLS/SCTP parameters
4. Base64 encode and print signal to console
5. Wait for user to paste remote signal
6. Decode remote signal
7. Start ICE/DTLS/SCTP transports
8. Sender: Open data channel, stream file in 65534-byte chunks
9. Receiver: Handle incoming data channel, write chunks to file

**Key Configuration:**
- Buffer size: 65534 bytes (max WebRTC data channel frame size)
- Timeout: 30 seconds after file transfer completion
- ICE Role: Controlling (sender) vs. Controlled (receiver)

### 3. Encryption Command (`cmd/encrypt.go` - 109 lines)

**Usage:** `ht encrypt -k "keyphrase" <filename>`

**Algorithm:** AES-256 in CTR (Counter) mode

**Process:**
1. Hash keyphrase using SHA256 → 32-byte AES key
2. Generate random 16-byte IV (Initialization Vector)
3. Stream-encrypt file in configurable chunks (default 1024 bytes)
4. Append IV to end of encrypted file
5. Output: `<filename>.enc`

**Key Functions:**
- `EncryptFile(cmd, args)` - Main encryption logic (line 57-109)

**Flags:**
- `-k, --key`: Keyphrase for encryption (required)
- `-b, --buffer`: Buffer size in bytes (default: 1024)

### 4. Decryption Command (`cmd/decrypt.go` - 114 lines)

**Usage:** `ht decrypt -k "keyphrase" <filename.enc>`

**Process:**
1. Extract IV from last 16 bytes of encrypted file
2. Hash keyphrase to derive AES key
3. Stream-decrypt file in chunks
4. Output: `<filename>.dec`

**Key Functions:**
- `DecryptFile(cmd, args)` - Main decryption logic

### 5. Data Channel Package (`pkg/datachannel/`)

**Purpose:** WebRTC signal encoding/decoding and file transfer handling

**Files:**

#### `signal.go` (18 lines)
Defines the Signal struct for WebRTC handshake:
```go
type Signal struct {
    ICECandidates    []*webrtc.ICECandidate
    ICEParameters    webrtc.ICEParameters
    DTLSParameters   webrtc.DTLSParameters
    SCTPCapabilities webrtc.SCTPCapabilities
}
```

#### `datachannel.go` (49 lines)
- `Encode(signal Signal) string` - Marshals Signal to JSON, base64 encodes
- `Decode(input string, signal *Signal)` - Base64 decodes, unmarshals JSON
- `MustReadStdin() string` - Reads multi-line input from stdin until double newline

#### `handlers.go` (64 lines)
- `FileTransferHandler(channel *webrtc.DataChannel)` - Main handler for incoming data channels
  - Creates file with channel label as filename
  - Registers OnMessage handler to write data chunks
  - Registers OnClose handler to finalize transfer
- `askForConfirmation(s string, in io.Reader) bool` - User confirmation prompt for incoming files.

#### `datachannel_test.go` (19 lines)
Skeleton tests (not implemented):
- `TestEncode(t *testing.T)` - Marked with `t.Error("Not yet implemented")`
- `TestMustReadStdin(t *testing.T)` - Empty

### 6. Hash Utilities (`pkg/hashutils/hashutils.go` - 22 lines)

**Purpose:** Cryptographic key derivation

**Key Function:**
- `FromKeyToAESKey(key string) []byte` - SHA256 hash of keyphrase → 32-byte AES key

---

## Dependencies and Build System

### Go Module (`go.mod`)

**Module:** `github.com/abrekhov/hypertunnel`
**Go Version:** 1.23

**Core Dependencies:**
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/pion/webrtc/v3` | v3.3.4 | WebRTC protocol implementation |
| `github.com/spf13/cobra` | v1.8.1 | CLI framework |
| `github.com/spf13/viper` | v1.19.0 | Configuration management |
| `github.com/sirupsen/logrus` | v1.9.3 | Structured logging |
| `github.com/chzyer/readline` | v1.5.1 | Enhanced readline for CLI |
| `github.com/mitchellh/go-homedir` | v1.1.0 | Cross-platform home directory |

**WebRTC Sub-dependencies (via Pion):**
- `pion/ice` - ICE protocol for NAT traversal
- `pion/dtls` - DTLS encryption layer
- `pion/sctp` - SCTP reliable streaming
- `pion/datachannel` - Data channel management
- `pion/stun` - STUN client for NAT discovery
- `pion/turn` - TURN relay fallback

### Build Commands

**Development Build:**
```bash
go build -o ht
```

**Install to GOBIN:**
```bash
export GOPATH=$HOME/go
export GOBIN="${GOPATH}/bin"
go install github.com/abrekhov/hypertunnel
```

**Cross-Platform Build:** Uses GoReleaser (see below)

### Linting and Code Quality

**golangci-lint Version:** v1.64.8 (CI/CD)

**IMPORTANT:** The project uses golangci-lint v1.64.8 in CI/CD. Always use this version for linting to ensure consistency.

**Installation:**
```bash
# Install specific version v1.64.8
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.64.8
```

**Configuration:** `.golangci.yaml`
- Compatible with golangci-lint v1.x
- For v2.x, the `version: 2` field must be added to the config
- Current config is optimized for v1.64.8

**Run linting:**
```bash
golangci-lint run ./...
```

**Key linters enabled:**
- `errcheck` - Check for unchecked errors
- `govet` - Vet examines Go source code
- `staticcheck` - Static analysis
- `gosec` - Security problems
- `revive` - Fast, configurable linter
- See `.golangci.yaml` for full list

**Note:** When fixing linting issues:
1. Only fix issues in files you're actively working on
2. Don't make sweeping changes to existing code unless explicitly required
3. Use `#nosec` comments with justification for intentional security exceptions
4. Pre-existing linting issues in `cmd/` files are acceptable

### GoReleaser Configuration (`.goreleaser.yaml`)

**Purpose:** Automated multi-platform binary releases

**Configuration:**
- **Entry Point:** `./main.go`
- **Binary Name:** `ht_{{ .Os }}_{{ .Arch }}`
- **Platforms:** Linux, macOS, Windows
- **Architectures:** amd64, arm64
- **CGO:** Disabled (`CGO_ENABLED=0`)
- **LDFLAGS:** `-s -w` (strip symbols and DWARF for smaller binaries)
- **Archive Format:** `binary` (no compression)
- **Checksums:** Generated in `checksums.txt`

**Output Binaries:**
- `ht_linux_amd64`
- `ht_linux_arm64`
- `ht_darwin_amd64`
- `ht_darwin_arm64`
- `ht_windows_amd64.exe`
- `ht_windows_arm64.exe`

### GitHub Actions CI/CD (`.github/workflows/release.yaml`)

**Trigger:** Git tags matching `v*.*.*` (e.g., `v1.0.0`)

**Workflow:**
1. Checkout repository with full history
2. Set up Go (latest stable)
3. Run GoReleaser with GitHub token
4. Publish release to GitHub Releases

---

## Development Workflows

### Adding New Features

1. **Create a feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Understand the existing code:**
   - Read relevant source files in `cmd/` or `pkg/`
   - Follow the modular structure (separate commands, packages)

3. **For new CLI commands:**
   - Create a new file in `cmd/` (e.g., `cmd/newcommand.go`)
   - Use Cobra command structure (see `encrypt.go` or `decrypt.go`)
   - Register command in `init()` function: `rootCmd.AddCommand(yourCmd)`

4. **For new utilities:**
   - Create package in `pkg/` if reusable
   - Follow existing naming conventions

5. **Test your changes:**
   ```bash
   go build -o ht
   ./ht --help
   ./ht your-command
   ```

6. **Commit and push:**
   ```bash
   git add .
   git commit -m "Add: Description of feature"
   git push -u origin feature/your-feature-name
   ```

### Making Bug Fixes

1. **Identify the bug location:**
   - Use `grep` to search for relevant code
   - Check `cmd/root.go` for connection logic
   - Check `pkg/datachannel/handlers.go` for transfer logic

2. **Fix the issue:**
   - Make minimal, focused changes
   - Preserve existing code style

3. **Test the fix:**
   - Build and run the binary
   - Test both sender and receiver modes

4. **Commit with clear message:**
   ```bash
   git commit -m "Fix: Description of bug fixed"
   ```

### Refactoring

**IMPORTANT:** Avoid over-engineering. The codebase is intentionally lean. Only refactor when:
- There's clear duplication
- Complexity is hindering maintenance
- Performance is measurably poor

**Guidelines:**
- Don't create abstractions for single-use code
- Don't add error handling for impossible scenarios
- Keep functions focused and simple

### Testing

**Current State:** Minimal testing infrastructure

**To Add Tests:**

1. Create test files alongside source: `filename_test.go`
2. Use Go's standard testing package:
   ```go
   package yourpackage
   import "testing"

   func TestYourFunction(t *testing.T) {
       // Test logic
   }
   ```

3. Run tests:
   ```bash
   go test ./...
   ```

**Priority Areas for Testing:**
- `pkg/datachannel/Encode()` and `Decode()`
- `pkg/hashutils/FromKeyToAESKey()`
- Encryption/decryption round-trip

### Release Process

1. **Update version in code (if needed)**

2. **Create and push a version tag:**
   ```bash
   git tag -a v1.0.0 -m "Release version 1.0.0"
   git push origin v1.0.0
   ```

3. **GitHub Actions automatically:**
   - Builds binaries for all platforms
   - Generates checksums
   - Creates GitHub Release
   - Uploads artifacts

---

## Code Conventions and Style

### File Organization

- **`cmd/`** - CLI commands (user-facing interface)
- **`pkg/`** - Internal packages (reusable logic)
- **`main.go`** - Entry point only, no business logic

### Naming Conventions

- **Packages:** Lowercase, single-word (e.g., `datachannel`, `hashutils`)
- **Exported functions:** PascalCase (e.g., `Connection`, `EncryptFile`)
- **Unexported functions:** camelCase (e.g., `askForConfirmation`)
- **Variables:** camelCase (e.g., `isOffer`, `keyphrase`)
- **Constants:** Not extensively used; follow Go conventions (PascalCase or UPPER_CASE)

### Import Aliases

Common aliases used in the codebase:
```go
import (
    webrtc "github.com/pion/webrtc/v3"
    log "github.com/sirupsen/logrus"
    homedir "github.com/mitchellh/go-homedir"
)
```

### Error Handling

**Pattern:** Use `cobra.CheckErr()` for fatal errors in CLI commands:
```go
gatherer, err := api.NewICEGatherer(iceOptions)
cobra.CheckErr(err)
```

**Alternative:** Use `logrus.Fatalln()` or `logrus.Panicln()` for critical errors:
```go
if keyphrase == "" {
    logrus.Fatalln("Keyphrase is empty!")
}
```

### Logging

**Library:** `github.com/sirupsen/logrus`

**Levels:**
- `log.Debugln()` - Verbose debug info (enabled with `-v` flag)
- `log.Infoln()` - Informational messages
- `log.Fatalln()` - Fatal errors (exits with code 1)
- `log.Panicln()` - Panic errors (exits with stack trace)

**Debug Mode:** Enabled via `--verbose` or `-v` flag:
```bash
./ht -v -f myfile.txt
```

### Configuration Management

**Library:** Viper

**Config File:** `$HOME/.hypertunnel.yaml` (optional)

**Override Order:**
1. Command-line flags (highest priority)
2. Environment variables (prefix with `HYPERTUNNEL_`)
3. Config file
4. Defaults (lowest priority)

**Custom Config:**
```bash
./ht --config /path/to/config.yaml
```

### License Headers

All Go source files include Apache 2.0 license headers:
```go
/*
Copyright © 2021 Anton Brekhov <anton@abrekhov.ru>

Licensed under the Apache License, Version 2.0 (the "License");
...
*/
```

**Note:** Some files have placeholder `NAME HERE <EMAIL ADDRESS>` - these should be updated if modified.

---

## Key Technical Details

### WebRTC Stack Layers

1. **ICE (Interactive Connectivity Establishment)**
   - NAT traversal using STUN/TURN
   - Discovers multiple candidate addresses
   - Negotiates best path between peers

2. **DTLS (Datagram Transport Layer Security)**
   - Encryption layer over UDP
   - Provides confidentiality and integrity

3. **SCTP (Stream Control Transmission Protocol)**
   - Reliable, ordered message delivery
   - Supports data channels

4. **Data Channels**
   - Application-level communication
   - Used for file transfer in HyperTunnel

### File Transfer Protocol

**Sender Mode (`ht -f myfile.txt`):**
1. Create WebRTC connection as "controlling" role
2. Open data channel with filename as label
3. Read file in 65534-byte chunks
4. Send chunks via `channel.Send(chunk)`
5. Wait 30 seconds after EOF, then exit

**Receiver Mode (`ht`):**
1. Create WebRTC connection as "controlled" role
2. Wait for incoming data channel
3. Create file with channel label as filename
4. Write received chunks to file
5. Close file when data channel closes

**Chunk Size:** 65534 bytes (WebRTC data channel max payload - 2 bytes overhead)

### Signal Exchange Protocol

**Format:** Compact base64-encoded HTCP (binary) with legacy JSON fallback

**Structure:**
```json
{
  "ICECandidates": [...],
  "ICEParameters": {...},
  "DTLSParameters": {...},
  "SCTPCapabilities": {...}
}
```

**User Workflow:**
1. First user runs `ht -f file.txt` or `ht`
2. Copy printed base64 signal
3. Second user runs complementary command
4. Paste first user's signal when prompted
5. Second user's signal is printed
6. First user pastes second user's signal
7. Connection established, file transfers

**Security Note:** Manual copy-paste prevents man-in-the-middle attacks if exchange happens over secure channel (e.g., encrypted messaging).

### Encryption Details

**Algorithm:** AES-256-CTR (Counter Mode)

**Why CTR Mode:**
- Stream cipher (no padding required)
- Parallelizable encryption/decryption
- Efficient for large files
- Random access to encrypted data

**Key Derivation:**
```
User Keyphrase → SHA256 Hash → 32-byte AES Key
```

**IV (Initialization Vector):**
- 16 bytes (AES block size)
- Randomly generated using `crypto/rand`
- Appended to encrypted file (last 16 bytes)

**Security Considerations:**
- Same keyphrase always produces same key (no salt)
- IV must be unique per encryption (guaranteed by random generation)
- Keyphrase strength determines security (user responsibility)

---

## Common Tasks for AI Assistants

### 1. Understanding User Intent

**File Transfer Questions:**
- "How do I send a file?" → Explain sender mode: `ht -f <file>`
- "How do I receive a file?" → Explain receiver mode: `ht`
- "File transfer failed" → Check firewall, NAT type, STUN connectivity

**Encryption Questions:**
- "How do I encrypt a file?" → `ht encrypt -k "keyphrase" <file>`
- "How do I decrypt a file?" → `ht decrypt -k "keyphrase" <file.enc>`
- "What encryption is used?" → AES-256-CTR

### 2. Code Navigation

**Finding Functions:**
- Connection logic: `cmd/root.go:98-246` (Connection function)
- Encryption: `cmd/encrypt.go:57-109` (EncryptFile function)
- Decryption: `cmd/decrypt.go` (DecryptFile function)
- Signal encoding: `pkg/datachannel/datachannel.go:10-18` (Encode function)
- File transfer handler: `pkg/datachannel/handlers.go:15-41` (FileTransferHandler)

**Key Variables:**
- `isOffer` - Determines sender (true) vs. receiver (false)
- `file` - Filename to transfer (from `-f` flag)
- `keyphrase` - Encryption/decryption key (from `-k` flag)
- `verbose` - Debug logging enabled (from `-v` flag)

### 3. Implementing Features

**Example: Add progress bar for file transfer**

Location: `cmd/root.go`, inside the data channel OnOpen handler (line 216-236)

Steps:
1. Add dependency: `github.com/schollz/progressbar/v3`
2. Get file size: Already available in `info.Size()`
3. Create progress bar before file read loop
4. Update progress bar after each `channel.Send()`

**Example: Add file compression**

Location: New package `pkg/compress/` or integrate into `cmd/root.go`

Steps:
1. Add compression before encryption (sender)
2. Add decompression after decryption (receiver)
3. Update data channel label to indicate compression
4. Consider using `compress/gzip` or `compress/zlib`

### 4. Debugging Issues

**Common Issues:**

1. **Connection hangs during signal exchange**
   - Check: `MustReadStdin()` expects double newline to terminate
   - Fix: Ensure user presses Enter twice after pasting signal

2. **File not received completely**
   - Check: Sender waits 30 seconds before closing (line 225)
   - Check: Data channel buffer size (65534 bytes)
   - Debug: Enable verbose logging with `-v`

3. **Encryption/decryption fails**
   - Check: Keyphrase matches between encrypt/decrypt
   - Check: File has `.enc` extension (convention)
   - Debug: Verify IV is correctly appended/extracted

4. **Build fails**
   - Check: Go version (requires 1.23+)
   - Run: `go mod tidy` to sync dependencies
   - Check: CGO is not required (disabled in GoReleaser)

### 5. Adding Documentation

**User Documentation:** Update `README.md`
- Keep installation instructions concise
- Provide usage examples
- Update roadmap as features are completed

**Code Documentation:** Add comments for exported functions
```go
// EncryptFile encrypts the specified file using AES-256-CTR.
// The encrypted output is written to <filename>.enc with the IV appended.
func EncryptFile(cmd *cobra.Command, args []string) {
    ...
}
```

**This File:** Update `CLAUDE.md` when:
- Architecture changes significantly
- New major features are added
- Conventions or workflows change

---

## Project Vision & Roadmap

### Vision Statement

Transform HyperTunnel into the **easiest, most robust, and beautiful P2P file/directory transfer tool** for any hosts behind NAT, without requiring public IPs. Make it universally accessible through standard package managers (apt, brew, etc.) with a delightful terminal user interface.

### Design Principles

1. **Simplicity First** - Zero configuration, works out of the box
2. **Beautiful UX** - Terminal interface should be a joy to use
3. **Robust & Reliable** - Handle edge cases, resume transfers, verify integrity
4. **Universal Access** - One-command install on any platform
5. **Secure by Default** - End-to-end encryption, no trust in relays

---

## Development Roadmap

### ✅ Phase 0: Foundation (COMPLETED)

**Status:** Done ✓

- [x] AES-256 file encryption/decryption
- [x] WebRTC P2P connection with NAT traversal
- [x] Single file transfer between peers
- [x] Manual signal exchange
- [x] Cross-platform builds via GoReleaser
- [x] Basic CLI with Cobra framework

---

### ✅ Phase 1: Core Functionality & Robustness (MOSTLY COMPLETE)

**Goal:** Make HyperTunnel production-ready for reliable file/directory transfers

#### 1.1 Directory Transfer Support ✅ DONE
- [x] **Recursive directory traversal**
  - Walk directory tree, preserve structure
  - Implemented in `pkg/archive/archive.go`

- [x] **Archive-based transfer**
  - Stream tar.gz on-the-fly
  - Compress during transfer for efficiency
  - Extract on receiver side automatically

- [x] **Metadata preservation**
  - File permissions preserved in tar archive
  - Timestamps preserved

**Implemented in:**
- `cmd/root.go` - Directory detection logic
- `pkg/archive/` - On-the-fly tar.gz streaming
- `pkg/datachannel/handlers.go` - Archive extraction

#### 1.2 Enhanced File Transfer Protocol (PARTIAL)
- [x] **Progress reporting** ✅
  - Track bytes sent/received
  - `pkg/transfer/progress.go` - Progress tracking

- [ ] **Resumable transfers** (FUTURE)
  - Generate transfer ID (hash of filename + size)
  - Track partial progress in temp file
  - Resume from last checkpoint on reconnection

- [x] **Integrity verification** ✅
  - Calculate SHA-256 checksum
  - `pkg/transfer/checksum.go` - Checksum verification

- [ ] **Automatic retry & recovery** (FUTURE)
  - Detect disconnections
  - Auto-reconnect with exponential backoff

#### 1.3 Connection Improvements (IN PROGRESS)
- [ ] **Symmetric connection startup**
  - Allow starting in any order (remove offer/answer dependency)
  - Both peers act as ICE controllers initially

- [ ] **Multiple STUN/TURN servers** (FUTURE)
  - Default list: Google, Mozilla, Cloudflare STUN
  - Support custom STUN/TURN via config

- [ ] **Connection quality monitoring** (FUTURE)
  - Track RTT (round-trip time)
  - Monitor packet loss

#### 1.4 Security Enhancements ✅ DONE
- [x] **Add auto-accept flag** ✅
  - `--auto-accept` flag implemented
  - Skip confirmation prompts for automation

- [ ] **Improved key derivation** (FUTURE)
  - Replace SHA256 with Argon2id KDF

- [x] **File overwrite protection** ✅
  - Prompt user before overwriting existing files
  - Works with `--auto-accept` for automation

#### 1.5 Testing & Quality ✅ DONE
- [x] **Unit tests**
  - `pkg/datachannel/datachannel_test.go`
  - `pkg/datachannel/handlers_test.go`
  - `pkg/hashutils/hashutils_test.go`
  - `pkg/archive/archive_test.go`
  - `pkg/transfer/*_test.go`

- [x] **Integration tests**
  - `integration_test.go` - End-to-end scenarios

- [ ] **Performance benchmarks** (FUTURE)
  - Benchmark encryption speed
  - Benchmark transfer throughput

- [x] **CI/CD testing pipeline** ✅
  - GitHub Actions workflow for tests

**Files created:
- `pkg/datachannel/datachannel_test.go` - Complete tests
- `pkg/transfer/transfer_test.go` - New tests
- `integration_test.go` - End-to-end scenarios
- `.github/workflows/test.yaml` - CI pipeline

**Estimated Effort:** 3-4 weeks (with testing)

---

### 🎨 Phase 2: Beautiful Terminal User Interface (IN PROGRESS)

**Goal:** Transform HyperTunnel into a delightful TUI experience using best-in-class Go libraries

**Status:** Basic TUI framework implemented (`pkg/tui/`)

---

#### ⚠️ CRITICAL DESIGN REQUIREMENT: Signal Copyability

**IMPORTANT:** Users commonly use HyperTunnel from remote VMs (SSH sessions) to transfer files P2P. The connection signal (base64 encoded string) MUST remain easily copyable.

**DO NOT:**
- Wrap signals in ASCII box borders (`╭──╮`, `│ │`, `╰──╯`)
- Put signals inside styled boxes that break copy-paste
- Add decorative characters around the signal

**DO:**
- Print the signal as a plain, undecorated string
- Add a clear label above it (e.g., "Your connection signal:")
- Use blank lines to separate it from other output
- Keep the signal on its own lines without surrounding characters

**Example - GOOD:**
```
Your connection signal:

eyJJQ0VDYW5kaWRhdGVzIjpbeyJGb3VuZGF0aW9uIjoiIiwiUHJpb3Jpd...

Paste the above signal to the other peer.
```

**Example - BAD (don't do this):**
```
┌────────────────────────────────────────────────┐
│ eyJJQ0VDYW5kaWRhdGVzIjpbeyJGb3VuZGF0aW9uIjoi │
│ IiwiUHJpb3JpdHkiOjIxMzAzNzY3...               │
└────────────────────────────────────────────────┘
```

**Why:** ASCII box characters become part of the selection when users copy from terminals, especially over SSH. This breaks the base64 decoding on the receiving end.

---

#### 2.1 TUI Library Selection

**Recommended Stack:**

1. **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** - TUI framework
   - Elm-inspired architecture (model-update-view)
   - Handles input, rendering, and state elegantly
   - Active development, excellent documentation
   - Used by: Glow, Soft Serve, VHS

2. **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** - Styling & layout
   - CSS-like styling for terminal output
   - Colors, borders, padding, margins
   - Responsive layouts
   - Pairs perfectly with Bubble Tea

3. **[Bubbles](https://github.com/charmbracelet/bubbles)** - Pre-built components
   - Progress bars (for file transfer)
   - Spinners (for connection setup)
   - Text inputs (for keyphrases)
   - Viewports (for long logs)
   - Lists (for file selection)

4. **[Glamour](https://github.com/charmbracelet/glamour)** - Markdown rendering
   - Render help text beautifully
   - Styled error messages
   - Rich documentation in-app

**Add dependencies:**
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/glamour@latest
```

#### 2.2 UI Components to Build

**2.2.1 Connection Screen**
```
╭─────────────────────────────────────────────╮
│  🚀 HyperTunnel - P2P File Transfer        │
│                                             │
│  Mode: Sender                               │
│  File: large-video.mp4 (1.2 GB)            │
│                                             │
│  ⏳ Establishing connection...             │
│  ◆ Gathering ICE candidates  [✓]           │
│  ◆ Waiting for peer signal   [⣾]           │
│                                             │
│  📋 Your connection code:                  │
│  ┌───────────────────────────────────────┐ │
│  │ eyJJQ0VDYW5kaWRhdGVzIjpbeyJGb3VuZGF0│ │
│  │ aW9uIjoiIiwiUHJpb3JpdHkiOjIxMzAzNzY3│ │
│  │ ...                                   │ │
│  └───────────────────────────────────────┘ │
│                                             │
│  [Copied to clipboard!]                    │
│                                             │
│  Press Ctrl+C to cancel                    │
╰─────────────────────────────────────────────╯
```

**2.2.2 Transfer Screen**
```
╭─────────────────────────────────────────────╮
│  📦 Transferring: large-video.mp4          │
│                                             │
│  Progress:  ████████████░░░░░░░  67%       │
│            812 MB / 1.2 GB                  │
│                                             │
│  Speed:     15.3 MB/s                       │
│  Time:      00:03:42 elapsed                │
│  ETA:       00:01:45 remaining              │
│                                             │
│  Connection: Direct P2P (low latency)       │
│  Encrypted: ✓ AES-256-CTR                  │
│  Verified:  ✓ Checksums match               │
│                                             │
│  Peer: 192.168.1.15:54321 (relay)          │
│  RTT: 23ms  |  Loss: 0.1%                   │
╰─────────────────────────────────────────────╯
```

**2.2.3 Directory Transfer Screen**
```
╭─────────────────────────────────────────────╮
│  📁 Transferring directory: my-project/    │
│                                             │
│  Files:     47 / 156  (30%)                 │
│  Size:      523 MB / 1.8 GB  (29%)          │
│                                             │
│  Current:   src/assets/logo.svg             │
│  Progress:  ██████████░░░░░░░░  52%         │
│                                             │
│  ✓ README.md                                │
│  ✓ package.json                             │
│  ✓ src/index.js                             │
│  ⣾ src/assets/logo.svg                      │
│  ○ src/components/App.jsx                   │
│  ○ ...                                      │
│                                             │
│  Speed: 18.2 MB/s  |  ETA: 00:02:15        │
╰─────────────────────────────────────────────╯
```

**2.2.4 Completion Screen**
```
╭─────────────────────────────────────────────╮
│                                             │
│          ✨ Transfer Complete! ✨           │
│                                             │
│  📦 large-video.mp4                         │
│  📊 1.2 GB transferred in 4m 23s            │
│  ⚡ Average speed: 14.8 MB/s                │
│                                             │
│  ✓ Integrity verified (SHA-256)            │
│  ✓ File saved to: ~/Downloads/             │
│                                             │
│  [ Press Enter to exit ]                   │
│                                             │
╰─────────────────────────────────────────────╯
```

#### 2.3 Interactive Features

- [ ] **File browser** (sender mode)
  - Navigate filesystem with arrow keys
  - Multi-select files/directories
  - Preview file size before sending
  - Filter by pattern (*.pdf, *.jpg, etc.)

- [ ] **QR code generation** (optional)
  - Generate QR code from signal for easy mobile exchange
  - Use `github.com/skip2/go-qrcode`
  - Display in terminal using Unicode blocks

- [ ] **Live logs viewer**
  - Scrollable debug log in bottom pane
  - Toggle visibility with 'd' key
  - Color-coded by severity (info/warn/error)

- [ ] **Keyboard shortcuts**
  - `?` - Show help overlay
  - `c` - Copy signal to clipboard
  - `p` - Paste peer signal
  - `d` - Toggle debug logs
  - `q` or `Ctrl+C` - Quit gracefully

- [ ] **Responsive layout**
  - Adapt to terminal size
  - Minimum 80x24, graceful degradation
  - Hide/collapse sections on small terminals

#### 2.4 Implementation Strategy

**New package structure:**
```
pkg/
├── tui/
│   ├── model.go          # Bubble Tea model (state)
│   ├── update.go         # Update logic (handle events)
│   ├── view.go           # Rendering (UI output)
│   ├── components/       # Reusable UI components
│   │   ├── progress.go   # Custom progress bar
│   │   ├── connection.go # Connection status widget
│   │   ├── filelist.go   # File browser
│   │   └── statusbar.go  # Bottom status bar
│   └── styles.go         # Lip Gloss styles
```

**Migration path:**
1. Keep existing CLI (`cmd/root.go`) as fallback
2. Add `--tui` flag to enable new interface
3. Default to TUI if terminal is interactive
4. Detect non-TTY (pipes, scripts) and use plain mode

**Files to modify:**
- `cmd/root.go` - Add TUI mode toggle
- `pkg/tui/` (new package) - All TUI logic
- `go.mod` - Add Charm dependencies

**Estimated Effort:** 2-3 weeks

---

### 📦 Phase 3: Distribution & Packaging (PARTIAL)

**Goal:** Make installation effortless - one command on any platform

**Status:** Multi-platform packaging with DEB/RPM/APK completed via GoReleaser.

**Completed:**
- [x] DEB packages for Debian/Ubuntu
- [x] RPM packages for Fedora/RHEL/CentOS
- [x] APK packages for Alpine Linux
- [x] Tar.gz archives for macOS/Linux
- [x] ZIP archives for Windows
- [x] GoReleaser configuration in `.goreleaser.yaml`

**Remaining:**
- [ ] Docker images (GHCR)
- [ ] Homebrew formula
- [ ] Scoop/Chocolatey for Windows
- [ ] AUR package for Arch Linux

#### 3.1 GitHub Container Registry (GHCR) (TODO)

**Purpose:** Docker images for containerized usage

**Setup:**
```dockerfile
# Dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -ldflags="-s -w" -o ht

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /build/ht /usr/local/bin/
ENTRYPOINT ["ht"]
```

**GitHub Actions workflow:**
```yaml
# .github/workflows/docker.yaml
name: Docker Build & Push

on:
  push:
    tags: ['v*.*.*']
  workflow_dispatch:

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4

      - name: Login to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            ghcr.io/abrekhov/hypertunnel:latest
            ghcr.io/abrekhov/hypertunnel:${{ github.ref_name }}
```

**Usage:**
```bash
docker pull ghcr.io/abrekhov/hypertunnel:latest
docker run -it ghcr.io/abrekhov/hypertunnel -f myfile.txt
```

**Tasks:**
- [ ] Create Dockerfile (multi-stage build)
- [ ] Add `.github/workflows/docker.yaml`
- [ ] Configure GHCR permissions in repo settings
- [ ] Test image locally before pushing
- [ ] Add Docker usage to README

#### 3.2 Debian/Ubuntu APT Repository

**Purpose:** `sudo apt install hypertunnel`

**Setup using GitHub Releases + PPA:**

**Option A: Using GoReleaser with nFPM**

Update `.goreleaser.yaml`:
```yaml
nfpms:
  - id: hypertunnel
    package_name: hypertunnel
    homepage: https://github.com/abrekhov/hypertunnel
    maintainer: Anton Brekhov <anton@abrekhov.ru>
    description: P2P file transfer tool using WebRTC
    license: Apache 2.0
    formats:
      - deb
      - rpm
    bindir: /usr/local/bin
    contents:
      - src: ./ht
        dst: /usr/local/bin/ht
        file_info:
          mode: 0755
```

**Option B: Launchpad PPA (official Ubuntu)**

1. Create Launchpad account
2. Generate GPG key for signing
3. Create debian packaging files:
```
debian/
├── changelog
├── control
├── copyright
├── rules
└── compat
```

**GitHub Actions automation:**
```yaml
# .github/workflows/deb.yaml
name: Build DEB Package

on:
  push:
    tags: ['v*.*.*']

jobs:
  deb:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5

      - name: Install nFPM
        run: |
          curl -sfL https://install.goreleaser.com/github.com/goreleaser/nfpm.sh | sh

      - name: Build DEB
        run: nfpm package --packager deb

      - name: Upload to Release
        uses: softprops/action-gh-release@v1
        with:
          files: '*.deb'
```

**Installation Instructions (for README):**
```bash
# Download latest .deb from releases
wget https://github.com/abrekhov/hypertunnel/releases/download/v1.0.0/hypertunnel_1.0.0_amd64.deb

# Install
sudo dpkg -i hypertunnel_1.0.0_amd64.deb

# Or using APT (if PPA is set up)
sudo add-apt-repository ppa:abrekhov/hypertunnel
sudo apt update
sudo apt install hypertunnel
```

**Tasks:**
- [ ] Add nFPM configuration to `.goreleaser.yaml`
- [ ] Create debian packaging files
- [ ] Set up Launchpad PPA (optional, for official repos)
- [ ] Test DEB installation on Ubuntu 22.04, 24.04
- [ ] Add apt installation to README
- [ ] Sign packages with GPG

#### 3.3 Homebrew (macOS & Linux)

**Purpose:** `brew install hypertunnel`

**Create Homebrew formula:**

1. Fork https://github.com/Homebrew/homebrew-core
2. Create formula file: `Formula/h/hypertunnel.rb`

```ruby
class Hypertunnel < Formula
  desc "P2P file transfer tool using WebRTC for NAT traversal"
  homepage "https://github.com/abrekhov/hypertunnel"
  url "https://github.com/abrekhov/hypertunnel/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "abc123..."
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./main.go"
  end

  test do
    assert_match "hypertunnel version", shell_output("#{bin}/ht --version")
  end
end
```

**Or use Homebrew Tap (faster, no review):**

1. Create repo: `homebrew-tap`
2. Add formula to `Formula/hypertunnel.rb`
3. Users install with: `brew tap abrekhov/tap && brew install hypertunnel`

**GoReleaser automation:**

Update `.goreleaser.yaml`:
```yaml
brews:
  - name: hypertunnel
    repository:
      owner: abrekhov
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    folder: Formula
    homepage: https://github.com/abrekhov/hypertunnel
    description: P2P file transfer tool using WebRTC
    license: Apache-2.0
    install: |
      bin.install "ht" => "hypertunnel"
    test: |
      system "#{bin}/hypertunnel", "--version"
```

**Tasks:**
- [ ] Create `abrekhov/homebrew-tap` repository
- [ ] Add brew configuration to `.goreleaser.yaml`
- [ ] Set up HOMEBREW_TAP_TOKEN secret in GitHub
- [ ] Test formula with `brew install --build-from-source`
- [ ] Submit to homebrew-core (optional, after stable release)
- [ ] Add brew installation to README

#### 3.4 Arch Linux AUR

**Purpose:** `yay -S hypertunnel` or `paru -S hypertunnel`

**Create PKGBUILD:**
```bash
# PKGBUILD
pkgname=hypertunnel
pkgver=1.0.0
pkgrel=1
pkgdesc="P2P file transfer tool using WebRTC"
arch=('x86_64' 'aarch64')
url="https://github.com/abrekhov/hypertunnel"
license=('Apache')
depends=()
makedepends=('go')
source=("$pkgname-$pkgver.tar.gz::$url/archive/v$pkgver.tar.gz")
sha256sums=('SKIP')

build() {
  cd "$pkgname-$pkgver"
  export CGO_ENABLED=0
  go build -ldflags="-s -w" -o ht
}

package() {
  cd "$pkgname-$pkgver"
  install -Dm755 ht "$pkgdir/usr/bin/hypertunnel"
}
```

**Publish to AUR:**
1. Create AUR account
2. Clone AUR repo: `git clone ssh://aur@aur.archlinux.org/hypertunnel.git`
3. Add PKGBUILD and .SRCINFO
4. Push to AUR

**Tasks:**
- [ ] Create PKGBUILD file
- [ ] Test on Arch Linux VM
- [ ] Publish to AUR
- [ ] Add AUR installation to README

#### 3.5 Windows Package Managers

**Scoop:**
```json
{
  "version": "1.0.0",
  "description": "P2P file transfer tool using WebRTC",
  "homepage": "https://github.com/abrekhov/hypertunnel",
  "license": "Apache-2.0",
  "architecture": {
    "64bit": {
      "url": "https://github.com/abrekhov/hypertunnel/releases/download/v1.0.0/ht_windows_amd64.exe",
      "bin": "ht_windows_amd64.exe",
      "hash": "sha256:..."
    }
  }
}
```

**Chocolatey:**
```xml
<?xml version="1.0" encoding="utf-8"?>
<package>
  <metadata>
    <id>hypertunnel</id>
    <version>1.0.0</version>
    <title>HyperTunnel</title>
    <authors>Anton Brekhov</authors>
    <licenseUrl>https://github.com/abrekhov/hypertunnel/blob/main/LICENSE</licenseUrl>
    <projectUrl>https://github.com/abrekhov/hypertunnel</projectUrl>
    <description>P2P file transfer tool using WebRTC</description>
  </metadata>
</package>
```

**Tasks:**
- [ ] Create Scoop manifest
- [ ] Submit to scoop-extras bucket
- [ ] Create Chocolatey package
- [ ] Submit to Chocolatey community feed
- [ ] Add Windows installation to README

#### 3.6 Documentation & Marketing

**Comprehensive README:**
- [ ] Add badges (Go version, license, downloads, CI status)
- [ ] GIF demos of TUI in action (use `vhs` by Charm)
- [ ] Installation matrix (all platforms)
- [ ] Quick start guide
- [ ] Advanced usage examples
- [ ] FAQ section
- [ ] Comparison with alternatives (magic-wormhole, croc, etc.)

**GitHub Repository polish:**
- [ ] Add topics/tags for discoverability
- [ ] Create issue templates
- [ ] Add pull request template
- [ ] Contributing guidelines
- [ ] Code of conduct
- [ ] Security policy (for responsible disclosure)

**Estimated Effort:** 2-3 weeks (packaging + testing)

---

### 🚀 Phase 4: Production Ready & Beyond

**Goal:** Polish, optimize, and expand use cases

#### 4.1 Advanced Features

- [ ] **SSH tunnel mode**
  - Forward SSH connections through WebRTC
  - Access machines behind NAT without port forwarding
  - Similar to ngrok but P2P

- [ ] **Port forwarding**
  - Generic TCP tunnel over WebRTC
  - Forward any service (HTTP, database, etc.)
  - Multiple concurrent tunnels

- [ ] **Group transfers**
  - One sender, multiple receivers
  - Broadcast file to many peers
  - Uses WebRTC multicast

- [ ] **Web UI (optional)**
  - Electron or Wails wrapper
  - Drag-and-drop file selection
  - QR code scanning for mobile
  - System tray integration

#### 4.2 Performance & Optimization

- [ ] **Parallel chunk streaming**
  - Send multiple chunks concurrently
  - Utilize full bandwidth
  - Out-of-order reassembly

- [ ] **Adaptive compression**
  - Detect file type (text vs binary vs pre-compressed)
  - Apply compression only when beneficial
  - Support zstd, brotli, gzip

- [ ] **Connection pooling**
  - Reuse connection for multiple files
  - Batch small files into single archive
  - Reduce handshake overhead

- [ ] **Memory optimization**
  - Stream large files without loading into RAM
  - Bounded buffer pools
  - Profile and optimize allocations

#### 4.3 Ecosystem Integration

- [ ] **Shell completions**
  - Bash, Zsh, Fish
  - Auto-generated with Cobra

- [ ] **Man pages**
  - Generated from CLI help
  - Installed with packages

- [ ] **systemd service** (optional)
  - Run as daemon for always-on receive mode
  - Socket activation

- [ ] **Plugins/Extensions**
  - Webhook on transfer complete
  - Custom encryption backends
  - Alternative signal exchanges (QR, NFC, etc.)

#### 4.4 Internationalization (i18n)

- [ ] Multi-language support
  - English (default)
  - Spanish, French, German, Chinese, Russian
  - Use `github.com/nicksnyder/go-i18n`

**Estimated Effort:** Ongoing (prioritize based on user feedback)

---

## Implementation Priority

### Completed ✅

1. ~~**Harden incoming file handling**~~ ✅
   - Overwrite prompt/flag implemented
   - Auto-accept flag (`--auto-accept`) implemented

2. ~~**Add basic testing**~~ ✅
   - Unit tests for all packages
   - Integration tests (`integration_test.go`)

3. ~~**Directory transfer**~~ ✅
   - Archive-based transfer with tar.gz
   - Automatic extraction on receiver

4. ~~**Progress tracking and checksums**~~ ✅
   - `pkg/transfer/progress.go`
   - `pkg/transfer/checksum.go`

5. ~~**Basic TUI**~~ ✅
   - Bubble Tea framework
   - `pkg/tui/` package

6. ~~**Packaging basics**~~ ✅
   - DEB/RPM/APK via GoReleaser

### Current Priority (Next Steps)

1. **Fix TUI signal display** (CRITICAL)
   - Remove ASCII box borders around signals
   - Ensure signals are easily copyable from SSH terminals
   - See "Signal Copyability" section in Phase 2

2. **Homebrew formula**
   - Publish to homebrew-tap

3. **Signal encoding polish**
   - Verify compact signal format copy/paste behavior
   - Document format and fallback behavior

---

## Success Metrics

### Current Status (v0.x)

- ✅ Robust file + directory transfer
- ✅ Basic TUI framework
- ✅ Multi-platform packaging (DEB/RPM/APK)
- ✅ Unit and integration tests
- ✅ Auto-accept and overwrite protection
- ✅ Compact signal encoding
- ⏳ TUI polish (signals must remain copyable)


### v1.0 Release Criteria

- ✅ Robust file + directory transfer
- ⏳ TUI with copyable signals (not in ASCII boxes!)
- ✅ Multi-platform packages
- ⏳ Homebrew/Scoop
- ✅ Zero critical security issues
- ⏳ Symmetric connection (start in any order)

---

## Getting Started

**For AI Assistants:**

When implementing features from this roadmap:

1. **Check phase dependencies** - Implement in order when possible
2. **Update this document** - Mark items complete, adjust estimates
3. **Write tests first** - TDD for new features
4. **Keep it simple** - Don't over-engineer
5. **Document as you go** - Update README and code comments
6. **Use TodoWrite tool** - Track progress on multi-step features
7. **CRITICAL: Keep signals copyable** - Never wrap in ASCII boxes!

**Next immediate steps:**
1. Fix TUI to not wrap signals in ASCII borders (`pkg/tui/connection.go`)
2. Create Homebrew formula
3. Verify compact signal format behavior

---

## Security Considerations

### Current Security Features

1. **Transport Encryption:** DTLS (built into WebRTC)
2. **File Encryption:** AES-256-CTR (optional, via `encrypt` command)
3. **Manual Signal Exchange:** Prevents automated MITM attacks
4. **No Central Server:** Reduces attack surface

### Potential Vulnerabilities

1. **Signal Exchange Integrity:** No authentication of signals
   - Mitigation: Users must exchange signals over trusted channel

2. **No File Integrity Verification:** No checksums or signatures
   - Impact: Corruption undetected, no tamper-evidence

3. **Weak Keyphrase Security:** Direct SHA256 hash, no salt or KDF
   - Impact: Dictionary attacks on weak passphrases
   - Improvement: Use PBKDF2, bcrypt, or Argon2

4. **File Overwrite Risk:** Receiver doesn't check for existing files correctly
   - Line 18-21 in handlers.go: Uses `os.IsExist(err)` but `os.Stat()` returns `nil` error if file doesn't exist
   - Correct check: `!os.IsNotExist(err)` or `err == nil`

### Recommendations for AI Assistants

**When implementing security features:**
- Always use cryptographically secure random sources (`crypto/rand`)
- Prefer established libraries over custom crypto
- Document security assumptions clearly
- Add input validation at system boundaries
- Consider timing attacks for sensitive operations

**When reviewing security issues:**
- Never disable security features without explicit user request
- Warn users about security implications of changes
- Follow principle of least privilege
- Default to secure configurations

---

## Performance Considerations

### Current Performance Characteristics

**Chunk Size:** 65534 bytes
- Optimal for WebRTC data channels
- Balances latency and throughput
- Hardcoded in `cmd/root.go:220`

**Encryption Buffer:** 1024 bytes (default, configurable)
- Smaller than transfer chunk size
- Allows fine-grained progress reporting
- Configurable via `-b` flag

**Network Overhead:**
- STUN/TURN discovery: ~1-5 seconds
- Signal exchange: Manual (user-dependent)
- Connection establishment: ~1-3 seconds
- Transfer: Limited by bandwidth and WebRTC stack

### Optimization Opportunities

1. **Parallel Chunk Processing**
   - Current: Sequential read-send loop
   - Improvement: Pipeline reads while sending

2. **Dynamic Buffer Sizing**
   - Current: Fixed 65534 bytes
   - Improvement: Adjust based on RTT and bandwidth

3. **Compression**
   - Not implemented
   - Potential: Add `compress/gzip` for compressible files

4. **Connection Reuse**
   - Current: One file per connection
   - Improvement: Keep connection alive for multiple files

---

## Git Configuration

### Gitignore Patterns (`.gitignore`)

```
wip             # Work-in-progress files/directories
build           # Build artifacts directory
Makefile        # Build scripts (not used currently)
ht*             # Binary files (ht, ht_*, ht.exe)
*txt            # Text files (likely logs/checksums)
```

**Note:** `*txt` pattern is broad and may exclude legitimate documentation. Consider narrowing to `*.log` or specific files.

### Branch Strategy

**Main Branch:** `main` (implied, not explicitly configured)

**Recommended Workflow:**
- Feature branches: `feature/feature-name`
- Bug fixes: `fix/bug-description`
- Releases: Version tags `v*.*.*`

**Protected Branches:** Not configured (recommended to protect `main`)

---

## Additional Resources

### Relevant Documentation

- **Pion WebRTC:** https://github.com/pion/webrtc
- **Cobra CLI:** https://github.com/spf13/cobra
- **Viper Config:** https://github.com/spf13/viper
- **GoReleaser:** https://goreleaser.com/
- **WebRTC Specification:** https://www.w3.org/TR/webrtc/

### Similar Projects (for inspiration)

- **magic-wormhole:** Python P2P file transfer (mentioned in README)
- **gfile:** Go file transfer tool (mentioned in README)
- **croc:** Secure file transfer with relay
- **teleport:** WebRTC-based file sharing

---

## Working with This Codebase - Quick Reference

### Build and Run
```bash
# Build
go build -o ht

# Run sender
./ht -f myfile.txt

# Run receiver
./ht

# Encrypt file
./ht encrypt -k "my secret" myfile.txt

# Decrypt file
./ht decrypt -k "my secret" myfile.txt.enc

# Debug mode
./ht -v -f myfile.txt
```

### Development
```bash
# Install dependencies
go mod download

# Format code
go fmt ./...

# Run linting
golangci-lint run ./...

# Run tests (when implemented)
go test ./...

# Build for all platforms
goreleaser build --snapshot --clean

# Clean build artifacts
rm -rf build ht ht_*
```

### File Locations Quick Reference
- Connection logic: `cmd/root.go`
- Encryption: `cmd/encrypt.go`
- Decryption: `cmd/decrypt.go`
- Signal encoding: `pkg/datachannel/datachannel.go`
- File transfer: `pkg/datachannel/handlers.go`
- Key hashing: `pkg/hashutils/hashutils.go`
- Dependencies: `go.mod`
- Release config: `.goreleaser.yaml`

---

## Conclusion

HyperTunnel is a focused, well-structured P2P file transfer tool with clear separation of concerns. The codebase is intentionally lean (~663 lines) and built on proven libraries (Pion WebRTC, Cobra CLI).

**Strengths:**
- Simple, understandable architecture
- Minimal dependencies
- Cross-platform support
- Secure by default (DTLS + optional AES-256)

**Areas for Improvement:**
- Test coverage
- File integrity verification
- File overwrite protection
- Progress reporting

**Philosophy:** Keep it simple. Don't over-engineer. Add features when clearly needed, not speculatively.

**Critical UX Note:** Signals MUST remain easily copyable from SSH terminals. Never wrap them in ASCII box borders.

---

**Document Version:** 1.1
**Last Updated:** 2026-01-16
**Maintainer:** AI-generated for AI assistants working on HyperTunnel
