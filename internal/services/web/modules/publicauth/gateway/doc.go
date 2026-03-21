// Package gateway maps external protocols into publicauth app contracts.
//
// It should stay focused on auth-service protocol normalization and avoid
// owning route or page composition concerns. The root publicauth composition
// builds this gateway once and shares the resulting app services across the
// stable public route owners.
package gateway
