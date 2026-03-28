package campaigntransport

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/testclients"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestCreateCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.CreateCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_MissingSystem(t *testing.T) {
	svc := NewCampaignService(newTestDeps().build())
	_, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		GmMode: statev1.GmMode_HUMAN,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_EmptyName(t *testing.T) {
	svc := NewCampaignService(newTestDeps().build())
	_, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:   "",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_InvalidGmMode(t *testing.T) {
	svc := NewCampaignService(newTestDeps().build())
	_, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode(99),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_MissingGmModeDefaultsToAI(t *testing.T) {
	ts := newTestDeps()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{
		store: ts.Event,
		resultsByType: map[command.Type]engine.Result{
			handler.CommandTypeCampaignCreateWithParticipants: {
				Decision: command.Accept(
					event.Event{
						CampaignID:  "campaign-123",
						Type:        event.Type("campaign.created"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						EntityType:  "campaign",
						EntityID:    "campaign-123",
						PayloadJSON: []byte(`{"name":"Test Campaign","locale":"en-US","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_AI","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
					},
					event.Event{
						CampaignID:  "campaign-123",
						Type:        event.Type("participant.joined"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						EntityType:  "participant",
						EntityID:    "participant-owner",
						PayloadJSON: []byte(`{"participant_id":"participant-owner","user_id":"user-123","name":"Owner","role":"PLAYER","controller":"HUMAN","campaign_access":"OWNER"}`),
					},
					event.Event{
						CampaignID:  "campaign-123",
						Type:        event.Type("participant.joined"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						EntityType:  "participant",
						EntityID:    "participant-ai",
						PayloadJSON: []byte(`{"participant_id":"participant-ai","name":"Oracle","role":"GM","controller":"AI","campaign_access":"MANAGER","pronouns":"it/its"}`),
					},
				),
			},
		},
	}

	svc := newTestCampaignService(
		withAuthClient(ts.withDomain(domain).build(), &testclients.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "owner"}}),
		runtimekit.FixedClock(now),
		runtimekit.FixedSequenceIDGenerator("campaign-123", "participant-owner", "participant-ai"),
	)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123"))
	resp, err := svc.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	})
	if err != nil {
		t.Fatalf("CreateCampaign returned error: %v", err)
	}
	if resp.GetCampaign().GetGmMode() != statev1.GmMode_AI {
		t.Fatalf("Campaign GmMode = %v, want %v", resp.GetCampaign().GetGmMode(), statev1.GmMode_AI)
	}
	if resp.GetOwnerParticipant().GetRole() != statev1.ParticipantRole_PLAYER {
		t.Fatalf("OwnerParticipant Role = %v, want %v", resp.GetOwnerParticipant().GetRole(), statev1.ParticipantRole_PLAYER)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("domain command count = %d, want %d", len(domain.commands), 1)
	}
	if domain.commands[0].Type != handler.CommandTypeCampaignCreateWithParticipants {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, handler.CommandTypeCampaignCreateWithParticipants)
	}
	var workflowPayload campaign.CreateWithParticipantsPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &workflowPayload); err != nil {
		t.Fatalf("decode create workflow payload: %v", err)
	}
	if workflowPayload.Campaign.GmMode != statev1.GmMode_AI.String() {
		t.Fatalf("create payload gm mode = %q, want %q", workflowPayload.Campaign.GmMode, statev1.GmMode_AI.String())
	}
	if len(workflowPayload.Participants) != 2 {
		t.Fatalf("participant payload count = %d, want %d", len(workflowPayload.Participants), 2)
	}
}

func TestCreateCampaign_MissingCreatorUserID(t *testing.T) {
	svc := NewCampaignService(withAuthClient(newTestDeps().build(), &testclients.FakeAuthClient{}))
	_, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_AllowsOwnerlessPublicStarterTemplate(t *testing.T) {
	ts := newTestDeps()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{
		store: ts.Event,
		resultsByType: map[command.Type]engine.Result{
			handler.CommandTypeCampaignCreateWithParticipants: {
				Decision: command.Accept(
					event.Event{
						CampaignID:  "campaign-123",
						Type:        event.Type("campaign.created"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						EntityType:  "campaign",
						EntityID:    "campaign-123",
						PayloadJSON: []byte(`{"name":"Starter Template","locale":"en-US","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_AI","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"ownerless template"}`),
					},
					event.Event{
						CampaignID:  "campaign-123",
						Type:        event.Type("participant.joined"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						EntityType:  "participant",
						EntityID:    "participant-owner",
						PayloadJSON: []byte(`{"participant_id":"participant-owner","name":"Unknown","role":"PLAYER","controller":"HUMAN","campaign_access":"OWNER","pronouns":"they/them"}`),
					},
					event.Event{
						CampaignID:  "campaign-123",
						Type:        event.Type("participant.joined"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						EntityType:  "participant",
						EntityID:    "participant-ai",
						PayloadJSON: []byte(`{"participant_id":"participant-ai","name":"Oracle","role":"GM","controller":"AI","campaign_access":"MANAGER","pronouns":"it/its"}`),
					},
				),
			},
		},
	}
	svc := newTestCampaignService(
		ts.withDomain(domain).build(),
		runtimekit.FixedClock(now),
		runtimekit.FixedSequenceIDGenerator("campaign-123", "participant-owner", "participant-ai"),
	)

	resp, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:         "Starter Template",
		System:       commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:       statev1.GmMode_AI,
		Intent:       statev1.CampaignIntent_STARTER,
		AccessPolicy: statev1.CampaignAccessPolicy_PUBLIC,
		ThemePrompt:  "ownerless template",
	})
	if err != nil {
		t.Fatalf("CreateCampaign returned error: %v", err)
	}
	if resp.GetOwnerParticipant().GetUserId() != "" {
		t.Fatalf("OwnerParticipant UserId = %q, want empty", resp.GetOwnerParticipant().GetUserId())
	}

	var workflowPayload campaign.CreateWithParticipantsPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &workflowPayload); err != nil {
		t.Fatalf("decode create workflow payload: %v", err)
	}
	if got := workflowPayload.Participants[0].UserID.String(); got != "" {
		t.Fatalf("owner payload user_id = %q, want empty", got)
	}
	if got := workflowPayload.Participants[0].CampaignAccess; got != "OWNER" {
		t.Fatalf("owner payload campaign_access = %q, want OWNER", got)
	}
	if got := workflowPayload.Participants[0].Pronouns; got != sharedpronouns.PronounTheyThem {
		t.Fatalf("owner payload pronouns = %q, want %q", got, sharedpronouns.PronounTheyThem)
	}
}

func TestCreateCampaign_RequiresDomainEngine(t *testing.T) {
	svc := newTestCampaignService(
		newTestDeps().build(),
		runtimekit.FixedClock(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)),
		runtimekit.FixedSequenceIDGenerator("campaign-123", "participant-123"),
	)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123"))
	_, err := svc.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateCampaign_Success(t *testing.T) {
	ts := newTestDeps()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{
		store: ts.Event,
		resultsByType: map[command.Type]engine.Result{
			handler.CommandTypeCampaignCreateWithParticipants: {
				Decision: command.Accept(
					event.Event{
						CampaignID:  "campaign-123",
						Type:        event.Type("campaign.created"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						EntityType:  "campaign",
						EntityID:    "campaign-123",
						PayloadJSON: []byte(`{"name":"Test Campaign","locale":"en-US","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
					},
					event.Event{
						CampaignID:  "campaign-123",
						Type:        event.Type("participant.joined"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						EntityType:  "participant",
						EntityID:    "participant-123",
						PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"Mysterious Person","role":"GM","controller":"HUMAN","campaign_access":"OWNER"}`),
					},
				),
			},
		},
	}
	svc := newTestCampaignService(
		withAuthClient(ts.withDomain(domain).build(), &testclients.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "owner"}}),
		runtimekit.FixedClock(now),
		runtimekit.FixedSequenceIDGenerator("campaign-123", "participant-123"),
	)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123"))
	resp, err := svc.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:        "Test Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      statev1.GmMode_HUMAN,
		ThemePrompt: "A dark fantasy adventure",
	})
	if err != nil {
		t.Fatalf("CreateCampaign returned error: %v", err)
	}

	if resp.Campaign == nil {
		t.Fatal("CreateCampaign response has nil campaign")
	}
	if resp.Campaign.Id != "campaign-123" {
		t.Errorf("Campaign ID = %q, want %q", resp.Campaign.Id, "campaign-123")
	}
	if resp.Campaign.Name != "Test Campaign" {
		t.Errorf("Campaign Name = %q, want %q", resp.Campaign.Name, "Test Campaign")
	}
	if resp.Campaign.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Errorf("Campaign System = %v, want %v", resp.Campaign.System, commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	}
	if resp.Campaign.Status != statev1.CampaignStatus_DRAFT {
		t.Errorf("Campaign Status = %v, want %v", resp.Campaign.Status, statev1.CampaignStatus_DRAFT)
	}
	if resp.Campaign.GmMode != statev1.GmMode_HUMAN {
		t.Errorf("Campaign GmMode = %v, want %v", resp.Campaign.GmMode, statev1.GmMode_HUMAN)
	}
	if resp.Campaign.ThemePrompt != "A dark fantasy adventure" {
		t.Errorf("Campaign ThemePrompt = %q, want %q", resp.Campaign.ThemePrompt, "A dark fantasy adventure")
	}
	if resp.OwnerParticipant == nil {
		t.Fatal("CreateCampaign response has nil owner participant")
	}
	if resp.OwnerParticipant.UserId != "user-123" {
		t.Errorf("OwnerParticipant UserId = %q, want %q", resp.OwnerParticipant.UserId, "user-123")
	}
	if got := len(ts.Event.Events["campaign-123"]); got != 2 {
		t.Fatalf("expected 2 events, got %d", got)
	}
	if ts.Event.Events["campaign-123"][0].Type != event.Type("campaign.created") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["campaign-123"][0].Type, event.Type("campaign.created"))
	}
	if ts.Event.Events["campaign-123"][1].Type != event.Type("participant.joined") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["campaign-123"][1].Type, event.Type("participant.joined"))
	}

	// Verify persisted
	stored, err := ts.Campaign.Get(context.Background(), "campaign-123")
	if err != nil {
		t.Fatalf("Campaign not persisted: %v", err)
	}
	if stored.Name != "Test Campaign" {
		t.Errorf("Stored campaign Name = %q, want %q", stored.Name, "Test Campaign")
	}
}

func TestCreateCampaign_UsesDomainEngine(t *testing.T) {
	ts := newTestDeps()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{
		store: ts.Event,
		resultsByType: map[command.Type]engine.Result{
			handler.CommandTypeCampaignCreateWithParticipants: {
				Decision: command.Accept(
					event.Event{
						CampaignID:  "campaign-123",
						Type:        event.Type("campaign.created"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						EntityType:  "campaign",
						EntityID:    "campaign-123",
						PayloadJSON: []byte(`{"name":"Test Campaign","locale":"en-US","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
					},
					event.Event{
						CampaignID:  "campaign-123",
						Type:        event.Type("participant.joined"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						EntityType:  "participant",
						EntityID:    "participant-123",
						PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"Owner","role":"GM","controller":"HUMAN","campaign_access":"OWNER"}`),
					},
				),
			},
		},
	}

	svc := newTestCampaignService(
		withAuthClient(ts.withDomain(domain).build(), &testclients.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "owner"}}),
		runtimekit.FixedClock(now),
		runtimekit.FixedSequenceIDGenerator("campaign-123", "participant-123"),
	)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123"))
	resp, err := svc.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:         "Test Campaign",
		Locale:       commonv1.Locale_LOCALE_EN_US,
		System:       commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:       statev1.GmMode_HUMAN,
		Intent:       statev1.CampaignIntent_STARTER,
		AccessPolicy: statev1.CampaignAccessPolicy_PUBLIC,
		ThemePrompt:  "A dark fantasy adventure",
	})
	if err != nil {
		t.Fatalf("CreateCampaign returned error: %v", err)
	}
	if resp.Campaign == nil {
		t.Fatal("CreateCampaign response has nil campaign")
	}
	if resp.Campaign.ThemePrompt != "A dark fantasy adventure" {
		t.Fatalf("Campaign ThemePrompt = %q, want %q", resp.Campaign.ThemePrompt, "A dark fantasy adventure")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != handler.CommandTypeCampaignCreateWithParticipants {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, handler.CommandTypeCampaignCreateWithParticipants)
	}
	if got := len(ts.Event.Events["campaign-123"]); got != 2 {
		t.Fatalf("expected 2 events, got %d", got)
	}
	if ts.Event.Events["campaign-123"][0].Type != event.Type("campaign.created") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["campaign-123"][0].Type, event.Type("campaign.created"))
	}
	if ts.Event.Events["campaign-123"][1].Type != event.Type("participant.joined") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["campaign-123"][1].Type, event.Type("participant.joined"))
	}
}

func TestCreateCampaign_ModeSpecificParticipantBootstrap(t *testing.T) {
	tests := []struct {
		name               string
		gmMode             statev1.GmMode
		wantOwnerRole      string
		wantOwnerProtoRole statev1.ParticipantRole
	}{
		{
			name:               "AI mode creates owner player and AI gm manager",
			gmMode:             statev1.GmMode_AI,
			wantOwnerRole:      "PLAYER",
			wantOwnerProtoRole: statev1.ParticipantRole_PLAYER,
		},
		{
			name:               "HYBRID mode creates owner gm and AI gm manager",
			gmMode:             statev1.GmMode_HYBRID,
			wantOwnerRole:      "GM",
			wantOwnerProtoRole: statev1.ParticipantRole_GM,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTestDeps()
			now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

			campaignPayload := fmt.Sprintf(`{"name":"Test Campaign","locale":"en-US","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"%s","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`,
				tc.gmMode.String())
			ownerJoinResultPayload := fmt.Sprintf(`{"participant_id":"participant-owner","user_id":"user-123","name":"Owner","role":"%s","controller":"HUMAN","campaign_access":"OWNER"}`,
				tc.wantOwnerRole)
			domain := &fakeDomainEngine{
				store: ts.Event,
				resultsByType: map[command.Type]engine.Result{
					handler.CommandTypeCampaignCreateWithParticipants: {
						Decision: command.Accept(
							event.Event{
								CampaignID:  "campaign-123",
								Type:        event.Type("campaign.created"),
								Timestamp:   now,
								ActorType:   event.ActorTypeSystem,
								EntityType:  "campaign",
								EntityID:    "campaign-123",
								PayloadJSON: []byte(campaignPayload),
							},
							event.Event{
								CampaignID:  "campaign-123",
								Type:        event.Type("participant.joined"),
								Timestamp:   now,
								ActorType:   event.ActorTypeSystem,
								EntityType:  "participant",
								EntityID:    "participant-owner",
								PayloadJSON: []byte(ownerJoinResultPayload),
							},
							event.Event{
								CampaignID:  "campaign-123",
								Type:        event.Type("participant.joined"),
								Timestamp:   now,
								ActorType:   event.ActorTypeSystem,
								EntityType:  "participant",
								EntityID:    "participant-ai",
								PayloadJSON: []byte(`{"participant_id":"participant-ai","name":"Oracle","role":"GM","controller":"AI","campaign_access":"MANAGER","pronouns":"it/its"}`),
							},
						),
					},
				},
			}
			svc := newTestCampaignService(
				withAuthClient(ts.withDomain(domain).build(), &testclients.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "owner"}}),
				runtimekit.FixedClock(now),
				runtimekit.FixedSequenceIDGenerator("campaign-123", "participant-owner", "participant-ai"),
			)

			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123"))
			resp, err := svc.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
				Name:        "Test Campaign",
				System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
				GmMode:      tc.gmMode,
				ThemePrompt: "A dark fantasy adventure",
			})
			if err != nil {
				t.Fatalf("CreateCampaign returned error: %v", err)
			}
			if resp.OwnerParticipant == nil {
				t.Fatal("CreateCampaign response has nil owner participant")
			}
			if resp.OwnerParticipant.Id != "participant-owner" {
				t.Fatalf("OwnerParticipant Id = %q, want %q", resp.OwnerParticipant.Id, "participant-owner")
			}
			if resp.OwnerParticipant.Role != tc.wantOwnerProtoRole {
				t.Fatalf("OwnerParticipant Role = %v, want %v", resp.OwnerParticipant.Role, tc.wantOwnerProtoRole)
			}
			if resp.OwnerParticipant.CampaignAccess != statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER {
				t.Fatalf("OwnerParticipant CampaignAccess = %v, want OWNER", resp.OwnerParticipant.CampaignAccess)
			}

			if len(domain.commands) != 1 {
				t.Fatalf("domain command count = %d, want %d", len(domain.commands), 1)
			}
			if domain.commands[0].Type != handler.CommandTypeCampaignCreateWithParticipants {
				t.Fatalf("command type = %s, want %s", domain.commands[0].Type, handler.CommandTypeCampaignCreateWithParticipants)
			}

			var workflowPayload campaign.CreateWithParticipantsPayload
			if err := json.Unmarshal(domain.commands[0].PayloadJSON, &workflowPayload); err != nil {
				t.Fatalf("decode create workflow payload: %v", err)
			}
			if len(workflowPayload.Participants) != 2 {
				t.Fatalf("participant payload count = %d, want %d", len(workflowPayload.Participants), 2)
			}

			ownerPayload := workflowPayload.Participants[0]
			if ownerPayload.ParticipantID != "participant-owner" {
				t.Fatalf("owner payload participant_id = %q, want %q", ownerPayload.ParticipantID, "participant-owner")
			}
			if ownerPayload.Role != tc.wantOwnerRole {
				t.Fatalf("owner payload role = %q, want %q", ownerPayload.Role, tc.wantOwnerRole)
			}
			if ownerPayload.Controller != "HUMAN" {
				t.Fatalf("owner payload controller = %q, want %q", ownerPayload.Controller, "HUMAN")
			}
			if ownerPayload.CampaignAccess != "OWNER" {
				t.Fatalf("owner payload campaign_access = %q, want %q", ownerPayload.CampaignAccess, "OWNER")
			}

			aiPayload := workflowPayload.Participants[1]
			if aiPayload.ParticipantID != "participant-ai" {
				t.Fatalf("ai payload participant_id = %q, want %q", aiPayload.ParticipantID, "participant-ai")
			}
			if aiPayload.UserID != "" {
				t.Fatalf("ai payload user_id = %q, want empty", aiPayload.UserID)
			}
			if aiPayload.Name != "Oracle" {
				t.Fatalf("ai payload name = %q, want %q", aiPayload.Name, "Oracle")
			}
			if aiPayload.Role != "GM" {
				t.Fatalf("ai payload role = %q, want %q", aiPayload.Role, "GM")
			}
			if aiPayload.Controller != "AI" {
				t.Fatalf("ai payload controller = %q, want %q", aiPayload.Controller, "AI")
			}
			if aiPayload.Pronouns != "it/its" {
				t.Fatalf("ai payload pronouns = %q, want %q", aiPayload.Pronouns, "it/its")
			}
			if aiPayload.CampaignAccess != "MANAGER" {
				t.Fatalf("ai payload campaign_access = %q, want %q", aiPayload.CampaignAccess, "MANAGER")
			}
		})
	}
}

func TestCreateCampaign_OwnerParticipantHydratesFromSocialProfile(t *testing.T) {
	ts := newTestDeps()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		handler.CommandTypeCampaignCreateWithParticipants: {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("campaign.created"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "campaign",
					EntityID:    "campaign-123",
					PayloadJSON: []byte(`{"name":"Test Campaign","locale":"en-US","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
				},
				event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("participant.joined"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "participant",
					EntityID:    "participant-123",
					PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"Profile Name","role":"GM","controller":"HUMAN","campaign_access":"OWNER","avatar_set_id":"creatures-v1","avatar_asset_id":"social-avatar","pronouns":"they/them"}`),
				},
			),
		},
	}}
	socialClient := &testclients.FakeSocialClient{Profile: &socialv1.UserProfile{
		UserId:        "user-123",
		Name:          "Profile Name",
		Pronouns:      sharedpronouns.ToProto("they/them"),
		AvatarSetId:   "creatures-v1",
		AvatarAssetId: "social-avatar",
	}}

	stores := ts.withDomain(domain).build()
	stores.Social = socialClient
	svc := newTestCampaignService(
		stores,
		runtimekit.FixedClock(now),
		runtimekit.FixedSequenceIDGenerator("campaign-123", "participant-123"),
	)

	_, err := svc.CreateCampaign(metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123")), &statev1.CreateCampaignRequest{
		Name:        "Test Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      statev1.GmMode_HUMAN,
		ThemePrompt: "A dark fantasy adventure",
	})
	if err != nil {
		t.Fatalf("CreateCampaign returned error: %v", err)
	}
	if socialClient.GetUserProfileCalls != 1 {
		t.Fatalf("GetUserProfile calls = %d, want %d", socialClient.GetUserProfileCalls, 1)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}

	var payload campaign.CreateWithParticipantsPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode create workflow payload: %v", err)
	}
	if len(payload.Participants) != 1 {
		t.Fatalf("participant payload count = %d, want %d", len(payload.Participants), 1)
	}
	ownerPayload := payload.Participants[0]
	if ownerPayload.Name != "Profile Name" {
		t.Fatalf("payload name = %q, want %q", ownerPayload.Name, "Profile Name")
	}
	if ownerPayload.AvatarSetID != "creatures-v1" {
		t.Fatalf("payload avatar_set_id = %q, want %q", ownerPayload.AvatarSetID, "creatures-v1")
	}
	if ownerPayload.AvatarAssetID != "social-avatar" {
		t.Fatalf("payload avatar_asset_id = %q, want %q", ownerPayload.AvatarAssetID, "social-avatar")
	}
	if ownerPayload.Pronouns != "they/them" {
		t.Fatalf("payload pronouns = %q, want %q", ownerPayload.Pronouns, "they/them")
	}
}

func TestCreateCampaign_OwnerParticipantFallsBackToDefaultPronounsWhenSocialPronounsMissing(t *testing.T) {
	ts := newTestDeps()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		handler.CommandTypeCampaignCreateWithParticipants: {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("campaign.created"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "campaign",
					EntityID:    "campaign-123",
					PayloadJSON: []byte(`{"name":"Test Campaign","locale":"en-US","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
				},
				event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("participant.joined"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "participant",
					EntityID:    "participant-123",
					PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"Profile Name","role":"GM","controller":"HUMAN","campaign_access":"OWNER","avatar_set_id":"creatures-v1","avatar_asset_id":"social-avatar","pronouns":"they/them"}`),
				},
			),
		},
	}}
	socialClient := &testclients.FakeSocialClient{Profile: &socialv1.UserProfile{
		UserId:        "user-123",
		Name:          "Profile Name",
		AvatarSetId:   "creatures-v1",
		AvatarAssetId: "social-avatar",
	}}

	stores := ts.withDomain(domain).build()
	stores.Social = socialClient
	svc := newTestCampaignService(
		stores,
		runtimekit.FixedClock(now),
		runtimekit.FixedSequenceIDGenerator("campaign-123", "participant-123"),
	)

	_, err := svc.CreateCampaign(metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123")), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	if err != nil {
		t.Fatalf("CreateCampaign returned error: %v", err)
	}

	var payload campaign.CreateWithParticipantsPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode create workflow payload: %v", err)
	}
	if len(payload.Participants) != 1 {
		t.Fatalf("participant payload count = %d, want %d", len(payload.Participants), 1)
	}
	if payload.Participants[0].Pronouns != "they/them" {
		t.Fatalf("payload pronouns = %q, want %q", payload.Participants[0].Pronouns, "they/them")
	}
}

func TestCreateCampaign_OwnerParticipantFallsBackToAuthUsernameWithoutSocialProfile(t *testing.T) {
	ts := newTestDeps()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		handler.CommandTypeCampaignCreateWithParticipants: {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("campaign.created"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "campaign",
					EntityID:    "campaign-123",
					PayloadJSON: []byte(`{"name":"Test Campaign","locale":"en-US","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
				},
				event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("participant.joined"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "participant",
					EntityID:    "participant-123",
					PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"owner-handle","role":"GM","controller":"HUMAN","campaign_access":"OWNER"}`),
				},
			),
		},
	}}
	authClient := &testclients.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "owner-handle"}}

	svc := newTestCampaignService(
		withAuthClient(ts.withDomain(domain).build(), authClient),
		runtimekit.FixedClock(now),
		runtimekit.FixedSequenceIDGenerator("campaign-123", "participant-123"),
	)

	_, err := svc.CreateCampaign(metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123")), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	if err != nil {
		t.Fatalf("CreateCampaign returned error: %v", err)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}

	var payload campaign.CreateWithParticipantsPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode create workflow payload: %v", err)
	}
	if len(payload.Participants) != 1 {
		t.Fatalf("participant payload count = %d, want %d", len(payload.Participants), 1)
	}
	ownerPayload := payload.Participants[0]
	if ownerPayload.Name != "owner-handle" {
		t.Fatalf("payload name = %q, want %q", ownerPayload.Name, "owner-handle")
	}
	if ownerPayload.AvatarSetID != "" {
		t.Fatalf("payload avatar_set_id = %q, want empty", ownerPayload.AvatarSetID)
	}
	if ownerPayload.AvatarAssetID != "" {
		t.Fatalf("payload avatar_asset_id = %q, want empty", ownerPayload.AvatarAssetID)
	}
	if ownerPayload.Pronouns != "they/them" {
		t.Fatalf("payload pronouns = %q, want %q", ownerPayload.Pronouns, "they/them")
	}
	if authClient.LastGetUserRequest == nil || authClient.LastGetUserRequest.GetUserId() != "user-123" {
		t.Fatalf("GetUser request = %#v, want user-123", authClient.LastGetUserRequest)
	}
}

func TestCreateCampaign_OwnerParticipantFallsBackToAuthUsernameForLocale(t *testing.T) {
	ts := newTestDeps()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		handler.CommandTypeCampaignCreateWithParticipants: {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("campaign.created"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "campaign",
					EntityID:    "campaign-123",
					PayloadJSON: []byte(`{"name":"Test Campaign","locale":"pt-BR","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
				},
				event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("participant.joined"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "participant",
					EntityID:    "participant-123",
					PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"apelido","role":"GM","controller":"HUMAN","campaign_access":"OWNER"}`),
				},
			),
		},
	}}
	authClient := &testclients.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "apelido"}}

	svc := newTestCampaignService(
		withAuthClient(ts.withDomain(domain).build(), authClient),
		runtimekit.FixedClock(now),
		runtimekit.FixedSequenceIDGenerator("campaign-123", "participant-123"),
	)

	_, err := svc.CreateCampaign(metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123")), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
		Locale: commonv1.Locale_LOCALE_PT_BR,
	})
	if err != nil {
		t.Fatalf("CreateCampaign returned error: %v", err)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}

	var payload campaign.CreateWithParticipantsPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode create workflow payload: %v", err)
	}
	if len(payload.Participants) != 1 {
		t.Fatalf("participant payload count = %d, want %d", len(payload.Participants), 1)
	}
	if payload.Participants[0].Name != "apelido" {
		t.Fatalf("payload name = %q, want %q", payload.Participants[0].Name, "apelido")
	}
	if authClient.LastGetUserRequest == nil || authClient.LastGetUserRequest.GetUserId() != "user-123" {
		t.Fatalf("GetUser request = %#v, want user-123", authClient.LastGetUserRequest)
	}
}

func TestCreateCampaign_AIUsesLocalizedNameAndOwnerFallsBackToAuthUsernameForLocale(t *testing.T) {
	ts := newTestDeps()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		handler.CommandTypeCampaignCreateWithParticipants: {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("campaign.created"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "campaign",
					EntityID:    "campaign-123",
					PayloadJSON: []byte(`{"name":"Test Campaign","locale":"pt-BR","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_AI","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
				},
				event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("participant.joined"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "participant",
					EntityID:    "participant-owner",
					PayloadJSON: []byte(`{"participant_id":"participant-owner","user_id":"user-123","name":"apelido","role":"PLAYER","controller":"HUMAN","campaign_access":"OWNER","pronouns":"they/them"}`),
				},
				event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("participant.joined"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "participant",
					EntityID:    "participant-ai",
					PayloadJSON: []byte(`{"participant_id":"participant-ai","name":"Oráculo","role":"GM","controller":"AI","campaign_access":"MANAGER","pronouns":"it/its"}`),
				},
			),
		},
	}}
	authClient := &testclients.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "apelido"}}

	svc := newTestCampaignService(
		withAuthClient(ts.withDomain(domain).build(), authClient),
		runtimekit.FixedClock(now),
		runtimekit.FixedSequenceIDGenerator("campaign-123", "participant-owner", "participant-ai"),
	)

	_, err := svc.CreateCampaign(metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123")), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_AI,
		Locale: commonv1.Locale_LOCALE_PT_BR,
	})
	if err != nil {
		t.Fatalf("CreateCampaign returned error: %v", err)
	}

	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}

	var workflowPayload campaign.CreateWithParticipantsPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &workflowPayload); err != nil {
		t.Fatalf("decode create workflow payload: %v", err)
	}
	if len(workflowPayload.Participants) != 2 {
		t.Fatalf("participant payload count = %d, want %d", len(workflowPayload.Participants), 2)
	}
	ownerPayload := workflowPayload.Participants[0]
	if ownerPayload.Name != "apelido" {
		t.Fatalf("owner payload name = %q, want %q", ownerPayload.Name, "apelido")
	}
	if ownerPayload.Role != "PLAYER" {
		t.Fatalf("owner payload role = %q, want %q", ownerPayload.Role, "PLAYER")
	}

	aiPayload := workflowPayload.Participants[1]
	if aiPayload.Name != "Oráculo" {
		t.Fatalf("ai payload name = %q, want %q", aiPayload.Name, "Oráculo")
	}
	if aiPayload.Controller != "AI" {
		t.Fatalf("ai payload controller = %q, want %q", aiPayload.Controller, "AI")
	}
	if authClient.LastGetUserRequest == nil || authClient.LastGetUserRequest.GetUserId() != "user-123" {
		t.Fatalf("GetUser request = %#v, want user-123", authClient.LastGetUserRequest)
	}
}
