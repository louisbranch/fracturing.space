Auth + Participant Vision

Vision
- Separate identity from campaign participation so the participant remains the principal for game-state access and permissions.
- Keep auth concerns (identity, external OAuth) in a dedicated service while game logic owns invitations, seats, and campaign rules.

Core Principles
- Participants are campaign-scoped seats; seats may be unclaimed.
- When claimed, a campaign has a 1:1 user to participant relationship.
- Participants can control multiple characters.
- Game service enforces permissions; auth service provides identity and join authorization.

Service Boundaries
- Auth service: user creation, join grants, future authentication/OAuth.
- Game service: campaign, participant seats, invites, seat limits, authorization checks.

Minimal Flows
- Create user (auth service).
- Create invite (game service).
- Authorize join (auth service issues join grant).
- Claim seat (game service consumes join grant and binds user to participant).

Concerns and Unknowns
- Join grant format and lifetime (signed token vs stored grant; single-use vs TTL).
- Seat reassignment rules and audit expectations.
- Fork policy for clearing or preserving user assignments on participants.
- Seat limit enforcement under concurrency.
- Auditing requirements for join, leave, and reassignment events.
- Future authentication and OAuth provider scope strategy.

Phases

Phase 0: Alignment and Minimal Spec
- Agree on entity model and service boundaries.
- Capture minimal flow diagrams and grant semantics.
- Enumerate concerns and unknowns without resolving them.

Phase 1: Minimum Join Capability (No Authentication)
- Auth service: create users; issue join grants for campaign invites.
- Game service: create invites; create or target participant seats; claim seat via join grant.
- Authorization: resolve user to participant for campaign access.
- Auditing: record join and seat reassignment events.

Later Phases (Not Yet Scheduled)
- Authentication and token validation.
- OAuth provider integration and external token storage.
- Voting policy framework and enforcement.
- Seat limit policies (min, max, active) with configurable rules.

Paths Forward (Recommendations)
- Start Phase 0 with an explicit entity diagram and sequence flows.
- In Phase 1, use a signed join grant format even without auth to keep interfaces stable.
- Defer all voting and OAuth decisions until baseline join flows are stable.

Join Grant Schema (Phase 1)
- Use a standard format: JWT (JWS, signed) or PASETO (v4 public) to avoid custom crypto.
- Recommended: JWT with EdDSA (Ed25519) and short TTL; keep it stateless.

Required Claims
- iss: auth service issuer.
- aud: game service audience.
- sub: user_id.
- exp: expiration time (short TTL).
- iat: issued-at.
- jti: unique grant id for audit.
- campaign_id: target campaign.
- invite_id: invite being claimed.

Optional Claims
- participant_id: when targeting a specific seat.
- role: intended participant role.
- seat_policy: optional policy version/tag.

Validation Rules (Game Service)
- Verify signature and aud/iss.
- Ensure grant is not expired.
- Ensure invite is valid and unclaimed.
- Enforce seat limits and unique (campaign_id, user_id).
- If participant_id is present, ensure it belongs to campaign and is unclaimed.
