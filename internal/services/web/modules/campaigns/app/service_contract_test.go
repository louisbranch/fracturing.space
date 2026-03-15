package app

import (
	"context"
	"net/http"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

func TestPackageServiceAndWorkflowResolverContracts(t *testing.T) {
	t.Parallel()

	if NewCatalogService(CatalogServiceConfig{}) != nil {
		t.Fatalf("NewCatalogService() returned non-nil service for missing deps")
	}
	if NewWorkspaceService(WorkspaceServiceConfig{}) != nil {
		t.Fatalf("NewWorkspaceService() returned non-nil service for missing deps")
	}
	if NewGameService(GameServiceConfig{}) != nil {
		t.Fatalf("NewGameService() returned non-nil service for missing deps")
	}
	if NewParticipantReadService(ParticipantReadServiceConfig{}, nil) != nil {
		t.Fatalf("NewParticipantReadService() returned non-nil service for missing deps")
	}
	if NewParticipantMutationService(ParticipantMutationServiceConfig{}, nil) != nil {
		t.Fatalf("NewParticipantMutationService() returned non-nil service for missing deps")
	}
	if NewAutomationReadService(AutomationReadServiceConfig{}, nil) != nil {
		t.Fatalf("NewAutomationReadService() returned non-nil service for missing deps")
	}
	if NewAutomationMutationService(AutomationMutationServiceConfig{}, nil) != nil {
		t.Fatalf("NewAutomationMutationService() returned non-nil service for missing deps")
	}
	if NewCharacterReadService(CharacterReadServiceConfig{}, nil) != nil {
		t.Fatalf("NewCharacterReadService() returned non-nil service for missing deps")
	}
	if NewCharacterControlService(CharacterControlServiceConfig{}, nil) != nil {
		t.Fatalf("NewCharacterControlService() returned non-nil service for missing deps")
	}
	if NewCharacterMutationService(CharacterMutationServiceConfig{}, nil) != nil {
		t.Fatalf("NewCharacterMutationService() returned non-nil service for missing deps")
	}
	if NewSessionReadService(SessionReadServiceConfig{}) != nil {
		t.Fatalf("NewSessionReadService() returned non-nil service for missing deps")
	}
	if NewSessionMutationService(SessionMutationServiceConfig{}, nil) != nil {
		t.Fatalf("NewSessionMutationService() returned non-nil service for missing deps")
	}
	if NewInviteReadService(InviteReadServiceConfig{}, nil) != nil {
		t.Fatalf("NewInviteReadService() returned non-nil service for missing deps")
	}
	if NewInviteMutationService(InviteMutationServiceConfig{}, nil) != nil {
		t.Fatalf("NewInviteMutationService() returned non-nil service for missing deps")
	}
	if NewConfigurationService(ConfigurationServiceConfig{}, nil) != nil {
		t.Fatalf("NewConfigurationService() returned non-nil service for missing deps")
	}
	if NewAuthorizationService(nil) != nil {
		t.Fatalf("NewAuthorizationService() returned non-nil service for missing deps")
	}
	if NewCharacterCreationPageService(CharacterCreationServiceConfig{}) != nil {
		t.Fatalf("NewCharacterCreationPageService() returned non-nil service for missing deps")
	}
	if NewCharacterCreationMutationService(CharacterCreationServiceConfig{}, nil) != nil {
		t.Fatalf("NewCharacterCreationMutationService() returned non-nil service for missing deps")
	}
}

func TestParseGameSystemContracts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  GameSystem
		ok    bool
	}{
		{input: "daggerheart", want: GameSystemDaggerheart, ok: true},
		{input: " Daggerheart ", want: GameSystemDaggerheart, ok: true},
		{input: "game_system_daggerheart", want: GameSystemDaggerheart, ok: true},
		{input: "unknown-system", want: GameSystemUnspecified, ok: false},
	}
	for _, tc := range tests {
		got, ok := ParseGameSystem(tc.input)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("ParseGameSystem(%q) = (%q, %v), want (%q, %v)", tc.input, got, ok, tc.want, tc.ok)
		}
	}
}

func TestPackageServiceMethodContracts(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		items:                     []CampaignSummary{{ID: "c1", Name: "Campaign"}},
		campaignName:              "Campaign",
		campaignWorkspace:         CampaignWorkspace{Name: "Campaign", System: "Daggerheart", GMMode: "Human", Status: "Active", Locale: "English (US)", Intent: "Standard", AccessPolicy: "Private", ParticipantCount: "1", CharacterCount: "1", CoverImageURL: "https://cdn.example.com/cover.png"},
		campaignParticipants:      []CampaignParticipant{{ID: "p1", Name: "Owner"}},
		campaignParticipant:       CampaignParticipant{ID: "p1", Name: "Owner", Role: "GM", CampaignAccess: "Owner"},
		campaignCharacters:        []CampaignCharacter{{ID: "char-1", Name: "Hero"}},
		campaignSessions:          []CampaignSession{{ID: "sess-1", Name: "Session One"}},
		campaignSessionReadiness:  CampaignSessionReadiness{Ready: true},
		campaignInvites:           []CampaignInvite{{ID: "inv-1", ParticipantID: "p1", RecipientUserID: "user-2", Status: "Pending"}},
		authorizationDecision:     AuthorizationDecision{Evaluated: true, Allowed: true},
		characterCreationProgress: CampaignCharacterCreationProgress{NextStep: 1},
		characterCreationCatalog:  CampaignCharacterCreationCatalog{},
		characterCreationProfile:  CampaignCharacterCreationProfile{},
	}
	svc := newService(gateway)
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
	if _, err := svc.CampaignParticipantCreator(ctx, "c1"); err != nil {
		t.Fatalf("CampaignParticipantCreator() error = %v", err)
	}
	if _, err := svc.CampaignParticipantEditor(ctx, "c1", "p1"); err != nil {
		t.Fatalf("CampaignParticipantEditor() error = %v", err)
	}
	if _, err := svc.CampaignCharacters(ctx, "c1", CharacterReadContext{}); err != nil {
		t.Fatalf("CampaignCharacters() error = %v", err)
	}
	if _, err := svc.CampaignCharacter(ctx, "c1", "char-1", CharacterReadContext{}); err != nil {
		t.Fatalf("CampaignCharacter() error = %v", err)
	}
	if _, err := svc.CampaignSessions(ctx, "c1"); err != nil {
		t.Fatalf("CampaignSessions() error = %v", err)
	}
	if _, err := svc.CampaignSessionReadiness(ctx, "c1", language.AmericanEnglish); err != nil {
		t.Fatalf("CampaignSessionReadiness() error = %v", err)
	}
	if _, err := svc.CampaignInvites(ctx, "c1"); err != nil {
		t.Fatalf("CampaignInvites() error = %v", err)
	}
	if _, err := svc.SearchInviteUsers(ctx, "c1", SearchInviteUsersInput{ViewerUserID: "user-1", Query: "al"}); err != nil {
		t.Fatalf("SearchInviteUsers() error = %v", err)
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
	if _, err := svc.CreateParticipant(ctx, "c1", CreateParticipantInput{Name: "Pending Seat", Role: "player", CampaignAccess: "member"}); err != nil {
		t.Fatalf("CreateParticipant() error = %v", err)
	}
	if err := svc.UpdateParticipant(ctx, "c1", UpdateParticipantInput{ParticipantID: "p1", Name: "Owner Prime", Role: "gm", Pronouns: "they/them"}); err != nil {
		t.Fatalf("UpdateParticipant() error = %v", err)
	}
	if err := svc.CreateInvite(ctx, "c1", CreateInviteInput{ParticipantID: "p1", RecipientUsername: "alice"}); err != nil {
		t.Fatalf("CreateInvite() error = %v", err)
	}
	if err := svc.RevokeInvite(ctx, "c1", RevokeInviteInput{InviteID: "inv-1"}); err != nil {
		t.Fatalf("RevokeInvite() error = %v", err)
	}
	if _, err := svc.creationPageService.CampaignCharacterCreationProgress(ctx, "c1", "char-1"); err != nil {
		t.Fatalf("CampaignCharacterCreationProgress() error = %v", err)
	}
	if _, err := svc.creationPageService.CampaignCharacterCreationCatalog(ctx, language.AmericanEnglish); err != nil {
		t.Fatalf("CampaignCharacterCreationCatalog() error = %v", err)
	}
	if _, err := svc.creationPageService.CampaignCharacterCreationProfile(ctx, "c1", "char-1"); err != nil {
		t.Fatalf("CampaignCharacterCreationProfile() error = %v", err)
	}
	if err := svc.creationMutationService.ApplyCharacterCreationStep(ctx, "c1", "char-1", &CampaignCharacterCreationStepInput{Details: &CampaignCharacterCreationStepDetails{}}); err != nil {
		t.Fatalf("ApplyCharacterCreationStep() error = %v", err)
	}
	if err := svc.creationMutationService.ResetCharacterCreationWorkflow(ctx, "c1", "char-1"); err != nil {
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
	if _, err := gw.CampaignCharacters(ctx, "c1", CharacterReadContext{}); err != nil {
		assertUnavailable(t, err, "CampaignCharacters")
	}
	if _, err := gw.CampaignCharacter(ctx, "c1", "char-1", CharacterReadContext{}); err != nil {
		assertUnavailable(t, err, "CampaignCharacter")
	}
	if _, err := gw.CampaignSessions(ctx, "c1"); err != nil {
		assertUnavailable(t, err, "CampaignSessions")
	}
	if _, err := gw.CampaignSessionReadiness(ctx, "c1", language.AmericanEnglish); err != nil {
		assertUnavailable(t, err, "CampaignSessionReadiness")
	}
	if _, err := gw.CampaignInvites(ctx, "c1"); err != nil {
		assertUnavailable(t, err, "CampaignInvites")
	}
	if _, err := gw.SearchInviteUsers(ctx, SearchInviteUsersInput{ViewerUserID: "user-1", Query: "al"}); err != nil {
		assertUnavailable(t, err, "SearchInviteUsers")
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
	assertUnavailable(t, gw.UpdateCharacter(ctx, "c1", "char-1", UpdateCharacterInput{}), "UpdateCharacter")
	if _, err := gw.CreateParticipant(ctx, "c1", CreateParticipantInput{Name: "Pending Seat", Role: "player", CampaignAccess: "member"}); err != nil {
		assertUnavailable(t, err, "CreateParticipant")
	}
	assertUnavailable(t, gw.UpdateParticipant(ctx, "c1", UpdateParticipantInput{ParticipantID: "p1"}), "UpdateParticipant")
	assertUnavailable(t, gw.CreateInvite(ctx, "c1", CreateInviteInput{ParticipantID: "p1", RecipientUsername: "alice"}), "CreateInvite")
	assertUnavailable(t, gw.RevokeInvite(ctx, "c1", RevokeInviteInput{InviteID: "inv-1"}), "RevokeInvite")
	assertUnavailable(t, gw.ApplyCharacterCreationStep(ctx, "c1", "char-1", &CampaignCharacterCreationStepInput{}), "ApplyCharacterCreationStep")
	assertUnavailable(t, gw.ResetCharacterCreationWorkflow(ctx, "c1", "char-1"), "ResetCharacterCreationWorkflow")
}

func TestCampaignInvitesSortsByStatusOrderThenID(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true},
		campaignInvites: []CampaignInvite{
			{ID: "inv-4", Status: "Revoked"},
			{ID: "inv-2", Status: "Declined"},
			{ID: "inv-5", Status: "Claimed"},
			{ID: "inv-1", Status: "Pending"},
			{ID: "inv-3", Status: "Pending"},
		},
	})

	invites, err := svc.inviteReadService.CampaignInvites(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignInvites() error = %v", err)
	}
	if len(invites) != 5 {
		t.Fatalf("len(invites) = %d, want 5", len(invites))
	}

	got := []string{
		invites[0].ID + ":" + invites[0].Status,
		invites[1].ID + ":" + invites[1].Status,
		invites[2].ID + ":" + invites[2].Status,
		invites[3].ID + ":" + invites[3].Status,
		invites[4].ID + ":" + invites[4].Status,
	}
	want := []string{
		"inv-1:Pending",
		"inv-3:Pending",
		"inv-5:Claimed",
		"inv-2:Declined",
		"inv-4:Revoked",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("invites[%d] = %q, want %q (full order: %+v)", i, got[i], want[i], got)
		}
	}
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
