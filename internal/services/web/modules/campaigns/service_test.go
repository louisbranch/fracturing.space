package campaigns

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc/metadata"
)

func TestListCampaignsSortsNewestFirst(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{items: []CampaignSummary{
		{
			ID:                "camp-old",
			Name:              "Older Campaign",
			CreatedAtUnixNano: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC).UnixNano(),
		},
		{
			ID:                "camp-new",
			Name:              "Newer Campaign",
			CreatedAtUnixNano: time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC).UnixNano(),
		},
	}})

	items, err := svc.listCampaigns(context.Background())
	if err != nil {
		t.Fatalf("listCampaigns() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != "camp-new" || items[1].ID != "camp-old" {
		t.Fatalf("campaign order = [%s, %s], want [camp-new, camp-old]", items[0].ID, items[1].ID)
	}
}

func TestNewServiceFailsClosedWhenGatewayMissing(t *testing.T) {
	t.Parallel()

	svc := newService(nil)
	_, err := svc.listCampaigns(context.Background())
	if err == nil {
		t.Fatalf("expected unavailable error for listCampaigns")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}

	_, err = svc.createCampaign(context.Background(), CreateCampaignInput{Name: "Starter"})
	if err == nil {
		t.Fatalf("expected unavailable error for createCampaign")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestMissingGatewayMutationMethodsFailClosed(t *testing.T) {
	t.Parallel()

	svc := newService(nil)
	ctx := contextWithResolvedUserID("user-1")
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "start session", run: func() error { return svc.startSession(ctx, "c1") }},
		{name: "end session", run: func() error { return svc.endSession(ctx, "c1") }},
		{name: "update participants", run: func() error { return svc.updateParticipants(ctx, "c1") }},
		{name: "create character", run: func() error { return svc.createCharacter(ctx, "c1") }},
		{name: "update character", run: func() error { return svc.updateCharacter(ctx, "c1") }},
		{name: "control character", run: func() error { return svc.controlCharacter(ctx, "c1") }},
		{name: "create invite", run: func() error { return svc.createInvite(ctx, "c1") }},
		{name: "revoke invite", run: func() error { return svc.revokeInvite(ctx, "c1") }},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			if err == nil {
				t.Fatalf("expected forbidden error")
			}
			if got := apperrors.HTTPStatus(err); got != http.StatusForbidden {
				t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusForbidden)
			}
		})
	}
}

func TestListCampaignsReturnsEmptySliceWhenGatewayReturnsNil(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{})
	items, err := svc.listCampaigns(context.Background())
	if err != nil {
		t.Fatalf("listCampaigns() error = %v", err)
	}
	if items == nil {
		t.Fatalf("listCampaigns() returned nil slice")
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}

func TestCreateCampaignValidatesName(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{})
	_, err := svc.createCampaign(context.Background(), CreateCampaignInput{Name: "   "})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestCreateCampaignRejectsEmptyGatewayResultID(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{createCampaignResult: CreateCampaignResult{CampaignID: "   "}})
	_, err := svc.createCampaign(context.Background(), CreateCampaignInput{Name: "Campaign"})
	if err == nil {
		t.Fatalf("expected unknown error for empty campaign id")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestCampaignNameFallsBackToCampaignIDOnGatewayErrors(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{campaignNameErr: errors.New("boom")})
	if got := svc.campaignName(context.Background(), "c-1"); got != "c-1" {
		t.Fatalf("campaignName() = %q, want %q", got, "c-1")
	}
}

func TestCampaignNameReturnsTrimmedGatewayName(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{campaignName: "  The Guild  "})
	if got := svc.campaignName(context.Background(), "c-1"); got != "The Guild" {
		t.Fatalf("campaignName() = %q, want %q", got, "The Guild")
	}
}

func TestCampaignWorkspaceReturnsGatewayValues(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{campaignWorkspace: CampaignWorkspace{
		ID:            "c-1",
		Name:          "The Guild",
		Theme:         "Storm coast",
		System:        "Daggerheart",
		GMMode:        "AI",
		CoverImageURL: "https://cdn.example.com/covers/the-guild.png",
	}})

	workspace, err := svc.campaignWorkspace(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("campaignWorkspace() error = %v", err)
	}
	if workspace.Name != "The Guild" {
		t.Fatalf("workspace.Name = %q, want %q", workspace.Name, "The Guild")
	}
	if workspace.Theme != "Storm coast" {
		t.Fatalf("workspace.Theme = %q, want %q", workspace.Theme, "Storm coast")
	}
	if workspace.System != "Daggerheart" {
		t.Fatalf("workspace.System = %q, want %q", workspace.System, "Daggerheart")
	}
	if workspace.GMMode != "AI" {
		t.Fatalf("workspace.GMMode = %q, want %q", workspace.GMMode, "AI")
	}
	if workspace.CoverImageURL != "https://cdn.example.com/covers/the-guild.png" {
		t.Fatalf("workspace.CoverImageURL = %q, want %q", workspace.CoverImageURL, "https://cdn.example.com/covers/the-guild.png")
	}
}

func TestCampaignWorkspaceReturnsGatewayError(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{campaignWorkspaceErr: errors.New("boom")})
	_, err := svc.campaignWorkspace(context.Background(), "c-1")
	if err == nil {
		t.Fatalf("expected campaignWorkspace() error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestCampaignParticipantsSortByName(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{campaignParticipants: []CampaignParticipant{
		{
			ID:             "p-z",
			Name:           "  Zara  ",
			Role:           "Player",
			CampaignAccess: "Member",
			Controller:     "Human",
			AvatarURL:      "/static/avatars/zara.png",
		},
		{
			ID:             "p-a",
			Name:           "Aria",
			Role:           "GM",
			CampaignAccess: "Owner",
			Controller:     "AI",
			AvatarURL:      "/static/avatars/aria.png",
		},
	}})

	participants, err := svc.campaignParticipants(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("campaignParticipants() error = %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("len(participants) = %d, want 2", len(participants))
	}
	if participants[0].Name != "Aria" || participants[1].Name != "Zara" {
		t.Fatalf("participant order = [%s, %s], want [Aria, Zara]", participants[0].Name, participants[1].Name)
	}
	if participants[0].Role != "GM" || participants[0].CampaignAccess != "Owner" || participants[0].Controller != "AI" {
		t.Fatalf("participant metadata = %#v, want role/access/controller labels", participants[0])
	}
}

func TestCampaignCharactersSortByName(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{campaignCharacters: []CampaignCharacter{
		{
			ID:         "ch-z",
			Name:       "  Zara  ",
			Kind:       "NPC",
			Controller: "Moss",
			AvatarURL:  "/static/avatars/zara.png",
		},
		{
			ID:         "ch-a",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
			AvatarURL:  "/static/avatars/aria.png",
		},
	}})

	characters, err := svc.campaignCharacters(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("campaignCharacters() error = %v", err)
	}
	if len(characters) != 2 {
		t.Fatalf("len(characters) = %d, want 2", len(characters))
	}
	if characters[0].Name != "Aria" || characters[1].Name != "Zara" {
		t.Fatalf("character order = [%s, %s], want [Aria, Zara]", characters[0].Name, characters[1].Name)
	}
	if characters[0].Kind != "PC" || characters[0].Controller != "Ariadne" {
		t.Fatalf("character metadata = %#v, want kind/controller labels", characters[0])
	}
}

func TestCampaignCharactersHydratesEditabilityFromBatchAuthorization(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignCharacters: []CampaignCharacter{
			{ID: "ch-z", Name: "Zara", Kind: "NPC", Controller: "Moss"},
			{ID: "ch-a", Name: "Aria", Kind: "PC", Controller: "Ariadne"},
		},
		batchAuthorizationDecisions: []campaignAuthorizationDecision{
			{CheckID: "ch-a", Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_RESOURCE_OWNER"},
			{CheckID: "ch-z", Evaluated: true, Allowed: false, ReasonCode: "AUTHZ_DENY_NOT_RESOURCE_OWNER"},
		},
	}
	svc := newService(gateway)

	characters, err := svc.campaignCharacters(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("campaignCharacters() error = %v", err)
	}
	if len(characters) != 2 {
		t.Fatalf("len(characters) = %d, want 2", len(characters))
	}
	if !characters[0].CanEdit {
		t.Fatalf("characters[0].CanEdit = %v, want true", characters[0].CanEdit)
	}
	if got := characters[0].EditReasonCode; got != "AUTHZ_ALLOW_RESOURCE_OWNER" {
		t.Fatalf("characters[0].EditReasonCode = %q, want %q", got, "AUTHZ_ALLOW_RESOURCE_OWNER")
	}
	if characters[1].CanEdit {
		t.Fatalf("characters[1].CanEdit = %v, want false", characters[1].CanEdit)
	}
	if got := characters[1].EditReasonCode; got != "AUTHZ_DENY_NOT_RESOURCE_OWNER" {
		t.Fatalf("characters[1].EditReasonCode = %q, want %q", got, "AUTHZ_DENY_NOT_RESOURCE_OWNER")
	}
	if gateway.batchAuthorizationCalls != 1 {
		t.Fatalf("batch authorization calls = %d, want 1", gateway.batchAuthorizationCalls)
	}
	if len(gateway.batchAuthorizationRequests) != 2 {
		t.Fatalf("batch authorization requests = %d, want 2", len(gateway.batchAuthorizationRequests))
	}
	for _, req := range gateway.batchAuthorizationRequests {
		if req.Action != statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE {
			t.Fatalf("batch authorization action = %v, want %v", req.Action, statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE)
		}
		if req.Resource != statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER {
			t.Fatalf("batch authorization resource = %v, want %v", req.Resource, statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER)
		}
		if req.Target == nil || strings.TrimSpace(req.Target.GetResourceId()) == "" {
			t.Fatalf("batch authorization target resource id was empty")
		}
	}
}

func TestCampaignCharactersHydratesEditabilityForDuplicateCharacterIDs(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		campaignCharacters: []CampaignCharacter{
			{ID: "ch-a", Name: "Aria", Kind: "PC", Controller: "Ariadne"},
			{ID: "ch-a", Name: "Aria Clone", Kind: "PC", Controller: "Ariadne"},
		},
		batchAuthorizationDecisions: []campaignAuthorizationDecision{
			{CheckID: "ch-a", Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_RESOURCE_OWNER"},
		},
	}
	svc := newService(gateway)

	characters, err := svc.campaignCharacters(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("campaignCharacters() error = %v", err)
	}
	if len(characters) != 2 {
		t.Fatalf("len(characters) = %d, want 2", len(characters))
	}
	for idx := range characters {
		if !characters[idx].CanEdit {
			t.Fatalf("characters[%d].CanEdit = %v, want true", idx, characters[idx].CanEdit)
		}
		if got := characters[idx].EditReasonCode; got != "AUTHZ_ALLOW_RESOURCE_OWNER" {
			t.Fatalf("characters[%d].EditReasonCode = %q, want %q", idx, got, "AUTHZ_ALLOW_RESOURCE_OWNER")
		}
	}
	if gateway.batchAuthorizationCalls != 1 {
		t.Fatalf("batch authorization calls = %d, want 1", gateway.batchAuthorizationCalls)
	}
	if len(gateway.batchAuthorizationRequests) != 1 {
		t.Fatalf("batch authorization requests = %d, want 1", len(gateway.batchAuthorizationRequests))
	}
}

func TestCampaignCharactersFailClosedWhenBatchAuthorizationErrors(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{
		campaignCharacters: []CampaignCharacter{{ID: "ch-a", Name: "Aria", Kind: "PC", Controller: "Ariadne"}},
		batchAuthorizationErr: apperrors.E(
			apperrors.KindUnavailable,
			"authorization unavailable",
		),
	})

	characters, err := svc.campaignCharacters(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("campaignCharacters() error = %v", err)
	}
	if len(characters) != 1 {
		t.Fatalf("len(characters) = %d, want 1", len(characters))
	}
	if characters[0].CanEdit {
		t.Fatalf("characters[0].CanEdit = %v, want false", characters[0].CanEdit)
	}
	if got := strings.TrimSpace(characters[0].EditReasonCode); got != "" {
		t.Fatalf("characters[0].EditReasonCode = %q, want empty", got)
	}
}

func TestCampaignParticipantsReturnsGatewayError(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{campaignParticipantsErr: apperrors.E(apperrors.KindUnavailable, "participants unavailable")})
	_, err := svc.campaignParticipants(context.Background(), "c-1")
	if err == nil {
		t.Fatalf("expected campaignParticipants() error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestCampaignCharactersReturnsGatewayError(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{campaignCharactersErr: apperrors.E(apperrors.KindUnavailable, "characters unavailable")})
	_, err := svc.campaignCharacters(context.Background(), "c-1")
	if err == nil {
		t.Fatalf("expected campaignCharacters() error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestMutationMethodsDelegateToGateway(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: campaignAuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
	}
	svc := newService(gateway)
	ctx := contextWithResolvedUserID("user-1")

	if err := svc.startSession(ctx, "c1"); err != nil {
		t.Fatalf("startSession() error = %v", err)
	}
	if err := svc.endSession(ctx, "c1"); err != nil {
		t.Fatalf("endSession() error = %v", err)
	}
	if err := svc.updateParticipants(ctx, "c1"); err != nil {
		t.Fatalf("updateParticipants() error = %v", err)
	}
	if err := svc.createCharacter(ctx, "c1"); err != nil {
		t.Fatalf("createCharacter() error = %v", err)
	}
	if err := svc.updateCharacter(ctx, "c1"); err != nil {
		t.Fatalf("updateCharacter() error = %v", err)
	}
	if err := svc.controlCharacter(ctx, "c1"); err != nil {
		t.Fatalf("controlCharacter() error = %v", err)
	}
	if err := svc.createInvite(ctx, "c1"); err != nil {
		t.Fatalf("createInvite() error = %v", err)
	}
	if err := svc.revokeInvite(ctx, "c1"); err != nil {
		t.Fatalf("revokeInvite() error = %v", err)
	}

	want := []string{"start", "end", "participants", "create-character", "update-character", "control-character", "create-invite", "revoke-invite"}
	if len(gateway.calls) != len(want) {
		t.Fatalf("len(calls) = %d, want %d (%v)", len(gateway.calls), len(want), gateway.calls)
	}
	for i := range want {
		if gateway.calls[i] != want[i] {
			t.Fatalf("calls[%d] = %q, want %q", i, gateway.calls[i], want[i])
		}
	}
}

func TestMutationMethodsDenyMemberCampaignAccess(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: campaignAuthorizationDecision{Evaluated: true, Allowed: false, ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED"},
	}
	svc := newService(gateway)
	err := svc.startSession(contextWithResolvedUserID("user-1"), "c1")
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
				authorizationDecision: campaignAuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
				campaignParticipants:  []CampaignParticipant{{ID: "p-1", UserID: "user-1", CampaignAccess: tc.access}},
			}
			svc := newService(gateway)
			if err := svc.startSession(contextWithResolvedUserID("user-1"), "c1"); err != nil {
				t.Fatalf("startSession() error = %v", err)
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
			authorizationDecision: campaignAuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
		}
		svc := newService(gateway)
		if err := svc.startSession(contextWithResolvedUserID("user-1"), "c1"); err != nil {
			t.Fatalf("startSession() error = %v", err)
		}
		if gateway.authorizationCalls != 1 {
			t.Fatalf("authorization calls = %d, want 1", gateway.authorizationCalls)
		}
		if len(gateway.authorizationRequests) != 1 {
			t.Fatalf("authorization requests = %d, want 1", len(gateway.authorizationRequests))
		}
		if got := gateway.authorizationRequests[0].Action; got != statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE {
			t.Fatalf("authorization action = %v, want %v", got, statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE)
		}
		if got := gateway.authorizationRequests[0].Resource; got != statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION {
			t.Fatalf("authorization resource = %v, want %v", got, statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION)
		}
		if len(gateway.calls) != 1 || gateway.calls[0] != "start" {
			t.Fatalf("mutation gateway calls = %v, want [start]", gateway.calls)
		}
	})

	t.Run("deny", func(t *testing.T) {
		t.Parallel()
		gateway := &campaignGatewayStub{
			authorizationDecision: campaignAuthorizationDecision{Evaluated: true, Allowed: false, ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED"},
		}
		svc := newService(gateway)
		err := svc.startSession(contextWithResolvedUserID("user-1"), "c1")
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
		if got := gateway.authorizationRequests[0].Action; got != statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE {
			t.Fatalf("authorization action = %v, want %v", got, statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE)
		}
		if got := gateway.authorizationRequests[0].Resource; got != statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION {
			t.Fatalf("authorization resource = %v, want %v", got, statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION)
		}
		if len(gateway.calls) != 0 {
			t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
		}
	})
}

func TestMutationMethodsRequestExpectedCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		run          func(service) error
		wantAction   statev1.AuthorizationAction
		wantResource statev1.AuthorizationResource
	}{
		{
			name: "start session",
			run: func(s service) error {
				return s.startSession(contextWithResolvedUserID("user-1"), "c1")
			},
			wantAction:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
			wantResource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION,
		},
		{
			name: "end session",
			run: func(s service) error {
				return s.endSession(contextWithResolvedUserID("user-1"), "c1")
			},
			wantAction:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
			wantResource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION,
		},
		{
			name: "update participants",
			run: func(s service) error {
				return s.updateParticipants(contextWithResolvedUserID("user-1"), "c1")
			},
			wantAction:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
			wantResource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT,
		},
		{
			name: "create character",
			run: func(s service) error {
				return s.createCharacter(contextWithResolvedUserID("user-1"), "c1")
			},
			wantAction:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
			wantResource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
		},
		{
			name: "update character",
			run: func(s service) error {
				return s.updateCharacter(contextWithResolvedUserID("user-1"), "c1")
			},
			wantAction:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
			wantResource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
		},
		{
			name: "control character",
			run: func(s service) error {
				return s.controlCharacter(contextWithResolvedUserID("user-1"), "c1")
			},
			wantAction:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
			wantResource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
		},
		{
			name: "create invite",
			run: func(s service) error {
				return s.createInvite(contextWithResolvedUserID("user-1"), "c1")
			},
			wantAction:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
			wantResource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_INVITE,
		},
		{
			name: "revoke invite",
			run: func(s service) error {
				return s.revokeInvite(contextWithResolvedUserID("user-1"), "c1")
			},
			wantAction:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
			wantResource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_INVITE,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gateway := &campaignGatewayStub{
				authorizationDecision: campaignAuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
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

func TestCharacterMutationMethodsAllowMemberCampaignAccess(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: campaignAuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
	}
	svc := newService(gateway)
	if err := svc.createCharacter(contextWithResolvedUserID("user-1"), "c1"); err != nil {
		t.Fatalf("createCharacter() error = %v", err)
	}
	if err := svc.updateCharacter(contextWithResolvedUserID("user-1"), "c1"); err != nil {
		t.Fatalf("updateCharacter() error = %v", err)
	}
	if len(gateway.calls) != 2 || gateway.calls[0] != "create-character" || gateway.calls[1] != "update-character" {
		t.Fatalf("mutation gateway calls = %v, want [create-character update-character]", gateway.calls)
	}
}

func TestMutationMethodsDenyWhenAuthorizationNotEvaluated(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{}
	svc := newService(gateway)
	err := svc.startSession(contextWithResolvedUserID("user-1"), "c1")
	if err == nil {
		t.Fatalf("expected forbidden error when authorization was not evaluated")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusForbidden {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusForbidden)
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
	}
}

func TestMutationMethodsDenyWhenAuthorizationGatewayErrors(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{authorizationErr: errors.New("authz unavailable")}
	svc := newService(gateway)
	err := svc.startSession(contextWithResolvedUserID("user-1"), "c1")
	if err == nil {
		t.Fatalf("expected forbidden error when authz check fails")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusForbidden {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusForbidden)
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("mutation gateway calls = %v, want none", gateway.calls)
	}
}

func contextWithResolvedUserID(userID string) context.Context {
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, userID))
}

type campaignGatewayStub struct {
	items                       []CampaignSummary
	listErr                     error
	campaignName                string
	campaignNameErr             error
	campaignWorkspace           CampaignWorkspace
	campaignWorkspaceErr        error
	campaignParticipants        []CampaignParticipant
	campaignParticipantsErr     error
	campaignCharacters          []CampaignCharacter
	campaignCharactersErr       error
	campaignSessions            []CampaignSession
	campaignSessionsErr         error
	campaignInvites             []CampaignInvite
	campaignInvitesErr          error
	createCampaignResult        CreateCampaignResult
	createCampaignErr           error
	lastCreateInput             CreateCampaignInput
	authorizationDecision       campaignAuthorizationDecision
	authorizationErr            error
	authorizationCalls          int
	authorizationRequests       []campaignAuthorizationRequest
	batchAuthorizationDecisions []campaignAuthorizationDecision
	batchAuthorizationErr       error
	batchAuthorizationCalls     int
	batchAuthorizationRequests  []campaignAuthorizationCheck
	calls                       []string
}

type campaignAuthorizationRequest struct {
	Action   statev1.AuthorizationAction
	Resource statev1.AuthorizationResource
	Target   *statev1.AuthorizationTarget
}

func (f *campaignGatewayStub) ListCampaigns(context.Context) ([]CampaignSummary, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.items, nil
}

func (f *campaignGatewayStub) CampaignName(context.Context, string) (string, error) {
	if f.campaignNameErr != nil {
		return "", f.campaignNameErr
	}
	return f.campaignName, nil
}

func (f *campaignGatewayStub) CampaignWorkspace(_ context.Context, campaignID string) (CampaignWorkspace, error) {
	if f.campaignWorkspaceErr != nil {
		return CampaignWorkspace{}, f.campaignWorkspaceErr
	}
	workspace := f.campaignWorkspace
	if strings.TrimSpace(workspace.ID) == "" {
		workspace.ID = campaignID
	}
	return workspace, nil
}

func (f *campaignGatewayStub) CampaignParticipants(context.Context, string) ([]CampaignParticipant, error) {
	if f.campaignParticipantsErr != nil {
		return nil, f.campaignParticipantsErr
	}
	return f.campaignParticipants, nil
}

func (f *campaignGatewayStub) CampaignCharacters(context.Context, string) ([]CampaignCharacter, error) {
	if f.campaignCharactersErr != nil {
		return nil, f.campaignCharactersErr
	}
	return f.campaignCharacters, nil
}

func (f *campaignGatewayStub) CampaignSessions(context.Context, string) ([]CampaignSession, error) {
	if f.campaignSessionsErr != nil {
		return nil, f.campaignSessionsErr
	}
	return f.campaignSessions, nil
}

func (f *campaignGatewayStub) CampaignInvites(context.Context, string) ([]CampaignInvite, error) {
	if f.campaignInvitesErr != nil {
		return nil, f.campaignInvitesErr
	}
	return f.campaignInvites, nil
}

func (f *campaignGatewayStub) CreateCampaign(_ context.Context, input CreateCampaignInput) (CreateCampaignResult, error) {
	if f != nil {
		// capture input for behavior assertions
		f.lastCreateInput = input
	}
	if f.createCampaignErr != nil {
		return CreateCampaignResult{}, f.createCampaignErr
	}
	if f.createCampaignResult.CampaignID == "" {
		return CreateCampaignResult{CampaignID: "created"}, nil
	}
	return f.createCampaignResult, nil
}

func (f *campaignGatewayStub) StartSession(context.Context, string) error {
	f.calls = append(f.calls, "start")
	return nil
}

func (f *campaignGatewayStub) EndSession(context.Context, string) error {
	f.calls = append(f.calls, "end")
	return nil
}

func (f *campaignGatewayStub) UpdateParticipants(context.Context, string) error {
	f.calls = append(f.calls, "participants")
	return nil
}

func (f *campaignGatewayStub) CreateCharacter(context.Context, string) error {
	f.calls = append(f.calls, "create-character")
	return nil
}

func (f *campaignGatewayStub) UpdateCharacter(context.Context, string) error {
	f.calls = append(f.calls, "update-character")
	return nil
}

func (f *campaignGatewayStub) ControlCharacter(context.Context, string) error {
	f.calls = append(f.calls, "control-character")
	return nil
}

func (f *campaignGatewayStub) CreateInvite(context.Context, string) error {
	f.calls = append(f.calls, "create-invite")
	return nil
}

func (f *campaignGatewayStub) RevokeInvite(context.Context, string) error {
	f.calls = append(f.calls, "revoke-invite")
	return nil
}

func (f *campaignGatewayStub) CanCampaignAction(
	_ context.Context,
	_ string,
	action statev1.AuthorizationAction,
	resource statev1.AuthorizationResource,
	target *statev1.AuthorizationTarget,
) (campaignAuthorizationDecision, error) {
	f.authorizationCalls++
	f.authorizationRequests = append(f.authorizationRequests, campaignAuthorizationRequest{Action: action, Resource: resource, Target: target})
	if f.authorizationErr != nil {
		return campaignAuthorizationDecision{}, f.authorizationErr
	}
	return f.authorizationDecision, nil
}

func (f *campaignGatewayStub) BatchCanCampaignAction(
	_ context.Context,
	_ string,
	checks []campaignAuthorizationCheck,
) ([]campaignAuthorizationDecision, error) {
	f.batchAuthorizationCalls++
	f.batchAuthorizationRequests = append(f.batchAuthorizationRequests, checks...)
	if f.batchAuthorizationErr != nil {
		return nil, f.batchAuthorizationErr
	}
	return f.batchAuthorizationDecisions, nil
}

func TestCreateCampaignForwardsInputToGateway(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{}
	svc := newService(gateway)

	input := CreateCampaignInput{
		Name:               "The Guild",
		System:             commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GMMode:             statev1.GmMode_AI,
		ThemePrompt:        "Storm coast",
		CreatorDisplayName: "Rhea",
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
	if gateway.lastCreateInput.CreatorDisplayName != input.CreatorDisplayName {
		t.Fatalf("CreatorDisplayName = %q, want %q", gateway.lastCreateInput.CreatorDisplayName, input.CreatorDisplayName)
	}
}
