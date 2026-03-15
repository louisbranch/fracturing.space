// Package chat implements session-scoped realtime transcript transport.
//
// It keeps WebSocket lifecycle, message sequencing, and fan-out isolated from
// gameplay authority so game and auth remain the source of truth for membership
// and session identity.
package chat
