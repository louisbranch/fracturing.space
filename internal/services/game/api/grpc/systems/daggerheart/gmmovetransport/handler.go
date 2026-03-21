package gmmovetransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gmconsequence"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns the Daggerheart GM-move transport endpoint.
type Handler struct {
	deps Dependencies
}

type gmMoveResolution struct {
	target                  daggerheart.GMMoveTarget
	opensGMConsequence      bool
	spotlightAdversary      *projectionstore.DaggerheartAdversary
	adversaryFeaturePayload *daggerheart.AdversaryFeatureApplyPayload
}

// NewHandler builds a Daggerheart GM-move transport handler from explicit
// read-store and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) ApplyGmMove(ctx context.Context, in *pb.DaggerheartApplyGmMoveRequest) (Result, error) {
	if in == nil {
		return Result{}, status.Error(codes.InvalidArgument, "apply gm move request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return Result{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return Result{}, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return Result{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	resolution, err := h.gmMoveTargetFromProto(ctx, campaignID, sessionID, in)
	if err != nil {
		return Result{}, err
	}
	fearSpent := int(in.GetFearSpent())
	if fearSpent <= 0 {
		return Result{}, status.Error(codes.InvalidArgument, "fear_spent must be greater than zero")
	}

	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return Result{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpSessionAction); err != nil {
		return Result{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart gm moves"); err != nil {
		return Result{}, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return Result{}, grpcerror.HandleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return Result{}, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if resolution.spotlightAdversary != nil {
		if err := h.validateAdversarySpotlight(ctx, campaignID, sessionID, *resolution.spotlightAdversary); err != nil {
			return Result{}, err
		}
	} else {
		if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
			return Result{}, err
		}
	}

	gmFearBefore := 0
	gmFearAfter := 0
	if snap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID); err == nil {
		gmFearBefore = snap.GMFear
		gmFearAfter = snap.GMFear
	}

	before, after, err := daggerheart.ApplyGMFearSpend(gmFearBefore, fearSpent)
	if err != nil {
		return Result{}, status.Error(codes.InvalidArgument, err.Error())
	}
	gmFearBefore = before
	gmFearAfter = after

	payloadJSON, err := json.Marshal(daggerheart.GMMoveApplyPayload{
		Target:    resolution.target,
		FearSpent: fearSpent,
	})
	if err != nil {
		return Result{}, grpcerror.Internal("encode gm move payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartGMMoveApply,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "session",
		EntityID:        sessionID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "gm move did not emit an event",
		ApplyErrMessage: "apply gm move event",
	}); err != nil {
		return Result{}, err
	}
	if resolution.adversaryFeaturePayload != nil {
		adversaryPayloadJSON, err := json.Marshal(resolution.adversaryFeaturePayload)
		if err != nil {
			return Result{}, grpcerror.Internal("encode adversary feature gm move payload", err)
		}
		if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandids.DaggerheartAdversaryFeatureApply,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       grpcmeta.RequestIDFromContext(ctx),
			InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
			EntityType:      "adversary",
			EntityID:        resolution.adversaryFeaturePayload.AdversaryID.String(),
			PayloadJSON:     adversaryPayloadJSON,
			MissingEventMsg: "gm move adversary feature stage did not emit an event",
			ApplyErrMessage: "apply adversary feature stage",
		}); err != nil {
			return Result{}, err
		}
	}
	if resolution.opensGMConsequence {
		if err := gmconsequence.Open(
			ctx,
			h.gmConsequenceDependencies(),
			campaignID,
			sessionID,
			sceneID,
			grpcmeta.RequestIDFromContext(ctx),
			grpcmeta.InvocationIDFromContext(ctx),
			nil,
		); err != nil {
			return Result{}, err
		}
	}
	if resolution.spotlightAdversary != nil {
		gateID, err := h.ensureGMConsequenceGate(ctx, campaignID, sessionID, sceneID)
		if err != nil {
			return Result{}, err
		}
		if err := h.recordAdversarySpotlight(ctx, campaignID, sessionID, sceneID, *resolution.spotlightAdversary, gateID); err != nil {
			return Result{}, err
		}
	}

	return Result{
		CampaignID:   campaignID,
		GMFearBefore: gmFearBefore,
		GMFearAfter:  gmFearAfter,
	}, nil
}

func (h *Handler) requireDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.SessionSpotlight == nil:
		return status.Error(codes.Internal, "session spotlight store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Content == nil:
		return status.Error(codes.Internal, "daggerheart content store is not configured")
	case h.deps.ExecuteDomainCommand == nil:
		return status.Error(codes.Internal, "domain command executor is not configured")
	case h.deps.ExecuteCoreCommand == nil:
		return status.Error(codes.Internal, "core command executor is not configured")
	default:
		return nil
	}
}

func (h *Handler) gmConsequenceDependencies() gmconsequence.Dependencies {
	return gmconsequence.Dependencies{
		SessionGate:        h.deps.SessionGate,
		SessionSpotlight:   h.deps.SessionSpotlight,
		ExecuteCoreCommand: h.deps.ExecuteCoreCommand,
	}
}

func gmMoveKindFromProto(kind pb.DaggerheartGmMoveKind) (daggerheart.GMMoveKind, error) {
	switch kind {
	case pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_INTERRUPT_AND_MOVE:
		return daggerheart.GMMoveKindInterruptAndMove, nil
	case pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE:
		return daggerheart.GMMoveKindAdditionalMove, nil
	default:
		return daggerheart.GMMoveKindUnspecified, status.Error(codes.InvalidArgument, "gm move kind is required")
	}
}

func gmMoveShapeFromProto(shape pb.DaggerheartGmMoveShape) (daggerheart.GMMoveShape, error) {
	switch shape {
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHOW_WORLD_REACTION:
		return daggerheart.GMMoveShapeShowWorldReaction, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_REVEAL_DANGER:
		return daggerheart.GMMoveShapeRevealDanger, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_FORCE_SPLIT:
		return daggerheart.GMMoveShapeForceSplit, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_MARK_STRESS:
		return daggerheart.GMMoveShapeMarkStress, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT:
		return daggerheart.GMMoveShapeShiftEnvironment, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SPOTLIGHT_ADVERSARY:
		return daggerheart.GMMoveShapeSpotlightAdversary, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CAPTURE_IMPORTANT_TARGET:
		return daggerheart.GMMoveShapeCaptureImportantTarget, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CUSTOM:
		return daggerheart.GMMoveShapeCustom, nil
	default:
		return daggerheart.GMMoveShapeUnspecified, status.Error(codes.InvalidArgument, "gm move shape is required")
	}
}

func (h *Handler) gmMoveTargetFromProto(ctx context.Context, campaignID, sessionID string, in *pb.DaggerheartApplyGmMoveRequest) (gmMoveResolution, error) {
	switch target := in.GetSpendTarget().(type) {
	case *pb.DaggerheartApplyGmMoveRequest_DirectMove:
		kind, err := gmMoveKindFromProto(target.DirectMove.GetKind())
		if err != nil {
			return gmMoveResolution{}, err
		}
		shape, err := gmMoveShapeFromProto(target.DirectMove.GetShape())
		if err != nil {
			return gmMoveResolution{}, err
		}
		description := strings.TrimSpace(target.DirectMove.GetDescription())
		if shape == daggerheart.GMMoveShapeCustom && description == "" {
			return gmMoveResolution{}, status.Error(codes.InvalidArgument, "gm move description is required for custom shape")
		}
		resolution := gmMoveResolution{
			target: daggerheart.GMMoveTarget{
				Type:        daggerheart.GMMoveTargetTypeDirectMove,
				Kind:        kind,
				Shape:       shape,
				Description: description,
			},
			opensGMConsequence: kind == daggerheart.GMMoveKindInterruptAndMove,
		}
		if shape == daggerheart.GMMoveShapeSpotlightAdversary {
			adversaryID, err := validate.RequiredID(target.DirectMove.GetAdversaryId(), "adversary_id")
			if err != nil {
				return gmMoveResolution{}, err
			}
			adversary, err := h.loadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
			if err != nil {
				return gmMoveResolution{}, err
			}
			resolution.target.AdversaryID = ids.AdversaryID(adversaryID)
			resolution.spotlightAdversary = &adversary
		}
		return resolution, nil
	case *pb.DaggerheartApplyGmMoveRequest_AdversaryFeature:
		adversaryID, err := validate.RequiredID(target.AdversaryFeature.GetAdversaryId(), "adversary_id")
		if err != nil {
			return gmMoveResolution{}, err
		}
		featureID, err := validate.RequiredID(target.AdversaryFeature.GetFeatureId(), "feature_id")
		if err != nil {
			return gmMoveResolution{}, err
		}
		adversary, err := h.loadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
		if err != nil {
			return gmMoveResolution{}, err
		}
		entry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, adversary.AdversaryEntryID)
		if err != nil {
			return gmMoveResolution{}, mapContentErr("get adversary entry", err)
		}
		feature, ok := findAdversaryFeature(entry, featureID)
		if !ok {
			return gmMoveResolution{}, status.Errorf(codes.InvalidArgument, "adversary feature %q was not found on adversary entry %q", featureID, adversary.AdversaryEntryID)
		}
		if strings.TrimSpace(feature.CostType) != "fear" || feature.Cost <= 0 {
			return gmMoveResolution{}, status.Errorf(codes.InvalidArgument, "adversary feature %q is not a fear spend", featureID)
		}
		if int(in.GetFearSpent()) != feature.Cost {
			return gmMoveResolution{}, status.Errorf(codes.InvalidArgument, "fear_spent must equal adversary feature cost %d", feature.Cost)
		}
		resolution := gmMoveResolution{target: daggerheart.GMMoveTarget{
			Type:        daggerheart.GMMoveTargetTypeAdversaryFeature,
			AdversaryID: ids.AdversaryID(adversaryID),
			FeatureID:   featureID,
			Description: strings.TrimSpace(target.AdversaryFeature.GetDescription()),
		}}
		if payload := stagedFearFeaturePayload(adversary, feature, ""); payload != nil {
			resolution.adversaryFeaturePayload = payload
		}
		return resolution, nil
	case *pb.DaggerheartApplyGmMoveRequest_EnvironmentFeature:
		environmentEntityID, err := validate.RequiredID(target.EnvironmentFeature.GetEnvironmentEntityId(), "environment_entity_id")
		if err != nil {
			return gmMoveResolution{}, err
		}
		featureID, err := validate.RequiredID(target.EnvironmentFeature.GetFeatureId(), "feature_id")
		if err != nil {
			return gmMoveResolution{}, err
		}
		environmentEntity, err := h.loadEnvironmentEntityForSession(ctx, campaignID, sessionID, environmentEntityID)
		if err != nil {
			return gmMoveResolution{}, err
		}
		env, err := h.deps.Content.GetDaggerheartEnvironment(ctx, environmentEntity.EnvironmentID)
		if err != nil {
			return gmMoveResolution{}, mapContentErr("get environment", err)
		}
		if _, ok := findEnvironmentFeature(env, featureID); !ok {
			return gmMoveResolution{}, status.Errorf(codes.InvalidArgument, "environment feature %q was not found on environment %q", featureID, environmentEntity.EnvironmentID)
		}
		return gmMoveResolution{target: daggerheart.GMMoveTarget{
			Type:                daggerheart.GMMoveTargetTypeEnvironmentFeature,
			EnvironmentEntityID: ids.EnvironmentEntityID(environmentEntityID),
			EnvironmentID:       environmentEntity.EnvironmentID,
			FeatureID:           featureID,
			Description:         strings.TrimSpace(target.EnvironmentFeature.GetDescription()),
		}}, nil
	case *pb.DaggerheartApplyGmMoveRequest_AdversaryExperience:
		adversaryID, err := validate.RequiredID(target.AdversaryExperience.GetAdversaryId(), "adversary_id")
		if err != nil {
			return gmMoveResolution{}, err
		}
		experienceName, err := validate.RequiredID(target.AdversaryExperience.GetExperienceName(), "experience_name")
		if err != nil {
			return gmMoveResolution{}, err
		}
		adversary, err := h.loadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
		if err != nil {
			return gmMoveResolution{}, err
		}
		entry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, adversary.AdversaryEntryID)
		if err != nil {
			return gmMoveResolution{}, mapContentErr("get adversary entry", err)
		}
		experience, ok := findAdversaryExperience(entry, experienceName)
		if !ok {
			return gmMoveResolution{}, status.Errorf(codes.InvalidArgument, "adversary experience %q was not found on adversary entry %q", experienceName, adversary.AdversaryEntryID)
		}
		if in.GetFearSpent() != 1 {
			return gmMoveResolution{}, status.Error(codes.InvalidArgument, "adversary experience spends must spend exactly 1 fear")
		}
		return gmMoveResolution{target: daggerheart.GMMoveTarget{
			Type:           daggerheart.GMMoveTargetTypeAdversaryExperience,
			AdversaryID:    ids.AdversaryID(adversaryID),
			ExperienceName: experienceName,
			Description:    strings.TrimSpace(target.AdversaryExperience.GetDescription()),
		}, adversaryFeaturePayload: &daggerheart.AdversaryFeatureApplyPayload{
			ActorAdversaryID:        ids.AdversaryID(adversaryID),
			AdversaryID:             ids.AdversaryID(adversaryID),
			FeatureID:               "experience:" + experienceName,
			FeatureStatesBefore:     toBridgeAdversaryFeatureStates(adversary.FeatureStates),
			FeatureStatesAfter:      toBridgeAdversaryFeatureStates(adversary.FeatureStates),
			PendingExperienceBefore: toBridgeAdversaryPendingExperience(adversary.PendingExperience),
			PendingExperienceAfter: &daggerheart.AdversaryPendingExperience{
				Name:     experience.Name,
				Modifier: experience.Modifier,
			},
		}}, nil
	default:
		return gmMoveResolution{}, status.Error(codes.InvalidArgument, "gm move spend_target is required")
	}
}

func (h *Handler) loadAdversaryForSession(ctx context.Context, campaignID, sessionID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	adversary, err := h.deps.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return projectionstore.DaggerheartAdversary{}, status.Error(codes.NotFound, "adversary not found")
		}
		return projectionstore.DaggerheartAdversary{}, grpcerror.Internal("load adversary", err)
	}
	if adversary.SessionID != sessionID {
		return projectionstore.DaggerheartAdversary{}, status.Error(codes.FailedPrecondition, "adversary is not in session")
	}
	return adversary, nil
}

func (h *Handler) validateAdversarySpotlight(ctx context.Context, campaignID, sessionID string, adversary projectionstore.DaggerheartAdversary) error {
	gate, gateOpen, err := h.currentGMConsequenceGate(ctx, campaignID, sessionID)
	if err != nil {
		return err
	}
	if gateOpen {
		spotlight, err := h.deps.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return grpcerror.Internal("load session spotlight", err)
			}
		} else if spotlight.SpotlightType != session.SpotlightTypeGM || strings.TrimSpace(spotlight.CharacterID) != "" {
			return status.Error(codes.FailedPrecondition, "session spotlight is not gm-owned")
		}
	}
	entry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, adversary.AdversaryEntryID)
	if err != nil {
		return mapContentErr("get adversary entry", err)
	}
	nextCount := 1
	if gateOpen && strings.TrimSpace(adversary.SpotlightGateID) == gate.GateID {
		nextCount = adversary.SpotlightCount + 1
	}
	if nextCount > daggerheart.AdversarySpotlightCap(entry) {
		return status.Errorf(codes.FailedPrecondition, "adversary spotlight cap reached for gate %s", gate.GateID)
	}
	return nil
}

func (h *Handler) recordAdversarySpotlight(ctx context.Context, campaignID, sessionID, sceneID string, adversary projectionstore.DaggerheartAdversary, gateID string) error {
	nextCount := 1
	if strings.TrimSpace(adversary.SpotlightGateID) == strings.TrimSpace(gateID) {
		nextCount = adversary.SpotlightCount + 1
	}
	payloadJSON, err := json.Marshal(daggerheart.AdversaryUpdatePayload{
		AdversaryID:      ids.AdversaryID(adversary.AdversaryID),
		AdversaryEntryID: adversary.AdversaryEntryID,
		Name:             adversary.Name,
		Kind:             adversary.Kind,
		SessionID:        ids.SessionID(adversary.SessionID),
		SceneID:          ids.SceneID(adversary.SceneID),
		Notes:            adversary.Notes,
		HP:               adversary.HP,
		HPMax:            adversary.HPMax,
		Stress:           adversary.Stress,
		StressMax:        adversary.StressMax,
		Evasion:          adversary.Evasion,
		Major:            adversary.Major,
		Severe:           adversary.Severe,
		Armor:            adversary.Armor,
		SpotlightGateID:  ids.GateID(gateID),
		SpotlightCount:   nextCount,
	})
	if err != nil {
		return grpcerror.Internal("encode adversary spotlight payload", err)
	}
	return h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartAdversaryUpdate,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "adversary",
		EntityID:        adversary.AdversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary spotlight update did not emit an event",
		ApplyErrMessage: "apply adversary spotlight update",
	})
}

func (h *Handler) currentGMConsequenceGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, bool, error) {
	gate, err := h.deps.SessionGate.GetOpenSessionGate(ctx, campaignID, sessionID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.SessionGate{}, false, nil
		}
		return storage.SessionGate{}, false, grpcerror.Internal("load session gate", err)
	}
	if strings.TrimSpace(gate.GateType) != "gm_consequence" {
		return storage.SessionGate{}, false, status.Errorf(codes.FailedPrecondition, "session gate is open: %s", gate.GateID)
	}
	return gate, true, nil
}

func (h *Handler) ensureGMConsequenceGate(ctx context.Context, campaignID, sessionID, sceneID string) (string, error) {
	gate, gateOpen, err := h.currentGMConsequenceGate(ctx, campaignID, sessionID)
	if err != nil {
		return "", err
	}
	if gateOpen {
		return gate.GateID, nil
	}
	resolution, err := gmconsequence.Resolve(
		ctx,
		h.gmConsequenceDependencies(),
		campaignID,
		sessionID,
		nil,
		grpcmeta.RequestIDFromContext(ctx),
	)
	if err != nil {
		return "", err
	}
	if resolution.NeedsGate {
		if err := h.deps.ExecuteCoreCommand(ctx, gmconsequence.CoreCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandids.SessionGateOpen,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       grpcmeta.RequestIDFromContext(ctx),
			InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
			EntityType:      "session_gate",
			EntityID:        resolution.GateID,
			PayloadJSON:     resolution.GatePayloadJSON,
			MissingEventMsg: "session gate open did not emit an event",
			ApplyErrMessage: "apply session gate event",
		}); err != nil {
			return "", err
		}
	}
	if resolution.NeedsSpotlight {
		if err := h.deps.ExecuteCoreCommand(ctx, gmconsequence.CoreCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandids.SessionSpotlightSet,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       grpcmeta.RequestIDFromContext(ctx),
			InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
			EntityType:      "session_spotlight",
			EntityID:        sessionID,
			PayloadJSON:     resolution.SpotlightPayloadJSON,
			MissingEventMsg: "session spotlight set did not emit an event",
			ApplyErrMessage: "apply spotlight event",
		}); err != nil {
			return "", err
		}
	}
	if strings.TrimSpace(resolution.GateID) == "" {
		return "", status.Error(codes.FailedPrecondition, "gm consequence gate is not open")
	}
	return resolution.GateID, nil
}

func (h *Handler) requireCurrentGMConsequenceGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error) {
	gate, gateOpen, err := h.currentGMConsequenceGate(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionGate{}, err
	}
	if !gateOpen {
		return storage.SessionGate{}, status.Error(codes.FailedPrecondition, "gm consequence gate is not open")
	}
	return gate, nil
}

func findAdversaryFeature(entry contentstore.DaggerheartAdversaryEntry, featureID string) (contentstore.DaggerheartAdversaryFeature, bool) {
	for _, feature := range entry.Features {
		if strings.TrimSpace(feature.ID) == featureID {
			return feature, true
		}
	}
	return contentstore.DaggerheartAdversaryFeature{}, false
}

func findEnvironmentFeature(env contentstore.DaggerheartEnvironment, featureID string) (contentstore.DaggerheartFeature, bool) {
	for _, feature := range env.Features {
		if strings.TrimSpace(feature.ID) == featureID {
			return feature, true
		}
	}
	return contentstore.DaggerheartFeature{}, false
}

func (h *Handler) loadEnvironmentEntityForSession(ctx context.Context, campaignID, sessionID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	if h.deps.Daggerheart == nil {
		return projectionstore.DaggerheartEnvironmentEntity{}, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	environmentEntity, err := h.deps.Daggerheart.GetDaggerheartEnvironmentEntity(ctx, campaignID, environmentEntityID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return projectionstore.DaggerheartEnvironmentEntity{}, status.Error(codes.NotFound, "environment entity not found")
		}
		return projectionstore.DaggerheartEnvironmentEntity{}, grpcerror.Internal("load environment entity", err)
	}
	if environmentEntity.SessionID != "" && environmentEntity.SessionID != sessionID {
		return projectionstore.DaggerheartEnvironmentEntity{}, status.Error(codes.FailedPrecondition, "environment entity is not in session")
	}
	return environmentEntity, nil
}

func findAdversaryExperience(entry contentstore.DaggerheartAdversaryEntry, experienceName string) (contentstore.DaggerheartAdversaryExperience, bool) {
	for _, experience := range entry.Experiences {
		if strings.EqualFold(strings.TrimSpace(experience.Name), experienceName) {
			return experience, true
		}
	}
	return contentstore.DaggerheartAdversaryExperience{}, false
}

func stagedFearFeaturePayload(adversary projectionstore.DaggerheartAdversary, feature contentstore.DaggerheartAdversaryFeature, focusedTargetID string) *daggerheart.AdversaryFeatureApplyPayload {
	automationStatus, rule := daggerheart.ResolveAdversaryFeatureRuntime(feature)
	if automationStatus != daggerheart.AdversaryFeatureAutomationStatusSupported || rule == nil {
		return nil
	}
	switch rule.Kind {
	case daggerheart.AdversaryFeatureRuleKindHiddenUntilNextAttack, daggerheart.AdversaryFeatureRuleKindDifficultyBonusWhileActive, daggerheart.AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit, daggerheart.AdversaryFeatureRuleKindFocusTargetDisadvantage:
	default:
		return nil
	}
	nextStates := upsertFeatureState(adversary.FeatureStates, projectionstore.DaggerheartAdversaryFeatureState{
		FeatureID:       strings.TrimSpace(feature.ID),
		Status:          stageStatusForRule(rule),
		FocusedTargetID: strings.TrimSpace(focusedTargetID),
	})
	return &daggerheart.AdversaryFeatureApplyPayload{
		ActorAdversaryID:        ids.AdversaryID(adversary.AdversaryID),
		AdversaryID:             ids.AdversaryID(adversary.AdversaryID),
		FeatureID:               strings.TrimSpace(feature.ID),
		FeatureStatesBefore:     toBridgeAdversaryFeatureStates(adversary.FeatureStates),
		FeatureStatesAfter:      toBridgeAdversaryFeatureStates(nextStates),
		PendingExperienceBefore: toBridgeAdversaryPendingExperience(adversary.PendingExperience),
		PendingExperienceAfter:  toBridgeAdversaryPendingExperience(adversary.PendingExperience),
	}
}

func stageStatusForRule(rule *daggerheart.AdversaryFeatureRule) string {
	switch rule.Kind {
	case daggerheart.AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit:
		return "ready"
	default:
		return "active"
	}
}

func upsertFeatureState(current []projectionstore.DaggerheartAdversaryFeatureState, next projectionstore.DaggerheartAdversaryFeatureState) []projectionstore.DaggerheartAdversaryFeatureState {
	updated := make([]projectionstore.DaggerheartAdversaryFeatureState, 0, len(current)+1)
	seen := false
	for _, state := range current {
		if strings.TrimSpace(state.FeatureID) == strings.TrimSpace(next.FeatureID) {
			updated = append(updated, next)
			seen = true
			continue
		}
		updated = append(updated, state)
	}
	if !seen {
		updated = append(updated, next)
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

func mapContentErr(action string, err error) error {
	if err == nil {
		return nil
	}
	if err == storage.ErrNotFound {
		return status.Errorf(codes.NotFound, "%s: %v", action, err)
	}
	return grpcerror.Internal(action, err)
}
