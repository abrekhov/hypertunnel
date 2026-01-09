# CLAUDE.md - HyperTunnel Codebase Guide for AI Assistants

This document provides a comprehensive overview of the HyperTunnel codebase for AI assistants working on this project.

## Project Overview

**HyperTunnel** is a peer-to-peer (P2P) secure file transfer tool written in Go that enables direct file transfers between two machines behind NAT without requiring a central server. It uses WebRTC technology for NAT traversal and DTLS for encryption.

**Key Features:**
- Direct P2P file transfer using WebRTC data channels
- NAT traversal via ICE/STUN/TURN protocols
- Built-in file encryption/decryption with AES-256-CTR
- Manual signal exchange (copy-paste) for security
- Cross-platform support (Linux, macOS, Windows)
- Minimal dependencies and simple CLI interface

**Project Stats:**
- Language: Go 1.23
- Total Go Files: 9
- Lines of Code: ~663
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
│   ├── root.go                   # Main command, connection logic (246 lines)
│   ├── encrypt.go                # File encryption command (109 lines)
│   └── decrypt.go                # File decryption command (114 lines)
├── pkg/                           # Internal reusable packages
│   ├── datachannel/              # WebRTC data channel utilities
│   │   ├── datachannel.go        # SDP encoding/decoding (49 lines)
│   │   ├── signal.go             # Signal struct for WebRTC handshake (18 lines)
│   │   ├── handlers.go           # File transfer handlers (64 lines)
│   │   └── datachannel_test.go   # Unit tests (skeleton only, 19 lines)
│   └── hashutils/                # Cryptographic utilities
│       └── hashutils.go          # Key hashing (22 lines)
├── main.go                        # Application entry point (22 lines)
├── go.mod                         # Go module dependencies
├── go.sum                         # Dependency checksums
├── .goreleaser.yaml              # GoReleaser configuration for builds
├── .gitignore                     # Git ignore patterns
├── LICENSE                        # Apache License 2.0
├── README.md                      # User-facing documentation
└── CLAUDE.md                      # This file
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
- `askForConfirmation(s string, in io.Reader) bool` - User confirmation (currently hardcoded to return `true`)

**IMPORTANT NOTE:** The confirmation function is currently bypassed (line 44 returns `true` immediately). The actual confirmation logic on lines 45-63 is unreachable.

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

**Format:** Base64-encoded JSON

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

## Roadmap and TODOs

From `README.md` and codebase analysis:

**Completed:**
- [x] AES-256 file encryption/decryption
- [x] WebRTC P2P connection with NAT traversal
- [x] Single file transfer between peers
- [x] Manual signal exchange
- [x] Cross-platform builds

**Pending (from README):**
- [ ] Start candidates in any order (currently order-dependent)
- [ ] Decompose and refactor code
- [ ] Directory transfer support
- [ ] Progress bar ("Barline")
- [ ] SSH server behind NAT
- [ ] Comprehensive tests
- [ ] Performance benchmarks

**Additional Improvements (not in roadmap):**
- [ ] Implement actual confirmation in `askForConfirmation()` (currently bypassed)
- [ ] Add integration tests for end-to-end file transfer
- [ ] Support resumable transfers
- [ ] Add metadata verification (checksums)
- [ ] Implement automatic TURN relay selection
- [ ] Add configuration for STUN/TURN servers
- [ ] Improve error messages for user clarity

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

4. **Confirmation Bypassed:** `askForConfirmation()` returns true (line 44 in handlers.go)
   - Impact: Files auto-accepted without user consent
   - Fix: Remove early return, enable actual user prompt

5. **File Overwrite Risk:** Receiver doesn't check for existing files correctly
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
- User confirmation implementation
- Progress reporting

**Philosophy:** Keep it simple. Don't over-engineer. Add features when clearly needed, not speculatively.

---

**Document Version:** 1.0
**Last Updated:** 2026-01-09
**Maintainer:** AI-generated for AI assistants working on HyperTunnel
