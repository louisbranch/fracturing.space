package session

import (
	"errors"
	"testing"
	"time"
)

func TestCreateSessionNormalizesInput(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateSessionInput{
		CampaignID: "  camp-123  ",
		Name:       "  First Session  ",
	}

	session, err := CreateSession(input, func() time.Time { return fixedTime }, func() (string, error) {
		return "sess-456", nil
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if session.ID != "sess-456" {
		t.Fatalf("expected id sess-456, got %q", session.ID)
	}
	if session.CampaignID != "camp-123" {
		t.Fatalf("expected trimmed campaign id, got %q", session.CampaignID)
	}
	if session.Name != "First Session" {
		t.Fatalf("expected trimmed name, got %q", session.Name)
	}
	if session.Status != SessionStatusActive {
		t.Fatalf("expected active status, got %v", session.Status)
	}
	if !session.StartedAt.Equal(fixedTime) {
		t.Fatalf("expected started_at to match fixed time, got %v", session.StartedAt)
	}
	if !session.UpdatedAt.Equal(fixedTime) {
		t.Fatalf("expected updated_at to match fixed time, got %v", session.UpdatedAt)
	}
	if session.EndedAt != nil {
		t.Fatalf("expected nil ended_at, got %v", session.EndedAt)
	}
}

func TestCreateSessionWithEmptyName(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateSessionInput{
		CampaignID: "camp-123",
		Name:       "   ",
	}

	session, err := CreateSession(input, func() time.Time { return fixedTime }, func() (string, error) {
		return "sess-456", nil
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if session.Name != "" {
		t.Fatalf("expected empty name, got %q", session.Name)
	}
	if session.Status != SessionStatusActive {
		t.Fatalf("expected active status, got %v", session.Status)
	}
}

func TestCreateSessionWithNoName(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateSessionInput{
		CampaignID: "camp-123",
		Name:       "",
	}

	session, err := CreateSession(input, func() time.Time { return fixedTime }, func() (string, error) {
		return "sess-456", nil
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if session.Name != "" {
		t.Fatalf("expected empty name, got %q", session.Name)
	}
}

func TestNormalizeCreateSessionInputValidation(t *testing.T) {
	tests := []struct {
		name  string
		input CreateSessionInput
		err   error
	}{
		{
			name: "empty campaign id",
			input: CreateSessionInput{
				CampaignID: "   ",
				Name:       "Session",
			},
			err: ErrEmptyCampaignID,
		},
		{
			name: "missing campaign id",
			input: CreateSessionInput{
				CampaignID: "",
				Name:       "Session",
			},
			err: ErrEmptyCampaignID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeCreateSessionInput(tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}

func TestNormalizeCreateSessionInputAllowsEmptyName(t *testing.T) {
	input := CreateSessionInput{
		CampaignID: "camp-123",
		Name:       "",
	}

	normalized, err := NormalizeCreateSessionInput(input)
	if err != nil {
		t.Fatalf("normalize input: %v", err)
	}

	if normalized.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", normalized.CampaignID)
	}
	if normalized.Name != "" {
		t.Fatalf("expected empty name, got %q", normalized.Name)
	}
}

func TestNormalizeCreateSessionInputTrimsWhitespace(t *testing.T) {
	input := CreateSessionInput{
		CampaignID: "  camp-123  ",
		Name:       "  Session Name  ",
	}

	normalized, err := NormalizeCreateSessionInput(input)
	if err != nil {
		t.Fatalf("normalize input: %v", err)
	}

	if normalized.CampaignID != "camp-123" {
		t.Fatalf("expected trimmed campaign id, got %q", normalized.CampaignID)
	}
	if normalized.Name != "Session Name" {
		t.Fatalf("expected trimmed name, got %q", normalized.Name)
	}
}

func TestCreateSessionIDGenerationFailure(t *testing.T) {
	input := CreateSessionInput{
		CampaignID: "camp-123",
		Name:       "Session",
	}

	_, err := CreateSession(input, time.Now, func() (string, error) {
		return "", errors.New("id generation failed")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "generate session id: id generation failed" {
		t.Fatalf("expected id generation error, got %v", err)
	}
}

func TestCreateSessionUsesDefaults(t *testing.T) {
	input := CreateSessionInput{
		CampaignID: "camp-123",
		Name:       "Session",
	}

	// Test with nil now and idGenerator - should use defaults
	session, err := CreateSession(input, nil, nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if session.ID == "" {
		t.Fatal("expected generated id")
	}
	if session.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", session.CampaignID)
	}
	if session.Name != "Session" {
		t.Fatalf("expected name Session, got %q", session.Name)
	}
	if session.Status != SessionStatusActive {
		t.Fatalf("expected active status, got %v", session.Status)
	}
	if session.StartedAt.IsZero() {
		t.Fatal("expected non-zero started_at")
	}
	if session.UpdatedAt.IsZero() {
		t.Fatal("expected non-zero updated_at")
	}
	if session.EndedAt != nil {
		t.Fatalf("expected nil ended_at, got %v", session.EndedAt)
	}
}

func TestCreateSessionTimestampsAreUTC(t *testing.T) {
	// Use a timezone-aware time
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, loc)

	input := CreateSessionInput{
		CampaignID: "camp-123",
		Name:       "Session",
	}

	session, err := CreateSession(input, func() time.Time { return fixedTime }, func() (string, error) {
		return "sess-456", nil
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Timestamps should be converted to UTC
	expectedUTC := fixedTime.UTC()
	if !session.StartedAt.Equal(expectedUTC) {
		t.Fatalf("expected UTC started_at %v, got %v", expectedUTC, session.StartedAt)
	}
	if !session.UpdatedAt.Equal(expectedUTC) {
		t.Fatalf("expected UTC updated_at %v, got %v", expectedUTC, session.UpdatedAt)
	}
	if session.StartedAt.Location() != time.UTC {
		t.Fatalf("expected UTC location for started_at, got %v", session.StartedAt.Location())
	}
	if session.UpdatedAt.Location() != time.UTC {
		t.Fatalf("expected UTC location for updated_at, got %v", session.UpdatedAt.Location())
	}
}

func TestCreateSessionAlwaysSetsActiveStatus(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateSessionInput{
		CampaignID: "camp-123",
		Name:       "Session",
	}

	session, err := CreateSession(input, func() time.Time { return fixedTime }, func() (string, error) {
		return "sess-456", nil
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if session.Status != SessionStatusActive {
		t.Fatalf("expected active status, got %v", session.Status)
	}
}
