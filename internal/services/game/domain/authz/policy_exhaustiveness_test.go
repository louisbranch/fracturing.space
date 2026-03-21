package authz

import (
	"fmt"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

// TestPolicyMatrixExhaustiveness verifies that every (Action, Resource) pair in
// the matrix has an explicit decision for every recognized role. When a new
// role is added to allRoles, this test fails until the matrix is updated.
func TestPolicyMatrixExhaustiveness(t *testing.T) {
	for _, entry := range policyMatrix {
		allowed := make(map[participant.CampaignAccess]bool, len(entry.AllowedRoles))
		for _, role := range entry.AllowedRoles {
			allowed[role] = true
		}
		for _, role := range allRoles {
			capability := Capability{Action: entry.Action, Resource: entry.Resource}
			decision := CanCampaignAccess(role, capability)
			if allowed[role] && !decision.Allowed {
				t.Errorf("role %q should be allowed for %s but was denied", role, capability.Label())
			}
			if !allowed[role] && decision.Allowed {
				t.Errorf("role %q should be denied for %s but was allowed", role, capability.Label())
			}
		}
	}
}

// TestPolicyMatrixNoDuplicateEntries verifies that no two entries share the
// same (Action, Resource) pair.
func TestPolicyMatrixNoDuplicateEntries(t *testing.T) {
	seen := make(map[string]bool, len(policyMatrix))
	for _, entry := range policyMatrix {
		key := string(entry.Action) + ":" + string(entry.Resource)
		if seen[key] {
			t.Fatalf("duplicate policy matrix entry for (%s, %s)", entry.Action, entry.Resource)
		}
		seen[key] = true
	}
}

// TestPolicyMatrixOnlyRecognizedRoles verifies that every role listed in the
// matrix is a member of allRoles.
func TestPolicyMatrixOnlyRecognizedRoles(t *testing.T) {
	recognized := make(map[participant.CampaignAccess]bool, len(allRoles))
	for _, r := range allRoles {
		recognized[r] = true
	}
	for _, entry := range policyMatrix {
		for _, role := range entry.AllowedRoles {
			if !recognized[role] {
				t.Errorf("unrecognized role %q in matrix entry (%s, %s)",
					role, entry.Action, entry.Resource)
			}
		}
	}
}

// TestPolicyCapabilityAccessorParity verifies that every Capability*() accessor
// maps to an entry in the policy matrix. When a new accessor is added, this
// test fails until a matrix entry exists.
func TestPolicyCapabilityAccessorParity(t *testing.T) {
	accessors := []struct {
		name string
		fn   func() Capability
	}{
		{"ReadCampaign", CapabilityReadCampaign},
		{"ReadInvites", CapabilityReadInvites},
		{"ManageCampaign", CapabilityManageCampaign},
		{"ManageParticipants", CapabilityManageParticipants},
		{"ManageInvites", CapabilityManageInvites},
		{"ManageSessions", CapabilityManageSessions},
		{"MutateCharacters", CapabilityMutateCharacters},
		{"ManageCharacters", CapabilityManageCharacters},
		{"TransferCharacterOwnership", CapabilityTransferCharacterOwnership},
	}

	for _, tc := range accessors {
		t.Run(tc.name, func(t *testing.T) {
			cap := tc.fn()
			key := string(cap.Action) + ":" + string(cap.Resource)
			if _, ok := matrixIndex[key]; !ok {
				t.Fatalf("Capability%s() (%s) has no policy matrix entry", tc.name, cap.Label())
			}
		})
	}
}

// TestPolicyMatrixCoversAllActions verifies that every recognized action
// appears in at least one matrix entry.
func TestPolicyMatrixCoversAllActions(t *testing.T) {
	used := make(map[Action]bool)
	for _, entry := range policyMatrix {
		used[entry.Action] = true
	}
	for _, action := range allActions {
		if !used[action] {
			t.Errorf("action %q has no policy matrix entries", action)
		}
	}
}

// TestPolicyMatrixCoversAllResources verifies that every recognized resource
// appears in at least one matrix entry.
func TestPolicyMatrixCoversAllResources(t *testing.T) {
	used := make(map[Resource]bool)
	for _, entry := range policyMatrix {
		used[entry.Resource] = true
	}
	for _, resource := range allResources {
		if !used[resource] {
			t.Errorf("resource %q has no policy matrix entries", resource)
		}
	}
}

// TestPolicyMatrixCollisionDetection validates that no two entries produce
// conflicting decisions. This is guaranteed by construction (each key appears
// once), but serves as a safety net if the index building logic changes.
func TestPolicyMatrixCollisionDetection(t *testing.T) {
	counts := make(map[string]int, len(policyMatrix))
	for _, entry := range policyMatrix {
		key := fmt.Sprintf("%s:%s", entry.Action, entry.Resource)
		counts[key]++
		if counts[key] > 1 {
			t.Errorf("collision: %s has %d entries", key, counts[key])
		}
	}
}
