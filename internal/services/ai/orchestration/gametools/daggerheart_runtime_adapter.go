package gametools

import (
	"context"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/daggerhearttools"
)

// daggerheartRuntimeAdapter keeps DirectSession as the generic session shell
// while delegating Daggerheart-only execution to the extracted package.
type daggerheartRuntimeAdapter struct {
	session *DirectSession
}

func (a daggerheartRuntimeAdapter) CharacterClient() statev1.CharacterServiceClient {
	return a.session.clients.Character
}

func (a daggerheartRuntimeAdapter) SessionClient() statev1.SessionServiceClient {
	return a.session.clients.Session
}

func (a daggerheartRuntimeAdapter) SnapshotClient() statev1.SnapshotServiceClient {
	return a.session.clients.Snapshot
}

func (a daggerheartRuntimeAdapter) DaggerheartClient() pb.DaggerheartServiceClient {
	return a.session.clients.Daggerheart
}

func (a daggerheartRuntimeAdapter) CallContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return outgoingContext(ctx, a.session.sc)
}

func (a daggerheartRuntimeAdapter) ResolveCampaignID(explicit string) string {
	return a.session.resolveCampaignID(explicit)
}

func (a daggerheartRuntimeAdapter) ResolveSessionID(explicit string) string {
	return a.session.resolveSessionID(explicit)
}

func (a daggerheartRuntimeAdapter) ResolveSceneID(ctx context.Context, campaignID, explicit string) (string, error) {
	return a.session.resolveSceneID(ctx, campaignID, explicit)
}

func wrapDaggerheartExecutor(executor daggerhearttools.ToolExecutor) toolExecutor {
	return func(session *DirectSession, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
		return executor(daggerheartRuntimeAdapter{session: session}, ctx, argsJSON)
	}
}
