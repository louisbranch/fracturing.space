package fork

import (
	"errors"
	"testing"
	"time"
)

func TestForkRequestValidate_RequiresSourceCampaignID(t *testing.T) {
	if err := (ForkRequest{}).Validate(); !errors.Is(err, ErrEmptyCampaignID) {
		t.Fatalf("expected ErrEmptyCampaignID, got %v", err)
	}
}

func TestCreateFork_DefaultsOriginAndNormalizesCreatedAt(t *testing.T) {
	now := time.Date(2026, 2, 1, 10, 11, 12, 0, time.FixedZone("local", -5*3600))
	got, err := CreateFork(
		CreateForkInput{SourceCampaignID: "camp-1"},
		"",
		42,
		func() time.Time { return now },
		func() (string, error) { return "fork-1", nil },
	)
	if err != nil {
		t.Fatalf("CreateFork: %v", err)
	}
	if got.SourceCampaignID != "camp-1" {
		t.Fatalf("SourceCampaignID = %q, want camp-1", got.SourceCampaignID)
	}
	if got.NewCampaignID != "fork-1" {
		t.Fatalf("NewCampaignID = %q, want fork-1", got.NewCampaignID)
	}
	if got.OriginCampaignID != "camp-1" {
		t.Fatalf("OriginCampaignID = %q, want camp-1", got.OriginCampaignID)
	}
	if got.ForkEventSeq != 42 {
		t.Fatalf("ForkEventSeq = %d, want 42", got.ForkEventSeq)
	}
	if !got.CreatedAt.Equal(now.UTC()) {
		t.Fatalf("CreatedAt = %s, want %s", got.CreatedAt, now.UTC())
	}
}

func TestCreateFork_PropagatesIDGenerationError(t *testing.T) {
	want := errors.New("boom")
	_, err := CreateFork(
		CreateForkInput{SourceCampaignID: "camp-1"},
		"origin-1",
		1,
		time.Now,
		func() (string, error) { return "", want },
	)
	if err == nil || !errors.Is(err, want) {
		t.Fatalf("expected wrapped id generation error, got %v", err)
	}
}

func TestLineageIsOriginal(t *testing.T) {
	if !((Lineage{CampaignID: "camp-1"}).IsOriginal()) {
		t.Fatal("expected original lineage when ParentCampaignID is empty")
	}
	if (Lineage{CampaignID: "camp-2", ParentCampaignID: "camp-1"}).IsOriginal() {
		t.Fatal("expected non-original lineage when ParentCampaignID is set")
	}
}
