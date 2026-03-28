---
title: "OpenViking Integration"
parent: "Platform surfaces"
nav_order: 19
status: canonical
owner: engineering
last_reviewed: "2026-03-25"
---

# OpenViking Integration

## Purpose

Define the current OpenViking boundary in the AI service, the non-authoritative
rules around it, and the follow-up adoption paths if deeper use is justified.
The current evaluation is about prompt bloat, weak cross-turn recall, and
diminishing returns from large narrative context blocks.

## Current Integration

OpenViking is an optional sidecar pinned locally to
`ghcr.io/volcengine/openviking:v0.2.10`. It does not own the campaign-turn
runtime. The AI service still owns orchestration, provider/tool-loop execution,
the GM/narrator contract, mechanics policy, and turn completion.

OpenViking currently contributes two seams:

- `openviking.PromptAugmenter`: `legacy` mirrors `story.md` and `memory.md`;
  `docs_aligned_supplement` now mirrors a generated phase guide, a generated
  `story-index.md`, the raw `story.md` tree as fallback source material, keeps
  prompt-time `memory.md`, uses shallow-first scoped resource retrieval plus
  session-aware memory search, and adds the top distinct rendered contexts
  within the configured section budget.
- `openviking.SessionSync`: called through `TurnMemorySync` to mirror completed
  turns into OpenViking sessions and trigger `commit()`. It can be disabled for
  augmentation-only evaluation.

When OpenViking is enabled, `legacy` suppresses raw `story.md` and `memory.md`,
while `docs_aligned_supplement` suppresses raw `story.md`, keeps raw
`memory.md`, and relies on an always-on phase guide plus story index before any
fallback to raw story retrieval.

## Boundary Rules

- Authoritative game state, interaction state, character state, and mechanics
  outcomes stay outside OpenViking.
- OpenViking may hold only non-authoritative context such as `story.md`,
  `memory.md`, session memory, operator notes, and retrieved summaries.
- Retrieved OpenViking material is advisory context; it does not override
  authoritative tool reads or committed game writes.
- If OpenViking is unavailable, prompt augmentation degrades to the existing
  AI-service behavior and post-turn sync remains best-effort.

## Responsibility Split

| Boundary | Owner |
| --- | --- |
| GM role contract, narrator discipline, tool whitelist, mechanics policy | AI service |
| Turn loop, completion policy, authoritative writes | AI service |
| Retrieval, session memory, context indexing | OpenViking |
| Stable resource/session conventions and degradation behavior | Shared contract |

## Evaluation Status

The first retrieval-before-prompt phase has now been exercised through the
intended four decision lanes on `gpt-5.4-mini` against the pinned OpenViking
`v0.2.10` sidecar.

| Lane | Baseline input tokens | OpenViking input tokens | Outcome | Assessment |
| --- | ---: | ---: | --- | --- |
| `Bootstrap` | 85,954 | 67,475 | `clean_pass` -> `clean_pass` | Clear win with valid retrieval and raw `story.md` suppression |
| `MechanicsReview` | 55,822 | 55,736 | `clean_pass` -> `clean_pass` | Effective parity after backing-file story rendering |
| `ReactionReview` | 69,096 | 56,388 | `clean_pass` -> `clean_pass` | Positive after repair; candidate still issued a duplicate memory update |
| `CapabilityLookup` | 64,799 | 65,128 | `clean_pass` -> `clean_pass` | Previously showed artifact get/upsert drift; re-evaluated March 27 with enriched queries and phase-aware sections — tool path now matches baseline cleanly |

Current recommendation: `Limited adoption`. All four decision lanes now pass
cleanly with `docs_aligned_supplement` mode. The CapabilityLookup drift
previously observed no longer reproduces after the enriched search query and
phase-aware section budget changes. Bootstrap remains the clearest positive
signal. The remaining adoption gate is the session-memory track.

Immediate next steps:

- continue the separate runtime session-memory track
- monitor CapabilityLookup stability across model version changes
- revisit default enablement after session-memory is decision-grade

## Reference Corpus (OV1)

An OV1 variant exists with L0/L1/L2 content layers (`FRACTURING_SPACE_AI_OPENVIKING_REFERENCE_CORPUS_ROOT`, opt-in, default off). Evaluated March 27: loading OV1 additively regressed over-research. Continue with V1 until phase-scoped or replacing V1 entirely.

## Session Memory Track

Session memory remains separate from the paid retrieval-before-prompt gate.
`TestOpenVikingSessionMemoryLive` proves session create/message append/commit/
search at the sidecar seam. It does not prove that AI-service runtime sync or
in-turn OpenViking memory retrieval are ready for default use.

Next steps for this track:

- make runtime session sync decision-grade in the AI service
- prove OpenViking-backed memory retrieval inside campaign turns
- then decide whether OpenViking memory should replace any curated recap flow

Decision outcomes remain:

- `No-go`
  OpenViking does not improve prompt load enough, or regressions outweigh the
  benefit.
- `Limited adoption`
  Keep OpenViking as a memory and retrieval sidecar only.
- `Deeper adoption`
  Proceed to a follow-up architecture change for tighter runtime integration.

## Future Feature Candidates

Official OpenViking docs describe several capabilities beyond the current
retrieval-before-prompt sidecar. These are candidates to adopt later if they
replace custom work without weakening the AI-service GM contract.

| Candidate | Why it matters | Why not yet |
| --- | --- | --- |
| [Skills API](https://github.com/volcengine/OpenViking/blob/main/docs/en/api/04-skills.md) and MCP-style tool advertising | Could replace a separate skill registry or tool-advertising layer | GM tool policy and turn rules still live in the AI service |
| [Filesystem](https://github.com/volcengine/OpenViking/blob/main/docs/en/api/03-filesystem.md), [Retrieval](https://github.com/volcengine/OpenViking/blob/main/docs/en/api/06-retrieval.md), and the [memory plugin example](https://github.com/volcengine/OpenViking/blob/main/examples/opencode-memory-plugin/README.md) | Could replace homegrown memory inspection, context browsing, and operator-debug tooling | Current work is intentionally constrained to retrieval-before-prompt |
| [README](https://github.com/volcengine/OpenViking/blob/main/README.md), [Sessions API](https://github.com/volcengine/OpenViking/blob/main/docs/en/api/05-sessions.md), and [Context Types](https://github.com/volcengine/OpenViking/blob/main/docs/en/concepts/02-context-types.md) | Offer retrieval observability, used-context recording, and commit-driven user/agent memory extraction | Only the sidecar seam is proven today; runtime memory integration is still follow-up work |

## Future Runtime-Substrate Option

If OpenViking later owns more of the generic agent substrate, the safe split is
still:

- OpenViking owns session history, long-term memory, retrieval, resource
  indexing, and generic skill/tool exposure
- AI service still owns GM policy, mechanics adjudication, authoritative tool
  policy, authoritative writes, and turn completion rules

That future shape is not "replace the AI service." It is "let OpenViking own
more generic agent plumbing while the AI service continues to impose the RPG GM
contract."

## Related Architecture

- [AI service architecture](ai-service-architecture.md)
- [Campaign AI orchestration](campaign-ai-orchestration.md)
- [Campaign AI agent system](campaign-ai-agent-system.md)
- [Campaign AI mechanics quality](campaign-ai-mechanics-quality.md)
- [Campaign AI GM guardrails](campaign-ai-gm-guardrails.md) — context/memory guardrails covering OpenViking boundary rules
- [Campaign AI evaluation strategy](campaign-ai-evaluation-strategy.md) — retrieval quality evaluation gaps and diagnostics coverage
- [Campaign AI session bootstrap](campaign-ai-session-bootstrap.md)
