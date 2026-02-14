package cursor

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	original := Cursor{
		Seq:        42,
		Dir:        DirectionForward,
		Reverse:    true,
		FilterHash: HashFilter("status = 'active'"),
		OrderHash:  HashFilter("seq desc"),
	}

	token, err := Encode(original)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	decoded, err := Decode(token)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}

	if decoded != original {
		t.Fatalf("cursor mismatch: %+v != %+v", decoded, original)
	}
}

func TestDecodeEmptyToken(t *testing.T) {
	_, err := Decode("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestDecodeInvalidBase64(t *testing.T) {
	_, err := Decode("not-base64@@")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecodeInvalidDirection(t *testing.T) {
	raw, err := json.Marshal(Cursor{Seq: 1, Dir: "sideways"})
	if err != nil {
		t.Fatalf("marshal cursor: %v", err)
	}
	token := base64.URLEncoding.EncodeToString(raw)

	_, err = Decode(token)
	if err == nil {
		t.Fatal("expected error for invalid direction")
	}
}

func TestHashFilter(t *testing.T) {
	if HashFilter("") != "" {
		t.Fatal("expected empty hash for empty filter")
	}

	hash := HashFilter("foo")
	if len(hash) != 16 {
		t.Fatalf("expected 16-char hash, got %d", len(hash))
	}

	if hash == HashFilter("bar") {
		t.Fatal("expected different hashes for different filters")
	}
}

func TestValidateFilterHash(t *testing.T) {
	c := NewForwardCursor(10, "status = 'active'", "seq asc")
	if err := ValidateFilterHash(c, "status = 'active'"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateFilterHash(c, "status = 'archived'"); err == nil {
		t.Fatal("expected error for mismatched filter")
	}
}

func TestValidateOrderHash(t *testing.T) {
	c := NewForwardCursor(10, "status = 'active'", "seq asc")
	if err := ValidateOrderHash(c, "seq asc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateOrderHash(c, "seq desc"); err == nil {
		t.Fatal("expected error for mismatched order")
	}
}

func TestPageCursorDirections(t *testing.T) {
	nextAsc := NewNextPageCursor(100, false, "", "")
	if nextAsc.Dir != DirectionForward {
		t.Fatalf("expected forward dir, got %s", nextAsc.Dir)
	}
	if nextAsc.Reverse {
		t.Fatal("expected forward cursor without reverse")
	}

	nextDesc := NewNextPageCursor(100, true, "", "")
	if nextDesc.Dir != DirectionBackward {
		t.Fatalf("expected backward dir, got %s", nextDesc.Dir)
	}

	prevAsc := NewPrevPageCursor(50, false, "", "")
	if prevAsc.Dir != DirectionBackward {
		t.Fatalf("expected backward dir, got %s", prevAsc.Dir)
	}
	if !prevAsc.Reverse {
		t.Fatal("expected reverse for prev cursor")
	}

	prevDesc := NewPrevPageCursor(50, true, "", "")
	if prevDesc.Dir != DirectionForward {
		t.Fatalf("expected forward dir, got %s", prevDesc.Dir)
	}
	if !prevDesc.Reverse {
		t.Fatal("expected reverse for prev cursor")
	}
}

func TestFilterAndOrderHashesDiffer(t *testing.T) {
	c := NewForwardCursor(10, "filter-a", "order-b")
	if c.FilterHash == "" || c.OrderHash == "" {
		t.Fatal("expected non-empty hashes")
	}
	if c.FilterHash == c.OrderHash {
		t.Fatal("expected filter and order hashes to differ")
	}
}
