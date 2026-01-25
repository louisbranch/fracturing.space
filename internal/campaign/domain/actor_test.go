package domain

import (
	"errors"
	"testing"
	"time"
)

func TestCreateActorNormalizesInput(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateActorInput{
		CampaignID: "camp-123",
		Name:       "  Alice  ",
		Kind:       ActorKindPC,
		Notes:      "  A brave warrior  ",
	}

	actor, err := CreateActor(input, func() time.Time { return fixedTime }, func() (string, error) {
		return "actor-456", nil
	})
	if err != nil {
		t.Fatalf("create actor: %v", err)
	}

	if actor.ID != "actor-456" {
		t.Fatalf("expected id actor-456, got %q", actor.ID)
	}
	if actor.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", actor.CampaignID)
	}
	if actor.Name != "Alice" {
		t.Fatalf("expected trimmed name, got %q", actor.Name)
	}
	if actor.Kind != ActorKindPC {
		t.Fatalf("expected kind PC, got %v", actor.Kind)
	}
	if actor.Notes != "A brave warrior" {
		t.Fatalf("expected trimmed notes, got %q", actor.Notes)
	}
	if !actor.CreatedAt.Equal(fixedTime) || !actor.UpdatedAt.Equal(fixedTime) {
		t.Fatalf("expected timestamps to match fixed time")
	}
}

func TestNormalizeCreateActorInputValidation(t *testing.T) {
	tests := []struct {
		name  string
		input CreateActorInput
		err   error
	}{
		{
			name: "empty campaign id",
			input: CreateActorInput{
				CampaignID: "   ",
				Name:       "Alice",
				Kind:       ActorKindPC,
			},
			err: ErrEmptyCampaignID,
		},
		{
			name: "empty name",
			input: CreateActorInput{
				CampaignID: "camp-123",
				Name:       "   ",
				Kind:       ActorKindPC,
			},
			err: ErrEmptyActorName,
		},
		{
			name: "missing kind",
			input: CreateActorInput{
				CampaignID: "camp-123",
				Name:       "Alice",
				Kind:       ActorKindUnspecified,
			},
			err: ErrInvalidActorKind,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeCreateActorInput(tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}
