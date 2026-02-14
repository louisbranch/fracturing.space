package config_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
)

// TestExitf_ExitsWithCode1 verifies that Exitf writes to stderr and exits
// with code 1. It uses the subprocess test pattern because os.Exit cannot be
// intercepted in-process.
func TestExitf_ExitsWithCode1(t *testing.T) {
	if os.Getenv("TEST_EXITF_SUBPROCESS") == "1" {
		config.Exitf("fatal: %s", "something broke")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestExitf_ExitsWithCode1$")
	cmd.Env = append(os.Environ(), "TEST_EXITF_SUBPROCESS=1")

	out, err := cmd.CombinedOutput()

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
	}
	if !strings.Contains(string(out), "fatal: something broke") {
		t.Fatalf("expected stderr to contain %q, got %q", "fatal: something broke", string(out))
	}
}
