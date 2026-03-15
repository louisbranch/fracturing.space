package notificationpayload

import "testing"

func TestParseInAppPayloadDropsUnsupportedActions(t *testing.T) {
	t.Parallel()

	payload, ok := ParseInAppPayload(`{
		"title":"Campaign invitation",
		"actions":[
			{"label":"View invitation","kind":"public_invite_view","target_id":"inv-1","method":"get","style":"primary"},
			{"label":"Bad action","kind":"external","target_id":"https://example.com"}
		]
	}`)
	if !ok {
		t.Fatal("ParseInAppPayload() = false, want true")
	}
	if len(payload.Actions) != 1 {
		t.Fatalf("actions len = %d, want 1", len(payload.Actions))
	}
	if payload.Actions[0].Kind != ActionKindPublicInviteView {
		t.Fatalf("action kind = %q, want %q", payload.Actions[0].Kind, ActionKindPublicInviteView)
	}
}

func TestActionConstructorsReturnCanonicalActions(t *testing.T) {
	t.Parallel()

	viewInvite := ViewInvitationAction(" inv-1 ")
	if viewInvite.Label != "View invitation" || viewInvite.Kind != ActionKindPublicInviteView || viewInvite.TargetID != "inv-1" || viewInvite.Method != ActionMethodGet || viewInvite.Style != ActionStylePrimary {
		t.Fatalf("view invite action = %+v, want canonical invite action", viewInvite)
	}

	openCampaign := OpenCampaignAction(" camp-1 ")
	if openCampaign.Label != "Open campaign" || openCampaign.Kind != ActionKindAppCampaignOpen || openCampaign.TargetID != "camp-1" || openCampaign.Method != ActionMethodGet || openCampaign.Style != ActionStylePrimary {
		t.Fatalf("open campaign action = %+v, want canonical campaign action", openCampaign)
	}
}
