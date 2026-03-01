package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
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
		ID:               "c-1",
		Name:             "The Guild",
		Theme:            "Storm coast",
		System:           "Daggerheart",
		GMMode:           "AI",
		Status:           "Active",
		Locale:           "English (US)",
		Intent:           "Standard",
		AccessPolicy:     "Public",
		ParticipantCount: "4",
		CharacterCount:   "1",
		CoverImageURL:    "https://cdn.example.com/covers/the-guild.png",
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
	if workspace.Status != "Active" {
		t.Fatalf("workspace.Status = %q, want %q", workspace.Status, "Active")
	}
	if workspace.Locale != "English (US)" {
		t.Fatalf("workspace.Locale = %q, want %q", workspace.Locale, "English (US)")
	}
	if workspace.Intent != "Standard" {
		t.Fatalf("workspace.Intent = %q, want %q", workspace.Intent, "Standard")
	}
	if workspace.AccessPolicy != "Public" {
		t.Fatalf("workspace.AccessPolicy = %q, want %q", workspace.AccessPolicy, "Public")
	}
	if workspace.ParticipantCount != "4" {
		t.Fatalf("workspace.ParticipantCount = %q, want %q", workspace.ParticipantCount, "4")
	}
	if workspace.CharacterCount != "1" {
		t.Fatalf("workspace.CharacterCount = %q, want %q", workspace.CharacterCount, "1")
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
		if req.Action != campaignAuthzActionMutate {
			t.Fatalf("batch authorization action = %v, want %v", req.Action, campaignAuthzActionMutate)
		}
		if req.Resource != campaignAuthzResourceCharacter {
			t.Fatalf("batch authorization resource = %v, want %v", req.Resource, campaignAuthzResourceCharacter)
		}
		if req.Target == nil || strings.TrimSpace(req.Target.ResourceID) == "" {
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
