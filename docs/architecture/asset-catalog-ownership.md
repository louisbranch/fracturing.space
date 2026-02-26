---
title: "Asset catalog ownership"
parent: "Architecture"
nav_order: 20
status: canonical
owner: engineering
last_reviewed: "2026-02-26"
---

# Asset Catalog Ownership and Runtime Strategy

## Purpose

Define ownership boundaries and shared mechanics for campaign covers and avatars before expanding image support to users, participants, and characters.

This document separates:

1. Entity ownership (which service owns each record).
2. Asset mechanics (catalog validation, deterministic default assignment, URL resolution).

## Ownership Boundaries

The owning domain service stores the image reference fields for its entities.

| Entity | Owning service | Reason |
| --- | --- | --- |
| Campaign cover | Game service | Campaign is a game-domain aggregate. |
| Participant avatar | Game service | Participant membership and seat state are game-domain concerns. |
| Character avatar | Game service | Character identity and control state are game-domain concerns. |
| User avatar | Auth service | User profile metadata is an auth/account concern. |

Cross-service writes are out of scope. Shared logic must be imported, not called via write APIs across service boundaries.

## Shared Catalog Contract

A shared asset-catalog package will be used by both game and auth services. It must provide:

1. Set and asset ID definitions.
2. ID validation and normalization.
3. Alias resolution to canonical IDs.
4. Deterministic default assignment.
5. URL resolution from canonical IDs plus runtime config.

The shared package does not own domain records or persistence.

## Data Model Contract

Store both set and asset identifiers for all image-bearing entities.

| Entity field pair | Purpose |
| --- | --- |
| `cover_set_id`, `cover_asset_id` | Campaign cover identity. |
| `avatar_set_id`, `avatar_asset_id` | Participant avatar identity. |
| `avatar_set_id`, `avatar_asset_id` | Character avatar identity. |
| `avatar_set_id`, `avatar_asset_id` | User account avatar identity. |

Rationale:

1. Keeps future set-switching explicit.
2. Avoids deriving set membership from asset IDs later.
3. Simplifies validation and migration.

## Deterministic Default Assignment

Default assignment must be deterministic and replay-safe.

Inputs:

1. `entity_type` (for example `campaign`, `participant`, `character`, `user`).
2. `entity_id`.
3. `set_id`.
4. Algorithm version label (for example `asset-default-v1`).

Behavior:

1. Build a stable input string from these fields.
2. Hash the input with a fixed algorithm.
3. Select from a stable ordered asset list in the set using modulo.
4. Persist selected canonical set/asset IDs.

No runtime non-deterministic random selection is allowed for defaults.

## Runtime Asset Hosting

Binary assets should not be stored in repo-embedded static bundles for large catalogs.

Required runtime model:

1. Host image binaries in object storage.
2. Serve through CDN.
3. Keep only compact metadata manifest in repo.

URL contract:

1. Key format is immutable and versioned (for example `v1/avatars/{set_id}/{asset_id}.webp`).
2. Services generate URLs from `ASSET_BASE_URL` plus manifest key data.
3. Rebuild is not required when only base URL or CDN origin changes.

## Alias and Compatibility Policy

Catalog entries may include aliases for backwards compatibility.

Rules:

1. Persist canonical IDs only after normalization.
2. Accept aliases at read/update boundaries during migration windows.
3. Keep explicit alias mapping until old IDs are removed from stored data.
4. Typo IDs are not preserved as aliases; stored values must use canonical IDs only.

## Migration and Rollout Order

1. Add schema fields and proto fields additively.
2. Deploy code that can read both old and new shapes.
3. Publish manifest version and object-store keys.
4. Enable new writes for set-aware fields.
5. Backfill legacy rows where needed.
6. Remove temporary fallback paths only after validation period.

## Non-Goals for This Phase

1. User-driven set switching UI.
2. Uploading custom images.
3. Cross-service orchestration for profile writes.

## Operational Expectations

1. Rollback must be possible by manifest version and runtime config.
2. Missing asset keys should fail visibly in logs and degrade to known default IDs.
3. Coverage/tests must include validation, alias normalization, deterministic assignment, and URL generation.
