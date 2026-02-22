package declarative

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestLoadManifest_LocalDevFixture(t *testing.T) {
	t.Parallel()

	path := filepath.Clean("../manifests/local-dev.json")
	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("load local-dev manifest: %v", err)
	}
	if manifest.Name != "local-dev" {
		t.Fatalf("manifest name = %q, want %q", manifest.Name, "local-dev")
	}
	if len(manifest.Users) < 3 {
		t.Fatalf("expected at least 3 users, got %d", len(manifest.Users))
	}
	if len(manifest.Campaigns) < 2 {
		t.Fatalf("expected at least 2 campaigns, got %d", len(manifest.Campaigns))
	}
}

func TestLoadManifest_RejectsAccountProfileField(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "manifest.json")
	content := `{
  "name": "local-dev",
  "users": [
    {
      "key": "alice",
      "email": "alice@example.com",
      "public_profile": {
        "username": "alice",
        "name": "Alice"
      },
      "account_profile": {
        "name": "deprecated"
      }
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("expected load failure for account_profile field")
	}
	if !strings.Contains(err.Error(), "account_profile") {
		t.Fatalf("error %q does not mention account_profile", err)
	}
}

func TestRunManifest_IdempotentSecondRun(t *testing.T) {
	t.Parallel()

	statePath := filepath.Join(t.TempDir(), "state.json")
	manifest := Manifest{
		Name: "local-dev",
		Users: []ManifestUser{
			{
				Key:    "alice",
				Email:  "alice@example.com",
				Locale: commonv1.Locale_LOCALE_EN_US.String(),
				PublicProfile: ManifestPublicProfile{
					Username: "alice",
					Name:     "Alice",
					Bio:      "GM of the local seed campaign.",
				},
			},
		},
		Campaigns: []ManifestCampaign{
			{
				Key:           "crimson_vale",
				OwnerUserKey:  "alice",
				Name:          "The Crimson Vale",
				GmMode:        gamev1.GmMode_HUMAN.String(),
				Intent:        gamev1.CampaignIntent_STARTER.String(),
				AccessPolicy:  gamev1.CampaignAccessPolicy_PUBLIC.String(),
				ThemePrompt:   "Seeded campaign for local development.",
				Participants:  []ManifestParticipant{},
				Characters:    []ManifestCharacter{},
				Sessions:      []ManifestSession{},
				Listing:       nil,
				ForkFrom:      "",
				ForkEventSeq:  0,
				ForkSessionID: "",
			},
		},
		Listings: []ManifestListing{
			{
				CampaignKey:                "crimson_vale",
				Title:                      "The Crimson Vale",
				Description:                "Starter listing for local dev.",
				RecommendedParticipantsMin: 3,
				RecommendedParticipantsMax: 5,
				DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER.String(),
				ExpectedDurationLabel:      "2-3 sessions",
				System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
			},
		},
	}

	deps := newFakeDeps()
	runner := newRunnerWithClients(Config{
		ManifestPath: "ignored.json",
		StatePath:    statePath,
		Verbose:      true,
	}, runnerDeps{
		auth:        deps.auth,
		connections: deps.connections,
		campaigns:   deps.game,
		listings:    deps.listing,
	})

	if err := runner.RunManifest(context.Background(), manifest); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := runner.RunManifest(context.Background(), manifest); err != nil {
		t.Fatalf("second run: %v", err)
	}

	if deps.auth.createUserCalls != 1 {
		t.Fatalf("create user calls = %d, want 1", deps.auth.createUserCalls)
	}
	if deps.game.createCampaignCalls != 1 {
		t.Fatalf("create campaign calls = %d, want 1", deps.game.createCampaignCalls)
	}
	if deps.listing.createCalls != 1 {
		t.Fatalf("create listing calls = %d, want 1", deps.listing.createCalls)
	}
}

func TestValidateManifest_RejectsMissingReferences(t *testing.T) {
	t.Parallel()

	manifest := Manifest{
		Name: "invalid",
		Users: []ManifestUser{
			{
				Key:   "alice",
				Email: "alice@example.com",
			},
		},
		Campaigns: []ManifestCampaign{
			{
				Key:          "broken-campaign",
				OwnerUserKey: "missing-user",
				Name:         "Broken",
				GmMode:       gamev1.GmMode_HUMAN.String(),
			},
		},
	}

	err := ValidateManifest(manifest)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "missing-user") {
		t.Fatalf("validation error %q does not mention missing owner reference", err)
	}
}

type fakeDeps struct {
	auth        *fakeAuthClient
	connections *fakeConnectionsClient
	game        *fakeGameClient
	listing     *fakeListingClient
}

func newFakeDeps() fakeDeps {
	return fakeDeps{
		auth:        &fakeAuthClient{},
		connections: &fakeConnectionsClient{},
		game:        &fakeGameClient{},
		listing:     &fakeListingClient{},
	}
}

type fakeAuthClient struct {
	createUserCalls int
	usersByID       map[string]*authv1.User
	idsByEmail      map[string]string
}

func (f *fakeAuthClient) ensure() {
	if f.usersByID == nil {
		f.usersByID = map[string]*authv1.User{}
	}
	if f.idsByEmail == nil {
		f.idsByEmail = map[string]string{}
	}
}

func (f *fakeAuthClient) CreateUser(_ context.Context, in *authv1.CreateUserRequest, _ ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	f.ensure()
	email := strings.ToLower(strings.TrimSpace(in.GetEmail()))
	if existingID, ok := f.idsByEmail[email]; ok {
		return &authv1.CreateUserResponse{User: f.usersByID[existingID]}, nil
	}
	f.createUserCalls++
	id := "user-" + strings.TrimPrefix(email, "alice@")
	user := &authv1.User{
		Id:        id,
		Email:     email,
		Locale:    in.GetLocale(),
		CreatedAt: timestamppb.Now(),
		UpdatedAt: timestamppb.Now(),
	}
	f.idsByEmail[email] = id
	f.usersByID[id] = user
	return &authv1.CreateUserResponse{User: user}, nil
}

func (f *fakeAuthClient) GetUser(_ context.Context, in *authv1.GetUserRequest, _ ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	f.ensure()
	user, ok := f.usersByID[in.GetUserId()]
	if !ok {
		return &authv1.GetUserResponse{}, nil
	}
	return &authv1.GetUserResponse{User: user}, nil
}

func (f *fakeAuthClient) ListUsers(_ context.Context, _ *authv1.ListUsersRequest, _ ...grpc.CallOption) (*authv1.ListUsersResponse, error) {
	f.ensure()
	users := make([]*authv1.User, 0, len(f.usersByID))
	for _, u := range f.usersByID {
		users = append(users, u)
	}
	return &authv1.ListUsersResponse{Users: users}, nil
}

func (f *fakeAuthClient) ListUserEmails(_ context.Context, in *authv1.ListUserEmailsRequest, _ ...grpc.CallOption) (*authv1.ListUserEmailsResponse, error) {
	f.ensure()
	user, ok := f.usersByID[in.GetUserId()]
	if !ok {
		return &authv1.ListUserEmailsResponse{}, nil
	}
	return &authv1.ListUserEmailsResponse{
		Emails: []*authv1.UserEmail{{Email: user.GetEmail()}},
	}, nil
}

type fakeConnectionsClient struct {
	setProfileCalls int
	addContactCalls int
}

func (f *fakeConnectionsClient) SetUserProfile(_ context.Context, _ *connectionsv1.SetUserProfileRequest, _ ...grpc.CallOption) (*connectionsv1.SetUserProfileResponse, error) {
	f.setProfileCalls++
	return &connectionsv1.SetUserProfileResponse{}, nil
}

func (f *fakeConnectionsClient) AddContact(_ context.Context, _ *connectionsv1.AddContactRequest, _ ...grpc.CallOption) (*connectionsv1.AddContactResponse, error) {
	f.addContactCalls++
	return &connectionsv1.AddContactResponse{}, nil
}

type fakeGameClient struct {
	createCampaignCalls int
	campaignsByID       map[string]*gamev1.Campaign
	campaignIDByKey     map[string]string
}

func (f *fakeGameClient) ensure() {
	if f.campaignsByID == nil {
		f.campaignsByID = map[string]*gamev1.Campaign{}
	}
	if f.campaignIDByKey == nil {
		f.campaignIDByKey = map[string]string{}
	}
}

func (f *fakeGameClient) CreateCampaign(_ context.Context, in *gamev1.CreateCampaignRequest, _ ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error) {
	f.ensure()
	theme := strings.TrimSpace(in.GetThemePrompt())
	if existingID, ok := f.campaignIDByKey[theme]; ok {
		return &gamev1.CreateCampaignResponse{
			Campaign: f.campaignsByID[existingID],
			OwnerParticipant: &gamev1.Participant{
				Id: "owner-" + existingID,
			},
		}, nil
	}
	f.createCampaignCalls++
	id := "camp-" + strings.ReplaceAll(strings.ToLower(in.GetName()), " ", "-")
	campaign := &gamev1.Campaign{
		Id:           id,
		Name:         in.GetName(),
		GmMode:       in.GetGmMode(),
		Intent:       in.GetIntent(),
		AccessPolicy: in.GetAccessPolicy(),
		System:       in.GetSystem(),
		ThemePrompt:  in.GetThemePrompt(),
		Status:       gamev1.CampaignStatus_DRAFT,
		CreatedAt:    timestamppb.Now(),
		UpdatedAt:    timestamppb.Now(),
	}
	f.campaignIDByKey[theme] = id
	f.campaignsByID[id] = campaign
	return &gamev1.CreateCampaignResponse{
		Campaign: campaign,
		OwnerParticipant: &gamev1.Participant{
			Id: "owner-" + id,
		},
	}, nil
}

func (f *fakeGameClient) GetCampaign(_ context.Context, in *gamev1.GetCampaignRequest, _ ...grpc.CallOption) (*gamev1.GetCampaignResponse, error) {
	f.ensure()
	campaign, ok := f.campaignsByID[in.GetCampaignId()]
	if !ok {
		return &gamev1.GetCampaignResponse{}, nil
	}
	return &gamev1.GetCampaignResponse{Campaign: campaign}, nil
}

func (f *fakeGameClient) ListCampaigns(_ context.Context, _ *gamev1.ListCampaignsRequest, _ ...grpc.CallOption) (*gamev1.ListCampaignsResponse, error) {
	f.ensure()
	campaigns := make([]*gamev1.Campaign, 0, len(f.campaignsByID))
	for _, c := range f.campaignsByID {
		campaigns = append(campaigns, c)
	}
	return &gamev1.ListCampaignsResponse{Campaigns: campaigns}, nil
}

type fakeListingClient struct {
	createCalls int
	listingByID map[string]*listingv1.CampaignListing
}

func (f *fakeListingClient) ensure() {
	if f.listingByID == nil {
		f.listingByID = map[string]*listingv1.CampaignListing{}
	}
}

func (f *fakeListingClient) CreateCampaignListing(_ context.Context, in *listingv1.CreateCampaignListingRequest, _ ...grpc.CallOption) (*listingv1.CreateCampaignListingResponse, error) {
	f.ensure()
	if listing, ok := f.listingByID[in.GetCampaignId()]; ok {
		return &listingv1.CreateCampaignListingResponse{Listing: listing}, nil
	}
	f.createCalls++
	listing := &listingv1.CampaignListing{
		CampaignId:                 in.GetCampaignId(),
		Title:                      in.GetTitle(),
		Description:                in.GetDescription(),
		RecommendedParticipantsMin: in.GetRecommendedParticipantsMin(),
		RecommendedParticipantsMax: in.GetRecommendedParticipantsMax(),
		DifficultyTier:             in.GetDifficultyTier(),
		ExpectedDurationLabel:      in.GetExpectedDurationLabel(),
		System:                     in.GetSystem(),
		CreatedAt:                  timestamppb.Now(),
		UpdatedAt:                  timestamppb.Now(),
	}
	f.listingByID[in.GetCampaignId()] = listing
	return &listingv1.CreateCampaignListingResponse{Listing: listing}, nil
}

func (f *fakeListingClient) GetCampaignListing(_ context.Context, in *listingv1.GetCampaignListingRequest, _ ...grpc.CallOption) (*listingv1.GetCampaignListingResponse, error) {
	f.ensure()
	listing, ok := f.listingByID[in.GetCampaignId()]
	if !ok {
		return &listingv1.GetCampaignListingResponse{}, nil
	}
	return &listingv1.GetCampaignListingResponse{Listing: listing}, nil
}
