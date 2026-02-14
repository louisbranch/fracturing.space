# Game Integration Scenarios (Draft)

This note captures high-level game scenarios to implement as integration tests once fixtures and flows are ready.

## Scenario proposals
- **Campaign lifecycle**: create campaign, set GM mode, list/get, end, archive, restore.
- **Participant onboarding**: create invite, claim invite, participant bound/unbound events, list participants.
- **Session flow**: start session, append gameplay events, enforce session lock on write actions, end session.
- **Character flow**: create character, update profile, assign default control, retrieve character sheet.
- **Event replay + snapshot**: emit events, build snapshot, fork campaign at event boundary, verify replay consistency.
- **Daggerheart mechanics**: roll action dice, resolve outcome, apply roll outcome and update snapshot state.

## Notes
- Scenarios should exercise both gRPC entrypoints and domain invariants.
- Prefer deterministic data to enable stable replays and snapshot verification.
