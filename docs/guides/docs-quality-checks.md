---
title: "Docs Quality Checks"
parent: "Guides"
nav_order: 2
---

# Docs Quality Checks

Use these checks before opening documentation-heavy PRs.

## Quick run

```sh
make docs-path-check
```

This verifies key docs do not reference broken repo paths.

## Drift spot checks (recommended)

For onboarding docs, run quick consistency greps:

```sh
rg -n "docker compose|make bootstrap" README.md docs/running
rg -n "FRACTURING_SPACE_MCP_HTTP_ADDR|http-addr|8081|8085" docs/running docs/reference/mcp.md internal/cmd/mcp internal/services/mcp
rg -n "defined by an email|independent of any email address" docs/project/domain-language.md docs/project/identity.md
rg -n "WEBAUTHN_RP_ORIGINS|MAGIC_LINK_BASE_URL" docs/running/configuration.md docs/project/oauth.md internal/services/auth
```
