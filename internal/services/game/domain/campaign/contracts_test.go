package campaign

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	testcontracts "github.com/louisbranch/fracturing.space/internal/services/game/domain/internaltest/contracts"
)

func TestRegisterRequiresRegistry(t *testing.T) {
	if err := RegisterCommands(nil); err == nil {
		t.Fatalf("expected error for nil command registry")
	}
	if err := RegisterEvents(nil); err == nil {
		t.Fatalf("expected error for nil event registry")
	}
}

func TestCampaignContractTypeLists(t *testing.T) {
	emittable := EmittableEventTypes()
	wantEmittable := []event.Type{
		EventTypeCreated,
		EventTypeUpdated,
		EventTypeAIBound,
		EventTypeAIUnbound,
		EventTypeAIAuthRotated,
		EventTypeForked,
	}
	if !testcontracts.EqualSlices(emittable, wantEmittable) {
		t.Fatalf("EmittableEventTypes() = %v, want %v", emittable, wantEmittable)
	}

	commands := DeciderHandledCommands()
	wantCommands := []command.Type{
		CommandTypeCreate,
		CommandTypeCreateWithParticipants,
		CommandTypeUpdate,
		CommandTypeAIBind,
		CommandTypeAIUnbind,
		CommandTypeAIAuthRotate,
		CommandTypeFork,
		CommandTypeEnd,
		CommandTypeArchive,
		CommandTypeRestore,
	}
	if !testcontracts.EqualSlices(commands, wantCommands) {
		t.Fatalf("DeciderHandledCommands() = %v, want %v", commands, wantCommands)
	}

	foldTypes := FoldHandledTypes()
	projectionTypes := ProjectionHandledTypes()
	if !testcontracts.EqualSlices(foldTypes, projectionTypes) {
		t.Fatalf("fold and projection types differ: fold=%v projection=%v", foldTypes, projectionTypes)
	}
}

func TestCampaignContractDeclarationsStayInParity(t *testing.T) {
	declaredCommandTypes := make([]command.Type, 0, len(campaignCommandContracts))
	for _, contract := range campaignCommandContracts {
		declaredCommandTypes = append(declaredCommandTypes, contract.definition.Type)
	}
	if testcontracts.HasDuplicates(declaredCommandTypes) {
		t.Fatalf("duplicate command declarations found: %v", declaredCommandTypes)
	}
	if !testcontracts.EqualSlices(DeciderHandledCommands(), declaredCommandTypes) {
		t.Fatalf("DeciderHandledCommands() = %v, want %v", DeciderHandledCommands(), declaredCommandTypes)
	}

	declaredEmittable := make([]event.Type, 0, len(campaignEventContracts))
	declaredProjection := make([]event.Type, 0, len(campaignEventContracts))
	for _, contract := range campaignEventContracts {
		if contract.emittable {
			declaredEmittable = append(declaredEmittable, contract.definition.Type)
		}
		if contract.projection {
			declaredProjection = append(declaredProjection, contract.definition.Type)
		}
	}
	if testcontracts.HasDuplicates(declaredEmittable) {
		t.Fatalf("duplicate emittable event declarations found: %v", declaredEmittable)
	}
	if testcontracts.HasDuplicates(declaredProjection) {
		t.Fatalf("duplicate projection event declarations found: %v", declaredProjection)
	}
	if !testcontracts.EqualSlices(EmittableEventTypes(), declaredEmittable) {
		t.Fatalf("EmittableEventTypes() = %v, want %v", EmittableEventTypes(), declaredEmittable)
	}
	if !testcontracts.EqualSlices(ProjectionHandledTypes(), declaredProjection) {
		t.Fatalf("ProjectionHandledTypes() = %v, want %v", ProjectionHandledTypes(), declaredProjection)
	}

	commands := command.NewRegistry()
	if err := RegisterCommands(commands); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if got, want := len(commands.ListDefinitions()), len(campaignCommandContracts); got != want {
		t.Fatalf("registered command definitions = %d, want %d", got, want)
	}

	events := event.NewRegistry()
	if err := RegisterEvents(events); err != nil {
		t.Fatalf("register events: %v", err)
	}
	if got, want := len(events.ListDefinitions()), len(campaignEventContracts); got != want {
		t.Fatalf("registered event definitions = %d, want %d", got, want)
	}
}

func TestNormalizeLabels(t *testing.T) {
	if got, ok := NormalizeStatus("CAMPAIGN_STATUS_ACTIVE"); !ok || got != StatusActive {
		t.Fatalf("NormalizeStatus = (%q, %v), want (%q, true)", got, ok, StatusActive)
	}
	if got, ok := NormalizeStatus("invalid"); ok || got != StatusUnspecified {
		t.Fatalf("NormalizeStatus invalid = (%q, %v), want (%q, false)", got, ok, StatusUnspecified)
	}

	if got, ok := NormalizeGmMode("GM_MODE_AI"); !ok || got != GmModeAI {
		t.Fatalf("NormalizeGmMode = (%q, %v), want (%q, true)", got, ok, GmModeAI)
	}
	if got, ok := NormalizeGmMode("invalid"); ok || got != GmModeUnspecified {
		t.Fatalf("NormalizeGmMode invalid = (%q, %v), want (%q, false)", got, ok, GmModeUnspecified)
	}

	if got, ok := NormalizeGameSystem("GAME_SYSTEM_DAGGERHEART"); !ok || got != GameSystemDaggerheart {
		t.Fatalf("NormalizeGameSystem = (%q, %v), want (%q, true)", got, ok, GameSystemDaggerheart)
	}
	if got, ok := NormalizeGameSystem("invalid"); ok || got != GameSystemUnspecified {
		t.Fatalf("NormalizeGameSystem invalid = (%q, %v), want (%q, false)", got, ok, GameSystemUnspecified)
	}

	if got := NormalizeIntent("CAMPAIGN_INTENT_SANDBOX"); got != IntentSandbox {
		t.Fatalf("NormalizeIntent = %q, want %q", got, IntentSandbox)
	}
	if got := NormalizeIntent("unknown"); got != IntentStandard {
		t.Fatalf("NormalizeIntent unknown = %q, want %q", got, IntentStandard)
	}

	if got := NormalizeAccessPolicy("CAMPAIGN_ACCESS_POLICY_PUBLIC"); got != AccessPolicyPublic {
		t.Fatalf("NormalizeAccessPolicy = %q, want %q", got, AccessPolicyPublic)
	}
	if got := NormalizeAccessPolicy("unknown"); got != AccessPolicyPrivate {
		t.Fatalf("NormalizeAccessPolicy unknown = %q, want %q", got, AccessPolicyPrivate)
	}
}

func TestNormalizeCreateInput(t *testing.T) {
	_, err := NormalizeCreateInput(CreateInput{
		Name:   "",
		System: GameSystemDaggerheart,
		GmMode: GmModeHuman,
	})
	if !errors.Is(err, ErrEmptyName) {
		t.Fatalf("expected ErrEmptyName, got %v", err)
	}

	_, err = NormalizeCreateInput(CreateInput{
		Name:   "Sunfall",
		System: GameSystemUnspecified,
		GmMode: GmModeHuman,
	})
	if !errors.Is(err, ErrInvalidGameSystem) {
		t.Fatalf("expected ErrInvalidGameSystem, got %v", err)
	}

	normalized, err := NormalizeCreateInput(CreateInput{
		Name:   "  Sunfall  ",
		System: GameSystemDaggerheart,
		GmMode: GmModeUnspecified,
	})
	if err != nil {
		t.Fatalf("NormalizeCreateInput: %v", err)
	}
	if normalized.Name != "Sunfall" {
		t.Fatalf("Name = %q, want Sunfall", normalized.Name)
	}
	if normalized.Locale != "en-US" {
		t.Fatalf("Locale = %s, want %s", normalized.Locale, "en-US")
	}
	if normalized.GmMode != GmModeAI {
		t.Fatalf("GmMode = %q, want %q", normalized.GmMode, GmModeAI)
	}
	if normalized.Intent != IntentStandard {
		t.Fatalf("Intent = %q, want %q", normalized.Intent, IntentStandard)
	}
	if normalized.AccessPolicy != AccessPolicyPrivate {
		t.Fatalf("AccessPolicy = %q, want %q", normalized.AccessPolicy, AccessPolicyPrivate)
	}
}

func TestIsStatusTransitionAllowed(t *testing.T) {
	if !IsStatusTransitionAllowed(StatusDraft, StatusActive) {
		t.Fatalf("expected draft->active to be allowed")
	}
	if IsStatusTransitionAllowed(StatusArchived, StatusCompleted) {
		t.Fatalf("expected archived->completed to be rejected")
	}
}

func TestFoldUpdatedRejectsCorruptPayload(t *testing.T) {
	_, err := Fold(State{}, event.Event{
		Type:        EventTypeUpdated,
		PayloadJSON: []byte(`{`),
	})
	if err == nil {
		t.Fatalf("expected fold error for corrupt update payload")
	}
}

func TestFoldForkedIsExplicitNoOp(t *testing.T) {
	initial := State{
		Created:     true,
		Name:        "Sunfall",
		GameSystem:  GameSystemDaggerheart,
		GmMode:      GmModeHuman,
		Status:      StatusActive,
		ThemePrompt: "Prompt",
	}
	updated, err := Fold(initial, event.Event{
		Type:        EventTypeForked,
		PayloadJSON: []byte(`{"parent_campaign_id":"camp-root","origin_campaign_id":"camp-root"}`),
	})
	if err != nil {
		t.Fatalf("Fold forked: %v", err)
	}
	if updated != initial {
		t.Fatalf("forked event should be projection no-op, got %+v vs %+v", updated, initial)
	}
}

func TestRegisterCommandsAndEvents_RejectUnknownTypes(t *testing.T) {
	commands := command.NewRegistry()
	if err := RegisterCommands(commands); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if _, err := commands.ValidateForDecision(command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.unknown"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{}`),
	}); !errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected ErrTypeUnknown, got %v", err)
	}

	events := event.NewRegistry()
	if err := RegisterEvents(events); err != nil {
		t.Fatalf("register events: %v", err)
	}
	if _, err := events.ValidateForAppend(event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.unknown"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: []byte(`{}`),
	}); !errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected ErrTypeUnknown, got %v", err)
	}
}
