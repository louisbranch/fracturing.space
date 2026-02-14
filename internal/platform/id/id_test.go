package id

import (
	"encoding/base32"
	"strings"
	"testing"
)

func TestNewIDFormat(t *testing.T) {
	id, err := NewID()
	if err != nil {
		t.Fatalf("new id: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty id")
	}
	if strings.Contains(id, "=") {
		t.Fatal("expected no padding")
	}
	if len(id) != 26 {
		t.Fatalf("expected 26-character id, got %d", len(id))
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < '2' || r > '7') {
			t.Fatalf("unexpected character %q in id", r)
		}
	}

	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(id))
	if err != nil {
		t.Fatalf("decode id: %v", err)
	}
	if len(decoded) != 16 {
		t.Fatalf("expected 16 decoded bytes, got %d", len(decoded))
	}
}

func TestNewIDSetsUUIDVersionAndVariant(t *testing.T) {
	id, err := NewID()
	if err != nil {
		t.Fatalf("new id: %v", err)
	}

	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(id))
	if err != nil {
		t.Fatalf("decode id: %v", err)
	}
	if len(decoded) != 16 {
		t.Fatalf("expected 16 decoded bytes, got %d", len(decoded))
	}

	version := decoded[6] >> 4
	if version != 4 {
		t.Fatalf("expected version 4, got %d", version)
	}
	variant := decoded[8] & 0xC0
	if variant != 0x80 {
		t.Fatalf("expected variant 0x80, got 0x%X", variant)
	}
}
