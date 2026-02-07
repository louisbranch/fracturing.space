# Game Systems Architecture

This document describes how Fracturing.Space supports multiple tabletop RPG systems through a pluggable architecture that separates system-agnostic core functionality from system-specific mechanics.

## Design Rationale

### Why Separate System-Agnostic from System-Specific State?

Fracturing.Space supports multiple TTRPGs (Daggerheart first, with architecture for D&D 5e, VtM, etc.). Each system has:

- **Unique resources**: Daggerheart has Hope/Stress/GM Fear; VtM has Blood Pool/Humanity; D&D 5e has Inspiration
- **Unique character attributes**: Daggerheart has damage thresholds; D&D has ability scores and saving throws
- **Unique mechanics**: Different dice systems, outcomes, and state transitions

By separating concerns, we can:
1. Share common infrastructure (campaigns, sessions, participants, event streams)
2. Allow systems to define their own state without polluting core abstractions
3. Add new systems without modifying existing code
4. Query and persist system-specific data with proper schema validation

### Why Behavior-Based Interfaces Over Data-Centric Interfaces?

Systems vary in *what data they track*, but share *operations* (heal, damage, spend/gain resources):

- A `ResourceHolder` interface lets Hope, Blood Pool, and Inspiration all be "resources" with gain/spend operations
- Code operates on behaviors, not on specific field names
- Adding a new resource type doesn't require changing core interfaces

**Example**: Instead of `GetHope()` and `SetHope()` methods specific to Daggerheart, we use:

```go
type ResourceHolder interface {
    GainResource(name string, amount int) (before, after int, err error)
    SpendResource(name string, amount int) (before, after int, err error)
    ResourceValue(name string) int
    ResourceCap(name string) int
}
```

This allows any system to define its own named resources that all work with the same interface.

## Architecture Overview

```
+---------------------------------------------------------------------+
|                       SYSTEM-AGNOSTIC LAYER                         |
+---------------------------------------------------------------------+
| internal/state/                                                     |
|   +-- campaign/      Campaign metadata (name, status, theme)        |
|   +-- participant/   Players and GM management                      |
|   +-- character/     Character identity (name, kind, notes)         |
|   +-- session/       Session lifecycle, event stream                |
|                                                                     |
| internal/core/                                                      |
|   +-- dice/          Generic dice rolling primitives                |
|   +-- check/         Difficulty check primitives                    |
|   +-- random/        RNG seed generation                            |
|                                                                     |
| api/proto/state/v1/  System-agnostic state protos                   |
|   +-- Campaign, Session, Character identity                         |
+---------------------------------------------------------------------+
                              |
                    GameSystem interface
                    (ID, Name, Handlers)
                              |
+---------------------------------------------------------------------+
|                      SYSTEM-SPECIFIC LAYER                          |
+---------------------------------------------------------------------+
| internal/systems/{system}/                                          |
|   +-- domain/        Mechanics (dice, outcomes, probability)        |
|   +-- state.go       CharacterState, SnapshotState implementations |
|   +-- content/       Compendium data (classes, items, etc.)         |
|                                                                     |
| internal/api/grpc/systems/{system}/                                 |
|   +-- service.go     gRPC service for system mechanics              |
|                                                                     |
| api/proto/systems/{system}/v1/                                      |
|   +-- mechanics.proto   Dice, outcomes                              |
|   +-- state.proto       CharacterState, SnapshotState              |
|   +-- service.proto     gRPC service definition                     |
|                                                                     |
| internal/storage/sqlite/migrations/                                 |
|   +-- 00X_{system}_tables.sql   Extension tables                    |
+---------------------------------------------------------------------+
```

## Core Interfaces

### GameSystem Interface

Every game system must implement the `GameSystem` interface in `internal/systems/registry.go`:

```go
type GameSystem interface {
    // ID returns the system identifier (matches GameSystem proto enum)
    ID() commonv1.GameSystem

    // Name returns the human-readable system name
    Name() string

    // StateFactory returns the factory for creating system-specific state
    StateFactory() StateFactory

    // OutcomeApplier returns the handler for applying roll outcomes
    OutcomeApplier() OutcomeApplier
}
```

### Behavior Interfaces

Systems implement behaviors they support:

```go
// Healable represents entities that can be healed
type Healable interface {
    Heal(amount int) (before, after int)
    MaxHP() int
}

// Damageable represents entities that can take damage
type Damageable interface {
    TakeDamage(amount int) (before, after int)
    CurrentHP() int
}

// ResourceHolder represents entities with named resources
type ResourceHolder interface {
    GainResource(name string, amount int) (before, after int, err error)
    SpendResource(name string, amount int) (before, after int, err error)
    ResourceValue(name string) int
    ResourceCap(name string) int
}
```

### State Factory

Creates initial state for characters and snapshots:

```go
type StateFactory interface {
    // NewCharacterState creates initial character state
    NewCharacterState(campaignID, characterID string, kind CharacterKind) (CharacterStateHandler, error)

    // NewSnapshotState creates initial snapshot state for a campaign
    NewSnapshotState(campaignID string) (SnapshotStateHandler, error)
}
```

### Outcome Applier

Handles applying roll outcomes to game state:

```go
type OutcomeApplier interface {
    // ApplyOutcome applies a roll outcome to the game state
    ApplyOutcome(ctx context.Context, outcome OutcomeContext) ([]StateChange, error)
}
```

## Guide: Adding a New Game System

This guide walks through adding a new game system using Vampire: The Masquerade (VtM) as an example.

### Step 1: Add the System Enum

Add the enum value to `api/proto/common/v1/game_system.proto`:

```protobuf
enum GameSystem {
  GAME_SYSTEM_UNSPECIFIED = 0;
  GAME_SYSTEM_DAGGERHEART = 1;
  GAME_SYSTEM_VTM = 2;  // Add your system here
}
```

Run `make proto` to regenerate Go code.

### Step 2: Create the System Package

Create `internal/systems/vtm/vtm.go`:

```go
package vtm

import (
    commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
    "github.com/louisbranch/fracturing.space/internal/systems"
)

func init() {
    systems.DefaultRegistry.Register(&System{})
}

type System struct{}

func (s *System) ID() commonv1.GameSystem {
    return commonv1.GameSystem_GAME_SYSTEM_VTM
}

func (s *System) Name() string {
    return "Vampire: The Masquerade"
}

// Ensure System implements GameSystem
var _ systems.GameSystem = (*System)(nil)
```

### Step 3: Define System-Specific State

Create `internal/systems/vtm/state.go` implementing the behavior interfaces:

```go
package vtm

import "fmt"

const (
    BloodPoolMax = 10  // Varies by generation
    HumanityMax  = 10
)

// VtMCharacterState implements Healable, Damageable, ResourceHolder
type VtMCharacterState struct {
    campaignID   string
    characterID  string
    health       int  // Health levels (0-7 for most vampires)
    healthMax    int
    willpower    int
    willpowerMax int
    bloodPool    int
    bloodPoolMax int
    humanity     int
}

// Healable implementation
func (s *VtMCharacterState) Heal(amount int) (before, after int) {
    before = s.health
    s.health = min(s.health+amount, s.healthMax)
    return before, s.health
}

func (s *VtMCharacterState) MaxHP() int {
    return s.healthMax
}

// Damageable implementation
func (s *VtMCharacterState) TakeDamage(amount int) (before, after int) {
    before = s.health
    s.health = max(s.health-amount, 0)
    return before, s.health
}

func (s *VtMCharacterState) CurrentHP() int {
    return s.health
}

// ResourceHolder implementation
func (s *VtMCharacterState) GainResource(name string, amount int) (before, after int, err error) {
    switch name {
    case "blood_pool":
        before = s.bloodPool
        s.bloodPool = min(s.bloodPool+amount, s.bloodPoolMax)
        return before, s.bloodPool, nil
    case "willpower":
        before = s.willpower
        s.willpower = min(s.willpower+amount, s.willpowerMax)
        return before, s.willpower, nil
    case "humanity":
        before = s.humanity
        s.humanity = min(s.humanity+amount, HumanityMax)
        return before, s.humanity, nil
    default:
        return 0, 0, fmt.Errorf("unknown VtM resource: %s", name)
    }
}

func (s *VtMCharacterState) SpendResource(name string, amount int) (before, after int, err error) {
    switch name {
    case "blood_pool":
        if s.bloodPool < amount {
            return 0, 0, fmt.Errorf("insufficient blood pool")
        }
        before = s.bloodPool
        s.bloodPool -= amount
        return before, s.bloodPool, nil
    case "willpower":
        if s.willpower < amount {
            return 0, 0, fmt.Errorf("insufficient willpower")
        }
        before = s.willpower
        s.willpower -= amount
        return before, s.willpower, nil
    default:
        return 0, 0, fmt.Errorf("unknown VtM resource: %s", name)
    }
}

func (s *VtMCharacterState) ResourceValue(name string) int {
    switch name {
    case "blood_pool":
        return s.bloodPool
    case "willpower":
        return s.willpower
    case "humanity":
        return s.humanity
    default:
        return 0
    }
}

func (s *VtMCharacterState) ResourceCap(name string) int {
    switch name {
    case "blood_pool":
        return s.bloodPoolMax
    case "willpower":
        return s.willpowerMax
    case "humanity":
        return HumanityMax
    default:
        return 0
    }
}
```

### Step 4: Create System-Specific Protos

Create `api/proto/systems/vtm/v1/state.proto`:

```protobuf
syntax = "proto3";

package systems.vtm.v1;

option go_package = "github.com/louisbranch/fracturing.space/api/gen/go/systems/vtm/v1;vtmv1";

// VtM-specific character profile extensions
message VtMProfile {
  int32 generation = 1;      // Vampire generation (determines blood pool max)
  string clan = 2;           // Clan (Brujah, Ventrue, etc.)
  int32 blood_pool_max = 3;  // Max blood pool based on generation
}

// VtM-specific character state
message VtMCharacterState {
  int32 health = 1;          // Health levels
  int32 willpower = 2;       // Willpower points
  int32 blood_pool = 3;      // Current blood pool
  int32 humanity = 4;        // Humanity score
}

// VtM-specific snapshot state (campaign-level)
message VtMSnapshot {
  // Domain influence, sect politics, etc.
  map<string, int32> domain_influence = 1;
}
```

### Step 5: Create Extension Tables

Create `internal/storage/sqlite/migrations/003_vtm_tables.sql`:

```sql
-- VtM character profile extensions
CREATE TABLE vtm_character_profiles (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    generation INTEGER NOT NULL DEFAULT 13,
    clan TEXT NOT NULL DEFAULT '',
    blood_pool_max INTEGER NOT NULL DEFAULT 10,
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);

-- VtM character state
CREATE TABLE vtm_character_states (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    health INTEGER NOT NULL DEFAULT 7,
    willpower INTEGER NOT NULL DEFAULT 5,
    blood_pool INTEGER NOT NULL DEFAULT 10,
    humanity INTEGER NOT NULL DEFAULT 7,
    PRIMARY KEY (campaign_id, character_id)
);

-- VtM snapshot state
CREATE TABLE vtm_snapshots (
    campaign_id TEXT PRIMARY KEY,
    FOREIGN KEY (campaign_id)
        REFERENCES campaigns(id) ON DELETE CASCADE
);

-- VtM domain influence (one row per faction)
CREATE TABLE vtm_domain_influence (
    campaign_id TEXT NOT NULL,
    faction TEXT NOT NULL,
    influence INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (campaign_id, faction),
    FOREIGN KEY (campaign_id)
        REFERENCES vtm_snapshots(campaign_id) ON DELETE CASCADE
);
```

### Step 6: Add sqlc Queries

Create query files in `internal/storage/sqlite/queries/vtm/`:

```sql
-- name: GetVtMCharacterState :one
SELECT * FROM vtm_character_states
WHERE campaign_id = ? AND character_id = ?;

-- name: UpsertVtMCharacterState :exec
INSERT INTO vtm_character_states (campaign_id, character_id, health, willpower, blood_pool, humanity)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (campaign_id, character_id) DO UPDATE SET
    health = excluded.health,
    willpower = excluded.willpower,
    blood_pool = excluded.blood_pool,
    humanity = excluded.humanity;
```

### Step 7: Implement gRPC Service

Create `internal/api/grpc/systems/vtm/service.go` following the pattern in `internal/api/grpc/systems/daggerheart/service.go`.

### Step 8: Add MCP Tools (Optional)

Create `internal/mcp/domain/vtm.go` following the pattern in `internal/mcp/domain/daggerheart.go`.

### Step 9: Add Integration Tests

Add tests in `internal/test/integration/` to verify the full stack works.

## Reference: Daggerheart Implementation

Daggerheart is the reference implementation for new systems:

| Component | Location |
|-----------|----------|
| System registration | `internal/systems/daggerheart/daggerheart.go` |
| State handlers | `internal/systems/daggerheart/state.go` |
| Domain logic | `internal/systems/daggerheart/domain/` |
| Proto definitions | `api/proto/systems/daggerheart/v1/` |
| gRPC service | `internal/api/grpc/systems/daggerheart/service.go` |
| Extension tables | `internal/storage/sqlite/migrations/002_daggerheart_tables.sql` |

### Daggerheart Resources

Daggerheart defines these resources via the `ResourceHolder` interface:

| Resource | Range | Scope |
|----------|-------|-------|
| Hope | 0-6 | Character |
| Stress | 0-StressMax | Character |
| GM Fear | 0-12 | Campaign (Snapshot) |

### Daggerheart Character Profile Extensions

- `stress_max`: Maximum stress before breakdown
- `evasion`: Target number for attacks
- `major_threshold`: Damage threshold for major wounds
- `severe_threshold`: Damage threshold for severe wounds

## Proto Extension Pattern

System-specific state is added to core protos using `oneof`:

```protobuf
// In state/v1/character.proto
message CharacterState {
  string campaign_id = 1;
  string character_id = 2;
  int32 hp = 3;  // HP is common across systems

  // System-specific state extension
  oneof system_state {
    systems.daggerheart.v1.DaggerheartCharacterState daggerheart = 10;
    systems.vtm.v1.VtMCharacterState vtm = 11;
  }
}
```

This approach:
- Maintains type safety in proto/gRPC layer
- Allows backward-compatible additions
- Keeps system-specific fields out of the core messages

## Storage Pattern

Extension tables follow a consistent pattern:

1. **Primary key**: `(campaign_id, character_id)` for character tables, `campaign_id` for snapshot tables
2. **Foreign key**: References the core table with `ON DELETE CASCADE`
3. **Defaults**: Sensible defaults for all fields (new characters work immediately)

Tables are created via numbered migrations (e.g., `002_daggerheart_tables.sql`, `003_vtm_tables.sql`).

## Testing a New System

1. Run `make proto` after proto changes
2. Run `make test` to verify unit tests pass
3. Run `make integration` to verify full stack
4. Run `make seed` to test with seed scenarios

Create system-specific seed scenarios in `internal/test/integration/fixtures/seed/` to exercise the full MCP -> gRPC -> SQLite path.
