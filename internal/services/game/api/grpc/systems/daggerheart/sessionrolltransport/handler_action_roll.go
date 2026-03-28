package sessionrolltransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) SessionActionRoll(ctx context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session action roll request is required")
	}
	if err := h.requireActionRollDependencies(); err != nil {
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
	sceneID := strings.TrimSpace(in.GetSceneId())
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	trait, err := validate.RequiredID(in.GetTrait(), "trait")
	if err != nil {
		return nil, err
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(c, "campaign system does not support daggerheart rolls"); err != nil {
		return nil, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return nil, err
	}

	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	rollContext := normalizeActionRollContext(in.GetContext())
	replaceHopeWithArmor := in.GetReplaceHopeWithArmor()
	baseArmorCurrent := 0
	if replaceHopeWithArmor {
		if h.deps.Content == nil {
			return nil, status.Error(codes.Internal, "content store is not configured")
		}
		if h.deps.ExecuteArmorBackedHopeSpend == nil {
			return nil, status.Error(codes.Internal, "armor-backed hope spend executor is not configured")
		}
		if strings.TrimSpace(profile.EquippedArmorID) == "" {
			return nil, status.Error(codes.FailedPrecondition, "replace_hope_with_armor requires equipped armor")
		}
		armor, err := h.deps.Content.GetDaggerheartArmor(ctx, profile.EquippedArmorID)
		if err != nil {
			return nil, grpcerror.HandleDomainErrorContext(ctx, err)
		}
		armorRules := rules.EffectiveArmorRules(&armor)
		if !armorRules.HopefulReplaceHopeWithArmor {
			return nil, status.Error(codes.FailedPrecondition, "equipped armor cannot replace hope with armor")
		}
		baseArmorCurrent = rules.CurrentBaseArmor(state, profile.ArmorMax)
	}

	modifierTotal, modifierList := normalizeActionModifiers(in.GetModifiers())
	for _, mod := range state.StatModifiers {
		if strings.EqualFold(mod.Target, trait) {
			modifierTotal, modifierList = appendActionModifier(
				modifierTotal,
				modifierList,
				"stat_modifier:"+mod.ID,
				mod.Delta,
			)
		}
	}
	if strings.EqualFold(trait, "spellcast") {
		modifierTotal, modifierList = appendActionModifier(
			modifierTotal,
			modifierList,
			"armor_spellcast",
			profile.SpellcastRollBonus,
		)
	}
	if rollContext == pb.ActionRollContext_ACTION_ROLL_CONTEXT_MOVE_SILENTLY {
		if h.deps.Content == nil {
			return nil, status.Error(codes.Internal, "content store is not configured")
		}
		if equippedArmorID := strings.TrimSpace(profile.EquippedArmorID); equippedArmorID != "" {
			armor, err := h.deps.Content.GetDaggerheartArmor(ctx, equippedArmorID)
			if err != nil {
				return nil, grpcerror.HandleDomainErrorContext(ctx, err)
			}
			armorRules := rules.EffectiveArmorRules(&armor)
			modifierTotal, modifierList = appendActionModifier(
				modifierTotal,
				modifierList,
				"armor_quiet",
				armorRules.SilentMovementBonus,
			)
		}
	}
	rollKind := normalizeRollKind(in.GetRollKind())
	advantage := int(in.GetAdvantage())
	disadvantage := int(in.GetDisadvantage())
	if advantage < 0 {
		advantage = 0
	}
	if disadvantage < 0 {
		disadvantage = 0
	}
	if in.GetUnderwater() && rollKind == pb.RollKind_ROLL_KIND_ACTION {
		disadvantage++
	}
	hopeSpends, err := normalizeHopeSpends(in.GetHopeSpends())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	spendEventCount := 0
	totalSpend := 0
	for _, spend := range hopeSpends {
		if spend.Amount > 0 {
			spendEventCount++
			totalSpend += spend.Amount
		}
	}
	if rollKind == pb.RollKind_ROLL_KIND_REACTION && spendEventCount > 0 {
		return nil, status.Error(codes.InvalidArgument, "reaction rolls cannot spend hope")
	}

	latestSeq, err := h.deps.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("load latest event seq", err)
	}
	rollSeq := latestSeq + uint64(spendEventCount) + 1

	seed, seedSource, rollMode, err := random.ResolveSeed(
		in.GetRng(),
		h.deps.SeedFunc,
		func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
	)
	if err != nil {
		if errors.Is(err, random.ErrSeedOutOfRange()) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcerror.Internal("failed to resolve seed", err)
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	if rollKind == pb.RollKind_ROLL_KIND_ACTION && spendEventCount > 0 {
		hopeBefore := state.Hope
		hopeAfter := hopeBefore
		if !replaceHopeWithArmor && hopeBefore < totalSpend {
			return nil, status.Error(codes.FailedPrecondition, "insufficient hope")
		}

		for _, spend := range hopeSpends {
			if spend.Amount <= 0 {
				continue
			}
			if replaceHopeWithArmor {
				if spend.Amount != 1 {
					return nil, status.Error(codes.FailedPrecondition, "replace_hope_with_armor only supports single-hope spends")
				}
				if baseArmorCurrent <= 0 {
					return nil, status.Error(codes.FailedPrecondition, "insufficient equipped armor")
				}
				beforeArmor, afterArmor, ok := rules.ArmorTotalAfterBaseSpend(state, profile.ArmorMax)
				if !ok {
					return nil, status.Error(codes.FailedPrecondition, "insufficient equipped armor")
				}
				baseArmorCurrent--
				if err := h.deps.ExecuteArmorBackedHopeSpend(ctx, ArmorBackedHopeSpendInput{
					CampaignID:   campaignID,
					SessionID:    sessionID,
					SceneID:      sceneID,
					RequestID:    requestID,
					InvocationID: invocationID,
					CharacterID:  characterID,
					Source:       spend.Source,
					ArmorBefore:  beforeArmor,
					ArmorAfter:   afterArmor,
				}); err != nil {
					return nil, err
				}
				continue
			}
			before := hopeAfter
			after := before - spend.Amount
			if err := h.deps.ExecuteHopeSpend(ctx, HopeSpendInput{
				CampaignID:   campaignID,
				SessionID:    sessionID,
				SceneID:      sceneID,
				RequestID:    requestID,
				InvocationID: invocationID,
				CharacterID:  characterID,
				Source:       spend.Source,
				Amount:       spend.Amount,
				HopeBefore:   before,
				HopeAfter:    after,
				RollSeq:      rollSeq,
			}); err != nil {
				return nil, err
			}
			hopeAfter = after
		}
	}

	difficulty := int(in.GetDifficulty())
	result, generateHopeFear, triggerGMMove, critNegatesEffects, err := resolveRoll(
		rollKind,
		daggerheartdomain.ActionRequest{
			Modifier:     modifierTotal,
			Difficulty:   &difficulty,
			Seed:         seed,
			Advantage:    advantage,
			Disadvantage: disadvantage,
		},
	)
	if err != nil {
		if errors.Is(err, daggerheartdomain.ErrInvalidDifficulty) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcerror.Internal("failed to roll action", err)
	}

	outcomeCode := outcomeToProto(result.Outcome).String()
	flavor := workflowtransport.OutcomeFlavorFromCode(outcomeCode)
	if !generateHopeFear {
		flavor = ""
	}

	results := map[string]any{
		"rng": map[string]any{
			"seed_used":   uint64(seed),
			"rng_algo":    random.RngAlgoMathRandV1,
			"seed_source": seedSource,
			"roll_mode":   rollMode.String(),
		},
		"dice": map[string]any{
			"hope_die":      result.Hope,
			"fear_die":      result.Fear,
			"advantage_die": result.AdvantageDie,
		},
		workflowtransport.KeyModifier: result.Modifier,
		"advantage_modifier":          result.AdvantageModifier,
		workflowtransport.KeyTotal:    result.Total,
		"difficulty":                  difficulty,
		"success":                     result.MeetsDifficulty,
		workflowtransport.KeyCrit:     result.IsCrit,
	}
	if len(modifierList) > 0 {
		results["modifiers"] = modifierList
	}

	systemMetadata := workflowtransport.RollSystemMetadata{
		CharacterID:  characterID,
		Trait:        trait,
		RollKind:     rollKind.String(),
		RollContext:  actionRollContextCode(rollContext),
		Outcome:      outcomeCode,
		Flavor:       flavor,
		HopeFear:     workflowtransport.BoolPtr(generateHopeFear),
		Crit:         workflowtransport.BoolPtr(result.IsCrit),
		CritNegates:  workflowtransport.BoolPtr(critNegatesEffects),
		GMMove:       workflowtransport.BoolPtr(triggerGMMove),
		Advantage:    workflowtransport.IntPtr(advantage),
		Disadvantage: workflowtransport.IntPtr(disadvantage),
		Underwater:   workflowtransport.BoolPtr(in.GetUnderwater()),
		Modifiers:    modifierList,
	}
	if countdownID := strings.TrimSpace(in.GetBreathSceneCountdownId()); countdownID != "" {
		systemMetadata.BreathCountdownID = countdownID
	}

	payloadJSON, err := json.Marshal(action.RollResolvePayload{
		RequestID:  requestID,
		RollSeq:    rollSeq,
		Results:    results,
		Outcome:    outcomeCode,
		SystemData: systemMetadata.MapValue(),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}

	rollSeqValue, err := h.deps.ExecuteActionRollResolve(ctx, RollResolveInput{
		CampaignID:      campaignID,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       requestID,
		InvocationID:    invocationID,
		EntityType:      "roll",
		EntityID:        requestID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "action roll did not emit an event",
	})
	if err != nil {
		return nil, err
	}

	failed := result.Difficulty != nil && !result.MeetsDifficulty
	if err := h.deps.AdvanceBreathCountdown(ctx, campaignID, sessionID, strings.TrimSpace(in.GetBreathSceneCountdownId()), failed); err != nil {
		return nil, err
	}

	return &pb.SessionActionRollResponse{
		RollSeq:    rollSeqValue,
		HopeDie:    int32(result.Hope),
		FearDie:    int32(result.Fear),
		Total:      int32(result.Total),
		Difficulty: int32(difficulty),
		Success:    result.MeetsDifficulty,
		Flavor:     flavor,
		Crit:       result.IsCrit,
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}, nil
}
