package adversarytransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerRequireDependencies(t *testing.T) {
	handler := NewHandler(Dependencies{})
	if _, err := handler.GetAdversary(testContext(), &pb.DaggerheartGetAdversaryRequest{CampaignId: "camp-1", AdversaryId: "adv-1"}); status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
