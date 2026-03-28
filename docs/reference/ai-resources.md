---
title: "AI orchestration resources"
parent: "Reference"
nav_order: 3
last_reviewed: "2026-03-23"
---

# AI orchestration resources

Resource URIs available to the current campaign-turn orchestration runtime.

This page describes the current production profile. It is Daggerheart-first
today; additional game-system resource families are future architecture work.

## Core campaign resources

- `context://current`
- `campaign://{campaign_id}`
- `campaign://{campaign_id}/participants`
- `campaign://{campaign_id}/characters`
- `campaign://{campaign_id}/sessions`
- `campaign://{campaign_id}/sessions/{session_id}/scenes`
- `campaign://{campaign_id}/interaction`
- `campaign://{campaign_id}/artifacts/{path}`

The production prompt path commonly reads `story.md` and `memory.md` through
the artifact URI family above; there is no separate artifact-directory contract
beyond the artifact-path reader.

Global campaign listing is intentionally excluded from the runtime profile.

## Daggerheart-specific resources

- `campaign://{campaign_id}/characters/{character_id}/sheet`

The Daggerheart prompt layer uses character-sheet resources to build always-on
mechanics and character briefs. That sheet surface is part of the current
runtime contract even though broader multi-system resource registration does not
exist yet.

## Verification

For implementation and current usage, inspect:

- `internal/services/ai/orchestration/context_sources_core.go`
- `internal/services/ai/orchestration/daggerheart/context_sources.go`
- `internal/services/ai/orchestration/gametools/resources_dispatch.go`
- `internal/services/ai/orchestration/gametools/resources_campaign.go`
- `internal/services/ai/orchestration/gametools/resources_artifacts.go`
- `internal/services/ai/orchestration/daggerhearttools/read_surfaces.go`
