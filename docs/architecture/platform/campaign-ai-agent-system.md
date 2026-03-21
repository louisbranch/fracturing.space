---
title: "Campaign AI Agent System"
parent: "Platform surfaces"
nav_order: 15
status: draft
owner: engineering
last_reviewed: "2026-03-18"
---

# Campaign AI Agent System

## Purpose

Define how the AI agent behaves as a Game Master and Narrator during campaign
turns: role separation, instruction composition, context assembly, and
extension points for new game systems.

This document extends [Campaign AI Orchestration](campaign-ai-orchestration.md)
(which defines the grant, tool, and turn-loop contract) with the **behavioral**
layer: what the agent is taught, how context is budgeted, and how contributors
add game-system-specific intelligence.

## Output Channel Model

The agent speaks through three output channels. Each maps to a distinct role:

| Channel | Role | Purpose |
|---------|------|---------|
| Tool calls (dice, scene, interaction tools) | **Game Master** | Adjudicate rules, resolve mechanics, manage authoritative game state |
| `interaction_scene_gm_output_commit` text | **Narrator** | In-character prose, atmosphere, NPC dialogue |
| Final model response (`OutputText`) | **Meta / OOC** | Conversational reply to the caller, summaries, coordination notes |

**Channel discipline**: The agent must never mix rules text into committed
narration, nor embed state-mutating decisions in free-form prose. Tool calls
are the sole authority for state changes; committed text is the sole authority
for in-character narration.

## Instruction Composition

Agent instructions are **markdown files on disk**, not Go string literals.
This enables iteration without recompilation and A/B testing via directory
swap.

### Directory Layout

```
data/instructions/
  v1/
    core/
      skills.md           # Core GM operating contract
      interaction.md      # Tool channel discipline, commit rules
      memory-guide.md     # How to manage structured memory
    daggerheart/
      skills.md           # Daggerheart-specific GM guidance
      reference-guide.md  # Reference lookup patterns
    # Future: dnd5e/, vampire/, etc.
```

### Composition Order

`campaigncontext/instructionset.Loader` composes skills guidance in this order:

1. `core/skills.md` — universal GM/Narrator contract
2. `{system}/skills.md` — game-system-specific guidance (e.g. Daggerheart)
3. `core/memory-guide.md` — memory management guidance
4. `{system}/reference-guide.md` — system reference lookup patterns

`core/interaction.md` is loaded separately as the prompt renderer's
interaction-contract text so startup can degrade missing fields independently
instead of disabling the full prompt path.

### Runtime Override

`FRACTURING_SPACE_AI_INSTRUCTIONS_ROOT` env var points to an alternative
instruction directory. Default: embedded `data/instructions/v1` via
`embed.FS`.

## Context Assembly Pipeline

Each turn, the prompt path first collects a typed **session brief** and then
renders the final prompt from that brief plus explicit render policy. The
collector is a `SessionBriefCollector` backed by a `ContextSourceRegistry`;
the renderer is a `PromptRenderer` chosen by AI startup in
`internal/services/ai/app`.

The default renderer still uses `BriefAssembler` to sort prompt sections by
priority and drop low-priority content when the token budget is tight.

### Priority Tiers

| Priority | Tier | Content |
|----------|------|---------|
| 100 | Critical | Skills contract, interaction state, turn input, authority |
| 200 | Important | Campaign metadata, active scene characters, current phase |
| 300 | Contextual | All participants, story.md, session recap, character list |
| 400 | Supplemental | Full character profiles, reference excerpts, memory.md |

### Token Budgeting

Token estimation uses a byte heuristic (~4 chars per token). Required sections
are never dropped. Non-required sections are dropped lowest-priority-first
when the assembled brief exceeds the budget.

### ContextSource Interface

Game systems contribute prompt sections via the `ContextSource` interface:

```go
type ContextSource interface {
    Collect(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error)
}
```

Core sources (campaign metadata, characters, scenes, interaction state) are
always present. System-specific sources (for example Daggerheart rules and
reference context) are registered into the same collector registry at the
composition root.

The important invariant is that prompt rendering consumes the typed
`SessionBrief`; it no longer re-parses already rendered prompt text to recover
bootstrap or interaction-state facts.

## Extension Points

### Adding a Game System

1. Create `data/instructions/v1/{system}/skills.md` with system-specific GM
   guidance.
2. Create `data/instructions/v1/{system}/reference-guide.md` with reference
   lookup patterns.
3. Implement `ContextSource` values for system-specific prompt sections.
4. Register those sources in the AI composition root.
5. Register the game system in the broader platform manifest when the rest of
   the platform needs to discover it.

### Modifying Agent Behavior

Edit the markdown instruction files under `data/instructions/v1/`. Changes
take effect on the next turn without recompilation when using
`FRACTURING_SPACE_AI_INSTRUCTIONS_ROOT`, or on next deploy when using the
embedded default instruction set.

## Relationship to Other Docs

- [Campaign AI Orchestration](campaign-ai-orchestration.md) — grant, tool
  policy, turn-loop mechanics
- [Campaign AI Session Bootstrap](campaign-ai-session-bootstrap.md) — session
  start readiness and bootstrap behavior
- [AI service contributor map](../../reference/ai-service-contributor-map.md) — package routing for contributors
