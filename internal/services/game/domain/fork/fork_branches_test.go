package fork

import (
	"errors"
	"testing"
	"time"
)

func TestForkPointIsSessionBoundary(t *testing.T) {
	if !(ForkPoint{SessionID: "sess-1"}).IsSessionBoundary() {
		t.Fatal("expected session boundary when SessionID is set")
	}
	if (ForkPoint{SessionID: ""}).IsSessionBoundary() {
		t.Fatal("expected non-session boundary when SessionID is empty")
	}
}

func TestCreateFork_RejectsMissingSourceCampaignID(t *testing.T) {
	_, err := CreateFork(
		CreateForkInput{SourceCampaignID: "   "},
		"",
		1,
		time.Now,
		func() (string, error) { return "fork-1", nil },
	)
	if !errors.Is(err, ErrEmptyCampaignID) {
		t.Fatalf("expected ErrEmptyCampaignID, got %v", err)
	}
}

func TestCreateFork_PanicsOnNilNow(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil now function")
		}
	}()
	_, _ = CreateFork(
		CreateForkInput{SourceCampaignID: "camp-1"},
		"origin-1",
		1,
		nil,
		func() (string, error) { return "fork-1", nil },
	)
}

func TestCreateFork_UsesDefaultIDGeneratorWhenNil(t *testing.T) {
	fixed := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	result, err := CreateFork(
		CreateForkInput{SourceCampaignID: "camp-1"},
		"origin-1",
		1,
		func() time.Time { return fixed },
		nil,
	)
	if err != nil {
		t.Fatalf("CreateFork() unexpected error: %v", err)
	}
	if result.NewCampaignID == "" {
		t.Fatal("expected generated campaign id")
	}
	if !result.CreatedAt.Equal(fixed.UTC()) {
		t.Fatalf("CreatedAt = %s, want %s", result.CreatedAt, fixed.UTC())
	}
}
