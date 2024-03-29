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

```bash
#First machine
./ht -f <file>
#Second machine
./ht
#Cross insert SPDs
```

## RoadMap

- [X] Encrypt file with key as stream
- [X] Decrypt file with key as stream
- [X] TCP/IP Connection through stun/turn/ice
- [X] ORTC connection behind NAT
- [X] Move one file between candidates behind NAT
- [ ] Start candidates in any order
- [ ] Decompose and refactor
- [ ] Directory transfer
- [ ] Barline
- [ ] SSH server behind NAT
- [ ] Tests
- [ ] Benchs
