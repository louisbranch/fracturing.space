# Daggerheart Audit Workspace

This directory holds generated and working audit artifacts for the Daggerheart
reference audit.

Files:

- `inventory.json`: normalized copy of the external reference index.
- `audit_matrix.json`: one working audit row per reference item.
- `rule_clauses.json`: clause-level split for rule, glossary, and playbook
  sources.
- `summary.json`: generated counts by kind, audit area, and normativity.
- `epics.json`: synthesized remediation epics with row coverage, scope, and
  implementation boundaries.
- `remediation_backlog.md`: reader-first rendering of the synthesized epics.
- `seed-findings.md`: confirmed starting findings from initial reconnaissance.

The generator lives at `go run ./internal/tools/daggerheartaudit`.

Recommended commands:

```bash
go run ./internal/tools/daggerheartaudit generate
go run ./internal/tools/daggerheartaudit check
```

These artifacts intentionally live under `.agents/plans/` because they are
working memory for the audit. Durable conclusions should be promoted to `docs/`
or follow-up implementation plans after review.
