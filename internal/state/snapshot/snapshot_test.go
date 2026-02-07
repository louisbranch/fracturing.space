package snapshot

import "testing"

func TestNewSnapshot(t *testing.T) {
	s := NewSnapshot("camp-123")

	if s.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", s.CampaignID)
	}
	if s.GmFear.CampaignID != "camp-123" {
		t.Fatalf("expected gm fear campaign id camp-123, got %q", s.GmFear.CampaignID)
	}
	if s.GmFear.Value != 0 {
		t.Fatalf("expected gm fear value 0, got %d", s.GmFear.Value)
	}
}

func TestSnapshotSetGmFear(t *testing.T) {
	s := NewSnapshot("camp-123")
	if s.GmFear.Value != 0 {
		t.Fatalf("expected initial gm fear 0, got %d", s.GmFear.Value)
	}

	s.SetGmFear(5)

	if s.GmFear.Value != 5 {
		t.Fatalf("expected gm fear 5, got %d", s.GmFear.Value)
	}
	if s.GmFear.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id preserved, got %q", s.GmFear.CampaignID)
	}
}

func TestSnapshotSetGmFearMultiple(t *testing.T) {
	s := NewSnapshot("camp-123")

	s.SetGmFear(3)
	s.SetGmFear(7)
	s.SetGmFear(2)

	if s.GmFear.Value != 2 {
		t.Fatalf("expected gm fear 2, got %d", s.GmFear.Value)
	}
}
