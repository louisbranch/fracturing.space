---
title: "Daggerheart Live Mechanics Matrix"
parent: "Reference"
nav_order: 27
status: canonical
owner: engineering
last_reviewed: "2026-03-22"
---

# Daggerheart Live Mechanics Matrix

Latest live acceptance matrix for the Daggerheart AI mechanics surface. This
table is tool-centric so future model runs can add new columns without changing
the shape of the document.

For the follow-on AI-service design guidance derived from these results, see
[Campaign AI Mechanics Quality](../architecture/platform/campaign-ai-mechanics-quality.md).

## Status enums

- `clean_pass`: lane passed with no recorded tool-call errors
- `pass_with_tool_errors`: lane passed, but the accepted run included at least
  one failed tool call before recovery
- `fail_tool_loop_limit`: lane failed because the orchestration loop exhausted
  its tool budget
- `fail_timeout`: lane failed because the live run timed out
- `fail_precondition_gap`: lane failed because setup/runtime preconditions did
  not permit the intended mechanic path
- `not_run`: no accepted live run recorded for that model yet

## Latest accepted runs

Post-fix live acceptance was executed on March 22, 2026 with `gpt-5-mini`.
Accepted artifacts are written locally under `.tmp/ai-live-captures/` as raw
captures, markdown reports, and `.summary.json` summaries.

| Tool | Source scenario | `gpt-5-mini` | Notes |
| --- | --- | --- | --- |
| `character_sheet_read` | `CapabilityLookup` | `clean_pass` | Character-capability read succeeded before scene framing. |
| `interaction_state_read` | `AttackReview` | `clean_pass` | Added as the primary diagnosis read for board-sensitive attack lanes. |
| `daggerheart_combat_board_read` | `SpotlightBoardReview` | `clean_pass` | Clean board-control run used no reference lookups and re-read the board after adversary/countdown updates. |
| `daggerheart_action_roll_resolve` | `MechanicsReview` | `clean_pass` | Action resolution plus GM review handoff completed cleanly. |
| `daggerheart_gm_move_apply` | `GMMovePlacementReview` | `clean_pass` | Clean placement run used a single direct-move spend target with no recovery call. |
| `daggerheart_adversary_create` | `GMMovePlacementReview` | `clean_pass` | Same clean placement lane as `daggerheart_gm_move_apply`. |
| `daggerheart_scene_countdown_create` | `CountdownTriggerReview` | `clean_pass` | Latest accepted run used fixed-start countdown creation and full trigger lifecycle. |
| `daggerheart_scene_countdown_advance` | `CountdownTriggerReview` | `clean_pass` | Countdown advanced to `TRIGGER_PENDING` cleanly. |
| `daggerheart_scene_countdown_resolve_trigger` | `CountdownTriggerReview` | `clean_pass` | Trigger resolution and board reread both completed in the accepted run. |
| `daggerheart_adversary_update` | `SpotlightBoardReview` | `clean_pass` | Same clean board-control lane as `daggerheart_combat_board_read`. |
| `daggerheart_attack_flow_resolve` | `AttackReview` | `clean_pass` | Default-profile attack flow succeeded after board/state diagnosis hardening. |
| `daggerheart_adversary_attack_flow_resolve` | `AdversaryAttackReview` | `clean_pass` | Adversary attack, memory update, and review resolution completed cleanly. |
| `daggerheart_group_action_flow_resolve` | `GroupActionReview` | `clean_pass` | Cooperative action lane passed once fixture names and extra-character setup were corrected. |
| `daggerheart_reaction_flow_resolve` | `ReactionReview` | `clean_pass` | Reaction flow completed cleanly. |
| `daggerheart_tag_team_flow_resolve` | `TagTeamReview` | `clean_pass` | Tag-team lane passed after prompt alignment with fixture character names. |
| `system_reference_search` | `PlaybookAttackReview` | `clean_pass` | Clean playbook lane used exactly one intentional search before the combat flow. |
| `system_reference_read` | `PlaybookAttackReview` | `clean_pass` | Same clean playbook lane as `system_reference_search`. |

## Scenario cleanliness

These rows are scenario-level, not tool-level. They exist so a lane can be
tracked as clean even when other runs for the same tools had recoverable
errors.

| Scenario | `gpt-5-mini` | Reference usage | Notes |
| --- | --- | --- | --- |
| `PlaybookAttackReview` | `clean_pass` | `1 search / 1 read` | Uses the bounded playbook lane: one explicit playbook consult, then sheet, board, attack flow, and review resolution. |
| `SpotlightBoardReview` | `clean_pass` | `0 search / 0 read` | Uses the short always-on guidance only; no reference lookup. |
| `GMMovePlacementReview` | `clean_pass` | `0 search / 0 read` | Uses the short always-on guidance only; no reference lookup. |

## Notes for future runs

- Add one column per model; do not replace older model columns.
- Use the latest accepted `.summary.json` artifact for each scenario/model pair.
- For scenario-level cleanliness, prefer the latest clean accepted summary, not
  merely the latest attempted run.
- If a newer run fails, do not overwrite a previous accepted status without an
  explicit product or engineering decision.
