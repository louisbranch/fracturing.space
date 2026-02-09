# OAuth System

This document describes the OAuth surface owned by the auth service and the MCP protected resource integration.

## Goals

- Auth service acts as an OAuth authorization server for first-party clients (MCP, web tools).
- Auth service can act as an OAuth client to external providers (Google, GitHub) to bootstrap or link identities.
- MCP HTTP transport advertises resource metadata and validates access tokens via introspection.

## Boundaries

- **Auth service**: owns OAuth authorization, token issuance, and external provider login.
- **MCP service**: a protected resource that validates bearer tokens and advertises `.well-known` metadata.
- **Game service**: unchanged; consumes auth identity for join grants and permissions.

## OAuth Server (Auth Service)

Endpoints:

- `GET /authorize` + `POST /authorize/login` + `POST /authorize/consent`
- `POST /token`
- `POST /introspect` (opaque tokens, protected by `X-Resource-Secret`)
- `GET /.well-known/oauth-authorization-server`

Token model:

- Access tokens are opaque and stored in SQLite.
- MCP validates tokens using `/introspect`.

## OAuth Client (External Providers)

Endpoints:

- `GET /oauth/providers/{provider}/start`
- `GET /oauth/providers/{provider}/callback`

External identities are stored in the auth database and linked to internal users.

## MCP Protected Resource Metadata

Endpoint:

- `GET /.well-known/oauth-protected-resource`

The MCP HTTP server advertises its authorization server and includes
`WWW-Authenticate: Bearer resource_metadata=...` on 401 responses.

## Configuration (Env)

Auth service:

- `FRACTURING_SPACE_AUTH_HTTP_ADDR` (default `localhost:8084`): HTTP listen address for OAuth endpoints.
- `FRACTURING_SPACE_OAUTH_ISSUER`: Issuer base URL for OAuth metadata (should match the auth HTTP base URL).
- `FRACTURING_SPACE_OAUTH_RESOURCE_SECRET`: Shared secret required by `/introspect`.
- `FRACTURING_SPACE_OAUTH_CLIENTS`: JSON array of OAuth clients (id, redirect URIs, name, auth method).
- `FRACTURING_SPACE_OAUTH_USERS`: JSON array of bootstrap users (username, password, display name).
- `FRACTURING_SPACE_OAUTH_LOGIN_REDIRECTS`: Comma-separated allowlist for external provider redirect URIs.
- `FRACTURING_SPACE_OAUTH_GOOGLE_CLIENT_ID`: Google OAuth client ID.
- `FRACTURING_SPACE_OAUTH_GOOGLE_CLIENT_SECRET`: Google OAuth client secret.
- `FRACTURING_SPACE_OAUTH_GOOGLE_REDIRECT_URI`: Redirect URI registered with Google.
- `FRACTURING_SPACE_OAUTH_GOOGLE_SCOPES`: Comma-separated scopes for Google OAuth (defaults to `openid,email,profile`).
- `FRACTURING_SPACE_OAUTH_GITHUB_CLIENT_ID`: GitHub OAuth client ID.
- `FRACTURING_SPACE_OAUTH_GITHUB_CLIENT_SECRET`: GitHub OAuth client secret.
- `FRACTURING_SPACE_OAUTH_GITHUB_REDIRECT_URI`: Redirect URI registered with GitHub.
- `FRACTURING_SPACE_OAUTH_GITHUB_SCOPES`: Comma-separated scopes for GitHub OAuth (defaults to `read:user,user:email`).

MCP service:

- `FRACTURING_SPACE_MCP_OAUTH_ISSUER`: Auth server issuer used for introspection (expected to match `FRACTURING_SPACE_OAUTH_ISSUER`).
- `FRACTURING_SPACE_MCP_OAUTH_RESOURCE_SECRET`: Shared secret presented to `/introspect`.

## Example OAuth Client Config (JSON)

```json
[
  {
    "client_id": "claude-desktop",
    "redirect_uris": ["http://localhost:8081/oauth/callback"],
    "client_name": "Claude Desktop",
    "token_endpoint_auth_method": "none"
  }
]
```

## Example OAuth User Bootstrap (JSON)

```json
[
  {
    "username": "demo",
    "password": "change-me",
    "display_name": "Demo User"
  }
]
```
