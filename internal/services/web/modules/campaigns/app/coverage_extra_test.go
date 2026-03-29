package app

import (
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestExportedCampaignAppCharacterWrappersDelegate(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{
			Evaluated:  true,
			Allowed:    true,
			ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL",
		},
		campaignCharacters: []CampaignCharacter{{
			ID:                 "char-1",
			Name:               "Hero",
			Kind:               "pc",
			Owner:              "Owner",
			OwnerParticipantID: "p-1",
		}},
		campaignParticipants: []CampaignParticipant{
			{ID: "p-1", Name: "Owner"},
			{ID: "p-2", Name: "Player"},
		},
	}
	svc := newService(gateway)
	ctx := contextWithResolvedUserID("user-1")

	editor, err := svc.CampaignCharacterEditor(ctx, "c1", "char-1", CharacterReadContext{})
	if err != nil {
		t.Fatalf("CampaignCharacterEditor() error = %v", err)
	}
	if editor.Character.ID != "char-1" || !editor.Character.CanEdit {
		t.Fatalf("CampaignCharacterEditor() = %#v", editor)
	}

	ownership, err := svc.CampaignCharacterOwnership(ctx, "c1", "char-1", CharacterReadContext{})
	if err != nil {
		t.Fatalf("CampaignCharacterOwnership() error = %v", err)
	}
	if !ownership.CanManageOwnership {
		t.Fatalf("CampaignCharacterOwnership().CanManageOwnership = false")
	}
	if ownership.CurrentOwnerName != "Owner" {
		t.Fatalf("CurrentOwnerName = %q, want %q", ownership.CurrentOwnerName, "Owner")
	}
	if len(ownership.Options) != 3 || !ownership.Options[1].Selected || ownership.Options[1].ParticipantID != "p-1" {
		t.Fatalf("CampaignCharacterOwnership().Options = %#v", ownership.Options)
	}

	if err := svc.UpdateCharacter(ctx, "c1", "char-1", UpdateCharacterInput{Name: "  Hero Prime  ", Pronouns: "  they/them  "}); err != nil {
		t.Fatalf("UpdateCharacter() error = %v", err)
	}
	if gateway.lastUpdateCharacterCampaignID != "c1" || gateway.lastUpdateCharacterID != "char-1" {
		t.Fatalf("update target = (%q, %q)", gateway.lastUpdateCharacterCampaignID, gateway.lastUpdateCharacterID)
	}
	if gateway.lastUpdateCharacterInput.Name != "Hero Prime" || gateway.lastUpdateCharacterInput.Pronouns != "they/them" {
		t.Fatalf("UpdateCharacter input = %#v", gateway.lastUpdateCharacterInput)
	}

	if err := svc.DeleteCharacter(ctx, "c1", "char-1"); err != nil {
		t.Fatalf("DeleteCharacter() error = %v", err)
	}
	if gateway.lastDeleteCharacterCampaignID != "c1" || gateway.lastDeleteCharacterID != "char-1" {
		t.Fatalf("delete target = (%q, %q)", gateway.lastDeleteCharacterCampaignID, gateway.lastDeleteCharacterID)
	}

	if err := svc.SetCharacterOwner(ctx, "c1", "char-1", "  p-2  "); err != nil {
		t.Fatalf("SetCharacterOwner() error = %v", err)
	}
	if gateway.lastSetCharacterOwnerCampaignID != "c1" || gateway.lastSetCharacterOwnerCharacterID != "char-1" {
		t.Fatalf("ownership target = (%q, %q)", gateway.lastSetCharacterOwnerCampaignID, gateway.lastSetCharacterOwnerCharacterID)
	}
	if gateway.lastSetCharacterOwnerParticipantID != "p-2" {
		t.Fatalf("participant id = %q, want %q", gateway.lastSetCharacterOwnerParticipantID, "p-2")
	}
}

func TestExportedCampaignAppAIBindingAndAuthorizationWrappers(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{
			Evaluated:  true,
			Allowed:    true,
			ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL",
		},
		campaignParticipants: []CampaignParticipant{{
			ID:             "p-1",
			UserID:         "user-1",
			Name:           "Owner",
			CampaignAccess: "owner",
		}},
		campaignAIAgents: []CampaignAIAgentOption{
			{ID: "agent-1", Label: "Narrator", Enabled: true},
			{ID: "", Label: "skip", Enabled: true},
			{ID: "agent-2", Label: "Guide", Enabled: false},
		},
	}
	svc := newService(gateway)
	ctx := contextWithResolvedUserID("user-1")

	summary, err := svc.CampaignAIBindingSummary(ctx, "c1", "agent-1", "hybrid")
	if err != nil {
		t.Fatalf("CampaignAIBindingSummary() error = %v", err)
	}
	if summary.Status != CampaignAIBindingStatusConfigured || !summary.CanManage {
		t.Fatalf("CampaignAIBindingSummary() = %#v", summary)
	}

	settings, err := svc.CampaignAIBindingSettings(ctx, "c1", "agent-2")
	if err != nil {
		t.Fatalf("CampaignAIBindingSettings() error = %v", err)
	}
	if settings.CurrentID != "agent-2" || settings.Unavailable {
		t.Fatalf("CampaignAIBindingSettings() = %#v", settings)
	}
	if len(settings.Options) != 2 {
		t.Fatalf("len(Options) = %d, want 2", len(settings.Options))
	}
	if settings.Options[1].ID != "agent-2" || !settings.Options[1].Selected {
		t.Fatalf("selected option = %#v", settings.Options)
	}

	if err := svc.UpdateCampaignAIBinding(ctx, "c1", UpdateCampaignAIBindingInput{AIAgentID: "  agent-2  "}); err != nil {
		t.Fatalf("UpdateCampaignAIBinding() error = %v", err)
	}
	if gateway.lastUpdateCampaignAIBindingInput.AIAgentID != "agent-2" {
		t.Fatalf("AIAgentID = %q, want %q", gateway.lastUpdateCampaignAIBindingInput.AIAgentID, "agent-2")
	}

	if err := svc.RequireManageParticipants(ctx, "c1"); err != nil {
		t.Fatalf("RequireManageParticipants() error = %v", err)
	}
	if err := svc.RequireManageSession(ctx, "c1"); err != nil {
		t.Fatalf("RequireManageSession() error = %v", err)
	}
	if err := svc.RequireManageInvites(ctx, "c1"); err != nil {
		t.Fatalf("RequireManageInvites() error = %v", err)
	}
	if err := svc.RequireMutateCharacters(ctx, "c1"); err != nil {
		t.Fatalf("RequireMutateCharacters() error = %v", err)
	}

	gateway.campaignParticipant = CampaignParticipant{
		ID:             "p-1",
		Name:           "Owner",
		CampaignAccess: "owner",
	}
	if err := svc.DeleteParticipant(ctx, "c1", "p-1"); err != nil {
		t.Fatalf("DeleteParticipant() error = %v", err)
	}
	if gateway.lastDeleteParticipantCampaignID != "c1" || gateway.lastDeleteParticipantID != "p-1" {
		t.Fatalf("delete participant target = (%q, %q)", gateway.lastDeleteParticipantCampaignID, gateway.lastDeleteParticipantID)
	}
}

func TestCampaignUnavailableGatewayExtraMethodsFailClosed(t *testing.T) {
	t.Parallel()

	gw := NewUnavailableGateway()
	ctx := contextWithResolvedUserID("user-1")

	assertUnavailable := func(t *testing.T, err error, name string) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s error = nil", name)
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
			t.Fatalf("%s HTTPStatus(err) = %d, want %d", name, got, http.StatusServiceUnavailable)
		}
	}

	if _, err := gw.CampaignAIAgents(ctx); err != nil {
		assertUnavailable(t, err, "CampaignAIAgents")
	}
	assertUnavailable(t, gw.UpdateCampaignAIBinding(ctx, "c1", UpdateCampaignAIBindingInput{AIAgentID: "agent-1"}), "UpdateCampaignAIBinding")
	assertUnavailable(t, gw.DeleteCharacter(ctx, "c1", "char-1"), "DeleteCharacter")
	assertUnavailable(t, gw.SetCharacterOwner(ctx, "c1", "char-1", "p-1"), "SetCharacterOwner")
	assertUnavailable(t, gw.DeleteParticipant(ctx, "c1", "p-1"), "DeleteParticipant")
	if _, err := gw.CanCampaignAction(ctx, "c1", campaignAuthzActionManage, campaignAuthzResourceCampaign, nil); err != nil {
		assertUnavailable(t, err, "CanCampaignAction")
	}
	if _, err := gw.BatchCanCampaignAction(ctx, "c1", []AuthorizationCheck{{CheckID: "check-1"}}); err != nil {
		assertUnavailable(t, err, "BatchCanCampaignAction")
	}
}
