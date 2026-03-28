package interactiontransport

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ConcludeSessionResult struct {
	SessionID         string
	EndedSceneIDs     []string
	CampaignCompleted bool
}

var requiredRecapHeadings = []string{
	"## Key Events",
	"## NPCs Met",
	"## Decisions Made",
	"## Unresolved Threads",
	"## Next Session Hooks",
}

var recapHeadingPattern = regexp.MustCompile(`(?m)^## [^\n]+$`)

func (a AIOrchestrationApplication) ConcludeSession(
	ctx context.Context,
	campaignID, sessionID, conclusion, summary string,
	endCampaign bool,
	epilogue string,
) (ConcludeSessionResult, error) {
	conclusion = strings.TrimSpace(conclusion)
	if conclusion == "" {
		return ConcludeSessionResult{}, status.Error(codes.InvalidArgument, "conclusion is required")
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return ConcludeSessionResult{}, status.Error(codes.InvalidArgument, "summary is required")
	}
	epilogue = strings.TrimSpace(epilogue)
	switch {
	case endCampaign && epilogue == "":
		return ConcludeSessionResult{}, status.Error(codes.InvalidArgument, "epilogue is required when end_campaign is true")
	case !endCampaign && epilogue != "":
		return ConcludeSessionResult{}, status.Error(codes.InvalidArgument, "epilogue is only allowed when end_campaign is true")
	}
	if err := validateSessionRecapSummary(summary); err != nil {
		return ConcludeSessionResult{}, err
	}

	sessionRecord, err := a.interaction.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return ConcludeSessionResult{}, grpcerror.OptionalLookupErrorContext(ctx, err, "load conclude session target")
	}
	if sessionRecord.Status == session.StatusEnded {
		if err := a.ensureCampaignCompletedIfRequested(ctx, campaignID, endCampaign); err != nil {
			return ConcludeSessionResult{}, err
		}
		return a.loadConcludedSessionResult(ctx, campaignID, sessionID)
	}

	activeSession, sessionInteraction, err := a.interaction.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return ConcludeSessionResult{}, err
	}
	if activeSession.ID != sessionID {
		return ConcludeSessionResult{}, status.Error(codes.FailedPrecondition, "session is not the active session")
	}

	conclusionScene, sceneInteraction, openScenes, err := a.resolveConclusionScene(ctx, campaignID, activeSession.ID, sessionInteraction)
	if err != nil {
		return ConcludeSessionResult{}, err
	}
	gmParticipantID := strings.TrimSpace(sessionInteraction.GMAuthorityParticipantID)
	if gmParticipantID == "" {
		actorID, actorType := handler.ResolveCommandActor(ctx)
		if actorType == command.ActorTypeParticipant {
			gmParticipantID = strings.TrimSpace(actorID)
		}
	}
	if gmParticipantID == "" {
		return ConcludeSessionResult{}, status.Error(codes.FailedPrecondition, "session gm authority is not assigned")
	}

	interactionPayload, err := a.interaction.buildGMInteractionPayload(&campaignv1.GMInteractionInput{
		Title: "Session Conclusion",
		Beats: []*campaignv1.GMInteractionInputBeat{
			{
				BeatId: "conclusion",
				Type:   campaignv1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_FICTION,
				Text:   conclusion,
			},
			{
				BeatId: "close",
				Type:   campaignv1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_GUIDANCE,
				Text:   "Session close. Carry forward the recap, unresolved threads, and consequences into the next session.",
			},
		},
	}, ids.SceneID(conclusionScene.SceneID), sceneInteraction.PhaseID, ids.ParticipantID(gmParticipantID))
	if err != nil {
		return ConcludeSessionResult{}, err
	}
	if err := a.interaction.executeSceneCommand(ctx, commandTypeSceneGMInteractionCommit, campaignID, activeSession.ID, conclusionScene.SceneID, interactionPayload, "scene.gm_interaction.commit"); err != nil {
		return ConcludeSessionResult{}, err
	}

	recapMarkdown := buildSessionRecapMarkdown(summary, epilogue)
	if err := a.interaction.executeSessionCommand(ctx, commandTypeSessionRecapRecord, campaignID, activeSession.ID, session.RecapRecordedPayload{
		SessionID: ids.SessionID(activeSession.ID),
		Markdown:  recapMarkdown,
	}, "session.recap.record"); err != nil {
		return ConcludeSessionResult{}, err
	}

	for _, openScene := range openScenes {
		if err := a.executeCampaignSceneEnd(ctx, campaignID, activeSession.ID, openScene.SceneID, "session_concluded"); err != nil {
			return ConcludeSessionResult{}, err
		}
	}

	if err := a.interaction.executeSessionCommand(ctx, commandTypeSessionEnd, campaignID, activeSession.ID, session.EndPayload{
		SessionID: ids.SessionID(activeSession.ID),
	}, "session.end"); err != nil {
		return ConcludeSessionResult{}, err
	}

	if err := a.ensureCampaignCompletedIfRequested(ctx, campaignID, endCampaign); err != nil {
		return ConcludeSessionResult{}, err
	}

	return a.loadConcludedSessionResult(ctx, campaignID, activeSession.ID)
}

func validateSessionRecapSummary(summary string) error {
	searchFrom := 0
	for _, heading := range requiredRecapHeadings {
		idx := strings.Index(summary[searchFrom:], heading)
		if idx < 0 {
			return status.Errorf(codes.InvalidArgument, "summary must contain heading %q", heading)
		}
		searchFrom += idx + len(heading)
	}
	headings := recapHeadingPattern.FindAllString(summary, -1)
	if len(headings) < len(requiredRecapHeadings) {
		return status.Error(codes.InvalidArgument, "summary must include all required headings")
	}
	for idx, heading := range requiredRecapHeadings {
		if headings[idx] != heading {
			return status.Error(codes.InvalidArgument, "summary headings must appear in the required order")
		}
	}
	return nil
}

func buildSessionRecapMarkdown(summary, epilogue string) string {
	if strings.TrimSpace(epilogue) == "" {
		return summary
	}
	return summary + "\n\n## Campaign Epilogue\n\n" + epilogue
}

func (a AIOrchestrationApplication) resolveConclusionScene(
	ctx context.Context,
	campaignID, sessionID string,
	sessionInteraction storage.SessionInteraction,
) (storage.SceneRecord, storage.SceneInteraction, []storage.SceneRecord, error) {
	openScenes, err := a.interaction.stores.Scene.ListOpenScenes(ctx, campaignID)
	if err != nil {
		return storage.SceneRecord{}, storage.SceneInteraction{}, nil, grpcerror.Internal("list open scenes", err)
	}
	sessionScenes := make([]storage.SceneRecord, 0, len(openScenes))
	for _, openScene := range openScenes {
		if strings.TrimSpace(openScene.SessionID) == sessionID {
			sessionScenes = append(sessionScenes, openScene)
		}
	}
	if len(sessionScenes) == 0 {
		return storage.SceneRecord{}, storage.SceneInteraction{}, nil, status.Error(codes.FailedPrecondition, "active session has no open scene for session conclusion")
	}
	sort.SliceStable(sessionScenes, func(i, j int) bool {
		if !sessionScenes[i].UpdatedAt.Equal(sessionScenes[j].UpdatedAt) {
			return sessionScenes[i].UpdatedAt.After(sessionScenes[j].UpdatedAt)
		}
		return sessionScenes[i].SceneID < sessionScenes[j].SceneID
	})

	sceneID := strings.TrimSpace(sessionInteraction.ActiveSceneID)
	if sceneID == "" || !containsOpenScene(sessionScenes, sceneID) {
		sceneID = sessionScenes[0].SceneID
	}
	selected, err := a.interaction.stores.Scene.GetScene(ctx, campaignID, sceneID)
	if err != nil {
		return storage.SceneRecord{}, storage.SceneInteraction{}, nil, grpcerror.OptionalLookupErrorContext(ctx, err, "load conclusion scene")
	}
	sceneInteraction, err := a.interaction.stores.SceneInteraction.GetSceneInteraction(ctx, campaignID, sceneID)
	if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load conclusion scene interaction"); lookupErr != nil {
		return storage.SceneRecord{}, storage.SceneInteraction{}, nil, lookupErr
	}
	if err != nil {
		sceneInteraction = storage.SceneInteraction{
			CampaignID: selected.CampaignID,
			SceneID:    selected.SceneID,
			SessionID:  selected.SessionID,
		}
	}
	return selected, sceneInteraction, sessionScenes, nil
}

func containsOpenScene(scenes []storage.SceneRecord, sceneID string) bool {
	for _, scene := range scenes {
		if scene.SceneID == sceneID {
			return true
		}
	}
	return false
}

func (a AIOrchestrationApplication) executeCampaignSceneEnd(ctx context.Context, campaignID, sessionID, sceneID, reason string) error {
	payload := scene.EndPayload{
		SceneID: ids.SceneID(sceneID),
		Reason:  strings.TrimSpace(reason),
	}
	return a.executeCampaignCommand(ctx, commandTypeSceneEnd, campaignID, sessionID, sceneID, "scene", sceneID, payload, "scene.end")
}

func (a AIOrchestrationApplication) executeCampaignCommand(
	ctx context.Context,
	commandType command.Type,
	campaignID, sessionID, sceneID, entityType, entityID string,
	payload any,
	label string,
) error {
	var payloadJSON []byte
	var err error
	if payload != nil {
		payloadJSON, err = json.Marshal(payload)
		if err != nil {
			return grpcerror.Internal("encode payload", err)
		}
	}
	actorID, actorType := handler.ResolveCommandActor(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		a.interaction.write,
		a.interaction.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandType,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    strings.TrimSpace(sessionID),
			SceneID:      strings.TrimSpace(sceneID),
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   entityType,
			EntityID:     entityID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents(label+" did not emit an event"),
	)
	return err
}

func (a AIOrchestrationApplication) ensureCampaignCompletedIfRequested(ctx context.Context, campaignID string, requested bool) error {
	if !requested {
		return nil
	}
	campaignRecord, err := a.interaction.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return grpcerror.OptionalLookupErrorContext(ctx, err, "load concluded session campaign")
	}
	if campaignRecord.Status != campaign.StatusCompleted {
		if err := a.executeCampaignCommand(ctx, commandTypeCampaignEnd, campaignID, "", "", "campaign", campaignID, nil, "campaign.end"); err != nil {
			return err
		}
		campaignRecord, err = a.interaction.stores.Campaign.Get(ctx, campaignID)
		if err != nil {
			return grpcerror.OptionalLookupErrorContext(ctx, err, "load completed campaign after conclude")
		}
	}
	if campaignRecord.Status != campaign.StatusCompleted {
		return status.Error(codes.Internal, "conclude session did not complete campaign")
	}
	return nil
}

func (a AIOrchestrationApplication) loadConcludedSessionResult(ctx context.Context, campaignID, sessionID string) (ConcludeSessionResult, error) {
	if _, err := a.interaction.stores.SessionRecap.GetSessionRecap(ctx, campaignID, sessionID); err != nil {
		return ConcludeSessionResult{}, grpcerror.OptionalLookupErrorContext(ctx, err, "load concluded session recap")
	}
	page, err := a.interaction.stores.Scene.ListScenes(ctx, campaignID, sessionID, 200, "")
	if err != nil {
		return ConcludeSessionResult{}, grpcerror.Internal("list concluded session scenes", err)
	}
	endedSceneIDs := make([]string, 0, len(page.Scenes))
	for _, scene := range page.Scenes {
		if scene.EndedAt != nil {
			endedSceneIDs = append(endedSceneIDs, scene.SceneID)
		}
	}
	campaignRecord, err := a.interaction.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return ConcludeSessionResult{}, grpcerror.OptionalLookupErrorContext(ctx, err, "load concluded session campaign")
	}
	return ConcludeSessionResult{
		SessionID:         sessionID,
		EndedSceneIDs:     endedSceneIDs,
		CampaignCompleted: campaignRecord.Status == campaign.StatusCompleted,
	}, nil
}
