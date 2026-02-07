package fork

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/errors"
)

func TestForkRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request ForkRequest
		wantErr error
	}{
		{
			name: "valid request",
			request: ForkRequest{
				SourceCampaignID: "camp-1",
				ForkPoint:        ForkPoint{EventSeq: 10},
			},
			wantErr: nil,
		},
		{
			name: "valid request with session boundary",
			request: ForkRequest{
				SourceCampaignID: "camp-1",
				ForkPoint:        ForkPoint{SessionID: "sess-1"},
			},
			wantErr: nil,
		},
		{
			name:    "empty source campaign ID",
			request: ForkRequest{},
			wantErr: ErrEmptyCampaignID,
		},
		{
			name: "whitespace source campaign ID",
			request: ForkRequest{
				SourceCampaignID: "   ",
			},
			wantErr: ErrEmptyCampaignID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr == nil && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr != nil && !errors.IsCode(err, errors.Code(tt.wantErr.(*errors.Error).Code)) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestForkPoint_IsSessionBoundary(t *testing.T) {
	tests := []struct {
		name      string
		forkPoint ForkPoint
		want      bool
	}{
		{
			name:      "event sequence only",
			forkPoint: ForkPoint{EventSeq: 10},
			want:      false,
		},
		{
			name:      "session boundary",
			forkPoint: ForkPoint{SessionID: "sess-1"},
			want:      true,
		},
		{
			name:      "both set - session takes precedence",
			forkPoint: ForkPoint{EventSeq: 10, SessionID: "sess-1"},
			want:      true,
		},
		{
			name:      "zero values",
			forkPoint: ForkPoint{},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.forkPoint.IsSessionBoundary(); got != tt.want {
				t.Errorf("IsSessionBoundary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateFork(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	fixedNow := func() time.Time { return fixedTime }
	fixedID := func() (string, error) { return "new-camp-id", nil }

	tests := []struct {
		name             string
		input            CreateForkInput
		originCampaignID string
		forkEventSeq     uint64
		wantErr          bool
	}{
		{
			name: "creates fork with existing origin",
			input: CreateForkInput{
				SourceCampaignID: "camp-1",
				ForkPoint:        ForkPoint{EventSeq: 50},
			},
			originCampaignID: "origin-camp",
			forkEventSeq:     50,
			wantErr:          false,
		},
		{
			name: "creates fork from original (no origin)",
			input: CreateForkInput{
				SourceCampaignID: "camp-1",
				ForkPoint:        ForkPoint{EventSeq: 50},
			},
			originCampaignID: "",
			forkEventSeq:     50,
			wantErr:          false,
		},
		{
			name: "empty source campaign ID",
			input: CreateForkInput{
				SourceCampaignID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fork, err := CreateFork(tt.input, tt.originCampaignID, tt.forkEventSeq, fixedNow, fixedID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fork.SourceCampaignID != tt.input.SourceCampaignID {
				t.Errorf("SourceCampaignID = %s, want %s", fork.SourceCampaignID, tt.input.SourceCampaignID)
			}
			if fork.NewCampaignID != "new-camp-id" {
				t.Errorf("NewCampaignID = %s, want new-camp-id", fork.NewCampaignID)
			}
			if fork.ForkEventSeq != tt.forkEventSeq {
				t.Errorf("ForkEventSeq = %d, want %d", fork.ForkEventSeq, tt.forkEventSeq)
			}
			if !fork.CreatedAt.Equal(fixedTime) {
				t.Errorf("CreatedAt = %v, want %v", fork.CreatedAt, fixedTime)
			}

			expectedOrigin := tt.originCampaignID
			if expectedOrigin == "" {
				expectedOrigin = tt.input.SourceCampaignID
			}
			if fork.OriginCampaignID != expectedOrigin {
				t.Errorf("OriginCampaignID = %s, want %s", fork.OriginCampaignID, expectedOrigin)
			}
		})
	}
}

func TestLineage_IsOriginal(t *testing.T) {
	tests := []struct {
		name    string
		lineage Lineage
		want    bool
	}{
		{
			name: "original campaign",
			lineage: Lineage{
				CampaignID:       "camp-1",
				ParentCampaignID: "",
				OriginCampaignID: "camp-1",
				Depth:            0,
			},
			want: true,
		},
		{
			name: "forked campaign",
			lineage: Lineage{
				CampaignID:       "camp-2",
				ParentCampaignID: "camp-1",
				ForkEventSeq:     50,
				OriginCampaignID: "camp-1",
				Depth:            1,
			},
			want: false,
		},
		{
			name: "deeply forked campaign",
			lineage: Lineage{
				CampaignID:       "camp-3",
				ParentCampaignID: "camp-2",
				ForkEventSeq:     100,
				OriginCampaignID: "camp-1",
				Depth:            2,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.lineage.IsOriginal(); got != tt.want {
				t.Errorf("IsOriginal() = %v, want %v", got, tt.want)
			}
		})
	}
}
