package participant

import "testing"

func TestNormalizeRole(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   Role
		wantOK bool
	}{
		{name: "gm short", input: "GM", want: RoleGM, wantOK: true},
		{name: "gm enum", input: "participant_role_gm", want: RoleGM, wantOK: true},
		{name: "player short", input: "player", want: RolePlayer, wantOK: true},
		{name: "player enum", input: "ROLE_PLAYER", want: RolePlayer, wantOK: true},
		{name: "blank", input: " ", want: RoleUnspecified, wantOK: false},
		{name: "invalid", input: "moderator", want: RoleUnspecified, wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NormalizeRole(tc.input)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("NormalizeRole(%q) = (%q, %v), want (%q, %v)", tc.input, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestNormalizeController(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   Controller
		wantOK bool
	}{
		{name: "human", input: " CONTROLLER_HUMAN ", want: ControllerHuman, wantOK: true},
		{name: "ai", input: "ai", want: ControllerAI, wantOK: true},
		{name: "blank", input: " ", want: ControllerUnspecified, wantOK: false},
		{name: "invalid", input: "bot", want: ControllerUnspecified, wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NormalizeController(tc.input)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("NormalizeController(%q) = (%q, %v), want (%q, %v)", tc.input, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestNormalizeCampaignAccess(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   CampaignAccess
		wantOK bool
	}{
		{name: "member", input: "member", want: CampaignAccessMember, wantOK: true},
		{name: "manager enum", input: "CAMPAIGN_ACCESS_MANAGER", want: CampaignAccessManager, wantOK: true},
		{name: "owner", input: " OWNER ", want: CampaignAccessOwner, wantOK: true},
		{name: "blank", input: " ", want: CampaignAccessUnspecified, wantOK: false},
		{name: "invalid", input: "guest", want: CampaignAccessUnspecified, wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NormalizeCampaignAccess(tc.input)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("NormalizeCampaignAccess(%q) = (%q, %v), want (%q, %v)", tc.input, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
