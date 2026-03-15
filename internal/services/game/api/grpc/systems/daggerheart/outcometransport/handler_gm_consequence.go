package outcometransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gmConsequenceResolution caches the follow-up gate and spotlight writes needed
// when a fear outcome requires a GM consequence.
type gmConsequenceResolution struct {
	needsGate            bool
	gateID               string
	gatePayloadJSON      []byte
	needsSpotlight       bool
	spotlightPayloadJSON []byte
}

// resolveGMConsequence computes the gate and spotlight repairs needed for a GM
// consequence without applying them yet.
func (h *Handler) resolveGMConsequence(
	ctx context.Context,
	campaignID, sessionID string,
	rollSeq uint64,
	rollRequestID string,
) (gmConsequenceResolution, error) {
	if h.deps.SessionGate == nil {
		return gmConsequenceResolution{}, status.Error(codes.Internal, "session gate store is not configured")
	}
	if h.deps.SessionSpotlight == nil {
		return gmConsequenceResolution{}, status.Error(codes.Internal, "session spotlight store is not configured")
	}

	var res gmConsequenceResolution

	gateOpen := false
	if _, err := h.deps.SessionGate.GetOpenSessionGate(ctx, campaignID, sessionID); err == nil {
		gateOpen = true
	} else if !errors.Is(err, storage.ErrNotFound) {
		return res, grpcerror.Internal("check session gate", err)
	}
	if !gateOpen {
		gateID, err := id.NewID()
		if err != nil {
			return res, grpcerror.Internal("generate gate id", err)
		}
		gateType, err := session.NormalizeGateType("gm_consequence")
		if err != nil {
			return res, grpcerror.Internal("normalize gate type", err)
		}
		payload := session.GateOpenedPayload{
			GateID:   ids.GateID(gateID),
			GateType: gateType,
			Reason:   "gm_consequence",
			Metadata: map[string]any{
				"roll_seq":   rollSeq,
				"request_id": rollRequestID,
			},
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return res, grpcerror.Internal("encode session gate payload", err)
		}
		res.needsGate = true
		res.gateID = gateID
		res.gatePayloadJSON = payloadJSON
	}

	spotlight, err := h.deps.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err == nil {
		if spotlight.SpotlightType == session.SpotlightTypeGM && strings.TrimSpace(spotlight.CharacterID) == "" {
			return res, nil
		}
	} else if !errors.Is(err, storage.ErrNotFound) {
		return res, grpcerror.Internal("check session spotlight", err)
	}

	spotlightPayload := session.SpotlightSetPayload{SpotlightType: string(session.SpotlightTypeGM)}
	payloadJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		return res, grpcerror.Internal("encode spotlight payload", err)
	}
	res.needsSpotlight = true
	res.spotlightPayloadJSON = payloadJSON
	return res, nil
}

// buildGMConsequenceOutcomeEffects reports the follow-up effects an outcome
// event should carry when a GM consequence is required.
func (h *Handler) buildGMConsequenceOutcomeEffects(
	ctx context.Context,
	campaignID string,
	sessionID string,
	rollSeq uint64,
	rollRequestID string,
) ([]action.OutcomeAppliedEffect, error) {
	res, err := h.resolveGMConsequence(ctx, campaignID, sessionID, rollSeq, rollRequestID)
	if err != nil {
		return nil, err
	}

	effects := make([]action.OutcomeAppliedEffect, 0, 2)
	if res.needsGate {
		effects = append(effects, action.OutcomeAppliedEffect{
			Type:        "session.gate_opened",
			EntityType:  "session_gate",
			EntityID:    res.gateID,
			PayloadJSON: res.gatePayloadJSON,
		})
	}
	if res.needsSpotlight {
		effects = append(effects, action.OutcomeAppliedEffect{
			Type:        "session.spotlight_set",
			EntityType:  "session_spotlight",
			EntityID:    sessionID,
			PayloadJSON: res.spotlightPayloadJSON,
		})
	}
	return effects, nil
}

// openGMConsequenceGate repairs the session gate and spotlight immediately for
// idempotent retries that must still surface an open consequence.
func (h *Handler) openGMConsequenceGate(ctx context.Context, campaignID, sessionID, sceneID string, rollSeq uint64, rollRequestID string) error {
	res, err := h.resolveGMConsequence(ctx, campaignID, sessionID, rollSeq, rollRequestID)
	if err != nil {
		return err
	}

	if res.needsGate {
		if err := h.deps.ExecuteCoreCommand(ctx, CoreCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandTypeSessionGateOpen,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       rollRequestID,
			InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
			EntityType:      "session_gate",
			EntityID:        res.gateID,
			PayloadJSON:     res.gatePayloadJSON,
			MissingEventMsg: "session gate open did not emit an event",
			ApplyErrMessage: "apply session gate event",
		}); err != nil {
			return err
		}
	}

	if res.needsSpotlight {
		if err := h.deps.ExecuteCoreCommand(ctx, CoreCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandTypeSessionSpotlightSet,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       rollRequestID,
			InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
			EntityType:      "session_spotlight",
			EntityID:        sessionID,
			PayloadJSON:     res.spotlightPayloadJSON,
			MissingEventMsg: "session spotlight set did not emit an event",
			ApplyErrMessage: "apply spotlight event",
		}); err != nil {
			return err
		}
	}

	return nil
}
