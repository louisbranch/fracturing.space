package command

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestAcceptDecision_ReturnsEventsOnly(t *testing.T) {
	evt := event.Event{CampaignID: "camp-1"}
	decision := Accept(evt)

	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
	if decision.Events[0].CampaignID != "camp-1" {
		t.Fatalf("event campaign id = %s, want %s", decision.Events[0].CampaignID, "camp-1")
	}
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
}

func TestDecisionValidate_ReturnsErrorForEmptyDecision(t *testing.T) {
	d := Decision{}
	if err := d.Validate(); err == nil {
		t.Fatal("expected error for empty decision")
	}
}

func TestDecisionValidate_AcceptsEventsOnly(t *testing.T) {
	d := Accept(event.Event{CampaignID: "camp-1"})
	if err := d.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecisionValidate_AcceptsRejectionsOnly(t *testing.T) {
	d := Reject(Rejection{Code: "NOPE"})
	if err := d.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSharedRejectionCodes_ExistAndFollowConvention(t *testing.T) {
	codes := map[string]string{
		"RejectionCodePayloadDecodeFailed":    RejectionCodePayloadDecodeFailed,
		"RejectionCodeCommandTypeUnsupported": RejectionCodeCommandTypeUnsupported,
	}
	for name, code := range codes {
		if code == "" {
			t.Errorf("%s is empty", name)
		}
		// Convention: SCREAMING_SNAKE_CASE, no dots, no lowercase.
		for _, c := range code {
			if c >= 'a' && c <= 'z' {
				t.Errorf("%s = %q contains lowercase characters", name, code)
				break
			}
		}
	}
}

func TestRejectDecision_ReturnsRejectionsOnly(t *testing.T) {
	rejection := Rejection{Code: "INVALID"}
	decision := Reject(rejection)

	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "INVALID" {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "INVALID")
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}
