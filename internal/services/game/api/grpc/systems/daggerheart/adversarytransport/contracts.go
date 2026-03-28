package adversarytransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

// CampaignStore loads campaign records for adversary transport checks.
type CampaignStore = daggerheartguard.CampaignStore

// SessionStore validates optional adversary session placement.
type SessionStore = daggerheartguard.SessionStore

// SessionGateStore guards against writes while a session gate is open.
type SessionGateStore = daggerheartguard.SessionGateStore

// DaggerheartStore loads Daggerheart adversary projections.
type DaggerheartStore interface {
	GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error)
	ListDaggerheartAdversaries(ctx context.Context, campaignID, sessionID string) ([]projectionstore.DaggerheartAdversary, error)
}

// ContentStore loads catalog-backed adversary definitions for runtime
// adversary transport.
type ContentStore interface {
	GetDaggerheartAdversaryEntry(ctx context.Context, id string) (contentstore.DaggerheartAdversaryEntry, error)
}

// DomainCommandInput captures the command metadata needed by the write callback.
type DomainCommandInput = workflowwrite.DomainCommandInput

// Dependencies defines the explicit seams used by the adversary transport.
type Dependencies struct {
	Campaign CampaignStore
	Session  SessionStore
	Gate     SessionGateStore

	Daggerheart DaggerheartStore
	Content     ContentStore

	GenerateID           func() (string, error)
	ExecuteDomainCommand func(ctx context.Context, in DomainCommandInput) error
}

// Handler owns Daggerheart adversary CRUD and read transport.
type Handler struct {
	deps Dependencies
}

// AdversaryToProto maps a Daggerheart adversary projection to protobuf form.
func AdversaryToProto(adversary projectionstore.DaggerheartAdversary) *pb.DaggerheartAdversary {
	return adversaryToProto(adversary)
}

// LoadAdversaryForSession loads an adversary and enforces session ownership
// when the adversary is session-bound.
func LoadAdversaryForSession(ctx context.Context, store DaggerheartStore, campaignID, sessionID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	return loadAdversaryForSession(ctx, store, campaignID, sessionID, adversaryID)
}
