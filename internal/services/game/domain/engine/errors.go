package engine

import "errors"

// PostPersistStage identifies which post-append step failed.
type PostPersistStage string

const (
	// PostPersistStageFold indicates in-memory fold failed after append.
	PostPersistStageFold PostPersistStage = "fold"
	// PostPersistStageSnapshot indicates snapshot persistence failed after append.
	PostPersistStageSnapshot PostPersistStage = "snapshot"
	// PostPersistStageCheckpoint indicates checkpoint persistence failed after append.
	PostPersistStageCheckpoint PostPersistStage = "checkpoint"
)

// PostPersistError captures failures that happen after events were already
// appended to the authoritative journal.
//
// This gives callers structured partial-success metadata so retries can be
// suppressed safely while still exposing what stage failed.
type PostPersistError struct {
	Stage      PostPersistStage
	CampaignID string
	LastSeq    uint64
	err        error
}

func (e *PostPersistError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *PostPersistError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// NonRetryable returns true from IsNonRetryable checks.
func (e *PostPersistError) NonRetryable() bool { return true }

func newPostPersistError(stage PostPersistStage, campaignID string, lastSeq uint64, err error) error {
	if err == nil {
		return nil
	}
	return &PostPersistError{
		Stage:      stage,
		CampaignID: campaignID,
		LastSeq:    lastSeq,
		err:        err,
	}
}

// AsPostPersistError extracts structured post-persist metadata from err.
func AsPostPersistError(err error) (*PostPersistError, bool) {
	var target *PostPersistError
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// IsNonRetryable returns true when the error (or any error in its chain)
// signals that the operation must not be retried. Use this in transport
// middleware to prevent accidental duplicate event creation after
// post-persist fold failures.
func IsNonRetryable(err error) bool {
	var target interface{ NonRetryable() bool }
	if errors.As(err, &target) {
		return target.NonRetryable()
	}
	return false
}
