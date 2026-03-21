package scenario

import (
	"context"
	"fmt"
	"slices"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

type expectedInteractionSlot struct {
	participantID string
	summaryText   string
	characterIDs  []string
	yielded       bool
	reviewStatus  string
	reviewReason  string
	reviewChars   []string
}

type expectedOOCPost struct {
	participantID string
	body          string
}

// requireInteractionClient fails fast when a test or runner wiring omits the
// interaction surface the scenario step depends on.
func (r *Runner) requireInteractionClient() (gamev1.InteractionServiceClient, error) {
	if r.env.interactionClient == nil {
		return nil, r.failf("interaction client is required")
	}
	return r.env.interactionClient, nil
}

func (r *Runner) runInteractionSetGMAuthorityStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	target := requiredString(step.Args, "participant")
	if target == "" {
		return r.failf("interaction_set_session_gm_authority participant is required")
	}
	participantIDValue, err := participantID(state, target)
	if err != nil {
		return err
	}
	_, err = client.SetSessionGMAuthority(ctx, &gamev1.SetSessionGMAuthorityRequest{
		CampaignId:    state.campaignID,
		ParticipantId: participantIDValue,
	})
	if err != nil {
		return fmt.Errorf("interaction_set_session_gm_authority: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionActivateSceneStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	sceneID, err := resolveInteractionSceneID(state, step.Args, true)
	if err != nil {
		return err
	}
	_, err = client.ActivateScene(ctx, &gamev1.ActivateSceneRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		return fmt.Errorf("interaction_activate_scene: %w", err)
	}
	state.activeSceneID = sceneID
	return nil
}

func (r *Runner) runInteractionRecordGMInteractionStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	sceneID, err := resolveInteractionSceneID(state, step.Args, false)
	if err != nil {
		return err
	}
	interaction, err := scenarioInteractionInputFromArgs(state, step.Args, nil)
	if err != nil {
		return err
	}
	_, err = client.RecordSceneGMInteraction(ctx, &gamev1.RecordSceneGMInteractionRequest{
		CampaignId:  state.campaignID,
		SceneId:     sceneID,
		Interaction: interaction,
	})
	if err != nil {
		return fmt.Errorf("interaction_record_scene_gm_interaction: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionStartPlayerPhaseStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	sceneID, err := resolveInteractionSceneID(state, step.Args, false)
	if err != nil {
		return err
	}
	characterIDs, err := resolveCharacterList(state, step.Args, "characters")
	if err != nil {
		return err
	}
	if len(characterIDs) == 0 {
		return r.failf("interaction_open_scene_player_phase characters are required")
	}
	interaction, err := scenarioInteractionInputFromArgs(state, step.Args, characterIDs)
	if err != nil {
		return err
	}
	_, err = client.OpenScenePlayerPhase(ctx, &gamev1.OpenScenePlayerPhaseRequest{
		CampaignId:   state.campaignID,
		SceneId:      sceneID,
		CharacterIds: characterIDs,
		Interaction:  interaction,
	})
	if err != nil {
		return fmt.Errorf("interaction_open_scene_player_phase: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionPostStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	sceneID, err := resolveInteractionSceneID(state, step.Args, false)
	if err != nil {
		return err
	}
	summaryText := strings.TrimSpace(optionalString(step.Args, "summary", optionalString(step.Args, "summary_text", optionalString(step.Args, "text", ""))))
	if summaryText == "" {
		return r.failf("interaction_submit_scene_player_action summary is required")
	}
	characterIDs, err := resolveCharacterList(state, step.Args, "characters")
	if err != nil {
		return err
	}
	yieldAfterPost := optionalBool(step.Args, "yield", optionalBool(step.Args, "yield", false))
	_, err = client.SubmitScenePlayerAction(ctx, &gamev1.SubmitScenePlayerActionRequest{
		CampaignId:   state.campaignID,
		SceneId:      sceneID,
		SummaryText:  summaryText,
		CharacterIds: characterIDs,
	})
	if err != nil {
		return fmt.Errorf("interaction_submit_scene_player_action: %w", err)
	}
	if yieldAfterPost {
		_, err = client.YieldScenePlayerPhase(ctx, &gamev1.YieldScenePlayerPhaseRequest{
			CampaignId: state.campaignID,
			SceneId:    sceneID,
		})
		if err != nil {
			return fmt.Errorf("interaction_submit_scene_player_action yield: %w", err)
		}
	}
	return nil
}

func (r *Runner) runInteractionYieldStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	sceneID, err := resolveInteractionSceneID(state, step.Args, false)
	if err != nil {
		return err
	}
	_, err = client.YieldScenePlayerPhase(ctx, &gamev1.YieldScenePlayerPhaseRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		return fmt.Errorf("interaction_yield_scene_player_phase: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionUnyieldStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	sceneID, err := resolveInteractionSceneID(state, step.Args, false)
	if err != nil {
		return err
	}
	_, err = client.WithdrawScenePlayerYield(ctx, &gamev1.WithdrawScenePlayerYieldRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		return fmt.Errorf("interaction_withdraw_scene_player_yield: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionEndPlayerPhaseStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	sceneID, err := resolveInteractionSceneID(state, step.Args, false)
	if err != nil {
		return err
	}
	reason := strings.TrimSpace(optionalString(step.Args, "reason", "gm_interrupted"))
	_, err = client.InterruptScenePlayerPhase(ctx, &gamev1.InterruptScenePlayerPhaseRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
		Reason:     reason,
	})
	if err != nil {
		return fmt.Errorf("interaction_interrupt_scene_player_phase: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionResolveReviewStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	sceneID, err := resolveInteractionSceneID(state, step.Args, false)
	if err != nil {
		return err
	}
	req := &gamev1.ResolveScenePlayerReviewRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
	}
	if _, ok := step.Args["revisions"]; ok {
		interaction, err := scenarioInteractionInputFromArgs(state, step.Args, nil)
		if err != nil {
			return err
		}
		revisions, err := parseScenePlayerRevisionRequests(state, step.Args, "revisions")
		if err != nil {
			return err
		}
		if len(revisions) == 0 {
			return r.failf("interaction_resolve_scene_player_review revisions are required")
		}
		req.Resolution = &gamev1.ResolveScenePlayerReviewRequest_RequestRevisions{
			RequestRevisions: &gamev1.ResolveScenePlayerReviewRequestRevisions{
				Interaction: interaction,
				Revisions:   revisions,
			},
		}
	} else if optionalBool(step.Args, "return_to_gm", false) {
		interaction, err := scenarioInteractionInputFromArgs(state, step.Args, nil)
		if err != nil {
			return err
		}
		req.Resolution = &gamev1.ResolveScenePlayerReviewRequest_ReturnToGm{
			ReturnToGm: &gamev1.ResolveScenePlayerReviewReturnToGM{
				Interaction: interaction,
			},
		}
	} else {
		characterIDs, err := resolveCharacterList(state, step.Args, "characters")
		if err != nil {
			return err
		}
		if len(characterIDs) == 0 {
			return r.failf("interaction_resolve_scene_player_review characters are required when advancing to players")
		}
		interaction, err := scenarioInteractionInputFromArgs(state, step.Args, characterIDs)
		if err != nil {
			return err
		}
		req.Resolution = &gamev1.ResolveScenePlayerReviewRequest_OpenNextPlayerPhase{
			OpenNextPlayerPhase: &gamev1.ResolveScenePlayerReviewOpenNextPlayerPhase{
				NextCharacterIds: characterIDs,
				Interaction:      interaction,
			},
		}
	}
	_, err = client.ResolveScenePlayerReview(ctx, req)
	if err != nil {
		return fmt.Errorf("interaction_resolve_scene_player_review: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionPauseOOCStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	_, err = client.OpenSessionOOC(ctx, &gamev1.OpenSessionOOCRequest{
		CampaignId: state.campaignID,
		Reason:     strings.TrimSpace(optionalString(step.Args, "reason", "")),
	})
	if err != nil {
		return fmt.Errorf("interaction_open_session_ooc: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionPostOOCStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	body := strings.TrimSpace(optionalString(step.Args, "body", ""))
	if body == "" {
		return r.failf("interaction_post_session_ooc body is required")
	}
	_, err = client.PostSessionOOC(ctx, &gamev1.PostSessionOOCRequest{
		CampaignId: state.campaignID,
		Body:       body,
	})
	if err != nil {
		return fmt.Errorf("interaction_post_session_ooc: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionReadyOOCStep(ctx context.Context, state *scenarioState) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	_, err = client.MarkOOCReadyToResume(ctx, &gamev1.MarkOOCReadyToResumeRequest{CampaignId: state.campaignID})
	if err != nil {
		return fmt.Errorf("interaction_mark_ooc_ready_to_resume: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionClearReadyOOCStep(ctx context.Context, state *scenarioState) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	_, err = client.ClearOOCReadyToResume(ctx, &gamev1.ClearOOCReadyToResumeRequest{CampaignId: state.campaignID})
	if err != nil {
		return fmt.Errorf("interaction_clear_ooc_ready_to_resume: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionResolveSessionOOCStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	req := &gamev1.ResolveSessionOOCRequest{CampaignId: state.campaignID}
	if optionalBool(step.Args, "resume_interrupted_phase", false) {
		req.Resolution = &gamev1.ResolveSessionOOCRequest_ResumeInterruptedPhase{
			ResumeInterruptedPhase: &gamev1.ResolveSessionOOCResumeInterruptedPhase{},
		}
	} else if optionalBool(step.Args, "return_to_gm", false) {
		sceneID, err := resolveInteractionSceneID(state, step.Args, true)
		if err != nil && !strings.Contains(err.Error(), "required") {
			return err
		}
		req.Resolution = &gamev1.ResolveSessionOOCRequest_ReturnToGm{
			ReturnToGm: &gamev1.ResolveSessionOOCReturnToGM{SceneId: sceneID},
		}
	} else {
		characterIDs, err := resolveCharacterList(state, step.Args, "characters")
		if err != nil {
			return err
		}
		if len(characterIDs) == 0 {
			return r.failf("interaction_resolve_session_ooc characters are required unless resume_interrupted_phase or return_to_gm is true")
		}
		sceneID, err := resolveInteractionSceneID(state, step.Args, false)
		if err != nil {
			return err
		}
		interaction, err := scenarioInteractionInputFromArgs(state, step.Args, characterIDs)
		if err != nil {
			return err
		}
		req.Resolution = &gamev1.ResolveSessionOOCRequest_OpenPlayerPhase{
			OpenPlayerPhase: &gamev1.ResolveSessionOOCOpenPlayerPhase{
				SceneId:          sceneID,
				NextCharacterIds: characterIDs,
				Interaction:      interaction,
			},
		}
	}
	_, err = client.ResolveSessionOOC(ctx, req)
	if err != nil {
		return fmt.Errorf("interaction_resolve_session_ooc: %w", err)
	}
	return nil
}

func scenarioSingleBeatInteraction(title string, beatType gamev1.GMInteractionBeatType, text string, characterIDs ...string) *gamev1.GMInteractionInput {
	return &gamev1.GMInteractionInput{
		Title:        title,
		CharacterIds: append([]string(nil), characterIDs...),
		Beats: []*gamev1.GMInteractionInputBeat{{
			BeatId: "beat-1",
			Type:   beatType,
			Text:   strings.TrimSpace(text),
		}},
	}
}

func currentInteractionPromptText(interaction *gamev1.GMInteraction) string {
	if interaction == nil {
		return ""
	}
	for _, beat := range interaction.GetBeats() {
		if beat == nil {
			continue
		}
		if beat.GetType() == gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT {
			return strings.TrimSpace(beat.GetText())
		}
	}
	return ""
}

func (r *Runner) runInteractionExpectStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	response, err := client.GetInteractionState(ctx, &gamev1.GetInteractionStateRequest{CampaignId: state.campaignID})
	if err != nil {
		return fmt.Errorf("interaction_expect: %w", err)
	}
	if response.GetState() == nil {
		return r.failf("interaction_expect returned empty state")
	}
	stateProto := response.GetState()
	if expectedSession, ok := step.Args["session"]; ok {
		if strings.TrimSpace(fmt.Sprint(expectedSession)) != strings.TrimSpace(stateProto.GetActiveSession().GetName()) {
			return r.assertf("interaction session = %q, want %q", stateProto.GetActiveSession().GetName(), fmt.Sprint(expectedSession))
		}
	}
	if expectedSceneName, ok := step.Args["active_scene"]; ok {
		if strings.TrimSpace(fmt.Sprint(expectedSceneName)) != strings.TrimSpace(stateProto.GetActiveScene().GetName()) {
			return r.assertf("interaction active_scene = %q, want %q", stateProto.GetActiveScene().GetName(), fmt.Sprint(expectedSceneName))
		}
	}
	if expectedStatus, ok := step.Args["phase_status"]; ok {
		actualStatus := normalizeScenePhaseStatus(stateProto.GetPlayerPhase().GetStatus())
		wantStatus := normalizeScenePhaseStatusString(fmt.Sprint(expectedStatus))
		if actualStatus != wantStatus {
			return r.assertf("interaction phase_status = %q, want %q", actualStatus, wantStatus)
		}
	}
	if expectedPrompt, ok := step.Args["prompt"]; ok {
		actualPrompt := currentInteractionPromptText(stateProto.GetActiveScene().GetCurrentInteraction())
		if strings.TrimSpace(fmt.Sprint(expectedPrompt)) != actualPrompt {
			return r.assertf("interaction prompt = %q, want %q", actualPrompt, fmt.Sprint(expectedPrompt))
		}
	}
	if expectedControlMode, ok := step.Args["control_mode"]; ok {
		actualMode := normalizeInteractionControlMode(stateProto.GetControl().GetMode())
		wantMode := normalizeInteractionControlModeString(fmt.Sprint(expectedControlMode))
		if actualMode != wantMode {
			return r.assertf("interaction control_mode = %q, want %q", actualMode, wantMode)
		}
	}
	if _, ok := step.Args["allowed_transitions"]; ok {
		expectedTransitions := normalizeInteractionTransitionStrings(readStringSlice(step.Args, "allowed_transitions"))
		actualTransitions := actualInteractionTransitions(stateProto.GetControl().GetAllowedTransitions())
		slices.Sort(expectedTransitions)
		slices.Sort(actualTransitions)
		if !slices.Equal(actualTransitions, expectedTransitions) {
			return r.assertf("interaction allowed_transitions = %v, want %v", actualTransitions, expectedTransitions)
		}
	}
	if expectedTransition, ok := step.Args["recommended_transition"]; ok {
		actualTransition := normalizeInteractionTransition(stateProto.GetControl().GetRecommendedTransition())
		wantTransition := normalizeInteractionTransitionString(fmt.Sprint(expectedTransition))
		if actualTransition != wantTransition {
			return r.assertf("interaction recommended_transition = %q, want %q", actualTransition, wantTransition)
		}
	}
	if _, ok := step.Args["acting_characters"]; ok {
		expectedIDs, err := resolveCharacterList(state, step.Args, "acting_characters")
		if err != nil {
			return err
		}
		actualIDs := append([]string(nil), stateProto.GetPlayerPhase().GetActingCharacterIds()...)
		slices.Sort(expectedIDs)
		slices.Sort(actualIDs)
		if !slices.Equal(actualIDs, expectedIDs) {
			return r.assertf("interaction acting_characters = %v, want %v", actualIDs, expectedIDs)
		}
	}
	if _, ok := step.Args["acting_participants"]; ok {
		expectedIDs, err := resolveParticipantList(state, step.Args, "acting_participants")
		if err != nil {
			return err
		}
		actualIDs := append([]string(nil), stateProto.GetPlayerPhase().GetActingParticipantIds()...)
		slices.Sort(expectedIDs)
		slices.Sort(actualIDs)
		if !slices.Equal(actualIDs, expectedIDs) {
			return r.assertf("interaction acting_participants = %v, want %v", actualIDs, expectedIDs)
		}
	}
	if _, ok := step.Args["gm_authority"]; ok {
		expectedID, err := participantID(state, requiredString(step.Args, "gm_authority"))
		if err != nil {
			return err
		}
		if stateProto.GetGmAuthorityParticipantId() != expectedID {
			return r.assertf("interaction gm_authority = %q, want %q", stateProto.GetGmAuthorityParticipantId(), expectedID)
		}
	}
	if expectedOpen, ok := readBool(step.Args, "ooc_open"); ok && stateProto.GetOoc().GetOpen() != expectedOpen {
		return r.assertf("interaction ooc_open = %v, want %v", stateProto.GetOoc().GetOpen(), expectedOpen)
	}
	if expectedPending, ok := readBool(step.Args, "ooc_resolution_pending"); ok && stateProto.GetOoc().GetResolutionPending() != expectedPending {
		return r.assertf("interaction ooc_resolution_pending = %v, want %v", stateProto.GetOoc().GetResolutionPending(), expectedPending)
	}
	if _, ok := step.Args["ooc_requested_by"]; ok {
		expectedID, err := participantID(state, requiredString(step.Args, "ooc_requested_by"))
		if err != nil {
			return err
		}
		if stateProto.GetOoc().GetRequestedByParticipantId() != expectedID {
			return r.assertf("interaction ooc_requested_by = %q, want %q", stateProto.GetOoc().GetRequestedByParticipantId(), expectedID)
		}
	}
	if expectedInterruptedScene, ok := step.Args["ooc_interrupted_scene"]; ok {
		expectedSceneID, err := resolveInteractionSceneID(state, map[string]any{"scene": expectedInterruptedScene}, true)
		if err != nil {
			return err
		}
		if stateProto.GetOoc().GetInterruptedSceneId() != expectedSceneID {
			return r.assertf("interaction ooc_interrupted_scene = %q, want %q", stateProto.GetOoc().GetInterruptedSceneId(), expectedSceneID)
		}
	}
	if expectedInterruptedStatus, ok := step.Args["ooc_interrupted_phase_status"]; ok {
		actualStatus := normalizeScenePhaseStatus(stateProto.GetOoc().GetInterruptedPhaseStatus())
		wantStatus := normalizeScenePhaseStatusString(fmt.Sprint(expectedInterruptedStatus))
		if actualStatus != wantStatus {
			return r.assertf("interaction ooc_interrupted_phase_status = %q, want %q", actualStatus, wantStatus)
		}
	}
	if _, ok := step.Args["ooc_ready"]; ok {
		expectedIDs, err := resolveParticipantList(state, step.Args, "ooc_ready")
		if err != nil {
			return err
		}
		actualIDs := append([]string(nil), stateProto.GetOoc().GetReadyToResumeParticipantIds()...)
		slices.Sort(expectedIDs)
		slices.Sort(actualIDs)
		if !slices.Equal(actualIDs, expectedIDs) {
			return r.assertf("interaction ooc_ready = %v, want %v", actualIDs, expectedIDs)
		}
	}
	if _, ok := step.Args["slots"]; ok {
		expectedSlots, err := parseExpectedInteractionSlots(state, step.Args, "slots")
		if err != nil {
			return err
		}
		actualSlots := actualInteractionSlots(stateProto.GetPlayerPhase().GetSlots())
		slices.SortFunc(expectedSlots, compareExpectedInteractionSlots)
		slices.SortFunc(actualSlots, compareExpectedInteractionSlots)
		if !slices.EqualFunc(actualSlots, expectedSlots, equalExpectedInteractionSlot) {
			return r.assertf("interaction slots = %v, want %v", actualSlots, expectedSlots)
		}
	}
	if _, ok := step.Args["ooc_posts"]; ok {
		expectedPosts, err := parseExpectedOOCPosts(state, step.Args, "ooc_posts")
		if err != nil {
			return err
		}
		actualPosts := actualOOCPosts(stateProto.GetOoc().GetPosts())
		if !slices.EqualFunc(actualPosts, expectedPosts, equalExpectedOOCPost) {
			return r.assertf("interaction ooc_posts = %v, want %v", actualPosts, expectedPosts)
		}
	}
	return nil
}

func resolveInteractionSceneID(state *scenarioState, args map[string]any, requireExplicit bool) (string, error) {
	sceneName := strings.TrimSpace(optionalString(args, "scene", ""))
	if sceneName == "" {
		if requireExplicit && strings.TrimSpace(state.activeSceneID) == "" {
			return "", fmt.Errorf("interaction scene is required")
		}
		if strings.TrimSpace(state.activeSceneID) == "" {
			return "", fmt.Errorf("interaction step requires an active scene")
		}
		return state.activeSceneID, nil
	}
	sceneID, ok := state.scenes[sceneName]
	if !ok {
		for key, value := range state.scenes {
			if strings.EqualFold(key, sceneName) {
				sceneID = value
				ok = true
				break
			}
		}
	}
	if !ok {
		return "", fmt.Errorf("unknown scene %q", sceneName)
	}
	return sceneID, nil
}

func resolveParticipantList(state *scenarioState, args map[string]any, key string) ([]string, error) {
	list := readStringSlice(args, key)
	ids := make([]string, 0, len(list))
	for _, name := range list {
		id, err := participantID(state, name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func normalizeScenePhaseStatus(status gamev1.ScenePhaseStatus) string {
	switch status {
	case gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS:
		return "PLAYERS"
	case gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW:
		return "GM_REVIEW"
	case gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM:
		return "GM"
	case gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_UNSPECIFIED:
		return "UNSPECIFIED"
	default:
		name := strings.TrimSpace(status.String())
		if strings.HasPrefix(name, "SCENE_PHASE_STATUS_") {
			name = strings.TrimPrefix(name, "SCENE_PHASE_STATUS_")
		}
		if name == "" {
			return fmt.Sprintf("UNKNOWN_%d", int32(status))
		}
		return name
	}
}

func normalizeScenePhaseStatusString(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PLAYERS", "SCENE_PHASE_STATUS_PLAYERS":
		return "PLAYERS"
	case "GM_REVIEW", "SCENE_PHASE_STATUS_GM_REVIEW":
		return "GM_REVIEW"
	case "GM", "SCENE_PHASE_STATUS_GM":
		return "GM"
	case "UNSPECIFIED", "SCENE_PHASE_STATUS_UNSPECIFIED":
		return "UNSPECIFIED"
	default:
		return strings.ToUpper(strings.TrimSpace(value))
	}
}

func normalizeInteractionControlMode(mode gamev1.InteractionControlMode) string {
	name := strings.TrimSpace(mode.String())
	if strings.HasPrefix(name, "INTERACTION_CONTROL_MODE_") {
		name = strings.TrimPrefix(name, "INTERACTION_CONTROL_MODE_")
	}
	if name == "" {
		return fmt.Sprintf("UNKNOWN_%d", int32(mode))
	}
	return name
}

func normalizeInteractionControlModeString(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	return strings.TrimPrefix(value, "INTERACTION_CONTROL_MODE_")
}

func normalizeInteractionTransition(transition gamev1.InteractionTransition) string {
	name := strings.TrimSpace(transition.String())
	if strings.HasPrefix(name, "INTERACTION_TRANSITION_") {
		name = strings.TrimPrefix(name, "INTERACTION_TRANSITION_")
	}
	if name == "" {
		return fmt.Sprintf("UNKNOWN_%d", int32(transition))
	}
	return name
}

func normalizeInteractionTransitionString(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	return strings.TrimPrefix(value, "INTERACTION_TRANSITION_")
}

func normalizeInteractionTransitionStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, normalizeInteractionTransitionString(value))
	}
	return out
}

func actualInteractionTransitions(transitions []gamev1.InteractionTransition) []string {
	out := make([]string, 0, len(transitions))
	for _, transition := range transitions {
		out = append(out, normalizeInteractionTransition(transition))
	}
	return out
}

func scenarioInteractionInputFromArgs(state *scenarioState, args map[string]any, fallbackCharacterIDs []string) (*gamev1.GMInteractionInput, error) {
	interactionArgs := readMap(args, "interaction")
	if len(interactionArgs) == 0 {
		return nil, fmt.Errorf("interaction is required")
	}
	title := strings.TrimSpace(optionalString(interactionArgs, "title", ""))
	if title == "" {
		return nil, fmt.Errorf("interaction title is required")
	}
	characterIDs, err := resolveCharacterList(state, interactionArgs, "character_ids")
	if err != nil {
		return nil, err
	}
	if len(characterIDs) == 0 {
		characterIDs = append([]string(nil), fallbackCharacterIDs...)
	}
	beatArgs := readMapSlice(interactionArgs, "beats")
	if len(beatArgs) == 0 {
		return nil, fmt.Errorf("interaction beats are required")
	}
	beats := make([]*gamev1.GMInteractionInputBeat, 0, len(beatArgs))
	for idx, beatArg := range beatArgs {
		text := strings.TrimSpace(optionalString(beatArg, "text", ""))
		if text == "" {
			return nil, fmt.Errorf("interaction beats[%d].text is required", idx)
		}
		beatType, err := parseScenarioGMInteractionBeatType(optionalString(beatArg, "type", ""))
		if err != nil {
			return nil, fmt.Errorf("interaction beats[%d].type: %w", idx, err)
		}
		beatID := strings.TrimSpace(optionalString(beatArg, "beat_id", ""))
		if beatID == "" {
			beatID = fmt.Sprintf("beat-%d", idx+1)
		}
		beats = append(beats, &gamev1.GMInteractionInputBeat{
			BeatId: beatID,
			Type:   beatType,
			Text:   text,
		})
	}
	result := &gamev1.GMInteractionInput{
		Title:        title,
		CharacterIds: characterIDs,
		Beats:        beats,
	}
	if illustrationArgs := readMap(interactionArgs, "illustration"); len(illustrationArgs) != 0 {
		imageURL := strings.TrimSpace(optionalString(illustrationArgs, "image_url", ""))
		alt := strings.TrimSpace(optionalString(illustrationArgs, "alt", ""))
		caption := strings.TrimSpace(optionalString(illustrationArgs, "caption", ""))
		if imageURL == "" {
			return nil, fmt.Errorf("interaction illustration image_url is required")
		}
		if alt == "" {
			return nil, fmt.Errorf("interaction illustration alt is required")
		}
		result.Illustration = &gamev1.GMInteractionInputIllustration{
			ImageUrl: imageURL,
			Alt:      alt,
			Caption:  caption,
		}
	}
	return result, nil
}

func parseScenarioGMInteractionBeatType(raw string) (gamev1.GMInteractionBeatType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "fiction":
		return gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_FICTION, nil
	case "prompt":
		return gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT, nil
	case "resolution":
		return gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_RESOLUTION, nil
	case "consequence":
		return gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_CONSEQUENCE, nil
	case "guidance":
		return gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_GUIDANCE, nil
	default:
		return gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_UNSPECIFIED, fmt.Errorf("unsupported beat type %q", raw)
	}
}

func parseExpectedInteractionSlots(state *scenarioState, args map[string]any, key string) ([]expectedInteractionSlot, error) {
	value, ok := args[key]
	if !ok {
		return nil, nil
	}
	list, ok := value.([]any)
	if !ok {
		if emptyTable, tableOK := value.(map[string]any); tableOK && len(emptyTable) == 0 {
			return []expectedInteractionSlot{}, nil
		}
		return nil, fmt.Errorf("%s must be a list", key)
	}
	slots := make([]expectedInteractionSlot, 0, len(list))
	for _, entry := range list {
		item, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s entries must be tables", key)
		}
		participantName := requiredString(item, "participant")
		if participantName == "" {
			return nil, fmt.Errorf("%s participant is required", key)
		}
		participantIDValue, err := participantID(state, participantName)
		if err != nil {
			return nil, err
		}
		characterIDs, err := resolveCharacterList(state, item, "characters")
		if err != nil {
			return nil, err
		}
		slices.Sort(characterIDs)
		reviewCharacterIDs, err := resolveCharacterList(state, item, "review_characters")
		if err != nil {
			return nil, err
		}
		slices.Sort(reviewCharacterIDs)
		slots = append(slots, expectedInteractionSlot{
			participantID: participantIDValue,
			summaryText:   strings.TrimSpace(optionalString(item, "summary", optionalString(item, "summary_text", ""))),
			characterIDs:  characterIDs,
			yielded:       optionalBool(item, "yielded", false),
			reviewStatus:  normalizeScenePlayerSlotReviewStatusString(optionalString(item, "review_status", "OPEN")),
			reviewReason:  strings.TrimSpace(optionalString(item, "review_reason", "")),
			reviewChars:   reviewCharacterIDs,
		})
	}
	return slots, nil
}

func actualInteractionSlots(slots []*gamev1.ScenePlayerSlot) []expectedInteractionSlot {
	result := make([]expectedInteractionSlot, 0, len(slots))
	for _, slot := range slots {
		characterIDs := append([]string(nil), slot.GetCharacterIds()...)
		slices.Sort(characterIDs)
		reviewCharacterIDs := append([]string(nil), slot.GetReviewCharacterIds()...)
		slices.Sort(reviewCharacterIDs)
		result = append(result, expectedInteractionSlot{
			participantID: slot.GetParticipantId(),
			summaryText:   strings.TrimSpace(slot.GetSummaryText()),
			characterIDs:  characterIDs,
			yielded:       slot.GetYielded(),
			reviewStatus:  normalizeScenePlayerSlotReviewStatus(slot.GetReviewStatus()),
			reviewReason:  strings.TrimSpace(slot.GetReviewReason()),
			reviewChars:   reviewCharacterIDs,
		})
	}
	return result
}

func compareExpectedInteractionSlots(left, right expectedInteractionSlot) int {
	if left.participantID < right.participantID {
		return -1
	}
	if left.participantID > right.participantID {
		return 1
	}
	if left.summaryText < right.summaryText {
		return -1
	}
	if left.summaryText > right.summaryText {
		return 1
	}
	if left.yielded != right.yielded {
		if !left.yielded {
			return -1
		}
		return 1
	}
	if left.reviewStatus < right.reviewStatus {
		return -1
	}
	if left.reviewStatus > right.reviewStatus {
		return 1
	}
	if left.reviewReason < right.reviewReason {
		return -1
	}
	if left.reviewReason > right.reviewReason {
		return 1
	}
	if cmp := strings.Compare(strings.Join(left.characterIDs, ","), strings.Join(right.characterIDs, ",")); cmp != 0 {
		return cmp
	}
	return strings.Compare(strings.Join(left.reviewChars, ","), strings.Join(right.reviewChars, ","))
}

func equalExpectedInteractionSlot(left, right expectedInteractionSlot) bool {
	return left.participantID == right.participantID &&
		left.summaryText == right.summaryText &&
		slices.Equal(left.characterIDs, right.characterIDs) &&
		left.yielded == right.yielded &&
		left.reviewStatus == right.reviewStatus &&
		left.reviewReason == right.reviewReason &&
		slices.Equal(left.reviewChars, right.reviewChars)
}

func normalizeScenePlayerSlotReviewStatus(status gamev1.ScenePlayerSlotReviewStatus) string {
	switch status {
	case gamev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_UNDER_REVIEW:
		return "UNDER_REVIEW"
	case gamev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_ACCEPTED:
		return "ACCEPTED"
	case gamev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_CHANGES_REQUESTED:
		return "CHANGES_REQUESTED"
	default:
		return "OPEN"
	}
}

func normalizeScenePlayerSlotReviewStatusString(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "UNDER_REVIEW", "SCENE_PLAYER_SLOT_REVIEW_STATUS_UNDER_REVIEW":
		return "UNDER_REVIEW"
	case "ACCEPTED", "SCENE_PLAYER_SLOT_REVIEW_STATUS_ACCEPTED":
		return "ACCEPTED"
	case "CHANGES_REQUESTED", "SCENE_PLAYER_SLOT_REVIEW_STATUS_CHANGES_REQUESTED":
		return "CHANGES_REQUESTED"
	default:
		return "OPEN"
	}
}

func parseScenePlayerRevisionRequests(state *scenarioState, args map[string]any, key string) ([]*gamev1.ScenePlayerRevisionRequest, error) {
	value, ok := args[key]
	if !ok {
		return nil, nil
	}
	list, ok := value.([]any)
	if !ok {
		if emptyTable, tableOK := value.(map[string]any); tableOK && len(emptyTable) == 0 {
			return []*gamev1.ScenePlayerRevisionRequest{}, nil
		}
		return nil, fmt.Errorf("%s must be a list", key)
	}
	revisions := make([]*gamev1.ScenePlayerRevisionRequest, 0, len(list))
	for _, entry := range list {
		item, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s entries must be tables", key)
		}
		participantName := requiredString(item, "participant")
		if participantName == "" {
			return nil, fmt.Errorf("%s participant is required", key)
		}
		participantIDValue, err := participantID(state, participantName)
		if err != nil {
			return nil, err
		}
		characterIDs, err := resolveCharacterList(state, item, "characters")
		if err != nil {
			return nil, err
		}
		revisions = append(revisions, &gamev1.ScenePlayerRevisionRequest{
			ParticipantId: participantIDValue,
			Reason:        strings.TrimSpace(optionalString(item, "reason", "")),
			CharacterIds:  characterIDs,
		})
	}
	return revisions, nil
}

func parseExpectedOOCPosts(state *scenarioState, args map[string]any, key string) ([]expectedOOCPost, error) {
	value, ok := args[key]
	if !ok {
		return nil, nil
	}
	list, ok := value.([]any)
	if !ok {
		if emptyTable, tableOK := value.(map[string]any); tableOK && len(emptyTable) == 0 {
			return []expectedOOCPost{}, nil
		}
		return nil, fmt.Errorf("%s must be a list", key)
	}
	posts := make([]expectedOOCPost, 0, len(list))
	for _, entry := range list {
		item, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s entries must be tables", key)
		}
		participantName := requiredString(item, "participant")
		if participantName == "" {
			return nil, fmt.Errorf("%s participant is required", key)
		}
		participantIDValue, err := participantID(state, participantName)
		if err != nil {
			return nil, err
		}
		body := strings.TrimSpace(optionalString(item, "body", ""))
		posts = append(posts, expectedOOCPost{
			participantID: participantIDValue,
			body:          body,
		})
	}
	return posts, nil
}

func actualOOCPosts(posts []*gamev1.OOCPost) []expectedOOCPost {
	result := make([]expectedOOCPost, 0, len(posts))
	for _, post := range posts {
		result = append(result, expectedOOCPost{
			participantID: post.GetParticipantId(),
			body:          strings.TrimSpace(post.GetBody()),
		})
	}
	return result
}

func equalExpectedOOCPost(left, right expectedOOCPost) bool {
	return left.participantID == right.participantID && left.body == right.body
}
