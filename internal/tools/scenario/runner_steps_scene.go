package scenario

import (
	"context"
	"fmt"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
)

func (r *Runner) runCreateSceneStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	name := requiredString(step.Args, "name")
	if name == "" {
		return r.failf("create_scene name is required")
	}

	request := &gamev1.CreateSceneRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		Name:        name,
		Description: optionalString(step.Args, "description", ""),
	}
	if activate, ok := readBool(step.Args, "activate"); ok {
		request.Activate = &activate
	}

	charNames := readStringSlice(step.Args, "characters")
	for _, charName := range charNames {
		id, err := actorID(state, charName)
		if err != nil {
			return fmt.Errorf("create_scene character %q: %w", charName, err)
		}
		request.CharacterIds = append(request.CharacterIds, id)
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}

	resp, err := r.env.sceneClient.CreateScene(ctx, request)
	if err != nil {
		return fmt.Errorf("create scene: %w", err)
	}

	sceneID := resp.GetSceneId()
	state.scenes[name] = sceneID
	if activeSceneID := resp.GetInteractionState().GetActiveScene().GetSceneId(); activeSceneID != "" {
		state.activeSceneID = activeSceneID
		r.logf("scene created: name=%s id=%s (active)", name, sceneID)
	} else {
		r.logf("scene created: name=%s id=%s", name, sceneID)
	}

	return r.requireEventTypesAfterSeq(ctx, state, before, scene.EventTypeCreated)
}

func (r *Runner) runEndSceneStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	name := requiredString(step.Args, "name")
	if name == "" {
		return r.failf("end_scene name is required")
	}
	sceneID, ok := state.scenes[name]
	if !ok {
		return r.failf("end_scene: unknown scene %q", name)
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}

	_, err = r.env.sceneClient.EndScene(ctx, &gamev1.EndSceneRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
		Reason:     optionalString(step.Args, "reason", ""),
	})
	if err != nil {
		return fmt.Errorf("end scene: %w", err)
	}

	if state.activeSceneID == sceneID {
		state.activeSceneID = ""
	}
	r.logf("scene ended: name=%s id=%s", name, sceneID)
	return r.requireEventTypesAfterSeq(ctx, state, before, scene.EventTypeEnded)
}

func (r *Runner) runSceneAddCharacterStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	sceneName := requiredString(step.Args, "scene")
	if sceneName == "" {
		return r.failf("scene_add_character scene is required")
	}
	sceneID, ok := state.scenes[sceneName]
	if !ok {
		return r.failf("scene_add_character: unknown scene %q", sceneName)
	}

	charName := requiredString(step.Args, "character")
	if charName == "" {
		return r.failf("scene_add_character character is required")
	}
	characterID, err := actorID(state, charName)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}

	_, err = r.env.sceneClient.AddCharacterToScene(ctx, &gamev1.AddCharacterToSceneRequest{
		CampaignId:  state.campaignID,
		SceneId:     sceneID,
		CharacterId: characterID,
	})
	if err != nil {
		return fmt.Errorf("add character to scene: %w", err)
	}

	r.logf("character %s added to scene %s", charName, sceneName)
	return r.requireEventTypesAfterSeq(ctx, state, before, scene.EventTypeCharacterAdded)
}

func (r *Runner) runSceneRemoveCharacterStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	sceneName := requiredString(step.Args, "scene")
	if sceneName == "" {
		return r.failf("scene_remove_character scene is required")
	}
	sceneID, ok := state.scenes[sceneName]
	if !ok {
		return r.failf("scene_remove_character: unknown scene %q", sceneName)
	}

	charName := requiredString(step.Args, "character")
	if charName == "" {
		return r.failf("scene_remove_character character is required")
	}
	characterID, err := actorID(state, charName)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}

	_, err = r.env.sceneClient.RemoveCharacterFromScene(ctx, &gamev1.RemoveCharacterFromSceneRequest{
		CampaignId:  state.campaignID,
		SceneId:     sceneID,
		CharacterId: characterID,
	})
	if err != nil {
		return fmt.Errorf("remove character from scene: %w", err)
	}

	r.logf("character %s removed from scene %s", charName, sceneName)
	return r.requireEventTypesAfterSeq(ctx, state, before, scene.EventTypeCharacterRemoved)
}

func (r *Runner) runSceneTransferCharacterStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	fromName := requiredString(step.Args, "from_scene")
	if fromName == "" {
		return r.failf("scene_transfer_character from_scene is required")
	}
	fromID, ok := state.scenes[fromName]
	if !ok {
		return r.failf("scene_transfer_character: unknown scene %q", fromName)
	}

	toName := requiredString(step.Args, "to_scene")
	if toName == "" {
		return r.failf("scene_transfer_character to_scene is required")
	}
	toID, ok := state.scenes[toName]
	if !ok {
		return r.failf("scene_transfer_character: unknown scene %q", toName)
	}

	charName := requiredString(step.Args, "character")
	if charName == "" {
		return r.failf("scene_transfer_character character is required")
	}
	characterID, err := actorID(state, charName)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}

	_, err = r.env.sceneClient.TransferCharacter(ctx, &gamev1.TransferCharacterRequest{
		CampaignId:    state.campaignID,
		SourceSceneId: fromID,
		TargetSceneId: toID,
		CharacterId:   characterID,
	})
	if err != nil {
		return fmt.Errorf("transfer character: %w", err)
	}

	r.logf("character %s transferred from %s to %s", charName, fromName, toName)
	// Transfer emits a remove + add pair.
	return r.requireEventTypesAfterSeq(ctx, state, before,
		scene.EventTypeCharacterRemoved, scene.EventTypeCharacterAdded)
}

func (r *Runner) runSceneTransitionStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	sceneName := requiredString(step.Args, "scene")
	if sceneName == "" {
		return r.failf("scene_transition scene is required")
	}
	sourceID, ok := state.scenes[sceneName]
	if !ok {
		return r.failf("scene_transition: unknown scene %q", sceneName)
	}

	newName := requiredString(step.Args, "name")
	if newName == "" {
		return r.failf("scene_transition name is required")
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}

	resp, err := r.env.sceneClient.TransitionScene(ctx, &gamev1.TransitionSceneRequest{
		CampaignId:    state.campaignID,
		SourceSceneId: sourceID,
		Name:          newName,
		Description:   optionalString(step.Args, "description", ""),
	})
	if err != nil {
		return fmt.Errorf("transition scene: %w", err)
	}

	newSceneID := resp.GetNewSceneId()
	state.scenes[newName] = newSceneID
	state.activeSceneID = newSceneID
	r.logf("scene transitioned: %s -> %s (id=%s, active)", sceneName, newName, newSceneID)

	// Transition emits: scene.created (new) + character_added per char + scene.ended (source).
	return r.requireEventTypesAfterSeq(ctx, state, before, scene.EventTypeCreated, scene.EventTypeEnded)
}

func (r *Runner) runSceneGateOpenStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	sceneName := requiredString(step.Args, "scene")
	if sceneName == "" {
		return r.failf("scene_gate_open scene is required")
	}
	sceneID, ok := state.scenes[sceneName]
	if !ok {
		return r.failf("scene_gate_open: unknown scene %q", sceneName)
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}

	_, err = r.env.sceneClient.OpenSceneGate(ctx, &gamev1.OpenSceneGateRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
		GateType:   optionalString(step.Args, "gate_type", "decision"),
		Reason:     optionalString(step.Args, "reason", ""),
		GateId:     optionalString(step.Args, "gate_id", ""),
	})
	if err != nil {
		return fmt.Errorf("open scene gate: %w", err)
	}

	r.logf("scene gate opened: scene=%s", sceneName)
	return r.requireEventTypesAfterSeq(ctx, state, before, scene.EventTypeGateOpened)
}

func (r *Runner) runSceneGateResolveStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	sceneName := requiredString(step.Args, "scene")
	if sceneName == "" {
		return r.failf("scene_gate_resolve scene is required")
	}
	sceneID, ok := state.scenes[sceneName]
	if !ok {
		return r.failf("scene_gate_resolve: unknown scene %q", sceneName)
	}

	gateID := requiredString(step.Args, "gate_id")
	if gateID == "" {
		return r.failf("scene_gate_resolve gate_id is required")
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}

	_, err = r.env.sceneClient.ResolveSceneGate(ctx, &gamev1.ResolveSceneGateRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
		GateId:     gateID,
		Decision:   optionalString(step.Args, "decision", ""),
	})
	if err != nil {
		return fmt.Errorf("resolve scene gate: %w", err)
	}

	r.logf("scene gate resolved: scene=%s gate=%s", sceneName, gateID)
	return r.requireEventTypesAfterSeq(ctx, state, before, scene.EventTypeGateResolved)
}

func (r *Runner) runSceneGateAbandonStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	sceneName := requiredString(step.Args, "scene")
	if sceneName == "" {
		return r.failf("scene_gate_abandon scene is required")
	}
	sceneID, ok := state.scenes[sceneName]
	if !ok {
		return r.failf("scene_gate_abandon: unknown scene %q", sceneName)
	}

	gateID := requiredString(step.Args, "gate_id")
	if gateID == "" {
		return r.failf("scene_gate_abandon gate_id is required")
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}

	_, err = r.env.sceneClient.AbandonSceneGate(ctx, &gamev1.AbandonSceneGateRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
		GateId:     gateID,
		Reason:     optionalString(step.Args, "reason", ""),
	})
	if err != nil {
		return fmt.Errorf("abandon scene gate: %w", err)
	}

	r.logf("scene gate abandoned: scene=%s gate=%s", sceneName, gateID)
	return r.requireEventTypesAfterSeq(ctx, state, before, scene.EventTypeGateAbandoned)
}

func (r *Runner) runSceneSetSpotlightStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	sceneName := requiredString(step.Args, "scene")
	if sceneName == "" {
		return r.failf("scene_set_spotlight scene is required")
	}
	sceneID, ok := state.scenes[sceneName]
	if !ok {
		return r.failf("scene_set_spotlight: unknown scene %q", sceneName)
	}

	spotlightType := optionalString(step.Args, "type", "character")
	request := &gamev1.SetSceneSpotlightRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
		Type:       spotlightType,
	}

	if spotlightType == "character" {
		target := requiredString(step.Args, "target")
		if target == "" {
			return r.failf("scene_set_spotlight character requires target")
		}
		characterID, err := actorID(state, target)
		if err != nil {
			return err
		}
		request.CharacterId = characterID
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}

	_, err = r.env.sceneClient.SetSceneSpotlight(ctx, request)
	if err != nil {
		return fmt.Errorf("set scene spotlight: %w", err)
	}

	r.logf("scene spotlight set: scene=%s type=%s", sceneName, spotlightType)
	return r.requireEventTypesAfterSeq(ctx, state, before, scene.EventTypeSpotlightSet)
}

func (r *Runner) runSceneClearSpotlightStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	sceneName := requiredString(step.Args, "scene")
	if sceneName == "" {
		return r.failf("scene_clear_spotlight scene is required")
	}
	sceneID, ok := state.scenes[sceneName]
	if !ok {
		return r.failf("scene_clear_spotlight: unknown scene %q", sceneName)
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}

	_, err = r.env.sceneClient.ClearSceneSpotlight(ctx, &gamev1.ClearSceneSpotlightRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
		Reason:     optionalString(step.Args, "reason", ""),
	})
	if err != nil {
		return fmt.Errorf("clear scene spotlight: %w", err)
	}

	r.logf("scene spotlight cleared: scene=%s", sceneName)
	return r.requireEventTypesAfterSeq(ctx, state, before, scene.EventTypeSpotlightCleared)
}

func (r *Runner) runUpdateSceneStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	name := requiredString(step.Args, "name")
	if name == "" {
		return r.failf("update_scene name is required")
	}
	sceneID, ok := state.scenes[name]
	if !ok {
		return r.failf("update_scene: unknown scene %q", name)
	}

	newName := optionalString(step.Args, "new_name", "")
	desc := optionalString(step.Args, "description", "")

	request := &gamev1.UpdateSceneRequest{
		CampaignId:  state.campaignID,
		SceneId:     sceneID,
		Name:        newName,
		Description: desc,
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}

	_, err = r.env.sceneClient.UpdateScene(ctx, request)
	if err != nil {
		return fmt.Errorf("update scene: %w", err)
	}

	// If renamed, update the scenes map.
	if newName != "" && newName != name {
		state.scenes[newName] = sceneID
		delete(state.scenes, name)
		r.logf("scene updated: %s -> %s", name, newName)
	} else {
		r.logf("scene updated: %s", name)
	}

	return r.requireEventTypesAfterSeq(ctx, state, before, scene.EventTypeUpdated)
}
