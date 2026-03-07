package storage

import apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"

// ErrNotFound indicates a requested persistence record is missing.
// Callers use this to differentiate between legitimate "no such entity" states
// and transport or data corruption failures.
var ErrNotFound = apperrors.New(apperrors.CodeNotFound, "record not found")

// ErrActiveSessionExists indicates a command tried to start a second active session
// for the same campaign, which would violate the single-active-session domain rule.
var ErrActiveSessionExists = apperrors.New(apperrors.CodeActiveSessionExists, "active session already exists for campaign")
