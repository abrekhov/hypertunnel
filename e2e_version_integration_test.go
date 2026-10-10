//go:build integration

package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestE2EVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	bin := filepath.Join(t.TempDir(), "ht")
	flags := "-X github.com/abrekhov/hypertunnel/cmd.version=1.2.3-e2e -X github.com/abrekhov/hypertunnel/cmd.commit=abc123 -X github.com/abrekhov/hypertunnel/cmd.date=2026-01-01T00:00:00Z"
	if out, err := exec.CommandContext(ctx, "go", "build", "-ldflags", flags, "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"command", []string{"version"}, "1.2.3-e2e\n"},
		{"flag", []string{"--version"}, "1.2.3-e2e\n"},
		{"metadata", []string{"version", "--long"}, "version\t1.2.3-e2e\ncommit\tabc123\ndate\t2026-01-01T00:00:00Z\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := exec.CommandContext(ctx, bin, tc.args...).CombinedOutput()
			if err != nil || string(out) != tc.want {
				t.Fatalf("version output: %v\nwant %q\ngot %q", err, tc.want, out)
			}
		})
	}
	if out, err := exec.CommandContext(ctx, bin, "version", "unexpected").CombinedOutput(); err == nil {
		t.Fatalf("extra arguments accepted: %s", out)
	}
}
