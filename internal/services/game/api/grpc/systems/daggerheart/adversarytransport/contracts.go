package adversarytransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignStore loads campaign records for adversary transport checks.
type CampaignStore interface {
	Get(ctx context.Context, campaignID string) (storage.CampaignRecord, error)
}

// SessionStore validates optional adversary session placement.
type SessionStore interface {
	GetSession(ctx context.Context, campaignID, sessionID string) (storage.SessionRecord, error)
}

// SessionGateStore guards against writes while a session gate is open.
type SessionGateStore interface {
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error)
}

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
type DomainCommandInput struct {
	CampaignID      string
	CommandType     command.Type
	SessionID       string
	SceneID         string
	RequestID       string
	InvocationID    string
	EntityType      string
	EntityID        string
	PayloadJSON     []byte
	MissingEventMsg string
	ApplyErrMessage string
}

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
