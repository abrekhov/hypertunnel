//go:build integration

package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestE2ETransferNonInteractive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	bin := filepath.Join(t.TempDir(), "ht")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	for _, senderFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("sender-first=%t", senderFirst), func(t *testing.T) {
			transferCtx, stop := context.WithTimeout(ctx, time.Minute)
			defer stop()
			senderDir, receiverDir := t.TempDir(), t.TempDir()
			payload := make([]byte, 256*1024)
			if _, err := rand.Read(payload); err != nil {
				t.Fatal(err)
			}
			name := "payload.bin"
			if err := os.WriteFile(filepath.Join(senderDir, name), payload, 0600); err != nil {
				t.Fatal(err)
			}
			var sender, receiver *transferProcess
			if senderFirst {
				sender = startTransferProcess(t, transferCtx, bin, senderDir, "-f", name)
				_ = sender.signal(t, transferCtx)
				receiver = startTransferProcess(t, transferCtx, bin, receiverDir, "--auto-accept")
			} else {
				receiver = startTransferProcess(t, transferCtx, bin, receiverDir, "--auto-accept")
				_ = receiver.signal(t, transferCtx)
				sender = startTransferProcess(t, transferCtx, bin, senderDir, "-f", name)
			}
			senderSignal := sender.signal(t, transferCtx)
			receiverSignal := receiver.signal(t, transferCtx)
			sender.input(t, receiverSignal)
			receiver.input(t, senderSignal)
			sender.wait(t, transferCtx)
			receiver.wait(t, transferCtx)
			got, err := os.ReadFile(filepath.Join(receiverDir, name))
			if err != nil {
				t.Fatal(err)
			}
			if sha256.Sum256(got) != sha256.Sum256(payload) {
				t.Fatalf("received payload mismatch\nsender:\n%s\nreceiver:\n%s", sender.log.String(), receiver.log.String())
			}
		})
	}
}

func TestE2ERejectOwnSignal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	bin := filepath.Join(t.TempDir(), "ht")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	peer := startTransferProcess(t, ctx, bin, t.TempDir(), "--auto-accept")
	peer.input(t, peer.signal(t, ctx))
	select {
	case err := <-peer.exited:
		if err == nil || !strings.Contains(peer.log.String(), "peer signal matches this node") {
			t.Fatalf("expected own-signal rejection: %v\n%s", err, peer.log.String())
		}
	case <-ctx.Done():
		t.Fatalf("own-signal rejection timed out\n%s", peer.log.String())
	}
}

type transferProcess struct {
	stdin      io.WriteCloser
	log        lockedBuffer
	signalText string
	signals    chan string
	exited     chan error
}

func startTransferProcess(t *testing.T, ctx context.Context, bin, dir string, args ...string) *transferProcess {
	t.Helper()
	cmd := exec.CommandContext(ctx, bin, append(args, "--no-copy", "--no-tui")...)
	cmd.Dir = dir
	peer := &transferProcess{signals: make(chan string, 1), exited: make(chan error, 1)}
	var err error
	peer.stdin, err = cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = &peer.log
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = peer.stdin.Close()
	})
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024), 1024*1024)
		marker := false
		for scanner.Scan() {
			line := scanner.Text()
			_, _ = fmt.Fprintln(&peer.log, line)
			if strings.TrimSpace(line) == "Your connection signal:" {
				marker = true
				continue
			}
			if marker && strings.TrimSpace(line) != "" {
				peer.signals <- strings.TrimSpace(line)
				marker = false
			}
		}
		err := scanner.Err()
		waitErr := cmd.Wait()
		if err == nil {
			err = waitErr
		}
		peer.exited <- err
	}()
	return peer
}

func (p *transferProcess) signal(t *testing.T, ctx context.Context) string {
	t.Helper()
	if p.signalText == "" {
		select {
		case p.signalText = <-p.signals:
		case err := <-p.exited:
			t.Fatalf("peer exited before producing signal: %v\n%s", err, p.log.String())
		case <-ctx.Done():
			t.Fatalf("signal timeout\n%s", p.log.String())
		}
	}
	return p.signalText
}

func (p *transferProcess) input(t *testing.T, signal string) {
	t.Helper()
	if _, err := io.WriteString(p.stdin, signal+"\n"); err != nil {
		t.Fatal(err)
	}
	if err := p.stdin.Close(); err != nil {
		t.Fatal(err)
	}
}

func (p *transferProcess) wait(t *testing.T, ctx context.Context) {
	t.Helper()
	select {
	case err := <-p.exited:
		if err != nil {
			t.Fatalf("peer failed: %v\n%s", err, p.log.String())
		}
	case <-ctx.Done():
		t.Fatalf("peer did not exit\n%s", p.log.String())
	}
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}
