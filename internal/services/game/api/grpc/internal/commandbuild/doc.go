// Package commandbuild builds canonical domain commands from validated
// transport inputs.
//
// Callers are gRPC handlers that have already passed transport validation
// (validate package) and loaded domain state. commandbuild converts
// transport-shaped fields into a command.Command envelope without
// performing additional validation — that responsibility belongs to the
// command.Registry at execution time.
package commandbuild
