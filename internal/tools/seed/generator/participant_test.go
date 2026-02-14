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

func TestCreateParticipants_CountLessThanOne(t *testing.T) {
	partSeq := 0
	part := &fakeParticipantCreator{
		create: func(_ context.Context, in *statev1.CreateParticipantRequest, _ ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
			partSeq++
			return &statev1.CreateParticipantResponse{
				Participant: &statev1.Participant{
					Id:          fmt.Sprintf("p-%d", partSeq),
					DisplayName: in.DisplayName,
					Role:        in.Role,
					Controller:  in.Controller,
				},
			}, nil
		},
	}
	auth := happyAuthCreator()
	inv := &fakeInviteManager{
		createInvite: func(_ context.Context, in *statev1.CreateInviteRequest, _ ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
			return &statev1.CreateInviteResponse{
				Invite: &statev1.Invite{Id: "inv-1"},
			}, nil
		},
		claimInvite: func(context.Context, *statev1.ClaimInviteRequest, ...grpc.CallOption) (*statev1.ClaimInviteResponse, error) {
			return &statev1.ClaimInviteResponse{}, nil
		},
	}
	g := newTestGen(1, testDeps(nil, part, inv, nil, nil, nil, auth))

	// count=0 should normalize to 1
	participants, err := g.createParticipants(context.Background(), "camp-1", "owner-1", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(participants))
	}
}

func TestCreateParticipants_FirstGmRestPlayer(t *testing.T) {
	var roles []statev1.ParticipantRole
	part := &fakeParticipantCreator{
		create: func(_ context.Context, in *statev1.CreateParticipantRequest, _ ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
			roles = append(roles, in.Role)
			return &statev1.CreateParticipantResponse{
				Participant: &statev1.Participant{
					Id:         fmt.Sprintf("p-%d", len(roles)),
					Role:       in.Role,
					Controller: in.Controller,
				},
			}, nil
		},
	}
	auth := happyAuthCreator()
	inv := &fakeInviteManager{
		createInvite: func(context.Context, *statev1.CreateInviteRequest, ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
			return &statev1.CreateInviteResponse{Invite: &statev1.Invite{Id: "inv-1"}}, nil
		},
		claimInvite: func(context.Context, *statev1.ClaimInviteRequest, ...grpc.CallOption) (*statev1.ClaimInviteResponse, error) {
			return &statev1.ClaimInviteResponse{}, nil
		},
	}
	g := newTestGen(1, testDeps(nil, part, inv, nil, nil, nil, auth))

	participants, err := g.createParticipants(context.Background(), "camp-1", "", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(participants) != 3 {
		t.Fatalf("expected 3 participants, got %d", len(participants))
	}
	if roles[0] != statev1.ParticipantRole_GM {
		t.Fatalf("first participant should be GM, got %v", roles[0])
	}
	for i := 1; i < len(roles); i++ {
		if roles[i] != statev1.ParticipantRole_PLAYER {
			t.Fatalf("participant %d should be PLAYER, got %v", i, roles[i])
		}
	}
}

func TestCreateParticipants_ControllerVariation(t *testing.T) {
	// With seed 42 the RNG should produce a mix of human and AI controllers.
	var controllers []statev1.Controller
	part := &fakeParticipantCreator{
		create: func(_ context.Context, in *statev1.CreateParticipantRequest, _ ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
			controllers = append(controllers, in.Controller)
			return &statev1.CreateParticipantResponse{
				Participant: &statev1.Participant{
					Id:         fmt.Sprintf("p-%d", len(controllers)),
					Controller: in.Controller,
				},
			}, nil
		},
	}
	auth := happyAuthCreator()
	inv := &fakeInviteManager{
		createInvite: func(context.Context, *statev1.CreateInviteRequest, ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
			return &statev1.CreateInviteResponse{Invite: &statev1.Invite{Id: "inv-1"}}, nil
		},
		claimInvite: func(context.Context, *statev1.ClaimInviteRequest, ...grpc.CallOption) (*statev1.ClaimInviteResponse, error) {
			return &statev1.ClaimInviteResponse{}, nil
		},
	}

	// Use seed 42 and enough participants to exercise the 20% AI branch.
	rng := rand.New(rand.NewSource(42))
	g := newGenerator(Config{Seed: 42}, rng, worldbuilder.New(rng), testDeps(nil, part, inv, nil, nil, nil, auth))

	_, err := g.createParticipants(context.Background(), "camp-1", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasHuman, hasAI := false, false
	for _, c := range controllers {
		if c == statev1.Controller_CONTROLLER_HUMAN {
			hasHuman = true
		}
		if c == statev1.Controller_CONTROLLER_AI {
			hasAI = true
		}
	}
	if !hasHuman {
		t.Fatal("expected at least one HUMAN controller")
	}
	if !hasAI {
		t.Fatal("expected at least one AI controller with seed 42 and 10 participants")
	}
}

func TestCreateParticipants_InviteCreateClaimFlow(t *testing.T) {
	// Use seed that exercises claimInvite=true branch (rng.Intn(4)==1 or 3).
	var inviteCreated, inviteClaimed bool
	part := &fakeParticipantCreator{
		create: func(_ context.Context, in *statev1.CreateParticipantRequest, _ ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
			return &statev1.CreateParticipantResponse{
				Participant: &statev1.Participant{
					Id:         "p-1",
					Controller: statev1.Controller_CONTROLLER_HUMAN,
				},
			}, nil
		},
	}
	auth := &fakeAuthProvider{
		createUser: func(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
			return &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}}, nil
		},
		issueJoinGrant: func(context.Context, *authv1.IssueJoinGrantRequest, ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
			return &authv1.IssueJoinGrantResponse{JoinGrant: "grant"}, nil
		},
	}
	inv := &fakeInviteManager{
		createInvite: func(context.Context, *statev1.CreateInviteRequest, ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
			inviteCreated = true
			return &statev1.CreateInviteResponse{Invite: &statev1.Invite{Id: "inv-1"}}, nil
		},
		claimInvite: func(context.Context, *statev1.ClaimInviteRequest, ...grpc.CallOption) (*statev1.ClaimInviteResponse, error) {
			inviteClaimed = true
			return &statev1.ClaimInviteResponse{}, nil
		},
	}

	// Seed 0 deterministically exercises the claimInvite=true branch.
	const seed int64 = 0
	rng := rand.New(rand.NewSource(seed))
	g := newGenerator(Config{}, rng, worldbuilder.New(rng), testDeps(nil, part, inv, nil, nil, nil, auth))

	_, err := g.createParticipants(context.Background(), "camp-1", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inviteCreated || !inviteClaimed {
		t.Fatal("seed 0 did not exercise the invite claim path")
	}
}

func TestCreateParticipants_CreateParticipantError(t *testing.T) {
	part := &fakeParticipantCreator{
		create: func(context.Context, *statev1.CreateParticipantRequest, ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
			return nil, fmt.Errorf("create failed")
		},
	}
	g := newTestGen(1, testDeps(nil, part, nil, nil, nil, nil, nil))

	_, err := g.createParticipants(context.Background(), "camp-1", "", 1)
	if err == nil {
		t.Fatal("expected error from CreateParticipant failure")
	}
}

func TestCreateParticipants_CreateInviteError(t *testing.T) {
	part := &fakeParticipantCreator{
		create: func(context.Context, *statev1.CreateParticipantRequest, ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
			return &statev1.CreateParticipantResponse{
				Participant: &statev1.Participant{
					Id:         "p-1",
					Controller: statev1.Controller_CONTROLLER_HUMAN,
				},
			}, nil
		},
	}
	auth := happyAuthCreator()
	inv := &fakeInviteManager{
		createInvite: func(context.Context, *statev1.CreateInviteRequest, ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
			return nil, fmt.Errorf("invite failed")
		},
	}
	g := newTestGen(1, testDeps(nil, part, inv, nil, nil, nil, auth))

	_, err := g.createParticipants(context.Background(), "camp-1", "", 1)
	if err == nil {
		t.Fatal("expected error from CreateInvite failure")
	}
}
