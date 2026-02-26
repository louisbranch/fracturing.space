---
name: architecture-refactor
description: Parallel-path refactor workflow for clean boundary cutovers
user-invocable: true
---

# Architecture-First Refactor

Use this skill when incremental edits are making structure worse, coupling is
spreading, or compatibility glue is piling up.

## When to Use

- A requested change crosses multiple packages or feature boundaries.
- Small edits require repeated adapters/wrappers to preserve old structure.
- A refactor has started to create duplicate roots or transitional file sprawl
  (for example, `feature_adapter_*` patterns without a clear deletion path).
- Existing package boundaries block clean naming, ownership, or test seams.

## Target Outcome

- Clear package boundaries that model domain/feature ownership.
- One active implementation path after cutover.
- Meaningful tests at stable seams.
- Durable rationale promoted to `docs/`.

## Workflow

1. Define target architecture
   - Name target packages/modules and responsibilities.
   - State explicit non-goals to prevent scope creep.
   - Identify stable contracts callers depend on.

2. Choose migration seams
   - Pick seam points (interfaces, handlers, application services) for the
     transition.
   - Decide what behavior must remain unchanged vs intentionally changed.

3. Build the new path in parallel
   - Implement the new package/module structure directly in the target shape.
   - Avoid mirroring legacy naming and layering into the new path.
   - Keep compatibility wrappers minimal and temporary.

4. Port behavior via contracts
   - Move behavior behind stable seams.
   - Add/adjust tests at those seams (unit/integration as appropriate).
   - Prefer durable assertions over implementation-detail checks.

5. Cut over callers
   - Switch callers to the new path in coherent slices.
   - Validate each slice with project verification commands.

6. Delete old path and shims
   - Remove legacy code paths, temporary adapters, and stale tests.
   - Confirm no duplicate authority remains.

7. Capture durable decisions
   - Promote boundary decisions and migration rationale to `docs/`.

## Rules

- Do not preserve internal compatibility by default.
- Temporary shims require explicit removal criteria in the plan.
- Prefer one clean cutover over indefinite dual-path operation.
- If a micro-fix worsens architecture, escalate to this workflow.

## Anti-Patterns to Avoid

- Adapter proliferation without removal criteria.
- Root-level duplication of feature stacks that stay coupled.
- Refactors that only move files while preserving problematic boundaries.
- Tests that lock old implementation details instead of new contracts.

## Verification

- Run `make test` and `make integration` after code changes.
- Run `make cover` when production behavior changes and report notable impact.

## Completion Checklist

- Target boundaries are visible in package/module layout.
- Callers use the new path.
- Legacy path and temporary shims are deleted.
- Meaningful seam tests pass.
- Documentation is updated for lasting decisions.

## Web/Web2 Migration Note

For `web` and `web2` style transitions, prefer a clean module path in parallel,
migrate route/service slices to that path, then delete old slice-by-slice
handlers and adapters once each cutover is validated.
