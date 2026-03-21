---
title: "AI model comparison — GM bootstrap"
parent: "Reference"
nav_order: 19
last_reviewed: "2026-03-18"
---

# AI model comparison — GM bootstrap

Comparison of OpenAI models running the campaign GM bootstrap scenario.
All runs use the same 30-tool profile, strict-mode schemas, and degraded-mode
prompt builder (no pre-loaded skills or memory-guide instructions).

Raw captures are committed in
`internal/test/integration/fixtures/captures/`.

## Test scenario

Prompt: _"Open the session, consult the Fear reference first, and update
memory.md with session notes about the harbor debt."_

The model must:

1. Search and read the Daggerheart Fear reference
2. Update memory.md with session notes
3. Create and activate a scene
4. Commit GM narration
5. Start the first player phase

## Token usage and cost

| Model | Input | Output | Reasoning | Total | Est. cost |
|-------|------:|-------:|----------:|------:|----------:|
| gpt-4.1-mini | 36,547 | 703 | 0 | 37,250 | ~$0.02 |
| gpt-5-nano | 97,139 | 6,054 | 4,032 | 103,193 | ~$0.04 |
| gpt-5-mini | 83,151 | 2,138 | 768 | 85,289 | ~$0.08 |

gpt-4.1-mini is cheapest per turn. gpt-5-nano uses 2.6x more input tokens but
lower per-token rates keep cost moderate. gpt-5-mini balances quality and cost.

## Tool calling reliability

| Model | Runs | Clean runs | Tool errors | Duplicate calls |
|-------|-----:|:----------:|:-----------:|:---------------:|
| gpt-4.1-mini | 6 | 3 | OOC precondition, duplicate scene_create, duplicate active_scene_set | Yes |
| gpt-5-nano | 1 | 1 | None | No |
| gpt-5-mini | 4 | 2 | OOC precondition (when memory prompt was ambiguous) | No |

gpt-4.1-mini had the most variable tool calling — 3 of 6 runs hit errors or
made duplicate calls. Both reasoning models (5-nano, 5-mini) produced clean
tool sequences after schema improvements.

**Key finding:** All three models failed with the original schemas (missing
`strict: true`, `campaign_id` noise, no `additionalProperties: false`). After
fixing schema quality, all three pass reliably. Schema quality matters more than
model size for tool calling.

## Tool sequence

All three models converge on the same 7-step optimal sequence when the run is
clean:

```
system_reference_search → system_reference_read → memory_write →
scene_create → interaction_active_scene_set →
interaction_scene_gm_output_commit → interaction_scene_player_phase_start
```

The memory write tool varies: gpt-4.1-mini and gpt-5-nano prefer
`campaign_memory_section_update`; gpt-5-mini uses `campaign_artifact_upsert`.

## Narrative quality

### Scene framing

**gpt-4.1-mini** — functional but generic:
> _As the first light of dawn breaks over Replay Harbor, salty air mixes with
> the tension of an impending debt collection. The harbor bustles quietly, with
> ships anchored and merchants preparing for the day._

**gpt-5-nano** — more specific, introduces the Black Lantern messenger:
> _Dawn waits over the harbor as the docks wake with creak and fog. The Black
> Lantern's sigil glows faintly on a messenger's coat as they approach the quay
> with a debt notice. The harbor master is late, the ledger open for all to see
> the owed favor._

**gpt-5-mini** — atmospheric and tactile:
> _Pre-dawn at the eastern quay of Replay Harbor. Salt-slick planks, stacked
> crates, and the low murmur of watchmen. Lantern light pools in rings; gulls
> roost on rigging. The Black Lantern's warning hangs in the air: a debt is to
> be collected at dawn._

### GM narration

**gpt-4.1-mini** — reads like a summary, tells rather than shows:
> _The dawn breaks gently over Replay Harbor, painting the sky with soft hues of
> pink and orange. The harbor, usually a cacophony of sounds and motion, holds a
> subdued tension this morning._

**gpt-5-nano** — more grounded, introduces environmental detail:
> _Dawn settles over Replay Harbor in a pale wash of gray and copper. The docks
> creak as seafog threads through rigging and crates, and the water at the quay
> glints with a thin, oily sheen._

**gpt-5-mini** — strongest prose, sensory language, tension through rhythm:
> _The quay breathes in slow, salt-sour gusts beneath a pale, thinning sky.
> Lanterns wobble along the planks; crates cast long, spidery shadows. Near the
> harbor master's office, a broad-shouldered man leans on the rail, eyes
> red-rimmed with worry._

### Player framing

**gpt-4.1-mini** — minimal, gives only a vague prompt:
> _The harbor master, indebted to the party, stands nearby, tension hanging in
> the salty morning air. It's now up to you to act. Will you negotiate,
> confront, or take another approach?_

**gpt-5-nano** — structured but slightly mechanical:
> _The Readiness Character 1 stands at the edge of the quay as the dawn light
> creeps across Replay Harbor. A Black Lantern messenger has delivered a debt
> notice. You may parley with the messenger..._

**gpt-5-mini** — concise, gives mechanical context:
> _Player phase — Readiness Character 1: You're on the quay before dawn.
> Describe your approach (speak to the harbor master, confront arrivals, hide,
> set a trap, gather information, etc.). If your action requires a roll, describe
> how you want to attempt it; I'll set the Difficulty and call for a roll when
> needed._

### Memory organization

**gpt-4.1-mini** — single-line note:
> _Remember: the harbor master owes the party a favor. A looming debt must be
> collected at the harbor at dawn, threatening tension in the air._

**gpt-5-nano** — structured with GM notes:
> _Harbor Debt (Opening Night): The harbor master owes the party a favor, and a
> debt is coming due at dawn. Fear is a resource for GM moves; use it to
> escalate tension as dawn approaches._

**gpt-5-mini** — structured session log with GM intent:
> _Session notes — Opening Night (2026-03-18):_
> - _The Black Lantern warned the party that a debt will be collected at dawn._
> - _Harbor debt: an unnamed creditor is expected to come to the harbor._
> - _The harbor master is indebted to the party — can be used as leverage._
> - _GM intent: open the scene at the harbor before dawn; track countdowns or
>   Fear if collectors escalate._

## Memory write sensitivity

gpt-5-mini skipped the memory write when prompted with "remember the harbor
debt" (2 consecutive runs). It correctly reasoned the information was already
in memory.md and saw no need to update. Once prompted explicitly with "update
memory.md with session notes," it wrote immediately.

**Implication for production:** The full prompt builder loads `memory-guide.md`
with explicit instructions about when to write memory. The degraded-mode builder
used by integration tests does not. Models need clear instruction about _when_
to write, not just tool availability.

## Recommendation

**Default test model: `gpt-5-mini`**

- Best narrative quality for GM output
- Reliable tool calling with strict schemas
- Structured memory organization
- ~$0.08/bootstrap turn — acceptable for integration testing
- Reasoning tokens enable better planning without explicit chain-of-thought

gpt-4.1-mini remains useful for cost-sensitive smoke tests via
`INTEGRATION_AI_MODEL=gpt-4.1-mini` override.
