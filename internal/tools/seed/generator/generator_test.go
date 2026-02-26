package generator

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
)

func TestNewSeededRNGDeterministic(t *testing.T) {
	first := NewSeededRNG(42, false)
	second := NewSeededRNG(42, false)

	if first.Int63() != second.Int63() {
		t.Fatal("expected deterministic RNG for same seed")
	}
	if first.Int63() != second.Int63() {
		t.Fatal("expected deterministic RNG sequence for same seed")
	}
}

func TestGeneratorRandomRangeMinGreaterThanMax(t *testing.T) {
	gen := &Generator{rng: rand.New(rand.NewSource(1))}
	if got := gen.randomRange(5, 3); got != 5 {
		t.Fatalf("expected min when min >= max, got %d", got)
	}
}

func TestGeneratorRandomRangeInclusive(t *testing.T) {
	gen := &Generator{rng: rand.New(rand.NewSource(2))}
	for i := 0; i < 10; i++ {
		value := gen.randomRange(2, 4)
		if value < 2 || value > 4 {
			t.Fatalf("value %d out of range", value)
		}
	}
}

func TestGeneratorGameSystem(t *testing.T) {
	gen := &Generator{rng: rand.New(rand.NewSource(3))}
	if got := gen.gameSystem(); got != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("expected daggerheart system, got %v", got)
	}
}

// fullFakeDeps returns a generatorDeps wired to happy-path fakes for all services.
// Suitable for Run()-level integration tests.
func fullFakeDeps() generatorDeps {
	partSeq := 0
	charSeq := 0
	sessSeq := 0
	return generatorDeps{
		campaigns: &fakeCampaignCreator{
			createCampaign: func(_ context.Context, in *statev1.CreateCampaignRequest, _ ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
				return &statev1.CreateCampaignResponse{
					Campaign:         &statev1.Campaign{Id: "camp-1", Name: in.Name},
					OwnerParticipant: &statev1.Participant{Id: "owner-1"},
				}, nil
			},
			endCampaign: func(context.Context, *statev1.EndCampaignRequest, ...grpc.CallOption) (*statev1.EndCampaignResponse, error) {
				return &statev1.EndCampaignResponse{}, nil
			},
			archiveCampaign: func(context.Context, *statev1.ArchiveCampaignRequest, ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error) {
				return &statev1.ArchiveCampaignResponse{}, nil
			},
		},
		participants: &fakeParticipantCreator{
			create: func(_ context.Context, in *statev1.CreateParticipantRequest, _ ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
				partSeq++
				return &statev1.CreateParticipantResponse{
					Participant: &statev1.Participant{
						Id:         fmt.Sprintf("p-%d", partSeq),
						Role:       in.Role,
						Controller: in.Controller,
					},
				}, nil
			},
		},
		invites: &fakeInviteManager{
			createInvite: func(context.Context, *statev1.CreateInviteRequest, ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
				return &statev1.CreateInviteResponse{Invite: &statev1.Invite{Id: "inv-1"}}, nil
			},
			claimInvite: func(context.Context, *statev1.ClaimInviteRequest, ...grpc.CallOption) (*statev1.ClaimInviteResponse, error) {
				return &statev1.ClaimInviteResponse{}, nil
			},
		},
		characters: &fakeCharacterCreator{
			create: func(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
				charSeq++
				return &statev1.CreateCharacterResponse{
					Character: &statev1.Character{Id: fmt.Sprintf("char-%d", charSeq)},
				}, nil
			},
			setDefaultControl: func(context.Context, *statev1.SetDefaultControlRequest, ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error) {
				return &statev1.SetDefaultControlResponse{}, nil
			},
		},
		sessions: &fakeSessionManager{
			startSession: func(context.Context, *statev1.StartSessionRequest, ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
				sessSeq++
				return &statev1.StartSessionResponse{
					Session: &statev1.Session{Id: fmt.Sprintf("s-%d", sessSeq)},
				}, nil
			},
			endSession: func(context.Context, *statev1.EndSessionRequest, ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
				return &statev1.EndSessionResponse{}, nil
			},
			listSessions: func(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
				return &statev1.ListSessionsResponse{}, nil
			},
		},
		events: &fakeEventAppender{
			appendEvent: func(context.Context, *statev1.AppendEventRequest, ...grpc.CallOption) (*statev1.AppendEventResponse, error) {
				return &statev1.AppendEventResponse{}, nil
			},
		},
		authClient: &fakeAuthProvider{
			createUser: func(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
				return &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}}, nil
			},
			issueJoinGrant: func(context.Context, *authv1.IssueJoinGrantRequest, ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
				return &authv1.IssueJoinGrantResponse{JoinGrant: "grant"}, nil
			},
		},
	}
}

func TestRun_DemoPreset(t *testing.T) {
	g := newTestGen(1, fullFakeDeps())
	g.config.Preset = PresetDemo

	if err := g.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_CampaignCountOverride(t *testing.T) {
	var campaignsCreated int
	deps := fullFakeDeps()
	camp := deps.campaigns.(*fakeCampaignCreator)
	origCreate := camp.createCampaign
	camp.createCampaign = func(ctx context.Context, in *statev1.CreateCampaignRequest, opts ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
		campaignsCreated++
		return origCreate(ctx, in, opts...)
	}

	g := newTestGen(1, deps)
	g.config.Preset = PresetDemo
	g.config.Campaigns = 3

	if err := g.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if campaignsCreated != 3 {
		t.Fatalf("expected 3 campaigns, got %d", campaignsCreated)
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	g := newTestGen(1, fullFakeDeps())
	g.config.Preset = PresetDemo

	err := g.Run(ctx)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestRun_VarietyPreset(t *testing.T) {
	g := newTestGen(42, fullFakeDeps())
	g.config.Preset = PresetVariety

	if err := g.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClose_NilSocialClient(t *testing.T) {
	g := &Generator{}
	if err := g.Close(); err != nil {
		t.Fatalf("expected nil error for nil social client, got %v", err)
	}
}

func TestGetPresetConfig_AllPresets(t *testing.T) {
	tests := []struct {
		preset    Preset
		campaigns int
	}{
		{PresetDemo, 1},
		{PresetVariety, 8},
		{PresetSessionHeavy, 2},
		{PresetStressTest, 50},
		{Preset("unknown"), 1}, // falls back to demo
	}
	for _, tc := range tests {
		t.Run(string(tc.preset), func(t *testing.T) {
			cfg := GetPresetConfig(tc.preset)
			if cfg.Campaigns != tc.campaigns {
				t.Fatalf("preset %q: expected %d campaigns, got %d", tc.preset, tc.campaigns, cfg.Campaigns)
			}
		})
	}
}
