package campaigntransport

import (
	"context"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestRotateCampaignAIAuthEpochValidation(t *testing.T) {
	err := rotateCampaignAIAuthEpoch(context.Background(), campaignCommandExecution{}, "camp-1", aiAuthRotateReasonCampaignAIBound, "actor-1", command.ActorTypeParticipant)
	assertStatusCode(t, err, codes.Internal)

	deps := campaignCommandExecution{
		Write: domainwrite.WritePath{Executor: &fakeDomainEngine{result: engine.Result{Decision: command.Accept()}}},
	}

	err = rotateCampaignAIAuthEpoch(context.Background(), deps, "", aiAuthRotateReasonCampaignAIBound, "actor-1", command.ActorTypeParticipant)
	assertStatusCode(t, err, codes.InvalidArgument)

	err = rotateCampaignAIAuthEpoch(context.Background(), deps, "camp-1", "", "actor-1", command.ActorTypeParticipant)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRotateCampaignAIAuthEpochSuccess(t *testing.T) {
	domain := &fakeDomainEngine{result: engine.Result{Decision: command.Accept()}}
	deps := campaignCommandExecution{
		Campaign: gametest.NewFakeCampaignStore(),
		Write:    domainwrite.WritePath{Executor: domain},
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.RequestIDHeader, "req-1",
		grpcmeta.InvocationIDHeader, "inv-1",
	))

	err := rotateCampaignAIAuthEpoch(ctx, deps, "camp-1", aiAuthRotateReasonCampaignAIBound, "actor-1", command.ActorTypeParticipant)
	if err != nil {
		t.Fatalf("rotate campaign ai auth epoch: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want %d", domain.calls, 1)
	}
	if got := string(domain.lastCommand.Type); got != string(handler.CommandTypeCampaignAIAuthRotate) {
		t.Fatalf("command type = %q, want %q", got, string(handler.CommandTypeCampaignAIAuthRotate))
	}
	if domain.lastCommand.ActorID != "actor-1" || domain.lastCommand.ActorType != command.ActorTypeParticipant {
		t.Fatalf("unexpected actor in command: %+v", domain.lastCommand)
	}
	payload := strings.TrimSpace(string(domain.lastCommand.PayloadJSON))
	if !strings.Contains(payload, aiAuthRotateReasonCampaignAIBound) {
		t.Fatalf("payload = %q, expected reason %q", payload, aiAuthRotateReasonCampaignAIBound)
	}
}
