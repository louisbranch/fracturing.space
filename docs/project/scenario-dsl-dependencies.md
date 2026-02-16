---
title: "Scenario DSL Dependencies"
parent: "Project"
nav_order: 21
---

# Scenario DSL Dependencies

This document records scenario DSL gaps found by comparing `internal/test/game/scenarios/*.lua` against the Lua bindings in `internal/tools/scenario/dsl.go` and `internal/test/game/lua_binding_test.go`.

## Missing DSL Bindings

None currently detected (last checked 2026-02-16).

Scope note: this report is limited to missing symbol bindings between scenario fixtures and registered DSL methods. It does not capture higher-level behavior placeholders or ambiguous mechanics flagged in comments.

For behavior-marker reconciliation, run `make scenario-missing-doc-check`.
