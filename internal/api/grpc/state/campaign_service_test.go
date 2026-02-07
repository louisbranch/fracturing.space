package state

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/state/v1"
	"github.com/louisbranch/fracturing.space/internal/state/campaign"
	"github.com/louisbranch/fracturing.space/internal/state/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Stores{})
	_, err := svc.CreateCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_MissingCampaignStore(t *testing.T) {
	svc := NewCampaignService(Stores{})
	_, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateCampaign_MissingEventStore(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore})
	_, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateCampaign_MissingSystem(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore, Event: eventStore})
	_, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		GmMode: statev1.GmMode_HUMAN,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_EmptyName(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore, Event: eventStore})
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
	svc := NewCampaignService(Stores{Campaign: campaignStore, Event: eventStore})
	_, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
		Name:   "Test Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	svc := &CampaignService{
		stores:      Stores{Campaign: campaignStore, Event: eventStore},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("campaign-123"),
	}

	resp, err := svc.CreateCampaign(context.Background(), &statev1.CreateCampaignRequest{
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

	// Verify persisted
	stored, err := campaignStore.Get(context.Background(), "campaign-123")
	if err != nil {
		t.Fatalf("Campaign not persisted: %v", err)
	}
	if stored.Name != "Test Campaign" {
		t.Errorf("Stored campaign Name = %q, want %q", stored.Name, "Test Campaign")
	}
}

func TestListCampaigns_NilRequest(t *testing.T) {
	svc := NewCampaignService(Stores{})
	_, err := svc.ListCampaigns(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListCampaigns_MissingCampaignStore(t *testing.T) {
	svc := NewCampaignService(Stores{})
	_, err := svc.ListCampaigns(context.Background(), &statev1.ListCampaignsRequest{})
	assertStatusCode(t, err, codes.Internal)
}

func TestListCampaigns_EmptyList(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	svc := NewCampaignService(Stores{Campaign: campaignStore})

	resp, err := svc.ListCampaigns(context.Background(), &statev1.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("ListCampaigns returned error: %v", err)
	}
	if len(resp.Campaigns) != 0 {
		t.Errorf("ListCampaigns returned %d campaigns, want 0", len(resp.Campaigns))
	}
}

func TestListCampaigns_WithCampaigns(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	now := time.Now().UTC()
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Name:   "Campaign One",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status: campaign.CampaignStatusDraft,
		GmMode: campaign.GmModeHuman,
	}
	campaignStore.campaigns["c2"] = campaign.Campaign{
		ID:        "c2",
		Name:      "Campaign Two",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.CampaignStatusActive,
		GmMode:    campaign.GmModeAI,
		CreatedAt: now,
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore})

	resp, err := svc.ListCampaigns(context.Background(), &statev1.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("ListCampaigns returned error: %v", err)
	}
	if len(resp.Campaigns) != 2 {
		t.Errorf("ListCampaigns returned %d campaigns, want 2", len(resp.Campaigns))
	}
}

func TestGetCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Stores{})
	_, err := svc.GetCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCampaign_MissingCampaignStore(t *testing.T) {
	svc := NewCampaignService(Stores{})
	_, err := svc.GetCampaign(context.Background(), &statev1.GetCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
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

func TestGetCampaign_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	now := time.Now().UTC()
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:        "c1",
		Name:      "Test Campaign",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.CampaignStatusDraft,
		GmMode:    campaign.GmModeHuman,
		CreatedAt: now,
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore})

	resp, err := svc.GetCampaign(context.Background(), &statev1.GetCampaignRequest{CampaignId: "c1"})
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

func TestEndCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Stores{})
	_, err := svc.EndCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndCampaign_MissingCampaignStore(t *testing.T) {
	svc := NewCampaignService(Stores{Session: newFakeSessionStore()})
	_, err := svc.EndCampaign(context.Background(), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestEndCampaign_MissingSessionStore(t *testing.T) {
	svc := NewCampaignService(Stores{Campaign: newFakeCampaignStore()})
	_, err := svc.EndCampaign(context.Background(), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
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
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.CampaignStatusActive,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: campaign.GmModeHuman,
	}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	svc := NewCampaignService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.EndCampaign(context.Background(), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestEndCampaign_DraftStatusDisallowed(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.CampaignStatusDraft,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: campaign.GmModeHuman,
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.EndCampaign(context.Background(), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestEndCampaign_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.CampaignStatusActive,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: campaign.GmModeHuman,
	}

	svc := &CampaignService{
		stores:      Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("campaign-123"),
	}

	resp, err := svc.EndCampaign(context.Background(), &statev1.EndCampaignRequest{CampaignId: "c1"})
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
	if stored.Status != campaign.CampaignStatusCompleted {
		t.Errorf("Stored campaign Status = %v, want %v", stored.Status, campaign.CampaignStatusCompleted)
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
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	svc := NewCampaignService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.ArchiveCampaign(context.Background(), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestArchiveCampaign_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.CampaignStatusCompleted,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: campaign.GmModeHuman,
	}

	svc := &CampaignService{
		stores:      Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("campaign-123"),
	}

	resp, err := svc.ArchiveCampaign(context.Background(), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ArchiveCampaign returned error: %v", err)
	}
	if resp.Campaign.Status != statev1.CampaignStatus_ARCHIVED {
		t.Errorf("Campaign Status = %v, want %v", resp.Campaign.Status, statev1.CampaignStatus_ARCHIVED)
	}
	if resp.Campaign.ArchivedAt == nil {
		t.Error("Campaign ArchivedAt is nil")
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
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}

	svc := NewCampaignService(Stores{Campaign: campaignStore, Event: eventStore})
	_, err := svc.RestoreCampaign(context.Background(), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestRestoreCampaign_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	archivedAt := now.Add(-24 * time.Hour)

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:         "c1",
		Name:       "Test Campaign",
		Status:     campaign.CampaignStatusArchived,
		ArchivedAt: &archivedAt,
		System:     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:     campaign.GmModeHuman,
	}

	svc := &CampaignService{
		stores:      Stores{Campaign: campaignStore, Event: eventStore},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("campaign-123"),
	}

	resp, err := svc.RestoreCampaign(context.Background(), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("RestoreCampaign returned error: %v", err)
	}
	if resp.Campaign.Status != statev1.CampaignStatus_DRAFT {
		t.Errorf("Campaign Status = %v, want %v", resp.Campaign.Status, statev1.CampaignStatus_DRAFT)
	}
	if resp.Campaign.ArchivedAt != nil {
		t.Error("Campaign ArchivedAt should be nil after restore")
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
