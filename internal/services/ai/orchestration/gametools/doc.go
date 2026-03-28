// Package gametools implements the orchestration.Session and
// orchestration.Dialer interfaces for direct game-service execution.
//
// It owns the concrete production tool registry, direct gRPC-backed tool
// session shell, generic resource URI dispatch, and non-system-specific tool
// execution used by campaign AI orchestration. Daggerheart-specific tool and
// resource execution is delegated to the sibling
// orchestration/daggerhearttools package through a narrow runtime seam.
package gametools
