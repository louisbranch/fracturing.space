---
title: "Docs Quality Checks"
parent: "Guides"
nav_order: 2
---

# Docs Quality Checks

Use these checks before opening documentation-heavy PRs.

## Quick run

```sh
make docs-check
```

This runs:

- `make docs-path-check` for backtick path references.
- `make docs-link-check` for markdown link validity.
- `make docs-index-check` for section index coverage.
- `make docs-lifecycle-check` to block roadmap/phase/backlog tracker artifacts in `docs/`.
- `make docs-architecture-budget-check` to enforce concise architecture pages (`<=150` lines for non-index pages).

## Drift spot checks (recommended)

For onboarding docs, run quick consistency greps:

```sh
rg -n "docker compose|make bootstrap" README.md docs/running
rg -n "FRACTURING_SPACE_MCP_HTTP_ADDR|http-addr|8081|8085" docs/running docs/reference/mcp.md internal/cmd/mcp internal/services/mcp
rg -n "defined by an email|independent of any email address" docs/architecture/foundations/domain-language.md docs/architecture/platform/identity-and-oauth.md
rg -n "WEBAUTHN_RP_ORIGINS|MAGIC_LINK_BASE_URL" docs/running/configuration.md docs/architecture/platform/identity-and-oauth.md internal/services/auth
```
