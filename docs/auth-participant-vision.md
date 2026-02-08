Auth + Participant Vision

Vision
- Separate identity from campaign participation so the participant remains the principal for game-state access and permissions.
- Keep auth concerns (identity, external OAuth) in a dedicated service while game logic owns invitations, seats, and campaign rules.

Core Principles
- Participants are campaign-scoped seats; seats may be unclaimed.
- When claimed, a campaign has a 1:1 user to participant relationship.
- Participants can control multiple characters.
- Game service enforces permissions; auth service provides identity and join authorization.
- Authorization decisions flow through a policy layer (v0 maps is_owner to allowed actions).

Service Boundaries
- Auth service: user creation, join grants, future authentication/OAuth.
- Game service: campaign, participant seats, invites, seat limits, authorization checks.

Minimal Flows
- Create user (auth service).
- Create invite (game service, seat-targeted).
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
- Game service: create seat-targeted invites; claim seat via join grant.
- Authorization: resolve user to participant for campaign access.
- Auditing: record join and seat reassignment events.
- Campaign creation: creator becomes a participant with capability to manage participants and invites.

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

Phase 1 Constraints (Implementation)
- Invites are seat-targeted only (participant_id is required).
- Invites are single-use and do not expire.
- Owners are expressed as a participant flag (is_owner) and evaluated via policy.Can(...).
- Owners can remove participants even if already claimed.

Claim Changes Needed (from current implementation)
- Add user_id on participant records and enforce unique (campaign_id, user_id) when claimed.
- Add a claim path in game service (InviteService.ClaimInvite or ParticipantService.ClaimSeat) that:
  - Validates a join grant from auth service (aud/iss/exp/jti/campaign_id/invite_id/user_id).
  - Verifies the invite is pending and targets the participant seat.
  - Binds user_id to the participant and marks invite as CLAIMED in one transaction.
- Auth service issues join grants for seat-targeted invites (signed token, short TTL).
- MCP/web tooling needs a claim operation that accepts the join grant and target invite.

Future Cases (Documented, Not Implemented)
- Open invites (anyone with link) and reusable tokens.
- Invites that create a new seat on claim.
- Seat limit checks at invite creation time.
- Ownership transfer and multi-owner capability rules.
