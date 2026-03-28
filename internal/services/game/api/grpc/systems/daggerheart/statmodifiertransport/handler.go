package statmodifiertransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns the Daggerheart stat modifier mutation transport endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a Daggerheart stat modifier transport handler from explicit
// read-store and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

// ApplyStatModifiers adds and/or removes stat modifiers on a character.
func (h *Handler) ApplyStatModifiers(ctx context.Context, in *pb.DaggerheartApplyStatModifiersRequest) (StatModifiersResult, error) {
	if in == nil {
		return StatModifiersResult{}, status.Error(codes.InvalidArgument, "apply stat modifiers request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return StatModifiersResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return StatModifiersResult{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return StatModifiersResult{}, err
	}

	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return StatModifiersResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return StatModifiersResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart stat modifiers"); err != nil {
		return StatModifiersResult{}, err
	}

	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return StatModifiersResult{}, err
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return StatModifiersResult{}, err
	}

	addViews, err := StatModifiersFromProto(in.GetAddModifiers())
	if err != nil {
		return StatModifiersResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	removeIDs, err := normalizeRemovalIDs(in.GetRemoveModifierIds())
	if err != nil {
		return StatModifiersResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(addViews) == 0 && len(removeIDs) == 0 {
		return StatModifiersResult{}, status.Error(codes.InvalidArgument, "add_modifiers or remove_modifier_ids are required")
	}

	normalizedAdd := StatModifierViewsToDomain(addViews)
	removeSet := make(map[string]struct{}, len(removeIDs))
	for _, id := range removeIDs {
		removeSet[id] = struct{}{}
	}
	for _, m := range normalizedAdd {
		if _, ok := removeSet[m.ID]; ok {
			return StatModifiersResult{}, status.Error(codes.InvalidArgument, "modifiers cannot be both added and removed")
		}
	}

	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return StatModifiersResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}

	before := ProjectionStatModifiersToDomain(state.StatModifiers)

	// Compute after = (current - removed) + added.
	afterMap := make(map[string]rules.StatModifierState, len(before)+len(normalizedAdd))
	for _, m := range before {
		afterMap[m.ID] = m
	}
	for _, id := range removeIDs {
		delete(afterMap, id)
	}
	for _, m := range normalizedAdd {
		afterMap[m.ID] = m
	}
	after := make([]rules.StatModifierState, 0, len(afterMap))
	for _, m := range afterMap {
		after = append(after, m)
	}
	after, err = rules.NormalizeStatModifiers(after)
	if err != nil {
		return StatModifiersResult{}, status.Error(codes.InvalidArgument, err.Error())
	}

	if rules.StatModifiersEqual(before, after) {
		return StatModifiersResult{}, status.Error(codes.FailedPrecondition, "no stat modifier changes to apply")
	}

	added, removed := rules.DiffStatModifiers(before, after)

	source := strings.TrimSpace(in.GetSource())
	payloadJSON, _ := json.Marshal(daggerheartpayload.StatModifierChangePayload{
		CharacterID:     ids.CharacterID(characterID),
		ModifiersBefore: before,
		ModifiersAfter:  after,
		Added:           added,
		Removed:         removed,
		Source:          source,
	})

	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartStatModifierChange,
		SessionID:       sessionID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "stat modifier change did not emit an event",
		ApplyErrMessage: "apply stat modifier event",
	}); err != nil {
		return StatModifiersResult{}, err
	}

	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return StatModifiersResult{}, grpcerror.Internal("load daggerheart state", err)
	}

	return StatModifiersResult{
		CharacterID:     characterID,
		ActiveModifiers: ProjectionStatModifiersToViews(updated.StatModifiers),
		Added:           DomainStatModifiersToViews(added),
		Removed:         DomainStatModifiersToViews(removed),
	}, nil
}

func normalizeRemovalIDs(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			return nil, status.Error(codes.InvalidArgument, "remove_modifier_ids cannot include empty values")
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result, nil
}

func (h *Handler) requireDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.ExecuteDomainCommand == nil:
		return status.Error(codes.Internal, "domain command executor is not configured")
	default:
		return nil
	}
}
