# HyperTunnel: reconciled roadmap

Draft based on main `d45c49468da30de3c094bada1cf793aa89bfd9c6` and a read-only GitHub review. No implementation or PR merge is implied by this plan.

## Product direction

A small CLI for transferring files and directories between two devices. Keep Pion WebRTC, trusted manual signal exchange and existing commands. Do not merge minimesh, add a VPN, or build distributed storage at this stage.

Manual mode needs no application signaling server, but the current connection setup uses an external STUN service. Direct connectivity is not guaranteed for every NAT.

## Verified baseline

| Area | Current state |
|---|---|
| File transfer | Implemented using ordered WebRTC data channels and DTLS |
| Directory transfer | Implemented, but archives are buffered in memory on both sides |
| Signal exchange | Compact binary/base64 with legacy JSON fallback; piped stdin supported |
| Signal display | Already printed without box borders; clipboard support exists |
| Interactive input | Current prompt says Ctrl+D; README still says Enter twice |
| Progress | CLI byte/speed reporting exists; receiver has no declared total size |
| Integrity | SHA-256 and metadata helpers exist but are not used by the active transfer path |
| Completion | Channel closure is treated as success; no receiver verification acknowledgement |
| Receive safety | Prompts exist, but the default timeout is 20 seconds and timeout/EOF can accept automatically |
| Destination handling | Remote channel labels become filesystem paths without active metadata validation |
| Archives | Filenames ending in .tar.gz are treated as directory transfers; symlink confinement needs hardening |
| NAT traversal | One Google STUN server configured; no configured TURN fallback |
| ICE roles | Symmetric startup merged: peers derive complementary roles from connection parameters and reject a pasted local signal |
| TUI | Models exist; CLI only renders a static welcome, not a running transfer UI |
| File encryption | Legacy AES-CTR with SHA-256-derived passphrase keys; no ciphertext authentication |
| Distribution | Release v0.0.12 includes binaries, archives and DEB/RPM/APK packages; Homebrew formula published |

Local build, CLI help, go vet and existing tests with race detection passed. The CLI transfer E2E test skips on macOS; no new real transfer was verified during this review. GitHub main test/release workflows report success. This is not a complete security audit or a fresh package-install verification.

## Next steps, in order

### 1. Reconcile documentation and development baseline

- Make this short roadmap the maintained status source and link it from README.
- Correct README claims about checksums, TURN, input termination and Homebrew status.
- Retain one canonical AGENTS.md and remove the stale case-colliding agents.md using a case-sensitive checkout; shorten duplicated AI guides.
- Align documented Go/linter versions with go.mod and CI.
- Review existing PRs before duplicating work; do not merge solely because old checks are green.
- Rework the CLI-process E2E harness to run on macOS and Linux, verify natural exit statuses and avoid hiding lifecycle failures with process kills/retries.

Acceptance: clean checkout on macOS and Linux, accurate documentation, and one actual sender/receiver transfer validated on both systems.

### 2. Safe, verified transfers

**Merged on main:** versioned metadata/ack protocol with SHA-256 + size verification, `.htpart` staging with publish-on-verified-rename, traversal-safe filenames from metadata, clear legacy-peer rejection, fail-closed EOF on the accept prompt, and CLI + in-process integration/E2E coverage (both startup orders, decline, directory, mismatch/truncation/oversize/empty).

**Remaining:**
- Confine destinations to an explicit receive directory; harden archive symlink escapes.
- Add size/resource bounds and data-channel backpressure. Replace directory-wide RAM buffering with disk-backed staging or bounded streaming; measure memory rather than assuming it is bounded.
- E2E coverage for overwrite refusal, malicious archive paths/symlinks, and a representative large transfer with measured memory.

Acceptance: CLI-process E2E cases cover empty/binary/Unicode files, directories, an ordinary .tar.gz file, malicious paths/symlinks, overwrite refusal, corrupted/truncated payloads and interrupted transfers. A representative large transfer completes with measured bounded memory. Add E2E tests only.

### 3. Simplify connection UX

- Keep the existing manual mode and undecorated copyable signals, including SSH and piped operation.
- Use consistent input instructions and clear gathering/connecting/transferring/verifying states. Keep plain CLI output as the default before investing in a full-screen TUI.
- Validate both startup orders with process E2E tests. Evaluate PR #33 only if role-independent negotiation is needed; do not log ICE credentials or assume this alone fixes start-order behavior.
- Optional, subject to a separate design decision: short pairing codes with an ephemeral signaling service. Authenticate exchanged signals using an established protocol, enforce expiry and attempt limits, and explain the new server dependency. Do not treat a code-to-signal lookup alone as secure pairing.

Acceptance: a documented two-device workflow works in either startup order, manual copy/paste remains usable, and all failures end with actionable messages rather than hanging. Symmetric startup and both startup orders are already covered by merged CLI E2E; remaining work is prompt/state polish.

### 4. Broader connectivity and production hardening

- Add configurable STUN/TURN endpoints and a relay-only test mode; distinguish direct and relayed connections. Do not promise universal NAT traversal.
- Replace optional legacy file encryption with a versioned authenticated format and a password KDF before promoting it as secure storage; decide compatibility explicitly.
- Add resumable transfers only after transfer identity, verification and partial-file handling are stable.
- Revalidate installer/Homebrew/packages and release only after required tests pass. Main currently auto-tags and triggers releases independently of test completion; gate that before publishing hardening changes.

Acceptance: restricted-NAT relay transfer is demonstrated, wrong passwords/tampered ciphertext fail safely, restart/resume verifies final content, and releases cannot publish failing builds.

## Open PR triage (resolved)

The PR backlog is cleared; further work continues directly on `main` without pull requests.

| PR | Outcome |
|---|---|
| #33 symmetric startup | Merged (`dfab28a`): complementary ICE roles without credential logging, own-signal rejection, CLI E2E for both startup orders on macOS/Linux, graceful shutdown fix, agents.md collision removed |
| #34 version command | Merged (`83bc358`): `ht version`, `--version`, `--long`, GoReleaser metadata stamping, CLI E2E for stamped output |
| #26 symmetric connection | Closed as superseded by merged #33 |
| #31 Codex E2E | Closed as superseded by merged CLI transfer E2E (#37, #33) |
| #32 local WebRTC E2E | Closed as superseded; its historical integration failures are covered by real CLI E2E |
| #36 Ubuntu PPA | Closed unimplemented: deferred pending Launchpad signing infrastructure; avoided Go/Pion downgrade |

Closed PR branches are retained on the remote for reference.

No open issues were returned by GitHub during this review. No PRs were merged or closed.

## Deferred scope

SSH tunneling, generic port forwarding, mesh discovery, QUIC transport, group transfer, file browsers, a web UI, additional package ecosystems and microbenchmark expansion. Prioritize dependable transfers and a simpler connection flow first.
