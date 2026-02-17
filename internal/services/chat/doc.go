// Package chat implements real-time participation transport for active campaigns.
//
// It keeps WebSocket lifecycle, message sequencing, and fan-out isolated from
// domain logic so game/session services remain the source of truth for campaign
// state transitions.
package chat
