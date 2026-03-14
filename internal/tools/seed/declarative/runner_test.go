package declarative

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
				Key:            "crimson_vale",
				OwnerUserKey:   "alice",
				Name:           "The Crimson Vale",
				GmMode:         gamev1.GmMode_HUMAN.String(),
				Intent:         gamev1.CampaignIntent_STARTER.String(),
				AccessPolicy:   gamev1.CampaignAccessPolicy_PUBLIC.String(),
				ThemePrompt:    "Seeded campaign for local development.",
				Participants:   []ManifestParticipant{},
				Characters:     []ManifestCharacter{},
				Sessions:       []ManifestSession{},
				DiscoveryEntry: nil,
				ForkFrom:       "",
				ForkEventSeq:   0,
				ForkSessionID:  "",
			},
		},
		DiscoveryEntries: []ManifestDiscoveryEntry{
			{
				CampaignKey:                "crimson_vale",
				Title:                      "The Crimson Vale",
				Description:                "Starter discovery entry for local dev.",
				RecommendedParticipantsMin: 3,
				RecommendedParticipantsMax: 5,
				DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER.String(),
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
		auth:      deps.auth,
		social:    deps.social,
		campaigns: deps.game,
		discovery: deps.discovery,
	})

	if err := runner.RunManifest(context.Background(), manifest); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := runner.RunManifest(context.Background(), manifest); err != nil {
		t.Fatalf("second run: %v", err)
	}

	if deps.auth.lookupUserByUsernameCalls != 1 {
		t.Fatalf("lookup user calls = %d, want 1", deps.auth.lookupUserByUsernameCalls)
	}
	if deps.game.createCampaignCalls != 1 {
		t.Fatalf("create campaign calls = %d, want 1", deps.game.createCampaignCalls)
	}
	if deps.discovery.createCalls != 1 {
		t.Fatalf("create discovery entry calls = %d, want 1", deps.discovery.createCalls)
	}
}

func TestRunManifest_IdempotentSecondRunPreservesStateEntries(t *testing.T) {
	t.Parallel()

	manifest := Manifest{
		Name: "local-dev",
		Users: []ManifestUser{
			{
				Key:   "alice",
				Email: "alice@example.com",
				PublicProfile: ManifestPublicProfile{
					Username: "alice",
					Name:     "Alice",
				},
			},
		},
		Campaigns: []ManifestCampaign{
			{
				Key:          "crimson_vale",
				OwnerUserKey: "alice",
				Name:         "The Crimson Vale",
			},
		},
	}

	stateStore := &fakeStateStore{state: defaultState()}
	deps := newFakeDeps()
	runner := newRunnerWithClients(Config{ManifestPath: "ignored.json"}, runnerDeps{
		auth:       deps.auth,
		social:     deps.social,
		campaigns:  deps.game,
		discovery:  deps.discovery,
		stateStore: stateStore,
	})

	if err := runner.RunManifest(context.Background(), manifest); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := runner.RunManifest(context.Background(), manifest); err != nil {
		t.Fatalf("second run: %v", err)
	}

	if deps.auth.lookupUserByUsernameCalls != 1 {
		t.Fatalf("lookup user calls = %d, want 1", deps.auth.lookupUserByUsernameCalls)
	}
	if deps.game.createCampaignCalls != 1 {
		t.Fatalf("create campaign calls = %d, want 1", deps.game.createCampaignCalls)
	}
	if stateStore.loadCalls != 2 {
		t.Fatalf("state load calls = %d, want 2", stateStore.loadCalls)
	}
	if stateStore.saveCalls != 2 {
		t.Fatalf("state save calls = %d, want 2", stateStore.saveCalls)
	}
	if len(stateStore.savedStates) != 2 {
		t.Fatalf("saved states = %d, want 2", len(stateStore.savedStates))
	}
	firstEntries := stateStore.savedStates[0].Entries
	secondEntries := stateStore.savedStates[1].Entries
	if !reflect.DeepEqual(firstEntries, secondEntries) {
		t.Fatalf("state entries changed between idempotent runs:\nfirst=%v\nsecond=%v", firstEntries, secondEntries)
	}
}

func TestRunnerApplyDiscoveryEntries_EntryNotFoundCreatesNewEntry(t *testing.T) {
	t.Parallel()

	discoveryClient := &fakeDiscoveryClient{
		getDiscoveryEntry: func(_ context.Context, in *discoveryv1.GetDiscoveryEntryRequest, _ ...grpc.CallOption) (*discoveryv1.GetDiscoveryEntryResponse, error) {
			if in.GetEntryId() == "camp-1" {
				return nil, status.Error(codes.NotFound, "missing discovery entry")
			}
			return &discoveryv1.GetDiscoveryEntryResponse{}, nil
		},
	}

	runner := newRunnerWithClients(Config{ManifestPath: "local"}, runnerDeps{discovery: discoveryClient})

	err := runner.applyDiscoveryEntries(context.Background(), Manifest{
		Name: "local",
		DiscoveryEntries: []ManifestDiscoveryEntry{{
			CampaignKey:    "crimson",
			Title:          "Campaign discovery entry",
			System:         commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
			DifficultyTier: discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER.String(),
		}},
	}, map[string]string{"crimson": "camp-1"})
	if err != nil {
		t.Fatalf("apply discovery entries: %v", err)
	}
	if discoveryClient.createCalls != 1 {
		t.Fatalf("create discovery entry calls = %d, want 1", discoveryClient.createCalls)
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

func TestRunManifest_RejectsUnsupportedGmMode(t *testing.T) {
	t.Parallel()

	manifest := Manifest{
		Name: "invalid-gm-mode",
		Users: []ManifestUser{
			{
				Key:   "alice",
				Email: "alice@example.com",
				PublicProfile: ManifestPublicProfile{
					Username: "alice",
					Name:     "Alice",
				},
			},
		},
		Campaigns: []ManifestCampaign{
			{
				Key:          "campaign-1",
				OwnerUserKey: "alice",
				Name:         "Broken Campaign",
				GmMode:       "BOGUS",
			},
		},
	}

	deps := newFakeDeps()
	runner := newRunnerWithClients(Config{ManifestPath: "ignored"}, runnerDeps{
		auth:      deps.auth,
		social:    deps.social,
		campaigns: deps.game,
		discovery: deps.discovery,
	})

	err := runner.RunManifest(context.Background(), manifest)
	if err == nil {
		t.Fatal("expected run failure for unsupported gm_mode")
	}
	if !strings.Contains(err.Error(), "unsupported gm_mode") {
		t.Fatalf("error %q does not mention unsupported gm_mode", err)
	}
	if deps.game.createCampaignCalls != 0 {
		t.Fatalf("create campaign calls = %d, want 0", deps.game.createCampaignCalls)
	}
}

type fakeDeps struct {
	auth      *fakeAuthClient
	social    *fakeSocialClient
	game      *fakeGameClient
	discovery *fakeDiscoveryClient
}

func newFakeDeps() fakeDeps {
	return fakeDeps{
		auth:      &fakeAuthClient{},
		social:    &fakeSocialClient{},
		game:      &fakeGameClient{},
		discovery: &fakeDiscoveryClient{},
	}
}

type fakeAuthClient struct {
	lookupUserByUsernameCalls int
	usersByID                 map[string]*authv1.User
	idsByUsername             map[string]string
}

func (f *fakeAuthClient) ensure() {
	if f.usersByID == nil {
		f.usersByID = map[string]*authv1.User{}
	}
	if f.idsByUsername == nil {
		f.idsByUsername = map[string]string{}
	}
}

func (f *fakeAuthClient) LookupUserByUsername(_ context.Context, in *authv1.LookupUserByUsernameRequest, _ ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error) {
	f.ensure()
	username := strings.TrimSpace(in.GetUsername())
	if existingID, ok := f.idsByUsername[username]; ok {
		return &authv1.LookupUserByUsernameResponse{User: f.usersByID[existingID]}, nil
	}
	f.lookupUserByUsernameCalls++
	id := "user-" + username
	user := &authv1.User{
		Id:        id,
		Username:  username,
		Locale:    commonv1.Locale_LOCALE_EN_US,
		CreatedAt: timestamppb.Now(),
		UpdatedAt: timestamppb.Now(),
	}
	f.idsByUsername[username] = id
	f.usersByID[id] = user
	return &authv1.LookupUserByUsernameResponse{User: user}, nil
}

func (f *fakeAuthClient) GetUser(_ context.Context, in *authv1.GetUserRequest, _ ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	f.ensure()
	user, ok := f.usersByID[in.GetUserId()]
	if !ok {
		return &authv1.GetUserResponse{}, nil
	}
	return &authv1.GetUserResponse{User: user}, nil
}

type fakeSocialClient struct {
	setProfileCalls int
	addContactCalls int
}

func (f *fakeSocialClient) SetUserProfile(_ context.Context, _ *socialv1.SetUserProfileRequest, _ ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error) {
	f.setProfileCalls++
	return &socialv1.SetUserProfileResponse{}, nil
}

func (f *fakeSocialClient) AddContact(_ context.Context, _ *socialv1.AddContactRequest, _ ...grpc.CallOption) (*socialv1.AddContactResponse, error) {
	f.addContactCalls++
	return &socialv1.AddContactResponse{}, nil
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

type fakeDiscoveryClient struct {
	getDiscoveryEntry    func(context.Context, *discoveryv1.GetDiscoveryEntryRequest, ...grpc.CallOption) (*discoveryv1.GetDiscoveryEntryResponse, error)
	createCalls          int
	createDiscoveryEntry func(context.Context, *discoveryv1.CreateDiscoveryEntryRequest, ...grpc.CallOption) (*discoveryv1.CreateDiscoveryEntryResponse, error)
	discoveryEntryByID   map[string]*discoveryv1.DiscoveryEntry
}

func (f *fakeDiscoveryClient) ensure() {
	if f.discoveryEntryByID == nil {
		f.discoveryEntryByID = map[string]*discoveryv1.DiscoveryEntry{}
	}
}

func (f *fakeDiscoveryClient) CreateDiscoveryEntry(ctx context.Context, in *discoveryv1.CreateDiscoveryEntryRequest, _ ...grpc.CallOption) (*discoveryv1.CreateDiscoveryEntryResponse, error) {
	if f.createDiscoveryEntry != nil {
		return f.createDiscoveryEntry(ctx, in)
	}
	f.ensure()
	entry := in.GetEntry()
	if entry == nil {
		return &discoveryv1.CreateDiscoveryEntryResponse{}, nil
	}
	if existing, ok := f.discoveryEntryByID[entry.GetEntryId()]; ok {
		return &discoveryv1.CreateDiscoveryEntryResponse{Entry: existing}, nil
	}
	f.createCalls++
	discoveryEntry := &discoveryv1.DiscoveryEntry{
		EntryId:                    entry.GetEntryId(),
		Kind:                       entry.GetKind(),
		SourceId:                   entry.GetSourceId(),
		Title:                      entry.GetTitle(),
		Description:                entry.GetDescription(),
		RecommendedParticipantsMin: entry.GetRecommendedParticipantsMin(),
		RecommendedParticipantsMax: entry.GetRecommendedParticipantsMax(),
		DifficultyTier:             entry.GetDifficultyTier(),
		ExpectedDurationLabel:      entry.GetExpectedDurationLabel(),
		System:                     entry.GetSystem(),
		GmMode:                     entry.GetGmMode(),
		Intent:                     entry.GetIntent(),
		Level:                      entry.GetLevel(),
		CharacterCount:             entry.GetCharacterCount(),
		Storyline:                  entry.GetStoryline(),
		Tags:                       append([]string(nil), entry.GetTags()...),
		CreatedAt:                  timestamppb.Now(),
		UpdatedAt:                  timestamppb.Now(),
	}
	f.discoveryEntryByID[entry.GetEntryId()] = discoveryEntry
	return &discoveryv1.CreateDiscoveryEntryResponse{Entry: discoveryEntry}, nil
}

func (f *fakeDiscoveryClient) GetDiscoveryEntry(ctx context.Context, in *discoveryv1.GetDiscoveryEntryRequest, _ ...grpc.CallOption) (*discoveryv1.GetDiscoveryEntryResponse, error) {
	if f.getDiscoveryEntry != nil {
		return f.getDiscoveryEntry(ctx, in)
	}
	f.ensure()
	discoveryEntry, ok := f.discoveryEntryByID[in.GetEntryId()]
	if !ok {
		return &discoveryv1.GetDiscoveryEntryResponse{}, nil
	}
	return &discoveryv1.GetDiscoveryEntryResponse{Entry: discoveryEntry}, nil
}

type fakeStateStore struct {
	state       seedState
	loadCalls   int
	saveCalls   int
	savedStates []seedState
}

func (f *fakeStateStore) Load(_ string) (seedState, error) {
	f.loadCalls++
	return cloneSeedState(f.state), nil
}

func (f *fakeStateStore) Save(_ string, state seedState) error {
	f.saveCalls++
	clone := cloneSeedState(state)
	f.state = clone
	f.savedStates = append(f.savedStates, clone)
	return nil
}

func cloneSeedState(state seedState) seedState {
	cloned := seedState{
		Version:   state.Version,
		UpdatedAt: state.UpdatedAt,
		Entries:   map[string]string{},
	}
	for key, value := range state.Entries {
		cloned.Entries[key] = value
	}
	return cloned
}
