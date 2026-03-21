package environmenttransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewHandler(deps Dependencies) *Handler {
	if deps.GenerateID == nil {
		deps.GenerateID = id.NewID
	}
	return &Handler{deps: deps}
}

func (h *Handler) CreateEnvironmentEntity(ctx context.Context, in *pb.DaggerheartCreateEnvironmentEntityRequest) (*pb.DaggerheartCreateEnvironmentEntityResponse, error) {
	if in == nil {
		return nil, invalidArgument("create environment entity request is required")
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
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
	environmentID, err := validate.RequiredID(in.GetEnvironmentId(), "environment id")
	if err != nil {
		return nil, err
	}
	notes := strings.TrimSpace(in.GetNotes())

	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart environment entities"); err != nil {
		return nil, err
	}
	if h.deps.Session == nil {
		return nil, internal("session store is not configured")
	}
	if _, err := h.deps.Session.GetSession(ctx, campaignID, sessionID); err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.Gate, campaignID, sessionID); err != nil {
		return nil, err
	}
	entry, err := h.deps.Content.GetDaggerheartEnvironment(ctx, environmentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "environment not found")
		}
		return nil, grpcerror.Internal("load environment", err)
	}
	difficulty := entry.Difficulty
	if in.Difficulty != nil {
		difficulty = int(in.Difficulty.GetValue())
	}
	if difficulty <= 0 {
		return nil, invalidArgument("difficulty must be greater than zero")
	}

	environmentEntityID, err := h.deps.GenerateID()
	if err != nil {
		return nil, grpcerror.Internal("generate environment entity id", err)
	}
	payloadJSON, err := json.Marshal(daggerheart.EnvironmentEntityCreatePayload{
		EnvironmentEntityID: ids.EnvironmentEntityID(environmentEntityID),
		EnvironmentID:       environmentID,
		Name:                entry.Name,
		Type:                entry.Type,
		Tier:                entry.Tier,
		Difficulty:          difficulty,
		SessionID:           ids.SessionID(sessionID),
		SceneID:             ids.SceneID(sceneID),
		Notes:               notes,
	})
	if err != nil {
		return nil, grpcerror.Internal("encode environment entity payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartEnvironmentEntityCreate,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "environment_entity",
		EntityID:        environmentEntityID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "environment entity create did not emit an event",
		ApplyErrMessage: "apply environment entity created event",
	}); err != nil {
		return nil, err
	}
	created, err := h.deps.Daggerheart.GetDaggerheartEnvironmentEntity(ctx, campaignID, environmentEntityID)
	if err != nil {
		return nil, grpcerror.Internal("load environment entity", err)
	}
	return &pb.DaggerheartCreateEnvironmentEntityResponse{EnvironmentEntity: environmentEntityToProto(created)}, nil
}

func (h *Handler) UpdateEnvironmentEntity(ctx context.Context, in *pb.DaggerheartUpdateEnvironmentEntityRequest) (*pb.DaggerheartUpdateEnvironmentEntityResponse, error) {
	if in == nil {
		return nil, invalidArgument("update environment entity request is required")
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
	environmentEntityID, err := validate.RequiredID(in.GetEnvironmentEntityId(), "environment entity id")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetSceneId()) == "" && in.Notes == nil && in.Difficulty == nil {
		return nil, invalidArgument("at least one field is required")
	}

	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart environment entities"); err != nil {
		return nil, err
	}

	current, err := h.deps.Daggerheart.GetDaggerheartEnvironmentEntity(ctx, campaignID, environmentEntityID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if current.SessionID != "" {
		if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.Gate, campaignID, current.SessionID); err != nil {
			return nil, err
		}
	}

	sceneID := current.SceneID
	if strings.TrimSpace(in.GetSceneId()) != "" {
		sceneID = strings.TrimSpace(in.GetSceneId())
	}
	notes := current.Notes
	if in.Notes != nil {
		notes = strings.TrimSpace(in.Notes.GetValue())
	}
	difficulty := current.Difficulty
	if in.Difficulty != nil {
		difficulty = int(in.Difficulty.GetValue())
	}
	if difficulty <= 0 {
		return nil, invalidArgument("difficulty must be greater than zero")
	}

	payloadJSON, err := json.Marshal(daggerheart.EnvironmentEntityUpdatePayload{
		EnvironmentEntityID: ids.EnvironmentEntityID(environmentEntityID),
		EnvironmentID:       current.EnvironmentID,
		Name:                current.Name,
		Type:                current.Type,
		Tier:                current.Tier,
		Difficulty:          difficulty,
		SessionID:           ids.SessionID(current.SessionID),
		SceneID:             ids.SceneID(sceneID),
		Notes:               notes,
	})
	if err != nil {
		return nil, grpcerror.Internal("encode environment entity payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartEnvironmentEntityUpdate,
		SessionID:       current.SessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "environment_entity",
		EntityID:        environmentEntityID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "environment entity update did not emit an event",
		ApplyErrMessage: "apply environment entity updated event",
	}); err != nil {
		return nil, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartEnvironmentEntity(ctx, campaignID, environmentEntityID)
	if err != nil {
		return nil, grpcerror.Internal("load environment entity", err)
	}
	return &pb.DaggerheartUpdateEnvironmentEntityResponse{EnvironmentEntity: environmentEntityToProto(updated)}, nil
}

func (h *Handler) DeleteEnvironmentEntity(ctx context.Context, in *pb.DaggerheartDeleteEnvironmentEntityRequest) (*pb.DaggerheartDeleteEnvironmentEntityResponse, error) {
	if in == nil {
		return nil, invalidArgument("delete environment entity request is required")
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
	environmentEntityID, err := validate.RequiredID(in.GetEnvironmentEntityId(), "environment entity id")
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
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart environment entities"); err != nil {
		return nil, err
	}

	current, err := h.deps.Daggerheart.GetDaggerheartEnvironmentEntity(ctx, campaignID, environmentEntityID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if current.SessionID != "" {
		if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.Gate, campaignID, current.SessionID); err != nil {
			return nil, err
		}
	}
	sceneID := current.SceneID
	if strings.TrimSpace(in.GetSceneId()) != "" {
		sceneID = strings.TrimSpace(in.GetSceneId())
	}

	payloadJSON, err := json.Marshal(daggerheart.EnvironmentEntityDeletePayload{
		EnvironmentEntityID: ids.EnvironmentEntityID(environmentEntityID),
		Reason:              strings.TrimSpace(in.GetReason()),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode environment entity delete payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartEnvironmentEntityDelete,
		SessionID:       current.SessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "environment_entity",
		EntityID:        environmentEntityID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "environment entity delete did not emit an event",
		ApplyErrMessage: "apply environment entity deleted event",
	}); err != nil {
		return nil, err
	}
	return &pb.DaggerheartDeleteEnvironmentEntityResponse{EnvironmentEntity: environmentEntityToProto(current)}, nil
}

func (h *Handler) GetEnvironmentEntity(ctx context.Context, in *pb.DaggerheartGetEnvironmentEntityRequest) (*pb.DaggerheartGetEnvironmentEntityResponse, error) {
	if in == nil {
		return nil, invalidArgument("get environment entity request is required")
	}
	if err := h.requireBaseDependencies(); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	environmentEntityID, err := validate.RequiredID(in.GetEnvironmentEntityId(), "environment entity id")
	if err != nil {
		return nil, err
	}
	environmentEntity, err := h.deps.Daggerheart.GetDaggerheartEnvironmentEntity(ctx, campaignID, environmentEntityID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	return &pb.DaggerheartGetEnvironmentEntityResponse{EnvironmentEntity: environmentEntityToProto(environmentEntity)}, nil
}

func (h *Handler) ListEnvironmentEntities(ctx context.Context, in *pb.DaggerheartListEnvironmentEntitiesRequest) (*pb.DaggerheartListEnvironmentEntitiesResponse, error) {
	if in == nil {
		return nil, invalidArgument("list environment entities request is required")
	}
	if err := h.requireBaseDependencies(); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	sceneID := ""
	if in.SceneId != nil {
		sceneID = strings.TrimSpace(in.SceneId.GetValue())
	}
	environmentEntities, err := h.deps.Daggerheart.ListDaggerheartEnvironmentEntities(ctx, campaignID, sessionID, sceneID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	resp := &pb.DaggerheartListEnvironmentEntitiesResponse{EnvironmentEntities: make([]*pb.DaggerheartEnvironmentEntity, 0, len(environmentEntities))}
	for _, environmentEntity := range environmentEntities {
		resp.EnvironmentEntities = append(resp.EnvironmentEntities, environmentEntityToProto(environmentEntity))
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
	return status.Error(codes.InvalidArgument, message)
}

func internal(message string) error {
	return status.Error(codes.Internal, message)
}
