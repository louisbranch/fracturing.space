package adversarytransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

// CreateAdversary validates campaign and optional session placement, emits the
// create command, and reloads the resulting projection.
func (h *Handler) CreateAdversary(ctx context.Context, in *pb.DaggerheartCreateAdversaryRequest) (*pb.DaggerheartCreateAdversaryResponse, error) {
	if in == nil {
		return nil, invalidArgument("create adversary request is required")
	}
	if err := h.requireBaseDependencies(); err != nil {
		return nil, err
	}
	if h.deps.ExecuteDomainCommand == nil {
		return nil, internal("domain command executor is not configured")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	name, err := validate.RequiredID(in.GetName(), "name")
	if err != nil {
		return nil, err
	}
	kind := strings.TrimSpace(in.GetKind())
	notes := strings.TrimSpace(in.GetNotes())
	sessionID := ""
	if in.SessionId != nil {
		sessionID = strings.TrimSpace(in.SessionId.GetValue())
	}
	sceneID := strings.TrimSpace(in.GetSceneId())

	stats, err := normalizeAdversaryStats(adversaryStatsInput{
		HP:            in.Hp,
		HPMax:         in.HpMax,
		Stress:        in.Stress,
		StressMax:     in.StressMax,
		Evasion:       in.Evasion,
		Major:         in.MajorThreshold,
		Severe:        in.SevereThreshold,
		Armor:         in.Armor,
		RequireFields: false,
	})
	if err != nil {
		return nil, invalidArgument(err.Error())
	}

	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart adversaries"); err != nil {
		return nil, err
	}

	if sessionID != "" {
		if h.deps.Session == nil {
			return nil, internal("session store is not configured")
		}
		if _, err := h.deps.Session.GetSession(ctx, campaignID, sessionID); err != nil {
			return nil, grpcerror.HandleDomainError(err)
		}
		if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.Gate, campaignID, sessionID); err != nil {
			return nil, err
		}
	}

	adversaryID, err := h.deps.GenerateID()
	if err != nil {
		return nil, grpcerror.Internal("generate adversary id", err)
	}
	payloadJSON, err := json.Marshal(daggerheart.AdversaryCreatePayload{
		AdversaryID: ids.AdversaryID(adversaryID),
		Name:        name,
		Kind:        kind,
		SessionID:   ids.SessionID(sessionID),
		Notes:       notes,
		HP:          stats.HP,
		HPMax:       stats.HPMax,
		Stress:      stats.Stress,
		StressMax:   stats.StressMax,
		Evasion:     stats.Evasion,
		Major:       stats.Major,
		Severe:      stats.Severe,
		Armor:       stats.Armor,
	})
	if err != nil {
		return nil, grpcerror.Internal("encode adversary payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartAdversaryCreate,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "adversary",
		EntityID:        adversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary create did not emit an event",
		ApplyErrMessage: "apply adversary created event",
	}); err != nil {
		return nil, err
	}
	created, err := h.deps.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, grpcerror.Internal("load adversary", err)
	}
	return &pb.DaggerheartCreateAdversaryResponse{Adversary: adversaryToProto(created)}, nil
}
