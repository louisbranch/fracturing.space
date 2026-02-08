package snapshot

import apperrors "github.com/louisbranch/fracturing.space/internal/errors"

const (
	// GmFearMax is the maximum GM fear cap.
	// TODO: move to a config file.
	GmFearMax = 12
)

var (
	// ErrInvalidGMFearAmount indicates a non-positive fear mutation amount.
	ErrInvalidGMFearAmount = apperrors.New(apperrors.CodeSnapshotInvalidGMFear, "gm fear amount must be greater than zero")
	// ErrInsufficientGMFear indicates the campaign has too little fear to spend.
	ErrInsufficientGMFear = apperrors.New(apperrors.CodeSnapshotInsufficientFear, "gm fear is insufficient")
	// ErrGMFearExceedsCap indicates the campaign fear would exceed the maximum.
	ErrGMFearExceedsCap = apperrors.New(apperrors.CodeSnapshotGMFearExceedsCap, "gm fear exceeds cap")
)

// GmFear represents the GM's fear resource for a campaign.
// This value is stored in snapshot projections derived from events.
type GmFear struct {
	CampaignID string
	Value      int
}

// ApplyGmFearGain returns a GmFear with increased value.
// Amount must be greater than zero.
func ApplyGmFearGain(fear GmFear, amount int) (GmFear, int, int, error) {
	if amount <= 0 {
		return GmFear{}, 0, 0, ErrInvalidGMFearAmount
	}
	before := fear.Value
	after := before + amount
	if after > GmFearMax {
		return GmFear{}, 0, 0, ErrGMFearExceedsCap
	}
	updated := fear
	updated.Value = after
	return updated, before, updated.Value, nil
}

// ApplyGmFearSpend returns a GmFear with reduced value.
// Amount must be greater than zero and cannot exceed the current fear.
func ApplyGmFearSpend(fear GmFear, amount int) (GmFear, int, int, error) {
	if amount <= 0 {
		return GmFear{}, 0, 0, ErrInvalidGMFearAmount
	}
	if fear.Value < amount {
		return GmFear{}, 0, 0, ErrInsufficientGMFear
	}
	before := fear.Value
	updated := fear
	updated.Value = before - amount
	return updated, before, updated.Value, nil
}
