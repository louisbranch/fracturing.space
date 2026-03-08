package game

import (
	"context"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestRotateCampaignAIAuthEpochValidation(t *testing.T) {
	err := rotateCampaignAIAuthEpoch(context.Background(), Stores{}, "camp-1", aiAuthRotateReasonSessionStarted, "actor-1", command.ActorTypeParticipant)
	assertStatusCode(t, err, codes.Internal)

	stores := Stores{
		Write: domainwriteexec.WritePath{Executor: &fakeDomainEngine{result: engine.Result{Decision: command.Accept()}}},
		SystemStores: systemmanifest.ProjectionStores{
			Daggerheart: newFakeDaggerheartStore(),
		},
	}

	err = rotateCampaignAIAuthEpoch(context.Background(), stores, "", aiAuthRotateReasonSessionStarted, "actor-1", command.ActorTypeParticipant)
	assertStatusCode(t, err, codes.InvalidArgument)

	err = rotateCampaignAIAuthEpoch(context.Background(), stores, "camp-1", "", "actor-1", command.ActorTypeParticipant)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRotateCampaignAIAuthEpochSuccess(t *testing.T) {
	domain := &fakeDomainEngine{result: engine.Result{Decision: command.Accept()}}
	stores := Stores{
		Write: domainwriteexec.WritePath{Executor: domain},
		SystemStores: systemmanifest.ProjectionStores{
			Daggerheart: newFakeDaggerheartStore(),
		},
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.RequestIDHeader, "req-1",
		grpcmeta.InvocationIDHeader, "inv-1",
	))

	err := rotateCampaignAIAuthEpoch(ctx, stores, "camp-1", aiAuthRotateReasonSessionStarted, "actor-1", command.ActorTypeParticipant)
	if err != nil {
		t.Fatalf("rotate campaign ai auth epoch: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want %d", domain.calls, 1)
	}
	if got := string(domain.lastCommand.Type); got != string(commandTypeCampaignAIAuthRotate) {
		t.Fatalf("command type = %q, want %q", got, string(commandTypeCampaignAIAuthRotate))
	}
	if domain.lastCommand.ActorID != "actor-1" || domain.lastCommand.ActorType != command.ActorTypeParticipant {
		t.Fatalf("unexpected actor in command: %+v", domain.lastCommand)
	}
	payload := strings.TrimSpace(string(domain.lastCommand.PayloadJSON))
	if !strings.Contains(payload, aiAuthRotateReasonSessionStarted) {
		t.Fatalf("payload = %q, expected reason %q", payload, aiAuthRotateReasonSessionStarted)
	}
}
