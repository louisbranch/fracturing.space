package adversarytransport

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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
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
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
	adversaryEntryID, err := validate.RequiredID(in.GetAdversaryEntryId(), "adversary entry id")
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
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart adversaries"); err != nil {
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
	entry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, adversaryEntryID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "adversary entry not found")
		}
		return nil, grpcerror.Internal("load adversary entry", err)
	}

	adversaryID, err := h.deps.GenerateID()
	if err != nil {
		return nil, grpcerror.Internal("generate adversary id", err)
	}
	payloadJSON, err := json.Marshal(daggerheart.AdversaryCreatePayload{
		AdversaryID:      ids.AdversaryID(adversaryID),
		AdversaryEntryID: adversaryEntryID,
		Name:             entry.Name,
		Kind:             entry.Role,
		SessionID:        ids.SessionID(sessionID),
		SceneID:          ids.SceneID(sceneID),
		Notes:            notes,
		HP:               entry.HP,
		HPMax:            entry.HP,
		Stress:           entry.Stress,
		StressMax:        entry.Stress,
		Evasion:          entry.Difficulty,
		Major:            entry.MajorThreshold,
		Severe:           entry.SevereThreshold,
		Armor:            entry.Armor,
		FeatureStates:    []daggerheart.AdversaryFeatureState{},
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
	if strings.TrimSpace(in.GetSceneId()) == "" && in.Notes == nil {
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

	sceneID := current.SceneID
	if strings.TrimSpace(in.GetSceneId()) != "" {
		sceneID = strings.TrimSpace(in.GetSceneId())
	}
	notes := current.Notes
	if in.Notes != nil {
		notes = strings.TrimSpace(in.Notes.GetValue())
	}
	sessionID := current.SessionID

	payloadJSON, err := json.Marshal(daggerheart.AdversaryUpdatePayload{
		AdversaryID:       ids.AdversaryID(adversaryID),
		AdversaryEntryID:  current.AdversaryEntryID,
		Name:              current.Name,
		Kind:              current.Kind,
		SessionID:         ids.SessionID(sessionID),
		SceneID:           ids.SceneID(sceneID),
		Notes:             notes,
		HP:                current.HP,
		HPMax:             current.HPMax,
		Stress:            current.Stress,
		StressMax:         current.StressMax,
		Evasion:           current.Evasion,
		Major:             current.Major,
		Severe:            current.Severe,
		Armor:             current.Armor,
		FeatureStates:     toBridgeAdversaryFeatureStates(current.FeatureStates),
		PendingExperience: toBridgeAdversaryPendingExperience(current.PendingExperience),
		SpotlightGateID:   ids.GateID(current.SpotlightGateID),
		SpotlightCount:    current.SpotlightCount,
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

func (h *Handler) ApplyAdversaryFeature(ctx context.Context, in *pb.DaggerheartApplyAdversaryFeatureRequest) (*pb.DaggerheartApplyAdversaryFeatureResponse, error) {
	if in == nil {
		return nil, invalidArgument("apply adversary feature request is required")
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
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
	}
	featureID, err := validate.RequiredID(in.GetFeatureId(), "feature id")
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
	if _, err := h.deps.Session.GetSession(ctx, campaignID, sessionID); err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.Gate, campaignID, sessionID); err != nil {
		return nil, err
	}
	adversary, err := loadAdversaryForSession(ctx, h.deps.Daggerheart, campaignID, sessionID, adversaryID)
	if err != nil {
		return nil, err
	}
	entry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, adversary.AdversaryEntryID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "adversary entry not found")
		}
		return nil, grpcerror.Internal("load adversary entry", err)
	}
	feature, ok := findEntryFeature(entry, featureID)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "adversary feature %q was not found on adversary entry %q", featureID, adversary.AdversaryEntryID)
	}
	automationStatus, rule := daggerheart.ResolveAdversaryFeatureRuntime(feature)
	if automationStatus != daggerheart.AdversaryFeatureAutomationStatusSupported || rule == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "adversary feature %q is not runtime-supported", featureID)
	}
	if strings.EqualFold(strings.TrimSpace(feature.CostType), "fear") {
		return nil, status.Errorf(codes.InvalidArgument, "fear-cost adversary feature %q must use ApplyGmMove", featureID)
	}
	nextStress := adversary.Stress
	if strings.EqualFold(strings.TrimSpace(feature.CostType), "stress") {
		if feature.Cost <= 0 {
			return nil, status.Errorf(codes.FailedPrecondition, "adversary feature %q has an invalid stress cost", featureID)
		}
		if adversary.Stress < feature.Cost {
			return nil, status.Errorf(codes.FailedPrecondition, "adversary %q does not have enough stress", adversaryID)
		}
		nextStress -= feature.Cost
	}
	nextFeatureStates := upsertAdversaryFeatureState(adversary.FeatureStates, daggerheart.AdversaryFeatureState{
		FeatureID:       featureID,
		Status:          featureApplyStateStatus(rule),
		FocusedTargetID: strings.TrimSpace(in.GetTargetCharacterId()),
	})
	payloadJSON, err := json.Marshal(daggerheart.AdversaryFeatureApplyPayload{
		ActorAdversaryID:        ids.AdversaryID(adversaryID),
		AdversaryID:             ids.AdversaryID(adversaryID),
		FeatureID:               featureID,
		TargetCharacterID:       ids.CharacterID(strings.TrimSpace(in.GetTargetCharacterId())),
		TargetAdversaryID:       ids.AdversaryID(strings.TrimSpace(in.GetTargetAdversaryId())),
		StressBefore:            intPtr(adversary.Stress),
		StressAfter:             intPtr(nextStress),
		FeatureStatesBefore:     toBridgeAdversaryFeatureStates(adversary.FeatureStates),
		FeatureStatesAfter:      toBridgeAdversaryFeatureStates(nextFeatureStates),
		PendingExperienceBefore: toBridgeAdversaryPendingExperience(adversary.PendingExperience),
		PendingExperienceAfter:  toBridgeAdversaryPendingExperience(adversary.PendingExperience),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode adversary feature payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartAdversaryFeatureApply,
		SessionID:       sessionID,
		SceneID:         strings.TrimSpace(in.GetSceneId()),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "adversary",
		EntityID:        adversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary feature apply did not emit an event",
		ApplyErrMessage: "apply adversary feature event",
	}); err != nil {
		return nil, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, grpcerror.Internal("load adversary", err)
	}
	return &pb.DaggerheartApplyAdversaryFeatureResponse{Adversary: adversaryToProto(updated)}, nil
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

func findEntryFeature(entry contentstore.DaggerheartAdversaryEntry, featureID string) (contentstore.DaggerheartAdversaryFeature, bool) {
	for _, feature := range entry.Features {
		if strings.TrimSpace(feature.ID) == strings.TrimSpace(featureID) {
			return feature, true
		}
	}
	return contentstore.DaggerheartAdversaryFeature{}, false
}

func featureApplyStateStatus(rule *daggerheart.AdversaryFeatureRule) string {
	switch rule.Kind {
	case daggerheart.AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit:
		return "ready"
	default:
		return "active"
	}
}

func upsertAdversaryFeatureState(current []projectionstore.DaggerheartAdversaryFeatureState, next daggerheart.AdversaryFeatureState) []projectionstore.DaggerheartAdversaryFeatureState {
	updated := make([]projectionstore.DaggerheartAdversaryFeatureState, 0, len(current)+1)
	seen := false
	for _, state := range current {
		if strings.TrimSpace(state.FeatureID) == strings.TrimSpace(next.FeatureID) {
			updated = append(updated, projectionstore.DaggerheartAdversaryFeatureState{
				FeatureID:       strings.TrimSpace(next.FeatureID),
				Status:          strings.TrimSpace(next.Status),
				FocusedTargetID: strings.TrimSpace(next.FocusedTargetID),
			})
			seen = true
			continue
		}
		updated = append(updated, state)
	}
	if !seen {
		updated = append(updated, projectionstore.DaggerheartAdversaryFeatureState{
			FeatureID:       strings.TrimSpace(next.FeatureID),
			Status:          strings.TrimSpace(next.Status),
			FocusedTargetID: strings.TrimSpace(next.FocusedTargetID),
		})
	}
	return updated
}

func toBridgeAdversaryFeatureStates(in []projectionstore.DaggerheartAdversaryFeatureState) []daggerheart.AdversaryFeatureState {
	out := make([]daggerheart.AdversaryFeatureState, 0, len(in))
	for _, state := range in {
		out = append(out, daggerheart.AdversaryFeatureState{
			FeatureID:       strings.TrimSpace(state.FeatureID),
			Status:          strings.TrimSpace(state.Status),
			FocusedTargetID: strings.TrimSpace(state.FocusedTargetID),
		})
	}
	return out
}

func toBridgeAdversaryPendingExperience(in *projectionstore.DaggerheartAdversaryPendingExperience) *daggerheart.AdversaryPendingExperience {
	if in == nil {
		return nil
	}
	return &daggerheart.AdversaryPendingExperience{
		Name:     strings.TrimSpace(in.Name),
		Modifier: in.Modifier,
	}
}

func intPtr(value int) *int {
	return &value
}

func invalidArgument(message string) error {
	return statusError(codes.InvalidArgument, message)
}

func internal(message string) error {
	return statusError(codes.Internal, message)
}
