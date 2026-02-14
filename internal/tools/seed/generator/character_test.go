package generator

import (
	"context"
	"fmt"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
)

func TestCreateCharacters_CountLessThanOne(t *testing.T) {
	g := newTestGen(1, testDeps(nil, nil, nil, &fakeCharacterCreator{}, nil, nil, nil))

	chars, err := g.createCharacters(context.Background(), "camp-1", 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chars != nil {
		t.Fatalf("expected nil, got %v", chars)
	}
}

func TestCreateCharacters_PCsAssignedToPlayers_NPCsToGM(t *testing.T) {
	var kinds []statev1.CharacterKind
	var controllerParticipants []string
	charSeq := 0
	char := &fakeCharacterCreator{
		create: func(_ context.Context, in *statev1.CreateCharacterRequest, _ ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
			charSeq++
			kinds = append(kinds, in.Kind)
			return &statev1.CreateCharacterResponse{
				Character: &statev1.Character{Id: fmt.Sprintf("char-%d", charSeq)},
			}, nil
		},
		setDefaultControl: func(_ context.Context, in *statev1.SetDefaultControlRequest, _ ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error) {
			controllerParticipants = append(controllerParticipants, in.ParticipantId.GetValue())
			return &statev1.SetDefaultControlResponse{}, nil
		},
	}

	participants := []*statev1.Participant{
		{Id: "gm-1", Role: statev1.ParticipantRole_GM},
		{Id: "player-1", Role: statev1.ParticipantRole_PLAYER},
		{Id: "player-2", Role: statev1.ParticipantRole_PLAYER},
	}

	g := newTestGen(1, testDeps(nil, nil, nil, char, nil, nil, nil))

	chars, err := g.createCharacters(context.Background(), "camp-1", 4, participants)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chars) != 4 {
		t.Fatalf("expected 4 characters, got %d", len(chars))
	}

	// First 2 are PCs (matching player count), rest are NPCs
	if kinds[0] != statev1.CharacterKind_PC || kinds[1] != statev1.CharacterKind_PC {
		t.Fatalf("first 2 should be PCs: %v", kinds)
	}
	if kinds[2] != statev1.CharacterKind_NPC || kinds[3] != statev1.CharacterKind_NPC {
		t.Fatalf("last 2 should be NPCs: %v", kinds)
	}

	// PCs assigned to players, NPCs to GM
	if controllerParticipants[0] != "player-1" || controllerParticipants[1] != "player-2" {
		t.Fatalf("PCs should be assigned to players: %v", controllerParticipants)
	}
	if controllerParticipants[2] != "gm-1" || controllerParticipants[3] != "gm-1" {
		t.Fatalf("NPCs should be assigned to GM: %v", controllerParticipants)
	}
}

func TestCreateCharacters_FallbackParticipant(t *testing.T) {
	// Only a GM participant (no players) — all characters are NPCs assigned to the GM.
	var controllerParticipants []string
	charSeq := 0
	char := &fakeCharacterCreator{
		create: func(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
			charSeq++
			return &statev1.CreateCharacterResponse{
				Character: &statev1.Character{Id: fmt.Sprintf("char-%d", charSeq)},
			}, nil
		},
		setDefaultControl: func(_ context.Context, in *statev1.SetDefaultControlRequest, _ ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error) {
			controllerParticipants = append(controllerParticipants, in.ParticipantId.GetValue())
			return &statev1.SetDefaultControlResponse{}, nil
		},
	}

	participants := []*statev1.Participant{
		{Id: "gm-1", Role: statev1.ParticipantRole_GM},
	}

	g := newTestGen(1, testDeps(nil, nil, nil, char, nil, nil, nil))

	chars, err := g.createCharacters(context.Background(), "camp-1", 2, participants)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chars) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(chars))
	}
	for i, cp := range controllerParticipants {
		if cp != "gm-1" {
			t.Fatalf("character %d: expected gm-1, got %s", i, cp)
		}
	}
}

func TestCreateCharacters_NoParticipants(t *testing.T) {
	charSeq := 0
	char := &fakeCharacterCreator{
		create: func(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
			charSeq++
			return &statev1.CreateCharacterResponse{
				Character: &statev1.Character{Id: fmt.Sprintf("char-%d", charSeq)},
			}, nil
		},
	}

	g := newTestGen(1, testDeps(nil, nil, nil, char, nil, nil, nil))

	_, err := g.createCharacters(context.Background(), "camp-1", 1, nil)
	if err == nil {
		t.Fatal("expected error when no participants available")
	}
}

func TestCreateCharacters_CreateError(t *testing.T) {
	char := &fakeCharacterCreator{
		create: func(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
			return nil, fmt.Errorf("create character failed")
		},
	}

	participants := []*statev1.Participant{
		{Id: "gm-1", Role: statev1.ParticipantRole_GM},
	}

	g := newTestGen(1, testDeps(nil, nil, nil, char, nil, nil, nil))

	_, err := g.createCharacters(context.Background(), "camp-1", 1, participants)
	if err == nil {
		t.Fatal("expected error from CreateCharacter failure")
	}
}

func TestCreateCharacters_SetDefaultControlError(t *testing.T) {
	char := &fakeCharacterCreator{
		create: func(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
			return &statev1.CreateCharacterResponse{
				Character: &statev1.Character{Id: "char-1"},
			}, nil
		},
		setDefaultControl: func(context.Context, *statev1.SetDefaultControlRequest, ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error) {
			return nil, fmt.Errorf("set control failed")
		},
	}

	participants := []*statev1.Participant{
		{Id: "gm-1", Role: statev1.ParticipantRole_GM},
	}

	g := newTestGen(1, testDeps(nil, nil, nil, char, nil, nil, nil))

	_, err := g.createCharacters(context.Background(), "camp-1", 1, participants)
	if err == nil {
		t.Fatal("expected error from SetDefaultControl failure")
	}
}

func TestCreateCharacters_NilParticipantSkipped(t *testing.T) {
	// Nil participants in the slice should be skipped without panic.
	charSeq := 0
	char := &fakeCharacterCreator{
		create: func(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
			charSeq++
			return &statev1.CreateCharacterResponse{
				Character: &statev1.Character{Id: fmt.Sprintf("char-%d", charSeq)},
			}, nil
		},
		setDefaultControl: func(context.Context, *statev1.SetDefaultControlRequest, ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error) {
			return &statev1.SetDefaultControlResponse{}, nil
		},
	}

	participants := []*statev1.Participant{
		nil,
		{Id: "gm-1", Role: statev1.ParticipantRole_GM},
	}

	g := newTestGen(1, testDeps(nil, nil, nil, char, nil, nil, nil))

	chars, err := g.createCharacters(context.Background(), "camp-1", 1, participants)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chars) != 1 {
		t.Fatalf("expected 1 character, got %d", len(chars))
	}
}
