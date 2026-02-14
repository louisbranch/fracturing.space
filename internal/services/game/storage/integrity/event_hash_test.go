package integrity

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
)

func TestEventHashDeterministic(t *testing.T) {
	ts := time.Date(2024, 2, 1, 10, 30, 0, 0, time.UTC)
	evt := event.Event{
		CampaignID:  "c1",
		Timestamp:   ts,
		Type:        event.TypeCampaignCreated,
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
		Type:        event.TypeCampaignCreated,
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
		Type:        event.TypeCampaignCreated,
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
		Type:        event.TypeCampaignCreated,
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
