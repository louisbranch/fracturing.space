---
title: "Contributors"
parent: "Audience"
nav_order: 1
---

# Contributors

Canonical onboarding path for developers adding features, fixing bugs, or improving docs.

## Need to know

1. Choose a runtime path: [Quickstart (Docker)](../running/quickstart.md) or [Local development (Go)](../running/local-dev.md).
2. Follow the contribution workflow: [Contributing guide](https://github.com/louisbranch/fracturing.space/blob/main/CONTRIBUTING.md).
3. Use the supported verification surface:
   - `make test` during normal implementation
   - `make smoke` when runtime paths need quick feedback
   - `make check` before opening or updating a PR
4. If you are changing the web service, read the [Web contributor map](../architecture/platform/web-contributor-map.md) and run:
   - `make web-architecture-check`
5. Use `make cover` or `make cover-critical-domain` only when you need focused standalone coverage output separate from `make check`.
6. Run docs checks for docs-heavy changes:
   - `make docs-check`

## Good to know

- File/package routing: [Contributor map](../reference/contributor-map.md)
- Game service routing and test placement: [Game service contributor map](../reference/game-service-contributor-map.md)
- Extension workflow: [Adding a command/event/system](../guides/adding-command-event-system.md)
- Web architecture boundaries: [Web architecture](../architecture/platform/web-architecture.md)
- Web routing and ownership map: [Web contributor map](../architecture/platform/web-contributor-map.md)
- Web module patterns: [Web module playbook](../guides/web-module-playbook.md)

## Reference

- Architecture start: [Architecture index](../architecture/index.md)
- Runtime operations: [Running index](../running/index.md)
- Verification workflow: [Verification commands](../running/verification.md)
