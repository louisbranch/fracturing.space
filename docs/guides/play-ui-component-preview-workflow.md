---
title: "Play UI Component Storybook Workflow"
parent: "Guides"
nav_order: 6
---

# Play UI Component Storybook Workflow

Use this workflow when building or revising isolated React components for the
`play` service UI.

## Purpose

The `play` UI now uses a Storybook-first component workflow for MVP frontend
development:

- isolated component review runs in Storybook inside the existing frontend package,
- component contracts stay explicit and prop-driven,
- runtime bootstrapping is not required for component iteration,
- fixtures and stories double as contributor-facing documentation.

The current catalog includes both:

- interaction workflow slices for active-play concepts such as scenes, phases,
  acting sets, player slots, OOC overlays, AI turn state, and the composition
  shell
- Daggerheart reference surfaces such as the character card and character sheet

## Run Storybook

Install the UI workspace dependencies first:

```sh
cd internal/services/play/ui
npm ci
```

Start Storybook:

```sh
npm run storybook
```

Open the component catalog:

```text
http://localhost:6006
```

`/` on the play service now points contributors to Storybook, and
`/preview/character-card` has been intentionally retired.

## Where to work

The Daggerheart reference slices live under:

- `internal/services/play/ui/src/systems/daggerheart/character-card/`
- `internal/services/play/ui/src/systems/daggerheart/character-sheet/`

The active interaction workflow slices live under:

- `internal/services/play/ui/src/interaction/player-hud/` (v2 — navbar shell)
- `internal/services/play/ui/src/interaction/legacy-player-hud/` (v1 — composer-focused)
- older interaction workflow slices that are pending removal now live under
  `internal/services/play/ui/src/interaction/legacy/`

Keep concerns separate inside each component slice:

- `contract.ts`
  exported prop and data types
- `fixtures.ts`
  canonical mock characters shared by stories and tests
- `CharacterCardOverview.stories.tsx`
  side-by-side reference matrix for the supported variants
- `CharacterCardVariants.stories.tsx`
  variant-only stories with a fixed fixture and realistic screen framing
- `CharacterCardFixtures.stories.tsx`
  fixture-only stories using the canonical mock data
- `CharacterCard.tsx`
  actual component implementation
- `StoryStage.tsx`
  shared Storybook-only screen wrappers for realistic card presentation

## Contributor rules

- Keep component inputs explicit and prop-driven.
- Reuse canonical fixtures instead of scattering inline mock objects.
- When a component is derived from an existing product surface, document that
  source in story descriptions and fixture comments.
- Let Storybook own navigation; do not rebuild fixture or variant selectors inside the component canvas.
- Keep alternate implementations behind the same exported component contract.
- Delete obsolete UI paths and tests once the new slice fully replaces them.

## Testing

The Character Card workflow uses the same conceptual inputs in both stories and
tests:

- component rendering tests live in Vitest/RTL next to the component slice
- Storybook stories reuse the same canonical fixtures and component contract
- `npm run build-storybook` verifies that the isolated documentation surface still builds

Use the package checks during iteration:

```sh
cd internal/services/play/ui
npm test
npm run typecheck
npm run build
npm run build-storybook
```

## Extending the pattern

When adding the next isolated component:

1. create a system-owned component slice with its own contract, fixtures, and stories
   for Daggerheart reference surfaces, or an interaction-owned slice under
   `src/interaction/player-hud/` for active HUD workflow surfaces (v2),
   or `src/interaction/legacy-player-hud/` for v1 surfaces pending migration
2. add Storybook stories that clearly separate overview, variants, and fixtures
3. write component tests against the exported contract, not runtime internals
4. remove temporary or superseded UI code instead of preserving compatibility by default
