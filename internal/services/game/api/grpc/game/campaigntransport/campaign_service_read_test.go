package campaigntransport

import (
	"context"
	"fmt"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	systems "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

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

	resp, err := svc.ListCampaigns(requestctx.WithParticipantID(context.Background(), "participant-1"), &statev1.ListCampaignsRequest{})
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
			System:    systems.SystemIDDaggerheart,
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
				System:    systems.SystemIDDaggerheart,
				Status:    campaign.StatusDraft,
				GmMode:    campaign.GmModeHuman,
				CreatedAt: time.Now().UTC(),
			},
			{
				ID:        "campaign-002",
				Name:      "Campaign 2",
				System:    systems.SystemIDDaggerheart,
				Status:    campaign.StatusCompleted,
				GmMode:    campaign.GmModeHuman,
				CreatedAt: time.Now().UTC(),
			},
			{
				ID:        "campaign-003",
				Name:      "Campaign 3",
				System:    systems.SystemIDDaggerheart,
				Status:    campaign.StatusActive,
				GmMode:    campaign.GmModeHuman,
				CreatedAt: time.Now().UTC(),
			},
			{
				ID:        "campaign-004",
				Name:      "Campaign 4",
				System:    systems.SystemIDDaggerheart,
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
	_, err := svc.GetCampaign(requestctx.WithParticipantID(context.Background(), "outsider-1"), &statev1.GetCampaignRequest{CampaignId: "c1"})
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

	resp, err := svc.GetCampaign(requestctx.WithParticipantID(context.Background(), "participant-1"), &statev1.GetCampaignRequest{CampaignId: "c1"})
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
	resp, err := svc.GetCampaign(requestctx.WithUserID(context.Background(), "user-1"), &statev1.GetCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetCampaign returned error: %v", err)
	}
	if resp.Campaign == nil || resp.Campaign.GetId() != "c1" {
		t.Fatalf("campaign = %+v, want id c1", resp.Campaign)
	}
}
