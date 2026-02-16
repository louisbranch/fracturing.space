package accessrequest

import (
	"errors"
	"testing"
	"time"
)

func TestNormalizeCreateInput(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateInput
		want    CreateInput
		wantErr error
	}{
		{
			name:    "missing requester user id",
			input:   CreateInput{OwnerUserID: "user-2", AgentID: "agent-1"},
			wantErr: ErrEmptyRequesterUserID,
		},
		{
			name:    "missing owner user id",
			input:   CreateInput{RequesterUserID: "user-1", AgentID: "agent-1"},
			wantErr: ErrEmptyOwnerUserID,
		},
		{
			name:    "missing agent id",
			input:   CreateInput{RequesterUserID: "user-1", OwnerUserID: "user-2"},
			wantErr: ErrEmptyAgentID,
		},
		{
			name:    "requester cannot request own agent",
			input:   CreateInput{RequesterUserID: "user-1", OwnerUserID: "user-1", AgentID: "agent-1"},
			wantErr: ErrRequesterIsOwner,
		},
		{
			name: "normalizes fields and defaults scope",
			input: CreateInput{
				RequesterUserID: " user-1 ",
				OwnerUserID:     " user-2 ",
				AgentID:         " agent-1 ",
				Scope:           " ",
				RequestNote:     " please allow invoke ",
			},
			want: CreateInput{
				RequesterUserID: "user-1",
				OwnerUserID:     "user-2",
				AgentID:         "agent-1",
				Scope:           ScopeInvoke,
				RequestNote:     "please allow invoke",
			},
		},
		{
			name: "rejects unsupported scope",
			input: CreateInput{
				RequesterUserID: "user-1",
				OwnerUserID:     "user-2",
				AgentID:         "agent-1",
				Scope:           "admin",
			},
			wantErr: ErrInvalidScope,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeCreateInput(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NormalizeCreateInput() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeCreateInput() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeCreateInput() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCreateAccessRequest(t *testing.T) {
	nowTime := time.Date(2026, 2, 16, 0, 40, 0, 0, time.UTC)
	got, err := Create(CreateInput{
		RequesterUserID: "user-1",
		OwnerUserID:     "user-2",
		AgentID:         "agent-1",
		Scope:           ScopeInvoke,
		RequestNote:     "please allow",
	}, func() time.Time { return nowTime }, func() (string, error) { return "request-1", nil })
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got.ID != "request-1" {
		t.Fatalf("id = %q, want %q", got.ID, "request-1")
	}
	if got.Status != StatusPending {
		t.Fatalf("status = %q, want %q", got.Status, StatusPending)
	}
	if got.Scope != ScopeInvoke {
		t.Fatalf("scope = %q, want %q", got.Scope, ScopeInvoke)
	}
	if got.CreatedAt != nowTime || got.UpdatedAt != nowTime {
		t.Fatalf("timestamps = (%s,%s), want %s", got.CreatedAt, got.UpdatedAt, nowTime)
	}
}

func TestReviewAccessRequest(t *testing.T) {
	nowTime := time.Date(2026, 2, 16, 0, 45, 0, 0, time.UTC)
	base := AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-2",
		AgentID:         "agent-1",
		Scope:           ScopeInvoke,
		Status:          StatusPending,
		CreatedAt:       nowTime.Add(-time.Minute),
		UpdatedAt:       nowTime.Add(-time.Minute),
	}

	got, err := Review(base, ReviewInput{
		ID:             "request-1",
		ReviewerUserID: "user-2",
		Decision:       DecisionApprove,
		ReviewNote:     "approved",
	}, func() time.Time { return nowTime })
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if got.Status != StatusApproved {
		t.Fatalf("status = %q, want %q", got.Status, StatusApproved)
	}
	if got.ReviewerUserID != "user-2" {
		t.Fatalf("reviewer_user_id = %q, want %q", got.ReviewerUserID, "user-2")
	}
	if got.ReviewedAt == nil || !got.ReviewedAt.Equal(nowTime) {
		t.Fatalf("reviewed_at = %v, want %v", got.ReviewedAt, nowTime)
	}
	if got.ReviewNote != "approved" {
		t.Fatalf("review_note = %q, want %q", got.ReviewNote, "approved")
	}

	_, err = Review(got, ReviewInput{
		ID:             "request-1",
		ReviewerUserID: "user-2",
		Decision:       DecisionDeny,
	}, func() time.Time { return nowTime.Add(time.Minute) })
	if !errors.Is(err, ErrNotPending) {
		t.Fatalf("second Review() error = %v, want %v", err, ErrNotPending)
	}
}

func TestRevokeAccessRequest(t *testing.T) {
	nowTime := time.Date(2026, 2, 16, 0, 50, 0, 0, time.UTC)
	base := AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-2",
		AgentID:         "agent-1",
		Scope:           ScopeInvoke,
		Status:          StatusApproved,
		ReviewerUserID:  "user-2",
		ReviewNote:      "approved earlier",
		CreatedAt:       nowTime.Add(-2 * time.Minute),
		UpdatedAt:       nowTime.Add(-time.Minute),
		ReviewedAt:      ptrTime(nowTime.Add(-time.Minute)),
	}

	got, err := Revoke(base, RevokeInput{
		ID:            "request-1",
		RevokerUserID: "user-2",
		RevokeNote:    "access removed",
	}, func() time.Time { return nowTime })
	if err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if got.Status != StatusRevoked {
		t.Fatalf("status = %q, want %q", got.Status, StatusRevoked)
	}
	if got.ReviewerUserID != "user-2" {
		t.Fatalf("reviewer_user_id = %q, want %q", got.ReviewerUserID, "user-2")
	}
	if got.ReviewNote != "access removed" {
		t.Fatalf("review_note = %q, want %q", got.ReviewNote, "access removed")
	}
	if !got.UpdatedAt.Equal(nowTime) {
		t.Fatalf("updated_at = %v, want %v", got.UpdatedAt, nowTime)
	}

	_, err = Revoke(got, RevokeInput{
		ID:            "request-1",
		RevokerUserID: "user-2",
	}, func() time.Time { return nowTime.Add(time.Minute) })
	if !errors.Is(err, ErrNotApproved) {
		t.Fatalf("second Revoke() error = %v, want %v", err, ErrNotApproved)
	}

	_, err = Revoke(base, RevokeInput{
		ID:            "request-1",
		RevokerUserID: "user-3",
	}, func() time.Time { return nowTime })
	if !errors.Is(err, ErrReviewerNotOwner) {
		t.Fatalf("non-owner Revoke() error = %v, want %v", err, ErrReviewerNotOwner)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
