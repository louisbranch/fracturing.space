package campaign

import "testing"

func TestValidateCampaignOperation(t *testing.T) {
	tests := []struct {
		name    string
		status  CampaignStatus
		op      CampaignOperation
		allowed bool
	}{
		{name: "draft session start allowed", status: CampaignStatusDraft, op: CampaignOpSessionStart, allowed: true},
		{name: "draft read allowed", status: CampaignStatusDraft, op: CampaignOpRead, allowed: true},
		{name: "draft session action blocked", status: CampaignStatusDraft, op: CampaignOpSessionAction, allowed: false},
		{name: "draft mutate allowed", status: CampaignStatusDraft, op: CampaignOpCampaignMutate, allowed: true},
		{name: "draft archive blocked", status: CampaignStatusDraft, op: CampaignOpArchive, allowed: false},
		{name: "active session start allowed", status: CampaignStatusActive, op: CampaignOpSessionStart, allowed: true},
		{name: "active read allowed", status: CampaignStatusActive, op: CampaignOpRead, allowed: true},
		{name: "active session action allowed", status: CampaignStatusActive, op: CampaignOpSessionAction, allowed: true},
		{name: "active mutate allowed", status: CampaignStatusActive, op: CampaignOpCampaignMutate, allowed: true},
		{name: "completed session action blocked", status: CampaignStatusCompleted, op: CampaignOpSessionAction, allowed: false},
		{name: "completed read allowed", status: CampaignStatusCompleted, op: CampaignOpRead, allowed: true},
		{name: "completed mutate blocked", status: CampaignStatusCompleted, op: CampaignOpCampaignMutate, allowed: false},
		{name: "completed archive allowed", status: CampaignStatusCompleted, op: CampaignOpArchive, allowed: true},
		{name: "archived restore allowed", status: CampaignStatusArchived, op: CampaignOpRestore, allowed: true},
		{name: "archived read allowed", status: CampaignStatusArchived, op: CampaignOpRead, allowed: true},
		{name: "archived mutate blocked", status: CampaignStatusArchived, op: CampaignOpCampaignMutate, allowed: false},
		{name: "unspecified blocked", status: CampaignStatusUnspecified, op: CampaignOpCampaignMutate, allowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCampaignOperation(tt.status, tt.op)
			if tt.allowed && err != nil {
				t.Fatalf("expected allowed, got %v", err)
			}
			if !tt.allowed && err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
