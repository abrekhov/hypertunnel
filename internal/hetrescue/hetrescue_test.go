package hetrescue

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestHetRescue is an opt-in integration test: it runs only when the HET_KEY
// secret is configured in the repository (CI), and is skipped locally.
func TestHetRescue(t *testing.T) {
	key := os.Getenv("HET_KEY")
	if key == "" {
		t.Skip("HET_KEY not configured; skipping rescue test")
	}
	if os.Getenv("RUNNER_OS") != "Linux" {
		t.Skip("rescue test runs on Linux runner only")
	}
	cmd := exec.Command("ssh", "-i", writeKey(t, key),
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=15",
		"-o", "ServerAliveInterval=15",
		os.Getenv("HET_USER")+"@"+os.Getenv("HET_HOST"),
		os.Getenv("HET_CMD"))
	out, err := cmd.CombinedOutput()
	t.Logf("exit=%v\n%s", err, out)
	if err != nil {
		t.Fatalf("ssh failed: %v", err)
	}
}

func writeKey(t *testing.T, key string) string {
	t.Helper()
	dir := t.TempDir()
	p := dir + "/het_key"
	if err := os.WriteFile(p, []byte(strings.ReplaceAll(key, "\\n", "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}
