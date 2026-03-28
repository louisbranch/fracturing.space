---
title: "Campaign AI GM Guardrails"
parent: "Platform surfaces"
nav_order: 22
status: canonical
owner: engineering
last_reviewed: "2026-03-28"
---

# Campaign AI GM Guardrails

## Purpose

Define the behavioral guardrails for AI agents acting as RPG Game Masters.
Each guardrail specifies the rule, enforcement level, and mechanism.

Enforcement levels: **Runtime** (orchestration loop), **Eval** (promptfoo
assertions), **Instruction** (agent instructions only).

## Turn Structure

| ID | Guardrail | Level | Mechanism |
|----|-----------|-------|-----------|
| T1 | Must commit authoritative interaction before final text | Runtime | `TurnController.HasCommittedOrResolvedInteraction()` |
| T2 | Must open player phase, conclude session, or OOC before completion | Runtime | `TurnController.ReadyForCompletion()` |
| T3 | Must not commit narration after opening player phase | Runtime | `TurnController.PlayerHandoffRegressed()` |
| T5 | Beat ordering follows interaction contract | Eval | `required_beat_types` / `forbidden_beat_types` |
| T8 | Prompt beats ask what the PC does, not what NPCs say | Eval + Instruction | `forbidden_prompt_phrases` + `skills.md` |

## Tool Usage

| ID | Guardrail | Level | Mechanism |
|----|-----------|-------|-----------|
| U1 | Only tools in the curated GM-safe allowlist | Runtime | `ToolPolicy.Allows()` |
| U2 | Max 8 provider loop steps per turn | Runtime | Step counter in `runner.go` |
| U4 | Reference lookup bounded per scenario | Eval | `max_reference_searches` / `max_reference_reads` |
| U5 | Sheet read before capability-sensitive mechanics | Instruction + Eval | `daggerheart/skills.md` + tool order assertions |

## Narrative Authority

| ID | Guardrail | Level | Mechanism |
|----|-----------|-------|-----------|
| N1 | Prompt beats for PC intention only | Eval + Instruction | `forbidden_prompt_phrases` |
| N4 | No engine vocabulary in player-facing beats | Instruction | `campaign-ai-agent-system.md` |
| N5 | Impossible declarations trigger clarification | Eval | `IntentImpossibleAction` scenario |
| N6 | Ambiguous intent triggers clarification | Eval | `IntentAmbiguousAction` scenario |

## Mechanics Correctness

| ID | Guardrail | Level | Mechanism |
|----|-----------|-------|-----------|
| M1 | Resource accounting matches tool outcomes | Eval | `expected_hope`, `expect_action_roll_hope_spend_source` |
| M2 | Character capability verification before adjudication | Instruction + Eval | Sheet-first rule, tool order prefix |
| M5 | Action roll modifier source matches declared action | Eval | `expect_action_roll_modifier_source` |

## Context and Memory

| ID | Guardrail | Level | Mechanism |
|----|-----------|-------|-----------|
| C1 | OpenViking non-authoritative only | Runtime | Architecture boundary |
| C2 | Graceful degradation when OpenViking unavailable | Runtime | Nil checks, error handling |
| C3 | Phase-aware context access map | Runtime | `BuildPhaseGuide()` + `BuildContextAccessMap()` |
| C5 | skills.md read-only | Eval | `expect_skills_read_only` |

## Safety

| ID | Guardrail | Level | Mechanism |
|----|-----------|-------|-----------|
| S1 | Session grant validation | Runtime | `auth_reference_policy.go` + signed JWT |
| S2 | Credential/grant revocation blocked while bound | Runtime | `usage_policy.go` |
| S4 | Prompt injection resistance | Eval | `RedTeamPromptInjection` scenario |
| S5 | Jailbreak resistance | Eval | `RedTeamJailbreak` scenario |
| S6 | Hallucination resistance | Eval | `RedTeamHallucination` scenario |
| S7 | Hijacking resistance | Eval | `RedTeamHijacking` scenario |
| S8 | Overreliance resistance | Eval | `RedTeamOverreliance` scenario |
| S9 | Excessive agency resistance | Eval | `RedTeamExcessiveAgency` scenario |

## Relationship to Other Docs

- [Campaign AI Agent System](campaign-ai-agent-system.md) — instruction composition
- [Campaign AI Orchestration](campaign-ai-orchestration.md) — runtime turn boundary
- [Campaign AI Mechanics Quality](campaign-ai-mechanics-quality.md) — mechanics design
- [Campaign AI Evaluation Strategy](campaign-ai-evaluation-strategy.md) — how guardrails are tested
- [OpenViking Integration](openviking-integration.md) — retrieval sidecar
