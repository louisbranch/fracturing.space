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
		return r.failf("interaction_set_gm_authority participant is required")
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
		return fmt.Errorf("interaction_set_gm_authority: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionSetActiveSceneStep(ctx context.Context, state *scenarioState, step Step) error {
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
	_, err = client.SetActiveScene(ctx, &gamev1.SetActiveSceneRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		return fmt.Errorf("interaction_set_active_scene: %w", err)
	}
	state.activeSceneID = sceneID
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
	frameText := strings.TrimSpace(optionalString(step.Args, "frame_text", optionalString(step.Args, "frame", "")))
	if frameText == "" {
		return r.failf("interaction_start_player_phase frame_text is required")
	}
	characterIDs, err := resolveCharacterList(state, step.Args, "characters")
	if err != nil {
		return err
	}
	if len(characterIDs) == 0 {
		return r.failf("interaction_start_player_phase characters are required")
	}
	_, err = client.StartScenePlayerPhase(ctx, &gamev1.StartScenePlayerPhaseRequest{
		CampaignId:   state.campaignID,
		SceneId:      sceneID,
		FrameText:    frameText,
		CharacterIds: characterIDs,
	})
	if err != nil {
		return fmt.Errorf("interaction_start_player_phase: %w", err)
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
		return r.failf("interaction_post summary is required")
	}
	characterIDs, err := resolveCharacterList(state, step.Args, "characters")
	if err != nil {
		return err
	}
	yieldAfterPost := optionalBool(step.Args, "yield", optionalBool(step.Args, "yield_after_post", false))
	_, err = client.SubmitScenePlayerPost(ctx, &gamev1.SubmitScenePlayerPostRequest{
		CampaignId:     state.campaignID,
		SceneId:        sceneID,
		SummaryText:    summaryText,
		CharacterIds:   characterIDs,
		YieldAfterPost: yieldAfterPost,
	})
	if err != nil {
		return fmt.Errorf("interaction_post: %w", err)
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
		return fmt.Errorf("interaction_yield: %w", err)
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
	_, err = client.UnyieldScenePlayerPhase(ctx, &gamev1.UnyieldScenePlayerPhaseRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		return fmt.Errorf("interaction_unyield: %w", err)
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
	_, err = client.EndScenePlayerPhase(ctx, &gamev1.EndScenePlayerPhaseRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
		Reason:     reason,
	})
	if err != nil {
		return fmt.Errorf("interaction_end_player_phase: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionAcceptPlayerPhaseStep(ctx context.Context, state *scenarioState, step Step) error {
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
	_, err = client.AcceptScenePlayerPhase(ctx, &gamev1.AcceptScenePlayerPhaseRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		return fmt.Errorf("interaction_accept_player_phase: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionRequestRevisionsStep(ctx context.Context, state *scenarioState, step Step) error {
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
	revisions, err := parseScenePlayerRevisionRequests(state, step.Args, "revisions")
	if err != nil {
		return err
	}
	if len(revisions) == 0 {
		return r.failf("interaction_request_revisions revisions are required")
	}
	_, err = client.RequestScenePlayerRevisions(ctx, &gamev1.RequestScenePlayerRevisionsRequest{
		CampaignId: state.campaignID,
		SceneId:    sceneID,
		Revisions:  revisions,
	})
	if err != nil {
		return fmt.Errorf("interaction_request_revisions: %w", err)
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
	_, err = client.PauseSessionForOOC(ctx, &gamev1.PauseSessionForOOCRequest{
		CampaignId: state.campaignID,
		Reason:     strings.TrimSpace(optionalString(step.Args, "reason", "")),
	})
	if err != nil {
		return fmt.Errorf("interaction_pause_ooc: %w", err)
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
		return r.failf("interaction_post_ooc body is required")
	}
	_, err = client.PostSessionOOC(ctx, &gamev1.PostSessionOOCRequest{
		CampaignId: state.campaignID,
		Body:       body,
	})
	if err != nil {
		return fmt.Errorf("interaction_post_ooc: %w", err)
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
		return fmt.Errorf("interaction_ready_ooc: %w", err)
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
		return fmt.Errorf("interaction_clear_ready_ooc: %w", err)
	}
	return nil
}

func (r *Runner) runInteractionResumeOOCStep(ctx context.Context, state *scenarioState) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	client, err := r.requireInteractionClient()
	if err != nil {
		return err
	}
	_, err = client.ResumeFromOOC(ctx, &gamev1.ResumeFromOOCRequest{CampaignId: state.campaignID})
	if err != nil {
		return fmt.Errorf("interaction_resume_ooc: %w", err)
	}
	return nil
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
	if expectedFrame, ok := step.Args["frame_text"]; ok {
		if strings.TrimSpace(fmt.Sprint(expectedFrame)) != strings.TrimSpace(stateProto.GetPlayerPhase().GetFrameText()) {
			return r.assertf("interaction frame_text = %q, want %q", stateProto.GetPlayerPhase().GetFrameText(), fmt.Sprint(expectedFrame))
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
