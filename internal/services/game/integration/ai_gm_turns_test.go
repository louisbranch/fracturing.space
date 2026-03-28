package integration

import "testing"

func TestAIGMTurnRequestedDedupeKey(t *testing.T) {
	t.Parallel()

	first := AIGMTurnRequestedDedupeKey("camp-1", 42)
	same := AIGMTurnRequestedDedupeKey("camp-1", 42)
	if first != same {
		t.Fatalf("same source event dedupe key mismatch: %q != %q", first, same)
	}

	if other := AIGMTurnRequestedDedupeKey("camp-1", 43); other == first {
		t.Fatalf("different source event seq should change dedupe key: %q", other)
	}

	if other := AIGMTurnRequestedDedupeKey("camp-2", 42); other == first {
		t.Fatalf("different campaign should change dedupe key: %q", other)
	}
}
