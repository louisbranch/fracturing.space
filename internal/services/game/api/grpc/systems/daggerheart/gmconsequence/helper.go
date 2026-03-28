package gmconsequence

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SessionGateStore reads the open gate for a session.
type SessionGateStore = daggerheartguard.SessionGateStore

// SessionSpotlightStore reads the current session spotlight state.
type SessionSpotlightStore interface {
	GetSessionSpotlight(ctx context.Context, campaignID, sessionID string) (storage.SessionSpotlight, error)
}

// CoreCommandInput describes one core command emitted while repairing GM
// consequence state.
type CoreCommandInput = workflowwrite.CoreCommandInput

// Dependencies groups the exact read stores and write callback used by the GM
// consequence helper.
type Dependencies struct {
	SessionGate        SessionGateStore
	SessionSpotlight   SessionSpotlightStore
	ExecuteCoreCommand func(ctx context.Context, in CoreCommandInput) error
}

// Resolution captures the gate and spotlight repairs required to surface a GM
// consequence immediately.
type Resolution struct {
	NeedsGate            bool
	GateID               string
	GatePayloadJSON      []byte
	NeedsSpotlight       bool
	SpotlightPayloadJSON []byte
}

// Resolve computes the gate and spotlight repairs needed for a GM consequence
// without applying them.
func Resolve(ctx context.Context, deps Dependencies, campaignID, sessionID string, rollSeq *uint64, requestID string) (Resolution, error) {
	if deps.SessionGate == nil {
		return Resolution{}, status.Error(codes.Internal, "session gate store is not configured")
	}
	if deps.SessionSpotlight == nil {
		return Resolution{}, status.Error(codes.Internal, "session spotlight store is not configured")
	}

	var res Resolution

	gateOpen := false
	if _, err := deps.SessionGate.GetOpenSessionGate(ctx, campaignID, sessionID); err == nil {
		gateOpen = true
	} else if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "check session gate"); lookupErr != nil {
		return res, lookupErr
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
		metadata := map[string]any{
			"request_id": requestID,
		}
		if rollSeq != nil {
			metadata["roll_seq"] = *rollSeq
		}
		payload := session.GateOpenedPayload{
			GateID:   ids.GateID(gateID),
			GateType: gateType,
			Reason:   "gm_consequence",
			Metadata: metadata,
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return res, grpcerror.Internal("encode session gate payload", err)
		}
		res.NeedsGate = true
		res.GateID = gateID
		res.GatePayloadJSON = payloadJSON
	}

	spotlight, err := deps.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err == nil {
		if spotlight.SpotlightType == session.SpotlightTypeGM && strings.TrimSpace(spotlight.CharacterID) == "" {
			return res, nil
		}
	} else if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "check session spotlight"); lookupErr != nil {
		return res, lookupErr
	}

	spotlightPayload := session.SpotlightSetPayload{SpotlightType: string(session.SpotlightTypeGM)}
	payloadJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		return res, grpcerror.Internal("encode spotlight payload", err)
	}
	res.NeedsSpotlight = true
	res.SpotlightPayloadJSON = payloadJSON
	return res, nil
}

// Open applies the gate and spotlight repairs needed for a GM consequence.
func Open(
	ctx context.Context,
	deps Dependencies,
	campaignID, sessionID, sceneID, requestID, invocationID string,
	rollSeq *uint64,
) error {
	if deps.ExecuteCoreCommand == nil {
		return status.Error(codes.Internal, "core command executor is not configured")
	}

	res, err := Resolve(ctx, deps, campaignID, sessionID, rollSeq, requestID)
	if err != nil {
		return err
	}

	if res.NeedsGate {
		if err := deps.ExecuteCoreCommand(ctx, CoreCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandids.SessionGateOpen,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       requestID,
			InvocationID:    invocationID,
			EntityType:      "session_gate",
			EntityID:        res.GateID,
			PayloadJSON:     res.GatePayloadJSON,
			MissingEventMsg: "session gate open did not emit an event",
			ApplyErrMessage: "apply session gate event",
		}); err != nil {
			return err
		}
	}

	if res.NeedsSpotlight {
		if err := deps.ExecuteCoreCommand(ctx, CoreCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandids.SessionSpotlightSet,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       requestID,
			InvocationID:    invocationID,
			EntityType:      "session_spotlight",
			EntityID:        sessionID,
			PayloadJSON:     res.SpotlightPayloadJSON,
			MissingEventMsg: "session spotlight set did not emit an event",
			ApplyErrMessage: "apply spotlight event",
		}); err != nil {
			return err
		}
	}

	return nil
}
