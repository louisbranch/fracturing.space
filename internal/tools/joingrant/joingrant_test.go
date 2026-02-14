package joingrant

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestRunRequiresOutput(t *testing.T) {
	if err := Run(nil, bytes.NewReader([]byte{1})); err == nil {
		t.Fatal("expected error when output is nil")
	}
}

func TestRunWritesKeys(t *testing.T) {
	buf := &bytes.Buffer{}
	reader := bytes.NewReader(bytes.Repeat([]byte{1}, 64))
	if err := Run(buf, reader); err != nil {
		t.Fatalf("run: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	private := strings.TrimPrefix(lines[0], "export FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY=")
	public := strings.TrimPrefix(lines[1], "export FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY=")
	if private == lines[0] || public == lines[1] {
		t.Fatalf("unexpected output format: %q", buf.String())
	}

	privateBytes, err := base64.RawStdEncoding.DecodeString(private)
	if err != nil {
		t.Fatalf("decode private key: %v", err)
	}
	publicBytes, err := base64.RawStdEncoding.DecodeString(public)
	if err != nil {
		t.Fatalf("decode public key: %v", err)
	}
	if len(privateBytes) != 64 {
		t.Fatalf("expected private key length 64, got %d", len(privateBytes))
	}
	if len(publicBytes) != 32 {
		t.Fatalf("expected public key length 32, got %d", len(publicBytes))
	}
}
