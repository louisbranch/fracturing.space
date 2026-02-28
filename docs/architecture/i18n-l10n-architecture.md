---
title: "I18n and l10n architecture"
parent: "Architecture"
nav_order: 12
status: canonical
owner: engineering
last_reviewed: "2026-02-28"
---

# I18n and l10n Architecture

## Intent

Fracturing.Space uses one platform-owned translation catalog system so:

- product copy is discoverable in one place,
- shared terms are translated once,
- translation completeness is measurable from source control.

## Source of Truth

All translatable copy lives in:

- `internal/platform/i18n/catalog/locales/en-US/*.yaml`
- `internal/platform/i18n/catalog/locales/pt-BR/*.yaml`

Catalog files are namespace-scoped (for example `core`, `web`, `admin`, `game`,
`notifications`, `errors`) and loaded by `internal/platform/i18n/catalog`.

At process startup, the platform loader validates catalogs and registers strings
into `golang.org/x/text/message`.

## Key Ownership Rules

- Shared cross-service keys must live in the `core` namespace.
- `core.*` keys are rejected outside the `core` namespace.
- A key can appear only once per locale across all namespaces.

This prevents duplicate authority and accidental term drift between services.

## Runtime Behavior

- Locale negotiation remains in `internal/services/shared/i18nhttp`.
- Message lookup continues through `message.Printer`.
- Error-code localization uses the `errors` namespace and falls back to `en-US`
  when a locale does not define that namespace.

## Validation and Reporting

Two tools are canonical:

- `make i18n-check`: validates catalog integrity and placeholder compatibility.
- `make i18n-status`: generates translator status artifacts in `docs/reference/`.

CI enforces both with:

- `make i18n-check`
- `make i18n-status-check`

## Non-goals

- External TMS integration.
- Automatic machine translation.
- Adding new supported locales without explicit product/architecture approval.
