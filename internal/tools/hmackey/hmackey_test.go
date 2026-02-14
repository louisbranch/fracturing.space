package hmackey

import (
	"bytes"
	"flag"
	"fmt"
	"strings"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("hmackey", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Bytes != 32 {
		t.Fatalf("expected default bytes 32, got %d", cfg.Bytes)
	}
}

func TestParseConfigOverride(t *testing.T) {
	fs := flag.NewFlagSet("hmackey", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-bytes", "16"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Bytes != 16 {
		t.Fatalf("expected bytes 16, got %d", cfg.Bytes)
	}
}

func TestRunRejectsInvalidBytes(t *testing.T) {
	if err := Run(Config{Bytes: 0}, &bytes.Buffer{}, bytes.NewReader(nil)); err == nil {
		t.Fatal("expected error for non-positive bytes")
	}
}

func TestRunWritesHex(t *testing.T) {
	buf := &bytes.Buffer{}
	reader := bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04})
	if err := Run(Config{Bytes: 4}, buf, reader); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got != "FRACTURING_SPACE_GAME_EVENT_HMAC_KEY=01020304" {
		t.Fatalf("expected env output, got %q", got)
	}
}

func TestRunNilOutput(t *testing.T) {
	if err := Run(Config{Bytes: 4}, nil, nil); err == nil {
		t.Fatal("expected error for nil output")
	}
}

func TestRunDefaultReader(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := Run(Config{Bytes: 4}, buf, nil); err != nil {
		t.Fatalf("run: %v", err)
	}
	// Default reader is crypto/rand, so output should be env key + 8 hex chars.
	got := strings.TrimSpace(buf.String())
	const prefix = "FRACTURING_SPACE_GAME_EVENT_HMAC_KEY="
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("expected env prefix, got %q", got)
	}
	if len(strings.TrimPrefix(got, prefix)) != 8 {
		t.Fatalf("expected 8 hex chars, got %d: %q", len(strings.TrimPrefix(got, prefix)), got)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, fmt.Errorf("read error") }

func TestRunReaderError(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := Run(Config{Bytes: 4}, buf, errReader{}); err == nil {
		t.Fatal("expected error from failing reader")
	}
}

func TestParseConfigBadArgs(t *testing.T) {
	fs := flag.NewFlagSet("hmackey", flag.ContinueOnError)
	fs.SetOutput(&bytes.Buffer{})
	if _, err := ParseConfig(fs, []string{"-invalid"}); err == nil {
		t.Fatal("expected error for unknown flag")
	}
}
