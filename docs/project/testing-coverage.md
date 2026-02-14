# Testing Coverage Notes

This note captures ongoing coverage work and heuristics used during the test drive.

## Coverage heuristics
- Prioritize domain logic with branch-heavy behavior (policy checks, translation, resource handling).
- Prefer high-risk paths: validation, parsing, error mapping, storage adapters, and integrity checks.
- Avoid spending time on doc-only or constant-only packages where coverage percent is misleading.

## Recent focus areas
- Campaign projection applier branch coverage (event application paths).
- Daggerheart state resource handling and clamping behavior.
- Core filter parsing/translation edge cases.
- Campaign policy operation validation and label mapping.
- Auth storage SQLite edge cases (nil DB guard, validation, stats).
- CLI entrypoints extracted into internal/cmd with unit coverage.
- Seed tooling moved under internal/tools/seed with tests carried over.

## Remaining candidates (low effort)
- internal/services/game/domain/narrative (no tests)
- internal/services/game/storage (interface-level or thin helpers)
- internal/services/auth (no tests)
- internal/services/game/api/grpc (public API validation helpers)

## Verification
- Run `make test` for unit coverage changes.
- Run `make integration` after code changes (required).
