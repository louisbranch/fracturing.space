package web

import (
	"context"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	"google.golang.org/grpc"
)

func TestNewDependencyBundleKeepsPartialModuleDependenciesExplicit(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	bundle := newDependencyBundle(
		principal.Dependencies{SessionClient: auth},
		modules.Dependencies{
			Campaigns: modules.CampaignDependencies{
				CampaignClient: defaultCampaignClient(),
			},
		},
	)
	if bundle == nil {
		t.Fatalf("expected non-nil dependency bundle")
	}
	if bundle.Modules.Campaigns.DiscoveryClient != nil {
		t.Fatalf("newDependencyBundle() unexpectedly completed discovery dependency")
	}
	if bundle.Modules.Campaigns.InviteClient != nil {
		t.Fatalf("newDependencyBundle() unexpectedly completed invite dependency")
	}
}

func defaultProtectedConfig(auth *fakeWebAuthClient) Config {
	account := &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{
		Profile: &authv1.AccountProfile{Username: "adventurer", Locale: commonv1.Locale_LOCALE_EN_US},
	}}
	notifications := fakeWebNotificationClient{}
	social := defaultSocialClient()
	return Config{
		PlayLaunchGrant: fakePlayLaunchGrantConfig(),
		Dependencies: newDependencyBundle(
			principal.Dependencies{
				SessionClient:      auth,
				AccountClient:      account,
				SocialClient:       social,
				NotificationClient: notifications,
			},
			modules.Dependencies{
				PublicAuth: modules.PublicAuthDependencies{
					AuthClient: auth,
				},
				Campaigns: modules.CampaignDependencies{
					CampaignClient:           defaultCampaignClient(),
					DiscoveryClient:          defaultDiscoveryClient(),
					AgentClient:              defaultAgentClient(),
					CampaignArtifactClient:   defaultCampaignArtifactClient(),
					ParticipantClient:        defaultParticipantClient(),
					CharacterClient:          defaultCharacterClient(),
					DaggerheartContentClient: defaultDaggerheartContentClient(),
					DaggerheartAssetClient:   defaultDaggerheartAssetClient(),
					SessionClient:            defaultSessionClient(),
					InviteClient:             defaultInviteClient(),
					SocialClient:             social,
					AuthClient:               auth,
					AuthorizationClient:      defaultAuthorizationClient(),
					ForkClient:               defaultForkClient(),
				},
				Invite: modules.InviteDependencies{
					InviteClient: defaultInviteClient(),
					AuthClient:   auth,
				},
				Settings: modules.SettingsDependencies{
					SocialClient:     social,
					AccountClient:    account,
					PasskeyClient:    auth,
					CredentialClient: fakeCredentialClient{},
					AgentClient:      fakeAgentClient{},
				},
				Notifications: modules.NotificationDependencies{
					NotificationClient: notifications,
				},
				Profile: modules.ProfileDependencies{
					AuthClient:   auth,
					SocialClient: social,
				},
			},
		),
	}
}

func fakePlayLaunchGrantConfig() playlaunchgrant.Config {
	return playlaunchgrant.Config{
		Issuer:   "fracturing-space-web",
		Audience: "fracturing-space-play",
		HMACKey:  []byte("0123456789abcdef0123456789abcdef"),
		TTL:      2 * time.Minute,
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
	return fakeWebInviteClient{response: &invitev1.ListInvitesResponse{Invites: []*invitev1.Invite{{
		Id:              "inv-1",
		CampaignId:      "c1",
		ParticipantId:   "p1",
		RecipientUserId: "user-2",
		Status:          invitev1.InviteStatus_PENDING,
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

func defaultDiscoveryClient() fakeWebDiscoveryClient {
	return fakeWebDiscoveryClient{}
}

func defaultAgentClient() fakeAgentClient {
	return fakeAgentClient{}
}

func defaultCampaignArtifactClient() fakeCampaignArtifactClient {
	return fakeCampaignArtifactClient{}
}

func defaultForkClient() fakeWebForkClient {
	return fakeWebForkClient{}
}

func (fakeWebDiscoveryClient) ListDiscoveryEntries(_ context.Context, _ *discoveryv1.ListDiscoveryEntriesRequest, _ ...grpc.CallOption) (*discoveryv1.ListDiscoveryEntriesResponse, error) {
	return &discoveryv1.ListDiscoveryEntriesResponse{}, nil
}

type fakeWebDiscoveryClient struct {
	discoveryv1.DiscoveryServiceClient
}

type fakeCampaignArtifactClient struct {
	aiv1.CampaignArtifactServiceClient
}

type fakeWebForkClient struct {
	statev1.ForkServiceClient
}
