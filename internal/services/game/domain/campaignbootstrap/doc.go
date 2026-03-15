// Package campaignbootstrap owns the one intentional campaign bootstrap
// workflow that emits events across aggregate boundaries.
//
// The campaign aggregate itself still owns campaign-local create/update/lifecycle
// decisions. This sibling package exists only for the cross-aggregate
// `campaign.create_with_participants` path, which must emit one
// `campaign.created` event and one `participant.joined` event per bootstrap
// participant in a single atomic decision.
package campaignbootstrap
