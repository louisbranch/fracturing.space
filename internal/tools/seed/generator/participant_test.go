package generator

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

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
					Id:         fmt.Sprintf("p-%d", partSeq),
					Name:       in.Name,
					Role:       in.Role,
					Controller: in.Controller,
				},
			}, nil
		},
	}
	g := newTestGen(1, testDeps(nil, part, nil, nil, nil, nil))

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
	g := newTestGen(1, testDeps(nil, part, nil, nil, nil, nil))

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
	// Use seed 42 and enough participants to exercise the 20% AI branch.
	rng := rand.New(rand.NewSource(42))
	g := newGenerator(Config{Seed: 42}, rng, worldbuilder.New(rng), testDeps(nil, part, nil, nil, nil, nil))

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

func TestCreateParticipants_HumanParticipantsDoNotRequireInviteFlow(t *testing.T) {
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
	g := newGenerator(Config{}, rand.New(rand.NewSource(0)), worldbuilder.New(rand.New(rand.NewSource(0))), testDeps(nil, part, nil, nil, nil, nil))

	_, err := g.createParticipants(context.Background(), "camp-1", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
	g := newTestGen(1, testDeps(nil, part, nil, nil, nil, nil))

	_, err := g.createParticipants(context.Background(), "camp-1", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
