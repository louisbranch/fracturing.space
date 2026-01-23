# MCP Tools and Resources

The MCP server communicates over stdio using JSON-RPC. Run it locally and point
your MCP client at the process stdin/stdout.

```sh
go run ./cmd/mcp
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

## Resources

- Campaign metadata fields: `name`, `gm_mode` (HUMAN, AI, HYBRID), `player_slots`, `theme_prompt`.
- Campaign storage: persisted in BoltDB at `DUALITY_DB_PATH` (default `data/duality.db`).
- MCP services: `duality.v1.DualityService` and `campaign.v1.CampaignService` over gRPC.
