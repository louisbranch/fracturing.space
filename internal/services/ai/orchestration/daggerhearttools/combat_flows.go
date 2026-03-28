package daggerhearttools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"google.golang.org/protobuf/proto"
)

type attackDamageSpecInput struct {
	Amount             int      `json:"amount,omitempty"`
	DamageType         string   `json:"damage_type"`
	ResistPhysical     bool     `json:"resist_physical,omitempty"`
	ResistMagic        bool     `json:"resist_magic,omitempty"`
	ImmunePhysical     bool     `json:"immune_physical,omitempty"`
	ImmuneMagic        bool     `json:"immune_magic,omitempty"`
	Direct             bool     `json:"direct,omitempty"`
	MassiveDamage      bool     `json:"massive_damage,omitempty"`
	Source             string   `json:"source,omitempty"`
	SourceCharacterIDs []string `json:"source_character_ids,omitempty"`
}

type standardAttackProfileInput struct {
	Trait          string         `json:"trait"`
	DamageDice     []rollDiceSpec `json:"damage_dice"`
	DamageModifier int            `json:"damage_modifier,omitempty"`
	AttackRange    string         `json:"attack_range"`
	DamageCritical bool           `json:"damage_critical,omitempty"`
}

type beastformAttackProfileInput struct{}

type damageArmorReactionInput struct {
	Resilient    *timeslowingArmorReactionInput `json:"resilient,omitempty"`
	Impenetrable *struct{}                      `json:"impenetrable,omitempty"`
}

type damageMitigationDecisionInput struct {
	BaseArmor string                    `json:"base_armor,omitempty"`
	Reaction  *damageArmorReactionInput `json:"reaction,omitempty"`
}

type attackFlowResolveInput struct {
	CharacterID              string                         `json:"character_id"`
	CheckpointID             string                         `json:"checkpoint_id,omitempty"`
	Difficulty               *int                           `json:"difficulty,omitempty"`
	Modifiers                []actionRollModifier           `json:"modifiers,omitempty"`
	HopeSpends               []actionRollHopeSpend          `json:"hope_spends,omitempty"`
	Underwater               bool                           `json:"underwater,omitempty"`
	BreathSceneCountdownID   string                         `json:"breath_scene_countdown_id,omitempty"`
	TargetID                 string                         `json:"target_id"`
	Damage                   *attackDamageSpecInput         `json:"damage"`
	RequireDamageRoll        *bool                          `json:"require_damage_roll,omitempty"`
	ActionRng                *rngRequest                    `json:"action_rng,omitempty"`
	DamageRng                *rngRequest                    `json:"damage_rng,omitempty"`
	SceneID                  string                         `json:"scene_id,omitempty"`
	ReplaceHopeWithArmor     bool                           `json:"replace_hope_with_armor,omitempty"`
	TargetIsAdversary        bool                           `json:"target_is_adversary,omitempty"`
	NearbyAdversaryIDs       []string                       `json:"nearby_adversary_ids,omitempty"`
	TargetMitigationDecision *damageMitigationDecisionInput `json:"target_mitigation_decision,omitempty"`
	StandardAttack           *standardAttackProfileInput    `json:"standard_attack,omitempty"`
	BeastformAttack          *beastformAttackProfileInput   `json:"beastform_attack,omitempty"`
}

type timeslowingArmorReactionInput struct {
	Rng *rngRequest `json:"rng,omitempty"`
}

type incomingAttackArmorReactionInput struct {
	Shifting    *struct{}                      `json:"shifting,omitempty"`
	Timeslowing *timeslowingArmorReactionInput `json:"timeslowing,omitempty"`
}

type adversaryAttackFlowResolveInput struct {
	AdversaryID              string                              `json:"adversary_id"`
	CheckpointID             string                              `json:"checkpoint_id,omitempty"`
	TargetID                 string                              `json:"target_id,omitempty"`
	TargetIDs                []string                            `json:"target_ids,omitempty"`
	Difficulty               *int                                `json:"difficulty,omitempty"`
	Advantage                int                                 `json:"advantage,omitempty"`
	Disadvantage             int                                 `json:"disadvantage,omitempty"`
	Damage                   *attackDamageSpecInput              `json:"damage"`
	RequireDamageRoll        *bool                               `json:"require_damage_roll,omitempty"`
	DamageCritical           bool                                `json:"damage_critical,omitempty"`
	AttackRng                *rngRequest                         `json:"attack_rng,omitempty"`
	DamageRng                *rngRequest                         `json:"damage_rng,omitempty"`
	SceneID                  string                              `json:"scene_id,omitempty"`
	TargetArmorReaction      *incomingAttackArmorReactionInput   `json:"target_armor_reaction,omitempty"`
	TargetDefenseDecision    *incomingAttackDefenseDecisionInput `json:"target_defense_decision,omitempty"`
	TargetMitigationDecision *damageMitigationDecisionInput      `json:"target_mitigation_decision,omitempty"`
	FeatureID                string                              `json:"feature_id,omitempty"`
	ContributorAdversaryIDs  []string                            `json:"contributor_adversary_ids,omitempty"`
}

type incomingAttackDefenseDecisionInput struct {
	DeclineArmorReaction bool                              `json:"decline_armor_reaction,omitempty"`
	ArmorReaction        *incomingAttackArmorReactionInput `json:"armor_reaction,omitempty"`
}

type incomingDamageResolveInput struct {
	CharacterID        string                         `json:"character_id"`
	CheckpointID       string                         `json:"checkpoint_id,omitempty"`
	Damage             *attackDamageSpecInput         `json:"damage"`
	RollSeq            *uint64                        `json:"roll_seq,omitempty"`
	RequireDamageRoll  *bool                          `json:"require_damage_roll,omitempty"`
	SceneID            string                         `json:"scene_id,omitempty"`
	MitigationDecision *damageMitigationDecisionInput `json:"mitigation_decision,omitempty"`
}

type groupActionSupporterInput struct {
	CharacterID string               `json:"character_id"`
	Trait       string               `json:"trait"`
	Modifiers   []actionRollModifier `json:"modifiers,omitempty"`
	Rng         *rngRequest          `json:"rng,omitempty"`
	Context     string               `json:"context,omitempty"`
}

type reactionFlowResolveInput struct {
	CharacterID          string               `json:"character_id"`
	Trait                string               `json:"trait"`
	Difficulty           int                  `json:"difficulty"`
	Modifiers            []actionRollModifier `json:"modifiers,omitempty"`
	ReactionRng          *rngRequest          `json:"reaction_rng,omitempty"`
	Advantage            int                  `json:"advantage,omitempty"`
	Disadvantage         int                  `json:"disadvantage,omitempty"`
	SceneID              string               `json:"scene_id,omitempty"`
	ReplaceHopeWithArmor bool                 `json:"replace_hope_with_armor,omitempty"`
}

type groupActionFlowResolveInput struct {
	LeaderCharacterID string                      `json:"leader_character_id"`
	LeaderTrait       string                      `json:"leader_trait"`
	Difficulty        int                         `json:"difficulty"`
	LeaderModifiers   []actionRollModifier        `json:"leader_modifiers,omitempty"`
	LeaderHopeSpends  []actionRollHopeSpend       `json:"leader_hope_spends,omitempty"`
	Supporters        []groupActionSupporterInput `json:"supporters"`
	LeaderRng         *rngRequest                 `json:"leader_rng,omitempty"`
	SceneID           string                      `json:"scene_id,omitempty"`
	LeaderContext     string                      `json:"leader_context,omitempty"`
}

type tagTeamParticipantInput struct {
	CharacterID string                `json:"character_id"`
	Trait       string                `json:"trait"`
	Modifiers   []actionRollModifier  `json:"modifiers,omitempty"`
	HopeSpends  []actionRollHopeSpend `json:"hope_spends,omitempty"`
	Rng         *rngRequest           `json:"rng,omitempty"`
}

type tagTeamFlowResolveInput struct {
	First               *tagTeamParticipantInput `json:"first"`
	Second              *tagTeamParticipantInput `json:"second"`
	Difficulty          int                      `json:"difficulty"`
	SelectedCharacterID string                   `json:"selected_character_id"`
	SceneID             string                   `json:"scene_id,omitempty"`
}

type attackOutcomeResultSummary struct {
	Outcome string `json:"outcome,omitempty"`
	Success bool   `json:"success"`
	Crit    bool   `json:"crit"`
	Flavor  string `json:"flavor,omitempty"`
}

type attackOutcomeSummary struct {
	RollSeq     uint64                      `json:"roll_seq"`
	CharacterID string                      `json:"character_id,omitempty"`
	Targets     []string                    `json:"targets,omitempty"`
	Result      *attackOutcomeResultSummary `json:"result,omitempty"`
}

type adversaryAttackOutcomeResultSummary struct {
	Success    bool `json:"success"`
	Crit       bool `json:"crit"`
	Roll       int  `json:"roll"`
	Total      int  `json:"total"`
	Difficulty int  `json:"difficulty"`
}

type adversaryAttackOutcomeSummary struct {
	RollSeq     uint64                               `json:"roll_seq"`
	AdversaryID string                               `json:"adversary_id,omitempty"`
	Targets     []string                             `json:"targets,omitempty"`
	Result      *adversaryAttackOutcomeResultSummary `json:"result,omitempty"`
}

type diceRollSummary struct {
	Sides   int   `json:"sides"`
	Results []int `json:"results"`
	Total   int   `json:"total"`
}

type damageRollSummary struct {
	RollSeq       uint64            `json:"roll_seq"`
	Rolls         []diceRollSummary `json:"rolls,omitempty"`
	BaseTotal     int               `json:"base_total"`
	Modifier      int               `json:"modifier"`
	CriticalBonus int               `json:"critical_bonus"`
	Total         int               `json:"total"`
	Critical      bool              `json:"critical"`
	Rng           *rngResult        `json:"rng,omitempty"`
}

type damagePreviewSummary struct {
	Severity     string `json:"severity,omitempty"`
	Marks        int    `json:"marks"`
	HPBefore     int    `json:"hp_before"`
	HPAfter      int    `json:"hp_after"`
	StressBefore int    `json:"stress_before"`
	StressAfter  int    `json:"stress_after"`
	ArmorBefore  int    `json:"armor_before"`
	ArmorAfter   int    `json:"armor_after"`
	ArmorSpent   int    `json:"armor_spent"`
}

type combatChoiceRequiredSummary struct {
	Stage                 string                `json:"stage,omitempty"`
	CharacterID           string                `json:"character_id,omitempty"`
	CheckpointID          string                `json:"checkpoint_id,omitempty"`
	OptionCodes           []string              `json:"option_codes,omitempty"`
	Reason                string                `json:"reason,omitempty"`
	DeclinePreview        *damagePreviewSummary `json:"decline_preview,omitempty"`
	SpendBaseArmorPreview *damagePreviewSummary `json:"spend_base_armor_preview,omitempty"`
}

type compactCharacterStateSummary struct {
	HP         int              `json:"hp"`
	Hope       int              `json:"hope"`
	Stress     int              `json:"stress"`
	Armor      int              `json:"armor"`
	LifeState  string           `json:"life_state,omitempty"`
	Conditions []conditionEntry `json:"conditions,omitempty"`
}

type characterDamageAppliedSummary struct {
	CharacterID string                        `json:"character_id"`
	State       *compactCharacterStateSummary `json:"state,omitempty"`
}

type adversaryDamageAppliedSummary struct {
	AdversaryID string            `json:"adversary_id"`
	Adversary   *adversarySummary `json:"adversary,omitempty"`
}

type attackFlowResolveResult struct {
	ActionRoll             *resolvedActionRollSummary     `json:"action_roll,omitempty"`
	RollOutcome            *resolvedRollOutcomeSummary    `json:"roll_outcome,omitempty"`
	AttackOutcome          *attackOutcomeSummary          `json:"attack_outcome,omitempty"`
	DamageRoll             *damageRollSummary             `json:"damage_roll,omitempty"`
	CharacterDamageApplied *characterDamageAppliedSummary `json:"character_damage_applied,omitempty"`
	AdversaryDamageApplied *adversaryDamageAppliedSummary `json:"adversary_damage_applied,omitempty"`
	ChoiceRequired         *combatChoiceRequiredSummary   `json:"choice_required,omitempty"`
}

type adversaryAttackRollSummary struct {
	RollSeq uint64     `json:"roll_seq"`
	Roll    int        `json:"roll"`
	Total   int        `json:"total"`
	Rolls   []int      `json:"rolls,omitempty"`
	Rng     *rngResult `json:"rng,omitempty"`
}

type adversaryAttackFlowResolveResult struct {
	AttackRoll         *adversaryAttackRollSummary     `json:"attack_roll,omitempty"`
	AttackOutcome      *adversaryAttackOutcomeSummary  `json:"attack_outcome,omitempty"`
	DamageRoll         *damageRollSummary              `json:"damage_roll,omitempty"`
	DamageApplied      *characterDamageAppliedSummary  `json:"damage_applied,omitempty"`
	DamageApplications []characterDamageAppliedSummary `json:"damage_applications,omitempty"`
	ChoiceRequired     *combatChoiceRequiredSummary    `json:"choice_required,omitempty"`
}

type incomingDamageResolveResult struct {
	CharacterDamageApplied *characterDamageAppliedSummary `json:"character_damage_applied,omitempty"`
	ChoiceRequired         *combatChoiceRequiredSummary   `json:"choice_required,omitempty"`
}

type groupActionSupporterRollSummary struct {
	CharacterID string                     `json:"character_id"`
	ActionRoll  *resolvedActionRollSummary `json:"action_roll,omitempty"`
	Success     bool                       `json:"success"`
}

type groupActionFlowResolveResult struct {
	LeaderRoll       *resolvedActionRollSummary        `json:"leader_roll,omitempty"`
	LeaderOutcome    *resolvedRollOutcomeSummary       `json:"leader_outcome,omitempty"`
	SupporterRolls   []groupActionSupporterRollSummary `json:"supporter_rolls,omitempty"`
	SupportModifier  int                               `json:"support_modifier"`
	SupportSuccesses int                               `json:"support_successes"`
	SupportFailures  int                               `json:"support_failures"`
}

type tagTeamFlowResolveResult struct {
	FirstRoll           *resolvedActionRollSummary  `json:"first_roll,omitempty"`
	SecondRoll          *resolvedActionRollSummary  `json:"second_roll,omitempty"`
	SelectedOutcome     *resolvedRollOutcomeSummary `json:"selected_outcome,omitempty"`
	SelectedCharacterID string                      `json:"selected_character_id,omitempty"`
	SelectedRollSeq     uint64                      `json:"selected_roll_seq"`
}

type reactionOutcomeResultSummary struct {
	Outcome            string `json:"outcome,omitempty"`
	Success            bool   `json:"success"`
	Crit               bool   `json:"crit"`
	CritNegatesEffects bool   `json:"crit_negates_effects"`
	EffectsNegated     bool   `json:"effects_negated"`
}

type reactionOutcomeSummary struct {
	RollSeq     uint64                        `json:"roll_seq"`
	CharacterID string                        `json:"character_id,omitempty"`
	Result      *reactionOutcomeResultSummary `json:"result,omitempty"`
}

type reactionFlowResolveResult struct {
	ActionRoll      *resolvedActionRollSummary  `json:"action_roll,omitempty"`
	RollOutcome     *resolvedRollOutcomeSummary `json:"roll_outcome,omitempty"`
	ReactionOutcome *reactionOutcomeSummary     `json:"reaction_outcome,omitempty"`
}

type inferredAttackProfile struct {
	Standard    *standardAttackProfileInput
	Beastform   *beastformAttackProfileInput
	Damage      *attackDamageSpecInput
	Description string
}

// AttackFlowResolve runs an authoritative Daggerheart attack flow.
func AttackFlowResolve(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input attackFlowResolveInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := runtime.ResolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	characterID := strings.TrimSpace(input.CharacterID)
	if characterID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("character_id is required")
	}
	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}
	boardPayload, err := loadDaggerheartCombatBoardPayload(runtime, callCtx, campaignID, sessionID)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	targetID, err := resolveAttackTargetID(strings.TrimSpace(input.TargetID), boardPayload)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	targetIsAdversary := input.TargetIsAdversary || boardPayload.hasAdversary(targetID)
	if strings.TrimSpace(input.CheckpointID) != "" {
		if targetIsAdversary {
			return orchestration.ToolResult{}, fmt.Errorf("checkpoint_id is only supported when the attack target is a character")
		}
		if input.TargetMitigationDecision == nil {
			return orchestration.ToolResult{}, fmt.Errorf("target_mitigation_decision is required when checkpoint_id is set")
		}
		return resumeCharacterDamageFromCheckpoint(
			runtime,
			callCtx,
			campaignID,
			sceneID,
			targetID,
			incomingDamageSpecToProto(input.Damage),
			[]string{characterID},
			input.RequireDamageRoll,
			input.TargetMitigationDecision,
			input.CheckpointID,
			func(resp *pb.DaggerheartApplyDamageResponse, choice *combatChoiceRequiredSummary) (orchestration.ToolResult, error) {
				return toolResultJSON(attackFlowResolveResult{
					CharacterDamageApplied: characterDamageAppliedSummaryFromProto(resp),
					ChoiceRequired:         choice,
				})
			},
		)
	}
	profile, err := resolveAttackProfile(runtime, callCtx, campaignID, characterID, input.StandardAttack, input.BeastformAttack)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	difficulty, err := resolveAttackDifficulty(runtime, callCtx, campaignID, targetID, targetIsAdversary, boardPayload, input.Difficulty)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	damageSpec := mergeAttackDamageSpec(input.Damage, profile.Damage)
	req := &pb.SessionAttackFlowRequest{
		CampaignId:               campaignID,
		SessionId:                sessionID,
		SceneId:                  sceneID,
		CharacterId:              characterID,
		Difficulty:               int32(difficulty),
		Modifiers:                actionRollModifiersToProto(input.Modifiers),
		HopeSpends:               actionRollHopeSpendsToProto(input.HopeSpends),
		Underwater:               input.Underwater,
		BreathSceneCountdownId:   strings.TrimSpace(input.BreathSceneCountdownID),
		TargetId:                 targetID,
		Damage:                   attackDamageSpecToProto(damageSpec),
		RequireDamageRoll:        boolDefaultTrue(input.RequireDamageRoll),
		ActionRng:                rngRequestToProto(input.ActionRng),
		DamageRng:                rngRequestToProto(input.DamageRng),
		ReplaceHopeWithArmor:     input.ReplaceHopeWithArmor,
		TargetIsAdversary:        targetIsAdversary,
		NearbyAdversaryIds:       compactStrings(input.NearbyAdversaryIDs),
		TargetMitigationDecision: damageMitigationDecisionToProto(input.TargetMitigationDecision),
		RequireDefenseChoice:     true,
	}
	if err := applyAttackProfile(req, profile.Standard, profile.Beastform); err != nil {
		return orchestration.ToolResult{}, err
	}
	resp, err := runtime.DaggerheartClient().SessionAttackFlow(callCtx, req)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("session attack flow failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("session attack flow response is missing")
	}
	return toolResultJSON(attackFlowResolveResult{
		ActionRoll:             actionRollSummaryFromProto(resp.GetActionRoll()),
		RollOutcome:            rollOutcomeSummaryFromProto(resp.GetRollOutcome()),
		AttackOutcome:          attackOutcomeSummaryFromProto(resp.GetAttackOutcome()),
		DamageRoll:             damageRollSummaryFromProto(resp.GetDamageRoll()),
		CharacterDamageApplied: characterDamageAppliedSummaryFromProto(resp.GetDamageApplied()),
		AdversaryDamageApplied: adversaryDamageAppliedSummaryFromProto(resp.GetAdversaryDamageApplied()),
		ChoiceRequired:         choiceRequiredSummaryFromProto(resp.GetChoiceRequired(), resp.GetDamageRoll()),
	})
}

// AdversaryAttackFlowResolve runs an authoritative Daggerheart adversary attack flow.
func AdversaryAttackFlowResolve(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input adversaryAttackFlowResolveInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := runtime.ResolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	adversaryID := strings.TrimSpace(input.AdversaryID)
	if adversaryID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("adversary_id is required")
	}
	targetIDs := compactStrings(append([]string{input.TargetID}, input.TargetIDs...))
	if len(targetIDs) == 0 {
		return orchestration.ToolResult{}, fmt.Errorf("target_id is required")
	}
	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}
	if strings.TrimSpace(input.CheckpointID) != "" {
		if input.TargetMitigationDecision == nil {
			return orchestration.ToolResult{}, fmt.Errorf("target_mitigation_decision is required when checkpoint_id is set")
		}
		return resumeCharacterDamageFromCheckpoint(
			runtime,
			callCtx,
			campaignID,
			sceneID,
			targetIDs[0],
			incomingDamageSpecToProto(input.Damage),
			[]string{adversaryID},
			input.RequireDamageRoll,
			input.TargetMitigationDecision,
			input.CheckpointID,
			func(resp *pb.DaggerheartApplyDamageResponse, choice *combatChoiceRequiredSummary) (orchestration.ToolResult, error) {
				return toolResultJSON(adversaryAttackFlowResolveResult{
					DamageApplied:  characterDamageAppliedSummaryFromProto(resp),
					ChoiceRequired: choice,
				})
			},
		)
	}
	difficulty, err := resolveCharacterDifficulty(runtime, callCtx, campaignID, targetIDs[0], input.Difficulty)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	targetDefenseDecision := incomingAttackDefenseDecisionToProto(input.TargetDefenseDecision)
	if targetDefenseDecision == nil && input.TargetArmorReaction != nil {
		targetDefenseDecision = &pb.DaggerheartIncomingAttackDefenseDecision{
			ArmorReaction: incomingAttackArmorReactionToProto(input.TargetArmorReaction),
		}
	}
	resp, err := runtime.DaggerheartClient().SessionAdversaryAttackFlow(callCtx, &pb.SessionAdversaryAttackFlowRequest{
		CampaignId:               campaignID,
		SessionId:                sessionID,
		SceneId:                  sceneID,
		AdversaryId:              adversaryID,
		TargetId:                 targetIDs[0],
		TargetIds:                targetIDs[1:],
		Difficulty:               int32(difficulty),
		Advantage:                int32(input.Advantage),
		Disadvantage:             int32(input.Disadvantage),
		Damage:                   attackDamageSpecToProto(input.Damage),
		RequireDamageRoll:        boolDefaultTrue(input.RequireDamageRoll),
		DamageCritical:           input.DamageCritical,
		AttackRng:                rngRequestToProto(input.AttackRng),
		DamageRng:                rngRequestToProto(input.DamageRng),
		TargetArmorReaction:      incomingAttackArmorReactionToProto(input.TargetArmorReaction),
		TargetDefenseDecision:    targetDefenseDecision,
		TargetMitigationDecision: damageMitigationDecisionToProto(input.TargetMitigationDecision),
		RequireDefenseChoice:     true,
		FeatureId:                strings.TrimSpace(input.FeatureID),
		ContributorAdversaryIds:  compactStrings(input.ContributorAdversaryIDs),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("session adversary attack flow failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("session adversary attack flow response is missing")
	}
	damageApplications := make([]characterDamageAppliedSummary, 0, len(resp.GetDamageApplications()))
	for _, application := range resp.GetDamageApplications() {
		summary := characterDamageAppliedSummaryFromProto(application)
		if summary != nil {
			damageApplications = append(damageApplications, *summary)
		}
	}
	return toolResultJSON(adversaryAttackFlowResolveResult{
		AttackRoll:         adversaryAttackRollSummaryFromProto(resp.GetAttackRoll()),
		AttackOutcome:      adversaryAttackOutcomeSummaryFromProto(resp.GetAttackOutcome()),
		DamageRoll:         damageRollSummaryFromProto(resp.GetDamageRoll()),
		DamageApplied:      characterDamageAppliedSummaryFromProto(resp.GetDamageApplied()),
		DamageApplications: damageApplications,
		ChoiceRequired:     choiceRequiredSummaryFromProto(resp.GetChoiceRequired(), resp.GetDamageRoll()),
	})
}

// IncomingDamageResolve applies incoming damage to a character while exposing
// optional mitigation choices instead of forcing the caller to stitch together
// lower-level damage mutations.
func IncomingDamageResolve(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input incomingDamageResolveInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	characterID := strings.TrimSpace(input.CharacterID)
	if characterID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("character_id is required")
	}
	if input.Damage == nil {
		return orchestration.ToolResult{}, fmt.Errorf("damage is required")
	}
	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}
	if strings.TrimSpace(input.CheckpointID) != "" {
		return resumeCharacterDamageFromCheckpoint(
			runtime,
			callCtx,
			campaignID,
			sceneID,
			characterID,
			incomingDamageSpecToProto(input.Damage),
			nil,
			input.RequireDamageRoll,
			input.MitigationDecision,
			input.CheckpointID,
			func(resp *pb.DaggerheartApplyDamageResponse, choice *combatChoiceRequiredSummary) (orchestration.ToolResult, error) {
				return toolResultJSON(incomingDamageResolveResult{
					CharacterDamageApplied: characterDamageAppliedSummaryFromProto(resp),
					ChoiceRequired:         choice,
				})
			},
		)
	}
	resp, err := runtime.DaggerheartClient().ApplyDamage(callCtx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:              campaignID,
		CharacterId:             characterID,
		SceneId:                 sceneID,
		Damage:                  incomingDamageSpecToProto(input.Damage),
		RollSeq:                 input.RollSeq,
		RequireDamageRoll:       boolDefaultTrue(input.RequireDamageRoll),
		MitigationDecision:      damageMitigationDecisionToProto(input.MitigationDecision),
		RequireMitigationChoice: true,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("apply damage failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("apply damage response is missing")
	}
	return toolResultJSON(incomingDamageResolveResult{
		CharacterDamageApplied: characterDamageAppliedSummaryFromProto(resp),
		ChoiceRequired:         choiceRequiredSummaryFromProto(resp.GetChoiceRequired(), nil),
	})
}

// GroupActionFlowResolve runs an authoritative Daggerheart group action flow.
func GroupActionFlowResolve(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input groupActionFlowResolveInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := runtime.ResolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	leaderID := strings.TrimSpace(input.LeaderCharacterID)
	if leaderID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("leader_character_id is required")
	}
	leaderTrait := strings.TrimSpace(input.LeaderTrait)
	if leaderTrait == "" {
		return orchestration.ToolResult{}, fmt.Errorf("leader_trait is required")
	}
	if len(input.Supporters) == 0 {
		return orchestration.ToolResult{}, fmt.Errorf("supporters are required")
	}
	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}
	supporters, err := groupActionSupportersToProto(input.Supporters)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	resp, err := runtime.DaggerheartClient().SessionGroupActionFlow(callCtx, &pb.SessionGroupActionFlowRequest{
		CampaignId:        campaignID,
		SessionId:         sessionID,
		SceneId:           sceneID,
		LeaderCharacterId: leaderID,
		LeaderTrait:       leaderTrait,
		Difficulty:        int32(input.Difficulty),
		LeaderModifiers:   actionRollModifiersToProto(input.LeaderModifiers),
		LeaderHopeSpends:  actionRollHopeSpendsToProto(input.LeaderHopeSpends),
		Supporters:        supporters,
		LeaderRng:         rngRequestToProto(input.LeaderRng),
		LeaderContext:     actionRollContextToProto(input.LeaderContext),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("session group action flow failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("session group action flow response is missing")
	}
	supporterRolls := make([]groupActionSupporterRollSummary, 0, len(resp.GetSupporterRolls()))
	for _, supporterRoll := range resp.GetSupporterRolls() {
		supporterRolls = append(supporterRolls, groupActionSupporterRollSummary{
			CharacterID: strings.TrimSpace(supporterRoll.GetCharacterId()),
			ActionRoll:  actionRollSummaryFromProto(supporterRoll.GetActionRoll()),
			Success:     supporterRoll.GetSuccess(),
		})
	}
	return toolResultJSON(groupActionFlowResolveResult{
		LeaderRoll:       actionRollSummaryFromProto(resp.GetLeaderRoll()),
		LeaderOutcome:    rollOutcomeSummaryFromProto(resp.GetLeaderOutcome()),
		SupporterRolls:   supporterRolls,
		SupportModifier:  int(resp.GetSupportModifier()),
		SupportSuccesses: int(resp.GetSupportSuccesses()),
		SupportFailures:  int(resp.GetSupportFailures()),
	})
}

// TagTeamFlowResolve runs an authoritative Daggerheart tag-team flow.
func TagTeamFlowResolve(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input tagTeamFlowResolveInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := runtime.ResolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	if input.First == nil {
		return orchestration.ToolResult{}, fmt.Errorf("first is required")
	}
	if input.Second == nil {
		return orchestration.ToolResult{}, fmt.Errorf("second is required")
	}
	selectedCharacterID := strings.TrimSpace(input.SelectedCharacterID)
	if selectedCharacterID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("selected_character_id is required")
	}
	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}
	first, err := tagTeamParticipantToProto("first", input.First)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	second, err := tagTeamParticipantToProto("second", input.Second)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	resp, err := runtime.DaggerheartClient().SessionTagTeamFlow(callCtx, &pb.SessionTagTeamFlowRequest{
		CampaignId:          campaignID,
		SessionId:           sessionID,
		SceneId:             sceneID,
		First:               first,
		Second:              second,
		Difficulty:          int32(input.Difficulty),
		SelectedCharacterId: selectedCharacterID,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("session tag team flow failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("session tag team flow response is missing")
	}
	return toolResultJSON(tagTeamFlowResolveResult{
		FirstRoll:           actionRollSummaryFromProto(resp.GetFirstRoll()),
		SecondRoll:          actionRollSummaryFromProto(resp.GetSecondRoll()),
		SelectedOutcome:     rollOutcomeSummaryFromProto(resp.GetSelectedOutcome()),
		SelectedCharacterID: strings.TrimSpace(resp.GetSelectedCharacterId()),
		SelectedRollSeq:     resp.GetSelectedRollSeq(),
	})
}

// ReactionFlowResolve runs an authoritative Daggerheart reaction flow.
func ReactionFlowResolve(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input reactionFlowResolveInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := runtime.ResolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	characterID := strings.TrimSpace(input.CharacterID)
	if characterID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("character_id is required")
	}
	trait := strings.TrimSpace(input.Trait)
	if trait == "" {
		return orchestration.ToolResult{}, fmt.Errorf("trait is required")
	}
	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}
	resp, err := runtime.DaggerheartClient().SessionReactionFlow(callCtx, &pb.SessionReactionFlowRequest{
		CampaignId:           campaignID,
		SessionId:            sessionID,
		SceneId:              sceneID,
		CharacterId:          characterID,
		Trait:                trait,
		Difficulty:           int32(input.Difficulty),
		Modifiers:            actionRollModifiersToProto(input.Modifiers),
		ReactionRng:          rngRequestToProto(input.ReactionRng),
		Advantage:            int32(input.Advantage),
		Disadvantage:         int32(input.Disadvantage),
		ReplaceHopeWithArmor: input.ReplaceHopeWithArmor,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("session reaction flow failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("session reaction flow response is missing")
	}
	return toolResultJSON(reactionFlowResolveResult{
		ActionRoll:      actionRollSummaryFromProto(resp.GetActionRoll()),
		RollOutcome:     rollOutcomeSummaryFromProto(resp.GetRollOutcome()),
		ReactionOutcome: reactionOutcomeSummaryFromProto(resp.GetReactionOutcome()),
	})
}

func actionRollSummaryFromProto(resp *pb.SessionActionRollResponse) *resolvedActionRollSummary {
	if resp == nil {
		return nil
	}
	return &resolvedActionRollSummary{
		RollSeq:    resp.GetRollSeq(),
		HopeDie:    int(resp.GetHopeDie()),
		FearDie:    int(resp.GetFearDie()),
		Total:      int(resp.GetTotal()),
		Difficulty: int(resp.GetDifficulty()),
		Success:    resp.GetSuccess(),
		Flavor:     strings.TrimSpace(resp.GetFlavor()),
		Crit:       resp.GetCrit(),
		Outcome:    sessionActionOutcomeLabel(resp),
		Rng:        rngResultFromProto(resp.GetRng()),
	}
}

func rollOutcomeSummaryFromProto(resp *pb.ApplyRollOutcomeResponse) *resolvedRollOutcomeSummary {
	if resp == nil {
		return nil
	}
	return &resolvedRollOutcomeSummary{
		RollSeq:              resp.GetRollSeq(),
		RequiresComplication: resp.GetRequiresComplication(),
		Updated:              resolvedOutcomeUpdateFromProto(resp.GetUpdated()),
	}
}

func attackOutcomeSummaryFromProto(resp *pb.DaggerheartApplyAttackOutcomeResponse) *attackOutcomeSummary {
	if resp == nil {
		return nil
	}
	result := &attackOutcomeSummary{
		RollSeq:     resp.GetRollSeq(),
		CharacterID: strings.TrimSpace(resp.GetCharacterId()),
		Targets:     compactStrings(resp.GetTargets()),
	}
	if attackResult := resp.GetResult(); attackResult != nil {
		result.Result = &attackOutcomeResultSummary{
			Outcome: attackResult.GetOutcome().String(),
			Success: attackResult.GetSuccess(),
			Crit:    attackResult.GetCrit(),
			Flavor:  strings.TrimSpace(attackResult.GetFlavor()),
		}
	}
	return result
}

func adversaryAttackRollSummaryFromProto(resp *pb.SessionAdversaryAttackRollResponse) *adversaryAttackRollSummary {
	if resp == nil {
		return nil
	}
	return &adversaryAttackRollSummary{
		RollSeq: resp.GetRollSeq(),
		Roll:    int(resp.GetRoll()),
		Total:   int(resp.GetTotal()),
		Rolls:   intSlice(resp.GetRolls()),
		Rng:     rngResultFromProto(resp.GetRng()),
	}
}

func adversaryAttackOutcomeSummaryFromProto(resp *pb.DaggerheartApplyAdversaryAttackOutcomeResponse) *adversaryAttackOutcomeSummary {
	if resp == nil {
		return nil
	}
	result := &adversaryAttackOutcomeSummary{
		RollSeq:     resp.GetRollSeq(),
		AdversaryID: strings.TrimSpace(resp.GetAdversaryId()),
		Targets:     compactStrings(resp.GetTargets()),
	}
	if attackResult := resp.GetResult(); attackResult != nil {
		result.Result = &adversaryAttackOutcomeResultSummary{
			Success:    attackResult.GetSuccess(),
			Crit:       attackResult.GetCrit(),
			Roll:       int(attackResult.GetRoll()),
			Total:      int(attackResult.GetTotal()),
			Difficulty: int(attackResult.GetDifficulty()),
		}
	}
	return result
}

func damageRollSummaryFromProto(resp *pb.SessionDamageRollResponse) *damageRollSummary {
	if resp == nil {
		return nil
	}
	rolls := make([]diceRollSummary, 0, len(resp.GetRolls()))
	for _, roll := range resp.GetRolls() {
		rolls = append(rolls, diceRollSummary{
			Sides:   int(roll.GetSides()),
			Results: intSlice(roll.GetResults()),
			Total:   int(roll.GetTotal()),
		})
	}
	return &damageRollSummary{
		RollSeq:       resp.GetRollSeq(),
		Rolls:         rolls,
		BaseTotal:     int(resp.GetBaseTotal()),
		Modifier:      int(resp.GetModifier()),
		CriticalBonus: int(resp.GetCriticalBonus()),
		Total:         int(resp.GetTotal()),
		Critical:      resp.GetCritical(),
		Rng:           rngResultFromProto(resp.GetRng()),
	}
}

func reactionOutcomeSummaryFromProto(resp *pb.DaggerheartApplyReactionOutcomeResponse) *reactionOutcomeSummary {
	if resp == nil {
		return nil
	}
	result := &reactionOutcomeSummary{
		RollSeq:     resp.GetRollSeq(),
		CharacterID: strings.TrimSpace(resp.GetCharacterId()),
	}
	if reactionResult := resp.GetResult(); reactionResult != nil {
		result.Result = &reactionOutcomeResultSummary{
			Outcome:            reactionResult.GetOutcome().String(),
			Success:            reactionResult.GetSuccess(),
			Crit:               reactionResult.GetCrit(),
			CritNegatesEffects: reactionResult.GetCritNegatesEffects(),
			EffectsNegated:     reactionResult.GetEffectsNegated(),
		}
	}
	return result
}

func characterDamageAppliedSummaryFromProto(resp *pb.DaggerheartApplyDamageResponse) *characterDamageAppliedSummary {
	if resp == nil {
		return nil
	}
	return &characterDamageAppliedSummary{
		CharacterID: strings.TrimSpace(resp.GetCharacterId()),
		State:       compactCharacterStateSummaryFromProto(resp.GetState()),
	}
}

func choiceRequiredSummaryFromProto(resp *pb.DaggerheartCombatChoiceRequired, damageRoll *pb.SessionDamageRollResponse) *combatChoiceRequiredSummary {
	if resp == nil {
		return nil
	}
	return &combatChoiceRequiredSummary{
		Stage:                 resp.GetStage().String(),
		CharacterID:           strings.TrimSpace(resp.GetCharacterId()),
		CheckpointID:          damageRollCheckpointID(resp.GetStage(), damageRoll),
		OptionCodes:           compactStrings(resp.GetOptionCodes()),
		Reason:                strings.TrimSpace(resp.GetReason()),
		DeclinePreview:        damagePreviewSummaryFromProto(resp.GetDeclinePreview()),
		SpendBaseArmorPreview: damagePreviewSummaryFromProto(resp.GetSpendBaseArmorPreview()),
	}
}

func damagePreviewSummaryFromProto(resp *pb.DaggerheartDamagePreview) *damagePreviewSummary {
	if resp == nil {
		return nil
	}
	return &damagePreviewSummary{
		Severity:     strings.TrimSpace(resp.GetSeverity()),
		Marks:        int(resp.GetMarks()),
		HPBefore:     int(resp.GetHpBefore()),
		HPAfter:      int(resp.GetHpAfter()),
		StressBefore: int(resp.GetStressBefore()),
		StressAfter:  int(resp.GetStressAfter()),
		ArmorBefore:  int(resp.GetArmorBefore()),
		ArmorAfter:   int(resp.GetArmorAfter()),
		ArmorSpent:   int(resp.GetArmorSpent()),
	}
}

func compactCharacterStateSummaryFromProto(state *pb.DaggerheartCharacterState) *compactCharacterStateSummary {
	if state == nil {
		return nil
	}
	return &compactCharacterStateSummary{
		HP:         int(state.GetHp()),
		Hope:       int(state.GetHope()),
		Stress:     int(state.GetStress()),
		Armor:      int(state.GetArmor()),
		LifeState:  daggerheartLifeStateToString(state.GetLifeState()),
		Conditions: conditionsFromProto(state.GetConditionStates()),
	}
}

func resumeCharacterDamageFromCheckpoint(
	runtime Runtime,
	ctx context.Context,
	campaignID string,
	sceneID string,
	characterID string,
	damage *pb.DaggerheartDamageRequest,
	sourceCharacterIDs []string,
	requireDamageRoll *bool,
	mitigationDecision *damageMitigationDecisionInput,
	checkpointID string,
	buildResult func(*pb.DaggerheartApplyDamageResponse, *combatChoiceRequiredSummary) (orchestration.ToolResult, error),
) (orchestration.ToolResult, error) {
	if damage == nil {
		return orchestration.ToolResult{}, fmt.Errorf("damage is required")
	}
	rollSeq, amount, err := parseDamageRollCheckpointID(checkpointID)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	damage = cloneDamageRequestForCheckpointResume(damage, sourceCharacterIDs, amount)
	resp, err := runtime.DaggerheartClient().ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:              campaignID,
		CharacterId:             strings.TrimSpace(characterID),
		SceneId:                 strings.TrimSpace(sceneID),
		Damage:                  damage,
		RollSeq:                 &rollSeq,
		RequireDamageRoll:       boolDefaultTrue(requireDamageRoll),
		MitigationDecision:      damageMitigationDecisionToProto(mitigationDecision),
		RequireMitigationChoice: true,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("resume damage from checkpoint failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("resume damage from checkpoint response is missing")
	}
	return buildResult(resp, choiceRequiredSummaryFromProto(resp.GetChoiceRequired(), nil))
}

func cloneDamageRequestForCheckpointResume(damage *pb.DaggerheartDamageRequest, extraSourceCharacterIDs []string, amount int32) *pb.DaggerheartDamageRequest {
	if damage == nil {
		return nil
	}
	clone := proto.Clone(damage).(*pb.DaggerheartDamageRequest)
	clone.Amount = amount
	clone.SourceCharacterIds = compactStrings(append(clone.GetSourceCharacterIds(), extraSourceCharacterIDs...))
	return clone
}

func damageRollCheckpointID(stage pb.DaggerheartCombatChoiceStage, damageRoll *pb.SessionDamageRollResponse) string {
	if stage != pb.DaggerheartCombatChoiceStage_DAGGERHEART_COMBAT_CHOICE_STAGE_DAMAGE_MITIGATION || damageRoll == nil || damageRoll.GetRollSeq() == 0 {
		return ""
	}
	return fmt.Sprintf("damage-roll:%d:%d", damageRoll.GetRollSeq(), damageRoll.GetTotal())
}

func parseDamageRollCheckpointID(checkpointID string) (uint64, int32, error) {
	value := strings.TrimSpace(checkpointID)
	if value == "" {
		return 0, 0, fmt.Errorf("checkpoint_id is required")
	}
	raw, ok := strings.CutPrefix(value, "damage-roll:")
	if !ok {
		return 0, 0, fmt.Errorf("unsupported checkpoint_id %q", checkpointID)
	}
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid checkpoint_id %q", checkpointID)
	}
	rollSeq, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || rollSeq == 0 {
		return 0, 0, fmt.Errorf("invalid checkpoint_id %q", checkpointID)
	}
	amount, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil || amount < 0 {
		return 0, 0, fmt.Errorf("invalid checkpoint_id %q", checkpointID)
	}
	return rollSeq, int32(amount), nil
}

func adversaryDamageAppliedSummaryFromProto(resp *pb.DaggerheartApplyAdversaryDamageResponse) *adversaryDamageAppliedSummary {
	if resp == nil {
		return nil
	}
	summary := &adversaryDamageAppliedSummary{
		AdversaryID: strings.TrimSpace(resp.GetAdversaryId()),
	}
	if adversary := resp.GetAdversary(); adversary != nil {
		converted := adversarySummaryFromProto(adversary)
		summary.Adversary = &converted
	}
	return summary
}

func applyAttackProfile(req *pb.SessionAttackFlowRequest, standard *standardAttackProfileInput, beastform *beastformAttackProfileInput) error {
	if req == nil {
		return fmt.Errorf("attack flow request is required")
	}
	standard, beastform = normalizeAttackProfiles(standard, beastform)
	selected := 0
	if standard != nil {
		selected++
	}
	if beastform != nil {
		selected++
	}
	switch {
	case selected == 0:
		return fmt.Errorf("no attack profile is available; read the character sheet and either use the primary weapon/default beastform attack or provide one explicit attack profile")
	case selected > 1:
		return fmt.Errorf("only one attack profile may be provided; omit standard_attack and beastform_attack to use the default inferred attack")
	}
	if beastform != nil {
		req.AttackProfile = &pb.SessionAttackFlowRequest_BeastformAttack{
			BeastformAttack: &pb.SessionBeastformAttackProfile{},
		}
		return nil
	}
	trait := strings.TrimSpace(standard.Trait)
	if trait == "" {
		return fmt.Errorf("standard_attack.trait is required")
	}
	if len(standard.DamageDice) == 0 {
		return fmt.Errorf("standard_attack.damage_dice are required")
	}
	attackRange := daggerheartAttackRangeToProto(standard.AttackRange)
	if attackRange == pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED {
		return fmt.Errorf("standard_attack.attack_range is required")
	}
	req.AttackProfile = &pb.SessionAttackFlowRequest_StandardAttack{
		StandardAttack: &pb.SessionStandardAttackProfile{
			Trait:          trait,
			DamageDice:     diceSpecsToProto(standard.DamageDice),
			DamageModifier: int32(standard.DamageModifier),
			AttackRange:    attackRange,
			DamageCritical: standard.DamageCritical,
		},
	}
	return nil
}

func normalizeAttackProfiles(standard *standardAttackProfileInput, beastform *beastformAttackProfileInput) (*standardAttackProfileInput, *beastformAttackProfileInput) {
	if standard == nil || beastform == nil {
		return standard, beastform
	}
	if standardAttackProfileIsZero(standard) {
		return nil, beastform
	}
	return standard, nil
}

func standardAttackProfileIsZero(standard *standardAttackProfileInput) bool {
	if standard == nil {
		return true
	}
	return strings.TrimSpace(standard.Trait) == "" &&
		len(standard.DamageDice) == 0 &&
		standard.DamageModifier == 0 &&
		strings.TrimSpace(standard.AttackRange) == "" &&
		!standard.DamageCritical
}

func attackDamageSpecToProto(input *attackDamageSpecInput) *pb.DaggerheartAttackDamageSpec {
	if input == nil {
		return nil
	}
	return &pb.DaggerheartAttackDamageSpec{
		DamageType:         daggerheartDamageTypeToProto(input.DamageType),
		ResistPhysical:     input.ResistPhysical,
		ResistMagic:        input.ResistMagic,
		ImmunePhysical:     input.ImmunePhysical,
		ImmuneMagic:        input.ImmuneMagic,
		Direct:             input.Direct,
		MassiveDamage:      input.MassiveDamage,
		Source:             strings.TrimSpace(input.Source),
		SourceCharacterIds: compactStrings(input.SourceCharacterIDs),
	}
}

func incomingDamageSpecToProto(input *attackDamageSpecInput) *pb.DaggerheartDamageRequest {
	if input == nil {
		return nil
	}
	return &pb.DaggerheartDamageRequest{
		Amount:             int32(input.Amount),
		DamageType:         daggerheartDamageTypeToProto(input.DamageType),
		ResistPhysical:     input.ResistPhysical,
		ResistMagic:        input.ResistMagic,
		ImmunePhysical:     input.ImmunePhysical,
		ImmuneMagic:        input.ImmuneMagic,
		Direct:             input.Direct,
		MassiveDamage:      input.MassiveDamage,
		Source:             strings.TrimSpace(input.Source),
		SourceCharacterIds: compactStrings(input.SourceCharacterIDs),
	}
}

func incomingAttackArmorReactionToProto(input *incomingAttackArmorReactionInput) *pb.DaggerheartIncomingAttackArmorReaction {
	if input == nil {
		return nil
	}
	selected := 0
	resp := &pb.DaggerheartIncomingAttackArmorReaction{}
	if input.Shifting != nil {
		selected++
		resp.Reaction = &pb.DaggerheartIncomingAttackArmorReaction_Shifting{
			Shifting: &pb.DaggerheartShiftingArmorReaction{},
		}
	}
	if input.Timeslowing != nil {
		selected++
		resp.Reaction = &pb.DaggerheartIncomingAttackArmorReaction_Timeslowing{
			Timeslowing: &pb.DaggerheartTimeslowingArmorReaction{
				Rng: rngRequestToProto(input.Timeslowing.Rng),
			},
		}
	}
	if selected != 1 {
		return nil
	}
	return resp
}

func incomingAttackDefenseDecisionToProto(input *incomingAttackDefenseDecisionInput) *pb.DaggerheartIncomingAttackDefenseDecision {
	if input == nil {
		return nil
	}
	resp := &pb.DaggerheartIncomingAttackDefenseDecision{
		DeclineArmorReaction: input.DeclineArmorReaction,
		ArmorReaction:        incomingAttackArmorReactionToProto(input.ArmorReaction),
	}
	if !resp.GetDeclineArmorReaction() && resp.GetArmorReaction() == nil {
		return nil
	}
	return resp
}

func damageArmorReactionToProto(input *damageArmorReactionInput) *pb.DaggerheartDamageArmorReaction {
	if input == nil {
		return nil
	}
	selected := 0
	resp := &pb.DaggerheartDamageArmorReaction{}
	if input.Resilient != nil {
		selected++
		resp.Reaction = &pb.DaggerheartDamageArmorReaction_Resilient{
			Resilient: &pb.DaggerheartResilientArmorReaction{
				Rng: rngRequestToProto(input.Resilient.Rng),
			},
		}
	}
	if input.Impenetrable != nil {
		selected++
		resp.Reaction = &pb.DaggerheartDamageArmorReaction_Impenetrable{
			Impenetrable: &pb.DaggerheartImpenetrableArmorReaction{},
		}
	}
	if selected != 1 {
		return nil
	}
	return resp
}

func damageMitigationDecisionToProto(input *damageMitigationDecisionInput) *pb.DaggerheartDamageMitigationDecision {
	if input == nil {
		return nil
	}
	resp := &pb.DaggerheartDamageMitigationDecision{
		ArmorReaction: damageArmorReactionToProto(input.Reaction),
	}
	switch strings.ToUpper(strings.TrimSpace(input.BaseArmor)) {
	case "SPEND":
		resp.BaseArmor = pb.DaggerheartBaseArmorDecision_DAGGERHEART_BASE_ARMOR_DECISION_SPEND
	case "DECLINE":
		resp.BaseArmor = pb.DaggerheartBaseArmorDecision_DAGGERHEART_BASE_ARMOR_DECISION_DECLINE
	default:
		resp.BaseArmor = pb.DaggerheartBaseArmorDecision_DAGGERHEART_BASE_ARMOR_DECISION_UNSPECIFIED
	}
	if resp.GetBaseArmor() == pb.DaggerheartBaseArmorDecision_DAGGERHEART_BASE_ARMOR_DECISION_UNSPECIFIED && resp.GetArmorReaction() == nil {
		return nil
	}
	return resp
}

func groupActionSupportersToProto(values []groupActionSupporterInput) ([]*pb.GroupActionSupporter, error) {
	result := make([]*pb.GroupActionSupporter, 0, len(values))
	for i, value := range values {
		characterID := strings.TrimSpace(value.CharacterID)
		if characterID == "" {
			return nil, fmt.Errorf("supporters[%d].character_id is required", i)
		}
		trait := strings.TrimSpace(value.Trait)
		if trait == "" {
			return nil, fmt.Errorf("supporters[%d].trait is required", i)
		}
		result = append(result, &pb.GroupActionSupporter{
			CharacterId: characterID,
			Trait:       trait,
			Modifiers:   actionRollModifiersToProto(value.Modifiers),
			Rng:         rngRequestToProto(value.Rng),
			Context:     actionRollContextToProto(value.Context),
		})
	}
	return result, nil
}

func tagTeamParticipantToProto(label string, input *tagTeamParticipantInput) (*pb.TagTeamParticipant, error) {
	if input == nil {
		return nil, fmt.Errorf("%s is required", label)
	}
	characterID := strings.TrimSpace(input.CharacterID)
	if characterID == "" {
		return nil, fmt.Errorf("%s.character_id is required", label)
	}
	trait := strings.TrimSpace(input.Trait)
	if trait == "" {
		return nil, fmt.Errorf("%s.trait is required", label)
	}
	return &pb.TagTeamParticipant{
		CharacterId: characterID,
		Trait:       trait,
		Modifiers:   actionRollModifiersToProto(input.Modifiers),
		HopeSpends:  actionRollHopeSpendsToProto(input.HopeSpends),
		Rng:         rngRequestToProto(input.Rng),
	}, nil
}

func diceSpecsToProto(values []rollDiceSpec) []*pb.DiceSpec {
	result := make([]*pb.DiceSpec, 0, len(values))
	for _, value := range values {
		if value.Count <= 0 || value.Sides <= 0 {
			continue
		}
		result = append(result, &pb.DiceSpec{
			Count: int32(value.Count),
			Sides: int32(value.Sides),
		})
	}
	return result
}

func resolveAttackTargetID(explicitTargetID string, board daggerheartCombatBoardPayload) (string, error) {
	if targetID := strings.TrimSpace(explicitTargetID); targetID != "" {
		return targetID, nil
	}
	switch strings.TrimSpace(board.Status) {
	case "NO_ACTIVE_SCENE":
		return "", fmt.Errorf("cannot infer target_id because the combat board has no active scene; use interaction_state_read or interaction_activate_scene, then retry")
	case "EMPTY_BOARD":
		return "", fmt.Errorf("cannot infer target_id because the combat board is empty; create or reveal a target first, then retry")
	case "NO_VISIBLE_ADVERSARY":
		return "", fmt.Errorf("cannot infer target_id because the active scene has no visible adversary; read daggerheart_combat_board_read and create or reveal the intended target")
	}
	if len(board.Adversaries) == 1 {
		return strings.TrimSpace(board.Adversaries[0].ID), nil
	}
	if len(board.Adversaries) > 1 {
		return "", fmt.Errorf("target_id is required when the combat board has multiple visible adversaries; read daggerheart_combat_board_read and specify the intended target_id")
	}
	return "", fmt.Errorf("target_id is required")
}

func resolveAttackDifficulty(runtime Runtime, ctx context.Context, campaignID, targetID string, targetIsAdversary bool, board daggerheartCombatBoardPayload, explicit *int) (int, error) {
	if explicit != nil {
		return *explicit, nil
	}
	if targetIsAdversary {
		for _, adversary := range board.Adversaries {
			if strings.TrimSpace(adversary.ID) == strings.TrimSpace(targetID) {
				return adversary.Evasion, nil
			}
		}
		return 0, fmt.Errorf("cannot infer difficulty for target %q from the combat board; read daggerheart_combat_board_read and retry", targetID)
	}
	return resolveCharacterDifficulty(runtime, ctx, campaignID, targetID, nil)
}

func resolveCharacterDifficulty(runtime Runtime, ctx context.Context, campaignID, characterID string, explicit *int) (int, error) {
	if explicit != nil {
		return *explicit, nil
	}
	sheet, err := loadCharacterSheetPayload(runtime, ctx, campaignID, characterID)
	if err != nil {
		return 0, err
	}
	if sheet.Daggerheart == nil || sheet.Daggerheart.Defenses == nil || sheet.Daggerheart.Defenses.Evasion == nil {
		return 0, fmt.Errorf("cannot infer difficulty for target %q from the current character sheet; call character_sheet_read and retry", characterID)
	}
	return *sheet.Daggerheart.Defenses.Evasion, nil
}

func resolveAttackProfile(runtime Runtime, ctx context.Context, campaignID, characterID string, explicitStandard *standardAttackProfileInput, explicitBeastform *beastformAttackProfileInput) (inferredAttackProfile, error) {
	if explicitStandard != nil || explicitBeastform != nil {
		return inferredAttackProfile{Standard: explicitStandard, Beastform: explicitBeastform}, nil
	}
	sheet, err := loadCharacterSheetPayload(runtime, ctx, campaignID, characterID)
	if err != nil {
		return inferredAttackProfile{}, err
	}
	if profile := inferredBeastformAttackProfile(sheet, characterID); profile != nil {
		return *profile, nil
	}
	if profile := inferredPrimaryWeaponAttackProfile(sheet, characterID); profile != nil {
		return *profile, nil
	}
	return inferredAttackProfile{}, fmt.Errorf("cannot infer an attack profile from the current character sheet; call character_sheet_read and provide one explicit attack profile")
}

func inferredPrimaryWeaponAttackProfile(sheet characterSheetPayload, characterID string) *inferredAttackProfile {
	if sheet.Daggerheart == nil || sheet.Daggerheart.Equipment == nil || sheet.Daggerheart.Equipment.PrimaryWeapon == nil {
		return nil
	}
	weapon := sheet.Daggerheart.Equipment.PrimaryWeapon
	damageDice, ok := parseDamageDiceString(weapon.DamageDice)
	if !ok {
		return nil
	}
	trait := strings.TrimSpace(weapon.Trait)
	attackRange := strings.TrimSpace(weapon.Range)
	if trait == "" || attackRange == "" {
		return nil
	}
	return &inferredAttackProfile{
		Standard: &standardAttackProfileInput{
			Trait:       trait,
			DamageDice:  damageDice,
			AttackRange: attackRange,
		},
		Damage: &attackDamageSpecInput{
			DamageType:         strings.TrimSpace(weapon.DamageType),
			Source:             firstNonEmpty(strings.TrimSpace(weapon.Name), "Primary weapon"),
			SourceCharacterIDs: compactStrings([]string{characterID}),
		},
		Description: "primary_weapon",
	}
}

func inferredBeastformAttackProfile(sheet characterSheetPayload, characterID string) *inferredAttackProfile {
	if sheet.Daggerheart == nil || sheet.Daggerheart.ClassState == nil || sheet.Daggerheart.ClassState.ActiveBeastform == nil {
		return nil
	}
	beastform := sheet.Daggerheart.ClassState.ActiveBeastform
	if strings.TrimSpace(beastform.AttackTrait) == "" || strings.TrimSpace(beastform.AttackRange) == "" || len(beastform.DamageDice) == 0 {
		return nil
	}
	damageDice := make([]rollDiceSpec, 0, len(beastform.DamageDice))
	for _, die := range beastform.DamageDice {
		if die.Count <= 0 || die.Sides <= 0 {
			continue
		}
		damageDice = append(damageDice, rollDiceSpec{Count: die.Count, Sides: die.Sides})
	}
	if len(damageDice) == 0 {
		return nil
	}
	return &inferredAttackProfile{
		Beastform: &beastformAttackProfileInput{},
		Damage: &attackDamageSpecInput{
			DamageType:         strings.TrimSpace(beastform.DamageType),
			Source:             firstNonEmpty(strings.TrimSpace(beastform.BeastformID), "Active beastform"),
			SourceCharacterIDs: compactStrings([]string{characterID}),
		},
		Description: "active_beastform",
	}
}

func mergeAttackDamageSpec(explicit, inferred *attackDamageSpecInput) *attackDamageSpecInput {
	switch {
	case explicit == nil && inferred == nil:
		return nil
	case explicit == nil:
		clone := *inferred
		return &clone
	case inferred == nil:
		return explicit
	}
	merged := *explicit
	if strings.TrimSpace(merged.DamageType) == "" {
		merged.DamageType = inferred.DamageType
	}
	if strings.TrimSpace(merged.Source) == "" {
		merged.Source = inferred.Source
	}
	if len(merged.SourceCharacterIDs) == 0 {
		merged.SourceCharacterIDs = append([]string(nil), inferred.SourceCharacterIDs...)
	}
	return &merged
}

var damageDiceRE = regexp.MustCompile(`(?i)(\d*)d(\d+)`)

func parseDamageDiceString(raw string) ([]rollDiceSpec, bool) {
	matches := damageDiceRE.FindAllStringSubmatch(strings.TrimSpace(raw), -1)
	if len(matches) == 0 {
		return nil, false
	}
	result := make([]rollDiceSpec, 0, len(matches))
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		count := 1
		if strings.TrimSpace(match[1]) != "" {
			parsed, err := strconv.Atoi(match[1])
			if err != nil || parsed <= 0 {
				return nil, false
			}
			count = parsed
		}
		sides, err := strconv.Atoi(match[2])
		if err != nil || sides <= 0 {
			return nil, false
		}
		result = append(result, rollDiceSpec{Count: count, Sides: sides})
	}
	return result, len(result) > 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func boolDefaultTrue(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

func (payload daggerheartCombatBoardPayload) hasAdversary(targetID string) bool {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return false
	}
	for _, adversary := range payload.Adversaries {
		if strings.TrimSpace(adversary.ID) == targetID {
			return true
		}
	}
	return false
}
