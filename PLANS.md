# ExecPlans (Execution Plans)

This file defines how to write and maintain an execution plan (ExecPlan). ExecPlans are living, self-contained documents that enable a novice to implement a complex change without external context.

## When to use an ExecPlan

Use an ExecPlan for complex features or significant refactors. The plan is the single source of truth for both design and implementation steps. If you are authoring or updating a plan, read this file first and follow it to the letter.

ExecPlans live in the `plans/` directory and must remain in sync with this guidance.

## Non-negotiable requirements

- Every ExecPlan is fully self-contained and readable by a novice.
- ExecPlans are living documents: update them as you learn, change course, or finish milestones.
- Define every term of art in plain language or do not use it.
- Anchor the plan with observable outcomes and acceptance checks.
- Validate with real commands and expected outputs.

## Formatting

- ExecPlans are plain Markdown files.
- If the file contains only the ExecPlan, do not wrap it in triple backticks.
- Use prose-first descriptions; checklists are permitted only in the `Progress` section.

## Required sections (must be present)

- Purpose / Big Picture
- Progress (checkboxes with timestamps)
- Surprises & Discoveries
- Decision Log
- Outcomes & Retrospective
- Context and Orientation
- Plan of Work
- Concrete Steps
- Validation and Acceptance
- Idempotence and Recovery
- Artifacts and Notes
- Interfaces and Dependencies

## Guidance

- Be explicit: name files, modules, functions, and commands precisely.
- Describe user-visible behavior, not just code changes.
- Record every key decision with rationale and date.
- Use milestones as narrative checkpoints that are independently verifiable.
- Include small evidence snippets (test output, diffs) when progress is made.

## Skeleton

    # <Short, action-oriented description>

    This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

    If PLANS.md is checked in, reference it here and note that this document must be maintained in accordance with it.

    ## Purpose / Big Picture

    ## Progress

    - [ ] (YYYY-MM-DD HH:MMZ) ...

    ## Surprises & Discoveries

    ## Decision Log

    - Decision: ...
      Rationale: ...
      Date/Author: ...

    ## Outcomes & Retrospective

    ## Context and Orientation

    ## Plan of Work

    ## Concrete Steps

    ## Validation and Acceptance

    ## Idempotence and Recovery

    ## Artifacts and Notes

    ## Interfaces and Dependencies
