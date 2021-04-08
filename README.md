# HyperTunnel

## Installation

```bash
go build -o ht
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

```bash
./ht
```


## RoadMap

- [X] Encrypt file with key as stream
- [X] Decrypt file with key as stream
- [X] TCP/IP Connection through stun/turn/ice
- [X] ORTC connection behind NAT
- [X] Move one file between candidates behind NAT
- [ ] Decompose and refactor
- [ ] Directory transfer
- [ ] Barline
- [ ] Tests
- [ ] Benchs
