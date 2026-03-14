// Package participanttransport owns protobuf mapping and enum conversions for
// participant-facing transport code.
//
// Keeping roster record and authorization enum mapping here gives campaign,
// participant, invite, communication, and authorization handlers a
// participant-owned import boundary instead of relying on root-package helper
// spillover.
package participanttransport
