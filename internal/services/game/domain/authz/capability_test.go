package authz

import (
	"testing"
)

func TestCapabilityAccessors_ReturnValidCapabilities(t *testing.T) {
	accessors := []struct {
		name string
		fn   func() Capability
		want string
	}{
		{"ReadCampaign", CapabilityReadCampaign, "read_campaign"},
		{"ReadInvites", CapabilityReadInvites, "read_invite"},
		{"ManageCampaign", CapabilityManageCampaign, "manage_campaign"},
		{"ManageParticipants", CapabilityManageParticipants, "manage_participant"},
		{"ManageInvites", CapabilityManageInvites, "manage_invite"},
		{"ManageSessions", CapabilityManageSessions, "manage_session"},
		{"MutateCharacters", CapabilityMutateCharacters, "mutate_character"},
		{"ManageCharacters", CapabilityManageCharacters, "manage_character"},
		{"TransferCharacterOwnership", CapabilityTransferCharacterOwnership, "transfer_ownership_character"},
	}
	for _, tc := range accessors {
		t.Run(tc.name, func(t *testing.T) {
			cap := tc.fn()
			if !cap.Valid() {
				t.Fatalf("%s returned invalid capability", tc.name)
			}
			if got := cap.Label(); got != tc.want {
				t.Fatalf("%s Label() = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestCapabilityLabel(t *testing.T) {
	tests := []struct {
		name string
		in   Capability
		want string
	}{
		{
			name: "valid capability",
			in: Capability{
				Action:   ActionManage,
				Resource: ResourceSession,
			},
			want: "manage_session",
		},
		{
			name: "trimmed label",
			in: Capability{
				Action:   Action(" mutate "),
				Resource: Resource(" character "),
			},
			want: "mutate_character",
		},
		{
			name: "missing action",
			in: Capability{
				Action:   ActionUnspecified,
				Resource: ResourceCharacter,
			},
			want: "unknown",
		},
		{
			name: "missing resource",
			in: Capability{
				Action:   ActionRead,
				Resource: ResourceUnspecified,
			},
			want: "unknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.Label(); got != tc.want {
				t.Fatalf("Label() = %q, want %q", got, tc.want)
			}
		})
	}
}
