package damagetransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) ApplyAdversaryDamage(ctx context.Context, in *pb.DaggerheartApplyAdversaryDamageRequest) (AdversaryDamageResult, error) {
	if in == nil {
		return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "apply adversary damage request is required")
	}
	if err := h.requireAdversaryDamageDependencies(); err != nil {
		return AdversaryDamageResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return AdversaryDamageResult{}, err
	}
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return AdversaryDamageResult{}, err
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return AdversaryDamageResult{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return AdversaryDamageResult{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(c, "campaign system does not support daggerheart damage"); err != nil {
		return AdversaryDamageResult{}, err
	}

	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return AdversaryDamageResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return AdversaryDamageResult{}, err
	}

	if in.Damage == nil {
		return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "damage is required")
	}
	if in.Damage.Amount < 0 {
		return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "damage amount must be non-negative")
	}
	if in.Damage.DamageType == pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "damage_type is required")
	}

	adversary, err := h.deps.LoadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
	if err != nil {
		return AdversaryDamageResult{}, err
	}
	entry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, adversary.AdversaryEntryID)
	if err != nil {
		return AdversaryDamageResult{}, status.Errorf(codes.FailedPrecondition, "adversary entry %q not found", adversary.AdversaryEntryID)
	}

	result, mitigated, err := ResolveAdversaryDamage(in.Damage, adversary)
	if err != nil {
		return AdversaryDamageResult{}, grpcerror.HandleDomainError(err)
	}
	if rules.AdversaryIsMinion(entry) && in.Damage.Amount > 0 {
		result.HPAfter = 0
	}

	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	requireDamageRoll := in.GetRequireDamageRoll()
	var rollSeq *uint64
	if in.RollSeq != nil {
		value := in.GetRollSeq()
		rollSeq = &value
	}
	sourceCharacterIDs := workflowtransport.NormalizeTargets(in.Damage.GetSourceCharacterIds())
	if requireDamageRoll && rollSeq == nil {
		return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "roll_seq is required when require_damage_roll is true")
	}
	if rollSeq != nil {
		rollEvent, err := h.deps.Event.GetEventBySeq(ctx, campaignID, *rollSeq)
		if err != nil {
			return AdversaryDamageResult{}, grpcerror.HandleDomainError(err)
		}
		if rollEvent.Type != action.EventTypeRollResolved {
			return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "roll_seq must reference action.roll_resolved")
		}
		var rollPayload action.RollResolvePayload
		if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
			return AdversaryDamageResult{}, grpcerror.Internal("decode damage roll payload", err)
		}
		rollMetadata, err := workflowtransport.DecodeRollSystemMetadata(rollPayload.SystemData)
		if err != nil {
			return AdversaryDamageResult{}, status.Errorf(codes.InvalidArgument, "invalid roll system_data: %v", err)
		}
		if rollMetadata.RollKindCode() != "damage_roll" {
			return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "roll_seq does not reference a damage roll")
		}
		if len(sourceCharacterIDs) > 0 {
			rollCharacterID := strings.TrimSpace(rollMetadata.CharacterID)
			if !containsString(sourceCharacterIDs, rollCharacterID) {
				return AdversaryDamageResult{}, status.Error(codes.InvalidArgument, "roll_seq does not match source character")
			}
		}
	}

	payloadJSON, err := json.Marshal(daggerheartpayload.AdversaryDamageApplyPayload{
		AdversaryID:        dhids.AdversaryID(adversaryID),
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           DamageSeverityString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         DamageTypeString(in.Damage.DamageType),
		RollSeq:            rollSeq,
		ResistPhysical:     in.Damage.ResistPhysical,
		ResistMagic:        in.Damage.ResistMagic,
		ImmunePhysical:     in.Damage.ImmunePhysical,
		ImmuneMagic:        in.Damage.ImmuneMagic,
		Direct:             in.Damage.Direct,
		MassiveDamage:      in.Damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             in.Damage.Source,
		SourceCharacterIDs: stringsToCharacterIDs(sourceCharacterIDs),
	})
	if err != nil {
		return AdversaryDamageResult{}, grpcerror.Internal("encode payload", err)
	}

	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartAdversaryDamageApply,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "adversary",
		EntityID:        adversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary damage did not emit an event",
		ApplyErrMessage: "apply adversary damage event",
	}); err != nil {
		return AdversaryDamageResult{}, err
	}
	if rules.AdversaryIsMinion(entry) && in.Damage.Amount > 0 {
		if err := h.deleteAdversary(ctx, campaignID, sessionID, sceneID, adversary.AdversaryID, "minion_defeated"); err != nil {
			return AdversaryDamageResult{}, err
		}
		if err := h.applyMinionSpillover(ctx, campaignID, sessionID, sceneID, adversary, int(in.Damage.Amount)); err != nil {
			return AdversaryDamageResult{}, err
		}
	}
	return AdversaryDamageResult{
		AdversaryID: adversaryID,
		Adversary:   applyAdversaryDamageResult(adversary, result),
	}, nil
}

func (h *Handler) requireAdversaryDamageDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Content == nil:
		return status.Error(codes.Internal, "daggerheart content store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.ExecuteSystemCommand == nil:
		return status.Error(codes.Internal, "system command executor is not configured")
	case h.deps.LoadAdversaryForSession == nil:
		return status.Error(codes.Internal, "adversary loader is not configured")
	default:
		return nil
	}
}

func (h *Handler) applyMinionSpillover(ctx context.Context, campaignID, sessionID, sceneID string, primary projectionstore.DaggerheartAdversary, damageAmount int) error {
	primaryEntry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, primary.AdversaryEntryID)
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "adversary entry %q not found", primary.AdversaryEntryID)
	}
	defeats := rules.AdversaryMinionSpilloverDefeats(primaryEntry, damageAmount)
	if defeats <= 0 {
		return nil
	}
	adversaries, err := h.deps.Daggerheart.ListDaggerheartAdversaries(ctx, campaignID, sessionID)
	if err != nil {
		return grpcerror.Internal("list adversaries", err)
	}
	remaining := defeats
	for _, candidate := range adversaries {
		if remaining == 0 {
			break
		}
		if candidate.AdversaryID == primary.AdversaryID || candidate.SceneID != primary.SceneID {
			continue
		}
		entry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, candidate.AdversaryEntryID)
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "adversary entry %q not found", candidate.AdversaryEntryID)
		}
		if !rules.AdversaryIsMinion(entry) {
			continue
		}
		if err := h.deleteAdversary(ctx, campaignID, sessionID, sceneID, candidate.AdversaryID, "minion_defeated"); err != nil {
			return err
		}
		remaining--
	}
	return nil
}

func (h *Handler) deleteAdversary(ctx context.Context, campaignID, sessionID, sceneID, adversaryID, reason string) error {
	payloadJSON, err := json.Marshal(daggerheartpayload.AdversaryDeletePayload{
		AdversaryID: dhids.AdversaryID(adversaryID),
		Reason:      reason,
	})
	if err != nil {
		return grpcerror.Internal("encode adversary delete payload", err)
	}
	return h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
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
		ApplyErrMessage: "apply adversary delete event",
	})
}
