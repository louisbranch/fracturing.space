package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (r *Runner) failf(format string, args ...any) error {
	return r.assertions.Failf(format, args...)
}

func (r *Runner) assertf(format string, args ...any) error {
	return r.assertions.Assertf(format, args...)
}

func (r *Runner) ensureCampaign(state *scenarioState) error {
	if state.campaignID == "" {
		return r.failf("campaign is required")
	}
	return nil
}

func (r *Runner) ensureSession(ctx context.Context, state *scenarioState) error {
	if state.campaignID == "" {
		return r.failf("campaign is required")
	}
	if state.sessionID != "" {
		return nil
	}
	response, err := r.env.sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId: state.campaignID,
		Name:       "Scenario Session",
	})
	if err != nil {
		return fmt.Errorf("auto start session: %w", err)
	}
	if response.GetSession() == nil {
		return r.failf("expected session")
	}
	state.sessionID = response.GetSession().GetId()
	return nil
}

func (r *Runner) latestSeq(ctx context.Context, state *scenarioState) (uint64, error) {
	if state.campaignID == "" {
		return 0, nil
	}
	response, err := r.env.eventClient.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: state.campaignID,
		PageSize:   1,
		OrderBy:    "seq desc",
	})
	if err != nil {
		return 0, fmt.Errorf("list events: %w", err)
	}
	if len(response.GetEvents()) == 0 {
		return 0, nil
	}
	return response.GetEvents()[0].GetSeq(), nil
}

func (r *Runner) requireEventTypesAfterSeq(ctx context.Context, state *scenarioState, before uint64, types ...event.Type) error {
	for _, eventType := range types {
		filter := fmt.Sprintf("type = \"%s\"", eventType)
		if state.sessionID != "" && isSessionEvent(string(eventType)) {
			filter = filter + fmt.Sprintf(" AND session_id = \"%s\"", state.sessionID)
		}
		response, err := r.env.eventClient.ListEvents(ctx, &gamev1.ListEventsRequest{
			CampaignId: state.campaignID,
			PageSize:   1,
			OrderBy:    "seq desc",
			Filter:     filter,
		})
		if err != nil {
			return fmt.Errorf("list events for %s: %w", eventType, err)
		}
		if len(response.GetEvents()) == 0 {
			if err := r.assertf("expected event %s", eventType); err != nil {
				return err
			}
			continue
		}
		if response.GetEvents()[0].GetSeq() <= before {
			if err := r.assertf("expected %s after seq %d", eventType, before); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) requireAnyEventTypesAfterSeq(ctx context.Context, state *scenarioState, before uint64, types ...event.Type) error {
	for _, eventType := range types {
		ok, err := r.hasEventTypeAfterSeq(ctx, state, before, eventType)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	labels := make([]string, 0, len(types))
	for _, eventType := range types {
		labels = append(labels, string(eventType))
	}
	return r.assertf("expected event after seq %d: %s", before, strings.Join(labels, ", "))
}

func (r *Runner) resolveOpenSessionGate(ctx context.Context, state *scenarioState, before uint64) error {
	filter := fmt.Sprintf("type = \"%s\"", event.TypeSessionGateOpened)
	if state.sessionID != "" {
		filter = filter + fmt.Sprintf(" AND session_id = \"%s\"", state.sessionID)
	}
	response, err := r.env.eventClient.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: state.campaignID,
		PageSize:   20,
		OrderBy:    "seq desc",
		Filter:     filter,
	})
	if err != nil {
		return fmt.Errorf("list events for %s: %w", event.TypeSessionGateOpened, err)
	}
	gateID := ""
	for _, evt := range response.GetEvents() {
		if evt.GetSeq() <= before {
			continue
		}
		var payload event.SessionGateOpenedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return fmt.Errorf("decode session gate payload: %w", err)
		}
		if strings.TrimSpace(payload.GateID) == "" {
			continue
		}
		gateID = payload.GateID
		break
	}
	if gateID == "" {
		return r.failf("session gate opened event not found")
	}
	_, err = r.env.sessionClient.ResolveSessionGate(ctx, &gamev1.ResolveSessionGateRequest{
		CampaignId: state.campaignID,
		SessionId:  state.sessionID,
		GateId:     gateID,
		Decision:   "allow",
	})
	if err != nil {
		return fmt.Errorf("resolve session gate: %w", err)
	}
	return r.requireEventTypesAfterSeq(ctx, state, before, event.TypeSessionGateResolved)
}

func (r *Runner) hasEventTypeAfterSeq(ctx context.Context, state *scenarioState, before uint64, eventType event.Type) (bool, error) {
	filter := fmt.Sprintf("type = \"%s\"", eventType)
	if state.sessionID != "" && isSessionEvent(string(eventType)) {
		filter = filter + fmt.Sprintf(" AND session_id = \"%s\"", state.sessionID)
	}
	response, err := r.env.eventClient.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: state.campaignID,
		PageSize:   1,
		OrderBy:    "seq desc",
		Filter:     filter,
	})
	if err != nil {
		return false, fmt.Errorf("list events for %s: %w", eventType, err)
	}
	if len(response.GetEvents()) == 0 {
		return false, nil
	}
	return response.GetEvents()[0].GetSeq() > before, nil
}

func isSessionEvent(eventType string) bool {
	return strings.HasPrefix(eventType, "action.") || strings.HasPrefix(eventType, "session.")
}

func (r *Runner) applyDefaultDaggerheartProfile(ctx context.Context, state *scenarioState, characterID string, args map[string]any) error {
	armorValue := optionalInt(args, "armor", 0)
	armorMaxValue := optionalInt(args, "armor_max", 0)
	profile := &daggerheartv1.DaggerheartProfile{
		Level:           int32(optionalInt(args, "level", 1)),
		HpMax:           int32(optionalInt(args, "hp_max", 6)),
		StressMax:       wrapperspb.Int32(int32(optionalInt(args, "stress_max", 6))),
		Evasion:         wrapperspb.Int32(int32(optionalInt(args, "evasion", 10))),
		MajorThreshold:  wrapperspb.Int32(int32(optionalInt(args, "major_threshold", 3))),
		SevereThreshold: wrapperspb.Int32(int32(optionalInt(args, "severe_threshold", 6))),
	}
	if armorMaxValue > 0 {
		profile.ArmorMax = wrapperspb.Int32(int32(armorMaxValue))
	} else if armorValue > 0 {
		profile.ArmorMax = wrapperspb.Int32(int32(armorValue))
	}
	if value := optionalInt(args, "armor_score", 0); value > 0 {
		profile.ArmorScore = wrapperspb.Int32(int32(value))
	}
	applyTraitValue(profile, "agility", args)
	applyTraitValue(profile, "strength", args)
	applyTraitValue(profile, "finesse", args)
	applyTraitValue(profile, "instinct", args)
	applyTraitValue(profile, "presence", args)
	applyTraitValue(profile, "knowledge", args)

	_, err := r.env.characterClient.PatchCharacterProfile(ctx, &gamev1.PatchCharacterProfileRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		SystemProfilePatch: &gamev1.PatchCharacterProfileRequest_Daggerheart{
			Daggerheart: profile,
		},
	})
	if err != nil {
		return fmt.Errorf("patch character profile: %w", err)
	}
	return nil
}

func (r *Runner) applyOptionalCharacterState(ctx context.Context, state *scenarioState, characterID string, args map[string]any) error {
	patch := &daggerheartv1.DaggerheartCharacterState{}
	hasPatch := false
	armorSet := false
	hpSet := false
	stressSet := false
	lifeStateSet := false
	if armor, ok := readInt(args, "armor"); ok {
		patch.Armor = int32(armor)
		hasPatch = true
		armorSet = true
	}
	if hp, ok := readInt(args, "hp"); ok {
		patch.Hp = int32(hp)
		hasPatch = true
		hpSet = true
	}
	if stress, ok := readInt(args, "stress"); ok {
		patch.Stress = int32(stress)
		hasPatch = true
		stressSet = true
	}
	if lifeState := optionalString(args, "life_state", ""); lifeState != "" {
		value, err := parseLifeState(lifeState)
		if err != nil {
			return err
		}
		patch.LifeState = value
		hasPatch = true
		lifeStateSet = true
	}
	if !hasPatch {
		return nil
	}
	// PatchCharacterState overwrites the full state, so merge with current values.
	current, err := r.getCharacterState(ctx, state, characterID)
	if err != nil {
		return err
	}
	if !hpSet {
		patch.Hp = current.GetHp()
	}
	patch.Hope = current.GetHope()
	patch.HopeMax = current.GetHopeMax()
	if !stressSet {
		patch.Stress = current.GetStress()
	}
	if !armorSet {
		patch.Armor = current.GetArmor()
	}
	if !lifeStateSet {
		patch.LifeState = current.GetLifeState()
	}
	_, err = r.env.snapshotClient.PatchCharacterState(ctx, &gamev1.PatchCharacterStateRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		SystemStatePatch: &gamev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: patch,
		},
	})
	if err != nil {
		return fmt.Errorf("patch character state: %w", err)
	}
	return nil
}

func applyTraitValue(profile *daggerheartv1.DaggerheartProfile, key string, args map[string]any) {
	value := optionalInt(args, key, 0)
	if value == 0 {
		return
	}
	boxed := wrapperspb.Int32(int32(value))
	switch key {
	case "agility":
		profile.Agility = boxed
	case "strength":
		profile.Strength = boxed
	case "finesse":
		profile.Finesse = boxed
	case "instinct":
		profile.Instinct = boxed
	case "presence":
		profile.Presence = boxed
	case "knowledge":
		profile.Knowledge = boxed
	}
}

func (r *Runner) getSnapshot(ctx context.Context, state *scenarioState) (*daggerheartv1.DaggerheartSnapshot, error) {
	response, err := r.env.snapshotClient.GetSnapshot(ctx, &gamev1.GetSnapshotRequest{CampaignId: state.campaignID})
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	if response.GetSnapshot() == nil || response.GetSnapshot().GetDaggerheart() == nil {
		return nil, r.failf("expected daggerheart snapshot")
	}
	return response.GetSnapshot().GetDaggerheart(), nil
}

func (r *Runner) getCharacterState(ctx context.Context, state *scenarioState, characterID string) (*daggerheartv1.DaggerheartCharacterState, error) {
	response, err := r.env.characterClient.GetCharacterSheet(ctx, &gamev1.GetCharacterSheetRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return nil, fmt.Errorf("get character sheet: %w", err)
	}
	if response.GetState() == nil || response.GetState().GetDaggerheart() == nil {
		return nil, r.failf("expected daggerheart character state")
	}
	return response.GetState().GetDaggerheart(), nil
}

func (r *Runner) getAdversary(ctx context.Context, state *scenarioState, adversaryID string) (*daggerheartv1.DaggerheartAdversary, error) {
	response, err := r.env.daggerheartClient.GetAdversary(ctx, &daggerheartv1.DaggerheartGetAdversaryRequest{
		CampaignId:  state.campaignID,
		AdversaryId: adversaryID,
	})
	if err != nil {
		return nil, fmt.Errorf("get adversary: %w", err)
	}
	if response.GetAdversary() == nil {
		return nil, r.failf("expected adversary")
	}
	return response.GetAdversary(), nil
}

func chooseActionSeed(args map[string]any, difficulty int) (uint64, error) {
	hint := strings.ToLower(optionalString(args, "outcome", ""))
	if hint == "" {
		return 42, nil
	}
	if seed, ok := cachedActionSeed(difficulty, hint); ok {
		return seed, nil
	}
	for seed := uint64(1); seed < 50000; seed++ {
		result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
			Modifier:   0,
			Difficulty: &difficulty,
			Seed:       int64(seed),
		})
		if err != nil {
			continue
		}
		if matchesOutcomeHint(result, hint) {
			cacheActionSeed(difficulty, hint, seed)
			return seed, nil
		}
	}
	return 0, fmt.Errorf("no seed found for outcome %q", hint)
}

type actionSeedKey struct {
	difficulty int
	hint       string
}

var (
	actionSeedMu    sync.Mutex
	actionSeedCache = map[actionSeedKey]uint64{}
)

func cachedActionSeed(difficulty int, hint string) (uint64, bool) {
	actionSeedMu.Lock()
	defer actionSeedMu.Unlock()
	seed, ok := actionSeedCache[actionSeedKey{difficulty: difficulty, hint: hint}]
	return seed, ok
}

func cacheActionSeed(difficulty int, hint string, seed uint64) {
	actionSeedMu.Lock()
	defer actionSeedMu.Unlock()
	actionSeedCache[actionSeedKey{difficulty: difficulty, hint: hint}] = seed
}

func matchesOutcomeHint(result daggerheartdomain.ActionResult, hint string) bool {
	switch hint {
	case "fear":
		return result.Outcome == daggerheartdomain.OutcomeRollWithFear ||
			result.Outcome == daggerheartdomain.OutcomeSuccessWithFear ||
			result.Outcome == daggerheartdomain.OutcomeFailureWithFear
	case "hope":
		return result.Outcome == daggerheartdomain.OutcomeRollWithHope ||
			result.Outcome == daggerheartdomain.OutcomeSuccessWithHope ||
			result.Outcome == daggerheartdomain.OutcomeFailureWithHope
	case "critical":
		return result.IsCrit
	case "failure_hope":
		return result.Outcome == daggerheartdomain.OutcomeFailureWithHope
	default:
		return false
	}
}

func resolveOutcomeSeed(args map[string]any, key string, difficulty int, fallback uint64) (uint64, error) {
	hint := optionalString(args, key, "")
	if hint == "" {
		return fallback, nil
	}
	return chooseActionSeed(map[string]any{"outcome": hint}, difficulty)
}

func buildActionRollModifiers(args map[string]any, key string) []*daggerheartv1.ActionRollModifier {
	value, ok := args[key]
	if !ok {
		return nil
	}
	list, ok := value.([]any)
	if !ok || len(list) == 0 {
		return nil
	}
	modifiers := make([]*daggerheartv1.ActionRollModifier, 0, len(list))
	for index, entry := range list {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		source := optionalString(item, "source", fmt.Sprintf("modifier_%d", index))
		value, ok := readInt(item, "value")
		if !ok {
			if isHopeSpendSource(source) {
				value = 0
			} else {
				continue
			}
		}
		modifiers = append(modifiers, &daggerheartv1.ActionRollModifier{
			Source: source,
			Value:  int32(value),
		})
	}
	return modifiers
}

func buildDamageDice(args map[string]any) []*daggerheartv1.DiceSpec {
	value, ok := args["damage_dice"]
	if !ok {
		return []*daggerheartv1.DiceSpec{{Sides: 6, Count: 1}}
	}
	list, ok := value.([]any)
	if !ok || len(list) == 0 {
		return []*daggerheartv1.DiceSpec{{Sides: 6, Count: 1}}
	}
	results := make([]*daggerheartv1.DiceSpec, 0, len(list))
	for _, entry := range list {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		sides := optionalInt(item, "sides", 6)
		count := optionalInt(item, "count", 1)
		results = append(results, &daggerheartv1.DiceSpec{Sides: int32(sides), Count: int32(count)})
	}
	if len(results) == 0 {
		return []*daggerheartv1.DiceSpec{{Sides: 6, Count: 1}}
	}
	return results
}

func buildDamageSpec(args map[string]any, actorID, source string) *daggerheartv1.DaggerheartAttackDamageSpec {
	damageType := parseDamageType(optionalString(args, "damage_type", "physical"))
	spec := &daggerheartv1.DaggerheartAttackDamageSpec{DamageType: damageType}
	if source != "" {
		spec.Source = source
	}
	if actorID != "" {
		spec.SourceCharacterIds = []string{actorID}
	}
	spec.ResistPhysical = optionalBool(args, "resist_physical", false)
	spec.ResistMagic = optionalBool(args, "resist_magic", false)
	spec.ImmunePhysical = optionalBool(args, "immune_physical", false)
	spec.ImmuneMagic = optionalBool(args, "immune_magic", false)
	spec.Direct = optionalBool(args, "direct", false)
	spec.MassiveDamage = optionalBool(args, "massive_damage", false)
	return spec
}

func buildDamageRequest(args map[string]any, actorID, source string, amount int32) *daggerheartv1.DaggerheartDamageRequest {
	damageType := parseDamageType(optionalString(args, "damage_type", "physical"))
	request := &daggerheartv1.DaggerheartDamageRequest{Amount: amount, DamageType: damageType}
	if source != "" {
		request.Source = source
	}
	if actorID != "" {
		request.SourceCharacterIds = []string{actorID}
	}
	request.ResistPhysical = optionalBool(args, "resist_physical", false)
	request.ResistMagic = optionalBool(args, "resist_magic", false)
	request.ImmunePhysical = optionalBool(args, "immune_physical", false)
	request.ImmuneMagic = optionalBool(args, "immune_magic", false)
	request.Direct = optionalBool(args, "direct", false)
	request.MassiveDamage = optionalBool(args, "massive_damage", false)
	return request
}

func buildDamageRequestWithSources(
	args map[string]any,
	source string,
	amount int32,
	sourceIDs []string,
) *daggerheartv1.DaggerheartDamageRequest {
	request := buildDamageRequest(args, "", source, amount)
	request.SourceCharacterIds = uniqueNonEmptyStrings(sourceIDs)
	return request
}

func (r *Runner) applyAdversaryDamage(
	ctx context.Context,
	state *scenarioState,
	adversaryID string,
	name string,
	damageRoll *daggerheartv1.SessionDamageRollResponse,
	args map[string]any,
) (bool, error) {
	before, err := r.getAdversary(ctx, state, adversaryID)
	if err != nil {
		return false, err
	}
	hpBefore := int(before.GetHp())
	armorBefore := int(before.GetArmor())
	majorThreshold := int(before.GetMajorThreshold())
	severeThreshold := int(before.GetSevereThreshold())

	amount := int(damageRoll.GetTotal())
	resistance := daggerheart.ResistanceProfile{
		ResistPhysical: optionalBool(args, "resist_physical", false),
		ResistMagic:    optionalBool(args, "resist_magic", false),
		ImmunePhysical: optionalBool(args, "immune_physical", false),
		ImmuneMagic:    optionalBool(args, "immune_magic", false),
	}
	adjusted := daggerheart.ApplyResistance(amount, damageTypesForArgs(args), resistance)
	if adjusted <= 0 {
		return false, nil
	}
	options := daggerheart.DamageOptions{EnableMassiveDamage: optionalBool(args, "massive_damage", false)}

	result, err := daggerheart.EvaluateDamage(adjusted, majorThreshold, severeThreshold, options)
	if err != nil {
		return false, fmt.Errorf("adversary damage: %w", err)
	}

	var app daggerheart.DamageApplication
	if optionalBool(args, "direct", false) {
		app, err = daggerheart.ApplyDamage(hpBefore, adjusted, majorThreshold, severeThreshold, options)
		if err != nil {
			return false, fmt.Errorf("adversary damage: %w", err)
		}
	} else {
		app = daggerheart.ApplyDamageWithArmor(hpBefore, armorBefore, result)
	}
	if app.HPAfter >= hpBefore && app.ArmorAfter >= armorBefore {
		if err := r.assertf("expected damage to affect hp or armor for %s", name); err != nil {
			return false, err
		}
	}

	update := &daggerheartv1.DaggerheartUpdateAdversaryRequest{
		CampaignId:  state.campaignID,
		AdversaryId: adversaryID,
	}
	if state.sessionID != "" {
		update.SessionId = wrapperspb.String(state.sessionID)
	}
	if app.HPAfter != hpBefore {
		update.Hp = wrapperspb.Int32(int32(app.HPAfter))
	}
	if app.ArmorAfter != armorBefore {
		update.Armor = wrapperspb.Int32(int32(app.ArmorAfter))
	}
	if update.Hp == nil && update.Armor == nil {
		if err := r.assertf("expected adversary damage to change hp or armor for %s", name); err != nil {
			return false, err
		}
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	if _, err := r.env.daggerheartClient.UpdateAdversary(ctxWithSession, update); err != nil {
		return false, fmt.Errorf("update adversary damage: %w", err)
	}
	after, err := r.getAdversary(ctx, state, adversaryID)
	if err != nil {
		return false, err
	}
	if after.GetHp() >= before.GetHp() && after.GetArmor() >= before.GetArmor() {
		if err := r.assertf("expected damage to affect hp or armor for %s", name); err != nil {
			return false, err
		}
	}
	return true, nil
}

func parseDamageType(value string) daggerheartv1.DaggerheartDamageType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "magic":
		return daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC
	case "mixed":
		return daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED
	default:
		return daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL
	}
}

func damageTypesForArgs(args map[string]any) daggerheart.DamageTypes {
	switch parseDamageType(optionalString(args, "damage_type", "physical")) {
	case daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC:
		return daggerheart.DamageTypes{Magic: true}
	case daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED:
		return daggerheart.DamageTypes{Physical: true, Magic: true}
	default:
		return daggerheart.DamageTypes{Physical: true}
	}
}

func adjustedDamageAmount(args map[string]any, amount int32) int {
	resistance := daggerheart.ResistanceProfile{
		ResistPhysical: optionalBool(args, "resist_physical", false),
		ResistMagic:    optionalBool(args, "resist_magic", false),
		ImmunePhysical: optionalBool(args, "immune_physical", false),
		ImmuneMagic:    optionalBool(args, "immune_magic", false),
	}
	return daggerheart.ApplyResistance(int(amount), damageTypesForArgs(args), resistance)
}

func expectDamageEffect(args map[string]any, roll *daggerheartv1.SessionDamageRollResponse) bool {
	if roll == nil {
		return false
	}
	return adjustedDamageAmount(args, roll.GetTotal()) > 0
}

func parseConditions(values []string) ([]daggerheartv1.DaggerheartCondition, error) {
	result := make([]daggerheartv1.DaggerheartCondition, 0, len(values))
	for _, value := range values {
		switch strings.ToUpper(strings.TrimSpace(value)) {
		case "VULNERABLE":
			result = append(result, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)
		case "RESTRAINED":
			result = append(result, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED)
		case "HIDDEN":
			result = append(result, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)
		default:
			return nil, fmt.Errorf("unknown condition %q", value)
		}
	}
	return result, nil
}

func parseGameSystem(value string) (commonv1.GameSystem, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "DAGGERHEART":
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, nil
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, fmt.Errorf("unsupported system %q", value)
	}
}

func parseGmMode(value string) (gamev1.GmMode, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HUMAN":
		return gamev1.GmMode_HUMAN, nil
	case "AI":
		return gamev1.GmMode_AI, nil
	default:
		return gamev1.GmMode_GM_MODE_UNSPECIFIED, fmt.Errorf("unsupported gm_mode %q", value)
	}
}

func parseCampaignIntent(value string) (gamev1.CampaignIntent, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "STANDARD":
		return gamev1.CampaignIntent_STANDARD, nil
	case "STARTER":
		return gamev1.CampaignIntent_STARTER, nil
	case "SANDBOX":
		return gamev1.CampaignIntent_SANDBOX, nil
	default:
		return gamev1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED, fmt.Errorf("unsupported intent %q", value)
	}
}

func parseCampaignAccessPolicy(value string) (gamev1.CampaignAccessPolicy, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PRIVATE":
		return gamev1.CampaignAccessPolicy_PRIVATE, nil
	case "RESTRICTED":
		return gamev1.CampaignAccessPolicy_RESTRICTED, nil
	case "PUBLIC":
		return gamev1.CampaignAccessPolicy_PUBLIC, nil
	default:
		return gamev1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED, fmt.Errorf("unsupported access policy %q", value)
	}
}

func parseCharacterKind(value string) (gamev1.CharacterKind, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PC":
		return gamev1.CharacterKind_PC, nil
	case "NPC":
		return gamev1.CharacterKind_NPC, nil
	default:
		return gamev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED, fmt.Errorf("unsupported character kind %q", value)
	}
}

func parseParticipantRole(value string) (gamev1.ParticipantRole, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PLAYER", "":
		return gamev1.ParticipantRole_PLAYER, nil
	case "GM":
		return gamev1.ParticipantRole_GM, nil
	default:
		return gamev1.ParticipantRole_ROLE_UNSPECIFIED, fmt.Errorf("unsupported participant role %q", value)
	}
}

func parseController(value string) (gamev1.Controller, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HUMAN", "":
		return gamev1.Controller_CONTROLLER_HUMAN, nil
	case "AI":
		return gamev1.Controller_CONTROLLER_AI, nil
	default:
		return gamev1.Controller_CONTROLLER_UNSPECIFIED, fmt.Errorf("unsupported controller %q", value)
	}
}

func parseControl(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "", nil
	}
	switch trimmed {
	case "participant", "gm", "none":
		return trimmed, nil
	default:
		return "", fmt.Errorf("unsupported control %q", value)
	}
}

func prefabOptions(name string) map[string]any {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "frodo":
		return map[string]any{
			"kind":             "PC",
			"armor":            1,
			"hp_max":           6,
			"stress_max":       6,
			"evasion":          10,
			"major_threshold":  3,
			"severe_threshold": 6,
		}
	default:
		return map[string]any{"kind": "PC"}
	}
}

func actorID(state *scenarioState, name string) (string, error) {
	id, ok := state.actors[name]
	if !ok {
		for key, value := range state.actors {
			if strings.EqualFold(key, name) {
				id = value
				ok = true
				break
			}
		}
	}
	if !ok {
		return "", fmt.Errorf("unknown actor %q", name)
	}
	return id, nil
}

func adversaryID(state *scenarioState, name string) (string, error) {
	id, ok := state.adversaries[name]
	if !ok {
		for key, value := range state.adversaries {
			if strings.EqualFold(key, name) {
				id = value
				ok = true
				break
			}
		}
	}
	if !ok {
		return "", fmt.Errorf("unknown adversary %q", name)
	}
	return id, nil
}

func resolveTargetID(state *scenarioState, name string) (string, bool, error) {
	if id, ok := state.actors[name]; ok {
		return id, false, nil
	}
	if id, ok := state.adversaries[name]; ok {
		return id, true, nil
	}
	for key, value := range state.actors {
		if strings.EqualFold(key, name) {
			return value, false, nil
		}
	}
	for key, value := range state.adversaries {
		if strings.EqualFold(key, name) {
			return value, true, nil
		}
	}
	return "", false, fmt.Errorf("unknown target %q", name)
}

func resolveCountdownID(state *scenarioState, args map[string]any) (string, error) {
	if countdownID := optionalString(args, "countdown_id", ""); countdownID != "" {
		return countdownID, nil
	}
	name := optionalString(args, "name", "")
	if name == "" {
		return "", nil
	}
	countdownID, ok := state.countdowns[name]
	if !ok {
		return "", fmt.Errorf("unknown countdown %q", name)
	}
	return countdownID, nil
}

func resolveOutcomeTargets(state *scenarioState, args map[string]any) ([]string, error) {
	list := readStringSlice(args, "targets")
	if len(list) == 0 {
		if name := optionalString(args, "target", ""); name != "" {
			list = []string{name}
		}
	}
	if len(list) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(list))
	for _, name := range list {
		id, err := actorID(state, name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func resolveAttackTargets(state *scenarioState, args map[string]any) ([]string, error) {
	list := readStringSlice(args, "targets")
	if len(list) == 0 {
		if name := optionalString(args, "target", ""); name != "" {
			list = []string{name}
		}
	}
	if len(list) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(list))
	for _, name := range list {
		id, _, err := resolveTargetID(state, name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func requireDamageDice(args map[string]any, context string) error {
	value, ok := args["damage_dice"]
	if !ok {
		return fmt.Errorf("%s requires damage_dice", context)
	}
	list, ok := value.([]any)
	if !ok || len(list) == 0 {
		return fmt.Errorf("%s damage_dice must be a list", context)
	}
	return nil
}

func requiredString(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if ok && text != "" {
		return text
	}
	return ""
}

func readInt(args map[string]any, key string) (int, bool) {
	value, ok := args[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return typed, true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func optionalString(args map[string]any, key, fallback string) string {
	value, ok := args[key]
	if !ok {
		return fallback
	}
	text, ok := value.(string)
	if ok && text != "" {
		return text
	}
	return fallback
}

func optionalInt(args map[string]any, key string, fallback int) int {
	value, ok := args[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return fallback
	}
}

func optionalBool(args map[string]any, key string, fallback bool) bool {
	value, ok := args[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		lower := strings.ToLower(strings.TrimSpace(typed))
		if lower == "true" || lower == "yes" || lower == "1" {
			return true
		}
		if lower == "false" || lower == "no" || lower == "0" {
			return false
		}
	}
	return fallback
}

func readBool(args map[string]any, key string) (bool, bool) {
	value, ok := args[key]
	if !ok {
		return false, false
	}
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		lower := strings.ToLower(strings.TrimSpace(typed))
		switch lower {
		case "true", "yes", "1":
			return true, true
		case "false", "no", "0":
			return false, true
		}
	}
	return false, false
}

type expectedDeltas struct {
	name        string
	characterID string
	hopeDelta   *int
	stressDelta *int
}

func (r *Runner) captureExpectedDeltas(
	ctx context.Context,
	state *scenarioState,
	args map[string]any,
	fallbackName string,
) (*expectedDeltas, *daggerheartv1.DaggerheartCharacterState, error) {
	hopeDelta, hopeOk := readInt(args, "expect_hope_delta")
	stressDelta, stressOk := readInt(args, "expect_stress_delta")
	if !hopeOk && !stressOk {
		return nil, nil, nil
	}
	name := optionalString(args, "expect_target", fallbackName)
	if strings.TrimSpace(name) == "" {
		return nil, nil, r.failf("expect_*_delta requires expect_target or a default character")
	}
	characterID, err := actorID(state, name)
	if err != nil {
		return nil, nil, err
	}
	before, err := r.getCharacterState(ctx, state, characterID)
	if err != nil {
		return nil, nil, err
	}
	spec := &expectedDeltas{name: name, characterID: characterID}
	if hopeOk {
		spec.hopeDelta = &hopeDelta
	}
	if stressOk {
		spec.stressDelta = &stressDelta
	}
	return spec, before, nil
}

func (r *Runner) assertExpectedDeltas(
	ctx context.Context,
	state *scenarioState,
	spec *expectedDeltas,
	before *daggerheartv1.DaggerheartCharacterState,
) error {
	if spec == nil || before == nil {
		return nil
	}
	after, err := r.getCharacterState(ctx, state, spec.characterID)
	if err != nil {
		return err
	}
	if spec.hopeDelta != nil {
		delta := int(after.GetHope()) - int(before.GetHope())
		if delta != *spec.hopeDelta {
			if err := r.assertf("hope delta for %s = %d, want %d", spec.name, delta, *spec.hopeDelta); err != nil {
				return err
			}
		}
	}
	if spec.stressDelta != nil {
		delta := int(after.GetStress()) - int(before.GetStress())
		if delta != *spec.stressDelta {
			if err := r.assertf("stress delta for %s = %d, want %d", spec.name, delta, *spec.stressDelta); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) assertExpectedSpotlight(ctx context.Context, state *scenarioState, args map[string]any) error {
	expected := strings.ToLower(strings.TrimSpace(optionalString(args, "expect_spotlight", "")))
	if expected == "" {
		return nil
	}
	if state.sessionID == "" {
		return r.failf("expect_spotlight requires an active session")
	}
	request := &gamev1.GetSessionSpotlightRequest{
		CampaignId: state.campaignID,
		SessionId:  state.sessionID,
	}
	if expected == "none" {
		if _, err := r.env.sessionClient.GetSessionSpotlight(ctx, request); err == nil {
			return r.failf("expected no session spotlight")
		}
		return nil
	}
	response, err := r.env.sessionClient.GetSessionSpotlight(ctx, request)
	if err != nil {
		return fmt.Errorf("get session spotlight: %w", err)
	}
	spotlight := response.GetSpotlight()
	if spotlight == nil {
		return r.failf("expected session spotlight")
	}
	if expected == "gm" {
		if spotlight.GetType() != gamev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM {
			return r.failf("spotlight type = %v, want GM", spotlight.GetType())
		}
		if spotlight.GetCharacterId() != "" {
			return r.failf("spotlight character id = %q, want empty", spotlight.GetCharacterId())
		}
		return nil
	}
	characterID, err := actorID(state, expected)
	if err != nil {
		return err
	}
	if spotlight.GetType() != gamev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER {
		return r.failf("spotlight type = %v, want CHARACTER", spotlight.GetType())
	}
	if spotlight.GetCharacterId() != characterID {
		return r.failf("spotlight character id = %q, want %q", spotlight.GetCharacterId(), characterID)
	}
	return nil
}

type damageFlagExpect struct {
	resistPhysical *bool
	resistMagic    *bool
	immunePhysical *bool
	immuneMagic    *bool
}

func readDamageFlagExpect(args map[string]any) (damageFlagExpect, bool) {
	expect := damageFlagExpect{}
	if value, ok := readBool(args, "resist_physical"); ok {
		expect.resistPhysical = &value
	}
	if value, ok := readBool(args, "resist_magic"); ok {
		expect.resistMagic = &value
	}
	if value, ok := readBool(args, "immune_physical"); ok {
		expect.immunePhysical = &value
	}
	if value, ok := readBool(args, "immune_magic"); ok {
		expect.immuneMagic = &value
	}
	if expect.resistPhysical == nil && expect.resistMagic == nil && expect.immunePhysical == nil && expect.immuneMagic == nil {
		return damageFlagExpect{}, false
	}
	return expect, true
}

func (r *Runner) assertDamageFlags(
	ctx context.Context,
	state *scenarioState,
	before uint64,
	targetID string,
	args map[string]any,
) error {
	expect, ok := readDamageFlagExpect(args)
	if !ok {
		return nil
	}
	filter := fmt.Sprintf("type = \"%s\"", daggerheart.EventTypeDamageApplied)
	if state.sessionID != "" {
		filter = filter + fmt.Sprintf(" AND session_id = \"%s\"", state.sessionID)
	}
	response, err := r.env.eventClient.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: state.campaignID,
		PageSize:   20,
		OrderBy:    "seq desc",
		Filter:     filter,
	})
	if err != nil {
		return fmt.Errorf("list damage events: %w", err)
	}
	var payload daggerheart.DamageAppliedPayload
	for _, evt := range response.GetEvents() {
		if evt.GetSeq() <= before {
			continue
		}
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return fmt.Errorf("decode damage payload: %w", err)
		}
		if targetID != "" && payload.CharacterID != targetID {
			continue
		}
		if expect.resistPhysical != nil && payload.ResistPhysical != *expect.resistPhysical {
			return r.assertf("resist_physical = %v, want %v", payload.ResistPhysical, *expect.resistPhysical)
		}
		if expect.resistMagic != nil && payload.ResistMagic != *expect.resistMagic {
			return r.assertf("resist_magic = %v, want %v", payload.ResistMagic, *expect.resistMagic)
		}
		if expect.immunePhysical != nil && payload.ImmunePhysical != *expect.immunePhysical {
			return r.assertf("immune_physical = %v, want %v", payload.ImmunePhysical, *expect.immunePhysical)
		}
		if expect.immuneMagic != nil && payload.ImmuneMagic != *expect.immuneMagic {
			return r.assertf("immune_magic = %v, want %v", payload.ImmuneMagic, *expect.immuneMagic)
		}
		return nil
	}
	return r.assertf("expected damage_applied after seq %d", before)
}

func isHopeSpendSource(source string) bool {
	normalized := normalizeModifierSource(source)
	switch normalized {
	case "experience", "help", "tag_team", "hope_feature":
		return true
	default:
		return false
	}
}

func normalizeModifierSource(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "_", "-", "_")
	return replacer.Replace(strings.ToLower(trimmed))
}

func readStringSlice(args map[string]any, key string) []string {
	value, ok := args[key]
	if !ok {
		return nil
	}
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	results := make([]string, 0, len(list))
	for _, entry := range list {
		text, ok := entry.(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(text)
		if trimmed != "" {
			results = append(results, trimmed)
		}
	}
	return results
}

func uniqueNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func resolveCharacterList(state *scenarioState, args map[string]any, key string) ([]string, error) {
	list := readStringSlice(args, key)
	if len(list) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(list))
	for _, name := range list {
		id, err := actorID(state, name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func allActorIDs(state *scenarioState) []string {
	if len(state.actors) == 0 {
		return nil
	}
	names := make([]string, 0, len(state.actors))
	for name := range state.actors {
		names = append(names, name)
	}
	sort.Strings(names)
	ids := make([]string, 0, len(names))
	for _, name := range names {
		ids = append(ids, state.actors[name])
	}
	return ids
}

func parseRestType(value string) (daggerheartv1.DaggerheartRestType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "short":
		return daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT, nil
	case "long":
		return daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG, nil
	default:
		return daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_UNSPECIFIED, fmt.Errorf("unsupported rest type %q", value)
	}
}

func parseCountdownKind(value string) (daggerheartv1.DaggerheartCountdownKind, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "progress":
		return daggerheartv1.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS, nil
	case "consequence":
		return daggerheartv1.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE, nil
	case "loop", "long_term":
		return daggerheartv1.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS, nil
	default:
		return daggerheartv1.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_UNSPECIFIED, fmt.Errorf("unsupported countdown kind %q", value)
	}
}

func parseCountdownDirection(value string) (daggerheartv1.DaggerheartCountdownDirection, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "increase":
		return daggerheartv1.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE, nil
	case "decrease":
		return daggerheartv1.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_DECREASE, nil
	default:
		return daggerheartv1.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_UNSPECIFIED, fmt.Errorf("unsupported countdown direction %q", value)
	}
}

func parseDowntimeMove(value string) (daggerheartv1.DaggerheartDowntimeMove, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "clear_all_stress":
		return daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS, nil
	case "repair_all_armor":
		return daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_REPAIR_ALL_ARMOR, nil
	case "prepare":
		return daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_PREPARE, nil
	case "work_on_project":
		return daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_WORK_ON_PROJECT, nil
	default:
		return daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_UNSPECIFIED, fmt.Errorf("unsupported downtime move %q", value)
	}
}

func parseDeathMove(value string) (daggerheartv1.DaggerheartDeathMove, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "blaze_of_glory":
		return daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_BLAZE_OF_GLORY, nil
	case "avoid_death":
		return daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH, nil
	case "risk_it_all":
		return daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL, nil
	default:
		return daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED, fmt.Errorf("unsupported death move %q", value)
	}
}

func parseLifeState(value string) (daggerheartv1.DaggerheartLifeState, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "alive":
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE, nil
	case "unconscious":
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS, nil
	case "blaze_of_glory":
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY, nil
	case "dead":
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD, nil
	default:
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED, fmt.Errorf("unsupported life_state %q", value)
	}
}

func withUserID(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, grpcmeta.UserIDHeader, userID)
}

func withSessionID(ctx context.Context, sessionID string) context.Context {
	if sessionID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, grpcmeta.SessionIDHeader, sessionID)
}

func withCampaignID(ctx context.Context, campaignID string) context.Context {
	if campaignID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, grpcmeta.CampaignIDHeader, campaignID)
}

func withParticipantID(ctx context.Context, participantID string) context.Context {
	if participantID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, grpcmeta.ParticipantIDHeader, participantID)
}
