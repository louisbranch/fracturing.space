package gmmovetransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gmconsequence"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- adversary / environment entity loaders ---

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

// --- adversary spotlight validation and recording ---

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
	if nextCount > rules.AdversarySpotlightCap(entry) {
		return status.Errorf(codes.FailedPrecondition, "adversary spotlight cap reached for gate %s", gate.GateID)
	}
	return nil
}

func (h *Handler) recordAdversarySpotlight(ctx context.Context, campaignID, sessionID, sceneID string, adversary projectionstore.DaggerheartAdversary, gateID string) error {
	nextCount := 1
	if strings.TrimSpace(adversary.SpotlightGateID) == strings.TrimSpace(gateID) {
		nextCount = adversary.SpotlightCount + 1
	}
	payloadJSON, err := json.Marshal(daggerheartpayload.AdversaryUpdatePayload{
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

// --- GM consequence gate management ---

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

// --- content lookup helpers ---

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

func findAdversaryExperience(entry contentstore.DaggerheartAdversaryEntry, experienceName string) (contentstore.DaggerheartAdversaryExperience, bool) {
	for _, experience := range entry.Experiences {
		if strings.EqualFold(strings.TrimSpace(experience.Name), experienceName) {
			return experience, true
		}
	}
	return contentstore.DaggerheartAdversaryExperience{}, false
}

// --- adversary feature staging helpers ---

func stagedFearFeaturePayload(adversary projectionstore.DaggerheartAdversary, feature contentstore.DaggerheartAdversaryFeature, focusedTargetID string) *daggerheartpayload.AdversaryFeatureApplyPayload {
	automationStatus, rule := rules.ResolveAdversaryFeatureRuntime(feature)
	if automationStatus != rules.AdversaryFeatureAutomationStatusSupported || rule == nil {
		return nil
	}
	switch rule.Kind {
	case rules.AdversaryFeatureRuleKindHiddenUntilNextAttack, rules.AdversaryFeatureRuleKindDifficultyBonusWhileActive, rules.AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit, rules.AdversaryFeatureRuleKindFocusTargetDisadvantage:
	default:
		return nil
	}
	nextStates := upsertFeatureState(adversary.FeatureStates, projectionstore.DaggerheartAdversaryFeatureState{
		FeatureID:       strings.TrimSpace(feature.ID),
		Status:          stageStatusForRule(rule),
		FocusedTargetID: strings.TrimSpace(focusedTargetID),
	})
	return &daggerheartpayload.AdversaryFeatureApplyPayload{
		ActorAdversaryID:        ids.AdversaryID(adversary.AdversaryID),
		AdversaryID:             ids.AdversaryID(adversary.AdversaryID),
		FeatureID:               strings.TrimSpace(feature.ID),
		FeatureStatesBefore:     toBridgeAdversaryFeatureStates(adversary.FeatureStates),
		FeatureStatesAfter:      toBridgeAdversaryFeatureStates(nextStates),
		PendingExperienceBefore: toBridgeAdversaryPendingExperience(adversary.PendingExperience),
		PendingExperienceAfter:  toBridgeAdversaryPendingExperience(adversary.PendingExperience),
	}
}

func stageStatusForRule(rule *rules.AdversaryFeatureRule) string {
	switch rule.Kind {
	case rules.AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit:
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

// --- proto/domain bridge mappers ---

func toBridgeAdversaryFeatureStates(in []projectionstore.DaggerheartAdversaryFeatureState) []rules.AdversaryFeatureState {
	out := make([]rules.AdversaryFeatureState, 0, len(in))
	for _, state := range in {
		out = append(out, rules.AdversaryFeatureState{
			FeatureID:       strings.TrimSpace(state.FeatureID),
			Status:          strings.TrimSpace(state.Status),
			FocusedTargetID: strings.TrimSpace(state.FocusedTargetID),
		})
	}
	return out
}

func toBridgeAdversaryPendingExperience(in *projectionstore.DaggerheartAdversaryPendingExperience) *rules.AdversaryPendingExperience {
	if in == nil {
		return nil
	}
	return &rules.AdversaryPendingExperience{
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
