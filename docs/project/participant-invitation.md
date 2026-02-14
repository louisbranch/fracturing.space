Participant Invitation Flow

Purpose
This document describes the current invitation flow and data records for campaigns, and notes follow-ups that are not yet implemented.

Domain Model (Current)
- Participants are campaign-scoped seats; seats may be unclaimed.
- Invites are seat-targeted and reference a participant seat.
- Invites may optionally target a specific user via recipient_user_id.
- Join grants are signed tokens issued by the auth service to authorize a specific user to claim a specific invite.
- Game service owns invites, seats, and enforcement; auth service owns identity and join grant issuance.

Current Flow
1) Create invite (game service)
   - API: game.v1.InviteService.CreateInvite
   - Inputs: campaign_id, participant_id
   - Checks: campaign mutable, caller can manage invites, participant seat exists.
   - Records: InviteCreated event; invites projection stores status PENDING.

2) Issue join grant (auth service)
   - API: auth.v1.AuthService.IssueJoinGrant
   - Inputs: user_id, campaign_id, invite_id, participant_id
   - Output: signed join_grant token, jti, expires_at.
   - Records: no persisted grant record; the token is the authority.

3) Claim invite (game service)
   - API: game.v1.InviteService.ClaimInvite
   - Inputs: campaign_id, invite_id, join_grant
   - Checks: join grant signature, iss/aud, exp/nbf, and claim match (campaign_id, invite_id, user_id).
   - Checks: invite is pending, participant seat unclaimed, user not already claimed for the campaign, jti unused.
   - Records: ParticipantBound event, InviteClaimed event (includes jti); projections bind participant.user_id and mark invite CLAIMED; claim index enforces unique (campaign_id, user_id).

4) Revoke invite (game service)
   - API: game.v1.InviteService.RevokeInvite
   - Records: InviteRevoked event; invites projection stores status REVOKED.

Enforcement Notes
- Invites may optionally store a recipient_user_id.
- Recipient enforcement happens at claim time when recipient_user_id is set.
- Anyone holding a valid join grant can claim the invite first; jti prevents reuse.

Data Records (Current)
- invites table: id, campaign_id, participant_id, status, created_by_participant_id, timestamps.
- participants table: campaign_id, id, user_id, display_name, role, controller, access fields.
- event journal: InviteCreated, InviteClaimed, InviteRevoked, ParticipantBound.
- claim index: unique (campaign_id, user_id) and jti de-duplication.

Follow-Ups and Open Questions
- Add claim-aware invite fields (claimed_by_user_id, claimed_at) or keep in events only.
- Persist join-grant jti in a projection index to avoid event-log scans during claims.
- Clarify seat reassignment rules and audit expectations (unbound or reassigned events).
- Define fork policy for clearing or preserving participant user assignments.
- Define seat limit enforcement under concurrency (creation vs claim-time checks).
- Document auditing requirements for join, leave, and reassignment events.
- Validate projection apply behavior when multiple events are emitted in one command.
- Decide on join grant format/lifetime and key rotation policy (issuer/audience, TTL, overlap).
- Define behavior for open invites (link-based) and invites that create new seats.
- Define ownership transfer and multi-owner capability rules.
- Determine whether to add a stored join grant or recipient binding for invite listings.

Related Docs
- docs/project/oauth.md

Join Grant Configuration (Current)
- Env: FRACTURING_SPACE_JOIN_GRANT_ISSUER
- Env: FRACTURING_SPACE_JOIN_GRANT_AUDIENCE
- Env: FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY
