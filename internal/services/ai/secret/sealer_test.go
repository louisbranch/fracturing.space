package secret

import "testing"

func TestNewAESGCMSealerRequiresValidKey(t *testing.T) {
	if _, err := NewAESGCMSealer([]byte("short")); err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestAESGCMSealerSealOpenRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	sealer, err := NewAESGCMSealer(key)
	if err != nil {
		t.Fatalf("new sealer: %v", err)
	}

	sealed, err := sealer.Seal("sk-123")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if sealed == "sk-123" {
		t.Fatal("expected encrypted output")
	}

	opened, err := sealer.Open(sealed)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if opened != "sk-123" {
		t.Fatalf("opened = %q, want %q", opened, "sk-123")
	}
}

func TestAESGCMSealerOpenRejectsInvalidCiphertext(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	sealer, err := NewAESGCMSealer(key)
	if err != nil {
		t.Fatalf("new sealer: %v", err)
	}

	if _, err := sealer.Open("not-base64"); err == nil {
		t.Fatal("expected error for invalid ciphertext")
	}
}
