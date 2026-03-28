package charactertransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestPatchCharacterProfile_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.PatchCharacterProfile(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_MissingCampaignId(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_MissingCharacterId(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_CampaignNotFound(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestPatchCharacterProfile_ProfileNotFound(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestPatchCharacterProfile_CompletedCampaignDisallowed(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.CompletedCampaignRecord("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	ts.Participant = characterManagerParticipantStore("c1")

	svc := NewService(ts.build())
	ctx := requestctx.WithParticipantID(context.Background(), "manager-1")
	_, err := svc.PatchCharacterProfile(ctx, &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10}},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestPatchCharacterProfile_DeniesMissingIdentity(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Participant = characterManagerParticipantStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		commandids.DaggerheartCharacterProfileReplace: {
			Decision: command.Accept(testDaggerheartProfileReplacedEvent(
				t,
				now,
				"c1",
				"ch1",
				event.ActorTypeParticipant,
				"",
				testDaggerheartProfile(func(profile *daggerheartstate.CharacterProfile) {
					profile.HpMax = 10
				}),
			)),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())

	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10}},
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestPatchCharacterProfile_DeniesMemberWhenNotOwner(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	now := time.Date(2026, 2, 20, 18, 20, 0, 0, time.UTC)

	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1":     gametest.MemberParticipantRecord("c1", "member-1"),
		"member-owner": gametest.MemberParticipantRecord("c1", "member-owner"),
	}
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", OwnerParticipantID: "member-owner", Name: "Hero", Kind: character.KindPC},
	}
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 6, StressMax: 6},
	}
	ts.Event.Events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "member-owner",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","name":"Hero","kind":"pc","owner_participant_id":"member-owner"}`),
		},
	}
	ts.Event.NextSeq["c1"] = 2

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		commandids.DaggerheartCharacterProfileReplace: {
			Decision: command.Accept(testDaggerheartProfileReplacedEvent(
				t,
				now,
				"c1",
				"ch1",
				event.ActorTypeParticipant,
				"member-1",
				testDaggerheartProfile(func(profile *daggerheartstate.CharacterProfile) {
					profile.HpMax = 10
				}),
			)),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	_, err := svc.PatchCharacterProfile(requestctx.WithParticipantID(context.Background(), "member-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10},
		},
	})
	assertStatusCode(t, err, codes.PermissionDenied)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestPatchCharacterProfile_NegativeHpMax(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12},
	}

	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: -1}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_ZeroHpMaxNoChange(t *testing.T) {
	// In proto3 patch semantics, HpMax=0 means "don't change" since 0 is the default value.
	// The original HpMax should be preserved.
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Participant = characterManagerParticipantStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		commandids.DaggerheartCharacterProfileReplace: {
			Decision: command.Accept(testDaggerheartProfileReplacedEvent(
				t,
				now,
				"c1",
				"ch1",
				event.ActorTypeSystem,
				"",
				testDaggerheartProfile(func(profile *daggerheartstate.CharacterProfile) {
					profile.HpMax = 12
					profile.StressMax = 6
				}),
			)),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	resp, err := svc.PatchCharacterProfile(requestctx.WithParticipantID(context.Background(), "manager-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 0}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterProfile returned error: %v", err)
	}
	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetHpMax() != 12 {
		t.Errorf("Profile HpMax = %d, want %d (unchanged)", dh.GetHpMax(), 12)
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != daggerheartpayload.EventTypeCharacterProfileReplaced {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, daggerheartpayload.EventTypeCharacterProfileReplaced)
	}
}

func TestPatchCharacterProfile_Success(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Participant = characterManagerParticipantStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6, Evasion: 10, MajorThreshold: 5, SevereThreshold: 10},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		commandids.DaggerheartCharacterProfileReplace: {
			Decision: command.Accept(testDaggerheartProfileReplacedEvent(
				t,
				now,
				"c1",
				"ch1",
				event.ActorTypeSystem,
				"",
				testDaggerheartProfile(func(profile *daggerheartstate.CharacterProfile) {
					profile.HpMax = 10
					profile.StressMax = 8
					profile.Evasion = 10
					profile.MajorThreshold = 5
					profile.SevereThreshold = 10
				}),
			)),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	resp, err := svc.PatchCharacterProfile(requestctx.WithParticipantID(context.Background(), "manager-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10, StressMax: wrapperspb.Int32(8)}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterProfile returned error: %v", err)
	}
	if resp.Profile == nil {
		t.Fatal("PatchCharacterProfile response has nil profile")
	}
	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetHpMax() != 10 {
		t.Errorf("Profile HpMax = %d, want %d", dh.GetHpMax(), 10)
	}
	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetStressMax().GetValue() != 8 {
		t.Errorf("Profile StressMax = %d, want %d", dh.GetStressMax().GetValue(), 8)
	}

	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetEvasion().GetValue() != 10 {
		t.Errorf("Profile Evasion = %d, want %d (unchanged)", dh.GetEvasion().GetValue(), 10)
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != daggerheartpayload.EventTypeCharacterProfileReplaced {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, daggerheartpayload.EventTypeCharacterProfileReplaced)
	}
}

func TestPatchCharacterProfile_SynthesizesDefaultsWhenProfileMissing(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Participant = characterManagerParticipantStore("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", OwnerParticipantID: "manager-1", Name: "Hero", Kind: character.KindPC},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		commandids.DaggerheartCharacterProfileReplace: {
			Decision: command.Accept(testDaggerheartProfileReplacedEvent(
				t,
				now,
				"c1",
				"ch1",
				event.ActorTypeSystem,
				"",
				testDaggerheartProfile(func(profile *daggerheartstate.CharacterProfile) {
					profile.HpMax = 10
				}),
			)),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	resp, err := svc.PatchCharacterProfile(requestctx.WithParticipantID(context.Background(), "manager-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterProfile returned error: %v", err)
	}
	if resp.Profile == nil || resp.Profile.GetDaggerheart() == nil {
		t.Fatal("PatchCharacterProfile response has nil daggerheart profile")
	}
	if got := resp.Profile.GetDaggerheart().GetHpMax(); got != 10 {
		t.Fatalf("profile hp_max = %d, want 10", got)
	}
	if got := resp.Profile.GetDaggerheart().GetStressMax().GetValue(); got != 6 {
		t.Fatalf("profile stress_max = %d, want default 6", got)
	}
}

func TestPatchCharacterProfile_UsesDomainEngine(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Participant = characterManagerParticipantStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6, Evasion: 10, MajorThreshold: 5, SevereThreshold: 10},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		commandids.DaggerheartCharacterProfileReplace: {
			Decision: command.Accept(testDaggerheartProfileReplacedEvent(
				t,
				now,
				"c1",
				"ch1",
				event.ActorTypeSystem,
				"",
				testDaggerheartProfile(func(profile *daggerheartstate.CharacterProfile) {
					profile.HpMax = 10
					profile.StressMax = 8
					profile.Evasion = 10
					profile.MajorThreshold = 5
					profile.SevereThreshold = 10
				}),
			)),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	resp, err := svc.PatchCharacterProfile(requestctx.WithParticipantID(context.Background(), "manager-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10, StressMax: wrapperspb.Int32(8)}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterProfile returned error: %v", err)
	}
	if resp.Profile == nil {
		t.Fatal("PatchCharacterProfile response has nil profile")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != commandids.DaggerheartCharacterProfileReplace {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, commandids.DaggerheartCharacterProfileReplace)
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != daggerheartpayload.EventTypeCharacterProfileReplaced {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, daggerheartpayload.EventTypeCharacterProfileReplaced)
	}
}

func TestPatchCharacterProfile_RejectsCreationWorkflowFields(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Participant = characterManagerParticipantStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {
			CampaignID:  "c1",
			CharacterID: "ch1",
			HpMax:       12,
			StressMax:   6,
		},
	}
	domain := &fakeDomainEngine{}

	svc := NewService(ts.withDomain(domain).build())

	_, err := svc.PatchCharacterProfile(requestctx.WithParticipantID(context.Background(), "manager-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{
			Agility: wrapperspb.Int32(3),
		}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDaggerheartExperiencesToProto(t *testing.T) {
	result := DaggerheartExperiencesToProto(nil)
	if result != nil {
		t.Fatalf("expected nil for nil input, got %v", result)
	}
	result = DaggerheartExperiencesToProto([]projectionstore.DaggerheartExperience{})
	if result != nil {
		t.Fatalf("expected nil for empty input, got %v", result)
	}

	result = DaggerheartExperiencesToProto([]projectionstore.DaggerheartExperience{
		{Name: "Stealth", Modifier: 3},
		{Name: "Insight", Modifier: -1},
	})
	if len(result) != 2 {
		t.Fatalf("expected 2 experiences, got %d", len(result))
	}
	if result[0].GetName() != "Stealth" || result[0].GetModifier() != 3 {
		t.Fatalf("experience 0 mismatch: %v", result[0])
	}
	if result[1].GetName() != "Insight" || result[1].GetModifier() != -1 {
		t.Fatalf("experience 1 mismatch: %v", result[1])
	}
}

func TestPatchCharacterProfile_HpMaxTooHigh(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 13}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_StressMaxTooHigh(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{StressMax: wrapperspb.Int32(13)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_NegativeEvasion(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{Evasion: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_NegativeMajorThreshold(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{MajorThreshold: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_NegativeSevereThreshold(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{SevereThreshold: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_NegativeProficiency(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{Proficiency: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_RequiresDomainEngine(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Participant = characterManagerParticipantStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12},
	}

	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(requestctx.WithParticipantID(context.Background(), "manager-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10}},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestPatchCharacterProfile_NegativeArmorScore(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{ArmorScore: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_ArmorMaxTooHigh(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{ArmorMax: wrapperspb.Int32(13)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_NegativeArmorMax(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{ArmorMax: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_EmptyExperienceName(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{
			Experiences: []*daggerheartv1.DaggerheartExperience{{Name: "", Modifier: 1}},
		}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_NegativeStressMax(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign = activeCampaignStore("c1")
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}

	svc := NewService(ts.build())
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{StressMax: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
