// Package auth provides authentication and authorization (FUTURE PLACEHOLDER).
//
// This package is a placeholder for future authN/authZ functionality.
//
// # Planned Features
//
// Authentication:
//   - API key authentication for MCP clients
//   - OAuth/OIDC integration for web clients
//   - Session token management
//
// Authorization:
//   - Role-based access control (GM vs Player)
//   - Campaign-scoped permissions
//   - Character sheet access control
//   - Action authorization (who can roll for whom)
//
// # Integration Points
//
// Auth will integrate with:
//   - gRPC interceptors for request authentication
//   - MCP tool handlers for client identification
//   - Storage layer for permission persistence
//   - Web handlers for session management
//
// # Status
//
// This package is not yet implemented. Current access is unauthenticated.
package auth
