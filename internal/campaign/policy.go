package campaign

import (
	"fmt"

	apperrors "github.com/louisbranch/fracturing.space/internal/errors"
)

// CampaignOperation describes a category of campaign operation for policy checks.
type CampaignOperation int

const (
	// CampaignOpUnspecified represents an invalid operation.
	CampaignOpUnspecified CampaignOperation = iota
	// CampaignOpRead represents read-only operations.
	// This is intentionally allowed for all statuses today, but will expand to enforce
	// user-scoped read capabilities (e.g., restricting which character sheets a player can view).
	CampaignOpRead
	// CampaignOpSessionStart represents starting a session.
	CampaignOpSessionStart
	// CampaignOpSessionAction represents session action rolls and outcomes.
	CampaignOpSessionAction
	// CampaignOpCampaignMutate represents campaign-level mutations.
	CampaignOpCampaignMutate
	// CampaignOpEnd represents ending a campaign.
	CampaignOpEnd
	// CampaignOpArchive represents archiving a campaign.
	CampaignOpArchive
	// CampaignOpRestore represents restoring a campaign.
	CampaignOpRestore
)

var (
	// ErrCampaignStatusDisallowsOperation indicates a status that disallows the requested operation.
	ErrCampaignStatusDisallowsOperation = apperrors.New(apperrors.CodeCampaignStatusDisallowsOp, "campaign status does not allow operation")
)

// ValidateCampaignOperation ensures the campaign status allows the requested operation.
func ValidateCampaignOperation(status CampaignStatus, op CampaignOperation) error {
	if op == CampaignOpUnspecified {
		return newStatusOpError(status, op)
	}
	if op == CampaignOpRead {
		return nil
	}

	switch status {
	case CampaignStatusDraft:
		switch op {
		case CampaignOpSessionStart, CampaignOpCampaignMutate:
			return nil
		case CampaignOpRestore:
			return newStatusOpError(status, op)
		case CampaignOpEnd, CampaignOpArchive, CampaignOpSessionAction:
			return newStatusOpError(status, op)
		default:
			return newStatusOpError(status, op)
		}
	case CampaignStatusActive:
		switch op {
		case CampaignOpSessionStart, CampaignOpSessionAction, CampaignOpCampaignMutate, CampaignOpEnd, CampaignOpArchive:
			return nil
		case CampaignOpRestore:
			return newStatusOpError(status, op)
		default:
			return newStatusOpError(status, op)
		}
	case CampaignStatusCompleted:
		switch op {
		case CampaignOpArchive:
			return nil
		case CampaignOpRestore, CampaignOpEnd, CampaignOpSessionStart, CampaignOpSessionAction, CampaignOpCampaignMutate:
			return newStatusOpError(status, op)
		default:
			return newStatusOpError(status, op)
		}
	case CampaignStatusArchived:
		switch op {
		case CampaignOpRestore:
			return nil
		case CampaignOpArchive, CampaignOpEnd, CampaignOpSessionStart, CampaignOpSessionAction, CampaignOpCampaignMutate:
			return newStatusOpError(status, op)
		default:
			return newStatusOpError(status, op)
		}
	default:
		return newStatusOpError(status, op)
	}
}

// newStatusOpError creates a structured error for disallowed status/operation combinations.
func newStatusOpError(status CampaignStatus, op CampaignOperation) *apperrors.Error {
	statusLabel := campaignStatusLabel(status)
	opLabel := campaignOperationLabel(op)
	return apperrors.WithMetadata(
		apperrors.CodeCampaignStatusDisallowsOp,
		fmt.Sprintf("campaign status %s does not allow operation %s", statusLabel, opLabel),
		map[string]string{"Status": statusLabel, "Operation": opLabel},
	)
}

func campaignOperationLabel(op CampaignOperation) string {
	switch op {
	case CampaignOpRead:
		return "READ"
	case CampaignOpSessionStart:
		return "SESSION_START"
	case CampaignOpSessionAction:
		return "SESSION_ACTION"
	case CampaignOpCampaignMutate:
		return "CAMPAIGN_MUTATE"
	case CampaignOpEnd:
		return "END"
	case CampaignOpArchive:
		return "ARCHIVE"
	case CampaignOpRestore:
		return "RESTORE"
	default:
		return "UNSPECIFIED"
	}
}
