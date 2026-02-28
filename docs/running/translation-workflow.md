---
title: "Translation workflow"
parent: "Running"
nav_order: 8
status: canonical
owner: engineering
last_reviewed: "2026-02-28"
---

# Translation Workflow

## Contributor flow

1. Run `make i18n-status`.
2. Open `docs/reference/i18n-status.md`.
3. Pick missing keys for your target locale.
4. Edit locale files under `internal/platform/i18n/catalog/locales/<locale>/`.
5. Run checks:

```bash
make i18n-check
make i18n-status-check
```

6. Submit a PR with updated catalog files and refreshed status artifacts.

## Maintainer rules

- New user-facing text must include `en-US` entries.
- Shared terms belong in the `core` namespace.
- Do not reintroduce service-local i18n registries.
- Keep status artifacts (`docs/reference/i18n-status.md`,
  `docs/reference/i18n-status.json`) in sync with catalog changes.

## Useful commands

```bash
make i18n-check
make i18n-status
make i18n-status-check
```
