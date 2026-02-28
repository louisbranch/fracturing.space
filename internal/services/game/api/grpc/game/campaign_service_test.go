package game

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type orderedCampaignStore struct {
	campaigns []storage.CampaignRecord
}

func (s *orderedCampaignStore) Put(_ context.Context, c storage.CampaignRecord) error {
	if s == nil {
		return fmt.Errorf("storage is not configured")
	}
	s.campaigns = append(s.campaigns, c)
	return nil
}

func (s *orderedCampaignStore) Get(_ context.Context, id string) (storage.CampaignRecord, error) {
	if s == nil {
		return storage.CampaignRecord{}, fmt.Errorf("storage is not configured")
	}
	for _, c := range s.campaigns {
		if c.ID == id {
			return c, nil
		}
	}
	return storage.CampaignRecord{}, storage.ErrNotFound
}

func (s *orderedCampaignStore) List(_ context.Context, pageSize int, pageToken string) (storage.CampaignPage, error) {
	if s == nil {
		return storage.CampaignPage{}, fmt.Errorf("storage is not configured")
	}
	if pageSize <= 0 {
		return storage.CampaignPage{}, fmt.Errorf("page size must be greater than zero")
	}

	page := storage.CampaignPage{
		Campaigns: make([]storage.CampaignRecord, 0, pageSize),
	}

	start := 0
	if pageToken != "" {
		for idx, c := range s.campaigns {
			if c.ID == pageToken {
				start = idx + 1
				break
			}
		}
	}
	if start < 0 || start > len(s.campaigns) {
		start = 0
	}

	end := start + pageSize
	if end > len(s.campaigns) {
		end = len(s.campaigns)
	}

	page.Campaigns = append(page.Campaigns, s.campaigns[start:end]...)
	if end < len(s.campaigns) {
		page.NextPageToken = s.campaigns[end-1].ID
	}
	return page, nil
}

func ownerParticipantStore(campaignID string) *fakeParticipantStore {
	store := newFakeParticipantStore()
	store.participants[campaignID] = map[string]storage.ParticipantRecord{
		"owner-1": {
			ID:             "owner-1",
			CampaignID:     campaignID,
			CampaignAccess: participant.CampaignAccessOwner,
		},
	}
	return store
}

func TestCreateCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Stores{})
	_, err := svc.CreateCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_MissingSystem(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := newFakeParticipantStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore, Event: eventStore, Participant: participantStore})
	_, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		GmMode: statev1.GmMode_HUMAN,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_EmptyName(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := newFakeParticipantStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore, Event: eventStore, Participant: participantStore})
	_, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:   "",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_MissingGmMode(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := newFakeParticipantStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore, Event: eventStore, Participant: participantStore})
	_, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_MissingCreatorUserID(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := newFakeParticipantStore()
	svc := NewCampaignServiceWithAuth(Stores{Campaign: campaignStore, Event: eventStore, Participant: participantStore}, &fakeAuthClient{})
	_, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := newFakeParticipantStore()
	svc := &CampaignService{
		stores:      Stores{Campaign: campaignStore, Event: eventStore, Participant: participantStore},
		clock:       fixedClock(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)),
		idGenerator: fixedSequenceIDGenerator("campaign-123", "participant-123"),
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123"))
	_, err := svc.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateCampaign_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := newFakeParticipantStore()
	fakeAuth := &fakeAuthClient{user: &authv1.User{Id: "user-123", Email: "creator"}}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{
		store: eventStore,
		resultsByType: map[command.Type]engine.Result{
			command.Type("campaign.create"): {
				Decision: command.Accept(event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("campaign.created"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "campaign",
					EntityID:    "campaign-123",
					PayloadJSON: []byte(`{"name":"Test Campaign","locale":"en-US","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
				}),
			},
			command.Type("participant.join"): {
				Decision: command.Accept(event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("participant.joined"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "participant",
					EntityID:    "participant-123",
					PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"creator","role":"GM","controller":"HUMAN","campaign_access":"OWNER"}`),
				}),
			},
		},
	}
	svc := &CampaignService{
		stores:      Stores{Campaign: campaignStore, Event: eventStore, Participant: participantStore, Domain: domain},
		clock:       fixedClock(now),
		idGenerator: fixedSequenceIDGenerator("campaign-123", "participant-123"),
		authClient:  fakeAuth,
	}

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
	if got := len(eventStore.events["campaign-123"]); got != 2 {
		t.Fatalf("expected 2 events, got %d", got)
	}
	if eventStore.events["campaign-123"][0].Type != event.Type("campaign.created") {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-123"][0].Type, event.Type("campaign.created"))
	}
	if eventStore.events["campaign-123"][1].Type != event.Type("participant.joined") {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-123"][1].Type, event.Type("participant.joined"))
	}

	// Verify persisted
	stored, err := campaignStore.Get(context.Background(), "campaign-123")
	if err != nil {
		t.Fatalf("Campaign not persisted: %v", err)
	}
	if stored.Name != "Test Campaign" {
		t.Errorf("Stored campaign Name = %q, want %q", stored.Name, "Test Campaign")
	}
}

func TestCreateCampaign_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := newFakeParticipantStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{
		store: eventStore,
		resultsByType: map[command.Type]engine.Result{
			command.Type("campaign.create"): {
				Decision: command.Accept(event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("campaign.created"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "campaign",
					EntityID:    "campaign-123",
					PayloadJSON: []byte(`{"name":"Test Campaign","locale":"en-US","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
				}),
			},
			command.Type("participant.join"): {
				Decision: command.Accept(event.Event{
					CampaignID:  "campaign-123",
					Type:        event.Type("participant.joined"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "participant",
					EntityID:    "participant-123",
					PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"Owner","role":"GM","controller":"HUMAN","campaign_access":"OWNER"}`),
				}),
			},
		},
	}

	svc := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Event:       eventStore,
			Participant: participantStore,
			Domain:      domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedSequenceIDGenerator("campaign-123", "participant-123"),
	}

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
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("campaign.create") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "campaign.create")
	}
	if domain.commands[1].Type != command.Type("participant.join") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "participant.join")
	}
	if got := len(eventStore.events["campaign-123"]); got != 2 {
		t.Fatalf("expected 2 events, got %d", got)
	}
	if eventStore.events["campaign-123"][0].Type != event.Type("campaign.created") {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-123"][0].Type, event.Type("campaign.created"))
	}
	if eventStore.events["campaign-123"][1].Type != event.Type("participant.joined") {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-123"][1].Type, event.Type("participant.joined"))
	}
}

func TestCreateCampaign_OwnerParticipantHydratesFromSocialProfile(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := newFakeParticipantStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("campaign.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-123",
				Type:        event.Type("campaign.created"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "campaign-123",
				PayloadJSON: []byte(`{"name":"Test Campaign","locale":"en-US","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
			}),
		},
		command.Type("participant.join"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-123",
				Type:        event.Type("participant.joined"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "participant-123",
				PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"Profile Name","role":"GM","controller":"HUMAN","campaign_access":"OWNER","avatar_set_id":"creatures-v1","avatar_asset_id":"social-avatar","pronouns":"they/them"}`),
			}),
		},
	}}
	socialClient := &fakeSocialClient{profile: &socialv1.UserProfile{
		UserId:        "user-123",
		Name:          "Profile Name",
		Pronouns:      "they/them",
		AvatarSetId:   "creatures-v1",
		AvatarAssetId: "social-avatar",
	}}

	svc := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Event:       eventStore,
			Participant: participantStore,
			Domain:      domain,
			Social:      socialClient,
		},
		clock:       fixedClock(now),
		idGenerator: fixedSequenceIDGenerator("campaign-123", "participant-123"),
	}

	_, err := svc.CreateCampaign(metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123")), &statev1.CreateCampaignRequest{
		Name:        "Test Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      statev1.GmMode_HUMAN,
		ThemePrompt: "A dark fantasy adventure",
	})
	if err != nil {
		t.Fatalf("CreateCampaign returned error: %v", err)
	}
	if socialClient.getUserProfileCalls != 1 {
		t.Fatalf("GetUserProfile calls = %d, want %d", socialClient.getUserProfileCalls, 1)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}

	var payload participant.JoinPayload
	if err := json.Unmarshal(domain.commands[1].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode participant payload: %v", err)
	}
	if payload.Name != "Profile Name" {
		t.Fatalf("payload name = %q, want %q", payload.Name, "Profile Name")
	}
	if payload.AvatarSetID != "creatures-v1" {
		t.Fatalf("payload avatar_set_id = %q, want %q", payload.AvatarSetID, "creatures-v1")
	}
	if payload.AvatarAssetID != "social-avatar" {
		t.Fatalf("payload avatar_asset_id = %q, want %q", payload.AvatarAssetID, "social-avatar")
	}
	if payload.Pronouns != "they/them" {
		t.Fatalf("payload pronouns = %q, want %q", payload.Pronouns, "they/them")
	}
}

func TestCreateCampaign_OwnerParticipantFallsBackToUserIDWithoutSocialOrAuth(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := newFakeParticipantStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("campaign.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-123",
				Type:        event.Type("campaign.created"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "campaign-123",
				PayloadJSON: []byte(`{"name":"Test Campaign","locale":"en-US","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN","intent":"STARTER","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
			}),
		},
		command.Type("participant.join"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-123",
				Type:        event.Type("participant.joined"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "participant-123",
				PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"user-123","role":"GM","controller":"HUMAN","campaign_access":"OWNER"}`),
			}),
		},
	}}

	svc := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Event:       eventStore,
			Participant: participantStore,
			Domain:      domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedSequenceIDGenerator("campaign-123", "participant-123"),
	}

	_, err := svc.CreateCampaign(metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123")), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	if err != nil {
		t.Fatalf("CreateCampaign returned error: %v", err)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}

	var payload participant.JoinPayload
	if err := json.Unmarshal(domain.commands[1].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode participant payload: %v", err)
	}
	if payload.Name != "user-123" {
		t.Fatalf("payload name = %q, want %q", payload.Name, "user-123")
	}
	if payload.AvatarSetID != "" {
		t.Fatalf("payload avatar_set_id = %q, want empty", payload.AvatarSetID)
	}
	if payload.AvatarAssetID != "" {
		t.Fatalf("payload avatar_asset_id = %q, want empty", payload.AvatarAssetID)
	}
	if payload.Pronouns != "" {
		t.Fatalf("payload pronouns = %q, want empty", payload.Pronouns)
	}
}

func TestListCampaigns_NilRequest(t *testing.T) {
	svc := NewCampaignService(Stores{})
	_, err := svc.ListCampaigns(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListCampaigns_DeniesMissingIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore, Participant: participantStore})

	resp, err := svc.ListCampaigns(context.Background(), &statev1.ListCampaignsRequest{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}
}

func TestListCampaigns_AllowsAdminOverride(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	now := time.Now().UTC()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Name: "Campaign One", Status: campaign.StatusActive, GmMode: campaign.GmModeHuman, CreatedAt: now}
	campaignStore.campaigns["c2"] = storage.CampaignRecord{ID: "c2", Name: "Campaign Two", Status: campaign.StatusDraft, GmMode: campaign.GmModeAI, CreatedAt: now}
	svc := NewCampaignService(Stores{Campaign: campaignStore})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
		grpcmeta.AuthzOverrideReasonHeader, "admin_dashboard",
	))
	resp, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("ListCampaigns returned error: %v", err)
	}
	if len(resp.Campaigns) != 2 {
		t.Fatalf("ListCampaigns returned %d campaigns, want 2", len(resp.Campaigns))
	}
}

func TestListCampaigns_DeniesAdminOverrideWithoutReason(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
	))
	_, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListCampaigns_WithParticipantIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Name:   "Campaign One",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status: campaign.StatusDraft,
		GmMode: campaign.GmModeHuman,
	}
	campaignStore.campaigns["c2"] = storage.CampaignRecord{
		ID:        "c2",
		Name:      "Campaign Two",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.StatusActive,
		GmMode:    campaign.GmModeAI,
		CreatedAt: now,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "c1", UserID: "user-1", Name: "Alice"},
	}
	participantStore.participants["c2"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "c2", UserID: "user-1", Name: "Alice"},
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Participant: participantStore})

	resp, err := svc.ListCampaigns(contextWithParticipantID("participant-1"), &statev1.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("ListCampaigns returned error: %v", err)
	}
	if len(resp.Campaigns) != 2 {
		t.Errorf("ListCampaigns returned %d campaigns, want 2", len(resp.Campaigns))
	}
	if participantStore.listCampaignIDsByParticipantCalls != 1 {
		t.Fatalf("ListCampaignIDsByParticipant calls = %d, want 1", participantStore.listCampaignIDsByParticipantCalls)
	}
}

func TestListCampaigns_UserScopedByMetadata(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Campaign One",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.StatusDraft,
		GmMode:    campaign.GmModeHuman,
		CreatedAt: now,
	}
	campaignStore.campaigns["c2"] = storage.CampaignRecord{
		ID:        "c2",
		Name:      "Campaign Two",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.StatusActive,
		GmMode:    campaign.GmModeAI,
		CreatedAt: now,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {ID: "p1", CampaignID: "c1", UserID: "user-123", Name: "Alice"},
	}
	participantStore.participants["c2"] = map[string]storage.ParticipantRecord{
		"p2": {ID: "p2", CampaignID: "c2", UserID: "user-999", Name: "Bob"},
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Participant: participantStore})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123"))

	resp, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("ListCampaigns returned error: %v", err)
	}
	if len(resp.Campaigns) != 1 {
		t.Fatalf("ListCampaigns returned %d campaigns, want 1", len(resp.Campaigns))
	}
	if participantStore.listCampaignIDsByUserCalls != 1 {
		t.Fatalf("ListCampaignIDsByUser calls = %d, want 1", participantStore.listCampaignIDsByUserCalls)
	}
	if participantStore.listByCampaignCalls != 0 {
		t.Fatalf("ListParticipantsByCampaign calls = %d, want 0", participantStore.listByCampaignCalls)
	}
	if resp.Campaigns[0].GetId() != "c1" {
		t.Fatalf("ListCampaigns campaign id = %q, want %q", resp.Campaigns[0].GetId(), "c1")
	}
}

func TestListCampaigns_UserScopedByMetadataAfterPageBoundary(t *testing.T) {
	campaignStore := &orderedCampaignStore{
		campaigns: make([]storage.CampaignRecord, 12),
	}
	for i := 1; i <= 12; i++ {
		campaignStore.campaigns[i-1] = storage.CampaignRecord{
			ID:        fmt.Sprintf("campaign-%03d", i),
			Name:      fmt.Sprintf("Campaign %d", i),
			System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			Status:    campaign.StatusDraft,
			GmMode:    campaign.GmModeHuman,
			CreatedAt: time.Now().UTC(),
		}
	}
	participantStore := newFakeParticipantStore()
	participantStore.participants["campaign-012"] = map[string]storage.ParticipantRecord{
		"p1": {ID: "p1", CampaignID: "campaign-012", UserID: "user-123", Name: "Alice"},
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Participant: participantStore})
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
	if participantStore.listCampaignIDsByUserCalls != 1 {
		t.Fatalf("ListCampaignIDsByUser calls = %d, want 1", participantStore.listCampaignIDsByUserCalls)
	}
}

func TestListCampaigns_UserScopedByMetadataQueryFailure(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Campaign One",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.StatusDraft,
		GmMode:    campaign.GmModeHuman,
		CreatedAt: time.Now().UTC(),
	}
	participantStore.listCampaignIDsByUserErr = fmt.Errorf("campaign index unavailable")
	svc := NewCampaignService(Stores{Campaign: campaignStore, Participant: participantStore})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-123"))

	_, err := svc.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	assertStatusCode(t, err, codes.Internal)
}

func TestGetCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Stores{})
	_, err := svc.GetCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCampaign_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore})
	_, err := svc.GetCampaign(context.Background(), &statev1.GetCampaignRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCampaign_NotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore})
	_, err := svc.GetCampaign(context.Background(), &statev1.GetCampaignRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetCampaign_DeniesMissingIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Test Campaign",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.StatusDraft,
		GmMode:    campaign.GmModeHuman,
		CreatedAt: now,
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetCampaign(context.Background(), &statev1.GetCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetCampaign_DeniesNonMember(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Test Campaign",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.StatusDraft,
		GmMode:    campaign.GmModeHuman,
		CreatedAt: now,
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetCampaign(contextWithParticipantID("outsider-1"), &statev1.GetCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetCampaign_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Test Campaign",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.StatusDraft,
		GmMode:    campaign.GmModeHuman,
		CreatedAt: now,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"participant-1": {
			ID:             "participant-1",
			CampaignID:     "c1",
			UserID:         "user-1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Participant: participantStore})

	resp, err := svc.GetCampaign(contextWithParticipantID("participant-1"), &statev1.GetCampaignRequest{CampaignId: "c1"})
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
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Test Campaign",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.StatusDraft,
		GmMode:    campaign.GmModeHuman,
		CreatedAt: now,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"participant-1": {
			ID:             "participant-1",
			CampaignID:     "c1",
			UserID:         "user-1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Participant: participantStore})
	resp, err := svc.GetCampaign(contextWithUserID("user-1"), &statev1.GetCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetCampaign returned error: %v", err)
	}
	if resp.Campaign == nil || resp.Campaign.GetId() != "c1" {
		t.Fatalf("campaign = %+v, want id c1", resp.Campaign)
	}
}

func TestEndCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Stores{})
	_, err := svc.EndCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndCampaign_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.EndCampaign(context.Background(), &statev1.EndCampaignRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndCampaign_NotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.EndCampaign(context.Background(), &statev1.EndCampaignRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestEndCampaign_ActiveSessionBlocks(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	participantStore := ownerParticipantStore("c1")
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.StatusActive,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: campaign.GmModeHuman,
	}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	svc := NewCampaignService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore})
	_, err := svc.EndCampaign(contextWithParticipantID("owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestEndCampaign_DraftStatusDisallowed(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	participantStore := ownerParticipantStore("c1")

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.StatusDraft,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: campaign.GmModeHuman,
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore})
	_, err := svc.EndCampaign(contextWithParticipantID("owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestEndCampaign_AllowsManagerAccess(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	participantStore := newFakeParticipantStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"manager-1": {
			ID:             "manager-1",
			CampaignID:     "c1",
			CampaignAccess: "manager",
		},
	}
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"completed"}}`),
		}),
	}}

	svc := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Session:     sessionStore,
			Participant: participantStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
	}

	resp, err := svc.EndCampaign(contextWithParticipantID("manager-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("EndCampaign returned error: %v", err)
	}
	if resp.GetCampaign().GetStatus() != statev1.CampaignStatus_COMPLETED {
		t.Fatalf("campaign status = %v, want %v", resp.GetCampaign().GetStatus(), statev1.CampaignStatus_COMPLETED)
	}
}

func TestEndCampaign_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	sessionStore := newFakeSessionStore()
	participantStore := ownerParticipantStore("c1")
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Event: eventStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.EndCampaign(contextWithParticipantID("owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestEndCampaign_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	participantStore := ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.StatusActive,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: campaign.GmModeHuman,
	}
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
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

	svc := &CampaignService{
		stores:      Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore, Domain: domain},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("campaign-123"),
	}

	resp, err := svc.EndCampaign(contextWithParticipantID("owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
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
	stored, _ := campaignStore.Get(context.Background(), "c1")
	if stored.Status != campaign.StatusCompleted {
		t.Errorf("Stored campaign Status = %v, want %v", stored.Status, campaign.StatusCompleted)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("campaign.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("campaign.updated"))
	}
}

func TestEndCampaign_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	participantStore := ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.StatusActive,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: campaign.GmModeHuman,
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
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

	svc := &CampaignService{
		stores: Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore, Domain: domain},
		clock:  fixedClock(now),
	}

	_, err := svc.EndCampaign(contextWithParticipantID("owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
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
	svc := NewCampaignService(Stores{})
	_, err := svc.ArchiveCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestArchiveCampaign_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.ArchiveCampaign(context.Background(), &statev1.ArchiveCampaignRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestArchiveCampaign_NotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.ArchiveCampaign(context.Background(), &statev1.ArchiveCampaignRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestArchiveCampaign_ActiveSessionBlocks(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	participantStore := ownerParticipantStore("c1")
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	svc := NewCampaignService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore})
	_, err := svc.ArchiveCampaign(contextWithParticipantID("owner-1"), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestArchiveCampaign_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	sessionStore := newFakeSessionStore()
	participantStore := ownerParticipantStore("c1")
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Event: eventStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.ArchiveCampaign(contextWithParticipantID("owner-1"), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestArchiveCampaign_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	participantStore := ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.StatusCompleted,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: campaign.GmModeHuman,
	}
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
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

	svc := &CampaignService{
		stores:      Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore, Domain: domain},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("campaign-123"),
	}

	resp, err := svc.ArchiveCampaign(contextWithParticipantID("owner-1"), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ArchiveCampaign returned error: %v", err)
	}
	if resp.Campaign.Status != statev1.CampaignStatus_ARCHIVED {
		t.Errorf("Campaign Status = %v, want %v", resp.Campaign.Status, statev1.CampaignStatus_ARCHIVED)
	}
	if resp.Campaign.ArchivedAt == nil {
		t.Error("Campaign ArchivedAt is nil")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("campaign.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("campaign.updated"))
	}
}

func TestArchiveCampaign_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	participantStore := ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.StatusCompleted,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: campaign.GmModeHuman,
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
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

	svc := &CampaignService{
		stores: Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore, Domain: domain},
		clock:  fixedClock(now),
	}

	_, err := svc.ArchiveCampaign(contextWithParticipantID("owner-1"), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
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
	svc := NewCampaignService(Stores{})
	_, err := svc.RestoreCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRestoreCampaign_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore, Event: eventStore})
	_, err := svc.RestoreCampaign(context.Background(), &statev1.RestoreCampaignRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRestoreCampaign_NotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore, Event: eventStore})
	_, err := svc.RestoreCampaign(context.Background(), &statev1.RestoreCampaignRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestRestoreCampaign_NotArchivedDisallowed(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := ownerParticipantStore("c1")
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	_, err := svc.RestoreCampaign(contextWithParticipantID("owner-1"), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestRestoreCampaign_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := ownerParticipantStore("c1")
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusArchived}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	_, err := svc.RestoreCampaign(contextWithParticipantID("owner-1"), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestRestoreCampaign_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	archivedAt := now.Add(-24 * time.Hour)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:         "c1",
		Name:       "Test Campaign",
		Status:     campaign.StatusArchived,
		ArchivedAt: &archivedAt,
		System:     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:     campaign.GmModeHuman,
	}
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
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

	svc := &CampaignService{
		stores:      Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("campaign-123"),
	}

	resp, err := svc.RestoreCampaign(contextWithParticipantID("owner-1"), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("RestoreCampaign returned error: %v", err)
	}
	if resp.Campaign.Status != statev1.CampaignStatus_DRAFT {
		t.Errorf("Campaign Status = %v, want %v", resp.Campaign.Status, statev1.CampaignStatus_DRAFT)
	}
	if resp.Campaign.ArchivedAt != nil {
		t.Error("Campaign ArchivedAt should be nil after restore")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("campaign.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("campaign.updated"))
	}
}

func TestRestoreCampaign_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	archivedAt := now.Add(-24 * time.Hour)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:         "c1",
		Name:       "Test Campaign",
		Status:     campaign.StatusArchived,
		ArchivedAt: &archivedAt,
		System:     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:     campaign.GmModeHuman,
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
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

	svc := &CampaignService{
		stores: Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain},
		clock:  fixedClock(now),
	}

	_, err := svc.RestoreCampaign(contextWithParticipantID("owner-1"), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
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
	svc := NewCampaignService(Stores{})
	_, err := svc.SetCampaignCover(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetCampaignCover_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	participantStore := ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:           "c1",
		Name:         "Test Campaign",
		Status:       campaign.StatusActive,
		CoverAssetID: "camp-cover-01",
		System:       commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:       campaign.GmModeHuman,
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
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

	svc := &CampaignService{
		stores: Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain},
		clock:  fixedClock(now),
	}

	resp, err := svc.SetCampaignCover(contextWithParticipantID("owner-1"), &statev1.SetCampaignCoverRequest{
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

// assertStatusCode verifies the gRPC status code for an error.
func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
	statusErr, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T", err)
	}
	if statusErr.Code() != want {
		t.Fatalf("status code = %v, want %v (message: %s)", statusErr.Code(), want, statusErr.Message())
	}
}
