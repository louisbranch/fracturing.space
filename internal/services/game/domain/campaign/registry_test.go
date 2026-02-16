package campaign

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestRegisterCommands_ValidatesCreatePayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"Sunfall","game_system":"daggerheart","gm_mode":"human"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":1,"game_system":"daggerheart","gm_mode":"human"}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesCreatedPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.created"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: []byte(`{"name":"Sunfall","game_system":"daggerheart","gm_mode":"human"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"name":1,"game_system":"daggerheart","gm_mode":"human"}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_CreatedRequiresEntityTargetAddressing(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	base := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.created"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"Sunfall","game_system":"daggerheart","gm_mode":"human"}`),
	}

	_, err := registry.ValidateForAppend(base)
	if err == nil {
		t.Fatal("expected missing entity type error")
	}
	if !errors.Is(err, event.ErrEntityTypeRequired) {
		t.Fatalf("expected ErrEntityTypeRequired, got %v", err)
	}

	withType := base
	withType.EntityType = "campaign"
	_, err = registry.ValidateForAppend(withType)
	if err == nil {
		t.Fatal("expected missing entity id error")
	}
	if !errors.Is(err, event.ErrEntityIDRequired) {
		t.Fatalf("expected ErrEntityIDRequired, got %v", err)
	}

	withTypeAndID := withType
	withTypeAndID.EntityID = "camp-1"
	if _, err := registry.ValidateForAppend(withTypeAndID); err != nil {
		t.Fatalf("valid addressed event rejected: %v", err)
	}
}

func TestRegisterCommands_ValidatesUpdatePayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"fields":{"status":"active"}}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"fields":{"status":1}}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesUpdatedPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.updated"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: []byte(`{"fields":{"status":"active"}}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"fields":{"status":1}}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterCommands_ValidatesForkPayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.fork"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"parent_campaign_id":"camp-0","fork_event_seq":3,"origin_campaign_id":"camp-root"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"parent_campaign_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesForkedPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.forked"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: []byte(`{"parent_campaign_id":"camp-0","fork_event_seq":3,"origin_campaign_id":"camp-root"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"parent_campaign_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterCommands_ValidatesStatusCommands(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	tests := []struct {
		name string
		cmd  command.Type
	}{
		{name: "end", cmd: command.Type("campaign.end")},
		{name: "archive", cmd: command.Type("campaign.archive")},
		{name: "restore", cmd: command.Type("campaign.restore")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validCommand := command.Command{
				CampaignID:  "camp-1",
				Type:        tt.cmd,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{}`),
			}
			if _, err := registry.ValidateForDecision(validCommand); err != nil {
				t.Fatalf("valid command rejected: %v", err)
			}

			invalidCommand := command.Command{
				CampaignID:  "camp-1",
				Type:        tt.cmd,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{"status":"active"}`),
			}
			_, err := registry.ValidateForDecision(invalidCommand)
			if err == nil {
				t.Fatal("expected error")
			}
			if errors.Is(err, command.ErrTypeUnknown) {
				t.Fatalf("expected payload validation error, got %v", err)
			}
		})
	}
}
