---
title: "Goimports Formatting Baseline"
---

## Description
Normalize formatting across the Go codebase so future changes start from a consistent baseline.

## Tasks
- Run goimports across the repository.
- Add a formatting check to CI.

## Next steps
- Keep goimports in the dev workflow via `make fmt-check` or CI.

## Out of scope
- Coverage policy changes.
