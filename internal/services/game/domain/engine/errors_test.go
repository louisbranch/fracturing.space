package engine

import (
	"errors"
	"testing"
)

func TestPostPersistError_MetadataAndClassification(t *testing.T) {
	cause := errors.New("snapshot failed")
	err := newPostPersistError(PostPersistStageSnapshot, "camp-1", 42, cause)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("expected wrapped cause, got %v", err)
	}
	if !IsNonRetryable(err) {
		t.Fatal("post-persist errors must be non-retryable")
	}

	meta, ok := AsPostPersistError(err)
	if !ok {
		t.Fatal("expected AsPostPersistError to match")
	}
	if meta.Stage != PostPersistStageSnapshot {
		t.Fatalf("stage = %q, want %q", meta.Stage, PostPersistStageSnapshot)
	}
	if meta.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %q, want camp-1", meta.CampaignID)
	}
	if meta.LastSeq != 42 {
		t.Fatalf("last seq = %d, want 42", meta.LastSeq)
	}
}

func TestAsPostPersistError_NoMatch(t *testing.T) {
	if _, ok := AsPostPersistError(errors.New("boom")); ok {
		t.Fatal("expected no match")
	}
}

func TestPostPersistError_NilBranches(t *testing.T) {
	if err := newPostPersistError(PostPersistStageFold, "camp-1", 1, nil); err != nil {
		t.Fatalf("newPostPersistError(nil cause) = %v, want nil", err)
	}

	var post *PostPersistError
	if got := post.Error(); got != "" {
		t.Fatalf("nil post-persist error string = %q, want empty", got)
	}
	if unwrapped := post.Unwrap(); unwrapped != nil {
		t.Fatalf("nil post-persist unwrap = %v, want nil", unwrapped)
	}
}
