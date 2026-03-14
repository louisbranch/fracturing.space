package game

// Standard pagination constants organized by tier.
//
// Tier rationale:
//   - pageSmall (10):  interactive list views where fewer items per page keep
//     response latency low and payloads compact (campaigns, participants, etc.).
//   - pageMedium (50): list views where the client can tolerate larger pages
//     (scenes, forks-max, events-default).
//   - pageLarge (200): bulk/background operations such as event replay, fork
//     event copying, and readiness pre-checks where throughput matters more
//     than payload size.
const (
	pageSmall  = 10
	pageMedium = 50
	pageLarge  = 200
)
