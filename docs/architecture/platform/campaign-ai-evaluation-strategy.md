---
title: "Campaign AI Evaluation Strategy"
parent: "Platform surfaces"
nav_order: 23
status: canonical
owner: engineering
last_reviewed: "2026-03-28"
---

# Campaign AI Evaluation Strategy

## Purpose

Define the evaluation framework for AI GM quality. Guardrail definitions
live in [Campaign AI GM Guardrails](campaign-ai-gm-guardrails.md).

## Architecture

Evaluation uses [promptfoo](https://www.promptfoo.dev/) as a non-invasive
reporting layer over the live Go integration harness. Each scenario runs
through the full AI service orchestration loop with a real provider and real
game-service tools.

Key decisions: non-gating (manual diagnostics, not CI gates); contract-based
assertions (structural correctness, not subjective quality); runtime-invalid
rows tracked separately from model-quality failures.

## Scenario Coverage (29 total)

| Category | Count | Examples |
|----------|-------|---------|
| Core mechanics | 7 | Bootstrap, HopeExperience, SubdueIntent, PlaybookAttackReview |
| Intent ladder | 5 | IntentHopeSpend, IntentImpossible, IntentAmbiguous |
| Existing registered | 5 | OOCReplace, SceneSwitch, GroupAction, CapabilityLookup |
| Red-team adversarial | 6 | PromptInjection, Jailbreak, Hallucination, Overreliance |
| Multi-turn | 3 | NarrativeContinuity, MemoryRecall, SessionPacing |
| Starter campaign | 2 | StarterActProgression, StarterConclusion |

## Assertion Dimensions (8 weighted)

Gradient scoring replaces binary pass/fail. Each dimension contributes a
proportional score.

| Dimension | Weight | Checks |
|-----------|--------|--------|
| tool_contract | 0.20 | required/forbidden tools, ordering |
| tool_arguments | 0.10 | required/forbidden arg values and keys |
| beat_contract | 0.15 | required/forbidden beat types |
| narrative_authority | 0.15 | player phase, forbidden prompt phrases |
| resource_accounting | 0.15 | Hope, modifier source, hope spend |
| reference_budget | 0.10 | search/read count limits |
| instruction_integrity | 0.05 | skills.md read-only |
| adversarial_resilience | 0.10 | forbidden output phrases |

## Regression Baseline

`tools/promptfoo/baseline.json` stores the best-known result per scenario.
`compare_baseline.js` classifies each run as regression, improvement,
unchanged, or new. Generated automatically on every eval run.

## Running Evaluations

```bash
make ai-eval-promptfoo                     # default preset
make ai-eval-promptfoo-core                # core scenarios, 1 repeat
make ai-eval-promptfoo-decision            # core, 3 repeats
make ai-eval-promptfoo-view                # web viewer
```

CI: `.github/workflows/ai-gm-eval.yml` with manual dispatch.

## Relationship to Other Docs

- [Campaign AI GM Guardrails](campaign-ai-gm-guardrails.md) — what the agent must/must not do
- [Campaign AI Mechanics Quality](campaign-ai-mechanics-quality.md) — mechanics design guidance
- [Campaign AI Agent System](campaign-ai-agent-system.md) — instruction composition
- [OpenViking Integration](openviking-integration.md) — retrieval sidecar
