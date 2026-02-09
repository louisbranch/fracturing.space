package participant

import (
	"errors"
	"testing"
	"time"
)

func TestCreateParticipantDefaults(t *testing.T) {
	input := CreateParticipantInput{
		CampaignID:  "camp-123",
		DisplayName: "Alice",
		Role:        ParticipantRolePlayer,
	}
	_, err := CreateParticipant(input, nil, nil)
	if err != nil {
		t.Fatalf("create participant: %v", err)
	}

	// id generator error
	_, err = CreateParticipant(input, nil, func() (string, error) { return "", errors.New("id generator error") })
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateParticipantNormalizesInput(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateParticipantInput{
		CampaignID:  "camp-123",
		DisplayName: "  Alice  ",
		Role:        ParticipantRolePlayer,
		Controller:  ControllerHuman,
	}

	participant, err := CreateParticipant(input, func() time.Time { return fixedTime }, func() (string, error) {
		return "part-456", nil
	})
	if err != nil {
		t.Fatalf("create participant: %v", err)
	}

	if participant.ID != "part-456" {
		t.Fatalf("expected id part-456, got %q", participant.ID)
	}
	if participant.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", participant.CampaignID)
	}
	if participant.DisplayName != "Alice" {
		t.Fatalf("expected trimmed display name, got %q", participant.DisplayName)
	}
	if participant.Role != ParticipantRolePlayer {
		t.Fatalf("expected role player, got %v", participant.Role)
	}
	if participant.Controller != ControllerHuman {
		t.Fatalf("expected controller human, got %v", participant.Controller)
	}
	if participant.CampaignAccess != CampaignAccessMember {
		t.Fatalf("expected access member, got %v", participant.CampaignAccess)
	}
	if !participant.CreatedAt.Equal(fixedTime) || !participant.UpdatedAt.Equal(fixedTime) {
		t.Fatalf("expected timestamps to match fixed time")
	}
}

func TestCreateParticipantDefaultsController(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateParticipantInput{
		CampaignID:  "camp-123",
		DisplayName: "Bob",
		Role:        ParticipantRoleGM,
		Controller:  ControllerUnspecified,
	}

	participant, err := CreateParticipant(input, func() time.Time { return fixedTime }, func() (string, error) {
		return "part-789", nil
	})
	if err != nil {
		t.Fatalf("create participant: %v", err)
	}

	if participant.Controller != ControllerHuman {
		t.Fatalf("expected default controller human, got %v", participant.Controller)
	}
	if participant.CampaignAccess != CampaignAccessMember {
		t.Fatalf("expected default access member, got %v", participant.CampaignAccess)
	}
}

func TestNormalizeCreateParticipantInputValidation(t *testing.T) {
	tests := []struct {
		name  string
		input CreateParticipantInput
		err   error
	}{
		{
			name: "empty campaign id",
			input: CreateParticipantInput{
				CampaignID:  "   ",
				DisplayName: "Alice",
				Role:        ParticipantRolePlayer,
			},
			err: ErrEmptyCampaignID,
		},
		{
			name: "empty display name",
			input: CreateParticipantInput{
				CampaignID:  "camp-123",
				DisplayName: "   ",
				Role:        ParticipantRolePlayer,
			},
			err: ErrEmptyDisplayName,
		},
		{
			name: "missing role",
			input: CreateParticipantInput{
				CampaignID:  "camp-123",
				DisplayName: "Alice",
				Role:        ParticipantRoleUnspecified,
			},
			err: ErrInvalidParticipantRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeCreateParticipantInput(tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}
