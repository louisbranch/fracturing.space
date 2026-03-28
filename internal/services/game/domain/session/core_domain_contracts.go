package session

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/coredomain"

// CoreDomainContracts returns the package-owned core-domain registration
// surface consumed by aggregate replay and engine bootstrap.
func CoreDomainContracts() coredomain.Contracts {
	return coredomain.Contracts{
		DomainName:             "session",
		RegisterCommands:       RegisterCommands,
		RegisterEvents:         RegisterEvents,
		EmittableEventTypes:    EmittableEventTypes,
		FoldHandledTypes:       FoldHandledTypes,
		DeciderHandledCommands: DeciderHandledCommands,
		ProjectionHandledTypes: ProjectionHandledTypes,
		RejectionCodes:         RejectionCodes,
	}
}
