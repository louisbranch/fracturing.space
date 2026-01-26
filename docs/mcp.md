# MCP Tools and Resources

The MCP server supports two transport modes: stdio (default) and HTTP.

## Transport Modes

### Stdio Transport (Default)

The MCP server communicates over stdio using JSON-RPC. Run it locally and point
your MCP client at the process stdin/stdout.

```sh
go run ./cmd/mcp
```

Alternatively, use the convenience script which resolves to the repo root automatically:

```sh
./scripts/mcp.sh
```

### HTTP Transport

The MCP server can also be exposed over HTTP for local use. This enables remote
clients to connect via HTTP requests.

```sh
go run ./cmd/mcp -transport=http -http-addr=localhost:8081 -addr=localhost:8080
```

**Note**: HTTP transport is intended for local use only. Security features
(authentication, TLS, rate limiting) are planned for future releases.

#### HTTP Endpoints

- `POST /mcp` - Send JSON-RPC requests
  - Content-Type: `application/json`
  - Request body: JSON-RPC message
  - Response: JSON-RPC response
  - Session management: Uses `mcp_session` cookie (set automatically on first request)

- `GET /mcp` - Server-Sent Events stream for streaming responses
  - Session management: Uses `mcp_session` cookie (set automatically on first request)
  - Response: `text/event-stream` with JSON-RPC notifications

- `GET /mcp/health` - Health check endpoint
  - Returns: `200 OK` when server is ready

#### Example HTTP Usage

```bash
# MCP uses cookies for session management (per spec)
# curl automatically handles cookies with -c and -b flags

# 1) First request: initialize session (cookie is set automatically)
curl -sS -c /tmp/mcp-cookies.txt -X POST http://localhost:8081/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "clientInfo": {
        "name": "test-client",
        "version": "0.1.0"
      }
    }
  }'

# 2) Send initialized notification (cookie is sent automatically)
curl -sS -b /tmp/mcp-cookies.txt -X POST http://localhost:8081/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "initialized",
    "params": {}
  }'

# 3) Subsequent request: cookie is sent automatically to reuse session
curl -sS -b /tmp/mcp-cookies.txt -X POST http://localhost:8081/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list",
    "params": {}
  }'

# 4) Check health
curl http://localhost:8081/mcp/health
```

The gRPC server must be running at `localhost:8080` (or the configured address).

For an OpenCode client configuration, see `opencode.jsonc`.

## Tools

### Context Management Tools

#### set_context

Sets the current context (campaign_id, optional session_id, optional participant_id) for subsequent tool calls. The context is stored in-memory and does not persist across process restarts. If an optional field is omitted, it is cleared from the context.

**Input:**

```json
{
  "campaign_id": "camp_abc123",
  "session_id": "sess_ghi789",
  "participant_id": "part_xyz789"
}
```

All fields except `campaign_id` are optional. To clear optional fields, omit them from the request.

**Output:**

```json
{
  "context": {
    "campaign_id": "camp_abc123",
    "session_id": "sess_ghi789",
    "participant_id": "part_xyz789"
  }
}
```

**Validation:**

- `campaign_id` must exist
- If `session_id` is provided: the session must exist and belong to `campaign_id`
- If `participant_id` is provided: the participant must exist and belong to `campaign_id`

**Errors:**

- `NotFound`: campaign, session, or participant does not exist
- `InvalidArgument`: empty strings provided, or mismatched ownership (e.g., session not in campaign)

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
  "participant_count": 0,
  "actor_count": 0,
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

### Session Service Tools

#### session_start

Starts a new session for a campaign. Enforces at most one ACTIVE session per campaign.

**Input:**

```json
{
  "campaign_id": "camp_abc123",
  "name": "Session 1: The Journey Begins"
}
```

**Output:**

```json
{
  "id": "sess_ghi789",
  "campaign_id": "camp_abc123",
  "name": "Session 1: The Journey Begins",
  "status": "ACTIVE",
  "started_at": "2025-01-15T11:00:00Z",
  "updated_at": "2025-01-15T11:00:00Z"
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
  "dice_model": "DUALITY_D12_V1",
  "total_formula": "hope + fear + modifier",
  "crit_rule": "HOPE_EQUALS_FEAR_IS_CRITICAL",
  "difficulty_rule": "TOTAL_MEETS_OR_EXCEEDS_DIFFICULTY",
  "outcomes": ["ROLL_WITH_HOPE", "ROLL_WITH_FEAR", "SUCCESS_WITH_HOPE", "SUCCESS_WITH_FEAR", "FAILURE_WITH_HOPE", "FAILURE_WITH_FEAR", "CRITICAL_SUCCESS"]
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
  "outcome": "SUCCESS_WITH_HOPE"
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
  "outcome": "SUCCESS_WITH_HOPE"
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
  "outcome": "SUCCESS_WITH_HOPE",
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
      "code": "SUM_DICE",
      "message": "Sum hope and fear dice",
      "data": { "hope": 8, "fear": 5, "base_total": 13 }
    },
    {
      "code": "APPLY_MODIFIER",
      "message": "Apply modifier to base total",
      "data": { "base_total": 13, "modifier": 2, "total": 15 }
    },
    {
      "code": "CHECK_CRIT",
      "message": "Check for critical outcome",
      "data": { "hope": 8, "fear": 5, "is_crit": false }
    },
    {
      "code": "CHECK_DIFFICULTY",
      "message": "Compare total against difficulty",
      "data": { "total": 15, "difficulty": 15, "meets_difficulty": true }
    },
    {
      "code": "SELECT_OUTCOME",
      "message": "Select final outcome based on roll",
      "data": { "outcome": "SUCCESS" }
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
    {"outcome": "CRITICAL_SUCCESS", "count": 12},
    {"outcome": "SUCCESS_WITH_HOPE", "count": 45},
    {"outcome": "SUCCESS_WITH_FEAR", "count": 40},
    {"outcome": "FAILURE_WITH_HOPE", "count": 25},
    {"outcome": "FAILURE_WITH_FEAR", "count": 22}
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

Returns a JSON object with a `campaigns` array of campaign metadata records. No dependencies.

**Response:**

```json
{
  "campaigns": [
    {
      "id": "camp_abc123",
      "name": "The Lost Expedition",
      "gm_mode": "HUMAN",
      "participant_count": 3,
      "actor_count": 2,
      "theme_prompt": "A dark fantasy campaign set in a cursed forest",
      "created_at": "2025-01-15T10:00:00Z",
      "updated_at": "2025-01-15T10:00:00Z"
    }
  ]
}
```

#### campaign://{campaign_id}

Returns a JSON object with a single `campaign` metadata record. Provides direct access to a campaign by ID without requiring a scan of campaigns://list. Requires `campaign_id` in the URI.

The `{campaign_id}` must be replaced with an actual campaign identifier when reading the resource. The URI must not contain additional path segments, query parameters, or fragments (e.g., `campaign://id/participants` should use the participant list resource instead).

**Response:**

```json
{
  "campaign": {
    "id": "camp_abc123",
    "name": "The Lost Expedition",
    "gm_mode": "HUMAN",
    "participant_count": 3,
    "actor_count": 2,
    "theme_prompt": "A dark fantasy campaign set in a cursed forest",
    "created_at": "2025-01-15T10:00:00Z",
    "updated_at": "2025-01-15T10:00:00Z"
  }
}
```

**Errors:**

- `NotFound`: campaign_id does not exist
- `InvalidArgument`: malformed campaign_id

#### campaign://{campaign_id}/participants

JSON listing of participants for a campaign. Depends on campaign (requires `campaign_id`).

The `{campaign_id}` must be replaced with an actual campaign identifier when reading the resource.

**Response:**

```json
{
  "participants": [
    {
      "id": "part_xyz789",
      "campaign_id": "camp_abc123",
      "display_name": "Alice",
      "role": "PLAYER",
      "controller": "HUMAN",
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:30:00Z"
    }
  ]
}
```

#### campaign://{campaign_id}/actors

JSON listing of actors for a campaign. Depends on campaign (requires `campaign_id`).

The `{campaign_id}` must be replaced with an actual campaign identifier when reading the resource.

**Response:**

```json
{
  "actors": [
    {
      "id": "actor_def456",
      "campaign_id": "camp_abc123",
      "name": "Thorin Ironforge",
      "kind": "PC",
      "notes": "Dwarf warrior with a mysterious past",
      "created_at": "2025-01-15T10:35:00Z",
      "updated_at": "2025-01-15T10:35:00Z"
    }
  ]
}
```

#### campaign://{campaign_id}/sessions

JSON listing of sessions for a campaign. Depends on campaign (requires `campaign_id`).

The `{campaign_id}` must be replaced with an actual campaign identifier when reading the resource.

**Response:**

```json
{
  "sessions": [
    {
      "id": "sess_ghi789",
      "campaign_id": "camp_abc123",
      "name": "Session 1: The Journey Begins",
      "status": "ACTIVE",
      "started_at": "2025-01-15T11:00:00Z",
      "updated_at": "2025-01-15T11:00:00Z"
    }
  ]
}
```

Note: The `ended_at` field is optional and only present for sessions that have ended.
