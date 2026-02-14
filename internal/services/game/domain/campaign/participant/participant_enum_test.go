package participant

import (
	"testing"
)

func TestParticipantRoleFromLabel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ParticipantRole
		wantErr bool
	}{
		{name: "gm", input: "GM", want: ParticipantRoleGM},
		{name: "player", input: "PLAYER", want: ParticipantRolePlayer},
		{name: "lowercase gm", input: "gm", want: ParticipantRoleGM},
		{name: "lowercase player", input: "player", want: ParticipantRolePlayer},
		{name: "whitespace trimmed", input: "  GM  ", want: ParticipantRoleGM},
		{name: "mixed case", input: "Player", want: ParticipantRolePlayer},
		{name: "empty string", input: "", wantErr: true},
		{name: "unknown value", input: "INVALID", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParticipantRoleFromLabel(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestControllerFromLabel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Controller
		wantErr bool
	}{
		{name: "short human", input: "HUMAN", want: ControllerHuman},
		{name: "prefixed human", input: "CONTROLLER_HUMAN", want: ControllerHuman},
		{name: "short ai", input: "AI", want: ControllerAI},
		{name: "prefixed ai", input: "CONTROLLER_AI", want: ControllerAI},
		{name: "lowercase", input: "human", want: ControllerHuman},
		{name: "whitespace trimmed", input: "  AI  ", want: ControllerAI},
		{name: "empty string", input: "", wantErr: true},
		{name: "unknown value", input: "INVALID", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ControllerFromLabel(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCampaignAccessFromLabel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CampaignAccess
		wantErr bool
	}{
		{name: "short member", input: "MEMBER", want: CampaignAccessMember},
		{name: "prefixed member", input: "CAMPAIGN_ACCESS_MEMBER", want: CampaignAccessMember},
		{name: "short manager", input: "MANAGER", want: CampaignAccessManager},
		{name: "prefixed manager", input: "CAMPAIGN_ACCESS_MANAGER", want: CampaignAccessManager},
		{name: "short owner", input: "OWNER", want: CampaignAccessOwner},
		{name: "prefixed owner", input: "CAMPAIGN_ACCESS_OWNER", want: CampaignAccessOwner},
		{name: "lowercase", input: "member", want: CampaignAccessMember},
		{name: "whitespace trimmed", input: "  OWNER  ", want: CampaignAccessOwner},
		{name: "mixed case", input: "Manager", want: CampaignAccessManager},
		{name: "empty string", input: "", wantErr: true},
		{name: "unknown value", input: "INVALID", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CampaignAccessFromLabel(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}
