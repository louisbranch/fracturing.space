# ExecPlan: Play Storybook Interaction Components

## Summary

Implement Storybook-first active-play interaction components in the `play` UI
package so contributors can review workflow slices with mocked data before a
live runtime exists.

## Tasks

- Add shared interaction view-model contracts and canonical workflow fixtures.
- Add isolated interaction component slices with stories and tests.
- Add one composition-focused interaction shell story that assembles the slices.
- Run UI package verification and adjust implementation until it passes.

## Out of Scope

- Live runtime transport, websocket, or fetch integration.
- Browser route changes outside the existing Storybook handoff.
- Game-service or play-app protocol wiring for real data.
