// Package gamebridge owns AI-to-game collaborator policy.
//
// It wraps the specific game RPCs the AI service depends on so startup can
// construct one explicit boundary and downstream packages can depend on small
// consumption-point interfaces instead of generated gRPC clients.
package gamebridge
