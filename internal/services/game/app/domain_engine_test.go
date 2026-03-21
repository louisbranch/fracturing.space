package server

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	storagesqlite "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite"
)

func TestDomainCampaignUpdateAfterCreate(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	keyring, err := integrity.KeyringFromEnv()
	if err != nil {
		t.Fatalf("load keyring: %v", err)
	}
	registries, err := engine.BuildRegistries(daggerheart.NewModule())
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}

	eventPath := filepath.Join(t.TempDir(), "game-events.db")
	store, err := storagesqlite.OpenEvents(eventPath, keyring, registries.Events)
	if err != nil {
		t.Fatalf("open event store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close event store: %v", closeErr)
		}
	})

	domainEngine, err := buildDomainEngine(store, registries)
	if err != nil {
		t.Fatalf("build domain engine: %v", err)
	}

	createPayload := campaign.CreatePayload{
		Name:       "Test Campaign",
		Locale:     "en-US",
		GameSystem: "GAME_SYSTEM_DAGGERHEART",
		GmMode:     "GM_MODE_HUMAN",
	}
	payloadJSON, err := json.Marshal(createPayload)
	if err != nil {
		t.Fatalf("marshal create payload: %v", err)
	}

	result, err := domainEngine.Execute(context.Background(), command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("execute campaign.create: %v", err)
	}
	if len(result.Decision.Rejections) > 0 {
		t.Fatalf("campaign.create rejected: %s", result.Decision.Rejections[0].Message)
	}
	if len(result.Decision.Events) == 0 {
		t.Fatal("campaign.create did not emit events")
	}

	updatePayload := campaign.UpdatePayload{Fields: map[string]string{"status": "active"}}
	updateJSON, err := json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("marshal update payload: %v", err)
	}

	updateResult, err := domainEngine.Execute(context.Background(), command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.update"),
		ActorType:   command.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: updateJSON,
	})
	if err != nil {
		t.Fatalf("execute campaign.update: %v", err)
	}
	if len(updateResult.Decision.Rejections) > 0 {
		t.Fatalf("campaign.update rejected: %s", updateResult.Decision.Rejections[0].Message)
	}
}
