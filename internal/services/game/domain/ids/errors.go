package ids

import "errors"

// ErrCampaignIDRequired indicates a missing campaign id.
//
// Defined here because campaign id validation is a cross-cutting concern
// used by event, command, replay, journal, and checkpoint packages — all
// of which already import ids.
var ErrCampaignIDRequired = errors.New("campaign id is required")
