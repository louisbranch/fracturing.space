package adversarytransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
)

// NewHandler builds an adversary transport handler from explicit reads and
// write callbacks.
func NewHandler(deps Dependencies) *Handler {
	if deps.GenerateID == nil {
		deps.GenerateID = id.NewID
	}
	return &Handler{deps: deps}
}

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

func (h *Handler) UpdateAdversary(ctx context.Context, in *pb.DaggerheartUpdateAdversaryRequest) (*pb.DaggerheartUpdateAdversaryResponse, error) {
	if in == nil {
		return nil, invalidArgument("update adversary request is required")
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
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
	}
	if in.Name == nil && in.Kind == nil && in.SessionId == nil && in.Notes == nil &&
		in.Hp == nil && in.HpMax == nil && in.Stress == nil && in.StressMax == nil &&
		in.Evasion == nil && in.MajorThreshold == nil && in.SevereThreshold == nil && in.Armor == nil {
		return nil, invalidArgument("at least one field is required")
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

	current, err := h.deps.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	currentSessionID := strings.TrimSpace(current.SessionID)
	if currentSessionID != "" {
		if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.Gate, campaignID, currentSessionID); err != nil {
			return nil, err
		}
	}

	name := current.Name
	if in.Name != nil {
		name, err = validate.RequiredID(in.Name.GetValue(), "name")
		if err != nil {
			return nil, err
		}
	}
	kind := current.Kind
	if in.Kind != nil {
		kind = strings.TrimSpace(in.Kind.GetValue())
	}
	sessionID := current.SessionID
	if in.SessionId != nil {
		sessionID = strings.TrimSpace(in.SessionId.GetValue())
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	notes := current.Notes
	if in.Notes != nil {
		notes = strings.TrimSpace(in.Notes.GetValue())
	}
	stats, err := normalizeAdversaryStats(adversaryStatsInput{
		HP:        in.Hp,
		HPMax:     in.HpMax,
		Stress:    in.Stress,
		StressMax: in.StressMax,
		Evasion:   in.Evasion,
		Major:     in.MajorThreshold,
		Severe:    in.SevereThreshold,
		Armor:     in.Armor,
		Current:   &current,
	})
	if err != nil {
		return nil, invalidArgument(err.Error())
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

	payloadJSON, err := json.Marshal(daggerheart.AdversaryUpdatePayload{
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
		CommandType:     commandids.DaggerheartAdversaryUpdate,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "adversary",
		EntityID:        adversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary update did not emit an event",
		ApplyErrMessage: "apply adversary updated event",
	}); err != nil {
		return nil, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, grpcerror.Internal("load adversary", err)
	}
	return &pb.DaggerheartUpdateAdversaryResponse{Adversary: adversaryToProto(updated)}, nil
}

func (h *Handler) DeleteAdversary(ctx context.Context, in *pb.DaggerheartDeleteAdversaryRequest) (*pb.DaggerheartDeleteAdversaryResponse, error) {
	if in == nil {
		return nil, invalidArgument("delete adversary request is required")
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
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
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

	current, err := h.deps.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	sessionID := strings.TrimSpace(current.SessionID)
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sessionID != "" {
		if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.Gate, campaignID, sessionID); err != nil {
			return nil, err
		}
	}
	payloadJSON, err := json.Marshal(daggerheart.AdversaryDeletePayload{
		AdversaryID: ids.AdversaryID(adversaryID),
		Reason:      strings.TrimSpace(in.GetReason()),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode adversary payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartAdversaryDelete,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "adversary",
		EntityID:        adversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary delete did not emit an event",
		ApplyErrMessage: "apply adversary deleted event",
	}); err != nil {
		return nil, err
	}
	return &pb.DaggerheartDeleteAdversaryResponse{Adversary: adversaryToProto(current)}, nil
}

func (h *Handler) GetAdversary(ctx context.Context, in *pb.DaggerheartGetAdversaryRequest) (*pb.DaggerheartGetAdversaryResponse, error) {
	if in == nil {
		return nil, invalidArgument("get adversary request is required")
	}
	if err := h.requireBaseDependencies(); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
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
	adversary, err := h.deps.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	return &pb.DaggerheartGetAdversaryResponse{Adversary: adversaryToProto(adversary)}, nil
}

func (h *Handler) ListAdversaries(ctx context.Context, in *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
	if in == nil {
		return nil, invalidArgument("list adversaries request is required")
	}
	if err := h.requireBaseDependencies(); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
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
	sessionID := ""
	if in.SessionId != nil {
		sessionID = strings.TrimSpace(in.SessionId.GetValue())
	}
	adversaries, err := h.deps.Daggerheart.ListDaggerheartAdversaries(ctx, campaignID, sessionID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	resp := &pb.DaggerheartListAdversariesResponse{Adversaries: make([]*pb.DaggerheartAdversary, 0, len(adversaries))}
	for _, adversary := range adversaries {
		resp.Adversaries = append(resp.Adversaries, adversaryToProto(adversary))
	}
	return resp, nil
}

func (h *Handler) requireBaseDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return internal("campaign store is not configured")
	case h.deps.Gate == nil:
		return internal("session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return internal("daggerheart store is not configured")
	default:
		return nil
	}
}

func invalidArgument(message string) error {
	return statusError(codes.InvalidArgument, message)
}

func internal(message string) error {
	return statusError(codes.Internal, message)
}
