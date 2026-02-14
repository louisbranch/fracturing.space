package campaign

import (
	"errors"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
)

func TestValidateCampaignOperation(t *testing.T) {
	tests := []struct {
		name    string
		status  CampaignStatus
		op      CampaignOperation
		allowed bool
	}{
		{name: "draft session start allowed", status: CampaignStatusDraft, op: CampaignOpSessionStart, allowed: true},
		{name: "draft read allowed", status: CampaignStatusDraft, op: CampaignOpRead, allowed: true},
		{name: "draft session action blocked", status: CampaignStatusDraft, op: CampaignOpSessionAction, allowed: false},
		{name: "draft mutate allowed", status: CampaignStatusDraft, op: CampaignOpCampaignMutate, allowed: true},
		{name: "draft archive blocked", status: CampaignStatusDraft, op: CampaignOpArchive, allowed: false},
		{name: "active session start allowed", status: CampaignStatusActive, op: CampaignOpSessionStart, allowed: true},
		{name: "active read allowed", status: CampaignStatusActive, op: CampaignOpRead, allowed: true},
		{name: "active session action allowed", status: CampaignStatusActive, op: CampaignOpSessionAction, allowed: true},
		{name: "active mutate allowed", status: CampaignStatusActive, op: CampaignOpCampaignMutate, allowed: true},
		{name: "completed session action blocked", status: CampaignStatusCompleted, op: CampaignOpSessionAction, allowed: false},
		{name: "completed read allowed", status: CampaignStatusCompleted, op: CampaignOpRead, allowed: true},
		{name: "completed mutate blocked", status: CampaignStatusCompleted, op: CampaignOpCampaignMutate, allowed: false},
		{name: "completed archive allowed", status: CampaignStatusCompleted, op: CampaignOpArchive, allowed: true},
		{name: "archived restore allowed", status: CampaignStatusArchived, op: CampaignOpRestore, allowed: true},
		{name: "archived read allowed", status: CampaignStatusArchived, op: CampaignOpRead, allowed: true},
		{name: "archived mutate blocked", status: CampaignStatusArchived, op: CampaignOpCampaignMutate, allowed: false},
		{name: "unspecified blocked", status: CampaignStatusUnspecified, op: CampaignOpCampaignMutate, allowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCampaignOperation(tt.status, tt.op)
			if tt.allowed && err != nil {
				t.Fatalf("expected allowed, got %v", err)
			}
			if !tt.allowed && err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidateCampaignOperationMetadata(t *testing.T) {
	err := ValidateCampaignOperation(CampaignStatusDraft, CampaignOpArchive)
	if err == nil {
		t.Fatal("expected error")
	}

	var domainErr *apperrors.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected domain error, got %T", err)
	}
	if domainErr.Code != apperrors.CodeCampaignStatusDisallowsOp {
		t.Fatalf("expected code %s, got %s", apperrors.CodeCampaignStatusDisallowsOp, domainErr.Code)
	}
	if domainErr.Metadata["Status"] != "DRAFT" {
		t.Fatalf("expected status metadata DRAFT, got %s", domainErr.Metadata["Status"])
	}
	if domainErr.Metadata["Operation"] != "ARCHIVE" {
		t.Fatalf("expected operation metadata ARCHIVE, got %s", domainErr.Metadata["Operation"])
	}
}

func TestCampaignLabelsFallback(t *testing.T) {
	if campaignStatusLabel(CampaignStatusUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected unspecified status label")
	}
	if campaignOperationLabel(CampaignOpUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected unspecified operation label")
	}
}

func TestValidateCampaignOperation_UnspecifiedOp(t *testing.T) {
	if err := ValidateCampaignOperation(CampaignStatusDraft, CampaignOpUnspecified); err == nil {
		t.Fatal("expected error for unspecified operation")
	}
}

func TestCampaignOperationLabels(t *testing.T) {
	labels := map[CampaignOperation]string{
		CampaignOpRead:           "READ",
		CampaignOpSessionStart:   "SESSION_START",
		CampaignOpSessionAction:  "SESSION_ACTION",
		CampaignOpCampaignMutate: "CAMPAIGN_MUTATE",
		CampaignOpEnd:            "END",
		CampaignOpArchive:        "ARCHIVE",
		CampaignOpRestore:        "RESTORE",
	}
	for op, label := range labels {
		if got := campaignOperationLabel(op); got != label {
			t.Fatalf("label for %v = %q, want %q", op, got, label)
		}
	}
}
