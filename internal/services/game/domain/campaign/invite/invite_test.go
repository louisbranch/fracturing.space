package invite

import (
	"strings"
	"testing"
	"time"
)

func TestCreateInviteSuccess(t *testing.T) {
	fixedTime := time.Date(2024, 2, 1, 10, 0, 0, 0, time.UTC)
	input := CreateInviteInput{
		CampaignID:             "  campaign-1 ",
		ParticipantID:          " participant-1 ",
		RecipientUserID:        " user-1 ",
		CreatedByParticipantID: " gm-1 ",
	}

	invite, err := CreateInvite(input, func() time.Time { return fixedTime }, func() (string, error) { return "invite-1", nil })
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}

	if invite.ID != "invite-1" {
		t.Fatalf("expected invite id invite-1, got %s", invite.ID)
	}
	if invite.CampaignID != "campaign-1" {
		t.Fatalf("expected campaign id campaign-1, got %s", invite.CampaignID)
	}
	if invite.ParticipantID != "participant-1" {
		t.Fatalf("expected participant id participant-1, got %s", invite.ParticipantID)
	}
	if invite.RecipientUserID != "user-1" {
		t.Fatalf("expected recipient user id user-1, got %s", invite.RecipientUserID)
	}
	if invite.CreatedByParticipantID != "gm-1" {
		t.Fatalf("expected creator participant id gm-1, got %s", invite.CreatedByParticipantID)
	}
	if invite.Status != StatusPending {
		t.Fatalf("expected pending status, got %v", invite.Status)
	}
	if !invite.CreatedAt.Equal(fixedTime) || !invite.UpdatedAt.Equal(fixedTime) {
		t.Fatal("expected created and updated timestamps to match fixed time")
	}
}

func TestCreateInviteValidation(t *testing.T) {
	_, err := CreateInvite(CreateInviteInput{ParticipantID: "p1"}, time.Now, func() (string, error) { return "id", nil })
	if err == nil {
		t.Fatal("expected error for missing campaign id")
	}

	_, err = CreateInvite(CreateInviteInput{CampaignID: "c1"}, time.Now, func() (string, error) { return "id", nil })
	if err == nil {
		t.Fatal("expected error for missing participant id")
	}
}

func TestNormalizeCreateInviteInputTrims(t *testing.T) {
	input := CreateInviteInput{
		CampaignID:             "  c1 ",
		ParticipantID:          " p1 ",
		RecipientUserID:        " user ",
		CreatedByParticipantID: " gm ",
	}

	normalized, err := NormalizeCreateInviteInput(input)
	if err != nil {
		t.Fatalf("normalize input: %v", err)
	}
	if normalized.CampaignID != "c1" || normalized.ParticipantID != "p1" {
		t.Fatal("expected campaign and participant ids to be trimmed")
	}
	if normalized.RecipientUserID != "user" || normalized.CreatedByParticipantID != "gm" {
		t.Fatal("expected recipient and creator ids to be trimmed")
	}
}

func TestStatusLabelRoundTrip(t *testing.T) {
	labels := []string{
		StatusLabel(StatusPending),
		StatusLabel(StatusClaimed),
		StatusLabel(StatusRevoked),
	}

	for _, label := range labels {
		if StatusFromLabel(label) == StatusUnspecified {
			t.Fatalf("expected label %s to map to status", label)
		}
	}

	if StatusFromLabel("  pending ") != StatusPending {
		t.Fatal("expected case-insensitive status parsing")
	}
	if StatusLabel(StatusUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected unspecified label for unknown status")
	}
	if StatusFromLabel(strings.Repeat("x", 5)) != StatusUnspecified {
		t.Fatal("expected unknown labels to map to unspecified")
	}
}
