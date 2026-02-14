package campaign

import (
	"errors"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
)

func TestCreateCampaignDefaults(t *testing.T) {
	input := CreateCampaignInput{
		Name:        "  The Glade  ",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
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
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
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
	if campaign.Status != CampaignStatusDraft {
		t.Fatalf("expected status draft, got %v", campaign.Status)
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
				System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
				GmMode: GmModeHuman,
			},
			err: ErrEmptyName,
		},
		{
			name: "missing gm mode",
			input: CreateCampaignInput{
				Name:   "Campaign",
				System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
				GmMode: GmModeUnspecified,
			},
			err: ErrInvalidGmMode,
		},
		{
			name: "missing game system",
			input: CreateCampaignInput{
				Name:   "Campaign",
				System: commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED,
				GmMode: GmModeHuman,
			},
			err: ErrInvalidGameSystem,
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

func TestTransitionCampaignStatusAllowed(t *testing.T) {
	baseTime := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	transitionTime := baseTime.Add(2 * time.Hour)

	t.Run("draft to active", func(t *testing.T) {
		campaign := Campaign{
			ID:        "camp-1",
			Status:    CampaignStatusDraft,
			CreatedAt: baseTime,
			UpdatedAt: baseTime,
		}
		updated, err := TransitionCampaignStatus(campaign, CampaignStatusActive, func() time.Time {
			return transitionTime
		})
		if err != nil {
			t.Fatalf("transition: %v", err)
		}
		if updated.Status != CampaignStatusActive {
			t.Fatalf("expected status ACTIVE, got %v", updated.Status)
		}
		if !updated.UpdatedAt.Equal(transitionTime) {
			t.Fatalf("expected updated_at %v, got %v", transitionTime, updated.UpdatedAt)
		}
		if updated.CompletedAt != nil {
			t.Fatalf("expected completed_at nil, got %v", updated.CompletedAt)
		}
		if updated.ArchivedAt != nil {
			t.Fatalf("expected archived_at nil, got %v", updated.ArchivedAt)
		}
	})

	t.Run("active to completed", func(t *testing.T) {
		campaign := Campaign{
			ID:        "camp-2",
			Status:    CampaignStatusActive,
			CreatedAt: baseTime,
			UpdatedAt: baseTime,
		}
		updated, err := TransitionCampaignStatus(campaign, CampaignStatusCompleted, func() time.Time {
			return transitionTime
		})
		if err != nil {
			t.Fatalf("transition: %v", err)
		}
		if updated.CompletedAt == nil || !updated.CompletedAt.Equal(transitionTime) {
			t.Fatalf("expected completed_at %v, got %v", transitionTime, updated.CompletedAt)
		}
	})

	t.Run("active to archived", func(t *testing.T) {
		campaign := Campaign{
			ID:        "camp-3",
			Status:    CampaignStatusActive,
			CreatedAt: baseTime,
			UpdatedAt: baseTime,
		}
		updated, err := TransitionCampaignStatus(campaign, CampaignStatusArchived, func() time.Time {
			return transitionTime
		})
		if err != nil {
			t.Fatalf("transition: %v", err)
		}
		if updated.ArchivedAt == nil || !updated.ArchivedAt.Equal(transitionTime) {
			t.Fatalf("expected archived_at %v, got %v", transitionTime, updated.ArchivedAt)
		}
		if updated.CompletedAt != nil {
			t.Fatalf("expected completed_at nil, got %v", updated.CompletedAt)
		}
	})

	t.Run("completed to archived preserves completed_at", func(t *testing.T) {
		completedAt := baseTime.Add(30 * time.Minute)
		campaign := Campaign{
			ID:          "camp-4",
			Status:      CampaignStatusCompleted,
			CreatedAt:   baseTime,
			UpdatedAt:   baseTime,
			CompletedAt: &completedAt,
		}
		updated, err := TransitionCampaignStatus(campaign, CampaignStatusArchived, func() time.Time {
			return transitionTime
		})
		if err != nil {
			t.Fatalf("transition: %v", err)
		}
		if updated.CompletedAt == nil || !updated.CompletedAt.Equal(completedAt) {
			t.Fatalf("expected completed_at %v, got %v", completedAt, updated.CompletedAt)
		}
		if updated.ArchivedAt == nil || !updated.ArchivedAt.Equal(transitionTime) {
			t.Fatalf("expected archived_at %v, got %v", transitionTime, updated.ArchivedAt)
		}
	})

	t.Run("archived to draft clears timestamps", func(t *testing.T) {
		completedAt := baseTime.Add(time.Hour)
		archivedAt := baseTime.Add(2 * time.Hour)
		campaign := Campaign{
			ID:          "camp-5",
			Status:      CampaignStatusArchived,
			CreatedAt:   baseTime,
			UpdatedAt:   baseTime,
			CompletedAt: &completedAt,
			ArchivedAt:  &archivedAt,
		}
		updated, err := TransitionCampaignStatus(campaign, CampaignStatusDraft, func() time.Time {
			return transitionTime
		})
		if err != nil {
			t.Fatalf("transition: %v", err)
		}
		if updated.Status != CampaignStatusDraft {
			t.Fatalf("expected status DRAFT, got %v", updated.Status)
		}
		if updated.CompletedAt != nil {
			t.Fatalf("expected completed_at cleared, got %v", updated.CompletedAt)
		}
		if updated.ArchivedAt != nil {
			t.Fatalf("expected archived_at cleared, got %v", updated.ArchivedAt)
		}
	})
}

func TestTransitionCampaignStatusDisallowed(t *testing.T) {
	tests := []struct {
		name string
		from CampaignStatus
		to   CampaignStatus
	}{
		{name: "draft to archived", from: CampaignStatusDraft, to: CampaignStatusArchived},
		{name: "draft to completed", from: CampaignStatusDraft, to: CampaignStatusCompleted},
		{name: "completed to active", from: CampaignStatusCompleted, to: CampaignStatusActive},
		{name: "completed to draft", from: CampaignStatusCompleted, to: CampaignStatusDraft},
		{name: "archived to completed", from: CampaignStatusArchived, to: CampaignStatusCompleted},
		{name: "archived to active", from: CampaignStatusArchived, to: CampaignStatusActive},
		{name: "active to draft", from: CampaignStatusActive, to: CampaignStatusDraft},
		{name: "unspecified to active", from: CampaignStatusUnspecified, to: CampaignStatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			campaign := Campaign{
				ID:        "camp-1",
				Status:    tt.from,
				CreatedAt: time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC),
			}
			_, err := TransitionCampaignStatus(campaign, tt.to, time.Now)
			if !errors.Is(err, ErrInvalidCampaignStatusTransition) {
				t.Fatalf("expected ErrInvalidCampaignStatusTransition, got %v", err)
			}
		})
	}
}

func TestTransitionCampaignStatusDisallowedMetadata(t *testing.T) {
	campaign := Campaign{ID: "camp-1", Status: CampaignStatusDraft}

	_, err := TransitionCampaignStatus(campaign, CampaignStatusArchived, func() time.Time { return time.Now().UTC() })
	if err == nil {
		t.Fatal("expected error")
	}

	var domainErr *apperrors.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected domain error, got %T", err)
	}
	if domainErr.Code != apperrors.CodeCampaignInvalidStatusTransition {
		t.Fatalf("expected code %s, got %s", apperrors.CodeCampaignInvalidStatusTransition, domainErr.Code)
	}
	if domainErr.Metadata["FromStatus"] != "DRAFT" {
		t.Fatalf("expected FromStatus DRAFT, got %s", domainErr.Metadata["FromStatus"])
	}
	if domainErr.Metadata["ToStatus"] != "ARCHIVED" {
		t.Fatalf("expected ToStatus ARCHIVED, got %s", domainErr.Metadata["ToStatus"])
	}
}

func TestTransitionCampaignStatusPreservesExistingTimestamps(t *testing.T) {
	baseTime := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	completedAt := baseTime.Add(30 * time.Minute)
	archivedAt := baseTime.Add(90 * time.Minute)
	updatedAt := baseTime.Add(2 * time.Hour)

	campaign := Campaign{
		ID:          "camp-1",
		Status:      CampaignStatusActive,
		CreatedAt:   baseTime,
		UpdatedAt:   baseTime,
		CompletedAt: &completedAt,
		ArchivedAt:  &archivedAt,
	}

	updated, err := TransitionCampaignStatus(campaign, CampaignStatusCompleted, func() time.Time { return updatedAt })
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if updated.CompletedAt == nil || !updated.CompletedAt.Equal(completedAt) {
		t.Fatalf("expected completed_at preserved, got %v", updated.CompletedAt)
	}
	if updated.ArchivedAt == nil || !updated.ArchivedAt.Equal(archivedAt) {
		t.Fatalf("expected archived_at preserved, got %v", updated.ArchivedAt)
	}
	if !updated.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected updated_at %v, got %v", updatedAt, updated.UpdatedAt)
	}
}
