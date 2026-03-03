package app

import (
	"context"
	"net/http"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

func TestNewServiceAndWorkflowResolverContracts(t *testing.T) {
	t.Parallel()

	svc := NewService(nil)
	if IsGatewayHealthy(nil) {
		t.Fatalf("IsGatewayHealthy(nil) = true, want false")
	}
	if svc.ResolveWorkflow("daggerheart") != nil {
		t.Fatalf("ResolveWorkflow() for nil workflow map = non-nil")
	}
	if _, err := svc.ListCampaigns(context.Background()); err == nil {
		t.Fatalf("expected unavailable error for nil gateway")
	}

	workflow := testCreationWorkflow{}
	svcWithWorkflow := NewServiceWithWorkflows(nil, map[string]CharacterCreationWorkflow{"daggerheart": workflow})
	if svcWithWorkflow.ResolveWorkflow(" DAggerHEART ") == nil {
		t.Fatalf("ResolveWorkflow() should normalize workflow key")
	}
}

func TestServiceExportedMethodContracts(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		items:                     []CampaignSummary{{ID: "c1", Name: "Campaign"}},
		campaignName:              "Campaign",
		campaignWorkspace:         CampaignWorkspace{Name: "Campaign", System: "Daggerheart", GMMode: "Human", Status: "Active", Locale: "English (US)", Intent: "Standard", AccessPolicy: "Private", ParticipantCount: "1", CharacterCount: "1", CoverImageURL: "https://cdn.example.com/cover.png"},
		campaignParticipants:      []CampaignParticipant{{ID: "p1", Name: "Owner"}},
		campaignParticipant:       CampaignParticipant{ID: "p1", Name: "Owner", Role: "GM", CampaignAccess: "Owner"},
		campaignCharacters:        []CampaignCharacter{{ID: "char-1", Name: "Hero"}},
		campaignSessions:          []CampaignSession{{ID: "sess-1", Name: "Session One"}},
		campaignInvites:           []CampaignInvite{{ID: "inv-1", ParticipantID: "p1", RecipientUserID: "user-2", Status: "Pending"}},
		authorizationDecision:     campaignAuthorizationDecision{Evaluated: true, Allowed: true},
		characterCreationProgress: CampaignCharacterCreationProgress{NextStep: 1},
		characterCreationCatalog:  CampaignCharacterCreationCatalog{},
		characterCreationProfile:  CampaignCharacterCreationProfile{},
	}
	svc := NewServiceWithWorkflows(gateway, map[string]CharacterCreationWorkflow{"daggerheart": testCreationWorkflow{}})
	ctx := contextWithResolvedUserID("user-1")

	if _, err := svc.ListCampaigns(ctx); err != nil {
		t.Fatalf("ListCampaigns() error = %v", err)
	}
	if _, err := svc.CreateCampaign(ctx, CreateCampaignInput{Name: "New Campaign"}); err != nil {
		t.Fatalf("CreateCampaign() error = %v", err)
	}
	if got := svc.CampaignName(ctx, "c1"); got != "Campaign" {
		t.Fatalf("CampaignName() = %q, want %q", got, "Campaign")
	}
	if _, err := svc.CampaignWorkspace(ctx, "c1"); err != nil {
		t.Fatalf("CampaignWorkspace() error = %v", err)
	}
	if _, err := svc.CampaignParticipants(ctx, "c1"); err != nil {
		t.Fatalf("CampaignParticipants() error = %v", err)
	}
	if _, err := svc.CampaignParticipantEditor(ctx, "c1", "p1"); err != nil {
		t.Fatalf("CampaignParticipantEditor() error = %v", err)
	}
	if _, err := svc.CampaignCharacters(ctx, "c1"); err != nil {
		t.Fatalf("CampaignCharacters() error = %v", err)
	}
	if _, err := svc.CampaignSessions(ctx, "c1"); err != nil {
		t.Fatalf("CampaignSessions() error = %v", err)
	}
	if _, err := svc.CampaignInvites(ctx, "c1"); err != nil {
		t.Fatalf("CampaignInvites() error = %v", err)
	}
	if err := svc.RequireManageCampaign(ctx, "c1"); err != nil {
		t.Fatalf("RequireManageCampaign() error = %v", err)
	}
	updatedName := "Updated Campaign"
	updatedTheme := "Updated Theme"
	updatedLocale := "pt-BR"
	if err := svc.UpdateCampaign(ctx, "c1", UpdateCampaignInput{
		Name:        &updatedName,
		ThemePrompt: &updatedTheme,
		Locale:      &updatedLocale,
	}); err != nil {
		t.Fatalf("UpdateCampaign() error = %v", err)
	}
	if err := svc.StartSession(ctx, "c1", StartSessionInput{Name: "Session"}); err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if err := svc.EndSession(ctx, "c1", EndSessionInput{SessionID: "sess-1"}); err != nil {
		t.Fatalf("EndSession() error = %v", err)
	}
	if _, err := svc.CreateCharacter(ctx, "c1", CreateCharacterInput{Name: "Hero", Kind: CharacterKindPC}); err != nil {
		t.Fatalf("CreateCharacter() error = %v", err)
	}
	if err := svc.UpdateParticipant(ctx, "c1", UpdateParticipantInput{ParticipantID: "p1", Name: "Owner Prime", Role: "gm", Pronouns: "they/them"}); err != nil {
		t.Fatalf("UpdateParticipant() error = %v", err)
	}
	if err := svc.CreateInvite(ctx, "c1", CreateInviteInput{ParticipantID: "p1", RecipientUserID: "user-2"}); err != nil {
		t.Fatalf("CreateInvite() error = %v", err)
	}
	if err := svc.RevokeInvite(ctx, "c1", RevokeInviteInput{InviteID: "inv-1"}); err != nil {
		t.Fatalf("RevokeInvite() error = %v", err)
	}
	if _, err := svc.CampaignCharacterCreation(ctx, "c1", "char-1", language.AmericanEnglish, testCreationWorkflow{}); err != nil {
		t.Fatalf("CampaignCharacterCreation() error = %v", err)
	}
	if _, err := svc.CampaignCharacterCreationProgress(ctx, "c1", "char-1"); err != nil {
		t.Fatalf("CampaignCharacterCreationProgress() error = %v", err)
	}
	if err := svc.ApplyCharacterCreationStep(ctx, "c1", "char-1", &CampaignCharacterCreationStepInput{Details: &CampaignCharacterCreationStepDetails{}}); err != nil {
		t.Fatalf("ApplyCharacterCreationStep() error = %v", err)
	}
	if err := svc.ResetCharacterCreationWorkflow(ctx, "c1", "char-1"); err != nil {
		t.Fatalf("ResetCharacterCreationWorkflow() error = %v", err)
	}
}

func TestUnavailableGatewayFailsClosedForAllMethods(t *testing.T) {
	t.Parallel()

	gw := NewUnavailableGateway()
	ctx := context.Background()

	assertUnavailable := func(t *testing.T, err error, name string) {
		t.Helper()
		if err == nil {
			t.Fatalf("expected %s unavailable error", name)
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
			t.Fatalf("%s HTTPStatus(err) = %d, want %d", name, got, http.StatusServiceUnavailable)
		}
	}

	if _, err := gw.ListCampaigns(ctx); err != nil {
		assertUnavailable(t, err, "ListCampaigns")
	}
	if _, err := gw.CampaignName(ctx, "c1"); err != nil {
		assertUnavailable(t, err, "CampaignName")
	}
	if _, err := gw.CampaignWorkspace(ctx, "c1"); err != nil {
		assertUnavailable(t, err, "CampaignWorkspace")
	}
	if _, err := gw.CampaignParticipants(ctx, "c1"); err != nil {
		assertUnavailable(t, err, "CampaignParticipants")
	}
	if _, err := gw.CampaignParticipant(ctx, "c1", "p1"); err != nil {
		assertUnavailable(t, err, "CampaignParticipant")
	}
	if _, err := gw.CampaignCharacters(ctx, "c1"); err != nil {
		assertUnavailable(t, err, "CampaignCharacters")
	}
	if _, err := gw.CampaignSessions(ctx, "c1"); err != nil {
		assertUnavailable(t, err, "CampaignSessions")
	}
	if _, err := gw.CampaignInvites(ctx, "c1"); err != nil {
		assertUnavailable(t, err, "CampaignInvites")
	}
	if _, err := gw.CharacterCreationProgress(ctx, "c1", "char-1"); err != nil {
		assertUnavailable(t, err, "CharacterCreationProgress")
	}
	if _, err := gw.CharacterCreationCatalog(ctx, language.AmericanEnglish); err != nil {
		assertUnavailable(t, err, "CharacterCreationCatalog")
	}
	if _, err := gw.CharacterCreationProfile(ctx, "c1", "char-1"); err != nil {
		assertUnavailable(t, err, "CharacterCreationProfile")
	}
	if _, err := gw.CreateCampaign(ctx, CreateCampaignInput{Name: "Campaign"}); err != nil {
		assertUnavailable(t, err, "CreateCampaign")
	}
	assertUnavailable(t, gw.UpdateCampaign(ctx, "c1", UpdateCampaignInput{}), "UpdateCampaign")
	assertUnavailable(t, gw.StartSession(ctx, "c1", StartSessionInput{Name: "Session"}), "StartSession")
	assertUnavailable(t, gw.EndSession(ctx, "c1", EndSessionInput{SessionID: "sess-1"}), "EndSession")
	if _, err := gw.CreateCharacter(ctx, "c1", CreateCharacterInput{Name: "Hero", Kind: CharacterKindPC}); err != nil {
		assertUnavailable(t, err, "CreateCharacter")
	}
	assertUnavailable(t, gw.UpdateParticipant(ctx, "c1", UpdateParticipantInput{ParticipantID: "p1"}), "UpdateParticipant")
	assertUnavailable(t, gw.CreateInvite(ctx, "c1", CreateInviteInput{ParticipantID: "p1", RecipientUserID: "user-2"}), "CreateInvite")
	assertUnavailable(t, gw.RevokeInvite(ctx, "c1", RevokeInviteInput{InviteID: "inv-1"}), "RevokeInvite")
	assertUnavailable(t, gw.ApplyCharacterCreationStep(ctx, "c1", "char-1", &CampaignCharacterCreationStepInput{}), "ApplyCharacterCreationStep")
	assertUnavailable(t, gw.ResetCharacterCreationWorkflow(ctx, "c1", "char-1"), "ResetCharacterCreationWorkflow")
}

func TestListingProjectionHelpersProvideDeterministicFallbacks(t *testing.T) {
	t.Parallel()

	short := TruncateCampaignTheme("  Misty harbor nights  ")
	if short != "Misty harbor nights" {
		t.Fatalf("TruncateCampaignTheme(short) = %q", short)
	}

	veryLong := strings.Repeat("x", 96)
	truncated := TruncateCampaignTheme(veryLong)
	if len([]rune(truncated)) <= campaignThemePromptLimit {
		t.Fatalf("expected truncated preview to include ellipsis")
	}

	fallbackCover := CampaignCoverImageURL("", "campaign-1", "invalid-set", "")
	if !strings.HasPrefix(fallbackCover, "/static/campaign-cover-fallback.svg") {
		t.Fatalf("CampaignCoverImageURL fallback = %q", fallbackCover)
	}

	if got := defaultCampaignCoverAssetID(); strings.TrimSpace(got) == "" {
		t.Fatalf("defaultCampaignCoverAssetID() returned empty")
	}
}
