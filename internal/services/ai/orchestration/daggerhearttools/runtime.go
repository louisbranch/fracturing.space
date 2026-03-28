package daggerhearttools

import (
	"context"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

// Runtime is the narrow fixed-authority seam the extracted Daggerheart tools
// need from the generic session shell.
type Runtime interface {
	CharacterClient() statev1.CharacterServiceClient
	SessionClient() statev1.SessionServiceClient
	SnapshotClient() statev1.SnapshotServiceClient
	DaggerheartClient() pb.DaggerheartServiceClient
	CallContext(ctx context.Context) (context.Context, context.CancelFunc)
	ResolveCampaignID(explicit string) string
	ResolveSessionID(explicit string) string
	ResolveSceneID(ctx context.Context, campaignID, explicit string) (string, error)
}

// ToolExecutor runs one Daggerheart-specific tool against the runtime seam.
type ToolExecutor func(Runtime, context.Context, []byte) (orchestration.ToolResult, error)
