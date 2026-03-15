package gateway

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
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
		Page: PageGatewayDeps{
			Workspace:     WorkspaceReadDeps{Campaign: &contractCampaignClient{}},
			SessionRead:   SessionReadDeps{Campaign: &contractCampaignClient{}, Session: &contractSessionClient{}},
			Authorization: AuthorizationDeps{Client: contractAuthorizationClient{}},
		},
		Catalog: CatalogGatewayDeps{
			Read:     CatalogReadDeps{Campaign: &contractCampaignClient{}},
			Mutation: CatalogMutationDeps{Campaign: &contractCampaignClient{}},
		},
		Overview: OverviewGatewayDeps{
			Participants:          ParticipantReadDeps{Participant: &contractParticipantClient{}},
			Workspace:             WorkspaceReadDeps{Campaign: &contractCampaignClient{}},
			Authorization:         AuthorizationDeps{Client: contractAuthorizationClient{}},
			AutomationRead:        AutomationReadDeps{Agent: &contractAgentClient{}},
			AutomationMutation:    AutomationMutationDeps{Campaign: &contractCampaignClient{}},
			ConfigurationMutation: ConfigurationMutationDeps{Campaign: &contractCampaignClient{}},
		},
		Participants: ParticipantGatewayDeps{
			Read:          ParticipantReadDeps{Participant: &contractParticipantClient{}},
			Mutation:      ParticipantMutationDeps{Participant: &contractParticipantClient{}},
			Workspace:     WorkspaceReadDeps{Campaign: &contractCampaignClient{}},
			Authorization: AuthorizationDeps{Client: contractAuthorizationClient{}},
		},
		Characters: CharacterGatewayDeps{
			Read: CharacterReadDeps{
				Character:          &fakeCharacterWorkflowClient{},
				Participant:        &contractParticipantClient{},
				DaggerheartContent: &fakeDaggerheartContentClient{},
			},
			Control:          CharacterControlMutationDeps{Character: &fakeCharacterWorkflowClient{}},
			Mutation:         CharacterMutationDeps{Character: &fakeCharacterWorkflowClient{}},
			Participants:     ParticipantReadDeps{Participant: &contractParticipantClient{}},
			Sessions:         SessionReadDeps{Campaign: &contractCampaignClient{}, Session: &contractSessionClient{}},
			Authorization:    AuthorizationDeps{Client: contractAuthorizationClient{}},
			CreationRead:     CharacterCreationReadDeps{Character: &fakeCharacterWorkflowClient{}, DaggerheartContent: &fakeDaggerheartContentClient{}, DaggerheartAsset: &fakeDaggerheartContentClient{}},
			CreationMutation: CharacterCreationMutationDeps{Character: &fakeCharacterWorkflowClient{}},
		},
		Sessions: SessionGatewayDeps{
			Mutation: SessionMutationDeps{Session: &contractSessionClient{}},
		},
		Invites: InviteGatewayDeps{
			Read: InviteReadDeps{
				Invite:      &contractInviteClient{},
				Participant: &contractParticipantClient{},
				Social:      &contractSocialClient{},
				Auth:        &contractAuthClient{},
			},
			Mutation:      InviteMutationDeps{Invite: &contractInviteClient{}, Auth: &contractAuthClient{}},
			Participants:  ParticipantReadDeps{Participant: &contractParticipantClient{}},
			Authorization: AuthorizationDeps{Client: contractAuthorizationClient{}},
		},
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
	gateway := GRPCGateway{Read: GRPCGatewayReadDeps{Campaign: client}, AssetBaseURL: "https://cdn.example.com"}

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
		Read: GRPCGatewayReadDeps{
			Participant: participantClient,
			Character:   characterClient,
			Session:     sessionClient,
			Invite:      inviteClient,
		},
		AssetBaseURL: "https://cdn.example.com",
	}

	participants, err := gateway.CampaignParticipants(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignParticipants() error = %v", err)
	}
	if len(participants) != 1 || participants[0].Name != "Lead" || participants[0].Role != "GM" || participants[0].CampaignAccess != "Owner" {
		t.Fatalf("participants = %#v", participants)
	}

	characters, err := gateway.CampaignCharacters(context.Background(), "c1", campaignapp.CharacterReadContext{})
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

func TestCampaignCharactersMarksViewerOwnedCharactersFromControllerParticipant(t *testing.T) {
	t.Parallel()

	participantClient := &contractParticipantClient{listResp: &statev1.ListParticipantsResponse{Participants: []*statev1.Participant{
		{Id: "p1", UserId: "user-1", Name: "Lead", Role: statev1.ParticipantRole_PLAYER, Controller: statev1.Controller_CONTROLLER_HUMAN},
		{Id: "p2", UserId: "user-2", Name: "Scout", Role: statev1.ParticipantRole_PLAYER, Controller: statev1.Controller_CONTROLLER_HUMAN},
	}}}
	characterClient := &fakeCharacterWorkflowClient{listResp: &statev1.ListCharactersResponse{Characters: []*statev1.Character{
		{Id: "char-1", Name: "Aria", Kind: statev1.CharacterKind_PC, ParticipantId: wrapperspb.String("p1")},
		{Id: "char-2", Name: "Bramble", Kind: statev1.CharacterKind_PC, ParticipantId: wrapperspb.String("p2")},
	}}}

	gateway := GRPCGateway{
		Read: GRPCGatewayReadDeps{
			Participant: participantClient,
			Character:   characterClient,
		},
	}

	characters, err := gateway.CampaignCharacters(context.Background(), "c1", campaignapp.CharacterReadContext{
		ViewerUserID: "user-1",
	})
	if err != nil {
		t.Fatalf("CampaignCharacters() error = %v", err)
	}
	if len(characters) != 2 {
		t.Fatalf("len(characters) = %d, want 2", len(characters))
	}
	if !characters[0].OwnedByViewer {
		t.Fatalf("expected first character to be viewer-owned: %#v", characters[0])
	}
	if characters[1].OwnedByViewer {
		t.Fatalf("expected second character not to be viewer-owned: %#v", characters[1])
	}
}

func TestCampaignCharactersMapsDaggerheartSummaryWhenProfileAndCatalogResolve(t *testing.T) {
	t.Parallel()

	characterClient := &fakeCharacterWorkflowClient{
		listResp: &statev1.ListCharactersResponse{Characters: []*statev1.Character{{
			Id:            "char-1",
			Name:          "Aria",
			Kind:          statev1.CharacterKind_PC,
			ParticipantId: wrapperspb.String("p1"),
		}}},
		profilesResp: &statev1.ListCharacterProfilesResponse{Profiles: []*statev1.CharacterProfile{{
			CampaignId:  "c1",
			CharacterId: "char-1",
			SystemProfile: &statev1.CharacterProfile_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{
				Level:       2,
				ClassId:     "warrior",
				SubclassId:  "guardian",
				AncestryId:  "drakona",
				CommunityId: "wanderborne",
			}},
		}}},
	}
	contentClient := &fakeDaggerheartContentClient{
		resp: &daggerheartv1.GetDaggerheartContentCatalogResponse{Catalog: &daggerheartv1.DaggerheartContentCatalog{
			Classes:    []*daggerheartv1.DaggerheartClass{{Id: "warrior", Name: "Warrior"}},
			Subclasses: []*daggerheartv1.DaggerheartSubclass{{Id: "guardian", Name: "Guardian"}},
			Heritages: []*daggerheartv1.DaggerheartHeritage{
				{Id: "drakona", Name: "Drakona", Kind: daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY},
				{Id: "wanderborne", Name: "Wanderborne", Kind: daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_COMMUNITY},
			},
		}},
	}
	gateway := GRPCGateway{
		Read: GRPCGatewayReadDeps{
			Participant:        &contractParticipantClient{listResp: &statev1.ListParticipantsResponse{Participants: []*statev1.Participant{{Id: "p1", Name: "Lead"}}}},
			Character:          characterClient,
			DaggerheartContent: contentClient,
		},
	}

	characters, err := gateway.CampaignCharacters(context.Background(), "c1", campaignapp.CharacterReadContext{
		System: "Daggerheart",
		Locale: language.BrazilianPortuguese,
	})
	if err != nil {
		t.Fatalf("CampaignCharacters() error = %v", err)
	}
	if len(characters) != 1 {
		t.Fatalf("len(characters) = %d, want 1", len(characters))
	}
	if characters[0].Daggerheart == nil {
		t.Fatalf("Daggerheart summary = nil, want populated summary")
	}
	if got := characters[0].Daggerheart.Level; got != 2 {
		t.Fatalf("Level = %d, want 2", got)
	}
	if got := characters[0].Daggerheart.ClassName; got != "Warrior" {
		t.Fatalf("ClassName = %q, want %q", got, "Warrior")
	}
	if got := characters[0].Daggerheart.SubclassName; got != "Guardian" {
		t.Fatalf("SubclassName = %q, want %q", got, "Guardian")
	}
	if got := characters[0].Daggerheart.AncestryName; got != "Drakona" {
		t.Fatalf("AncestryName = %q, want %q", got, "Drakona")
	}
	if got := characters[0].Daggerheart.CommunityName; got != "Wanderborne" {
		t.Fatalf("CommunityName = %q, want %q", got, "Wanderborne")
	}
	if characterClient.lastProfilesReq == nil {
		t.Fatalf("expected profile listing request")
	}
	if contentClient.lastReq == nil || contentClient.lastReq.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("catalog locale request = %#v, want pt-BR request", contentClient.lastReq)
	}
	if contentClient.lastAssetMapReq != nil {
		t.Fatalf("characters page should not fetch asset map: %#v", contentClient.lastAssetMapReq)
	}
}

func TestCampaignCharactersSkipsDaggerheartSummaryWhenCatalogNamesDoNotResolve(t *testing.T) {
	t.Parallel()

	characterClient := &fakeCharacterWorkflowClient{
		listResp: &statev1.ListCharactersResponse{Characters: []*statev1.Character{{Id: "char-1", Name: "Aria", Kind: statev1.CharacterKind_PC}}},
		profilesResp: &statev1.ListCharacterProfilesResponse{Profiles: []*statev1.CharacterProfile{{
			CampaignId:  "c1",
			CharacterId: "char-1",
			SystemProfile: &statev1.CharacterProfile_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{
				Level:       2,
				ClassId:     "warrior",
				SubclassId:  "guardian",
				AncestryId:  "drakona",
				CommunityId: "missing-community",
			}},
		}}},
	}
	contentClient := &fakeDaggerheartContentClient{
		resp: &daggerheartv1.GetDaggerheartContentCatalogResponse{Catalog: &daggerheartv1.DaggerheartContentCatalog{
			Classes:    []*daggerheartv1.DaggerheartClass{{Id: "warrior", Name: "Warrior"}},
			Subclasses: []*daggerheartv1.DaggerheartSubclass{{Id: "guardian", Name: "Guardian"}},
			Heritages:  []*daggerheartv1.DaggerheartHeritage{{Id: "drakona", Name: "Drakona", Kind: daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY}},
		}},
	}
	gateway := GRPCGateway{
		Read: GRPCGatewayReadDeps{
			Character:          characterClient,
			DaggerheartContent: contentClient,
		},
	}

	characters, err := gateway.CampaignCharacters(context.Background(), "c1", campaignapp.CharacterReadContext{
		System: "Daggerheart",
		Locale: language.AmericanEnglish,
	})
	if err != nil {
		t.Fatalf("CampaignCharacters() error = %v", err)
	}
	if len(characters) != 1 {
		t.Fatalf("len(characters) = %d, want 1", len(characters))
	}
	if characters[0].Daggerheart != nil {
		t.Fatalf("Daggerheart summary = %#v, want nil when names are incomplete", characters[0].Daggerheart)
	}
}

func TestCampaignCharactersSkipsDaggerheartReadsForNonDaggerheartOptions(t *testing.T) {
	t.Parallel()

	characterClient := &fakeCharacterWorkflowClient{
		listResp: &statev1.ListCharactersResponse{Characters: []*statev1.Character{{Id: "char-1", Name: "Aria", Kind: statev1.CharacterKind_PC}}},
	}
	contentClient := &fakeDaggerheartContentClient{}
	gateway := GRPCGateway{
		Read: GRPCGatewayReadDeps{
			Character:          characterClient,
			DaggerheartContent: contentClient,
		},
	}

	characters, err := gateway.CampaignCharacters(context.Background(), "c1", campaignapp.CharacterReadContext{
		System: "Pathfinder",
		Locale: language.AmericanEnglish,
	})
	if err != nil {
		t.Fatalf("CampaignCharacters() error = %v", err)
	}
	if len(characters) != 1 {
		t.Fatalf("len(characters) = %d, want 1", len(characters))
	}
	if characters[0].Daggerheart != nil {
		t.Fatalf("Daggerheart summary = %#v, want nil", characters[0].Daggerheart)
	}
	if characterClient.lastProfilesReq != nil {
		t.Fatalf("non-daggerheart character read should not request profiles: %#v", characterClient.lastProfilesReq)
	}
	if contentClient.lastReq != nil {
		t.Fatalf("non-daggerheart character read should not request content catalog: %#v", contentClient.lastReq)
	}
}

func TestSearchInviteUsersMapsSocialResults(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{
		Read: GRPCGatewayReadDeps{
			Social: &contractSocialClient{searchResp: &socialv1.SearchUsersResponse{
				Users: []*socialv1.SearchUserResult{{
					UserId:    "user-2",
					Username:  "alice",
					Name:      "Alice",
					IsContact: true,
				}},
			}},
		},
	}

	results, err := gateway.SearchInviteUsers(context.Background(), campaignapp.SearchInviteUsersInput{
		ViewerUserID: "user-1",
		Query:        "al",
		Limit:        8,
	})
	if err != nil {
		t.Fatalf("SearchInviteUsers() error = %v", err)
	}
	if len(results) != 1 || results[0].Username != "alice" || !results[0].IsContact {
		t.Fatalf("results = %#v", results)
	}
}

func TestCampaignSessionReadinessMapsResponse(t *testing.T) {
	t.Parallel()

	client := &contractCampaignClient{
		readinessResp: &statev1.GetCampaignSessionReadinessResponse{
			Readiness: &statev1.CampaignSessionReadiness{
				Ready: false,
				Blockers: []*statev1.CampaignSessionReadinessBlocker{
					{
						Code:    "SESSION_READINESS_AI_GM_PARTICIPANT_REQUIRED",
						Message: "Campaign readiness requires at least one AI-controlled GM participant for AI GM mode",
						Metadata: map[string]string{
							"campaign_id": "c1",
						},
					},
				},
			},
		},
	}
	gateway := GRPCGateway{Read: GRPCGatewayReadDeps{Campaign: client}}

	readiness, err := gateway.CampaignSessionReadiness(context.Background(), "c1", language.BrazilianPortuguese)
	if err != nil {
		t.Fatalf("CampaignSessionReadiness() error = %v", err)
	}
	if readiness.Ready {
		t.Fatalf("readiness.Ready = %v, want false", readiness.Ready)
	}
	if len(readiness.Blockers) != 1 {
		t.Fatalf("len(readiness.Blockers) = %d, want 1", len(readiness.Blockers))
	}
	if got := readiness.Blockers[0].Code; got != "SESSION_READINESS_AI_GM_PARTICIPANT_REQUIRED" {
		t.Fatalf("blocker code = %q, want %q", got, "SESSION_READINESS_AI_GM_PARTICIPANT_REQUIRED")
	}
	if got := readiness.Blockers[0].Metadata["campaign_id"]; got != "c1" {
		t.Fatalf("blocker metadata campaign_id = %q, want %q", got, "c1")
	}
	if client.lastReadinessReq == nil {
		t.Fatalf("expected readiness request capture")
	}
	if client.lastReadinessReq.GetCampaignId() != "c1" {
		t.Fatalf("readiness request campaign_id = %q, want %q", client.lastReadinessReq.GetCampaignId(), "c1")
	}
	if client.lastReadinessReq.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("readiness request locale = %v, want %v", client.lastReadinessReq.GetLocale(), commonv1.Locale_LOCALE_PT_BR)
	}
}

func TestCampaignSessionReadinessMapsErrors(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{Read: GRPCGatewayReadDeps{Campaign: &contractCampaignClient{readinessErr: status.Error(codes.FailedPrecondition, "blocked")}}}
	if _, err := gateway.CampaignSessionReadiness(context.Background(), "c1", language.AmericanEnglish); err == nil {
		t.Fatalf("expected CampaignSessionReadiness() error")
	} else if got := apperrors.LocalizationKey(err); got != "error.web.message.failed_to_load_session_readiness" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.failed_to_load_session_readiness")
	}

	gateway = GRPCGateway{Read: GRPCGatewayReadDeps{Campaign: &contractCampaignClient{readinessResp: &statev1.GetCampaignSessionReadinessResponse{}}}}
	if _, err := gateway.CampaignSessionReadiness(context.Background(), "c1", language.AmericanEnglish); err == nil {
		t.Fatalf("expected nil-readiness response to fail closed")
	}
}

func TestCampaignSessionReadinessEmptyCampaignIDReturnsReady(t *testing.T) {
	t.Parallel()

	client := &contractCampaignClient{readinessErr: status.Error(codes.Internal, "should not be called")}
	gateway := GRPCGateway{Read: GRPCGatewayReadDeps{Campaign: client}}

	readiness, err := gateway.CampaignSessionReadiness(context.Background(), "   ", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("CampaignSessionReadiness() error = %v", err)
	}
	if !readiness.Ready {
		t.Fatalf("readiness.Ready = %v, want true", readiness.Ready)
	}
	if len(readiness.Blockers) != 0 {
		t.Fatalf("len(readiness.Blockers) = %d, want 0", len(readiness.Blockers))
	}
	if client.lastReadinessReq != nil {
		t.Fatalf("expected readiness RPC not to be called for empty campaign id")
	}
}

func TestCampaignParticipantMapsSingleResponse(t *testing.T) {
	t.Parallel()

	participantClient := &contractParticipantClient{
		getResp: &statev1.GetParticipantResponse{Participant: &statev1.Participant{
			Id:             "p1",
			UserId:         "user-1",
			Name:           "Lead",
			Role:           statev1.ParticipantRole_GM,
			CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
			Controller:     statev1.Controller_CONTROLLER_HUMAN,
		}},
	}
	gateway := GRPCGateway{Read: GRPCGatewayReadDeps{Participant: participantClient}, AssetBaseURL: "https://cdn.example.com"}

	participant, err := gateway.CampaignParticipant(context.Background(), "c1", "p1")
	if err != nil {
		t.Fatalf("CampaignParticipant() error = %v", err)
	}
	if participant.ID != "p1" || participant.Name != "Lead" || participant.Role != "GM" || participant.CampaignAccess != "Owner" {
		t.Fatalf("participant = %#v", participant)
	}
}

func TestCreateParticipantMapsInputAndErrors(t *testing.T) {
	t.Parallel()

	participantClient := &contractParticipantClient{}
	gateway := GRPCGateway{Mutation: GRPCGatewayMutationDeps{Participant: participantClient}}

	created, err := gateway.CreateParticipant(context.Background(), "c1", campaignapp.CreateParticipantInput{
		Name:           "Pending Seat",
		Role:           "player",
		CampaignAccess: "manager",
	})
	if err != nil {
		t.Fatalf("CreateParticipant() error = %v", err)
	}
	if created.ParticipantID != "participant-created" {
		t.Fatalf("created.ParticipantID = %q, want %q", created.ParticipantID, "participant-created")
	}
	if participantClient.createReq == nil {
		t.Fatalf("expected CreateParticipant request")
	}
	if participantClient.createReq.GetCampaignId() != "c1" {
		t.Fatalf("create campaign id = %q, want %q", participantClient.createReq.GetCampaignId(), "c1")
	}
	if participantClient.createReq.GetController() != statev1.Controller_CONTROLLER_HUMAN {
		t.Fatalf("create controller = %v, want %v", participantClient.createReq.GetController(), statev1.Controller_CONTROLLER_HUMAN)
	}
	if participantClient.createReq.GetCampaignAccess() != statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER {
		t.Fatalf("create access = %v, want %v", participantClient.createReq.GetCampaignAccess(), statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER)
	}

	if _, err := gateway.CreateParticipant(context.Background(), "c1", campaignapp.CreateParticipantInput{Name: "Pending Seat", Role: "bad", CampaignAccess: "member"}); err == nil {
		t.Fatalf("expected role validation error")
	}
	if _, err := gateway.CreateParticipant(context.Background(), "c1", campaignapp.CreateParticipantInput{Name: "Pending Seat", Role: "player", CampaignAccess: "bad"}); err == nil {
		t.Fatalf("expected access validation error")
	}

	participantClient.createErr = status.Error(codes.InvalidArgument, "invalid create")
	if _, err := gateway.CreateParticipant(context.Background(), "c1", campaignapp.CreateParticipantInput{Name: "Pending Seat", Role: "player", CampaignAccess: "member"}); err == nil {
		t.Fatalf("expected create transport error")
	} else if got := apperrors.LocalizationKey(err); got != "error.web.message.failed_to_create_participant" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.failed_to_create_participant")
	}
}

func TestUpdateParticipantMapsInputAndErrors(t *testing.T) {
	t.Parallel()

	participantClient := &contractParticipantClient{}
	gateway := GRPCGateway{Mutation: GRPCGatewayMutationDeps{Participant: participantClient}}

	err := gateway.UpdateParticipant(context.Background(), "c1", campaignapp.UpdateParticipantInput{
		ParticipantID:  "p1",
		Name:           "Lead Prime",
		Role:           "gm",
		Pronouns:       "they/them",
		CampaignAccess: "manager",
	})
	if err != nil {
		t.Fatalf("UpdateParticipant() error = %v", err)
	}
	if participantClient.updateReq == nil {
		t.Fatalf("expected UpdateParticipant request")
	}
	if participantClient.updateReq.GetCampaignId() != "c1" || participantClient.updateReq.GetParticipantId() != "p1" {
		t.Fatalf("update request ids = %#v", participantClient.updateReq)
	}
	if participantClient.updateReq.GetRole() != statev1.ParticipantRole_GM {
		t.Fatalf("update role = %v, want %v", participantClient.updateReq.GetRole(), statev1.ParticipantRole_GM)
	}
	if participantClient.updateReq.GetCampaignAccess() != statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER {
		t.Fatalf("update access = %v, want %v", participantClient.updateReq.GetCampaignAccess(), statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER)
	}
	if got := strings.TrimSpace(participantClient.updateReq.GetName().GetValue()); got != "Lead Prime" {
		t.Fatalf("update name = %q, want %q", got, "Lead Prime")
	}

	if err := gateway.UpdateParticipant(context.Background(), "c1", campaignapp.UpdateParticipantInput{ParticipantID: "p1", Role: "bad"}); err == nil {
		t.Fatalf("expected role validation error")
	}
	if err := gateway.UpdateParticipant(context.Background(), "c1", campaignapp.UpdateParticipantInput{ParticipantID: "p1", Role: "gm", CampaignAccess: "bad"}); err == nil {
		t.Fatalf("expected access validation error")
	}

	participantClient.updateErr = status.Error(codes.InvalidArgument, "invalid update")
	if err := gateway.UpdateParticipant(context.Background(), "c1", campaignapp.UpdateParticipantInput{ParticipantID: "p1", Name: "Lead Prime", Role: "gm"}); err == nil {
		t.Fatalf("expected update transport error")
	} else if got := apperrors.LocalizationKey(err); got != "error.web.message.failed_to_update_participant" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.failed_to_update_participant")
	}
}

func TestCampaignAIAgentsAndBindingMutations(t *testing.T) {
	t.Parallel()

	agentClient := &contractAgentClient{listResp: &aiv1.ListAgentsResponse{Agents: []*aiv1.Agent{
		nil,
		{Id: "agent-active", Label: "alpha", Status: aiv1.AgentStatus_AGENT_STATUS_ACTIVE, AuthState: aiv1.AgentAuthState_AGENT_AUTH_STATE_READY},
		{Id: "agent-inactive", Label: "beta", Status: aiv1.AgentStatus_AGENT_STATUS_UNSPECIFIED},
	}}}
	campaignClient := &contractCampaignClient{}
	gateway := GRPCGateway{
		Read:     GRPCGatewayReadDeps{Agent: agentClient},
		Mutation: GRPCGatewayMutationDeps{Campaign: campaignClient},
	}

	options, err := gateway.CampaignAIAgents(context.Background())
	if err != nil {
		t.Fatalf("CampaignAIAgents() error = %v", err)
	}
	if len(options) != 2 {
		t.Fatalf("len(options) = %d, want 2", len(options))
	}
	if !options[0].Enabled || options[0].Label != "alpha" {
		t.Fatalf("options[0] = %#v", options[0])
	}
	if options[1].Enabled || options[1].Label != "beta" {
		t.Fatalf("options[1] = %#v", options[1])
	}
	if agentClient.lastListReq == nil || agentClient.lastListReq.GetPageSize() != campaignAIAgentsPageSize {
		t.Fatalf("lastListReq = %#v, want page size %d", agentClient.lastListReq, campaignAIAgentsPageSize)
	}

	if err := gateway.UpdateCampaignAIBinding(context.Background(), "c1", campaignapp.UpdateCampaignAIBindingInput{AIAgentID: "agent-active"}); err != nil {
		t.Fatalf("UpdateCampaignAIBinding(set) error = %v", err)
	}
	if campaignClient.lastSetAIBindingReq == nil || campaignClient.lastSetAIBindingReq.GetAiAgentId() != "agent-active" {
		t.Fatalf("lastSetAIBindingReq = %#v", campaignClient.lastSetAIBindingReq)
	}

	if err := gateway.UpdateCampaignAIBinding(context.Background(), "c1", campaignapp.UpdateCampaignAIBindingInput{}); err != nil {
		t.Fatalf("UpdateCampaignAIBinding(clear) error = %v", err)
	}
	if campaignClient.lastClearAIBindingReq == nil || campaignClient.lastClearAIBindingReq.GetCampaignId() != "c1" {
		t.Fatalf("lastClearAIBindingReq = %#v", campaignClient.lastClearAIBindingReq)
	}

	campaignClient.setAIBindingErr = status.Error(codes.FailedPrecondition, "blocked")
	if err := gateway.UpdateCampaignAIBinding(context.Background(), "c1", campaignapp.UpdateCampaignAIBindingInput{AIAgentID: "agent-active"}); err == nil {
		t.Fatalf("expected conflict transport error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusConflict {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusConflict)
	}
}

func TestCreateCampaignMapsInputAndValidatesResponse(t *testing.T) {
	t.Parallel()

	client := &contractCampaignClient{}
	gateway := GRPCGateway{Mutation: GRPCGatewayMutationDeps{Campaign: client}}

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

func TestUpdateCampaignMapsInputAndErrors(t *testing.T) {
	t.Parallel()

	client := &contractCampaignClient{}
	gateway := GRPCGateway{Mutation: GRPCGatewayMutationDeps{Campaign: client}}

	name := "Campaign Prime"
	theme := "Updated theme"
	locale := "pt-BR"
	err := gateway.UpdateCampaign(context.Background(), "c1", campaignapp.UpdateCampaignInput{
		Name:        &name,
		ThemePrompt: &theme,
		Locale:      &locale,
	})
	if err != nil {
		t.Fatalf("UpdateCampaign() error = %v", err)
	}
	if client.lastUpdateReq == nil {
		t.Fatalf("expected UpdateCampaign request")
	}
	if client.lastUpdateReq.GetCampaignId() != "c1" {
		t.Fatalf("request campaign id = %q, want %q", client.lastUpdateReq.GetCampaignId(), "c1")
	}
	if got := strings.TrimSpace(client.lastUpdateReq.GetName().GetValue()); got != "Campaign Prime" {
		t.Fatalf("request name = %q, want %q", got, "Campaign Prime")
	}
	if got := strings.TrimSpace(client.lastUpdateReq.GetThemePrompt().GetValue()); got != "Updated theme" {
		t.Fatalf("request theme prompt = %q, want %q", got, "Updated theme")
	}
	if client.lastUpdateReq.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("request locale = %v, want %v", client.lastUpdateReq.GetLocale(), commonv1.Locale_LOCALE_PT_BR)
	}

	invalidLocale := "es-ES"
	if err := gateway.UpdateCampaign(context.Background(), "c1", campaignapp.UpdateCampaignInput{Locale: &invalidLocale}); err == nil {
		t.Fatalf("expected locale validation error")
	}

	client.updateErr = status.Error(codes.InvalidArgument, "invalid update")
	if err := gateway.UpdateCampaign(context.Background(), "c1", campaignapp.UpdateCampaignInput{}); err == nil {
		t.Fatalf("expected UpdateCampaign transport error")
	} else if got := apperrors.LocalizationKey(err); got != "error.web.message.failed_to_update_campaign" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.failed_to_update_campaign")
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
	gateway = GRPCGateway{
		Mutation: GRPCGatewayMutationDeps{
			Session: sessionClient,
			Invite:  inviteClient,
			Auth: &contractAuthClient{
				lookupResp: &authv1.LookupUserByUsernameResponse{
					User: &authv1.User{Id: "user-2", Username: "alice"},
				},
			},
		},
	}

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

	if err := gateway.CreateInvite(context.Background(), "c1", campaignapp.CreateInviteInput{ParticipantID: "p1", RecipientUsername: "alice"}); err == nil {
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

	gateway.Authorization = GRPCGatewayAuthorizationDeps{Client: contractAuthorizationClient{canResp: &statev1.CanResponse{
		Allowed:             true,
		ReasonCode:          "AUTHZ_ALLOW_RESOURCE_OWNER",
		ActorCampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
	}}}
	decision, err = gateway.CanCampaignAction(context.Background(), "c1", campaignapp.AuthorizationActionMutate, campaignapp.AuthorizationResourceCharacter, &campaignapp.AuthorizationTarget{ResourceID: "char-1"})
	if err != nil {
		t.Fatalf("CanCampaignAction() error = %v", err)
	}
	if !decision.Evaluated || !decision.Allowed || decision.ReasonCode != "AUTHZ_ALLOW_RESOURCE_OWNER" {
		t.Fatalf("decision = %#v", decision)
	}
	if decision.ActorCampaignAccess != "Owner" {
		t.Fatalf("decision.ActorCampaignAccess = %q, want %q", decision.ActorCampaignAccess, "Owner")
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
	if got := inviteStatusLabel(statev1.InviteStatus_DECLINED); got != "Declined" {
		t.Fatalf("inviteStatusLabel(declined) = %q", got)
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

func TestCreateInviteResolvesPlainUsernameBeforeGameRPC(t *testing.T) {
	t.Parallel()

	inviteClient := &contractInviteClient{}
	authClient := &contractAuthClient{
		lookupResp: &authv1.LookupUserByUsernameResponse{
			User: &authv1.User{Id: "user-2", Username: "alice"},
		},
	}
	gateway := GRPCGateway{Mutation: GRPCGatewayMutationDeps{Invite: inviteClient, Auth: authClient}}

	if err := gateway.CreateInvite(context.Background(), "c1", campaignapp.CreateInviteInput{
		ParticipantID:     "p1",
		RecipientUsername: "alice",
	}); err != nil {
		t.Fatalf("CreateInvite() error = %v", err)
	}
	if authClient.lastLookup == nil || authClient.lastLookup.GetUsername() != "alice" {
		t.Fatalf("lookup request = %#v, want username alice", authClient.lastLookup)
	}
	if inviteClient.lastCreate == nil {
		t.Fatal("expected create invite request to be sent")
	}
	if inviteClient.lastCreate.GetRecipientUserId() != "user-2" {
		t.Fatalf("RecipientUserId = %q, want %q", inviteClient.lastCreate.GetRecipientUserId(), "user-2")
	}
}

func TestCreateInviteMapsUnknownRecipientUsernameToValidationError(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{
		Mutation: GRPCGatewayMutationDeps{
			Invite: &contractInviteClient{},
			Auth:   &contractAuthClient{lookupErr: status.Error(codes.NotFound, "user not found")},
		},
	}

	err := gateway.CreateInvite(context.Background(), "c1", campaignapp.CreateInviteInput{
		ParticipantID:     "p1",
		RecipientUsername: "missing-user",
	})
	if err == nil {
		t.Fatal("expected CreateInvite error")
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.recipient_username_was_not_found" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.recipient_username_was_not_found")
	}
}

func TestCampaignInvitesHydratesRecipientUsernamesOncePerUniqueUser(t *testing.T) {
	t.Parallel()

	authClient := &contractAuthClient{
		getUserRespByID: map[string]*authv1.GetUserResponse{
			"user-2": {User: &authv1.User{Id: "user-2", Username: "river"}},
			"user-3": {User: &authv1.User{Id: "user-3", Username: "ember"}},
		},
	}
	gateway := inviteReadGateway{read: InviteReadDeps{
		Invite: &contractInviteClient{listResp: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{
			{Id: "inv-1", ParticipantId: "p1", RecipientUserId: "user-2", Status: statev1.InviteStatus_PENDING},
			{Id: "inv-2", ParticipantId: "p2", RecipientUserId: "user-2", Status: statev1.InviteStatus_PENDING},
			{Id: "inv-3", ParticipantId: "p3", RecipientUserId: "user-3", Status: statev1.InviteStatus_PENDING},
		}}},
		Participant: &contractParticipantClient{listResp: &statev1.ListParticipantsResponse{Participants: []*statev1.Participant{
			{Id: "p1", Name: "Seat One"},
			{Id: "p2", Name: "Seat Two"},
			{Id: "p3", Name: "Seat Three"},
		}}},
		Auth: authClient,
	}}

	invites, err := gateway.CampaignInvites(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignInvites() error = %v", err)
	}
	if len(invites) != 3 {
		t.Fatalf("len(invites) = %d, want 3", len(invites))
	}
	if invites[0].RecipientUsername != "river" || invites[1].RecipientUsername != "river" || invites[2].RecipientUsername != "ember" {
		t.Fatalf("recipient usernames = %#v", invites)
	}

	if got := authClient.getUserRequestUserIDs(); len(got) != 2 {
		t.Fatalf("GetUser request count = %d, want 2", len(got))
	} else {
		want := map[string]struct{}{"user-2": {}, "user-3": {}}
		for _, userID := range got {
			if _, ok := want[userID]; !ok {
				t.Fatalf("unexpected GetUser request for %q in %v", userID, got)
			}
			delete(want, userID)
		}
		if len(want) != 0 {
			t.Fatalf("missing GetUser requests for %v", want)
		}
	}
}

func TestCampaignInvitesFallsBackWhenRecipientLookupFails(t *testing.T) {
	t.Parallel()

	gateway := inviteReadGateway{read: InviteReadDeps{
		Invite: &contractInviteClient{listResp: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{
			{Id: "inv-1", ParticipantId: "p1", RecipientUserId: "user-2", Status: statev1.InviteStatus_PENDING},
		}}},
		Participant: &contractParticipantClient{listResp: &statev1.ListParticipantsResponse{Participants: []*statev1.Participant{
			{Id: "p1", Name: "Seat One"},
		}}},
		Auth: &contractAuthClient{
			getUserErrByID: map[string]error{"user-2": status.Error(codes.Unavailable, "boom")},
		},
	}}

	invites, err := gateway.CampaignInvites(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignInvites() error = %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("len(invites) = %d, want 1", len(invites))
	}
	if invites[0].RecipientUsername != "" {
		t.Fatalf("RecipientUsername = %q, want empty fallback", invites[0].RecipientUsername)
	}
	if !invites[0].HasRecipient {
		t.Fatalf("HasRecipient = false, want true")
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
	listResp         *statev1.ListCampaignsResponse
	listErr          error
	getResp          *statev1.GetCampaignResponse
	getErr           error
	readinessResp    *statev1.GetCampaignSessionReadinessResponse
	readinessErr     error
	lastReadinessReq *statev1.GetCampaignSessionReadinessRequest

	createResp            *statev1.CreateCampaignResponse
	createErr             error
	lastCreateReq         *statev1.CreateCampaignRequest
	updateResp            *statev1.UpdateCampaignResponse
	updateErr             error
	lastUpdateReq         *statev1.UpdateCampaignRequest
	setAIBindingResp      *statev1.SetCampaignAIBindingResponse
	setAIBindingErr       error
	lastSetAIBindingReq   *statev1.SetCampaignAIBindingRequest
	clearAIBindingResp    *statev1.ClearCampaignAIBindingResponse
	clearAIBindingErr     error
	lastClearAIBindingReq *statev1.ClearCampaignAIBindingRequest
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

func (c *contractCampaignClient) GetCampaignSessionReadiness(_ context.Context, req *statev1.GetCampaignSessionReadinessRequest, _ ...grpc.CallOption) (*statev1.GetCampaignSessionReadinessResponse, error) {
	c.lastReadinessReq = req
	if c.readinessErr != nil {
		return nil, c.readinessErr
	}
	if c.readinessResp != nil {
		return c.readinessResp, nil
	}
	return &statev1.GetCampaignSessionReadinessResponse{
		Readiness: &statev1.CampaignSessionReadiness{Ready: true},
	}, nil
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

func (c *contractCampaignClient) UpdateCampaign(_ context.Context, req *statev1.UpdateCampaignRequest, _ ...grpc.CallOption) (*statev1.UpdateCampaignResponse, error) {
	c.lastUpdateReq = req
	if c.updateErr != nil {
		return nil, c.updateErr
	}
	if c.updateResp != nil {
		return c.updateResp, nil
	}
	return &statev1.UpdateCampaignResponse{Campaign: &statev1.Campaign{Id: strings.TrimSpace(req.GetCampaignId())}}, nil
}

func (c *contractCampaignClient) SetCampaignAIBinding(_ context.Context, req *statev1.SetCampaignAIBindingRequest, _ ...grpc.CallOption) (*statev1.SetCampaignAIBindingResponse, error) {
	c.lastSetAIBindingReq = req
	if c.setAIBindingErr != nil {
		return nil, c.setAIBindingErr
	}
	if c.setAIBindingResp != nil {
		return c.setAIBindingResp, nil
	}
	return &statev1.SetCampaignAIBindingResponse{}, nil
}

func (c *contractCampaignClient) ClearCampaignAIBinding(_ context.Context, req *statev1.ClearCampaignAIBindingRequest, _ ...grpc.CallOption) (*statev1.ClearCampaignAIBindingResponse, error) {
	c.lastClearAIBindingReq = req
	if c.clearAIBindingErr != nil {
		return nil, c.clearAIBindingErr
	}
	if c.clearAIBindingResp != nil {
		return c.clearAIBindingResp, nil
	}
	return &statev1.ClearCampaignAIBindingResponse{}, nil
}

type contractParticipantClient struct {
	listResp  *statev1.ListParticipantsResponse
	listErr   error
	getResp   *statev1.GetParticipantResponse
	getErr    error
	createReq *statev1.CreateParticipantRequest
	createErr error
	updateReq *statev1.UpdateParticipantRequest
	updateErr error
}

type contractAgentClient struct {
	listResp    *aiv1.ListAgentsResponse
	listErr     error
	lastListReq *aiv1.ListAgentsRequest
}

func (c *contractAgentClient) ListAgents(_ context.Context, req *aiv1.ListAgentsRequest, _ ...grpc.CallOption) (*aiv1.ListAgentsResponse, error) {
	c.lastListReq = req
	if c.listErr != nil {
		return nil, c.listErr
	}
	if c.listResp != nil {
		return c.listResp, nil
	}
	return &aiv1.ListAgentsResponse{}, nil
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

func (c *contractParticipantClient) GetParticipant(context.Context, *statev1.GetParticipantRequest, ...grpc.CallOption) (*statev1.GetParticipantResponse, error) {
	if c.getErr != nil {
		return nil, c.getErr
	}
	if c.getResp != nil {
		return c.getResp, nil
	}
	return &statev1.GetParticipantResponse{}, nil
}

func (c *contractParticipantClient) CreateParticipant(_ context.Context, req *statev1.CreateParticipantRequest, _ ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
	c.createReq = req
	if c.createErr != nil {
		return nil, c.createErr
	}
	return &statev1.CreateParticipantResponse{Participant: &statev1.Participant{Id: "participant-created"}}, nil
}

func (c *contractParticipantClient) UpdateParticipant(_ context.Context, req *statev1.UpdateParticipantRequest, _ ...grpc.CallOption) (*statev1.UpdateParticipantResponse, error) {
	c.updateReq = req
	if c.updateErr != nil {
		return nil, c.updateErr
	}
	return &statev1.UpdateParticipantResponse{Participant: &statev1.Participant{Id: strings.TrimSpace(req.GetParticipantId())}}, nil
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
	listResp   *statev1.ListInvitesResponse
	listErr    error
	createErr  error
	claimErr   error
	declineErr error
	revokeErr  error
	lastCreate *statev1.CreateInviteRequest
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

func (c *contractInviteClient) GetPublicInvite(context.Context, *statev1.GetPublicInviteRequest, ...grpc.CallOption) (*statev1.GetPublicInviteResponse, error) {
	return &statev1.GetPublicInviteResponse{Invite: &statev1.Invite{}}, nil
}

func (c *contractInviteClient) CreateInvite(_ context.Context, req *statev1.CreateInviteRequest, _ ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
	c.lastCreate = req
	if c.createErr != nil {
		return nil, c.createErr
	}
	return &statev1.CreateInviteResponse{}, nil
}

func (c *contractInviteClient) ClaimInvite(context.Context, *statev1.ClaimInviteRequest, ...grpc.CallOption) (*statev1.ClaimInviteResponse, error) {
	if c.claimErr != nil {
		return nil, c.claimErr
	}
	return &statev1.ClaimInviteResponse{}, nil
}

func (c *contractInviteClient) DeclineInvite(context.Context, *statev1.DeclineInviteRequest, ...grpc.CallOption) (*statev1.DeclineInviteResponse, error) {
	if c.declineErr != nil {
		return nil, c.declineErr
	}
	return &statev1.DeclineInviteResponse{}, nil
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

type contractSocialClient struct {
	searchResp *socialv1.SearchUsersResponse
	searchErr  error
}

type contractAuthClient struct {
	mu              sync.Mutex
	lookupResp      *authv1.LookupUserByUsernameResponse
	lookupErr       error
	lastLookup      *authv1.LookupUserByUsernameRequest
	getUserRespByID map[string]*authv1.GetUserResponse
	getUserErrByID  map[string]error
	getUserReqs     []*authv1.GetUserRequest
}

func (c *contractAuthClient) LookupUserByUsername(_ context.Context, req *authv1.LookupUserByUsernameRequest, _ ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error) {
	c.lastLookup = req
	if c.lookupErr != nil {
		return nil, c.lookupErr
	}
	if c.lookupResp != nil {
		return c.lookupResp, nil
	}
	return &authv1.LookupUserByUsernameResponse{}, nil
}

func (c *contractAuthClient) GetUser(_ context.Context, req *authv1.GetUserRequest, _ ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	c.mu.Lock()
	c.getUserReqs = append(c.getUserReqs, req)
	var err error
	if c.getUserErrByID != nil {
		err = c.getUserErrByID[req.GetUserId()]
	}
	var resp *authv1.GetUserResponse
	if c.getUserRespByID != nil {
		resp = c.getUserRespByID[req.GetUserId()]
	}
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if resp != nil {
		return resp, nil
	}
	return &authv1.GetUserResponse{User: &authv1.User{}}, nil
}

func (c *contractAuthClient) IssueJoinGrant(context.Context, *authv1.IssueJoinGrantRequest, ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	return &authv1.IssueJoinGrantResponse{JoinGrant: "grant"}, nil
}

func (c *contractAuthClient) getUserRequestUserIDs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]string, 0, len(c.getUserReqs))
	for _, req := range c.getUserReqs {
		if req == nil {
			continue
		}
		result = append(result, req.GetUserId())
	}
	return result
}

func (c *contractSocialClient) SearchUsers(context.Context, *socialv1.SearchUsersRequest, ...grpc.CallOption) (*socialv1.SearchUsersResponse, error) {
	if c.searchErr != nil {
		return nil, c.searchErr
	}
	if c.searchResp != nil {
		return c.searchResp, nil
	}
	return &socialv1.SearchUsersResponse{}, nil
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

	gateway := GRPCGateway{Read: GRPCGatewayReadDeps{Campaign: &contractCampaignClient{listErr: errors.New("boom")}}}
	if _, err := gateway.ListCampaigns(context.Background()); err == nil {
		t.Fatalf("expected ListCampaigns() client error")
	}
}

func TestCampaignWorkspaceForwardsClientErrors(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{Read: GRPCGatewayReadDeps{Campaign: &contractCampaignClient{getErr: errors.New("boom")}}}
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
	if _, err := gateway.CampaignCharacters(context.Background(), "c1", campaignapp.CharacterReadContext{}); err == nil {
		t.Fatalf("expected missing character client error")
	}
}

func TestEmptyCampaignIDReturnsEmptyCollections(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{
		Read: GRPCGatewayReadDeps{
			Participant: &contractParticipantClient{},
			Character:   &fakeCharacterWorkflowClient{},
			Session:     &contractSessionClient{},
			Invite:      &contractInviteClient{},
		},
	}

	participants, err := gateway.CampaignParticipants(context.Background(), " ")
	if err != nil {
		t.Fatalf("CampaignParticipants() error = %v", err)
	}
	if len(participants) != 0 {
		t.Fatalf("len(participants) = %d, want 0", len(participants))
	}

	characters, err := gateway.CampaignCharacters(context.Background(), " ", campaignapp.CharacterReadContext{})
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

	gateway := GRPCGateway{Mutation: GRPCGatewayMutationDeps{Session: &contractSessionClient{}, Invite: &contractInviteClient{}}}

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

	gateway := GRPCGateway{Authorization: GRPCGatewayAuthorizationDeps{Client: contractAuthorizationClient{}}}
	decision, err := gateway.CanCampaignAction(context.Background(), "   ", campaignapp.AuthorizationActionMutate, campaignapp.AuthorizationResourceCharacter, nil)
	if err != nil {
		t.Fatalf("CanCampaignAction() error = %v", err)
	}
	if decision.Evaluated {
		t.Fatalf("expected unevaluated decision when campaign id is empty")
	}

	gateway.Authorization = GRPCGatewayAuthorizationDeps{Client: contractAuthorizationClient{nilCan: true}}
	decision, err = gateway.CanCampaignAction(context.Background(), "c1", campaignapp.AuthorizationActionMutate, campaignapp.AuthorizationResourceCharacter, nil)
	if err != nil {
		t.Fatalf("CanCampaignAction() error = %v", err)
	}
	if decision.Evaluated {
		t.Fatalf("expected unevaluated decision when response is nil")
	}

	gateway.Authorization = GRPCGatewayAuthorizationDeps{Client: contractAuthorizationClient{canErr: errors.New("boom")}}
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
	if got := mapCampaignAuthorizationResourceToProto(campaignapp.AuthorizationResourceCampaign); got != statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN {
		t.Fatalf("mapCampaignAuthorizationResourceToProto(campaign) = %v", got)
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
	target := mapCampaignAuthorizationTargetToProto(&campaignapp.AuthorizationTarget{
		ResourceID:              " part-1 ",
		TargetParticipantID:     " part-1 ",
		TargetCampaignAccess:    "owner",
		RequestedCampaignAccess: "manager",
		ParticipantOperation:    campaignapp.ParticipantGovernanceOperationAccessChange,
	})
	if target == nil {
		t.Fatalf("mapCampaignAuthorizationTargetToProto(participant governance) = nil")
	}
	if strings.TrimSpace(target.GetTargetParticipantId()) != "part-1" {
		t.Fatalf("target participant id = %q, want %q", target.GetTargetParticipantId(), "part-1")
	}
	if target.GetTargetCampaignAccess() != statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER {
		t.Fatalf("target campaign access = %v, want %v", target.GetTargetCampaignAccess(), statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER)
	}
	if target.GetRequestedCampaignAccess() != statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER {
		t.Fatalf("requested campaign access = %v, want %v", target.GetRequestedCampaignAccess(), statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER)
	}
	if target.GetParticipantOperation() != statev1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_ACCESS_CHANGE {
		t.Fatalf("participant operation = %v, want %v", target.GetParticipantOperation(), statev1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_ACCESS_CHANGE)
	}
}
