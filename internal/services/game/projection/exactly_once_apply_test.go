package projection

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sqlitecoreprojection "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/coreprojection"
)

func TestBuildExactlyOnceApplySkipsDuplicateSeq(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projections.db")
	store, err := sqlitecoreprojection.Open(path)
	if err != nil {
		t.Fatalf("open projection store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close projection store: %v", closeErr)
		}
	})

	now := time.Date(2026, 2, 18, 20, 0, 0, 0, time.UTC)
	if err := store.Put(context.Background(), storage.CampaignRecord{
		ID:               "camp-outbox-exactly-once",
		Name:             "Exactly Once",
		Locale:           "en-US",
		System:           bridge.SystemIDDaggerheart,
		Status:           campaign.StatusDraft,
		GmMode:           campaign.GmModeHuman,
		Intent:           campaign.IntentStandard,
		AccessPolicy:     campaign.AccessPolicyPrivate,
		ParticipantCount: 0,
		CharacterCount:   0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("seed campaign: %v", err)
	}

	apply, err := BuildExactlyOnceApply(store, nil)
	if err != nil {
		t.Fatalf("build projection apply: %v", err)
	}
	if apply == nil {
		t.Fatal("expected projection apply callback")
	}

	payload, err := json.Marshal(participant.JoinPayload{
		ParticipantID:  "part-apply-once",
		Name:           "Rook",
		Role:           "player",
		Controller:     "human",
		CampaignAccess: "member",
	})
	if err != nil {
		t.Fatalf("marshal participant payload: %v", err)
	}

	evt := event.Event{
		CampaignID:  "camp-outbox-exactly-once",
		Seq:         501,
		Type:        event.Type("participant.joined"),
		Timestamp:   now.Add(time.Second),
		EntityType:  "participant",
		EntityID:    "part-apply-once",
		PayloadJSON: payload,
	}

	if err := apply(context.Background(), evt); err != nil {
		t.Fatalf("first projection apply: %v", err)
	}
	if err := apply(context.Background(), evt); err != nil {
		t.Fatalf("duplicate projection apply: %v", err)
	}

	campaignRecord, err := store.Get(context.Background(), string(evt.CampaignID))
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaignRecord.ParticipantCount != 1 {
		t.Fatalf("expected participant count 1 after duplicate apply, got %d", campaignRecord.ParticipantCount)
	}
}
