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

- `duality_action_roll`: rolls Duality dice and returns the outcome with the roll context.
- `duality_outcome`: evaluates a deterministic outcome from known Hope/Fear dice without rolling.
- `duality_explain`: returns a deterministic explanation for a known Hope/Fear outcome.
- `duality_probability`: computes exact outcome counts across all duality dice combinations.
- `duality_rules_version`: returns the ruleset semantics used for Duality roll evaluation.
- `roll_dice`: rolls arbitrary dice pools and returns the individual results.
- `campaign_create`: creates a new campaign metadata record.
- `participant_create`: creates a participant (GM or player) for a campaign.

## Resources

- `campaigns://list`: JSON listing of campaign metadata records (id, name, gm_mode,
  player_count, theme_prompt, created_at, updated_at).
- `campaign://{campaign_id}/participants`: JSON listing of participants for a campaign
  (id, campaign_id, display_name, role, controller, created_at, updated_at). The
  `{campaign_id}` must be replaced with an actual campaign identifier when reading
  the resource.

Planned MCP resources that will expand what the client can ask the MCP server to
retrieve or manage:

- Campaign lookup by id.
- Session state, GM state, and actor records for active campaigns.
- Event streams for campaign timelines.
- MCP services: `duality.v1.DualityService` and `campaign.v1.CampaignService` over gRPC.
