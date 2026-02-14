package hmackey

import (
	"bytes"
	"flag"
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
	if got := strings.TrimSpace(buf.String()); got != "01020304" {
		t.Fatalf("expected hex output, got %q", got)
	}
}
