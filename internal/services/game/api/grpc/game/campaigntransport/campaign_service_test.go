package campaigntransport

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
		withAuthClient(ts.withDomain(domain).build(), &gametest.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "owner"}}),
		gametest.FixedClock(now),
		gametest.FixedSequenceIDGenerator("campaign-123", "participant-owner", "participant-ai"),
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
	svc := NewCampaignService(withAuthClient(newTestDeps().build(), &gametest.FakeAuthClient{}))
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
		gametest.FixedClock(now),
		gametest.FixedSequenceIDGenerator("campaign-123", "participant-owner", "participant-ai"),
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
		gametest.FixedClock(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)),
		gametest.FixedSequenceIDGenerator("campaign-123", "participant-123"),
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
		withAuthClient(ts.withDomain(domain).build(), &gametest.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "owner"}}),
		gametest.FixedClock(now),
		gametest.FixedSequenceIDGenerator("campaign-123", "participant-123"),
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
		withAuthClient(ts.withDomain(domain).build(), &gametest.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "owner"}}),
		gametest.FixedClock(now),
		gametest.FixedSequenceIDGenerator("campaign-123", "participant-123"),
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
				withAuthClient(ts.withDomain(domain).build(), &gametest.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "owner"}}),
				gametest.FixedClock(now),
				gametest.FixedSequenceIDGenerator("campaign-123", "participant-owner", "participant-ai"),
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
	socialClient := &gametest.FakeSocialClient{Profile: &socialv1.UserProfile{
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
		gametest.FixedClock(now),
		gametest.FixedSequenceIDGenerator("campaign-123", "participant-123"),
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
	socialClient := &gametest.FakeSocialClient{Profile: &socialv1.UserProfile{
		UserId:        "user-123",
		Name:          "Profile Name",
		AvatarSetId:   "creatures-v1",
		AvatarAssetId: "social-avatar",
	}}

	stores := ts.withDomain(domain).build()
	stores.Social = socialClient
	svc := newTestCampaignService(
		stores,
		gametest.FixedClock(now),
		gametest.FixedSequenceIDGenerator("campaign-123", "participant-123"),
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
	authClient := &gametest.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "owner-handle"}}

	svc := newTestCampaignService(
		withAuthClient(ts.withDomain(domain).build(), authClient),
		gametest.FixedClock(now),
		gametest.FixedSequenceIDGenerator("campaign-123", "participant-123"),
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
	authClient := &gametest.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "apelido"}}

	svc := newTestCampaignService(
		withAuthClient(ts.withDomain(domain).build(), authClient),
		gametest.FixedClock(now),
		gametest.FixedSequenceIDGenerator("campaign-123", "participant-123"),
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
	authClient := &gametest.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "apelido"}}

	svc := newTestCampaignService(
		withAuthClient(ts.withDomain(domain).build(), authClient),
		gametest.FixedClock(now),
		gametest.FixedSequenceIDGenerator("campaign-123", "participant-owner", "participant-ai"),
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

func TestListCampaigns_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.ListCampaigns(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListCampaigns_DeniesMissingIdentity(t *testing.T) {
	svc := NewCampaignService(newTestDeps().build())

	resp, err := svc.ListCampaigns(context.Background(), &statev1.ListCampaignsRequest{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}
}

func TestListCampaigns_AllowsAdminOverride(t *testing.T) {
	ts := newTestDeps()
	now := time.Now().UTC()
	ts.Campaign.Campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Campaign One",
		Status:    campaign.StatusActive,
		GmMode:    campaign.GmModeHuman,
		CreatedAt: now,
	}
	ts.Campaign.Campaigns["c2"] = storage.CampaignRecord{
		ID:        "c2",
		Name:      "Campaign Two",
		Status:    campaign.StatusDraft,
		GmMode:    campaign.GmModeAI,
		CreatedAt: now,
	}
	svc := NewCampaignService(ts.build())

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
		grpcmeta.AuthzOverrideReasonHeader, "admin_dashboard",
		grpcmeta.UserIDHeader, "user-admin-1",
	))
	resp, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("ListCampaigns returned error: %v", err)
	}
	if len(resp.Campaigns) != 2 {
		t.Fatalf("ListCampaigns returned %d campaigns, want 2", len(resp.Campaigns))
	}
	byID := make(map[string]*statev1.Campaign, len(resp.Campaigns))
	for _, campaignRecord := range resp.Campaigns {
		byID[campaignRecord.GetId()] = campaignRecord
	}
	if byID["c1"] == nil {
		t.Fatal("campaign c1 missing from response")
	}
	if byID["c2"] == nil {
		t.Fatal("campaign c2 missing from response")
	}
}

func TestListCampaigns_DeniesAdminOverrideWithoutReason(t *testing.T) {
	svc := NewCampaignService(newTestDeps().build())

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
		grpcmeta.UserIDHeader, "user-admin-1",
	))
	_, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListCampaigns_DeniesAdminOverrideWithoutPrincipal(t *testing.T) {
	svc := NewCampaignService(newTestDeps().build())

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
		grpcmeta.AuthzOverrideReasonHeader, "admin_dashboard",
	))
	_, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListCampaigns_WithParticipantIdentity(t *testing.T) {
	ts := newTestDeps()
	now := time.Now().UTC()
	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecord("c1", "Campaign One", campaign.StatusDraft, campaign.GmModeHuman)
	ts.Campaign.Campaigns["c2"] = gametest.DaggerheartCampaignRecordWithCreatedAt("c2", "Campaign Two", campaign.StatusActive, campaign.GmModeAI, now)
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"participant-1": gametest.UserParticipantRecord("c1", "participant-1", "user-1", "Alice"),
	}
	ts.Participant.Participants["c2"] = map[string]storage.ParticipantRecord{
		"participant-1": gametest.UserParticipantRecord("c2", "participant-1", "user-1", "Alice"),
	}

	svc := NewCampaignService(ts.build())

	resp, err := svc.ListCampaigns(gametest.ContextWithParticipantID("participant-1"), &statev1.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("ListCampaigns returned error: %v", err)
	}
	if len(resp.Campaigns) != 2 {
		t.Errorf("ListCampaigns returned %d campaigns, want 2", len(resp.Campaigns))
	}
	if ts.Participant.ListCampaignIDsByParticipantCalls != 1 {
		t.Fatalf("ListCampaignIDsByParticipant calls = %d, want 1", ts.Participant.ListCampaignIDsByParticipantCalls)
	}
}

func TestListCampaigns_UserScopedByMetadata(t *testing.T) {
	ts := newTestDeps()
	now := time.Now().UTC()
	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecordWithCreatedAt("c1", "Campaign One", campaign.StatusDraft, campaign.GmModeHuman, now)
	ts.Campaign.Campaigns["c2"] = gametest.DaggerheartCampaignRecordWithCreatedAt("c2", "Campaign Two", campaign.StatusActive, campaign.GmModeAI, now)
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": gametest.UserParticipantRecord("c1", "p1", "user-123", "Alice"),
	}
	ts.Participant.Participants["c2"] = map[string]storage.ParticipantRecord{
		"p2": gametest.UserParticipantRecord("c2", "p2", "user-999", "Bob"),
	}

	svc := NewCampaignService(ts.build())
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123"))

	resp, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("ListCampaigns returned error: %v", err)
	}
	if len(resp.Campaigns) != 1 {
		t.Fatalf("ListCampaigns returned %d campaigns, want 1", len(resp.Campaigns))
	}
	if ts.Participant.ListCampaignIDsByUserCalls != 1 {
		t.Fatalf("ListCampaignIDsByUser calls = %d, want 1", ts.Participant.ListCampaignIDsByUserCalls)
	}
	if ts.Participant.ListByCampaignCalls != 0 {
		t.Fatalf("ListParticipantsByCampaign calls = %d, want 0", ts.Participant.ListByCampaignCalls)
	}
	if resp.Campaigns[0].GetId() != "c1" {
		t.Fatalf("ListCampaigns campaign id = %q, want %q", resp.Campaigns[0].GetId(), "c1")
	}
}

func TestListCampaigns_UserScopedByMetadataAfterPageBoundary(t *testing.T) {
	ts := newTestDeps()
	orderedStore := &orderedCampaignStore{
		Campaigns: make([]storage.CampaignRecord, 12),
	}
	for i := 1; i <= 12; i++ {
		orderedStore.Campaigns[i-1] = storage.CampaignRecord{
			ID:        fmt.Sprintf("campaign-%03d", i),
			Name:      fmt.Sprintf("Campaign %d", i),
			System:    bridge.SystemIDDaggerheart,
			Status:    campaign.StatusDraft,
			GmMode:    campaign.GmModeHuman,
			CreatedAt: time.Now().UTC(),
		}
	}
	ts.Participant.Participants["campaign-012"] = map[string]storage.ParticipantRecord{
		"p1": gametest.UserParticipantRecord("campaign-012", "p1", "user-123", "Alice"),
	}

	stores := ts.build()
	stores.Campaign = orderedStore
	svc := NewCampaignService(stores)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123"))

	resp, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("ListCampaigns returned error: %v", err)
	}
	if len(resp.Campaigns) != 1 {
		t.Fatalf("ListCampaigns returned %d campaigns, want 1", len(resp.Campaigns))
	}
	if resp.Campaigns[0].GetId() != "campaign-012" {
		t.Fatalf("ListCampaigns campaign id = %q, want %q", resp.Campaigns[0].GetId(), "campaign-012")
	}
	if ts.Participant.ListCampaignIDsByUserCalls != 1 {
		t.Fatalf("ListCampaignIDsByUser calls = %d, want 1", ts.Participant.ListCampaignIDsByUserCalls)
	}
}

func TestListCampaigns_UserScopedByMetadataAppliesStatusFilterAcrossPages(t *testing.T) {
	ts := newTestDeps()
	orderedStore := &orderedCampaignStore{
		Campaigns: []storage.CampaignRecord{
			{
				ID:        "campaign-001",
				Name:      "Campaign 1",
				System:    bridge.SystemIDDaggerheart,
				Status:    campaign.StatusDraft,
				GmMode:    campaign.GmModeHuman,
				CreatedAt: time.Now().UTC(),
			},
			{
				ID:        "campaign-002",
				Name:      "Campaign 2",
				System:    bridge.SystemIDDaggerheart,
				Status:    campaign.StatusCompleted,
				GmMode:    campaign.GmModeHuman,
				CreatedAt: time.Now().UTC(),
			},
			{
				ID:        "campaign-003",
				Name:      "Campaign 3",
				System:    bridge.SystemIDDaggerheart,
				Status:    campaign.StatusActive,
				GmMode:    campaign.GmModeHuman,
				CreatedAt: time.Now().UTC(),
			},
			{
				ID:        "campaign-004",
				Name:      "Campaign 4",
				System:    bridge.SystemIDDaggerheart,
				Status:    campaign.StatusArchived,
				GmMode:    campaign.GmModeHuman,
				CreatedAt: time.Now().UTC(),
			},
		},
	}
	for _, record := range orderedStore.Campaigns {
		ts.Participant.Participants[record.ID] = map[string]storage.ParticipantRecord{
			"p1": gametest.UserParticipantRecord(record.ID, "p1", "user-123", "Alice"),
		}
	}

	stores := ts.build()
	stores.Campaign = orderedStore
	svc := NewCampaignService(stores)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123"))

	firstPage, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{
		PageSize: 1,
		Statuses: []statev1.CampaignStatus{
			statev1.CampaignStatus_DRAFT,
			statev1.CampaignStatus_ACTIVE,
		},
	})
	if err != nil {
		t.Fatalf("ListCampaigns first page returned error: %v", err)
	}
	if len(firstPage.GetCampaigns()) != 1 {
		t.Fatalf("first page campaigns = %d, want 1", len(firstPage.GetCampaigns()))
	}
	if got := firstPage.GetCampaigns()[0].GetId(); got != "campaign-001" {
		t.Fatalf("first page campaign id = %q, want %q", got, "campaign-001")
	}
	if got := firstPage.GetNextPageToken(); got != "campaign-001" {
		t.Fatalf("first page next token = %q, want %q", got, "campaign-001")
	}

	secondPage, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{
		PageSize:  1,
		PageToken: firstPage.GetNextPageToken(),
		Statuses: []statev1.CampaignStatus{
			statev1.CampaignStatus_DRAFT,
			statev1.CampaignStatus_ACTIVE,
		},
	})
	if err != nil {
		t.Fatalf("ListCampaigns second page returned error: %v", err)
	}
	if len(secondPage.GetCampaigns()) != 1 {
		t.Fatalf("second page campaigns = %d, want 1", len(secondPage.GetCampaigns()))
	}
	if got := secondPage.GetCampaigns()[0].GetId(); got != "campaign-003" {
		t.Fatalf("second page campaign id = %q, want %q", got, "campaign-003")
	}
	if got := secondPage.GetNextPageToken(); got != "campaign-003" {
		t.Fatalf("second page next token = %q, want %q", got, "campaign-003")
	}

	thirdPage, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{
		PageSize:  1,
		PageToken: secondPage.GetNextPageToken(),
		Statuses: []statev1.CampaignStatus{
			statev1.CampaignStatus_DRAFT,
			statev1.CampaignStatus_ACTIVE,
		},
	})
	if err != nil {
		t.Fatalf("ListCampaigns third page returned error: %v", err)
	}
	if len(thirdPage.GetCampaigns()) != 0 {
		t.Fatalf("third page campaigns = %d, want 0", len(thirdPage.GetCampaigns()))
	}
	if got := thirdPage.GetNextPageToken(); got != "" {
		t.Fatalf("third page next token = %q, want empty", got)
	}
}

func TestListCampaigns_UserScopedByMetadataQueryFailure(t *testing.T) {
	ts := newTestDeps()
	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecordWithCreatedAt("c1", "Campaign One", campaign.StatusDraft, campaign.GmModeHuman, time.Now().UTC())
	ts.Participant.ListCampaignIDsByUserErr = fmt.Errorf("campaign index unavailable")
	svc := NewCampaignService(ts.build())
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123"))

	_, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	assertStatusCode(t, err, codes.Internal)
}

func TestGetCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.GetCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCampaign_MissingCampaignId(t *testing.T) {
	svc := NewCampaignService(newTestDeps().build())
	_, err := svc.GetCampaign(context.Background(), &statev1.GetCampaignRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCampaign_NotFound(t *testing.T) {
	svc := NewCampaignService(newTestDeps().build())
	_, err := svc.GetCampaign(context.Background(), &statev1.GetCampaignRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetCampaign_DeniesMissingIdentity(t *testing.T) {
	ts := newTestDeps()
	now := time.Now().UTC()
	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatusAndCreatedAt(campaign.StatusDraft, now)

	svc := NewCampaignService(ts.build())
	_, err := svc.GetCampaign(context.Background(), &statev1.GetCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetCampaign_DeniesNonMember(t *testing.T) {
	ts := newTestDeps()
	now := time.Now().UTC()
	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatusAndCreatedAt(campaign.StatusDraft, now)

	svc := NewCampaignService(ts.build())
	_, err := svc.GetCampaign(gametest.ContextWithParticipantID("outsider-1"), &statev1.GetCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetCampaign_Success(t *testing.T) {
	ts := newTestDeps()
	now := time.Now().UTC()
	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatusAndCreatedAt(campaign.StatusDraft, now)
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"participant-1": gametest.MemberUserParticipantRecord("c1", "participant-1", "user-1", ""),
	}

	svc := NewCampaignService(ts.build())

	resp, err := svc.GetCampaign(gametest.ContextWithParticipantID("participant-1"), &statev1.GetCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetCampaign returned error: %v", err)
	}
	if resp.Campaign == nil {
		t.Fatal("GetCampaign response has nil campaign")
	}
	if resp.Campaign.Id != "c1" {
		t.Errorf("Campaign ID = %q, want %q", resp.Campaign.Id, "c1")
	}
	if resp.Campaign.Name != "Test Campaign" {
		t.Errorf("Campaign Name = %q, want %q", resp.Campaign.Name, "Test Campaign")
	}
}

func TestGetCampaign_SuccessByUserIDFallback(t *testing.T) {
	ts := newTestDeps()
	now := time.Now().UTC()
	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatusAndCreatedAt(campaign.StatusDraft, now)
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"participant-1": gametest.MemberUserParticipantRecord("c1", "participant-1", "user-1", ""),
	}

	svc := NewCampaignService(ts.build())
	resp, err := svc.GetCampaign(gametest.ContextWithUserID("user-1"), &statev1.GetCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetCampaign returned error: %v", err)
	}
	if resp.Campaign == nil || resp.Campaign.GetId() != "c1" {
		t.Fatalf("campaign = %+v, want id c1", resp.Campaign)
	}
}

func TestEndCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.EndCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndCampaign_MissingCampaignId(t *testing.T) {
	svc := NewCampaignService(newTestDeps().withSession().build())
	_, err := svc.EndCampaign(context.Background(), &statev1.EndCampaignRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndCampaign_NotFound(t *testing.T) {
	svc := NewCampaignService(newTestDeps().withSession().build())
	_, err := svc.EndCampaign(context.Background(), &statev1.EndCampaignRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestEndCampaign_ActiveSessionBlocks(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatus(campaign.StatusActive)
	ts.Session.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	ts.Session.ActiveSession["c1"] = "s1"

	svc := NewCampaignService(ts.build())
	_, err := svc.EndCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestEndCampaign_DraftStatusDisallowed(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")

	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatus(campaign.StatusDraft)

	svc := NewCampaignService(ts.build())
	_, err := svc.EndCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestEndCampaign_AllowsManagerAccess(t *testing.T) {
	ts := newTestDeps().withSession()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"manager-1": gametest.ManagerParticipantRecord("c1", "manager-1"),
	}
	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"completed"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), gametest.FixedClock(now), nil)

	resp, err := svc.EndCampaign(gametest.ContextWithParticipantID("manager-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("EndCampaign returned error: %v", err)
	}
	if resp.GetCampaign().GetStatus() != statev1.CampaignStatus_COMPLETED {
		t.Fatalf("campaign status = %v, want %v", resp.GetCampaign().GetStatus(), statev1.CampaignStatus_COMPLETED)
	}
}

func TestEndCampaign_RequiresDomainEngine(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewCampaignService(ts.build())
	_, err := svc.EndCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestEndCampaign_Success(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatus(campaign.StatusActive)
	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"completed"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), gametest.FixedClock(now), gametest.FixedIDGenerator("campaign-123"))

	resp, err := svc.EndCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("EndCampaign returned error: %v", err)
	}
	if resp.Campaign.Status != statev1.CampaignStatus_COMPLETED {
		t.Errorf("Campaign Status = %v, want %v", resp.Campaign.Status, statev1.CampaignStatus_COMPLETED)
	}
	if resp.Campaign.CompletedAt == nil {
		t.Error("Campaign CompletedAt is nil")
	}

	// Verify persisted
	stored, _ := ts.Campaign.Get(context.Background(), "c1")
	if stored.Status != campaign.StatusCompleted {
		t.Errorf("Stored campaign Status = %v, want %v", stored.Status, campaign.StatusCompleted)
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("campaign.updated") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("campaign.updated"))
	}
}

func TestEndCampaign_UsesDomainEngine(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatus(campaign.StatusActive)

	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"completed"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), gametest.FixedClock(now), nil)

	_, err := svc.EndCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("EndCampaign returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("campaign.end") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "campaign.end")
	}
}

func TestArchiveCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.ArchiveCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestArchiveCampaign_MissingCampaignId(t *testing.T) {
	svc := NewCampaignService(newTestDeps().withSession().build())
	_, err := svc.ArchiveCampaign(context.Background(), &statev1.ArchiveCampaignRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestArchiveCampaign_NotFound(t *testing.T) {
	svc := NewCampaignService(newTestDeps().withSession().build())
	_, err := svc.ArchiveCampaign(context.Background(), &statev1.ArchiveCampaignRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestArchiveCampaign_ActiveSessionBlocks(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Session.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	ts.Session.ActiveSession["c1"] = "s1"

	svc := NewCampaignService(ts.build())
	_, err := svc.ArchiveCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestArchiveCampaign_RequiresDomainEngine(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewCampaignService(ts.build())
	_, err := svc.ArchiveCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestArchiveCampaign_Success(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatus(campaign.StatusCompleted)
	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"archived"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), gametest.FixedClock(now), gametest.FixedIDGenerator("campaign-123"))

	resp, err := svc.ArchiveCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ArchiveCampaign returned error: %v", err)
	}
	if resp.Campaign.Status != statev1.CampaignStatus_ARCHIVED {
		t.Errorf("Campaign Status = %v, want %v", resp.Campaign.Status, statev1.CampaignStatus_ARCHIVED)
	}
	if resp.Campaign.ArchivedAt == nil {
		t.Error("Campaign ArchivedAt is nil")
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("campaign.updated") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("campaign.updated"))
	}
}

func TestArchiveCampaign_UsesDomainEngine(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatus(campaign.StatusCompleted)

	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"archived"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), gametest.FixedClock(now), nil)

	_, err := svc.ArchiveCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ArchiveCampaign returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("campaign.archive") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "campaign.archive")
	}
}

func TestRestoreCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.RestoreCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRestoreCampaign_MissingCampaignId(t *testing.T) {
	svc := NewCampaignService(newTestDeps().build())
	_, err := svc.RestoreCampaign(context.Background(), &statev1.RestoreCampaignRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRestoreCampaign_NotFound(t *testing.T) {
	svc := NewCampaignService(newTestDeps().build())
	_, err := svc.RestoreCampaign(context.Background(), &statev1.RestoreCampaignRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestRestoreCampaign_NotArchivedDisallowed(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewCampaignService(ts.build())
	_, err := svc.RestoreCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestRestoreCampaign_RequiresDomainEngine(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	ts.Campaign.Campaigns["c1"] = gametest.ArchivedCampaignRecord("c1")

	svc := NewCampaignService(ts.build())
	_, err := svc.RestoreCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestRestoreCampaign_Success(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	archivedAt := now.Add(-24 * time.Hour)

	ts.Campaign.Campaigns["c1"] = gametest.TestArchivedCampaignRecord(archivedAt)
	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"draft"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), gametest.FixedClock(now), gametest.FixedIDGenerator("campaign-123"))

	resp, err := svc.RestoreCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("RestoreCampaign returned error: %v", err)
	}
	if resp.Campaign.Status != statev1.CampaignStatus_DRAFT {
		t.Errorf("Campaign Status = %v, want %v", resp.Campaign.Status, statev1.CampaignStatus_DRAFT)
	}
	if resp.Campaign.ArchivedAt != nil {
		t.Error("Campaign ArchivedAt should be nil after restore")
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("campaign.updated") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("campaign.updated"))
	}
}

func TestRestoreCampaign_UsesDomainEngine(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	archivedAt := now.Add(-24 * time.Hour)

	ts.Campaign.Campaigns["c1"] = gametest.TestArchivedCampaignRecord(archivedAt)

	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"draft"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), gametest.FixedClock(now), nil)

	_, err := svc.RestoreCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("RestoreCampaign returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("campaign.restore") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "campaign.restore")
	}
}

func TestSetCampaignCover_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.SetCampaignCover(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.UpdateCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCampaign_InvalidLocaleRejected(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.UpdateCampaign(context.Background(), &statev1.UpdateCampaignRequest{
		CampaignId: "c1",
		Locale:     commonv1.Locale(99),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCampaign_Success(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	storedCampaign := gametest.TestCampaignRecordWithStatus(campaign.StatusActive)
	storedCampaign.Name = "Old Name"
	storedCampaign.ThemePrompt = "Old theme"
	storedCampaign.Locale = "en-US"
	ts.Campaign.Campaigns["c1"] = storedCampaign

	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"name":"New Name","theme_prompt":"New theme","locale":"pt-BR"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), gametest.FixedClock(now), nil)

	resp, err := svc.UpdateCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.UpdateCampaignRequest{
		CampaignId:  "c1",
		Name:        wrapperspb.String("  New Name  "),
		ThemePrompt: wrapperspb.String("  New theme  "),
		Locale:      commonv1.Locale_LOCALE_PT_BR,
	})
	if err != nil {
		t.Fatalf("UpdateCampaign returned error: %v", err)
	}
	if resp.GetCampaign().GetName() != "New Name" {
		t.Fatalf("campaign name = %q, want %q", resp.GetCampaign().GetName(), "New Name")
	}
	if resp.GetCampaign().GetThemePrompt() != "New theme" {
		t.Fatalf("campaign theme = %q, want %q", resp.GetCampaign().GetThemePrompt(), "New theme")
	}
	if resp.GetCampaign().GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("campaign locale = %v, want %v", resp.GetCampaign().GetLocale(), commonv1.Locale_LOCALE_PT_BR)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("campaign.update") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "campaign.update")
	}

	var payload campaign.UpdatePayload
	if err := json.Unmarshal(domain.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.Fields["name"] != "New Name" {
		t.Fatalf("payload name = %q, want %q", payload.Fields["name"], "New Name")
	}
	if payload.Fields["theme_prompt"] != "New theme" {
		t.Fatalf("payload theme_prompt = %q, want %q", payload.Fields["theme_prompt"], "New theme")
	}
	if payload.Fields["locale"] != "pt-BR" {
		t.Fatalf("payload locale = %q, want %q", payload.Fields["locale"], "pt-BR")
	}
}

func TestUpdateCampaign_NoOpSkipsDomainCommand(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	storedCampaign := gametest.TestCampaignRecordWithStatus(campaign.StatusActive)
	storedCampaign.Name = "Existing Name"
	storedCampaign.ThemePrompt = "Existing theme"
	storedCampaign.Locale = "en-US"
	ts.Campaign.Campaigns["c1"] = storedCampaign

	domain := &fakeDomainEngine{store: ts.Event}
	svc := newTestCampaignService(ts.withDomain(domain).build(), gametest.FixedClock(now), nil)

	resp, err := svc.UpdateCampaign(gametest.ContextWithParticipantID("owner-1"), &statev1.UpdateCampaignRequest{
		CampaignId:  "c1",
		Name:        wrapperspb.String("Existing Name"),
		ThemePrompt: wrapperspb.String("Existing theme"),
		Locale:      commonv1.Locale_LOCALE_EN_US,
	})
	if err != nil {
		t.Fatalf("UpdateCampaign returned error: %v", err)
	}
	if domain.calls != 0 {
		t.Fatalf("expected no domain command for no-op update, got %d", domain.calls)
	}
	if resp.GetCampaign().GetName() != "Existing Name" {
		t.Fatalf("campaign name = %q, want %q", resp.GetCampaign().GetName(), "Existing Name")
	}
	if resp.GetCampaign().GetThemePrompt() != "Existing theme" {
		t.Fatalf("campaign theme = %q, want %q", resp.GetCampaign().GetThemePrompt(), "Existing theme")
	}
	if resp.GetCampaign().GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("campaign locale = %v, want %v", resp.GetCampaign().GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}
}

func TestSetCampaignCover_Success(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	storedCampaign := gametest.TestCampaignRecordWithStatus(campaign.StatusActive)
	storedCampaign.CoverAssetID = "camp-cover-01"
	ts.Campaign.Campaigns["c1"] = storedCampaign

	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"cover_asset_id":"camp-cover-04"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), gametest.FixedClock(now), nil)

	resp, err := svc.SetCampaignCover(gametest.ContextWithParticipantID("owner-1"), &statev1.SetCampaignCoverRequest{
		CampaignId:   "c1",
		CoverAssetId: "camp-cover-04",
	})
	if err != nil {
		t.Fatalf("SetCampaignCover returned error: %v", err)
	}
	if resp.GetCampaign().GetCoverAssetId() != "camp-cover-04" {
		t.Fatalf("campaign cover asset id = %q, want %q", resp.GetCampaign().GetCoverAssetId(), "camp-cover-04")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("campaign.update") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "campaign.update")
	}

	var payload campaign.UpdatePayload
	if err := json.Unmarshal(domain.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.Fields["cover_asset_id"] != "camp-cover-04" {
		t.Fatalf("cover_asset_id command field = %q, want %q", payload.Fields["cover_asset_id"], "camp-cover-04")
	}
}
