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
		// Missing coverage: explicit branches
		{name: "draft restore blocked", status: CampaignStatusDraft, op: CampaignOpRestore, allowed: false},
		{name: "draft end blocked", status: CampaignStatusDraft, op: CampaignOpEnd, allowed: false},
		{name: "active end allowed", status: CampaignStatusActive, op: CampaignOpEnd, allowed: true},
		{name: "active archive allowed", status: CampaignStatusActive, op: CampaignOpArchive, allowed: true},
		{name: "active restore blocked", status: CampaignStatusActive, op: CampaignOpRestore, allowed: false},
		{name: "completed restore blocked", status: CampaignStatusCompleted, op: CampaignOpRestore, allowed: false},
		{name: "completed end blocked", status: CampaignStatusCompleted, op: CampaignOpEnd, allowed: false},
		{name: "completed session start blocked", status: CampaignStatusCompleted, op: CampaignOpSessionStart, allowed: false},
		{name: "archived archive blocked", status: CampaignStatusArchived, op: CampaignOpArchive, allowed: false},
		{name: "archived end blocked", status: CampaignStatusArchived, op: CampaignOpEnd, allowed: false},
		{name: "archived session start blocked", status: CampaignStatusArchived, op: CampaignOpSessionStart, allowed: false},
		{name: "archived session action blocked", status: CampaignStatusArchived, op: CampaignOpSessionAction, allowed: false},
		// Unknown operation hits inner default case
		{name: "draft unknown op blocked", status: CampaignStatusDraft, op: CampaignOperation(99), allowed: false},
		{name: "active unknown op blocked", status: CampaignStatusActive, op: CampaignOperation(99), allowed: false},
		{name: "completed unknown op blocked", status: CampaignStatusCompleted, op: CampaignOperation(99), allowed: false},
		{name: "archived unknown op blocked", status: CampaignStatusArchived, op: CampaignOperation(99), allowed: false},
		// Invalid status hits outer default case
		{name: "invalid status read allowed", status: CampaignStatus(99), op: CampaignOpRead, allowed: true},
		{name: "invalid status non-read blocked", status: CampaignStatus(99), op: CampaignOpCampaignMutate, allowed: false},
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
