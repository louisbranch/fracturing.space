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
3. Run baseline verification:
   - `make test`
   - `make runtime-smoke`
4. Before opening or updating a PR:
   - `make verify-pr`
5. When production behavior changes, check coverage:
   - `make cover`
   - `make cover-critical-domain` for game-domain changes
6. Run docs checks for docs-heavy changes:
   - `make docs-check`

## Good to know

- File/package routing: [Contributor map](../reference/contributor-map.md)
- Extension workflow: [Adding a command/event/system](../guides/adding-command-event-system.md)
- Web module patterns: [Web module playbook](../guides/web-module-playbook.md)

## Reference

- Architecture start: [Architecture index](../architecture/index.md)
- Runtime operations: [Running index](../running/index.md)
