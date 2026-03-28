package engine

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"

// CoreDomain aliases the aggregate-owned core-domain registration descriptor.
//
// The aggregate package owns the authoritative built-in core-domain inventory
// because that inventory must stay aligned with fold dispatch wiring. Engine
// startup and validators consume those registrations directly through this
// alias instead of maintaining an engine-local inventory wrapper.
type CoreDomain = aggregate.CoreDomainRegistration
