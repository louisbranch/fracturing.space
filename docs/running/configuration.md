---
title: "Configuration"
parent: "Running"
nav_order: 5
status: canonical
owner: engineering
last_reviewed: "2026-02-26"
---

# Configuration

This is the canonical runtime configuration reference for contributor and
operator workflows. Other docs should link here instead of restating defaults.

For setup steps, see [quickstart](quickstart.md) or
[local development](local-dev.md).

## Environment variables

### Game

- `FRACTURING_SPACE_GAME_EVENTS_DB_PATH`: event journal SQLite path. Default: `data/game-events.db`.
- `FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH`: projections SQLite path. Default: `data/game-projections.db`.
- `FRACTURING_SPACE_GAME_CONTENT_DB_PATH`: content SQLite path. Default: `data/game-content.db`.
- `FRACTURING_SPACE_GAME_DOMAIN_ENABLED`: enable domain-engine write path. Default: `true`.
- `FRACTURING_SPACE_GAME_COMPATIBILITY_APPEND_ENABLED`: allow direct compatibility `EventService.AppendEvent` fallback when domain is disabled. Default: `false`.
- `FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_ENABLED`: enqueue projection-apply outbox rows on append. Default: `false`.
- `FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_SHADOW_WORKER_ENABLED`: enable outbox shadow worker (requires outbox enabled). Default: `false`.
- `FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_WORKER_ENABLED`: enable outbox apply worker (requires outbox enabled). Default: `false`.
- `FRACTURING_SPACE_GAME_EVENT_HMAC_KEY`: root secret used to sign event chain hashes. Required. Generate with `go run ./cmd/hmac-key`.
- `FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS`: optional comma-separated key ring (`key_id=secret`).
- `FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID`: active key id when using the key ring. Default: `v1`.

### Auth + OAuth

- `FRACTURING_SPACE_AUTH_DB_PATH`: auth SQLite path. Default: `data/auth.db`.
- `FRACTURING_SPACE_AUTH_PORT`: gRPC port for auth service. Default: `8083`.
- `FRACTURING_SPACE_AUTH_HTTP_ADDR`: HTTP bind address for OAuth endpoints. Default: `localhost:8084`.
- `FRACTURING_SPACE_OAUTH_ISSUER`: external OAuth issuer URL. Defaults to the auth HTTP address when unset.
- `FRACTURING_SPACE_OAUTH_LOGIN_UI_URL`: external login UI URL for redirects (web login server).
- `FRACTURING_SPACE_OAUTH_LOGIN_REDIRECTS`: comma-separated list of allowed login redirect URLs.
- `FRACTURING_SPACE_OAUTH_CLIENTS`: JSON array of registered OAuth clients.
- `FRACTURING_SPACE_OAUTH_RESOURCE_SECRET`: shared secret for resource introspection.
- `FRACTURING_SPACE_OAUTH_TOKEN_TTL`: OAuth access-token TTL. Default: `1h`.
- `FRACTURING_SPACE_OAUTH_CODE_TTL`: OAuth authorization-code TTL. Default: `10m`.
- `FRACTURING_SPACE_OAUTH_PENDING_TTL`: pending authorization TTL for browser login handoff. Default: `15m`.
- `FRACTURING_SPACE_OAUTH_FIRST_PARTY_CLIENT_ID`: client ID for the first-party web login client. When set (along with redirect URI), registers a trusted OAuth client that skips the consent screen. Default: unset.
- `FRACTURING_SPACE_OAUTH_FIRST_PARTY_REDIRECT_URI`: redirect URI for the first-party web login client. Required together with the client ID.
- `FRACTURING_SPACE_OAUTH_GOOGLE_CLIENT_ID`: Google OAuth client ID.
- `FRACTURING_SPACE_OAUTH_GOOGLE_CLIENT_SECRET`: Google OAuth client secret.
- `FRACTURING_SPACE_OAUTH_GOOGLE_REDIRECT_URI`: Google OAuth redirect URI.
- `FRACTURING_SPACE_OAUTH_GOOGLE_SCOPES`: comma-separated Google scopes. Default when provider is configured but unset: `openid,email,profile`.
- `FRACTURING_SPACE_OAUTH_GITHUB_CLIENT_ID`: GitHub OAuth client ID.
- `FRACTURING_SPACE_OAUTH_GITHUB_CLIENT_SECRET`: GitHub OAuth client secret.
- `FRACTURING_SPACE_OAUTH_GITHUB_REDIRECT_URI`: GitHub OAuth redirect URI.
- `FRACTURING_SPACE_OAUTH_GITHUB_SCOPES`: comma-separated GitHub scopes. Default when provider is configured but unset: `read:user,user:email`.
- `FRACTURING_SPACE_MAGIC_LINK_BASE_URL`: magic-link base URL. Default: `http://localhost:8086/magic`.
- `FRACTURING_SPACE_MAGIC_LINK_TTL`: magic-link TTL. Default: `15m`.

### AI

- `FRACTURING_SPACE_AI_PORT`: gRPC port for AI service. Default: `8087`.
- `FRACTURING_SPACE_AI_DB_PATH`: AI SQLite path. Default: `data/ai.db`.
- `FRACTURING_SPACE_AI_ENCRYPTION_KEY`: base64-encoded AES key used to encrypt provider secrets at rest (must decode to 16/24/32 bytes).

### Notifications

- `FRACTURING_SPACE_NOTIFICATIONS_PORT`: gRPC port for notifications service. Default: `8088`.
- `FRACTURING_SPACE_NOTIFICATIONS_DB_PATH`: notifications SQLite path. Default: `data/notifications.db`.
- `FRACTURING_SPACE_NOTIFICATIONS_EMAIL_DELIVERY_ENABLED`: enable dispatch attempts to an email sender implementation. Default: `false`.
- `FRACTURING_SPACE_NOTIFICATIONS_EMAIL_DELIVERY_WORKER_ENABLED`: enable background observation of pending email deliveries. Default: `false`.
- `FRACTURING_SPACE_NOTIFICATIONS_EMAIL_DELIVERY_WORKER_POLL_INTERVAL`: poll cadence for pending email delivery checks. Default: `5s`.

Notifications channel routing is service-owned by `message_type`; callers only create intents.
The onboarding welcome message type (`auth.onboarding.welcome`) is email-only and does not surface in in-app inbox reads.
User-configurable per-message-type delivery preferences are planned but not yet available.

### User hub

- `FRACTURING_SPACE_USERHUB_PORT`: gRPC port for userhub service. Default: `8092`.
- `FRACTURING_SPACE_USERHUB_GAME_ADDR`: game gRPC dependency address. Default: `game:8082`.
- `FRACTURING_SPACE_USERHUB_SOCIAL_ADDR`: social gRPC dependency address. Default: `social:8090`.
- `FRACTURING_SPACE_USERHUB_NOTIFICATIONS_ADDR`: notifications gRPC dependency address. Default: `notifications:8088`.
- `FRACTURING_SPACE_USERHUB_CACHE_FRESH_TTL`: fresh response cache TTL for dashboard aggregation. Default: `15s`.
- `FRACTURING_SPACE_USERHUB_CACHE_STALE_TTL`: stale fallback cache TTL for dependency degradation. Default: `2m`.
- `FRACTURING_SPACE_USERHUB_DIAL_TIMEOUT`: gRPC dependency dial timeout. Default: `2s`.

### Worker

- `FRACTURING_SPACE_WORKER_PORT`: gRPC port for worker health endpoint. Default: `8089`.
- `FRACTURING_SPACE_WORKER_AUTH_ADDR`: auth gRPC dependency address. Default: `auth:8083`.
- `FRACTURING_SPACE_WORKER_SOCIAL_ADDR`: social gRPC dependency address. Default: `social:8090`.
- `FRACTURING_SPACE_WORKER_NOTIFICATIONS_ADDR`: notifications gRPC dependency address. Default: `notifications:8088`.
- `FRACTURING_SPACE_WORKER_DB_PATH`: worker SQLite path for durable attempt logs. Default: `data/worker.db`.
- `FRACTURING_SPACE_WORKER_CONSUMER`: auth outbox consumer identifier. Default: `worker-onboarding`.
- `FRACTURING_SPACE_WORKER_POLL_INTERVAL`: auth outbox poll interval. Default: `2s`.
- `FRACTURING_SPACE_WORKER_LEASE_TTL`: auth outbox lease duration. Default: `30s`.
- `FRACTURING_SPACE_WORKER_MAX_ATTEMPTS`: max processing attempts before dead-letter. Default: `8`.
- `FRACTURING_SPACE_WORKER_RETRY_BACKOFF`: base retry delay before exponential backoff. Default: `5s`.
- `FRACTURING_SPACE_WORKER_RETRY_MAX_DELAY`: upper bound for retry delay growth. Default: `5m`.
- `FRACTURING_SPACE_WORKER_DIAL_TIMEOUT`: gRPC dependency dial timeout. Default: `2s`.

### WebAuthn / Passkeys

- `FRACTURING_SPACE_WEBAUTHN_RP_ID`: WebAuthn relying party ID (the domain the user sees). Default: `localhost`.
- `FRACTURING_SPACE_WEBAUTHN_RP_DISPLAY_NAME`: display name shown during passkey prompts. Defaults to the app name.
- `FRACTURING_SPACE_WEBAUTHN_RP_ORIGINS`: comma-separated list of allowed WebAuthn origins. Runtime default: `http://localhost:8086`.
- `FRACTURING_SPACE_WEBAUTHN_SESSION_TTL`: passkey session TTL. Default: `5m`.

For web-login-first local flows, many contributors set
`FRACTURING_SPACE_WEBAUTHN_RP_ORIGINS=http://localhost:8080` in `.env`.

### Join grants

- `FRACTURING_SPACE_JOIN_GRANT_ISSUER`: join grant issuer.
- `FRACTURING_SPACE_JOIN_GRANT_AUDIENCE`: join grant audience.
- `FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY`: base64 Ed25519 public key for verification (game). Generate with `go run ./cmd/join-grant-key`.
- `FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY`: base64 Ed25519 private key for signing (auth). Generate with `go run ./cmd/join-grant-key`.
- `FRACTURING_SPACE_JOIN_GRANT_TTL`: join grant TTL. Default: `5m`.

### MCP

Internal gRPC dependencies default to Compose service DNS names (`service:port`). For direct local binary workflows, override these values to `localhost` in `.env.local`.

- `FRACTURING_SPACE_GAME_ADDR`: game gRPC address used by MCP, admin, and web. Default: `game:8082`.
- `FRACTURING_SPACE_MCP_HTTP_ADDR`: HTTP bind address for MCP when using HTTP transport. Default: `localhost:8085`.
- `FRACTURING_SPACE_MCP_TRANSPORT`: transport type (`stdio` or `http`). Default: `stdio`.
- `FRACTURING_SPACE_MCP_ALLOWED_HOSTS`: comma-separated allowed Host/Origin values for MCP HTTP. Defaults to loopback-only when unset.
- `FRACTURING_SPACE_MCP_OAUTH_ISSUER`: OAuth issuer URL for MCP token validation.
- `FRACTURING_SPACE_MCP_OAUTH_RESOURCE_SECRET`: shared secret for MCP token introspection.

### Admin

- `FRACTURING_SPACE_ADMIN_ADDR`: HTTP bind address for the admin dashboard. Default: `:8081`.
- `FRACTURING_SPACE_ADMIN_DB_PATH`: admin SQLite path. Default: `data/admin.db`.
- `FRACTURING_SPACE_AUTH_ADDR`: auth gRPC address used by the game, admin dashboard, and web login server. Default: `auth:8083`.

### Web

- `FRACTURING_SPACE_WEB_HTTP_ADDR`: HTTP bind address for the web login server. Default: `localhost:8080`.
- `FRACTURING_SPACE_CHAT_HTTP_ADDR`: chat HTTP bind address used for campaign chat fallback host candidates. Default: `localhost:8086`.
- `FRACTURING_SPACE_WEB_AUTH_BASE_URL`: external auth base URL for login redirects.
- `FRACTURING_SPACE_WEB_AUTH_ADDR`: auth gRPC address used by the web login server. Default: `auth:8083`.
- `FRACTURING_SPACE_NOTIFICATIONS_ADDR`: notifications gRPC address used by the web login server. Default: `notifications:8088`.
- `FRACTURING_SPACE_WEB_DIAL_TIMEOUT`: gRPC dial timeout for the web login server. Default: `2s`.
- `FRACTURING_SPACE_WEB_OAUTH_CLIENT_ID`: first-party OAuth client ID used by the web server. Default: `fracturing-space`.
- `FRACTURING_SPACE_WEB_CALLBACK_URL`: public OAuth callback URL (e.g., `http://localhost:8080/auth/callback`).
- `FRACTURING_SPACE_WEB_AUTH_TOKEN_URL`: internal auth token endpoint for server-to-server code exchange. Defaults to `{AuthBaseURL}/token`.
- `FRACTURING_SPACE_ASSET_BASE_URL`: external base URL for campaign cover and avatar assets (object storage/CDN origin).
- `FRACTURING_SPACE_ASSET_VERSION`: version prefix for generated asset keys. Default: `v1`.

### Docker + Caddy (Compose defaults)

- `FRACTURING_SPACE_DOMAIN`: base domain for subdomain routing (e.g., `example.com`).
- `FRACTURING_SPACE_PUBLIC_SCHEME`: external scheme for URLs (`http` or `https`).
- `FRACTURING_SPACE_PUBLIC_PORT`: optional port suffix (include the leading `:`; e.g., `:8080` for local).
- `FRACTURING_SPACE_BIND_ADDR`: bind address for Caddy ports (local defaults to `127.0.0.1`).
- `FRACTURING_SPACE_HTTP_PORT`: host port mapped to Caddy HTTP (default `8080` for local).
- `FRACTURING_SPACE_HTTPS_PORT`: host port mapped to Caddy HTTPS (default `8443` for local).
- `FRACTURING_SPACE_CADDY_AUTO_HTTPS`: Caddy `auto_https` setting (`off` for local, `on` for prod).
- `FRACTURING_SPACE_CADDY_SITE_PREFIX`: Caddy site prefix (`http://` for local, empty for production TLS).
- `FRACTURING_SPACE_CADDY_EMAIL`: optional Caddy global directive (e.g., `email ops@example.com`).

### Images

- `FRACTURING_SPACE_IMAGE_REGISTRY`: container registry host. Default: `ghcr.io`.
- `FRACTURING_SPACE_IMAGE_NAMESPACE`: container namespace/org. Default: `fracturing-space`.
- `FRACTURING_SPACE_IMAGE_TAG`: image tag used in Compose. Default: `dev`.

## MCP Server Configuration

### Command-line Flags

The MCP server (`cmd/mcp`) accepts the following flags:

- `-addr`: game server address. Default: `game:8082`
- `-http-addr`: HTTP server address (for HTTP transport). Default: `localhost:8085`
  
  When running the `cmd/mcp` binary, this value is provided by the flag definition. When constructing the MCP server programmatically and leaving the HTTP address empty in the `Config` struct, the server also falls back to `localhost:8085` internally.
- `-transport`: Transport type (`stdio` or `http`). Default: `stdio`

### Address Overrides

The MCP server accepts flags for gRPC and HTTP addresses. If `FRACTURING_SPACE_GAME_ADDR`
or `FRACTURING_SPACE_MCP_HTTP_ADDR` are set, they provide defaults when the matching flag
is omitted. Command-line flags take precedence over env values.

### Transport Selection

The MCP server supports `stdio` (default) and `http` transports. See
[Quickstart](quickstart.md) or [Local development](local-dev.md) for run commands and
[MCP tools and resources](../reference/mcp.md) for HTTP endpoint details.

## Admin Dashboard Configuration

### Command-line Flags

The admin dashboard (`cmd/admin`) accepts the following flags:

- `-http-addr`: HTTP server address. Default: `:8081`
- `-grpc-addr`: game server address. Default: `game:8082`
- `-auth-addr`: auth server address. Default: `auth:8083`

### Address Overrides

The admin dashboard accepts flags for gRPC and HTTP addresses. If `FRACTURING_SPACE_ADMIN_ADDR`
or `FRACTURING_SPACE_GAME_ADDR` are set, they provide defaults when the matching flag is
omitted. Command-line flags take precedence over env values.

## Web Configuration

### Command-line Flags

The web server (`cmd/web`) accepts the following flags:

- `-http-addr`: HTTP server address. Default: `localhost:8080`
- `-chat-http-addr`: Chat HTTP server address. Default: `localhost:8086`
- `-auth-addr`: auth gRPC dependency address. Default: `auth:8083`
- `-social-addr`: social gRPC dependency address. Default: `social:8090`
- `-game-addr`: game gRPC dependency address. Default: `game:8082`
- `-ai-addr`: AI gRPC dependency address. Default: `ai:8087`
- `-notifications-addr`: notifications gRPC dependency address. Default: `notifications:8088`
- `-userhub-addr`: userhub gRPC dependency address. Default: `userhub:8092`
- `-asset-base-url`: external base URL used for image asset delivery.
- `-enable-experimental-modules`: enables experimental module surfaces.

### Address Overrides

The web server accepts flags for HTTP and upstream dependency addresses. If
`FRACTURING_SPACE_WEB_HTTP_ADDR`, `FRACTURING_SPACE_CHAT_HTTP_ADDR`,
`FRACTURING_SPACE_AUTH_ADDR`, `FRACTURING_SPACE_SOCIAL_ADDR`,
`FRACTURING_SPACE_GAME_ADDR`, `FRACTURING_SPACE_AI_ADDR`,
`FRACTURING_SPACE_NOTIFICATIONS_ADDR`, or `FRACTURING_SPACE_USERHUB_ADDR` are
set, they provide defaults when matching flags are omitted. Asset URL defaults come from
`FRACTURING_SPACE_ASSET_BASE_URL`. Command-line flags take precedence over env
values.

## AI Service Configuration

### Command-line Flags

The AI service (`cmd/ai`) accepts the following flags:

- `-port`: gRPC server port. Default: `8087`

### Address Overrides

The AI service accepts the `-port` flag. If `FRACTURING_SPACE_AI_PORT` is set,
it provides the default when the flag is omitted. Command-line flags take
precedence over env values.

## Notifications Service Configuration

### Command-line Flags

The notifications service (`cmd/notifications`) accepts the following flags:

- `-port`: gRPC server port. Default: `8088`

### Address Overrides

The notifications service accepts the `-port` flag. If
`FRACTURING_SPACE_NOTIFICATIONS_PORT` is set, it provides the default when the
flag is omitted. Command-line flags take precedence over env values.

## Worker Service Configuration

### Command-line Flags

The worker service (`cmd/worker`) accepts the following flags:

- `-port`: worker health gRPC server port. Default: `8089`
- `-auth-addr`: auth gRPC dependency address. Default: `auth:8083`
- `-notifications-addr`: notifications gRPC dependency address. Default: `notifications:8088`
- `-db-path`: worker SQLite path. Default: `data/worker.db`
- `-consumer`: auth outbox consumer identifier. Default: `worker-onboarding`
- `-poll-interval`: auth outbox poll interval. Default: `2s`
- `-lease-ttl`: auth outbox lease duration. Default: `30s`
- `-max-attempts`: max attempts before dead-letter. Default: `8`
- `-retry-backoff`: base retry delay. Default: `5s`
- `-retry-max-delay`: max retry delay. Default: `5m`
- `-dial-timeout`: gRPC dependency dial timeout. Default: `2s`

### Address Overrides

The worker service accepts flags for dependency addresses and runtime timing. If
the corresponding `FRACTURING_SPACE_WORKER_*` variables are set, they provide
defaults when flags are omitted. Command-line flags take precedence over env
values.

## User Hub Service Configuration

### Command-line Flags

The user hub service (`cmd/userhub`) accepts the following flags:

- `-port`: userhub gRPC server port. Default: `8092`
- `-game-addr`: game gRPC dependency address. Default: `game:8082`
- `-social-addr`: social gRPC dependency address. Default: `social:8090`
- `-notifications-addr`: notifications gRPC dependency address. Default: `notifications:8088`
- `-cache-fresh-ttl`: fresh dashboard cache TTL. Default: `15s`
- `-cache-stale-ttl`: stale dashboard fallback TTL. Default: `2m`
- `-dial-timeout`: gRPC dependency dial timeout. Default: `2s`

### Address Overrides

The user hub service accepts flags for dependency addresses and cache/runtime
timing. If corresponding `FRACTURING_SPACE_USERHUB_*` variables are set, they
provide defaults when flags are omitted. Command-line flags take precedence over
env values.
