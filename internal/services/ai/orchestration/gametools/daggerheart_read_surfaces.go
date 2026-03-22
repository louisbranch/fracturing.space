package gametools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

type characterSheetReadInput struct {
	CharacterID string `json:"character_id"`
}

type characterSheetPayload struct {
	Character   characterIdentitySummary        `json:"character"`
	Daggerheart *daggerheartCharacterSheetState `json:"daggerheart,omitempty"`
}

type characterIdentitySummary struct {
	ID                 string   `json:"id"`
	CampaignID         string   `json:"campaign_id,omitempty"`
	Name               string   `json:"name,omitempty"`
	Kind               string   `json:"kind,omitempty"`
	OwnerParticipantID string   `json:"owner_participant_id,omitempty"`
	Pronouns           string   `json:"pronouns,omitempty"`
	Aliases            []string `json:"aliases,omitempty"`
	Notes              string   `json:"notes,omitempty"`
}

type daggerheartCharacterSheetState struct {
	Level                  int                            `json:"level,omitempty"`
	Class                  *contentReference              `json:"class,omitempty"`
	Subclass               *contentReference              `json:"subclass,omitempty"`
	Heritage               *heritageSummary               `json:"heritage,omitempty"`
	Traits                 *traitSummary                  `json:"traits,omitempty"`
	Experiences            []experienceSummary            `json:"experiences,omitempty"`
	Resources              *resourceSummary               `json:"resources,omitempty"`
	Defenses               *defenseSummary                `json:"defenses,omitempty"`
	Equipment              *equipmentSummary              `json:"equipment,omitempty"`
	DomainCards            []domainCardSummary            `json:"domain_cards,omitempty"`
	ActiveClassFeatures    []activeFeatureSummary         `json:"active_class_features,omitempty"`
	ActiveSubclassFeatures []activeSubclassFeatureSummary `json:"active_subclass_features,omitempty"`
	Conditions             []conditionEntry               `json:"conditions,omitempty"`
	TemporaryArmor         []temporaryArmorEntry          `json:"temporary_armor,omitempty"`
	StatModifiers          []statModifierEntry            `json:"stat_modifiers,omitempty"`
	Companion              *companionSummary              `json:"companion,omitempty"`
	ClassState             *classStateSummary             `json:"class_state,omitempty"`
	SubclassState          *subclassStateSummary          `json:"subclass_state,omitempty"`
	Background             string                         `json:"background,omitempty"`
	Connections            string                         `json:"connections,omitempty"`
	Description            string                         `json:"description,omitempty"`
}

type contentReference struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type heritageSummary struct {
	Ancestry  string `json:"ancestry,omitempty"`
	Community string `json:"community,omitempty"`
}

type traitSummary struct {
	Agility   int `json:"agility,omitempty"`
	Strength  int `json:"strength,omitempty"`
	Finesse   int `json:"finesse,omitempty"`
	Instinct  int `json:"instinct,omitempty"`
	Presence  int `json:"presence,omitempty"`
	Knowledge int `json:"knowledge,omitempty"`
}

type experienceSummary struct {
	Name     string `json:"name,omitempty"`
	Modifier int    `json:"modifier,omitempty"`
}

type resourceSummary struct {
	HP        int    `json:"hp,omitempty"`
	HPMax     int    `json:"hp_max,omitempty"`
	Hope      int    `json:"hope,omitempty"`
	HopeMax   int    `json:"hope_max,omitempty"`
	Stress    int    `json:"stress,omitempty"`
	StressMax *int   `json:"stress_max,omitempty"`
	Armor     int    `json:"armor,omitempty"`
	ArmorMax  *int   `json:"armor_max,omitempty"`
	LifeState string `json:"life_state,omitempty"`
}

type defenseSummary struct {
	Evasion            *int `json:"evasion,omitempty"`
	ArmorScore         *int `json:"armor_score,omitempty"`
	Proficiency        *int `json:"proficiency,omitempty"`
	MajorThreshold     *int `json:"major_threshold,omitempty"`
	SevereThreshold    *int `json:"severe_threshold,omitempty"`
	SpellcastRollBonus *int `json:"spellcast_roll_bonus,omitempty"`
}

type equipmentSummary struct {
	PrimaryWeapon   *weaponSummary     `json:"primary_weapon,omitempty"`
	SecondaryWeapon *weaponSummary     `json:"secondary_weapon,omitempty"`
	ActiveArmor     *armorSummary      `json:"active_armor,omitempty"`
	Consumables     []contentReference `json:"consumables,omitempty"`
}

type weaponSummary struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Trait      string `json:"trait,omitempty"`
	Range      string `json:"range,omitempty"`
	DamageDice string `json:"damage_dice,omitempty"`
	DamageType string `json:"damage_type,omitempty"`
	Feature    string `json:"feature,omitempty"`
}

type armorSummary struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	BaseScore *int   `json:"base_score,omitempty"`
	Feature   string `json:"feature,omitempty"`
}

type domainCardSummary struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Domain string `json:"domain,omitempty"`
}

type activeFeatureSummary struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Level       int    `json:"level,omitempty"`
	HopeFeature bool   `json:"hope_feature,omitempty"`
}

type activeSubclassFeatureSummary struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Level       int    `json:"level,omitempty"`
	Origin      string `json:"origin,omitempty"`
	Rank        string `json:"rank,omitempty"`
	Class       string `json:"class,omitempty"`
	Subclass    string `json:"subclass,omitempty"`
	Domain      string `json:"domain,omitempty"`
}

type companionSummary struct {
	Name              string              `json:"name,omitempty"`
	AnimalKind        string              `json:"animal_kind,omitempty"`
	Evasion           int                 `json:"evasion,omitempty"`
	AttackDescription string              `json:"attack_description,omitempty"`
	AttackRange       string              `json:"attack_range,omitempty"`
	DamageDieSides    int                 `json:"damage_die_sides,omitempty"`
	DamageType        string              `json:"damage_type,omitempty"`
	Status            string              `json:"status,omitempty"`
	ActiveExperience  string              `json:"active_experience,omitempty"`
	Experiences       []experienceSummary `json:"experiences,omitempty"`
}

type classStateSummary struct {
	AttackBonusUntilRest            int                 `json:"attack_bonus_until_rest,omitempty"`
	EvasionBonusUntilHitOrRest      int                 `json:"evasion_bonus_until_hit_or_rest,omitempty"`
	DifficultyPenaltyUntilRest      int                 `json:"difficulty_penalty_until_rest,omitempty"`
	FocusTargetID                   string              `json:"focus_target_id,omitempty"`
	StrangePatternsNumber           int                 `json:"strange_patterns_number,omitempty"`
	RallyDice                       []int               `json:"rally_dice,omitempty"`
	PrayerDice                      []int               `json:"prayer_dice,omitempty"`
	Unstoppable                     *unstoppableSummary `json:"unstoppable,omitempty"`
	ChannelRawPowerUsedThisLongRest bool                `json:"channel_raw_power_used_this_long_rest,omitempty"`
	ActiveBeastform                 *beastformSummary   `json:"active_beastform,omitempty"`
}

type unstoppableSummary struct {
	Active           bool `json:"active,omitempty"`
	CurrentValue     int  `json:"current_value,omitempty"`
	DieSides         int  `json:"die_sides,omitempty"`
	UsedThisLongRest bool `json:"used_this_long_rest,omitempty"`
}

type beastformSummary struct {
	BeastformID            string          `json:"beastform_id,omitempty"`
	BaseTrait              string          `json:"base_trait,omitempty"`
	AttackTrait            string          `json:"attack_trait,omitempty"`
	TraitBonus             int             `json:"trait_bonus,omitempty"`
	EvasionBonus           int             `json:"evasion_bonus,omitempty"`
	AttackRange            string          `json:"attack_range,omitempty"`
	DamageDice             []damageDieSpec `json:"damage_dice,omitempty"`
	DamageBonus            int             `json:"damage_bonus,omitempty"`
	DamageType             string          `json:"damage_type,omitempty"`
	EvolutionTraitOverride string          `json:"evolution_trait_override,omitempty"`
	DropOnAnyHPMark        bool            `json:"drop_on_any_hp_mark,omitempty"`
}

type damageDieSpec struct {
	Count int `json:"count,omitempty"`
	Sides int `json:"sides,omitempty"`
}

type subclassStateSummary struct {
	BattleRitualUsedThisLongRest           bool   `json:"battle_ritual_used_this_long_rest,omitempty"`
	GiftedPerformerRelaxingSongUses        int    `json:"gifted_performer_relaxing_song_uses,omitempty"`
	GiftedPerformerEpicSongUses            int    `json:"gifted_performer_epic_song_uses,omitempty"`
	GiftedPerformerHeartbreakingSongUses   int    `json:"gifted_performer_heartbreaking_song_uses,omitempty"`
	ContactsEverywhereUsesThisSession      int    `json:"contacts_everywhere_uses_this_session,omitempty"`
	ContactsEverywhereActionDieBonus       int    `json:"contacts_everywhere_action_die_bonus,omitempty"`
	ContactsEverywhereDamageDiceBonusCount int    `json:"contacts_everywhere_damage_dice_bonus_count,omitempty"`
	SparingTouchUsesThisLongRest           int    `json:"sparing_touch_uses_this_long_rest,omitempty"`
	ElementalistActionBonus                int    `json:"elementalist_action_bonus,omitempty"`
	ElementalistDamageBonus                int    `json:"elementalist_damage_bonus,omitempty"`
	TranscendenceActive                    bool   `json:"transcendence_active,omitempty"`
	TranscendenceTraitBonusTarget          string `json:"transcendence_trait_bonus_target,omitempty"`
	TranscendenceTraitBonusValue           int    `json:"transcendence_trait_bonus_value,omitempty"`
	TranscendenceProficiencyBonus          int    `json:"transcendence_proficiency_bonus,omitempty"`
	TranscendenceEvasionBonus              int    `json:"transcendence_evasion_bonus,omitempty"`
	TranscendenceSevereThresholdBonus      int    `json:"transcendence_severe_threshold_bonus,omitempty"`
	ClarityOfNatureUsedThisLongRest        bool   `json:"clarity_of_nature_used_this_long_rest,omitempty"`
	ElementalChannel                       string `json:"elemental_channel,omitempty"`
	NemesisTargetID                        string `json:"nemesis_target_id,omitempty"`
	RousingSpeechUsedThisLongRest          bool   `json:"rousing_speech_used_this_long_rest,omitempty"`
	WardensProtectionUsedThisLongRest      bool   `json:"wardens_protection_used_this_long_rest,omitempty"`
}

type daggerheartCombatBoardPayload struct {
	Status           string                       `json:"status,omitempty"`
	Issues           []combatBoardDiagnosticIssue `json:"issues,omitempty"`
	RecommendedTools []string                     `json:"recommended_tools,omitempty"`
	GmFear           int                          `json:"gm_fear,omitempty"`
	SessionID        string                       `json:"session_id,omitempty"`
	SceneID          string                       `json:"scene_id,omitempty"`
	Spotlight        *sessionSpotlightSummary     `json:"spotlight,omitempty"`
	Countdowns       []countdownSummary           `json:"countdowns,omitempty"`
	Adversaries      []adversarySummary           `json:"adversaries,omitempty"`
}

type sessionSpotlightSummary struct {
	Type        string `json:"type,omitempty"`
	CharacterID string `json:"character_id,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type combatBoardDiagnosticIssue struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type adversarySummary struct {
	ID                string                    `json:"id,omitempty"`
	Name              string                    `json:"name,omitempty"`
	Kind              string                    `json:"kind,omitempty"`
	SceneID           string                    `json:"scene_id,omitempty"`
	Notes             string                    `json:"notes,omitempty"`
	HP                int                       `json:"hp,omitempty"`
	HPMax             int                       `json:"hp_max,omitempty"`
	Stress            int                       `json:"stress,omitempty"`
	StressMax         int                       `json:"stress_max,omitempty"`
	Evasion           int                       `json:"evasion,omitempty"`
	MajorThreshold    int                       `json:"major_threshold,omitempty"`
	SevereThreshold   int                       `json:"severe_threshold,omitempty"`
	Armor             int                       `json:"armor,omitempty"`
	SpotlightGateID   string                    `json:"spotlight_gate_id,omitempty"`
	SpotlightCount    int                       `json:"spotlight_count,omitempty"`
	Conditions        []conditionEntry          `json:"conditions,omitempty"`
	Features          []adversaryFeatureSummary `json:"features,omitempty"`
	PendingExperience *experienceSummary        `json:"pending_experience,omitempty"`
}

type adversaryFeatureSummary struct {
	FeatureID       string `json:"feature_id,omitempty"`
	Status          string `json:"status,omitempty"`
	FocusedTargetID string `json:"focused_target_id,omitempty"`
}

// characterSheetRead returns one authoritative character sheet summary so the
// AI can inspect what a character currently has and can do before using
// mechanics tools or narrating a rules-specific option.
func (s *DirectSession) characterSheetRead(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input characterSheetReadInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	if strings.TrimSpace(input.CharacterID) == "" {
		return orchestration.ToolResult{}, fmt.Errorf("character_id is required")
	}
	payload, err := s.loadCharacterSheetPayload(ctx, campaignID, input.CharacterID)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	return toolResultJSON(payload)
}

// daggerheartCombatBoardRead returns the current Daggerheart combat board state
// for the bound campaign/session so the AI can reason about Fear, spotlight,
// visible countdowns, and adversaries from one authoritative read surface.
func (s *DirectSession) daggerheartCombatBoardRead(ctx context.Context, _ []byte) (orchestration.ToolResult, error) {
	campaignID := s.resolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := s.resolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	payload, err := s.loadDaggerheartCombatBoardPayload(ctx, campaignID, sessionID)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	return toolResultJSON(payload)
}

func (s *DirectSession) loadCharacterSheetPayload(ctx context.Context, campaignID, characterID string) (characterSheetPayload, error) {
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Character.GetCharacterSheet(callCtx, &statev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: strings.TrimSpace(characterID),
	})
	if err != nil {
		return characterSheetPayload{}, fmt.Errorf("get character sheet failed: %w", err)
	}
	if resp == nil || resp.GetCharacter() == nil {
		return characterSheetPayload{}, fmt.Errorf("character sheet response is missing")
	}
	return buildCharacterSheetPayload(resp), nil
}

func (s *DirectSession) loadDaggerheartCombatBoardPayload(ctx context.Context, campaignID, sessionID string) (daggerheartCombatBoardPayload, error) {
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	snapshotResp, err := s.clients.Snapshot.GetSnapshot(callCtx, &statev1.GetSnapshotRequest{CampaignId: campaignID})
	if err != nil {
		return daggerheartCombatBoardPayload{}, fmt.Errorf("get snapshot failed: %w", err)
	}
	spotlightResp, err := s.clients.Session.GetSessionSpotlight(callCtx, &statev1.GetSessionSpotlightRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		if st, ok := status.FromError(err); !ok || st.Code() != codes.NotFound {
			return daggerheartCombatBoardPayload{}, fmt.Errorf("get session spotlight failed: %w", err)
		}
		spotlightResp = &statev1.GetSessionSpotlightResponse{}
	}
	sceneID, sceneErr := s.resolveSceneID(callCtx, campaignID, "")
	adversariesResp := &pb.DaggerheartListAdversariesResponse{}
	countdownsResp := &pb.DaggerheartListSceneCountdownsResponse{}
	if sceneErr == nil && strings.TrimSpace(sceneID) != "" {
		adversariesResp, err = s.clients.Daggerheart.ListAdversaries(callCtx, &pb.DaggerheartListAdversariesRequest{
			CampaignId: campaignID,
			SessionId:  wrapperspb.String(sessionID),
		})
		if err != nil {
			return daggerheartCombatBoardPayload{}, fmt.Errorf("list adversaries failed: %w", err)
		}
		countdownsResp, err = s.clients.Daggerheart.ListSceneCountdowns(callCtx, &pb.DaggerheartListSceneCountdownsRequest{
			CampaignId: campaignID,
			SessionId:  sessionID,
			SceneId:    sceneID,
		})
		if err != nil {
			return daggerheartCombatBoardPayload{}, fmt.Errorf("list scene countdowns failed: %w", err)
		}
	}
	payload := buildDaggerheartCombatBoardPayload(sessionID, sceneID, snapshotResp, spotlightResp, countdownsResp, adversariesResp)
	return applyCombatBoardDiagnostics(payload, sceneErr), nil
}

func buildCharacterSheetPayload(resp *statev1.GetCharacterSheetResponse) characterSheetPayload {
	char := resp.GetCharacter()
	payload := characterSheetPayload{
		Character: characterIdentitySummary{
			ID:                 char.GetId(),
			CampaignID:         char.GetCampaignId(),
			Name:               char.GetName(),
			Kind:               characterKindToString(char.GetKind()),
			OwnerParticipantID: strings.TrimSpace(char.GetParticipantId().GetValue()),
			Pronouns:           sharedpronouns.FromProto(char.GetPronouns()),
			Aliases:            cloneStrings(char.GetAliases()),
			Notes:              char.GetNotes(),
		},
	}

	profile := resp.GetProfile().GetDaggerheart()
	state := resp.GetState().GetDaggerheart()
	if profile == nil && state == nil {
		return payload
	}

	daggerheart := &daggerheartCharacterSheetState{
		Level:       0,
		Class:       nil,
		Subclass:    nil,
		Heritage:    nil,
		Traits:      nil,
		Experiences: nil,
		Resources:   resourcesFromProto(profile, state),
		Defenses:    defensesFromProto(profile),
		Equipment:   equipmentFromProto(profile),
		DomainCards: nil,
		Background:  "",
		Connections: "",
		Description: "",
	}
	if profile != nil {
		daggerheart.Level = int(profile.GetLevel())
		daggerheart.Class = contentRefFromID(profile.GetClassId())
		daggerheart.Subclass = contentRefFromID(profile.GetSubclassId())
		daggerheart.Heritage = heritageFromProto(profile.GetHeritage())
		daggerheart.Traits = traitsFromProto(profile)
		daggerheart.Experiences = experiencesFromProto(profile.GetExperiences())
		daggerheart.DomainCards = domainCardsFromProto(profile.GetDomainCardIds())
		daggerheart.Background = strings.TrimSpace(profile.GetBackground())
		daggerheart.Connections = strings.TrimSpace(profile.GetConnections())
		daggerheart.Description = strings.TrimSpace(profile.GetDescription())
		daggerheart.ActiveClassFeatures = activeClassFeaturesFromProto(profile.GetActiveClassFeatures())
		daggerheart.ActiveSubclassFeatures = activeSubclassFeaturesFromProto(profile.GetActiveSubclassFeatures())
	}
	if state != nil {
		daggerheart.Conditions = conditionsFromProto(state.GetConditionStates())
		daggerheart.TemporaryArmor = temporaryArmorFromProto(state.GetTemporaryArmorBuckets())
		daggerheart.StatModifiers = statModifiersFromProto(state.GetStatModifiers())
		daggerheart.ClassState = classStateFromProto(state.GetClassState())
		daggerheart.SubclassState = subclassStateFromProto(state.GetSubclassState())
		if companion := companionFromProto(profile.GetCompanionSheet(), state.GetCompanionState()); companion != nil {
			daggerheart.Companion = companion
		}
	} else if companion := companionFromProto(profile.GetCompanionSheet(), nil); companion != nil {
		daggerheart.Companion = companion
	}

	payload.Daggerheart = daggerheart
	return payload
}

func buildDaggerheartCombatBoardPayload(sessionID, sceneID string, snapshotResp *statev1.GetSnapshotResponse, spotlightResp *statev1.GetSessionSpotlightResponse, countdownsResp *pb.DaggerheartListSceneCountdownsResponse, adversariesResp *pb.DaggerheartListAdversariesResponse) daggerheartCombatBoardPayload {
	payload := daggerheartCombatBoardPayload{
		SessionID: strings.TrimSpace(sessionID),
		SceneID:   strings.TrimSpace(sceneID),
	}
	if snapshotResp != nil && snapshotResp.GetSnapshot() != nil && snapshotResp.GetSnapshot().GetDaggerheart() != nil {
		payload.GmFear = int(snapshotResp.GetSnapshot().GetDaggerheart().GetGmFear())
	}
	if spotlight := spotlightResp.GetSpotlight(); spotlight != nil {
		payload.Spotlight = &sessionSpotlightSummary{
			Type:        sessionSpotlightTypeToString(spotlight.GetType()),
			CharacterID: strings.TrimSpace(spotlight.GetCharacterId()),
			UpdatedAt:   formatTimestamp(spotlight.GetUpdatedAt()),
		}
	}
	if countdownsResp != nil {
		payload.Countdowns = make([]countdownSummary, 0, len(countdownsResp.GetCountdowns()))
		for _, countdown := range countdownsResp.GetCountdowns() {
			payload.Countdowns = append(payload.Countdowns, countdownSummaryFromSceneProto(countdown))
		}
	}
	if adversariesResp == nil {
		return payload
	}
	payload.Adversaries = make([]adversarySummary, 0, len(adversariesResp.GetAdversaries()))
	for _, adversary := range adversariesResp.GetAdversaries() {
		if payload.SceneID != "" && strings.TrimSpace(adversary.GetSceneId()) != payload.SceneID {
			continue
		}
		entry := adversarySummary{
			ID:              adversary.GetId(),
			Name:            adversary.GetName(),
			Kind:            strings.TrimSpace(adversary.GetKind()),
			SceneID:         adversary.GetSceneId(),
			Notes:           adversary.GetNotes(),
			HP:              int(adversary.GetHp()),
			HPMax:           int(adversary.GetHpMax()),
			Stress:          int(adversary.GetStress()),
			StressMax:       int(adversary.GetStressMax()),
			Evasion:         int(adversary.GetEvasion()),
			MajorThreshold:  int(adversary.GetMajorThreshold()),
			SevereThreshold: int(adversary.GetSevereThreshold()),
			Armor:           int(adversary.GetArmor()),
			SpotlightGateID: strings.TrimSpace(adversary.GetSpotlightGateId()),
			SpotlightCount:  int(adversary.GetSpotlightCount()),
			Conditions:      conditionsFromProto(adversary.GetConditionStates()),
			Features:        adversaryFeaturesFromProto(adversary.GetFeatureStates()),
		}
		if pending := adversary.GetPendingExperience(); pending != nil && strings.TrimSpace(pending.GetName()) != "" {
			entry.PendingExperience = &experienceSummary{Name: pending.GetName(), Modifier: int(pending.GetModifier())}
		}
		payload.Adversaries = append(payload.Adversaries, entry)
	}
	return payload
}

func applyCombatBoardDiagnostics(payload daggerheartCombatBoardPayload, sceneErr error) daggerheartCombatBoardPayload {
	switch {
	case sceneErr != nil || strings.TrimSpace(payload.SceneID) == "":
		payload.Status = "NO_ACTIVE_SCENE"
		payload.Issues = []combatBoardDiagnosticIssue{{
			Code:    "no_active_scene",
			Message: "No active scene is set for the current session, so scene-local adversaries and countdowns are unavailable.",
		}}
		payload.RecommendedTools = []string{"interaction_state_read", "interaction_activate_scene", "scene_create"}
	case len(payload.Adversaries) == 0 && len(payload.Countdowns) == 0:
		payload.Status = "EMPTY_BOARD"
		payload.Issues = []combatBoardDiagnosticIssue{{
			Code:    "empty_board",
			Message: "The active scene has no visible adversaries or scene countdowns on the combat board.",
		}}
		payload.RecommendedTools = []string{"daggerheart_adversary_create", "daggerheart_scene_countdown_create", "interaction_state_read"}
	case len(payload.Adversaries) == 0:
		payload.Status = "NO_VISIBLE_ADVERSARY"
		payload.Issues = []combatBoardDiagnosticIssue{{
			Code:    "no_visible_adversary",
			Message: "The active scene board is loaded, but there is no visible adversary to target.",
		}}
		payload.RecommendedTools = []string{"daggerheart_adversary_create", "interaction_state_read"}
	default:
		payload.Status = "READY"
	}
	return payload
}

func heritageFromProto(value *pb.DaggerheartHeritageSelection) *heritageSummary {
	if value == nil {
		return nil
	}
	ancestry := strings.TrimSpace(value.GetAncestryName())
	community := strings.TrimSpace(value.GetCommunityName())
	if ancestry == "" && community == "" {
		return nil
	}
	return &heritageSummary{Ancestry: ancestry, Community: community}
}

func traitsFromProto(profile *pb.DaggerheartProfile) *traitSummary {
	if profile == nil {
		return nil
	}
	traits := &traitSummary{
		Agility:   int(profile.GetAgility().GetValue()),
		Strength:  int(profile.GetStrength().GetValue()),
		Finesse:   int(profile.GetFinesse().GetValue()),
		Instinct:  int(profile.GetInstinct().GetValue()),
		Presence:  int(profile.GetPresence().GetValue()),
		Knowledge: int(profile.GetKnowledge().GetValue()),
	}
	if *traits == (traitSummary{}) {
		return nil
	}
	return traits
}

func experiencesFromProto(values []*pb.DaggerheartExperience) []experienceSummary {
	result := make([]experienceSummary, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value.GetName())
		if name == "" {
			continue
		}
		result = append(result, experienceSummary{Name: name, Modifier: int(value.GetModifier())})
	}
	return result
}

func resourcesFromProto(profile *pb.DaggerheartProfile, state *pb.DaggerheartCharacterState) *resourceSummary {
	if profile == nil && state == nil {
		return nil
	}
	resources := &resourceSummary{}
	if state != nil {
		resources.HP = int(state.GetHp())
		resources.Hope = int(state.GetHope())
		resources.HopeMax = int(state.GetHopeMax())
		resources.Stress = int(state.GetStress())
		resources.Armor = int(state.GetArmor())
		resources.LifeState = daggerheartLifeStateToString(state.GetLifeState())
	}
	if profile != nil {
		resources.HPMax = int(profile.GetHpMax())
		resources.StressMax = intPtrFromWrapper(profile.GetStressMax())
		resources.ArmorMax = intPtrFromWrapper(profile.GetArmorMax())
	}
	if *resources == (resourceSummary{}) {
		return nil
	}
	return resources
}

func defensesFromProto(profile *pb.DaggerheartProfile) *defenseSummary {
	if profile == nil {
		return nil
	}
	defenses := &defenseSummary{
		Evasion:            intPtrFromWrapper(profile.GetEvasion()),
		ArmorScore:         intPtrFromWrapper(profile.GetArmorScore()),
		Proficiency:        intPtrFromWrapper(profile.GetProficiency()),
		MajorThreshold:     intPtrFromWrapper(profile.GetMajorThreshold()),
		SevereThreshold:    intPtrFromWrapper(profile.GetSevereThreshold()),
		SpellcastRollBonus: intPtrFromWrapper(profile.GetSpellcastRollBonus()),
	}
	if *defenses == (defenseSummary{}) {
		return nil
	}
	return defenses
}

func equipmentFromProto(profile *pb.DaggerheartProfile) *equipmentSummary {
	if profile == nil {
		return nil
	}
	equipment := &equipmentSummary{
		PrimaryWeapon:   weaponFromProto(profile.GetPrimaryWeapon()),
		SecondaryWeapon: weaponFromProto(profile.GetSecondaryWeapon()),
		ActiveArmor:     armorFromProto(profile.GetActiveArmor()),
	}
	if item := contentRefFromID(profile.GetStartingPotionItemId()); item != nil {
		equipment.Consumables = append(equipment.Consumables, *item)
	}
	if equipment.PrimaryWeapon == nil && equipment.SecondaryWeapon == nil && equipment.ActiveArmor == nil && len(equipment.Consumables) == 0 {
		return nil
	}
	return equipment
}

func weaponFromProto(value *pb.DaggerheartSheetWeaponSummary) *weaponSummary {
	if value == nil {
		return nil
	}
	weapon := &weaponSummary{
		ID:         strings.TrimSpace(value.GetId()),
		Name:       strings.TrimSpace(value.GetName()),
		Trait:      strings.TrimSpace(value.GetTrait()),
		Range:      strings.TrimSpace(value.GetRange()),
		DamageDice: strings.TrimSpace(value.GetDamageDice()),
		DamageType: strings.TrimSpace(value.GetDamageType()),
		Feature:    strings.TrimSpace(value.GetFeature()),
	}
	if *weapon == (weaponSummary{}) {
		return nil
	}
	return weapon
}

func armorFromProto(value *pb.DaggerheartSheetArmorSummary) *armorSummary {
	if value == nil {
		return nil
	}
	armor := &armorSummary{
		ID:        strings.TrimSpace(value.GetId()),
		Name:      strings.TrimSpace(value.GetName()),
		BaseScore: intPtrIfNonZero(value.GetBaseScore()),
		Feature:   strings.TrimSpace(value.GetFeature()),
	}
	if *armor == (armorSummary{}) {
		return nil
	}
	return armor
}

func domainCardsFromProto(ids []string) []domainCardSummary {
	result := make([]domainCardSummary, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		name, domain := domainCardLabelFromID(id)
		result = append(result, domainCardSummary{ID: id, Name: name, Domain: domain})
	}
	return result
}

func activeClassFeaturesFromProto(values []*pb.DaggerheartActiveClassFeature) []activeFeatureSummary {
	result := make([]activeFeatureSummary, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value.GetName())
		if name == "" {
			continue
		}
		result = append(result, activeFeatureSummary{
			ID:          value.GetId(),
			Name:        name,
			Description: strings.TrimSpace(value.GetDescription()),
			Level:       int(value.GetLevel()),
			HopeFeature: value.GetHopeFeature(),
		})
	}
	return result
}

func activeSubclassFeaturesFromProto(values []*pb.DaggerheartActiveSubclassTrackFeatures) []activeSubclassFeatureSummary {
	var result []activeSubclassFeatureSummary
	for _, value := range values {
		track := value.GetTrack()
		appendFeatures := func(rank string, features []*pb.DaggerheartActiveSubclassFeature) {
			for _, feature := range features {
				name := strings.TrimSpace(feature.GetName())
				if name == "" {
					continue
				}
				result = append(result, activeSubclassFeatureSummary{
					ID:          feature.GetId(),
					Name:        name,
					Description: strings.TrimSpace(feature.GetDescription()),
					Level:       int(feature.GetLevel()),
					Origin:      daggerheartSubclassTrackOriginToString(track.GetOrigin()),
					Rank:        rank,
					Class:       contentLabelFromID(track.GetClassId()),
					Subclass:    contentLabelFromID(track.GetSubclassId()),
					Domain:      contentLabelFromID(track.GetDomainId()),
				})
			}
		}
		appendFeatures("FOUNDATION", value.GetFoundationFeatures())
		appendFeatures("SPECIALIZATION", value.GetSpecializationFeatures())
		appendFeatures("MASTERY", value.GetMasteryFeatures())
	}
	return result
}

func companionFromProto(sheet *pb.DaggerheartCompanionSheet, state *pb.DaggerheartCompanionState) *companionSummary {
	if sheet == nil && state == nil {
		return nil
	}
	companion := &companionSummary{}
	if sheet != nil {
		companion.Name = strings.TrimSpace(sheet.GetName())
		companion.AnimalKind = strings.TrimSpace(sheet.GetAnimalKind())
		companion.Evasion = int(sheet.GetEvasion())
		companion.AttackDescription = strings.TrimSpace(sheet.GetAttackDescription())
		companion.AttackRange = strings.TrimSpace(sheet.GetAttackRange())
		companion.DamageDieSides = int(sheet.GetDamageDieSides())
		companion.DamageType = strings.TrimSpace(sheet.GetDamageType())
		for _, experience := range sheet.GetExperiences() {
			if strings.TrimSpace(experience.GetName()) == "" {
				continue
			}
			companion.Experiences = append(companion.Experiences, experienceSummary{
				Name:     strings.TrimSpace(experience.GetName()),
				Modifier: int(experience.GetModifier()),
			})
		}
	}
	if state != nil {
		companion.Status = strings.TrimSpace(state.GetStatus())
		companion.ActiveExperience = strings.TrimSpace(state.GetActiveExperienceId())
	}
	if companion.Name == "" &&
		companion.AnimalKind == "" &&
		companion.Evasion == 0 &&
		companion.AttackDescription == "" &&
		companion.AttackRange == "" &&
		companion.DamageDieSides == 0 &&
		companion.DamageType == "" &&
		companion.Status == "" &&
		companion.ActiveExperience == "" &&
		len(companion.Experiences) == 0 {
		return nil
	}
	return companion
}

func classStateFromProto(value *pb.DaggerheartClassState) *classStateSummary {
	if value == nil {
		return nil
	}
	state := &classStateSummary{
		AttackBonusUntilRest:            int(value.GetAttackBonusUntilRest()),
		EvasionBonusUntilHitOrRest:      int(value.GetEvasionBonusUntilHitOrRest()),
		DifficultyPenaltyUntilRest:      int(value.GetDifficultyPenaltyUntilRest()),
		FocusTargetID:                   strings.TrimSpace(value.GetFocusTargetId()),
		StrangePatternsNumber:           int(value.GetStrangePatternsNumber()),
		RallyDice:                       intSlice(value.GetRallyDice()),
		PrayerDice:                      intSlice(value.GetPrayerDice()),
		ChannelRawPowerUsedThisLongRest: value.GetChannelRawPowerUsedThisLongRest(),
		ActiveBeastform:                 beastformFromProto(value.GetActiveBeastform()),
	}
	if unstoppable := unstoppableFromProto(value.GetUnstoppable()); unstoppable != nil {
		state.Unstoppable = unstoppable
	}
	if state.AttackBonusUntilRest == 0 &&
		state.EvasionBonusUntilHitOrRest == 0 &&
		state.DifficultyPenaltyUntilRest == 0 &&
		state.FocusTargetID == "" &&
		state.StrangePatternsNumber == 0 &&
		len(state.RallyDice) == 0 &&
		len(state.PrayerDice) == 0 &&
		state.Unstoppable == nil &&
		!state.ChannelRawPowerUsedThisLongRest &&
		state.ActiveBeastform == nil {
		return nil
	}
	return state
}

func unstoppableFromProto(value *pb.DaggerheartUnstoppableState) *unstoppableSummary {
	if value == nil {
		return nil
	}
	summary := &unstoppableSummary{
		Active:           value.GetActive(),
		CurrentValue:     int(value.GetCurrentValue()),
		DieSides:         int(value.GetDieSides()),
		UsedThisLongRest: value.GetUsedThisLongRest(),
	}
	if *summary == (unstoppableSummary{}) {
		return nil
	}
	return summary
}

func beastformFromProto(value *pb.DaggerheartActiveBeastformState) *beastformSummary {
	if value == nil {
		return nil
	}
	summary := &beastformSummary{
		BeastformID:            strings.TrimSpace(value.GetBeastformId()),
		BaseTrait:              strings.TrimSpace(value.GetBaseTrait()),
		AttackTrait:            strings.TrimSpace(value.GetAttackTrait()),
		TraitBonus:             int(value.GetTraitBonus()),
		EvasionBonus:           int(value.GetEvasionBonus()),
		AttackRange:            strings.TrimSpace(value.GetAttackRange()),
		DamageBonus:            int(value.GetDamageBonus()),
		DamageType:             strings.TrimSpace(value.GetDamageType()),
		EvolutionTraitOverride: strings.TrimSpace(value.GetEvolutionTraitOverride()),
		DropOnAnyHPMark:        value.GetDropOnAnyHpMark(),
	}
	for _, die := range value.GetDamageDice() {
		summary.DamageDice = append(summary.DamageDice, damageDieSpec{
			Count: int(die.GetCount()),
			Sides: int(die.GetSides()),
		})
	}
	if summary.BeastformID == "" &&
		summary.BaseTrait == "" &&
		summary.AttackTrait == "" &&
		summary.TraitBonus == 0 &&
		summary.EvasionBonus == 0 &&
		summary.AttackRange == "" &&
		len(summary.DamageDice) == 0 &&
		summary.DamageBonus == 0 &&
		summary.DamageType == "" &&
		summary.EvolutionTraitOverride == "" &&
		!summary.DropOnAnyHPMark {
		return nil
	}
	return summary
}

func subclassStateFromProto(value *pb.DaggerheartSubclassState) *subclassStateSummary {
	if value == nil {
		return nil
	}
	state := &subclassStateSummary{
		BattleRitualUsedThisLongRest:           value.GetBattleRitualUsedThisLongRest(),
		GiftedPerformerRelaxingSongUses:        int(value.GetGiftedPerformerRelaxingSongUses()),
		GiftedPerformerEpicSongUses:            int(value.GetGiftedPerformerEpicSongUses()),
		GiftedPerformerHeartbreakingSongUses:   int(value.GetGiftedPerformerHeartbreakingSongUses()),
		ContactsEverywhereUsesThisSession:      int(value.GetContactsEverywhereUsesThisSession()),
		ContactsEverywhereActionDieBonus:       int(value.GetContactsEverywhereActionDieBonus()),
		ContactsEverywhereDamageDiceBonusCount: int(value.GetContactsEverywhereDamageDiceBonusCount()),
		SparingTouchUsesThisLongRest:           int(value.GetSparingTouchUsesThisLongRest()),
		ElementalistActionBonus:                int(value.GetElementalistActionBonus()),
		ElementalistDamageBonus:                int(value.GetElementalistDamageBonus()),
		TranscendenceActive:                    value.GetTranscendenceActive(),
		TranscendenceTraitBonusTarget:          strings.TrimSpace(value.GetTranscendenceTraitBonusTarget()),
		TranscendenceTraitBonusValue:           int(value.GetTranscendenceTraitBonusValue()),
		TranscendenceProficiencyBonus:          int(value.GetTranscendenceProficiencyBonus()),
		TranscendenceEvasionBonus:              int(value.GetTranscendenceEvasionBonus()),
		TranscendenceSevereThresholdBonus:      int(value.GetTranscendenceSevereThresholdBonus()),
		ClarityOfNatureUsedThisLongRest:        value.GetClarityOfNatureUsedThisLongRest(),
		ElementalChannel:                       strings.TrimSpace(value.GetElementalChannel()),
		NemesisTargetID:                        strings.TrimSpace(value.GetNemesisTargetId()),
		RousingSpeechUsedThisLongRest:          value.GetRousingSpeechUsedThisLongRest(),
		WardensProtectionUsedThisLongRest:      value.GetWardensProtectionUsedThisLongRest(),
	}
	if *state == (subclassStateSummary{}) {
		return nil
	}
	return state
}

func adversaryFeaturesFromProto(values []*pb.DaggerheartAdversaryFeatureState) []adversaryFeatureSummary {
	result := make([]adversaryFeatureSummary, 0, len(values))
	for _, value := range values {
		result = append(result, adversaryFeatureSummary{
			FeatureID:       strings.TrimSpace(value.GetFeatureId()),
			Status:          daggerheartAdversaryFeatureStateStatusToString(value.GetStatus()),
			FocusedTargetID: strings.TrimSpace(value.GetFocusedTargetId()),
		})
	}
	return result
}

func conditionsFromProto(values []*pb.DaggerheartConditionState) []conditionEntry {
	result := make([]conditionEntry, 0, len(values))
	for _, value := range values {
		label := strings.TrimSpace(value.GetLabel())
		if label == "" {
			label = strings.TrimSpace(value.GetCode())
		}
		if label == "" {
			continue
		}
		clearTriggers := make([]string, 0, len(value.GetClearTriggers()))
		for _, trigger := range value.GetClearTriggers() {
			clearTriggers = append(clearTriggers, daggerheartConditionClearTriggerToString(trigger))
		}
		result = append(result, conditionEntry{Label: label, ClearTriggers: clearTriggers})
	}
	return result
}

func temporaryArmorFromProto(values []*pb.DaggerheartTemporaryArmorBucket) []temporaryArmorEntry {
	result := make([]temporaryArmorEntry, 0, len(values))
	for _, value := range values {
		result = append(result, temporaryArmorEntry{
			Source: strings.TrimSpace(value.GetSource()),
			Amount: int(value.GetAmount()),
		})
	}
	return result
}

func statModifiersFromProto(values []*pb.DaggerheartStatModifier) []statModifierEntry {
	result := make([]statModifierEntry, 0, len(values))
	for _, value := range values {
		result = append(result, statModifierEntry{
			Target: strings.TrimSpace(value.GetTarget()),
			Delta:  int(value.GetDelta()),
			Label:  strings.TrimSpace(value.GetLabel()),
		})
	}
	return result
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

func contentRefFromID(id string) *contentReference {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	return &contentReference{ID: id, Name: contentLabelFromID(id)}
}

func contentLabelFromID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.IndexAny(value, ".:"); idx >= 0 && idx < len(value)-1 {
		value = value[idx+1:]
	}
	return humanizeContentSlug(value)
}

func domainCardLabelFromID(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	if idx := strings.IndexAny(value, ".:"); idx >= 0 && idx < len(value)-1 {
		value = value[idx+1:]
	}
	domain := ""
	if idx := strings.Index(value, "-"); idx >= 0 && idx < len(value)-1 {
		domain = humanizeContentSlug(value[:idx])
		value = value[idx+1:]
	}
	return humanizeContentSlug(value), domain
}

func humanizeContentSlug(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.NewReplacer("-", " ", "_", " ").Replace(value)
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return ""
	}
	for i, part := range parts {
		lower := strings.ToLower(part)
		parts[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(parts, " ")
}

func intPtrIfNonZero(value int32) *int {
	if value == 0 {
		return nil
	}
	converted := int(value)
	return &converted
}

func intPtrFromWrapper(value *wrapperspb.Int32Value) *int {
	if value == nil {
		return nil
	}
	converted := int(value.GetValue())
	return &converted
}
