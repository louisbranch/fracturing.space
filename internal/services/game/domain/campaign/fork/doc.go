// Package fork provides campaign forking capabilities.
//
// Forking allows creating a new campaign that branches from an existing
// campaign's timeline at a specific point. This enables use cases like:
//
//   - Preset campaigns that others can fork and play
//   - Rewinding from disasters (e.g., TPK) by forking before the event
//   - "What if" scenarios by exploring alternate timelines
//
// # Current Implementation
//
// A fork creates a new campaign and records its relationship to the parent:
//   - parent_campaign_id: The immediate parent campaign
//   - fork_event_seq: The event sequence at which the fork occurred
//   - origin_campaign_id: The root of the lineage (for deep fork chains)
//
// The forked campaign is created via the event journal and then replays source
// campaign events up to the fork point to rebuild projections. Fork points can
// be defined by event sequence or by an ended session boundary. Participant
// events can be omitted if copy_participants is false, and the fork emits a
// campaign.forked event in the new campaign journal. When forking at the latest
// event and snapshot projections are already available, the fork can seed
// snapshot projections (GM fear, character states) and skip re-applying snapshot
// events.
//
// # Future: Snapshot-accelerated replay
//
// State reconstruction will eventually use snapshots plus event replay:
//  1. Find the nearest snapshot before the target point
//  2. Replay events from the snapshot to the target point
//
// Snapshots captured at event sequences can bound replay to fewer events.
package fork
