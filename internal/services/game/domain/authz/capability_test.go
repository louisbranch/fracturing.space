package authz

import "testing"

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
