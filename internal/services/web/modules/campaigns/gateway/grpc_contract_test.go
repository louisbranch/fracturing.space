package gateway

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestNewGRPCGatewayRequiresCompleteDependencies(t *testing.T) {
	t.Parallel()

	degraded := NewGRPCGateway(GRPCGatewayDeps{})
	if _, ok := degraded.(GRPCGateway); ok {
		t.Fatalf("expected incomplete dependency set to return unavailable gateway")
	}

	ready := NewGRPCGateway(GRPCGatewayDeps{
		CampaignClient:           &contractCampaignClient{},
		ParticipantClient:        &contractParticipantClient{},
		CharacterClient:          &fakeCharacterWorkflowClient{},
		DaggerheartContentClient: &fakeDaggerheartContentClient{},
		SessionClient:            &contractSessionClient{},
		InviteClient:             &contractInviteClient{},
		AuthorizationClient:      contractAuthorizationClient{},
	})
	if _, ok := ready.(GRPCGateway); !ok {
		t.Fatalf("expected complete dependency set to return grpc gateway")
	}
}

func TestListCampaignsAndWorkspaceMapping(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.January, 2, 10, 0, 0, 0, time.UTC)
	client := &contractCampaignClient{
		listResp: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{
			nil,
			{
				Id:               "c1",
				Name:             "",
				ThemePrompt:      "A long stormy voyage",
				ParticipantCount: 3,
				CharacterCount:   5,
				CreatedAt:        timestamppb.New(createdAt),
				UpdatedAt:        timestamppb.New(updatedAt),
			},
		}},
		getResp: &statev1.GetCampaignResponse{Campaign: &statev1.Campaign{
			Id:               "c1",
			Name:             "",
			System:           commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode:           statev1.GmMode_HYBRID,
			Status:           statev1.CampaignStatus_ACTIVE,
			Locale:           commonv1.Locale_LOCALE_PT_BR,
			Intent:           statev1.CampaignIntent_STANDARD,
			AccessPolicy:     statev1.CampaignAccessPolicy_PRIVATE,
			ParticipantCount: 3,
			CharacterCount:   5,
		}},
	}
	gateway := GRPCGateway{Client: client, AssetBaseURL: "https://cdn.example.com"}

	list, err := gateway.ListCampaigns(context.Background())
	if err != nil {
		t.Fatalf("ListCampaigns() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].Name != "c1" {
		t.Fatalf("campaign name = %q, want id fallback", list[0].Name)
	}
	if list[0].ParticipantCount != "3" || list[0].CharacterCount != "5" {
		t.Fatalf("counts = (%q,%q), want (3,5)", list[0].ParticipantCount, list[0].CharacterCount)
	}
	if list[0].CreatedAtUnixNano != createdAt.UnixNano() {
		t.Fatalf("CreatedAtUnixNano = %d, want %d", list[0].CreatedAtUnixNano, createdAt.UnixNano())
	}
	if list[0].UpdatedAtUnixNano != updatedAt.UnixNano() {
		t.Fatalf("UpdatedAtUnixNano = %d, want %d", list[0].UpdatedAtUnixNano, updatedAt.UnixNano())
	}

	name, err := gateway.CampaignName(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignName() error = %v", err)
	}
	if name != "" {
		t.Fatalf("CampaignName() = %q, want empty when backend name is empty", name)
	}

	workspace, err := gateway.CampaignWorkspace(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignWorkspace() error = %v", err)
	}
	if workspace.ID != "c1" {
		t.Fatalf("workspace.ID = %q, want %q", workspace.ID, "c1")
	}
	if workspace.Name != "c1" {
		t.Fatalf("workspace.Name = %q, want id fallback", workspace.Name)
	}
	if workspace.System != "Daggerheart" || workspace.GMMode != "Hybrid" || workspace.Status != "Active" {
		t.Fatalf("workspace labels = %#v", workspace)
	}
	if workspace.Locale != "Portuguese (Brazil)" || workspace.Intent != "Standard" || workspace.AccessPolicy != "Private" {
		t.Fatalf("workspace labels = %#v", workspace)
	}

	client.getResp = &statev1.GetCampaignResponse{}
	if _, err := gateway.CampaignWorkspace(context.Background(), "c1"); err == nil {
		t.Fatalf("expected not found error when campaign is missing")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("CampaignWorkspace HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

func TestEntityReadersMapParticipantsCharactersSessionsAndInvites(t *testing.T) {
	t.Parallel()

	participantClient := &contractParticipantClient{listResp: &statev1.ListParticipantsResponse{Participants: []*statev1.Participant{{
		Id:             "p1",
		UserId:         "user-1",
		Name:           "Lead",
		Role:           statev1.ParticipantRole_GM,
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
		Controller:     statev1.Controller_CONTROLLER_HUMAN,
	}}}}
	characterClient := &fakeCharacterWorkflowClient{listResp: &statev1.ListCharactersResponse{Characters: []*statev1.Character{{
		Id:            "char-1",
		Name:          "Aria",
		Kind:          statev1.CharacterKind_PC,
		ParticipantId: wrapperspb.String("p1"),
	}}}}
	sessionClient := &contractSessionClient{listResp: &statev1.ListSessionsResponse{Sessions: []*statev1.Session{{
		Id:     "sess-1",
		Name:   "Session One",
		Status: statev1.SessionStatus_SESSION_ACTIVE,
	}}}}
	inviteClient := &contractInviteClient{listResp: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{{
		Id:              "inv-1",
		ParticipantId:   "p1",
		RecipientUserId: "user-2",
		Status:          statev1.InviteStatus_PENDING,
	}}}}

	gateway := GRPCGateway{
		ParticipantClient: participantClient,
		CharacterClient:   characterClient,
		SessionClient:     sessionClient,
		InviteClient:      inviteClient,
		AssetBaseURL:      "https://cdn.example.com",
	}

	participants, err := gateway.CampaignParticipants(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignParticipants() error = %v", err)
	}
	if len(participants) != 1 || participants[0].Name != "Lead" || participants[0].Role != "GM" || participants[0].CampaignAccess != "Owner" {
		t.Fatalf("participants = %#v", participants)
	}

	characters, err := gateway.CampaignCharacters(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignCharacters() error = %v", err)
	}
	if len(characters) != 1 || characters[0].Controller != "Lead" || characters[0].Kind != "PC" {
		t.Fatalf("characters = %#v", characters)
	}

	sessions, err := gateway.CampaignSessions(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignSessions() error = %v", err)
	}
	if len(sessions) != 1 || sessions[0].Status != "Active" {
		t.Fatalf("sessions = %#v", sessions)
	}

	invites, err := gateway.CampaignInvites(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignInvites() error = %v", err)
	}
	if len(invites) != 1 || invites[0].Status != "Pending" {
		t.Fatalf("invites = %#v", invites)
	}
}

func TestCreateCampaignMapsInputAndValidatesResponse(t *testing.T) {
	t.Parallel()

	client := &contractCampaignClient{}
	gateway := GRPCGateway{Client: client}

	created, err := gateway.CreateCampaign(context.Background(), campaignapp.CreateCampaignInput{
		Name:        "Smoke Campaign",
		Locale:      language.MustParse("pt-BR"),
		System:      campaignapp.GameSystemDaggerheart,
		GMMode:      campaignapp.GmModeHybrid,
		ThemePrompt: "Route contract",
	})
	if err != nil {
		t.Fatalf("CreateCampaign() error = %v", err)
	}
	if created.CampaignID != "c-created" {
		t.Fatalf("CreateCampaign() id = %q, want %q", created.CampaignID, "c-created")
	}
	if client.lastCreateReq == nil {
		t.Fatalf("expected CreateCampaign request")
	}
	if client.lastCreateReq.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("request locale = %v, want %v", client.lastCreateReq.GetLocale(), commonv1.Locale_LOCALE_PT_BR)
	}
	if client.lastCreateReq.GetSystem() != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("request system = %v, want %v", client.lastCreateReq.GetSystem(), commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	}
	if client.lastCreateReq.GetGmMode() != statev1.GmMode_HYBRID {
		t.Fatalf("request gm mode = %v, want %v", client.lastCreateReq.GetGmMode(), statev1.GmMode_HYBRID)
	}
	if client.lastCreateReq.GetThemePrompt() != "Route contract" {
		t.Fatalf("request theme prompt = %q, want %q", client.lastCreateReq.GetThemePrompt(), "Route contract")
	}

	client.createResp = &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{Id: "   "}}
	if _, err := gateway.CreateCampaign(context.Background(), campaignapp.CreateCampaignInput{Name: "Missing ID", Locale: language.Und}); err == nil {
		t.Fatalf("expected CreateCampaign() error for empty campaign id")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusInternalServerError {
		t.Fatalf("CreateCampaign() HTTPStatus(err) = %d, want %d", got, http.StatusInternalServerError)
	}

	client.createErr = errors.New("create failed")
	if _, err := gateway.CreateCampaign(context.Background(), campaignapp.CreateCampaignInput{Name: "Broken", Locale: language.AmericanEnglish}); err == nil {
		t.Fatalf("expected CreateCampaign() transport error")
	}
}

func TestMutationReadersValidateAndMapTransportErrors(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{}
	if err := gateway.StartSession(context.Background(), "c1", campaignapp.StartSessionInput{Name: "Session"}); err == nil {
		t.Fatalf("expected unavailable StartSession error")
	}
	if err := gateway.EndSession(context.Background(), "c1", campaignapp.EndSessionInput{SessionID: "sess-1"}); err == nil {
		t.Fatalf("expected unavailable EndSession error")
	}
	if err := gateway.CreateInvite(context.Background(), "c1", campaignapp.CreateInviteInput{ParticipantID: "p1"}); err == nil {
		t.Fatalf("expected unavailable CreateInvite error")
	}
	if err := gateway.RevokeInvite(context.Background(), "c1", campaignapp.RevokeInviteInput{InviteID: "inv-1"}); err == nil {
		t.Fatalf("expected unavailable RevokeInvite error")
	}

	sessionClient := &contractSessionClient{startErr: status.Error(codes.InvalidArgument, "bad session"), endErr: status.Error(codes.InvalidArgument, "bad session")}
	inviteClient := &contractInviteClient{createErr: status.Error(codes.InvalidArgument, "bad invite"), revokeErr: status.Error(codes.InvalidArgument, "bad invite")}
	gateway = GRPCGateway{SessionClient: sessionClient, InviteClient: inviteClient}

	if err := gateway.StartSession(context.Background(), "c1", campaignapp.StartSessionInput{Name: "Session"}); err == nil {
		t.Fatalf("expected StartSession mapping error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("StartSession HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	} else if got := apperrors.LocalizationKey(err); got != "error.web.message.failed_to_start_session" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.failed_to_start_session")
	} else {
		assertAppErrorKind(t, err, apperrors.KindInvalidInput)
	}

	sessionClient.startErr = status.Error(codes.FailedPrecondition, "campaign readiness requires at least one player participant")
	if err := gateway.StartSession(context.Background(), "c1", campaignapp.StartSessionInput{Name: "Session"}); err == nil {
		t.Fatalf("expected StartSession conflict mapping error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusConflict {
		t.Fatalf("StartSession conflict HTTPStatus(err) = %d, want %d", got, http.StatusConflict)
	} else if got := apperrors.LocalizationKey(err); got != "error.web.message.failed_to_start_session" {
		t.Fatalf("LocalizationKey(conflict err) = %q, want %q", got, "error.web.message.failed_to_start_session")
	} else {
		assertAppErrorKind(t, err, apperrors.KindConflict)
	}

	if err := gateway.EndSession(context.Background(), "c1", campaignapp.EndSessionInput{SessionID: "sess-1"}); err == nil {
		t.Fatalf("expected EndSession mapping error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("EndSession HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	} else if got := apperrors.LocalizationKey(err); got != "error.web.message.failed_to_end_session" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.failed_to_end_session")
	} else {
		assertAppErrorKind(t, err, apperrors.KindInvalidInput)
	}

	sessionClient.endErr = status.Error(codes.FailedPrecondition, "session is not active")
	if err := gateway.EndSession(context.Background(), "c1", campaignapp.EndSessionInput{SessionID: "sess-1"}); err == nil {
		t.Fatalf("expected EndSession conflict mapping error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusConflict {
		t.Fatalf("EndSession conflict HTTPStatus(err) = %d, want %d", got, http.StatusConflict)
	} else if got := apperrors.LocalizationKey(err); got != "error.web.message.failed_to_end_session" {
		t.Fatalf("LocalizationKey(conflict err) = %q, want %q", got, "error.web.message.failed_to_end_session")
	} else {
		assertAppErrorKind(t, err, apperrors.KindConflict)
	}

	if err := gateway.CreateInvite(context.Background(), "c1", campaignapp.CreateInviteInput{ParticipantID: "p1", RecipientUserID: "user-2"}); err == nil {
		t.Fatalf("expected CreateInvite mapping error")
	} else if got := apperrors.LocalizationKey(err); got != "error.web.message.failed_to_create_invite" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.failed_to_create_invite")
	}
	if err := gateway.RevokeInvite(context.Background(), "c1", campaignapp.RevokeInviteInput{InviteID: "inv-1"}); err == nil {
		t.Fatalf("expected RevokeInvite mapping error")
	} else if got := apperrors.LocalizationKey(err); got != "error.web.message.failed_to_revoke_invite" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.failed_to_revoke_invite")
	}

	if err := gateway.EndSession(context.Background(), "c1", campaignapp.EndSessionInput{}); err == nil {
		t.Fatalf("expected session_id validation error")
	}
	if err := gateway.CreateInvite(context.Background(), "c1", campaignapp.CreateInviteInput{}); err == nil {
		t.Fatalf("expected participant_id validation error")
	}
	if err := gateway.RevokeInvite(context.Background(), "c1", campaignapp.RevokeInviteInput{}); err == nil {
		t.Fatalf("expected invite_id validation error")
	}
}

func assertAppErrorKind(t *testing.T, err error, want apperrors.Kind) {
	t.Helper()
	var appErr apperrors.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected typed app error, got %T (%v)", err, err)
	}
	if appErr.Kind != want {
		t.Fatalf("appErr.Kind = %q, want %q", appErr.Kind, want)
	}
}

func TestCanCampaignActionAndHelperMappings(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{}
	decision, err := gateway.CanCampaignAction(context.Background(), "c1", campaignapp.AuthorizationActionMutate, campaignapp.AuthorizationResourceCharacter, &campaignapp.AuthorizationTarget{ResourceID: "char-1"})
	if err != nil {
		t.Fatalf("CanCampaignAction() with nil client error = %v", err)
	}
	if decision.Evaluated {
		t.Fatalf("expected unevaluated decision with nil auth client")
	}

	gateway.AuthorizationClient = contractAuthorizationClient{canResp: &statev1.CanResponse{Allowed: true, ReasonCode: "AUTHZ_ALLOW_RESOURCE_OWNER"}}
	decision, err = gateway.CanCampaignAction(context.Background(), "c1", campaignapp.AuthorizationActionMutate, campaignapp.AuthorizationResourceCharacter, &campaignapp.AuthorizationTarget{ResourceID: "char-1"})
	if err != nil {
		t.Fatalf("CanCampaignAction() error = %v", err)
	}
	if !decision.Evaluated || !decision.Allowed || decision.ReasonCode != "AUTHZ_ALLOW_RESOURCE_OWNER" {
		t.Fatalf("decision = %#v", decision)
	}

	if got := campaignSystemLabel(commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED); got != "Unspecified" {
		t.Fatalf("campaignSystemLabel() = %q", got)
	}
	if got := campaignGMModeLabel(statev1.GmMode_HUMAN); got != "Human" {
		t.Fatalf("campaignGMModeLabel() = %q", got)
	}
	if got := campaignGMModeLabel(statev1.GmMode_AI); got != "AI" {
		t.Fatalf("campaignGMModeLabel() = %q", got)
	}
	if got := campaignGMModeLabel(statev1.GmMode_GM_MODE_UNSPECIFIED); got != "Unspecified" {
		t.Fatalf("campaignGMModeLabel() unspecified = %q", got)
	}
	if got := campaignStatusLabel(statev1.CampaignStatus_ARCHIVED); got != "Archived" {
		t.Fatalf("campaignStatusLabel() = %q", got)
	}
	if got := campaignStatusLabel(statev1.CampaignStatus_DRAFT); got != "Draft" {
		t.Fatalf("campaignStatusLabel(draft) = %q", got)
	}
	if got := campaignStatusLabel(statev1.CampaignStatus_ACTIVE); got != "Active" {
		t.Fatalf("campaignStatusLabel(active) = %q", got)
	}
	if got := campaignStatusLabel(statev1.CampaignStatus_COMPLETED); got != "Completed" {
		t.Fatalf("campaignStatusLabel(completed) = %q", got)
	}
	if got := campaignStatusLabel(statev1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED); got != "Unspecified" {
		t.Fatalf("campaignStatusLabel(unspecified) = %q", got)
	}
	if got := campaignLocaleLabel(commonv1.Locale_LOCALE_EN_US); got != "English (US)" {
		t.Fatalf("campaignLocaleLabel() = %q", got)
	}
	if got := campaignLocaleLabel(commonv1.Locale_LOCALE_PT_BR); got != "Portuguese (Brazil)" {
		t.Fatalf("campaignLocaleLabel(pt-BR) = %q", got)
	}
	if got := campaignLocaleLabel(commonv1.Locale_LOCALE_UNSPECIFIED); got != "Unspecified" {
		t.Fatalf("campaignLocaleLabel(unspecified) = %q", got)
	}
	if got := campaignIntentLabel(statev1.CampaignIntent_STARTER); got != "Starter" {
		t.Fatalf("campaignIntentLabel() = %q", got)
	}
	if got := campaignIntentLabel(statev1.CampaignIntent_STANDARD); got != "Standard" {
		t.Fatalf("campaignIntentLabel(standard) = %q", got)
	}
	if got := campaignIntentLabel(statev1.CampaignIntent_SANDBOX); got != "Sandbox" {
		t.Fatalf("campaignIntentLabel(sandbox) = %q", got)
	}
	if got := campaignIntentLabel(statev1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED); got != "Unspecified" {
		t.Fatalf("campaignIntentLabel(unspecified) = %q", got)
	}
	if got := campaignAccessPolicyLabel(statev1.CampaignAccessPolicy_RESTRICTED); got != "Restricted" {
		t.Fatalf("campaignAccessPolicyLabel() = %q", got)
	}
	if got := campaignAccessPolicyLabel(statev1.CampaignAccessPolicy_PRIVATE); got != "Private" {
		t.Fatalf("campaignAccessPolicyLabel(private) = %q", got)
	}
	if got := campaignAccessPolicyLabel(statev1.CampaignAccessPolicy_PUBLIC); got != "Public" {
		t.Fatalf("campaignAccessPolicyLabel(public) = %q", got)
	}
	if got := campaignAccessPolicyLabel(statev1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED); got != "Unspecified" {
		t.Fatalf("campaignAccessPolicyLabel(unspecified) = %q", got)
	}
	if got := participantDisplayName(nil); got != "Unknown participant" {
		t.Fatalf("participantDisplayName(nil) = %q", got)
	}
	if got := participantDisplayName(&statev1.Participant{Name: "  Dana  "}); got != "Dana" {
		t.Fatalf("participantDisplayName(name) = %q", got)
	}
	if got := participantDisplayName(&statev1.Participant{UserId: " user-1 "}); got != "user-1" {
		t.Fatalf("participantDisplayName(user) = %q", got)
	}
	if got := participantDisplayName(&statev1.Participant{Id: " p-1 "}); got != "p-1" {
		t.Fatalf("participantDisplayName(id) = %q", got)
	}
	if got := participantRoleLabel(statev1.ParticipantRole_PLAYER); got != "Player" {
		t.Fatalf("participantRoleLabel() = %q", got)
	}
	if got := participantRoleLabel(statev1.ParticipantRole_GM); got != "GM" {
		t.Fatalf("participantRoleLabel(gm) = %q", got)
	}
	if got := participantRoleLabel(statev1.ParticipantRole(99)); got != "Unspecified" {
		t.Fatalf("participantRoleLabel(unspecified) = %q", got)
	}
	if got := participantCampaignAccessLabel(statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER); got != "Member" {
		t.Fatalf("participantCampaignAccessLabel() = %q", got)
	}
	if got := participantCampaignAccessLabel(statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER); got != "Manager" {
		t.Fatalf("participantCampaignAccessLabel(manager) = %q", got)
	}
	if got := participantCampaignAccessLabel(statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER); got != "Owner" {
		t.Fatalf("participantCampaignAccessLabel(owner) = %q", got)
	}
	if got := participantCampaignAccessLabel(statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED); got != "Unspecified" {
		t.Fatalf("participantCampaignAccessLabel(unspecified) = %q", got)
	}
	if got := participantControllerLabel(statev1.Controller_CONTROLLER_AI); got != "AI" {
		t.Fatalf("participantControllerLabel() = %q", got)
	}
	if got := participantControllerLabel(statev1.Controller_CONTROLLER_HUMAN); got != "Human" {
		t.Fatalf("participantControllerLabel(human) = %q", got)
	}
	if got := participantControllerLabel(statev1.Controller_CONTROLLER_UNSPECIFIED); got != "Unspecified" {
		t.Fatalf("participantControllerLabel(unspecified) = %q", got)
	}
	if got := characterDisplayName(nil); got != "Unknown character" {
		t.Fatalf("characterDisplayName(nil) = %q", got)
	}
	if got := characterDisplayName(&statev1.Character{Name: "  Aria  "}); got != "Aria" {
		t.Fatalf("characterDisplayName(name) = %q", got)
	}
	if got := characterDisplayName(&statev1.Character{Id: " char-1 "}); got != "char-1" {
		t.Fatalf("characterDisplayName(id) = %q", got)
	}
	if got := characterKindLabel(statev1.CharacterKind_NPC); got != "NPC" {
		t.Fatalf("characterKindLabel() = %q", got)
	}
	if got := characterKindLabel(statev1.CharacterKind_PC); got != "PC" {
		t.Fatalf("characterKindLabel(pc) = %q", got)
	}
	if got := characterKindLabel(statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED); got != "Unspecified" {
		t.Fatalf("characterKindLabel(unspecified) = %q", got)
	}
	if got := sessionStatusLabel(statev1.SessionStatus_SESSION_ENDED); got != "Ended" {
		t.Fatalf("sessionStatusLabel() = %q", got)
	}
	if got := sessionStatusLabel(statev1.SessionStatus_SESSION_ACTIVE); got != "Active" {
		t.Fatalf("sessionStatusLabel(active) = %q", got)
	}
	if got := sessionStatusLabel(statev1.SessionStatus_SESSION_STATUS_UNSPECIFIED); got != "Unspecified" {
		t.Fatalf("sessionStatusLabel(unspecified) = %q", got)
	}
	if got := inviteStatusLabel(statev1.InviteStatus_CLAIMED); got != "Claimed" {
		t.Fatalf("inviteStatusLabel() = %q", got)
	}
	if got := inviteStatusLabel(statev1.InviteStatus_PENDING); got != "Pending" {
		t.Fatalf("inviteStatusLabel(pending) = %q", got)
	}
	if got := inviteStatusLabel(statev1.InviteStatus_REVOKED); got != "Revoked" {
		t.Fatalf("inviteStatusLabel(revoked) = %q", got)
	}
	if got := inviteStatusLabel(statev1.InviteStatus_INVITE_STATUS_UNSPECIFIED); got != "Unspecified" {
		t.Fatalf("inviteStatusLabel(unspecified) = %q", got)
	}
	if got := timestampString(nil); got != "" {
		t.Fatalf("timestampString(nil) = %q", got)
	}
	timestamp := timestamppb.Now()
	if got := timestampString(timestamp); got == "" {
		t.Fatalf("timestampString(non-nil) = empty, want formatted value")
	}
	if got := int32ValueString(nil); got != "" {
		t.Fatalf("int32ValueString(nil) = %q", got)
	}
	if got := int32ValueString(wrapperspb.Int32(-7)); got != "-7" {
		t.Fatalf("int32ValueString(value) = %q, want %q", got, "-7")
	}
	if got := daggerheartHeritageKindLabel(daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY); got != "ancestry" {
		t.Fatalf("daggerheartHeritageKindLabel(ancestry) = %q", got)
	}
	if got := daggerheartHeritageKindLabel(daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_COMMUNITY); got != "community" {
		t.Fatalf("daggerheartHeritageKindLabel(community) = %q", got)
	}
	if got := daggerheartHeritageKindLabel(daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_UNSPECIFIED); got != "" {
		t.Fatalf("daggerheartHeritageKindLabel(unspecified) = %q, want empty", got)
	}
	if got := daggerheartWeaponCategoryLabel(daggerheartv1.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_PRIMARY); got != "primary" {
		t.Fatalf("daggerheartWeaponCategoryLabel(primary) = %q", got)
	}
	if got := daggerheartWeaponCategoryLabel(daggerheartv1.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_SECONDARY); got != "secondary" {
		t.Fatalf("daggerheartWeaponCategoryLabel(secondary) = %q", got)
	}
	if got := daggerheartWeaponCategoryLabel(daggerheartv1.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_UNSPECIFIED); got != "" {
		t.Fatalf("daggerheartWeaponCategoryLabel(unspecified) = %q, want empty", got)
	}
	if got := mapGameSystemToProto(campaignapp.GameSystemDaggerheart); got != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("mapGameSystemToProto() = %v", got)
	}
	if got := mapGameSystemToProto(campaignapp.GameSystemUnspecified); got != commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		t.Fatalf("mapGameSystemToProto(unspecified) = %v", got)
	}
	if got := mapGmModeToProto(campaignapp.GmModeHybrid); got != statev1.GmMode_HYBRID {
		t.Fatalf("mapGmModeToProto() = %v", got)
	}
	if got := mapGmModeToProto(campaignapp.GmModeHuman); got != statev1.GmMode_HUMAN {
		t.Fatalf("mapGmModeToProto(human) = %v", got)
	}
	if got := mapGmModeToProto(campaignapp.GmModeAI); got != statev1.GmMode_AI {
		t.Fatalf("mapGmModeToProto(ai) = %v", got)
	}
	if got := mapGmModeToProto(campaignapp.GmModeUnspecified); got != statev1.GmMode_GM_MODE_UNSPECIFIED {
		t.Fatalf("mapGmModeToProto(unspecified) = %v", got)
	}
	if got := mapCharacterKindToProto(campaignapp.CharacterKindPC); got != statev1.CharacterKind_PC {
		t.Fatalf("mapCharacterKindToProto(pc) = %v", got)
	}
	if got := mapCharacterKindToProto(campaignapp.CharacterKindNPC); got != statev1.CharacterKind_NPC {
		t.Fatalf("mapCharacterKindToProto(npc) = %v", got)
	}
	if got := mapCharacterKindToProto(campaignapp.CharacterKindUnspecified); got != statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED {
		t.Fatalf("mapCharacterKindToProto(unspecified) = %v", got)
	}

	if got := campaignCreatedAtUnixNano(nil); got != 0 {
		t.Fatalf("campaignCreatedAtUnixNano(nil) = %d, want 0", got)
	}
	if got := campaignUpdatedAtUnixNano(nil); got != 0 {
		t.Fatalf("campaignUpdatedAtUnixNano(nil) = %d, want 0", got)
	}
	createdOnly := timestamppb.Now()
	if got := campaignUpdatedAtUnixNano(&statev1.Campaign{CreatedAt: createdOnly}); got != createdOnly.AsTime().UTC().UnixNano() {
		t.Fatalf("campaignUpdatedAtUnixNano(created-only) = %d, want %d", got, createdOnly.AsTime().UTC().UnixNano())
	}
}

func TestMapCampaignCharacterCreationStepToProtoWrapper(t *testing.T) {
	t.Parallel()

	step := &campaignapp.CampaignCharacterCreationStepInput{Details: &campaignapp.CampaignCharacterCreationStepDetails{}}
	protoStep, err := MapCampaignCharacterCreationStepToProto(step)
	if err != nil {
		t.Fatalf("MapCampaignCharacterCreationStepToProto() error = %v", err)
	}
	if protoStep == nil {
		t.Fatalf("expected non-nil proto step")
	}
}

type contractCampaignClient struct {
	listResp *statev1.ListCampaignsResponse
	listErr  error
	getResp  *statev1.GetCampaignResponse
	getErr   error

	createResp    *statev1.CreateCampaignResponse
	createErr     error
	lastCreateReq *statev1.CreateCampaignRequest
}

func (c *contractCampaignClient) ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	if c.listErr != nil {
		return nil, c.listErr
	}
	if c.listResp != nil {
		return c.listResp, nil
	}
	return &statev1.ListCampaignsResponse{}, nil
}

func (c *contractCampaignClient) GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	if c.getErr != nil {
		return nil, c.getErr
	}
	if c.getResp != nil {
		return c.getResp, nil
	}
	return &statev1.GetCampaignResponse{}, nil
}

func (c *contractCampaignClient) CreateCampaign(_ context.Context, req *statev1.CreateCampaignRequest, _ ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	c.lastCreateReq = req
	if c.createErr != nil {
		return nil, c.createErr
	}
	if c.createResp != nil {
		return c.createResp, nil
	}
	return &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{Id: "c-created"}}, nil
}

type contractParticipantClient struct {
	listResp *statev1.ListParticipantsResponse
	listErr  error
}

func (c *contractParticipantClient) ListParticipants(context.Context, *statev1.ListParticipantsRequest, ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	if c.listErr != nil {
		return nil, c.listErr
	}
	if c.listResp != nil {
		return c.listResp, nil
	}
	return &statev1.ListParticipantsResponse{}, nil
}

type contractSessionClient struct {
	listResp *statev1.ListSessionsResponse
	listErr  error
	startErr error
	endErr   error
}

func (c *contractSessionClient) ListSessions(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	if c.listErr != nil {
		return nil, c.listErr
	}
	if c.listResp != nil {
		return c.listResp, nil
	}
	return &statev1.ListSessionsResponse{}, nil
}

func (c *contractSessionClient) StartSession(context.Context, *statev1.StartSessionRequest, ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
	if c.startErr != nil {
		return nil, c.startErr
	}
	return &statev1.StartSessionResponse{}, nil
}

func (c *contractSessionClient) EndSession(context.Context, *statev1.EndSessionRequest, ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
	if c.endErr != nil {
		return nil, c.endErr
	}
	return &statev1.EndSessionResponse{}, nil
}

type contractInviteClient struct {
	listResp  *statev1.ListInvitesResponse
	listErr   error
	createErr error
	revokeErr error
}

func (c *contractInviteClient) ListInvites(context.Context, *statev1.ListInvitesRequest, ...grpc.CallOption) (*statev1.ListInvitesResponse, error) {
	if c.listErr != nil {
		return nil, c.listErr
	}
	if c.listResp != nil {
		return c.listResp, nil
	}
	return &statev1.ListInvitesResponse{}, nil
}

func (c *contractInviteClient) CreateInvite(context.Context, *statev1.CreateInviteRequest, ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
	if c.createErr != nil {
		return nil, c.createErr
	}
	return &statev1.CreateInviteResponse{}, nil
}

func (c *contractInviteClient) RevokeInvite(context.Context, *statev1.RevokeInviteRequest, ...grpc.CallOption) (*statev1.RevokeInviteResponse, error) {
	if c.revokeErr != nil {
		return nil, c.revokeErr
	}
	return &statev1.RevokeInviteResponse{}, nil
}

type contractAuthorizationClient struct {
	canResp *statev1.CanResponse
	canErr  error
	nilCan  bool
}

func (c contractAuthorizationClient) Can(context.Context, *statev1.CanRequest, ...grpc.CallOption) (*statev1.CanResponse, error) {
	if c.canErr != nil {
		return nil, c.canErr
	}
	if c.nilCan {
		return nil, nil
	}
	if c.canResp != nil {
		return c.canResp, nil
	}
	return &statev1.CanResponse{}, nil
}

func (c contractAuthorizationClient) BatchCan(context.Context, *statev1.BatchCanRequest, ...grpc.CallOption) (*statev1.BatchCanResponse, error) {
	return &statev1.BatchCanResponse{}, nil
}

func TestListCampaignsForwardsClientErrors(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{Client: &contractCampaignClient{listErr: errors.New("boom")}}
	if _, err := gateway.ListCampaigns(context.Background()); err == nil {
		t.Fatalf("expected ListCampaigns() client error")
	}
}

func TestCampaignWorkspaceForwardsClientErrors(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{Client: &contractCampaignClient{getErr: errors.New("boom")}}
	if _, err := gateway.CampaignWorkspace(context.Background(), "c1"); err == nil {
		t.Fatalf("expected CampaignWorkspace() client error")
	}
}

func TestParticipantAndCharacterClientRequired(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{}
	if _, err := gateway.CampaignParticipants(context.Background(), "c1"); err == nil {
		t.Fatalf("expected missing participant client error")
	}
	if _, err := gateway.CampaignCharacters(context.Background(), "c1"); err == nil {
		t.Fatalf("expected missing character client error")
	}
}

func TestEmptyCampaignIDReturnsEmptyCollections(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{
		ParticipantClient: &contractParticipantClient{},
		CharacterClient:   &fakeCharacterWorkflowClient{},
		SessionClient:     &contractSessionClient{},
		InviteClient:      &contractInviteClient{},
	}

	participants, err := gateway.CampaignParticipants(context.Background(), " ")
	if err != nil {
		t.Fatalf("CampaignParticipants() error = %v", err)
	}
	if len(participants) != 0 {
		t.Fatalf("len(participants) = %d, want 0", len(participants))
	}

	characters, err := gateway.CampaignCharacters(context.Background(), " ")
	if err != nil {
		t.Fatalf("CampaignCharacters() error = %v", err)
	}
	if len(characters) != 0 {
		t.Fatalf("len(characters) = %d, want 0", len(characters))
	}

	sessions, err := gateway.CampaignSessions(context.Background(), " ")
	if err != nil {
		t.Fatalf("CampaignSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("len(sessions) = %d, want 0", len(sessions))
	}

	invites, err := gateway.CampaignInvites(context.Background(), " ")
	if err != nil {
		t.Fatalf("CampaignInvites() error = %v", err)
	}
	if len(invites) != 0 {
		t.Fatalf("len(invites) = %d, want 0", len(invites))
	}
}

func TestMutationValidationForCampaignID(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{SessionClient: &contractSessionClient{}, InviteClient: &contractInviteClient{}}

	if err := gateway.StartSession(context.Background(), " ", campaignapp.StartSessionInput{Name: "Session"}); err == nil {
		t.Fatalf("expected campaign id validation error")
	}
	if err := gateway.EndSession(context.Background(), " ", campaignapp.EndSessionInput{SessionID: "sess-1"}); err == nil {
		t.Fatalf("expected campaign id validation error")
	}
	if err := gateway.CreateInvite(context.Background(), " ", campaignapp.CreateInviteInput{ParticipantID: "p1"}); err == nil {
		t.Fatalf("expected campaign id validation error")
	}
	if err := gateway.RevokeInvite(context.Background(), " ", campaignapp.RevokeInviteInput{InviteID: "inv-1"}); err == nil {
		t.Fatalf("expected campaign id validation error")
	}
}

func TestCanCampaignActionEdgeCases(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{AuthorizationClient: contractAuthorizationClient{}}
	decision, err := gateway.CanCampaignAction(context.Background(), "   ", campaignapp.AuthorizationActionMutate, campaignapp.AuthorizationResourceCharacter, nil)
	if err != nil {
		t.Fatalf("CanCampaignAction() error = %v", err)
	}
	if decision.Evaluated {
		t.Fatalf("expected unevaluated decision when campaign id is empty")
	}

	gateway.AuthorizationClient = contractAuthorizationClient{nilCan: true}
	decision, err = gateway.CanCampaignAction(context.Background(), "c1", campaignapp.AuthorizationActionMutate, campaignapp.AuthorizationResourceCharacter, nil)
	if err != nil {
		t.Fatalf("CanCampaignAction() error = %v", err)
	}
	if decision.Evaluated {
		t.Fatalf("expected unevaluated decision when response is nil")
	}

	gateway.AuthorizationClient = contractAuthorizationClient{canErr: errors.New("boom")}
	if _, err := gateway.CanCampaignAction(context.Background(), "c1", campaignapp.AuthorizationActionMutate, campaignapp.AuthorizationResourceCharacter, nil); err == nil {
		t.Fatalf("expected CanCampaignAction() transport error")
	}
}

func TestAuthorizationProtoMappers(t *testing.T) {
	t.Parallel()

	if got := mapCampaignAuthorizationActionToProto(campaignapp.AuthorizationActionManage); got != statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE {
		t.Fatalf("mapCampaignAuthorizationActionToProto(manage) = %v", got)
	}
	if got := mapCampaignAuthorizationActionToProto(campaignapp.AuthorizationActionMutate); got != statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE {
		t.Fatalf("mapCampaignAuthorizationActionToProto(mutate) = %v", got)
	}
	if got := mapCampaignAuthorizationResourceToProto(campaignapp.AuthorizationResourceSession); got != statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION {
		t.Fatalf("mapCampaignAuthorizationResourceToProto(session) = %v", got)
	}
	if got := mapCampaignAuthorizationResourceToProto(campaignapp.AuthorizationResourceParticipant); got != statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT {
		t.Fatalf("mapCampaignAuthorizationResourceToProto(participant) = %v", got)
	}
	if got := mapCampaignAuthorizationResourceToProto(campaignapp.AuthorizationResourceInvite); got != statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_INVITE {
		t.Fatalf("mapCampaignAuthorizationResourceToProto(invite) = %v", got)
	}
	if got := mapCampaignAuthorizationTargetToProto(nil); got != nil {
		t.Fatalf("mapCampaignAuthorizationTargetToProto(nil) = %#v", got)
	}
	if got := mapCampaignAuthorizationTargetToProto(&campaignapp.AuthorizationTarget{ResourceID: "   "}); got != nil {
		t.Fatalf("mapCampaignAuthorizationTargetToProto(empty) = %#v", got)
	}
	if got := mapCampaignAuthorizationTargetToProto(&campaignapp.AuthorizationTarget{ResourceID: " char-1 "}); got == nil || strings.TrimSpace(got.GetResourceId()) != "char-1" {
		t.Fatalf("mapCampaignAuthorizationTargetToProto(valid) = %#v", got)
	}
}
