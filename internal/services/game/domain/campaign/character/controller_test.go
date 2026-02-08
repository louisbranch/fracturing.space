package character

import (
	"strings"
	"testing"
)

func TestCharacterControllerValidate(t *testing.T) {
	tests := []struct {
		name       string
		controller CharacterController
		wantErr    error
	}{
		{
			name: "valid GM controller",
			controller: CharacterController{
				IsGM: true,
			},
			wantErr: nil,
		},
		{
			name: "valid participant controller",
			controller: CharacterController{
				ParticipantID: "participant-123",
			},
			wantErr: nil,
		},
		{
			name: "invalid: both set",
			controller: CharacterController{
				IsGM:          true,
				ParticipantID: "participant-123",
			},
			wantErr: ErrInvalidCharacterController,
		},
		{
			name: "invalid: neither set",
			controller: CharacterController{
				IsGM:          false,
				ParticipantID: "",
			},
			wantErr: ErrInvalidCharacterController,
		},
		{
			name: "invalid: empty participant ID",
			controller: CharacterController{
				ParticipantID: "   ",
			},
			wantErr: ErrInvalidCharacterController,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.controller.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if err != tt.wantErr && !isError(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestNewGmController(t *testing.T) {
	ctrl := NewGmController()
	if !ctrl.IsGM {
		t.Fatal("expected IsGM to be true")
	}
	if ctrl.ParticipantID != "" {
		t.Fatalf("expected empty participant ID, got %q", ctrl.ParticipantID)
	}
	if err := ctrl.Validate(); err != nil {
		t.Fatalf("expected valid controller, got error: %v", err)
	}
}

func TestNewParticipantController(t *testing.T) {
	tests := []struct {
		name          string
		participantID string
		wantErr       error
	}{
		{
			name:          "valid participant ID",
			participantID: "participant-123",
			wantErr:       nil,
		},
		{
			name:          "empty participant ID",
			participantID: "",
			wantErr:       ErrEmptyParticipantID,
		},
		{
			name:          "whitespace participant ID",
			participantID: "   ",
			wantErr:       ErrEmptyParticipantID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl, err := NewParticipantController(tt.participantID)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if ctrl.ParticipantID != strings.TrimSpace(tt.participantID) {
					t.Fatalf("expected participant ID %q, got %q", strings.TrimSpace(tt.participantID), ctrl.ParticipantID)
				}
				if ctrl.IsGM {
					t.Fatal("expected IsGM to be false")
				}
			} else {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if err != tt.wantErr && !isError(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestMustNewParticipantController(t *testing.T) {
	ctrl := MustNewParticipantController("participant-123")
	if ctrl.ParticipantID != "participant-123" {
		t.Fatalf("expected participant ID participant-123, got %q", ctrl.ParticipantID)
	}
	if ctrl.IsGM {
		t.Fatal("expected IsGM to be false")
	}
}

func isError(err, target error) bool {
	return err.Error() == target.Error()
}
