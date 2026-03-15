package handler

// Standard pagination constants organized by tier.
//
// Tier rationale:
//   - PageSmall (10):  interactive list views where fewer items per page keep
//     response latency low and payloads compact (campaigns, participants, etc.).
//   - PageMedium (50): list views where the client can tolerate larger pages
//     (scenes, forks-max, events-default).
//   - PageLarge (200): bulk/background operations such as event replay, fork
//     event copying, and readiness pre-checks where throughput matters more
//     than payload size.
const (
	PageSmall  = 10
	PageMedium = 50
	PageLarge  = 200
)
