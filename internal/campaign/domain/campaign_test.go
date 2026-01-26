package domain

import (
	"errors"
	"testing"
	"time"
)

func TestCreateCampaignDefaults(t *testing.T) {
	input := CreateCampaignInput{
		Name:        "  The Glade  ",
		GmMode:      GmModeHuman,
		ThemePrompt: "moss and mist",
	}

	_, err := CreateCampaign(input, nil, nil)
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
}

func TestCreateCampaignNormalizesInput(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateCampaignInput{
		Name:        "  The Glade  ",
		GmMode:      GmModeHuman,
		ThemePrompt: "moss and mist",
	}

	campaign, err := CreateCampaign(input, func() time.Time { return fixedTime }, func() (string, error) {
		return "camp123", nil
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}

	if campaign.ID != "camp123" {
		t.Fatalf("expected id camp123, got %q", campaign.ID)
	}
	if campaign.Name != "The Glade" {
		t.Fatalf("expected trimmed name, got %q", campaign.Name)
	}
	if campaign.GmMode != GmModeHuman {
		t.Fatalf("expected gm mode human, got %v", campaign.GmMode)
	}
	if campaign.ParticipantCount != 0 {
		t.Fatalf("expected 0 participant count, got %d", campaign.ParticipantCount)
	}
	if campaign.CharacterCount != 0 {
		t.Fatalf("expected 0 character count, got %d", campaign.CharacterCount)
	}
	if campaign.ThemePrompt != "moss and mist" {
		t.Fatalf("expected theme prompt preserved, got %q", campaign.ThemePrompt)
	}
	if !campaign.CreatedAt.Equal(fixedTime) || !campaign.UpdatedAt.Equal(fixedTime) {
		t.Fatalf("expected timestamps to match fixed time")
	}
}

func TestNormalizeCreateCampaignInputValidation(t *testing.T) {
	tests := []struct {
		name  string
		input CreateCampaignInput
		err   error
	}{
		{
			name: "empty name",
			input: CreateCampaignInput{
				Name:   "   ",
				GmMode: GmModeHuman,
			},
			err: ErrEmptyName,
		},
		{
			name: "missing gm mode",
			input: CreateCampaignInput{
				Name:   "Campaign",
				GmMode: GmModeUnspecified,
			},
			err: ErrInvalidGmMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeCreateCampaignInput(tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}
