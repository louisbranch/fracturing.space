package web

import (
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
)

func defaultProtectedConfig(auth *fakeWebAuthClient) Config {
	account := &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{
		Profile: &authv1.AccountProfile{Username: "adventurer", Locale: commonv1.Locale_LOCALE_EN_US},
	}}
	social := defaultSocialClient()
	return Config{
		Dependencies: newDependencyBundle(
			PrincipalDependencies{
				SessionClient: auth,
				AccountClient: account,
				SocialClient:  social,
			},
			modules.Dependencies{
				PublicAuth: modules.PublicAuthDependencies{
					AuthClient: auth,
				},
				Campaigns: modules.CampaignDependencies{
					CampaignClient:           defaultCampaignClient(),
					ParticipantClient:        defaultParticipantClient(),
					CharacterClient:          defaultCharacterClient(),
					DaggerheartContentClient: defaultDaggerheartContentClient(),
					DaggerheartAssetClient:   defaultDaggerheartAssetClient(),
					SessionClient:            defaultSessionClient(),
					InviteClient:             defaultInviteClient(),
					AuthClient:               auth,
					AuthorizationClient:      defaultAuthorizationClient(),
				},
				Settings: modules.SettingsDependencies{
					SocialClient:     social,
					AccountClient:    account,
					PasskeyClient:    auth,
					CredentialClient: fakeCredentialClient{},
					AgentClient:      fakeAgentClient{},
				},
				Profile: modules.ProfileDependencies{
					AuthClient:   auth,
					SocialClient: social,
				},
			},
		),
	}
}

func defaultSocialClient() *fakeSocialClient {
	return &fakeSocialClient{getUserProfileResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{Name: "Adventurer"}}}
}

func defaultCampaignClient() fakeCampaignClient {
	return fakeCampaignClient{response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{{Id: "c1", Name: "Campaign"}}}}
}

func defaultParticipantClient() fakeWebParticipantClient {
	return fakeWebParticipantClient{response: &statev1.ListParticipantsResponse{Participants: []*statev1.Participant{{
		Id:             "p1",
		CampaignId:     "c1",
		UserId:         "user-1",
		Name:           "Owner",
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
	}}}}
}

func defaultCharacterClient() fakeWebCharacterClient {
	return fakeWebCharacterClient{response: &statev1.ListCharactersResponse{Characters: []*statev1.Character{{
		Id:   "char-1",
		Name: "Aria",
		Kind: statev1.CharacterKind_PC,
	}}}}
}

func defaultSessionClient() fakeWebSessionClient {
	return fakeWebSessionClient{response: &statev1.ListSessionsResponse{Sessions: []*statev1.Session{{
		Id:     "sess-1",
		Name:   "Session One",
		Status: statev1.SessionStatus_SESSION_ACTIVE,
	}}}}
}

func defaultInviteClient() fakeWebInviteClient {
	return fakeWebInviteClient{response: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{{
		Id:              "inv-1",
		CampaignId:      "c1",
		ParticipantId:   "p1",
		RecipientUserId: "user-2",
		Status:          statev1.InviteStatus_PENDING,
	}}}}
}

func defaultDaggerheartContentClient() fakeWebDaggerheartContentClient {
	return fakeWebDaggerheartContentClient{response: &daggerheartv1.GetDaggerheartContentCatalogResponse{Catalog: &daggerheartv1.DaggerheartContentCatalog{}}}
}

func defaultDaggerheartAssetClient() fakeWebDaggerheartAssetClient {
	return fakeWebDaggerheartAssetClient{response: &daggerheartv1.GetDaggerheartAssetMapResponse{AssetMap: &daggerheartv1.DaggerheartAssetMap{}}}
}

func defaultAuthorizationClient() fakeWebAuthorizationClient {
	return fakeWebAuthorizationClient{}
}
