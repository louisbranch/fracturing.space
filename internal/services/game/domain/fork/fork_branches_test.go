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

func TestForkRequestValidate_RejectsWhitespaceSourceCampaignID(t *testing.T) {
	err := (ForkRequest{SourceCampaignID: "   "}).Validate()
	if !errors.Is(err, ErrEmptyCampaignID) {
		t.Fatalf("expected ErrEmptyCampaignID, got %v", err)
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

func TestCreateFork_UsesDefaultNowWhenNil(t *testing.T) {
	before := time.Now().UTC()
	result, err := CreateFork(
		CreateForkInput{SourceCampaignID: "camp-1"},
		"origin-1",
		1,
		nil,
		func() (string, error) { return "fork-1", nil },
	)
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("CreateFork() unexpected error: %v", err)
	}
	if result.NewCampaignID != "fork-1" {
		t.Fatalf("NewCampaignID = %q, want fork-1", result.NewCampaignID)
	}
	if result.CreatedAt.Before(before) || result.CreatedAt.After(after) {
		t.Fatalf("CreatedAt = %s, expected within [%s, %s]", result.CreatedAt, before, after)
	}
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
