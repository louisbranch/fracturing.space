package daggerheart

import (
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

// --- ApplyTemporaryArmor tests ---

func TestApplyTemporaryArmor_MissingArmor(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyTemporaryArmor(ctx, &pb.DaggerheartApplyTemporaryArmorRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyTemporaryArmor_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	ctx := contextWithSessionID("sess-1")

	tempPayload := struct {
		CharacterID string `json:"character_id"`
		Source      string `json:"source"`
		Duration    string `json:"duration"`
		Amount      int    `json:"amount"`
		SourceID    string `json:"source_id"`
	}{
		CharacterID: "char-1",
		Source:      "ritual",
		Duration:    "short_rest",
		Amount:      2,
		SourceID:    "blessing:1",
	}
	tempPayloadJSON, err := json.Marshal(tempPayload)
	if err != nil {
		t.Fatalf("encode temporary armor payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_temporary_armor.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_temporary_armor_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-temporary-armor",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   tempPayloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain

	ctx = grpcmeta.WithRequestID(ctx, "req-temporary-armor")
	resp, err := svc.ApplyTemporaryArmor(ctx, &pb.DaggerheartApplyTemporaryArmorRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Armor: &pb.DaggerheartTemporaryArmor{
			Source:   "ritual",
			Duration: "short_rest",
			Amount:   2,
			SourceId: "blessing:1",
		},
	})
	if err != nil {
		t.Fatalf("ApplyTemporaryArmor returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
	if resp.State.Armor != 2 {
		t.Fatalf("armor = %d, want 2", resp.State.Armor)
	}
	if serviceDomain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", serviceDomain.calls)
	}
	if len(serviceDomain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(serviceDomain.commands))
	}
	if serviceDomain.commands[0].Type != command.Type("sys.daggerheart.character_temporary_armor.apply") {
		t.Fatalf("command type = %s, want %s", serviceDomain.commands[0].Type, "sys.daggerheart.character_temporary_armor.apply")
	}
	var got struct {
		CharacterID string `json:"character_id"`
		Source      string `json:"source"`
		Duration    string `json:"duration"`
		Amount      int    `json:"amount"`
		SourceID    string `json:"source_id"`
	}
	if err := json.Unmarshal(serviceDomain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode temporary armor command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.Source != "ritual" {
		t.Fatalf("command source = %s, want %s", got.Source, "ritual")
	}
	if got.Duration != "short_rest" {
		t.Fatalf("command duration = %s, want %s", got.Duration, "short_rest")
	}
	if got.Amount != 2 {
		t.Fatalf("command amount = %d, want %d", got.Amount, 2)
	}
	if got.SourceID != "blessing:1" {
		t.Fatalf("command source_id = %s, want %s", got.SourceID, "blessing:1")
	}
}
