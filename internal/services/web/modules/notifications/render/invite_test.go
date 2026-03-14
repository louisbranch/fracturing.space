package render

import "testing"

func TestRenderInAppCampaignInviteCreated(t *testing.T) {
	t.Parallel()

	out := RenderInApp(nil, Input{
		MessageType: MessageTypeCampaignInviteCreated,
		PayloadJSON: `{"campaign_name":"Skyfall","participant_name":"Scout","inviter_username":"gm"}`,
	})

	if out.Title != "Campaign invitation" {
		t.Fatalf("title = %q, want %q", out.Title, "Campaign invitation")
	}
	if out.BodyText != "You were invited to Skyfall as Scout. Invited by @gm." {
		t.Fatalf("body = %q, want invite-created copy", out.BodyText)
	}
}

func TestRenderInAppCampaignInviteAcceptedAndDeclined(t *testing.T) {
	t.Parallel()

	accepted := RenderInApp(nil, Input{
		MessageType: MessageTypeCampaignInviteAccepted,
		PayloadJSON: `{"campaign_name":"Skyfall","participant_name":"Scout","recipient_username":"ada"}`,
	})
	if accepted.Title != "Invitation accepted" || accepted.BodyText != "@ada accepted Scout in Skyfall." {
		t.Fatalf("accepted = %+v, want accepted copy", accepted)
	}

	declined := RenderInApp(nil, Input{
		MessageType: MessageTypeCampaignInviteDeclined,
		PayloadJSON: `{"campaign_name":"Skyfall","participant_name":"Scout","recipient_username":"ada"}`,
	})
	if declined.Title != "Invitation declined" || declined.BodyText != "@ada declined Scout in Skyfall." {
		t.Fatalf("declined = %+v, want declined copy", declined)
	}
}

func TestRenderInAppCampaignInvitePayloadFallback(t *testing.T) {
	t.Parallel()

	out := RenderInApp(nil, Input{
		MessageType: MessageTypeCampaignInviteDeclined,
		PayloadJSON: `{"campaign_name":`,
	})
	if out.Title != "Notification" || out.BodyText != "You have a new notification." {
		t.Fatalf("fallback = %+v, want generic copy", out)
	}
}
