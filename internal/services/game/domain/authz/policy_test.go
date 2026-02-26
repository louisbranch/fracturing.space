package authz

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

func TestPolicyTableInviteReadManagerOwnerOnly(t *testing.T) {
	table := PolicyTable()
	allowed := map[participant.CampaignAccess]bool{}
	for _, row := range table {
		if row.Action != ActionRead || row.Resource != ResourceInvite {
			continue
		}
		allowed[row.Role] = true
	}
	if !allowed[participant.CampaignAccessOwner] {
		t.Fatal("owner should be allowed invite reads")
	}
	if !allowed[participant.CampaignAccessManager] {
		t.Fatal("manager should be allowed invite reads")
	}
	if allowed[participant.CampaignAccessMember] {
		t.Fatal("member should not be allowed invite reads")
	}
}

func TestCapabilityFromActionResource(t *testing.T) {
	capability, ok := CapabilityFromActionResource(ActionManage, ResourceSession)
	if !ok {
		t.Fatal("expected manage/session capability to be recognized")
	}
	if capability != CapabilityManageSessions {
		t.Fatalf("capability = %#v, want %#v", capability, CapabilityManageSessions)
	}
	if _, ok := CapabilityFromActionResource(ActionManage, ResourceUnspecified); ok {
		t.Fatal("expected unspecified resource capability to be rejected")
	}
}

func TestCanCampaignAccess(t *testing.T) {
	tests := []struct {
		name       string
		access     participant.CampaignAccess
		capability Capability
		allowed    bool
		reasonCode string
	}{
		{
			name:       "owner can manage campaign",
			access:     participant.CampaignAccessOwner,
			capability: CapabilityManageCampaign,
			allowed:    true,
			reasonCode: ReasonAllowAccessLevel,
		},
		{
			name:       "manager cannot manage campaign",
			access:     participant.CampaignAccessManager,
			capability: CapabilityManageCampaign,
			allowed:    false,
			reasonCode: ReasonDenyAccessLevelRequired,
		},
		{
			name:       "member can mutate characters",
			access:     participant.CampaignAccessMember,
			capability: CapabilityMutateCharacters,
			allowed:    true,
			reasonCode: ReasonAllowAccessLevel,
		},
		{
			name:       "member cannot read invites",
			access:     participant.CampaignAccessMember,
			capability: CapabilityReadInvites,
			allowed:    false,
			reasonCode: ReasonDenyAccessLevelRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := CanCampaignAccess(tt.access, tt.capability)
			if decision.Allowed != tt.allowed {
				t.Fatalf("allowed = %v, want %v", decision.Allowed, tt.allowed)
			}
			if decision.ReasonCode != tt.reasonCode {
				t.Fatalf("reason = %q, want %q", decision.ReasonCode, tt.reasonCode)
			}
		})
	}
}

func TestCanCharacterMutation(t *testing.T) {
	denied := CanCharacterMutation(participant.CampaignAccessMember, "member-1", "owner-1")
	if denied.Allowed {
		t.Fatal("expected member mutation with non-owned character to be denied")
	}
	if denied.ReasonCode != ReasonDenyNotResourceOwner {
		t.Fatalf("reason = %q, want %q", denied.ReasonCode, ReasonDenyNotResourceOwner)
	}

	allowed := CanCharacterMutation(participant.CampaignAccessMember, "member-1", "member-1")
	if !allowed.Allowed {
		t.Fatal("expected member mutation with owned character to be allowed")
	}
	if allowed.ReasonCode != ReasonAllowResourceOwner {
		t.Fatalf("reason = %q, want %q", allowed.ReasonCode, ReasonAllowResourceOwner)
	}

	manager := CanCharacterMutation(participant.CampaignAccessManager, "manager-1", "owner-1")
	if !manager.Allowed {
		t.Fatal("expected manager mutation to be allowed")
	}
	if manager.ReasonCode != ReasonAllowAccessLevel {
		t.Fatalf("reason = %q, want %q", manager.ReasonCode, ReasonAllowAccessLevel)
	}
}

func TestCanParticipantAccessChange(t *testing.T) {
	tests := []struct {
		name            string
		actorAccess     participant.CampaignAccess
		targetAccess    participant.CampaignAccess
		requestedAccess participant.CampaignAccess
		ownerCount      int
		allowed         bool
		reasonCode      string
	}{
		{
			name:            "manager cannot promote to owner",
			actorAccess:     participant.CampaignAccessManager,
			targetAccess:    participant.CampaignAccessMember,
			requestedAccess: participant.CampaignAccessOwner,
			ownerCount:      1,
			allowed:         false,
			reasonCode:      ReasonDenyManagerOwnerMutationForbidden,
		},
		{
			name:            "manager cannot mutate owner target",
			actorAccess:     participant.CampaignAccessManager,
			targetAccess:    participant.CampaignAccessOwner,
			requestedAccess: participant.CampaignAccessManager,
			ownerCount:      2,
			allowed:         false,
			reasonCode:      ReasonDenyTargetIsOwner,
		},
		{
			name:            "owner cannot demote final owner",
			actorAccess:     participant.CampaignAccessOwner,
			targetAccess:    participant.CampaignAccessOwner,
			requestedAccess: participant.CampaignAccessManager,
			ownerCount:      1,
			allowed:         false,
			reasonCode:      ReasonDenyLastOwnerGuard,
		},
		{
			name:            "owner can demote owner when another owner remains",
			actorAccess:     participant.CampaignAccessOwner,
			targetAccess:    participant.CampaignAccessOwner,
			requestedAccess: participant.CampaignAccessManager,
			ownerCount:      2,
			allowed:         true,
			reasonCode:      ReasonAllowAccessLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := CanParticipantAccessChange(tt.actorAccess, tt.targetAccess, tt.requestedAccess, tt.ownerCount)
			if decision.Allowed != tt.allowed {
				t.Fatalf("allowed = %v, want %v", decision.Allowed, tt.allowed)
			}
			if decision.ReasonCode != tt.reasonCode {
				t.Fatalf("reason = %q, want %q", decision.ReasonCode, tt.reasonCode)
			}
		})
	}
}

func TestCanParticipantRemoval(t *testing.T) {
	managerTargetOwner := CanParticipantRemoval(participant.CampaignAccessManager, participant.CampaignAccessOwner, 2)
	if managerTargetOwner.Allowed {
		t.Fatal("expected manager owner-target removal to be denied")
	}
	if managerTargetOwner.ReasonCode != ReasonDenyTargetIsOwner {
		t.Fatalf("reason = %q, want %q", managerTargetOwner.ReasonCode, ReasonDenyTargetIsOwner)
	}

	ownerLastOwner := CanParticipantRemoval(participant.CampaignAccessOwner, participant.CampaignAccessOwner, 1)
	if ownerLastOwner.Allowed {
		t.Fatal("expected removal of final owner to be denied")
	}
	if ownerLastOwner.ReasonCode != ReasonDenyLastOwnerGuard {
		t.Fatalf("reason = %q, want %q", ownerLastOwner.ReasonCode, ReasonDenyLastOwnerGuard)
	}

	ownerNonOwner := CanParticipantRemoval(participant.CampaignAccessOwner, participant.CampaignAccessMember, 1)
	if !ownerNonOwner.Allowed {
		t.Fatal("expected owner removal of non-owner target to be allowed")
	}
	if ownerNonOwner.ReasonCode != ReasonAllowAccessLevel {
		t.Fatalf("reason = %q, want %q", ownerNonOwner.ReasonCode, ReasonAllowAccessLevel)
	}
}
