package generator

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/tools/seed/worldbuilder"
	"google.golang.org/grpc"
)

// sequenceSource returns a deterministic series of values to force duplicate names
// while keeping participant controllers human for CreateUser calls.
type sequenceSource struct {
	values []int64
	index  int
}

func (s *sequenceSource) Int63() int64 {
	if len(s.values) == 0 {
		return 0
	}
	if s.index >= len(s.values) {
		return 0
	}
	value := s.values[s.index]
	s.index++
	return value
}

func (s *sequenceSource) Seed(seed int64) {
	s.index = 0
}

func TestGenerator_UniqueUserDisplayNames(t *testing.T) {
	var usernames []string
	auth := &fakeAuthProvider{
		createUser: func(_ context.Context, in *authv1.CreateUserRequest, _ ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
			usernames = append(usernames, in.GetUsername())
			return &authv1.CreateUserResponse{User: &authv1.User{Id: fmt.Sprintf("user-%d", len(usernames))}}, nil
		},
		issueJoinGrant: func(context.Context, *authv1.IssueJoinGrantRequest, ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
			return &authv1.IssueJoinGrantResponse{JoinGrant: "grant"}, nil
		},
	}
	camp := &fakeCampaignCreator{
		createCampaign: func(_ context.Context, _ *statev1.CreateCampaignRequest, _ ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
			return &statev1.CreateCampaignResponse{
				Campaign:         &statev1.Campaign{Id: "camp-1", Name: "camp"},
				OwnerParticipant: &statev1.Participant{Id: "owner-1"},
			}, nil
		},
	}
	partSeq := 0
	part := &fakeParticipantCreator{
		create: func(_ context.Context, in *statev1.CreateParticipantRequest, _ ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
			partSeq++
			return &statev1.CreateParticipantResponse{
				Participant: &statev1.Participant{
					Id:          fmt.Sprintf("p-%d", partSeq),
					DisplayName: in.DisplayName,
					Controller:  in.Controller,
				},
			}, nil
		},
	}
	inv := &fakeInviteManager{
		createInvite: func(context.Context, *statev1.CreateInviteRequest, ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
			return &statev1.CreateInviteResponse{Invite: &statev1.Invite{Id: "inv-1"}}, nil
		},
		claimInvite: func(context.Context, *statev1.ClaimInviteRequest, ...grpc.CallOption) (*statev1.ClaimInviteResponse, error) {
			return &statev1.ClaimInviteResponse{}, nil
		},
	}

	mid := int64(1 << 62)
	rng := rand.New(&sequenceSource{values: []int64{0, 0, 0, 0, mid, 0, 0, mid, 0, 0}})
	g := newGenerator(Config{}, rng, worldbuilder.New(rng), testDeps(camp, part, inv, nil, nil, nil, auth))

	if _, _, err := g.createCampaign(context.Background(), statev1.GmMode_HUMAN); err != nil {
		t.Fatalf("unexpected campaign error: %v", err)
	}
	if _, err := g.createParticipants(context.Background(), "camp-1", "", 2); err != nil {
		t.Fatalf("unexpected participant error: %v", err)
	}

	if len(usernames) < 3 {
		t.Fatalf("expected at least 3 users created, got %d", len(usernames))
	}
	seen := make(map[string]struct{})
	for _, name := range usernames {
		if _, ok := seen[name]; ok {
			t.Fatalf("expected unique CreateUser usernames, got duplicate %q", name)
		}
		seen[name] = struct{}{}
	}
}
