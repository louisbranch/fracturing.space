# MCP Tools and Resources

The MCP server communicates over stdio using JSON-RPC. Run it locally and point
your MCP client at the process stdin/stdout.

```sh
go run ./cmd/mcp
```

Alternatively, use the convenience script which resolves to the repo root automatically:

```sh
./scripts/mcp.sh
```

The gRPC server must be running at `localhost:8080` (or the configured address).

For an OpenCode client configuration, see `opencode.jsonc`.

## Tools

### Campaign Service Tools

#### campaign_create

Creates a new campaign metadata record.

**Input:**

```json
{
  "name": "The Lost Expedition",
  "gm_mode": "HUMAN",
  "theme_prompt": "A dark fantasy campaign set in a cursed forest"
}
```

**Output:**

```json
{
  "id": "camp_abc123",
  "name": "The Lost Expedition",
  "gm_mode": "HUMAN",
  "player_count": 0,
  "theme_prompt": "A dark fantasy campaign set in a cursed forest"
}
```

#### participant_create

Creates a participant (GM or player) for a campaign.

**Input:**

```json
{
  "campaign_id": "camp_abc123",
  "display_name": "Alice",
  "role": "PLAYER",
  "controller": "HUMAN"
}
```

**Output:**

```json
{
  "id": "part_xyz789",
  "campaign_id": "camp_abc123",
  "display_name": "Alice",
  "role": "PLAYER",
  "controller": "HUMAN",
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

#### actor_create

Creates an actor (PC or NPC) for a campaign.

**Input:**

```json
{
  "campaign_id": "camp_abc123",
  "name": "Thorin Ironforge",
  "kind": "PC",
  "notes": "Dwarf warrior with a mysterious past"
}
```

**Output:**

```json
{
  "id": "actor_def456",
  "campaign_id": "camp_abc123",
  "name": "Thorin Ironforge",
  "kind": "PC",
  "notes": "Dwarf warrior with a mysterious past",
  "created_at": "2025-01-15T10:35:00Z",
  "updated_at": "2025-01-15T10:35:00Z"
}
```

#### actor_control_set

Sets the default controller (GM or participant) for an actor in a campaign.

**Input:**

```json
{
  "campaign_id": "camp_abc123",
  "actor_id": "actor_def456",
  "controller": "part_xyz789"
}
```

**Output:**

```json
{
  "campaign_id": "camp_abc123",
  "actor_id": "actor_def456",
  "controller": "part_xyz789"
}
```

### Duality Service Tools

#### duality_rules_version

Returns the ruleset semantics used for Duality roll evaluation.

**Input:**

```json
{}
```

**Output:**

```json
{
  "system": "Daggerheart",
  "module": "Duality",
  "rules_version": "1.0.0",
  "dice_model": "Hope d12 + Fear d12",
  "total_formula": "hope + fear + modifier",
  "crit_rule": "Critical success when hope == fear",
  "difficulty_rule": "Total must meet or exceed difficulty",
  "outcomes": ["CRITICAL_SUCCESS", "SUCCESS", "FAILURE"]
}
```

#### duality_action_roll

Rolls Duality dice and returns the outcome with the roll context.

**Input:**

```json
{
  "modifier": 2,
  "difficulty": 15
}
```

**Output:**

```json
{
  "hope": 8,
  "fear": 5,
  "modifier": 2,
  "difficulty": 15,
  "total": 15,
  "is_crit": false,
  "meets_difficulty": true,
  "outcome": "SUCCESS"
}
```

#### duality_outcome

Evaluates a deterministic outcome from known Hope/Fear dice without rolling.

**Input:**

```json
{
  "hope": 8,
  "fear": 5,
  "modifier": 2,
  "difficulty": 15
}
```

**Output:**

```json
{
  "hope": 8,
  "fear": 5,
  "modifier": 2,
  "difficulty": 15,
  "total": 15,
  "is_crit": false,
  "meets_difficulty": true,
  "outcome": "SUCCESS"
}
```

#### duality_explain

Returns a deterministic explanation for a known Hope/Fear outcome.

**Input:**

```json
{
  "hope": 8,
  "fear": 5,
  "modifier": 2,
  "difficulty": 15,
  "request_id": "req_123"
}
```

**Output:**

```json
{
  "hope": 8,
  "fear": 5,
  "modifier": 2,
  "difficulty": 15,
  "total": 15,
  "is_crit": false,
  "meets_difficulty": true,
  "outcome": "SUCCESS",
  "rules_version": "1.0.0",
  "intermediates": {
    "base_total": 13,
    "total": 15,
    "is_crit": false,
    "meets_difficulty": true,
    "hope_gt_fear": true,
    "fear_gt_hope": false
  },
  "steps": [
    {
      "code": "CALCULATE_BASE",
      "message": "Calculate base total from dice",
      "data": {"hope": 8, "fear": 5, "base_total": 13}
    }
  ]
}
```

#### duality_probability

Computes exact outcome counts across all duality dice combinations.

**Input:**

```json
{
  "modifier": 2,
  "difficulty": 15
}
```

**Output:**

```json
{
  "total_outcomes": 144,
  "crit_count": 12,
  "success_count": 85,
  "failure_count": 47,
  "outcome_counts": [
    {"outcome": 0, "count": 12},
    {"outcome": 1, "count": 85},
    {"outcome": 2, "count": 47}
  ]
}
```

#### roll_dice

Rolls arbitrary dice pools and returns the individual results.

**Input:**

```json
{
  "dice": [
    {"sides": 20, "count": 2},
    {"sides": 6, "count": 1}
  ]
}
```

**Output:**

```json
{
  "rolls": [
    {
      "sides": 20,
      "results": [15, 8],
      "total": 23
    },
    {
      "sides": 6,
      "results": [4],
      "total": 4
    }
  ],
  "total": 27
}
```

## Resources

### Campaign Resources

#### campaigns://list

JSON listing of campaign metadata records. No dependencies.

Fields: `id`, `name`, `gm_mode`, `player_count`, `theme_prompt`, `created_at`, `updated_at`.

#### campaign://{campaign_id}/participants

JSON listing of participants for a campaign. Depends on campaign (requires `campaign_id`).

The `{campaign_id}` must be replaced with an actual campaign identifier when reading the resource.

Fields: `id`, `campaign_id`, `display_name`, `role`, `controller`, `created_at`, `updated_at`.

#### campaign://{campaign_id}/actors

JSON listing of actors for a campaign. Depends on campaign (requires `campaign_id`).

The `{campaign_id}` must be replaced with an actual campaign identifier when reading the resource.

Fields: `id`, `campaign_id`, `name`, `kind`, `notes`, `created_at`, `updated_at`.

Planned MCP resources that will expand what the client can ask the MCP server to
retrieve or manage:

- Campaign lookup by id.
- Session state, GM state, and actor records for active campaigns.
- Event streams for campaign timelines.
- MCP services: `duality.v1.DualityService` and `campaign.v1.CampaignService` over gRPC.
