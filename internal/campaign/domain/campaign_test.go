package domain

import (
	"encoding/base32"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCreateCampaignNormalizesInput(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateCampaignInput{
		Name:        "  The Glade  ",
		GmMode:      GmModeHuman,
		PlayerSlots: 4,
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
	if campaign.PlayerSlots != 4 {
		t.Fatalf("expected 4 player slots, got %d", campaign.PlayerSlots)
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
				Name:        "   ",
				GmMode:      GmModeHuman,
				PlayerSlots: 3,
			},
			err: ErrEmptyName,
		},
		{
			name: "missing gm mode",
			input: CreateCampaignInput{
				Name:        "Campaign",
				GmMode:      GmModeUnspecified,
				PlayerSlots: 3,
			},
			err: ErrInvalidGmMode,
		},
		{
			name: "invalid player slots",
			input: CreateCampaignInput{
				Name:        "Campaign",
				GmMode:      GmModeHuman,
				PlayerSlots: 0,
			},
			err: ErrInvalidPlayerSlots,
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

func TestNewCampaignIDFormat(t *testing.T) {
	id, err := NewCampaignID()
	if err != nil {
		t.Fatalf("new campaign id: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty id")
	}
	if strings.Contains(id, "=") {
		t.Fatal("expected no padding")
	}
	if len(id) != 26 {
		t.Fatalf("expected 26-character id, got %d", len(id))
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < '2' || r > '7') {
			t.Fatalf("unexpected character %q in id", r)
		}
	}

	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(id))
	if err != nil {
		t.Fatalf("decode id: %v", err)
	}
	if len(decoded) != 16 {
		t.Fatalf("expected 16 decoded bytes, got %d", len(decoded))
	}
}
