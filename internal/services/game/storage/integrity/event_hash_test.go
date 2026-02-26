package integrity

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestEventHashDeterministic(t *testing.T) {
	ts := time.Date(2024, 2, 1, 10, 30, 0, 0, time.UTC)
	evt := event.Event{
		CampaignID:  "c1",
		Timestamp:   ts,
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"demo"}`),
	}

	first, err := EventHash(evt)
	if err != nil {
		t.Fatalf("event hash: %v", err)
	}

	second, err := EventHash(evt)
	if err != nil {
		t.Fatalf("event hash: %v", err)
	}

	if first != second {
		t.Fatalf("expected deterministic hash, got %s and %s", first, second)
	}
}

func TestEventHashChangesWithOptionalFields(t *testing.T) {
	ts := time.Date(2024, 2, 1, 10, 30, 0, 0, time.UTC)
	base := event.Event{
		CampaignID:  "c1",
		Timestamp:   ts,
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"demo"}`),
	}

	baseline, err := EventHash(base)
	if err != nil {
		t.Fatalf("event hash: %v", err)
	}

	withSession := base
	withSession.SessionID = "s1"
	hashSession, err := EventHash(withSession)
	if err != nil {
		t.Fatalf("event hash: %v", err)
	}

	if baseline == hashSession {
		t.Fatal("expected hash to change when optional fields change")
	}
}

func TestChainHashRequiresEventHash(t *testing.T) {
	evt := event.Event{
		CampaignID:  "c1",
		Seq:         10,
		Timestamp:   time.Date(2024, 2, 1, 10, 30, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"demo"}`),
	}

	_, err := ChainHash(evt, "prev")
	if err == nil {
		t.Fatal("expected error when event hash is missing")
	}
}

func TestChainHashDeterministic(t *testing.T) {
	evt := event.Event{
		CampaignID:  "c1",
		Seq:         10,
		Hash:        "eventhash",
		Timestamp:   time.Date(2024, 2, 1, 10, 30, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"demo"}`),
	}

	first, err := ChainHash(evt, "prev")
	if err != nil {
		t.Fatalf("chain hash: %v", err)
	}
	second, err := ChainHash(evt, "prev")
	if err != nil {
		t.Fatalf("chain hash: %v", err)
	}
	if first != second {
		t.Fatalf("expected deterministic chain hash, got %s and %s", first, second)
	}
}

// TestHashParityWithDomainPackage asserts that the integrity package and the
// domain event package produce identical hashes for the same event. This guards
// against canonical-field drift when a new envelope field is added to only one
// of the two implementations.
func TestHashParityWithDomainPackage(t *testing.T) {
	// Representative event with ALL optional fields populated to maximize
	// surface area for drift detection.
	ts := time.Date(2024, 6, 15, 8, 45, 30, 123456789, time.UTC)
	evt := event.Event{
		CampaignID:    "camp-parity",
		Seq:           42,
		Hash:          "abcdef1234567890",
		Type:          event.Type("sys.daggerheart.character.hp_patched"),
		Timestamp:     ts,
		ActorType:     event.ActorTypeParticipant,
		ActorID:       "actor-1",
		SessionID:     "sess-1",
		RequestID:     "req-1",
		InvocationID:  "inv-1",
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      "DAGGERHEART",
		SystemVersion: "0.4.2",
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
		PayloadJSON:   []byte(`{"hp_after":5,"source":"damage"}`),
	}

	t.Run("EventHash", func(t *testing.T) {
		domainHash, err := event.EventHash(evt)
		if err != nil {
			t.Fatalf("domain EventHash: %v", err)
		}
		integrityHash, err := EventHash(evt)
		if err != nil {
			t.Fatalf("integrity EventHash: %v", err)
		}
		if domainHash != integrityHash {
			t.Fatalf("EventHash drift: domain=%s integrity=%s", domainHash, integrityHash)
		}
	})

	t.Run("ChainHash", func(t *testing.T) {
		domainChain, err := event.ChainHash(evt, "prev-hash-abc")
		if err != nil {
			t.Fatalf("domain ChainHash: %v", err)
		}
		integrityChain, err := ChainHash(evt, "prev-hash-abc")
		if err != nil {
			t.Fatalf("integrity ChainHash: %v", err)
		}
		if domainChain != integrityChain {
			t.Fatalf("ChainHash drift: domain=%s integrity=%s", domainChain, integrityChain)
		}
	})
}
