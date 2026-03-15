package adversarytransport

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func TestAdversaryToProtoSession(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	proto := AdversaryToProto(projectionstore.DaggerheartAdversary{
		AdversaryID: "adv-1",
		CampaignID:  "camp-1",
		Name:        "Rival",
		SessionID:   "sess-1",
		HP:          4,
		HPMax:       6,
		Conditions:  []string{"hidden"},
		CreatedAt:   created,
		UpdatedAt:   updated,
	})
	if proto.GetSessionId().GetValue() != "sess-1" {
		t.Fatal("expected session id wrapper")
	}
	if proto.GetCreatedAt().AsTime().UTC() != created {
		t.Fatal("expected created time to map")
	}
	if len(proto.GetConditions()) != 1 {
		t.Fatalf("expected conditions to map, got %v", proto.GetConditions())
	}
}

func TestAdversaryToProtoNoSession(t *testing.T) {
	proto := AdversaryToProto(projectionstore.DaggerheartAdversary{
		AdversaryID: "adv-2",
		CampaignID:  "camp-1",
		Name:        "Shadow",
		HP:          6,
		HPMax:       8,
		Stress:      2,
		StressMax:   4,
		Evasion:     10,
		Major:       7,
		Severe:      14,
		Armor:       2,
		Conditions:  []string{"vulnerable"},
	})
	if proto.GetSessionId() != nil {
		t.Fatal("expected nil session id wrapper when no session")
	}
	if proto.GetId() != "adv-2" || proto.GetName() != "Shadow" {
		t.Fatalf("proto metadata mismatch: %v", proto)
	}
}
