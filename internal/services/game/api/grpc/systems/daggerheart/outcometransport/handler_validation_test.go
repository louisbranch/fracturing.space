package outcometransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"google.golang.org/grpc/codes"

	validate "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
)

func TestValidateCampaignIDFromContext(t *testing.T) {
	ctx := testSessionContext("", "sess-1")
	_, err := validate.RequiredID(grpcmeta.CampaignIDFromContext(ctx), "campaign id")
	assertStatusCode(t, err, codes.InvalidArgument)

	ctx = testSessionContext("camp-1", "sess-1")
	campaignID, err := validate.RequiredID(grpcmeta.CampaignIDFromContext(ctx), "campaign id")
	if err != nil {
		t.Fatalf("validate.RequiredID returned error: %v", err)
	}
	if campaignID != "camp-1" {
		t.Fatalf("campaignID = %q, want %q", campaignID, "camp-1")
	}
}

func TestHandlerValidateSessionOutcome(t *testing.T) {
	handler, events, _ := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{
		requestID: "roll-1",
		outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		metadata: workflowtransport.RollSystemMetadata{
			CharacterID: "char-1",
			RollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
			HopeFear:    workflowtransport.BoolPtr(true),
		},
	})

	pre, err := handler.validateSessionOutcome(testSessionContext("camp-1", "sess-1"), "", roll.Seq)
	if err != nil {
		t.Fatalf("validateSessionOutcome returned error: %v", err)
	}
	if pre.campaignID != "camp-1" || pre.sessionID != "sess-1" {
		t.Fatalf("prelude ids = %+v", pre)
	}
	if pre.rollRequestID != "roll-1" {
		t.Fatalf("rollRequestID = %q, want %q", pre.rollRequestID, "roll-1")
	}
	if pre.rollMetadata.CharacterID != "char-1" {
		t.Fatalf("rollMetadata.CharacterID = %q", pre.rollMetadata.CharacterID)
	}
}
