//go:build integration

/*
 *   Copyright © 2021-2026 Anton Brekhov <anton@abrekhov.ru>
 *   All rights reserved.
 */

package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestE2ETransferNonInteractive(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("E2E transfer test is supported only on linux in CI")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	binDir := t.TempDir()
	htPath := filepath.Join(binDir, "ht")
	build := exec.CommandContext(ctx, "go", "build", "-o", htPath, "./main.go")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	buildOut, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build ht: %v\n%s", err, string(buildOut))
	}

	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		if attempt > 1 {
			t.Logf("retrying E2E transfer (attempt %d/2)", attempt)
		}
		if err := runTransferAttempt(ctx, htPath); err != nil {
			lastErr = err
			continue
		}
		return
	}

	if lastErr == nil {
		lastErr = errors.New("unknown failure")
	}
	t.Fatalf("E2E transfer failed after 2 attempts: %v", lastErr)
}

func runTransferAttempt(parent context.Context, htPath string) error {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()

	senderDir, err := os.MkdirTemp("", "ht-e2e-sender-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(senderDir)

	receiverDir, err := os.MkdirTemp("", "ht-e2e-receiver-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(receiverDir)

	dummyName := "dummy.bin"
	dummyPath := filepath.Join(senderDir, dummyName)
	if err := writeRandomFile(dummyPath, 256*1024); err != nil {
		return fmt.Errorf("write dummy file: %w", err)
	}
	sentHash, err := sha256File(dummyPath)
	if err != nil {
		return fmt.Errorf("hash sent file: %w", err)
	}

	// Start receiver.
	receiver := exec.CommandContext(ctx, htPath, "--auto-accept", "--no-copy")
	receiver.Dir = receiverDir
	receiverIn, err := receiver.StdinPipe()
	if err != nil {
		return err
	}
	receiverStdout, err := receiver.StdoutPipe()
	if err != nil {
		return err
	}
	receiverStderr, err := receiver.StderrPipe()
	if err != nil {
		return err
	}

	receiverLog := &lockedBuffer{}
	receiverSignalCh := make(chan string, 1)
	receiverDrainDone := make(chan struct{})
	go func() {
		defer close(receiverDrainDone)
		drainOutput(receiverStdout, receiverLog, receiverSignalCh)
	}()
	go func() {
		_, _ = io.Copy(receiverLog, receiverStderr)
	}()

	if err := receiver.Start(); err != nil {
		return err
	}
	defer func() {
		_ = receiver.Process.Kill()
	}()

	var receiverSignal string
	select {
	case receiverSignal = <-receiverSignalCh:
	case <-ctx.Done():
		return fmt.Errorf("receiver did not produce signal: %w\nreceiver log:\n%s", ctx.Err(), receiverLog.String())
	}

	// Start sender.
	sender := exec.CommandContext(ctx, htPath, "--file", dummyPath, "--no-copy")
	sender.Dir = senderDir
	senderIn, err := sender.StdinPipe()
	if err != nil {
		return err
	}
	senderStdout, err := sender.StdoutPipe()
	if err != nil {
		return err
	}
	senderStderr, err := sender.StderrPipe()
	if err != nil {
		return err
	}

	senderLog := &lockedBuffer{}
	senderSignalCh := make(chan string, 1)
	senderDrainDone := make(chan struct{})
	go func() {
		defer close(senderDrainDone)
		drainOutput(senderStdout, senderLog, senderSignalCh)
	}()
	go func() {
		_, _ = io.Copy(senderLog, senderStderr)
	}()

	if err := sender.Start(); err != nil {
		return err
	}
	defer func() {
		_ = sender.Process.Kill()
	}()

	// Provide receiver signal to sender, then EOF.
	if _, err := io.WriteString(senderIn, receiverSignal+"\n"); err != nil {
		return fmt.Errorf("write receiver signal to sender stdin: %w", err)
	}
	if err := senderIn.Close(); err != nil {
		return err
	}

	// Get sender signal and provide it to receiver.
	var senderSignal string
	select {
	case senderSignal = <-senderSignalCh:
	case <-ctx.Done():
		return fmt.Errorf("sender did not produce signal: %w\nsender log:\n%s", ctx.Err(), senderLog.String())
	}

	if _, err := io.WriteString(receiverIn, senderSignal+"\n"); err != nil {
		return fmt.Errorf("write sender signal to receiver stdin: %w", err)
	}
	if err := receiverIn.Close(); err != nil {
		return err
	}

	receivedPath := filepath.Join(receiverDir, dummyName)
	if err := waitForChecksum(ctx, receivedPath, sentHash); err != nil {
		return fmt.Errorf("transfer did not complete: %w\nreceiver log:\n%s\nsender log:\n%s", err, receiverLog.String(), senderLog.String())
	}

	// Best-effort shutdown (ht may not exit on its own in sender mode).
	_ = sender.Process.Kill()
	_ = receiver.Process.Kill()
	_ = sender.Wait()
	_ = receiver.Wait()
	<-receiverDrainDone
	<-senderDrainDone

	return nil
}

func drainOutput(r io.Reader, log io.Writer, signalCh chan<- string) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	seenMarker := false
	sentSignal := false
	for scanner.Scan() {
		line := scanner.Text()
		_, _ = io.WriteString(log, line+"\n")
		if strings.TrimSpace(line) == "Your connection signal:" {
			seenMarker = true
			continue
		}
		if seenMarker {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !sentSignal {
				signalCh <- trimmed
				sentSignal = true
			}
		}
	}
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func waitForChecksum(ctx context.Context, path, expected string) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			h, err := sha256File(path)
			if err != nil {
				continue
			}
			if h == expected {
				return nil
			}
		}
	}
}

func writeRandomFile(path string, size int) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 32*1024)
	remaining := size
	for remaining > 0 {
		chunk := remaining
		if chunk > len(buf) {
			chunk = len(buf)
		}
		if _, err := rand.Read(buf[:chunk]); err != nil {
			return err
		}
		if _, err := f.Write(buf[:chunk]); err != nil {
			return err
		}
		remaining -= chunk
	}
	return f.Close()
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
