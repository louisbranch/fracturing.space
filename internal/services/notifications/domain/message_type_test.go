package domain

import "testing"

func TestNormalizeMessageType(t *testing.T) {
	t.Parallel()

	if got := NormalizeMessageType("  AUTH.ONBOARDING.WELCOME  "); got != MessageTypeOnboardingWelcome {
		t.Fatalf("NormalizeMessageType = %q, want %q", got, MessageTypeOnboardingWelcome)
	}
}

func TestResolveDeliveryPolicy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		messageType string
		wantInApp   bool
		wantEmail   bool
	}{
		{name: "onboarding canonical", messageType: MessageTypeOnboardingWelcome, wantInApp: false, wantEmail: true},
		{name: "onboarding versioned", messageType: MessageTypeOnboardingWelcomeV1, wantInApp: false, wantEmail: true},
		{name: "generic default", messageType: "campaign.invite", wantInApp: true, wantEmail: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ResolveDeliveryPolicy(tc.messageType)
			if got.InApp != tc.wantInApp {
				t.Fatalf("ResolveDeliveryPolicy(%q).InApp = %v, want %v", tc.messageType, got.InApp, tc.wantInApp)
			}
			if got.Email != tc.wantEmail {
				t.Fatalf("ResolveDeliveryPolicy(%q).Email = %v, want %v", tc.messageType, got.Email, tc.wantEmail)
			}
		})
	}
}
