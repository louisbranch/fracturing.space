package app

import (
	"context"
	"errors"
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestMissingGatewayMutationMethodsFailClosed(t *testing.T) {
	t.Parallel()

	svc := newService(nil)
	ctx := contextWithResolvedUserID("user-1")
	tests := []struct {
		name       string
		run        func() error
		wantStatus int
	}{
		{name: "update campaign", run: func() error { return svc.updateCampaign(ctx, "c1", UpdateCampaignInput{}) }, wantStatus: http.StatusForbidden},
		{name: "update campaign ai binding", run: func() error {
			return svc.updateCampaignAIBinding(ctx, "c1", UpdateCampaignAIBindingInput{AIAgentID: "agent-1"})
		}, wantStatus: http.StatusForbidden},
		{name: "start session", run: func() error { return svc.startSession(ctx, "c1", StartSessionInput{Name: "Session One"}) }, wantStatus: http.StatusForbidden},
		{name: "end session", run: func() error { return svc.endSession(ctx, "c1", EndSessionInput{SessionID: "sess-1"}) }, wantStatus: http.StatusForbidden},
		{name: "create character", run: func() error {
			_, err := svc.createCharacter(ctx, "c1", CreateCharacterInput{Name: "Hero", Kind: CharacterKindPC})
			return err
		}, wantStatus: http.StatusForbidden},
		{name: "create participant", run: func() error {
			_, err := svc.createParticipant(ctx, "c1", CreateParticipantInput{Name: "Pending Seat", Role: "player", CampaignAccess: "member"})
			return err
		}, wantStatus: http.StatusForbidden},
		{name: "update participant", run: func() error {
			return svc.updateParticipant(ctx, "c1", UpdateParticipantInput{ParticipantID: "p-1", Name: "Player One", Role: "player"})
		}, wantStatus: http.StatusServiceUnavailable},
		{name: "create invite", run: func() error {
			return svc.createInvite(ctx, "c1", CreateInviteInput{ParticipantID: "p-1", RecipientUsername: "alice"})
		}, wantStatus: http.StatusForbidden},
		{name: "revoke invite", run: func() error { return svc.revokeInvite(ctx, "c1", RevokeInviteInput{InviteID: "inv-1"}) }, wantStatus: http.StatusForbidden},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			if err == nil {
				t.Fatalf("expected forbidden error")
			}
			if got := apperrors.HTTPStatus(err); got != tc.wantStatus {
				t.Fatalf("HTTPStatus(err) = %d, want %d", got, tc.wantStatus)
			}
		})
	}
}
func TestMutationMethodsDelegateToGateway(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{
			Evaluated:           true,
			Allowed:             true,
			ReasonCode:          "AUTHZ_ALLOW_ACCESS_LEVEL",
			ActorCampaignAccess: "owner",
		},
	}
	svc := newService(gateway)
	ctx := contextWithResolvedUserID("user-1")

	if err := svc.startSession(ctx, "c1", StartSessionInput{Name: "Session One"}); err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	updatedName := "Campaign One"
	if err := svc.updateCampaign(ctx, "c1", UpdateCampaignInput{Name: &updatedName}); err != nil {
		t.Fatalf("UpdateCampaign() error = %v", err)
	}
	if err := svc.updateCampaignAIBinding(ctx, "c1", UpdateCampaignAIBindingInput{AIAgentID: "agent-1"}); err != nil {
		t.Fatalf("UpdateCampaignAIBinding() error = %v", err)
	}
	if err := svc.endSession(ctx, "c1", EndSessionInput{SessionID: "sess-1"}); err != nil {
		t.Fatalf("EndSession() error = %v", err)
	}
	if _, err := svc.createCharacter(ctx, "c1", CreateCharacterInput{Name: "Hero", Kind: CharacterKindPC}); err != nil {
		t.Fatalf("createCharacter() error = %v", err)
	}
	if _, err := svc.createParticipant(ctx, "c1", CreateParticipantInput{Name: "Pending Seat", Role: "player", CampaignAccess: "manager"}); err != nil {
		t.Fatalf("CreateParticipant() error = %v", err)
	}
	gateway.campaignParticipant = CampaignParticipant{
		ID:             "p-1",
		Name:           "Player One",
		Role:           "player",
		CampaignAccess: "member",
	}
	if err := svc.updateParticipant(ctx, "c1", UpdateParticipantInput{ParticipantID: "p-1", Name: "Player Prime", Role: "gm"}); err != nil {
		t.Fatalf("UpdateParticipant() error = %v", err)
	}
	if err := svc.createInvite(ctx, "c1", CreateInviteInput{ParticipantID: "p-1", RecipientUsername: "alice"}); err != nil {
		t.Fatalf("CreateInvite() error = %v", err)
	}
	if err := svc.revokeInvite(ctx, "c1", RevokeInviteInput{InviteID: "inv-1"}); err != nil {
		t.Fatalf("RevokeInvite() error = %v", err)
	}

	want := []string{"start", "update-campaign", "update-campaign-ai-binding", "end", "create-character", "create-participant", "update-participant", "create-invite", "revoke-invite"}
	if len(gateway.calls) != len(want) {
		t.Fatalf("len(calls) = %d, want %d (%v)", len(gateway.calls), len(want), gateway.calls)
	}
	for i := range want {
		if gateway.calls[i] != want[i] {
			t.Fatalf("calls[%d] = %q, want %q", i, gateway.calls[i], want[i])
		}
	}
	if gateway.lastStartSessionInput.Name != "Session One" {
		t.Fatalf("start session input name = %q, want %q", gateway.lastStartSessionInput.Name, "Session One")
	}
	if gateway.lastEndSessionInput.SessionID != "sess-1" {
		t.Fatalf("end session input session id = %q, want %q", gateway.lastEndSessionInput.SessionID, "sess-1")
	}
	if gateway.lastCreateParticipantInput.Name != "Pending Seat" || gateway.lastCreateParticipantInput.Role != "player" || gateway.lastCreateParticipantInput.CampaignAccess != "manager" {
		t.Fatalf("create participant input = %#v", gateway.lastCreateParticipantInput)
	}
	if gateway.lastCreateInviteInput.ParticipantID != "p-1" || gateway.lastCreateInviteInput.RecipientUsername != "alice" {
		t.Fatalf("create invite input = %#v", gateway.lastCreateInviteInput)
	}
	if gateway.lastRevokeInviteInput.InviteID != "inv-1" {
		t.Fatalf("revoke invite input invite id = %q, want %q", gateway.lastRevokeInviteInput.InviteID, "inv-1")
	}
}

func TestMutationMethodsDenyMemberCampaignAccess(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: false, ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED"},
	}
	svc := newService(gateway)
	err := svc.startSession(contextWithResolvedUserID("user-1"), "c1", StartSessionInput{Name: "Session One"})
	if err == nil {
		t.Fatalf("expected forbidden error for member mutation attempt")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusForbidden {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusForbidden)
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
	}
}

func TestMutationMethodsAllowManagerAndOwnerCampaignAccess(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		access string
	}{
		{name: "manager", access: "Manager"},
		{name: "owner", access: "Owner"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gateway := &campaignGatewayStub{
				authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
				campaignParticipants:  []CampaignParticipant{{ID: "p-1", UserID: "user-1", CampaignAccess: tc.access}},
			}
			svc := newService(gateway)
			if err := svc.startSession(contextWithResolvedUserID("user-1"), "c1", StartSessionInput{Name: "Session One"}); err != nil {
				t.Fatalf("StartSession() error = %v", err)
			}
			if len(gateway.calls) != 1 || gateway.calls[0] != "start" {
				t.Fatalf("mutation gateway calls = %v, want [start]", gateway.calls)
			}
		})
	}
}

func TestMutationMethodsUseAuthorizationGatewayDecision(t *testing.T) {
	t.Parallel()

	t.Run("allow", func(t *testing.T) {
		t.Parallel()
		gateway := &campaignGatewayStub{
			authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
		}
		svc := newService(gateway)
		if err := svc.startSession(contextWithResolvedUserID("user-1"), "c1", StartSessionInput{Name: "Session One"}); err != nil {
			t.Fatalf("StartSession() error = %v", err)
		}
		if gateway.authorizationCalls != 1 {
			t.Fatalf("authorization calls = %d, want 1", gateway.authorizationCalls)
		}
		if len(gateway.authorizationRequests) != 1 {
			t.Fatalf("authorization requests = %d, want 1", len(gateway.authorizationRequests))
		}
		if got := gateway.authorizationRequests[0].Action; got != campaignAuthzActionManage {
			t.Fatalf("authorization action = %v, want %v", got, campaignAuthzActionManage)
		}
		if got := gateway.authorizationRequests[0].Resource; got != campaignAuthzResourceSession {
			t.Fatalf("authorization resource = %v, want %v", got, campaignAuthzResourceSession)
		}
		if len(gateway.calls) != 1 || gateway.calls[0] != "start" {
			t.Fatalf("mutation gateway calls = %v, want [start]", gateway.calls)
		}
	})

	t.Run("deny", func(t *testing.T) {
		t.Parallel()
		gateway := &campaignGatewayStub{
			authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: false, ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED"},
		}
		svc := newService(gateway)
		err := svc.startSession(contextWithResolvedUserID("user-1"), "c1", StartSessionInput{Name: "Session One"})
		if err == nil {
			t.Fatal("expected forbidden error")
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusForbidden {
			t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusForbidden)
		}
		if gateway.authorizationCalls != 1 {
			t.Fatalf("authorization calls = %d, want 1", gateway.authorizationCalls)
		}
		if len(gateway.authorizationRequests) != 1 {
			t.Fatalf("authorization requests = %d, want 1", len(gateway.authorizationRequests))
		}
		if got := gateway.authorizationRequests[0].Action; got != campaignAuthzActionManage {
			t.Fatalf("authorization action = %v, want %v", got, campaignAuthzActionManage)
		}
		if got := gateway.authorizationRequests[0].Resource; got != campaignAuthzResourceSession {
			t.Fatalf("authorization resource = %v, want %v", got, campaignAuthzResourceSession)
		}
		if len(gateway.calls) != 0 {
			t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
		}
	})
}

func TestCreateParticipantRejectsHumanGMForAIGMCampaigns(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignWorkspace: CampaignWorkspace{ID: "c1", Name: "Campaign", GMMode: "ai"},
		authorizationDecision: AuthorizationDecision{
			Evaluated: true,
			Allowed:   true,
		},
	}
	svc := newService(gateway)

	_, err := svc.createParticipant(contextWithResolvedUserID("user-1"), "c1", CreateParticipantInput{
		Name:           "Pending GM",
		Role:           "gm",
		CampaignAccess: "member",
	})
	if err == nil {
		t.Fatalf("expected createParticipant() invalid input error")
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.ai_gm_campaign_disallows_human_gm_participants" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.ai_gm_campaign_disallows_human_gm_participants")
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
	}
}

func TestUpdateParticipantRejectsHumanGMForAIGMCampaigns(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignWorkspace: CampaignWorkspace{ID: "c1", Name: "Campaign", GMMode: "ai"},
		campaignParticipant: CampaignParticipant{
			ID:             "p-1",
			Name:           "Pending GM",
			Role:           "player",
			CampaignAccess: "member",
			Controller:     "human",
		},
		authorizationDecision: AuthorizationDecision{
			Evaluated: true,
			Allowed:   true,
		},
	}
	svc := newService(gateway)

	err := svc.updateParticipant(contextWithResolvedUserID("user-1"), "c1", UpdateParticipantInput{
		ParticipantID: "p-1",
		Name:          "Pending GM",
		Role:          "gm",
	})
	if err == nil {
		t.Fatalf("expected updateParticipant() invalid input error")
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.ai_gm_campaign_disallows_human_gm_participants" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.ai_gm_campaign_disallows_human_gm_participants")
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
	}
}

func TestMutationMethodsRequestExpectedCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		run          func(testServiceBundle) error
		wantAction   AuthorizationAction
		wantResource AuthorizationResource
	}{
		{
			name: "update campaign",
			run: func(s testServiceBundle) error {
				name := "Campaign Prime"
				theme := "New theme"
				locale := "pt-BR"
				return s.updateCampaign(contextWithResolvedUserID("user-1"), "c1", UpdateCampaignInput{
					Name:        &name,
					ThemePrompt: &theme,
					Locale:      &locale,
				})
			},
			wantAction:   campaignAuthzActionManage,
			wantResource: campaignAuthzResourceCampaign,
		},
		{
			name: "start session",
			run: func(s testServiceBundle) error {
				return s.startSession(contextWithResolvedUserID("user-1"), "c1", StartSessionInput{Name: "Session One"})
			},
			wantAction:   campaignAuthzActionManage,
			wantResource: campaignAuthzResourceSession,
		},
		{
			name: "end session",
			run: func(s testServiceBundle) error {
				return s.endSession(contextWithResolvedUserID("user-1"), "c1", EndSessionInput{SessionID: "sess-1"})
			},
			wantAction:   campaignAuthzActionManage,
			wantResource: campaignAuthzResourceSession,
		},
		{
			name: "create character",
			run: func(s testServiceBundle) error {
				_, err := s.createCharacter(contextWithResolvedUserID("user-1"), "c1", CreateCharacterInput{Name: "Hero", Kind: CharacterKindPC})
				return err
			},
			wantAction:   campaignAuthzActionMutate,
			wantResource: campaignAuthzResourceCharacter,
		},
		{
			name: "create invite",
			run: func(s testServiceBundle) error {
				return s.createInvite(contextWithResolvedUserID("user-1"), "c1", CreateInviteInput{ParticipantID: "p-1", RecipientUsername: "alice"})
			},
			wantAction:   campaignAuthzActionManage,
			wantResource: campaignAuthzResourceInvite,
		},
		{
			name: "revoke invite",
			run: func(s testServiceBundle) error {
				return s.revokeInvite(contextWithResolvedUserID("user-1"), "c1", RevokeInviteInput{InviteID: "inv-1"})
			},
			wantAction:   campaignAuthzActionManage,
			wantResource: campaignAuthzResourceInvite,
		},
		{
			name: "update participant",
			run: func(s testServiceBundle) error {
				return s.updateParticipant(contextWithResolvedUserID("user-1"), "c1", UpdateParticipantInput{
					ParticipantID: "p-1",
					Name:          "Player Prime",
					Role:          "gm",
					Pronouns:      "they/them",
				})
			},
			wantAction:   campaignAuthzActionManage,
			wantResource: campaignAuthzResourceParticipant,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gateway := &campaignGatewayStub{
				authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
				campaignParticipant: CampaignParticipant{
					ID:             "p-1",
					Name:           "Player One",
					Role:           "player",
					CampaignAccess: "member",
				},
			}
			svc := newService(gateway)
			if err := tc.run(svc); err != nil {
				t.Fatalf("mutation call error = %v", err)
			}
			if len(gateway.authorizationRequests) != 1 {
				t.Fatalf("authorization requests = %d, want 1", len(gateway.authorizationRequests))
			}
			req := gateway.authorizationRequests[0]
			if req.Action != tc.wantAction {
				t.Fatalf("authorization action = %v, want %v", req.Action, tc.wantAction)
			}
			if req.Resource != tc.wantResource {
				t.Fatalf("authorization resource = %v, want %v", req.Resource, tc.wantResource)
			}
		})
	}
}

func TestCreateCharacterAllowsMemberCampaignAccess(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
	}
	svc := newService(gateway)
	if _, err := svc.createCharacter(contextWithResolvedUserID("user-1"), "c1", CreateCharacterInput{Name: "Hero", Kind: CharacterKindPC}); err != nil {
		t.Fatalf("createCharacter() error = %v", err)
	}
	if len(gateway.calls) != 1 || gateway.calls[0] != "create-character" {
		t.Fatalf("mutation gateway calls = %v, want [create-character]", gateway.calls)
	}
}

func TestCreateCharacterRejectsActiveSession(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
		campaignSessions:      []CampaignSession{{ID: "sess-1", Status: "active"}},
	}
	svc := newService(gateway)

	_, err := svc.createCharacter(contextWithResolvedUserID("user-1"), "c1", CreateCharacterInput{Name: "Hero", Kind: CharacterKindPC})
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusConflict {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusConflict)
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.active_session_blocks_character_mutation" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.active_session_blocks_character_mutation")
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
	}
}

func TestUpdateCharacterUsesTargetScopedAuthorizationAndRejectsActiveSession(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
		campaignSessions:      []CampaignSession{{ID: "sess-1", Status: "active"}},
	}
	svc := newService(gateway)

	err := svc.updateCharacter(contextWithResolvedUserID("user-1"), "c1", "char-1", UpdateCharacterInput{Name: "Hero Prime", Pronouns: "they/them"})
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusConflict {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusConflict)
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.active_session_blocks_character_mutation" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.active_session_blocks_character_mutation")
	}
	if len(gateway.authorizationRequests) != 1 {
		t.Fatalf("authorization requests = %d, want 1", len(gateway.authorizationRequests))
	}
	req := gateway.authorizationRequests[0]
	if req.Action != campaignAuthzActionMutate {
		t.Fatalf("authorization action = %v, want %v", req.Action, campaignAuthzActionMutate)
	}
	if req.Resource != campaignAuthzResourceCharacter {
		t.Fatalf("authorization resource = %v, want %v", req.Resource, campaignAuthzResourceCharacter)
	}
	if req.Target == nil || req.Target.ResourceID != "char-1" {
		t.Fatalf("authorization target = %#v, want character target char-1", req.Target)
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
	}
}

func TestDeleteCharacterUsesTargetScopedAuthorizationAndDelegates(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
	}
	svc := newService(gateway)

	if err := svc.deleteCharacter(contextWithResolvedUserID("user-1"), "c1", "char-1"); err != nil {
		t.Fatalf("deleteCharacter() error = %v", err)
	}
	if len(gateway.authorizationRequests) != 1 {
		t.Fatalf("authorization requests = %d, want 1", len(gateway.authorizationRequests))
	}
	req := gateway.authorizationRequests[0]
	if req.Action != campaignAuthzActionMutate || req.Resource != campaignAuthzResourceCharacter {
		t.Fatalf("authorization request = %+v, want mutate character", req)
	}
	if req.Target == nil || req.Target.ResourceID != "char-1" {
		t.Fatalf("authorization target = %#v, want char-1", req.Target)
	}
	if len(gateway.calls) != 1 || gateway.calls[0] != "delete-character" {
		t.Fatalf("mutation gateway calls = %v, want [delete-character]", gateway.calls)
	}
}

func TestSetCharacterControllerUsesManageAuthorizationAndDelegates(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
	}
	svc := newService(gateway)

	if err := svc.setCharacterController(contextWithResolvedUserID("user-1"), "c1", "char-1", "p-2"); err != nil {
		t.Fatalf("setCharacterController() error = %v", err)
	}
	if len(gateway.authorizationRequests) != 1 {
		t.Fatalf("authorization requests = %d, want 1", len(gateway.authorizationRequests))
	}
	req := gateway.authorizationRequests[0]
	if req.Action != campaignAuthzActionManage || req.Resource != campaignAuthzResourceCharacter {
		t.Fatalf("authorization request = %+v, want manage character", req)
	}
	if req.Target == nil || req.Target.ResourceID != "char-1" {
		t.Fatalf("authorization target = %#v, want char-1", req.Target)
	}
	if gateway.lastSetCharacterControllerParticipantID != "p-2" {
		t.Fatalf("participant id = %q, want %q", gateway.lastSetCharacterControllerParticipantID, "p-2")
	}
	if len(gateway.calls) != 1 || gateway.calls[0] != "set-character-controller" {
		t.Fatalf("mutation gateway calls = %v, want [set-character-controller]", gateway.calls)
	}
}

func TestClaimCharacterControlRequiresViewerParticipantAndDelegates(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignParticipants: []CampaignParticipant{
			{ID: "p-1", UserID: "user-1", Name: "Ariadne"},
		},
	}
	svc := newService(gateway)

	if err := svc.claimCharacterControl(contextWithResolvedUserID("user-1"), "c1", "char-1", "user-1"); err != nil {
		t.Fatalf("claimCharacterControl() error = %v", err)
	}
	if len(gateway.calls) != 1 || gateway.calls[0] != "claim-character-control" {
		t.Fatalf("mutation gateway calls = %v, want [claim-character-control]", gateway.calls)
	}
	if gateway.authorizationCalls != 0 {
		t.Fatalf("authorization calls = %d, want 0", gateway.authorizationCalls)
	}
}

func TestReleaseCharacterControlRejectsUserWithoutParticipantSeat(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignParticipants: []CampaignParticipant{
			{ID: "p-2", UserID: "user-2", Name: "Moss"},
		},
	}
	svc := newService(gateway)

	err := svc.releaseCharacterControl(contextWithResolvedUserID("user-1"), "c1", "char-1", "user-1")
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusForbidden {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusForbidden)
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
	}
}

func TestCreateCharacterValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true}})

	if _, err := svc.createCharacter(contextWithResolvedUserID("user-1"), "c1", CreateCharacterInput{Name: "", Kind: CharacterKindPC}); err == nil {
		t.Fatalf("expected validation error for empty name")
	}
	if _, err := svc.createCharacter(contextWithResolvedUserID("user-1"), "c1", CreateCharacterInput{Name: "Hero", Kind: CharacterKindUnspecified}); err == nil {
		t.Fatalf("expected validation error for unspecified kind")
	}
}

func TestUpdateParticipantDelegatesToGateway(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignParticipant: CampaignParticipant{
			ID:             "p-1",
			Name:           "Player One",
			Role:           "player",
			CampaignAccess: "member",
			Pronouns:       "they/them",
		},
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
	}
	svc := newService(gateway)

	err := svc.updateParticipant(contextWithResolvedUserID("user-1"), "c1", UpdateParticipantInput{
		ParticipantID:  "p-1",
		Name:           "Player Prime",
		Role:           "gm",
		Pronouns:       "she/her",
		CampaignAccess: "manager",
	})
	if err != nil {
		t.Fatalf("updateParticipant() error = %v", err)
	}
	if len(gateway.calls) != 1 || gateway.calls[0] != "update-participant" {
		t.Fatalf("mutation gateway calls = %v, want [update-participant]", gateway.calls)
	}
	if gateway.lastUpdateParticipantInput.Role != "gm" {
		t.Fatalf("updated role = %q, want %q", gateway.lastUpdateParticipantInput.Role, "gm")
	}
	if gateway.lastUpdateParticipantInput.CampaignAccess != "manager" {
		t.Fatalf("updated access = %q, want %q", gateway.lastUpdateParticipantInput.CampaignAccess, "manager")
	}
}

func TestUpdateParticipantAllowsSelfOwnedProfileChanges(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignParticipant: CampaignParticipant{
			ID:             "p-1",
			UserID:         "user-1",
			Name:           "Player One",
			Role:           "player",
			CampaignAccess: "member",
			Pronouns:       "she/her",
		},
		campaignWorkspace: CampaignWorkspace{ID: "c1", Name: "Campaign", GMMode: "human"},
		authorizationDecision: AuthorizationDecision{
			Evaluated:  true,
			Allowed:    false,
			ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED",
		},
	}
	svc := newService(gateway)

	err := svc.updateParticipant(contextWithResolvedUserID("user-1"), "c1", UpdateParticipantInput{
		ParticipantID:  "p-1",
		Name:           "Player Prime",
		Role:           "player",
		Pronouns:       "they/them",
		CampaignAccess: "member",
	})
	if err != nil {
		t.Fatalf("updateParticipant() error = %v", err)
	}
	if gateway.lastUpdateParticipantInput.Name != "Player Prime" {
		t.Fatalf("updated name = %q, want %q", gateway.lastUpdateParticipantInput.Name, "Player Prime")
	}
	if gateway.lastUpdateParticipantInput.Pronouns != "they/them" {
		t.Fatalf("updated pronouns = %q, want %q", gateway.lastUpdateParticipantInput.Pronouns, "they/them")
	}
}

func TestUpdateParticipantValidatesRoleAndAccess(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignParticipant: CampaignParticipant{
			ID:             "p-1",
			Name:           "Player One",
			Role:           "player",
			CampaignAccess: "member",
		},
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true},
	}
	svc := newService(gateway)

	if err := svc.updateParticipant(contextWithResolvedUserID("user-1"), "c1", UpdateParticipantInput{ParticipantID: "p-1", Name: "Player One", Role: "invalid"}); err == nil {
		t.Fatalf("expected role validation error")
	}
	if err := svc.updateParticipant(contextWithResolvedUserID("user-1"), "c1", UpdateParticipantInput{ParticipantID: "p-1", Name: "Player One", Role: "player", CampaignAccess: "invalid"}); err == nil {
		t.Fatalf("expected access validation error")
	}
}

func TestUpdateParticipantRejectsAIInvariantTampering(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignParticipant: CampaignParticipant{
			ID:             "p-ai",
			Name:           "Caretaker",
			Role:           "gm",
			CampaignAccess: "member",
			Controller:     "ai",
			Pronouns:       "it/its",
		},
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true},
	}
	svc := newService(gateway)

	err := svc.updateParticipant(contextWithResolvedUserID("user-1"), "c1", UpdateParticipantInput{
		ParticipantID:  "p-ai",
		Name:           "Caretaker",
		Role:           "player",
		Pronouns:       "it/its",
		CampaignAccess: "member",
	})
	if err == nil {
		t.Fatalf("expected conflict error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusConflict {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusConflict)
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.participant_ai_role_and_access_are_fixed" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.participant_ai_role_and_access_are_fixed")
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
	}
}

func TestUpdateCampaignAIBindingRequiresOwnerAccess(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{
			Evaluated:           true,
			Allowed:             true,
			ActorCampaignAccess: "manager",
		},
	}
	svc := newService(gateway)

	err := svc.updateCampaignAIBinding(contextWithResolvedUserID("user-1"), "c1", UpdateCampaignAIBindingInput{
		AIAgentID: "agent-1",
	})
	if err == nil {
		t.Fatalf("expected forbidden error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusForbidden {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusForbidden)
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
	}
}

func TestUpdateCampaignAIBindingDelegatesForOwner(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{
			Evaluated:           true,
			Allowed:             true,
			ActorCampaignAccess: "owner",
		},
	}
	svc := newService(gateway)

	input := UpdateCampaignAIBindingInput{AIAgentID: "agent-1"}
	if err := svc.updateCampaignAIBinding(contextWithResolvedUserID("user-1"), "c1", input); err != nil {
		t.Fatalf("updateCampaignAIBinding() error = %v", err)
	}
	if len(gateway.calls) != 1 || gateway.calls[0] != "update-campaign-ai-binding" {
		t.Fatalf("mutation gateway calls = %v, want [update-campaign-ai-binding]", gateway.calls)
	}
	if gateway.lastUpdateCampaignAIBindingInput != input {
		t.Fatalf("lastUpdateCampaignAIBindingInput = %#v, want %#v", gateway.lastUpdateCampaignAIBindingInput, input)
	}
}

func TestUpdateCampaignAIBindingFallsBackToParticipantAccessWhenAuthzOmitsActorAccess(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignParticipants: []CampaignParticipant{{
			ID:             "p-owner",
			UserID:         "user-1",
			CampaignAccess: "owner",
		}},
		authorizationDecision: AuthorizationDecision{
			Evaluated: true,
			Allowed:   true,
		},
	}
	svc := newService(gateway)

	if err := svc.updateCampaignAIBinding(contextWithResolvedUserID("user-1"), "c1", UpdateCampaignAIBindingInput{
		AIAgentID: "agent-1",
	}); err != nil {
		t.Fatalf("updateCampaignAIBinding() error = %v", err)
	}
	if len(gateway.calls) != 1 || gateway.calls[0] != "update-campaign-ai-binding" {
		t.Fatalf("mutation gateway calls = %v, want [update-campaign-ai-binding]", gateway.calls)
	}
}

func TestUpdateParticipantRequiresMeaningfulChange(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignParticipant: CampaignParticipant{
			ID:             "p-1",
			Name:           "Player One",
			Role:           "player",
			CampaignAccess: "member",
			Pronouns:       "they/them",
		},
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true},
	}
	svc := newService(gateway)

	err := svc.updateParticipant(contextWithResolvedUserID("user-1"), "c1", UpdateParticipantInput{
		ParticipantID:  "p-1",
		Name:           "Player One",
		Role:           "player",
		Pronouns:       "they/them",
		CampaignAccess: "member",
	})
	if err == nil {
		t.Fatalf("expected at least-one-field error")
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.at_least_one_participant_field_is_required" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.at_least_one_participant_field_is_required")
	}
}

func TestUpdateCampaignDelegatesToGateway(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignWorkspace: CampaignWorkspace{
			ID:     "c1",
			Name:   "Campaign One",
			Theme:  "Old Theme",
			Locale: "en_us",
		},
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true},
	}
	svc := newService(gateway)

	name := "Campaign Prime"
	theme := "New Theme"
	locale := "pt"
	err := svc.updateCampaign(contextWithResolvedUserID("user-1"), "c1", UpdateCampaignInput{
		Name:        &name,
		ThemePrompt: &theme,
		Locale:      &locale,
	})
	if err != nil {
		t.Fatalf("updateCampaign() error = %v", err)
	}
	if len(gateway.calls) != 1 || gateway.calls[0] != "update-campaign" {
		t.Fatalf("mutation gateway calls = %v, want [update-campaign]", gateway.calls)
	}
	if gateway.lastUpdateCampaignInput.Locale == nil || *gateway.lastUpdateCampaignInput.Locale != "pt-BR" {
		t.Fatalf("updated locale = %#v, want %q", gateway.lastUpdateCampaignInput.Locale, "pt-BR")
	}
	if gateway.lastUpdateCampaignInput.Name == nil || *gateway.lastUpdateCampaignInput.Name != "Campaign Prime" {
		t.Fatalf("updated name = %#v, want %q", gateway.lastUpdateCampaignInput.Name, "Campaign Prime")
	}
	if gateway.lastUpdateCampaignInput.ThemePrompt == nil || *gateway.lastUpdateCampaignInput.ThemePrompt != "New Theme" {
		t.Fatalf("updated theme = %#v, want %q", gateway.lastUpdateCampaignInput.ThemePrompt, "New Theme")
	}
}

func TestUpdateCampaignNoOpReturnsNil(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignWorkspace: CampaignWorkspace{
			ID:     "c1",
			Name:   "Campaign One",
			Theme:  "Old Theme",
			Locale: "en_us",
		},
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true},
	}
	svc := newService(gateway)

	name := "Campaign One"
	theme := "Old Theme"
	locale := "en-US"
	err := svc.updateCampaign(contextWithResolvedUserID("user-1"), "c1", UpdateCampaignInput{
		Name:        &name,
		ThemePrompt: &theme,
		Locale:      &locale,
	})
	if err != nil {
		t.Fatalf("updateCampaign() error = %v", err)
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
	}
}

func TestUpdateCampaignValidatesLocale(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignWorkspace:     CampaignWorkspace{ID: "c1", Name: "Campaign One", Locale: "en_us"},
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true},
	}
	svc := newService(gateway)

	name := "Campaign Prime"
	theme := "New Theme"
	locale := "es-ES"
	err := svc.updateCampaign(contextWithResolvedUserID("user-1"), "c1", UpdateCampaignInput{
		Name:        &name,
		ThemePrompt: &theme,
		Locale:      &locale,
	})
	if err == nil {
		t.Fatalf("expected locale validation error")
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.campaign_locale_value_is_invalid" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.campaign_locale_value_is_invalid")
	}
}

func TestEndSessionValidatesSessionID(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true},
	})
	err := svc.endSession(contextWithResolvedUserID("user-1"), "c1", EndSessionInput{SessionID: "   "})
	if err == nil {
		t.Fatalf("expected validation error for empty session id")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.session_id_is_required" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.session_id_is_required")
	}
}

func TestCreateInviteValidatesParticipantID(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true},
	})
	err := svc.createInvite(contextWithResolvedUserID("user-1"), "c1", CreateInviteInput{
		ParticipantID:     "   ",
		RecipientUsername: "alice",
	})
	if err == nil {
		t.Fatalf("expected validation error for empty participant id")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.participant_id_is_required" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.participant_id_is_required")
	}
}

func TestRevokeInviteValidatesInviteID(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true},
	})
	err := svc.revokeInvite(contextWithResolvedUserID("user-1"), "c1", RevokeInviteInput{InviteID: "   "})
	if err == nil {
		t.Fatalf("expected validation error for empty invite id")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.invite_id_is_required" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.invite_id_is_required")
	}
}

func TestServiceCreateCharacterRejectsEmptyCreatedCharacterID(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision:    AuthorizationDecision{Evaluated: true, Allowed: true},
		createCharacterResult:    CreateCharacterResult{CharacterID: "   "},
		createCharacterResultSet: true,
	}
	svc := newService(gateway)

	_, err := svc.createCharacter(contextWithResolvedUserID("user-1"), "c1", CreateCharacterInput{Name: "Hero", Kind: CharacterKindPC})
	if err == nil {
		t.Fatalf("expected empty character id error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestMutationMethodsDenyWhenAuthorizationNotEvaluated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(testServiceBundle) error
	}{
		{
			name: "start session",
			run: func(s testServiceBundle) error {
				return s.startSession(contextWithResolvedUserID("user-1"), "c1", StartSessionInput{Name: "Session One"})
			},
		},
		{
			name: "end session",
			run: func(s testServiceBundle) error {
				return s.endSession(contextWithResolvedUserID("user-1"), "c1", EndSessionInput{SessionID: "sess-1"})
			},
		},
		{
			name: "create invite",
			run: func(s testServiceBundle) error {
				return s.createInvite(contextWithResolvedUserID("user-1"), "c1", CreateInviteInput{
					ParticipantID:     "p-1",
					RecipientUsername: "alice",
				})
			},
		},
		{
			name: "revoke invite",
			run: func(s testServiceBundle) error {
				return s.revokeInvite(contextWithResolvedUserID("user-1"), "c1", RevokeInviteInput{InviteID: "inv-1"})
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gateway := &campaignGatewayStub{}
			svc := newService(gateway)
			err := tc.run(svc)
			if err == nil {
				t.Fatalf("expected forbidden error when authorization was not evaluated")
			}
			if got := apperrors.HTTPStatus(err); got != http.StatusForbidden {
				t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusForbidden)
			}
			if len(gateway.calls) != 0 {
				t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
			}
		})
	}
}

func TestMutationMethodsDenyWhenAuthorizationGatewayErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(testServiceBundle) error
	}{
		{
			name: "start session",
			run: func(s testServiceBundle) error {
				return s.startSession(contextWithResolvedUserID("user-1"), "c1", StartSessionInput{Name: "Session One"})
			},
		},
		{
			name: "end session",
			run: func(s testServiceBundle) error {
				return s.endSession(contextWithResolvedUserID("user-1"), "c1", EndSessionInput{SessionID: "sess-1"})
			},
		},
		{
			name: "create invite",
			run: func(s testServiceBundle) error {
				return s.createInvite(contextWithResolvedUserID("user-1"), "c1", CreateInviteInput{
					ParticipantID:     "p-1",
					RecipientUsername: "alice",
				})
			},
		},
		{
			name: "revoke invite",
			run: func(s testServiceBundle) error {
				return s.revokeInvite(contextWithResolvedUserID("user-1"), "c1", RevokeInviteInput{InviteID: "inv-1"})
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gateway := &campaignGatewayStub{authorizationErr: errors.New("authz unavailable")}
			svc := newService(gateway)
			err := tc.run(svc)
			if err == nil {
				t.Fatalf("expected forbidden error when authz check fails")
			}
			if got := apperrors.HTTPStatus(err); got != http.StatusForbidden {
				t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusForbidden)
			}
			if len(gateway.calls) != 0 {
				t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
			}
		})
	}
}
func TestCreateCampaignForwardsInputToGateway(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{}
	svc := newService(gateway)

	input := CreateCampaignInput{
		Name:        "The Guild",
		System:      GameSystemDaggerheart,
		GMMode:      GmModeAI,
		ThemePrompt: "Storm coast",
	}

	if _, err := svc.createCampaign(context.Background(), input); err != nil {
		t.Fatalf("createCampaign() error = %v", err)
	}

	if gateway.lastCreateInput.Name != input.Name {
		t.Fatalf("Name = %q, want %q", gateway.lastCreateInput.Name, input.Name)
	}
	if gateway.lastCreateInput.System != input.System {
		t.Fatalf("System = %v, want %v", gateway.lastCreateInput.System, input.System)
	}
	if gateway.lastCreateInput.GMMode != input.GMMode {
		t.Fatalf("GMMode = %v, want %v", gateway.lastCreateInput.GMMode, input.GMMode)
	}
	if gateway.lastCreateInput.ThemePrompt != input.ThemePrompt {
		t.Fatalf("ThemePrompt = %q, want %q", gateway.lastCreateInput.ThemePrompt, input.ThemePrompt)
	}
}
