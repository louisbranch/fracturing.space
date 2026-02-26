---
name: schema
description: Schema and proto evolution with explicit compatibility decisions
user-invocable: true
---

# Schema Changes

Prefer clear domain models and maintainable schemas. Internal backward compatibility is optional unless a product/API contract requires it.

## Compatibility Decision First

- If external clients or retained production data are affected, treat compatibility as an explicit product decision and document it.
- If no compatibility requirement exists, prefer clean schema/proto design over preserving legacy shapes.

## Database Migrations

- Favor simple, reviewable migrations with clear intent.
- For prototype/internal tables where data can be rebuilt, dropping and recreating can be cleaner than long `ALTER TABLE` chains.
- When data must be preserved, use additive/transform migrations and include backfill/verification steps.
- Keep each migration focused on one domain change.

## Proto Fields

- Internal, unstable protos may be renumbered and regrouped for clarity when compatibility is not required.
- Stable or externally consumed protos must keep field numbers stable and must not reuse retired numbers.

When adding or reorganizing proto fields, keep numbering intentional and regenerate with `make proto`.

## Documentation

- Record schema/proto rationale in `docs/` when it changes domain language, ownership boundaries, or migration policy.
