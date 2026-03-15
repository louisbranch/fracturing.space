# Session 26: Cross-Cutting Architectural Synthesis

## Status: `complete`

## Review Summary

Across 25 sessions reviewing ~1,063 Go files and ~196k LOC (86k production + 110k test), the game service demonstrates strong fundamentals: clean event-sourcing architecture, proper CQRS separation, well-designed registries with startup validation, correct Go patterns (interfaces at consumer, generics for type safety), and comprehensive testing. The service is ready for external contributors with targeted improvements.

## Aggregate Finding Counts

| Session | Critical | High | Medium | Low | Info | Total |
|---------|----------|------|--------|-----|------|-------|
| 1. Domain Primitives | 0 | 0 | 3 | 5 | 2 | 10 |
| 2. Registries | 0 | 0 | 1 | 4 | 5 | 10 |
| 3. Aggregate/Fold | 0 | 0 | 2 | 2 | 6 | 10 |
| 4. Campaign/Participant | 0 | 0 | 3 | 2 | 5 | 10 |
| 5. Character/Invite/Action | 0 | 0 | 0 | 1 | 6 | 7 |
| 6. Session/Scene | 0 | 2 | 4 | 0 | 2 | 8 |
| 7. Engine | 0 | 0 | 1 | 0 | 7 | 8 |
| 8. Engine Registries | 0 | 0 | 0 | 2 | 4 | 6 |
| 9. Module/Bridge | 0 | 0 | 0 | 0 | 6 | 6 |
| 10. Cross-Aggregate | 0 | 0 | 0 | 1 | 5 | 6 |
| 11. Core Utilities | 0 | 0 | 0 | 1 | 4 | 5 |
| 12. Daggerheart Domain | 0 | 1 | 4 | 1 | 1 | 7 |
| 13. Daggerheart Mechanics | 0 | 1 | 2 | 0 | 3 | 6 |
| 14. Storage Contracts | 0 | 0 | 1 | 1 | 4 | 6 |
| 15. Storage SQLite | 0 | 2 | 2 | 0 | 1 | 5 |
| 16. Storage Daggerheart | 0 | 0 | 1 | 0 | 2 | 3 |
| 17. Projection | 0 | 1 | 1 | 0 | 4 | 6 |
| 18. App Bootstrap | 0 | 0 | 2 | 0 | 4 | 6 |
| 19. API Infrastructure | 0 | 0 | 2 | 1 | 3 | 6 |
| 20. API Handler | 0 | 1 | 0 | 1 | 3 | 5 |
| 21. API Transport Part 1 | 0 | 1 | 0 | 0 | 4 | 5 |
| 22. API Transport Part 2 | 1 | 0 | 2 | 1 | 1 | 5 |
| 23. Daggerheart API Service | 0 | 1 | 2 | 0 | 1 | 4 |
| 24. Daggerheart API Transports | 0 | 1 | 1 | 0 | 3 | 5 |
| 25. Integration/Observability | 0 | 0 | 0 | 2 | 3 | 5 |
| **TOTAL** | **1** | **11** | **34** | **25** | **92** | **163** |

## Prioritized Action Plan

### Priority 1: Critical — Immediate Decomposition (1 finding)

#### P1.1: Split interaction_application.go (1,354 lines)
- **Session**: 22
- **Impact**: Largest production file in the entire service. Handles 4 distinct interaction types in one file.
- **Action**: Split into `interaction_gate.go`, `interaction_spotlight.go`, `interaction_ooc.go`, `interaction_ai_turn.go`.
- **Effort**: Medium (mechanical split, existing tests guide correctness)

### Priority 2: High — God Package Decomposition (11 findings)

#### P2.1: Split domain/bridge/daggerheart/ root package (67+ files)
- **Session**: 12
- **Impact**: Largest package. Contributors can't find relevant files.
- **Action**: Extract sub-packages: `decider/`, `fold/`, `mechanics/`.

#### P2.2: Split storage/sqlite/coreprojection/ (41 files)
- **Session**: 15
- **Impact**: God storage package covering all entities.
- **Action**: Split into per-entity store packages.

#### P2.3: Split api/grpc/systems/daggerheart/ root (60+ files)
- **Session**: 23
- **Impact**: Flat package with 60+ files including 42 test files.
- **Action**: Move root handlers to sub-packages.

#### P2.4: Split session domain aggregate (33 files, registry at 648 lines)
- **Session**: 6
- **Impact**: Most complex aggregate. Gate subsystem alone is 8 files.
- **Action**: Extract `session/gate/` sub-package; split registry by concern.

#### P2.5: Split store_events.go (908 lines)
- **Session**: 15
- **Impact**: Core event journal in one file.
- **Action**: Split into append/query/replay/integrity files.

#### P2.6: Generate or split gametest/fakes.go (1,523 lines)
- **Session**: 20
- **Impact**: Hand-written fakes drift from interfaces.
- **Action**: Use mock generator or split per-interface.

#### P2.7: Split apply_interaction.go (531 lines)
- **Session**: 17
- **Impact**: Mirrors session complexity in projection layer.
- **Action**: Split by concern (gate/spotlight/ooc/ai_turn).

#### P2.8: Split large Daggerheart transport handlers (6 files over 600 lines)
- **Session**: 24
- **Impact**: 6 handler files (608-951 lines) are hard to navigate.
- **Action**: Split each by RPC method groups.

#### P2.9: Split campaigntransport/ (31 files)
- **Session**: 21
- **Impact**: Too many concerns in one transport package.
- **Action**: Extract AI/readiness/fork sub-packages.

#### P2.10: Move contentstore/projectionstore out of domain
- **Session**: 13
- **Impact**: Storage contracts inside domain layer violates dependency direction.
- **Action**: Move to `storage/contracts_daggerheart.go`.

#### P2.11: Split applier_test.go (3,940 lines)
- **Session**: 17
- **Impact**: Largest test file. Hard to navigate.
- **Action**: Split per-entity test files.

### Priority 3: Medium — Structural Improvements (34 findings)

Key medium-priority actions grouped by theme:

#### Naming and Documentation
- Fix commandids doc.go misleading description (S1-F4)
- Document SpotlightType canonical location (S6-F3)
- Rename daggerheart/domain/ sub-package to avoid collision (S13-F3)
- Document interceptor chain ordering (S19-F3)
- Add "frozen" warning to encoding/canonical.go (S11-F3)
- Add compat file removal criteria (S12-F5)

#### State Structure
- Group AI turn fields into sub-struct on session.State (S6-F5)
- Extract shared GateState struct for session/scene (S6-F7)
- Group player phase fields into sub-struct on scene.State (S6-F6)

#### Code Organization
- Split contracts_campaign_participant_invite_character.go per entity (S14-F1)
- Split adapter.go/payload.go/mechanics_manifest.go by concern (S12-F2/F3/F4)
- Extract decide.Flow helper duplication (S3-F8)
- Normalize scene event entity addressing (S3-F4)

#### Safety and Validation
- Document Registry thread safety during build phase (S2-F2)
- Verify session lock interceptor granularity and release (S19-F4)
- Verify authorization on all endpoints (S22-F4)

#### Testing
- Extract server test infrastructure to apptest/ (S18-F4)
- Add audit event conformance tests (S25-F3)
- Add fork semantics documentation (S10-F6)

### Priority 4: Low — Polish (25 findings)

These are documentation improvements, minor naming consistency fixes, and optional refactors that improve clarity but don't affect correctness or contributor experience significantly.

## Dependency Graph Assessment

- **No import cycles detected**: The package structure is acyclic. `ids` and `core/` are leaf packages. Domain packages import only `ids`, `event`, `command`, and `core/`. Storage imports domain for enum types (acceptable). API imports domain and storage. No reverse dependencies.
- **Fan-out**: The `app/` bootstrap and `api/grpc/game/` handler packages have the highest fan-out (they wire everything). This is expected for composition roots.

## Layer Boundary Violations

- **Storage → domain enum coupling**: Storage contracts import `campaign.Status`, `participant.Role`, etc. This is pragmatic and acceptable for stable value types.
- **contentstore/projectionstore inside domain**: This is the only true layer violation. Storage contracts should not be defined in domain packages.
- **No transport → domain internal leakage**: Transport packages only use domain types through public APIs and interfaces.

## God Packages (Ranked by File Count)

| Files | Package | Layer | Action |
|-------|---------|-------|--------|
| 67+ | `domain/bridge/daggerheart/` | Domain | Split |
| 60+ | `api/grpc/systems/daggerheart/` | API | Move to sub-packages |
| 41 | `storage/sqlite/coreprojection/` | Storage | Split per-entity |
| 33 | `domain/session/` | Domain | Extract gate sub-package |
| 31 | `api/grpc/game/campaigntransport/` | API | Extract sub-packages |
| 26 | `domain/scene/` | Domain | Acceptable (distinct concern) |

## Breaking Change Recommendations

1. **Rename daggerheart `domain/` sub-package** → `types/` or `model/` to avoid naming collision with parent domain layer.
2. **Move system-specific IDs** (`AdversaryID`, `CountdownID`) out of core `ids` package.
3. **Normalize scene event EntityID** to always contain SceneID (not GateID), with GateID in payload.
4. **Move storage contracts** out of domain layer for Daggerheart.

## Strengths to Preserve

1. **Event-sourcing fundamentals**: Fold/decide/replay pipeline is well-designed
2. **Registry + startup validation**: Catches misconfiguration before first request
3. **Interface-at-consumer pattern**: Proper Go interface placement throughout
4. **Consistent patterns**: Same patterns across core and system packages
5. **Test infrastructure**: Architecture tests protect structural invariants
6. **Documentation**: Doc comments are thorough with "why" context
7. **Decision/Rejection model**: Clean separation of accepted mutations from domain rejections
8. **Module system**: Game-system plugin architecture is well-abstracted

## Documentation Gaps for Contributors

1. **"How to add a new aggregate"** — Reference registries_builder.go pattern
2. **"How to add a new game system"** — Reference game-system skill + module testkit
3. **"Interceptor chain ordering"** — Document the required order
4. **"Event naming conventions"** — Past-tense events, imperative commands
5. **"State machine diagrams"** — Invite lifecycle, campaign status, session lifecycle
6. **"Forking semantics"** — What transfers, what resets
