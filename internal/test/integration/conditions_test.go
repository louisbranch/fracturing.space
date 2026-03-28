//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type conditionChangedPayload struct {
	CharacterID string                  `json:"character_id"`
	Added       []conditionStatePayload `json:"added,omitempty"`
	Removed     []conditionStatePayload `json:"removed,omitempty"`
}

type adversaryConditionChangedPayload struct {
	AdversaryID string                  `json:"adversary_id"`
	Added       []conditionStatePayload `json:"added,omitempty"`
	Removed     []conditionStatePayload `json:"removed,omitempty"`
}

type conditionStatePayload struct {
	Code     string `json:"code,omitempty"`
	Standard string `json:"standard,omitempty"`
}

func TestDaggerheartApplyConditions(t *testing.T) {
	grpcAddr, authAddr, stopServer := startGRPCServer(t)
	defer stopServer()

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	defer conn.Close()

	campaignClient := gamev1.NewCampaignServiceClient(conn)
	characterClient := gamev1.NewCharacterServiceClient(conn)
	participantClient := gamev1.NewParticipantServiceClient(conn)
	sessionClient := gamev1.NewSessionServiceClient(conn)
	eventClient := gamev1.NewEventServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()
	userID := createAuthUser(t, authAddr, "conditions-gm")
	ctx = withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:        "Conditions Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      gamev1.GmMode_HUMAN,
		ThemePrompt: "conditions",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()
	ownerParticipantID := createCampaign.GetOwnerParticipant().GetId()

	characterID := createCharacter(t, ctx, characterClient, campaignID, "Condition Hero")
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, characterID)
	ensureSessionStartReadiness(t, ctx, participantClient, characterClient, campaignID, ownerParticipantID, characterID)

	startSession := startSessionWithDefaultControllers(t, ctx, sessionClient, characterClient, campaignID, "Condition Session")
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()
	sessionCtx := withSessionID(ctx, sessionID)

	addResp, err := daggerheartClient.ApplyConditions(sessionCtx, &daggerheartv1.DaggerheartApplyConditionsRequest{
		CampaignId:    campaignID,
		CharacterId:   characterID,
		AddConditions: []*daggerheartv1.DaggerheartConditionState{protoStandardConditionState(daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)},
		Source:        "test",
	})
	if err != nil {
		t.Fatalf("apply conditions add: %v", err)
	}
	if addResp.GetState() == nil {
		t.Fatal("expected state in add response")
	}
	if !hasCondition(addResp.GetState().GetConditionStates(), daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN) {
		t.Fatal("expected hidden condition after add")
	}

	changeResp, err := daggerheartClient.ApplyConditions(sessionCtx, &daggerheartv1.DaggerheartApplyConditionsRequest{
		CampaignId:         campaignID,
		CharacterId:        characterID,
		AddConditions:      []*daggerheartv1.DaggerheartConditionState{protoStandardConditionState(daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)},
		RemoveConditionIds: []string{conditionCode(daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)},
		Source:             "test",
	})
	if err != nil {
		t.Fatalf("apply conditions change: %v", err)
	}
	if changeResp.GetState() == nil {
		t.Fatal("expected state in change response")
	}
	if hasCondition(changeResp.GetState().GetConditionStates(), daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN) {
		t.Fatal("expected hidden condition cleared")
	}
	if !hasCondition(changeResp.GetState().GetConditionStates(), daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE) {
		t.Fatal("expected vulnerable condition after change")
	}

	if err := findConditionChange(ctx, eventClient, campaignID, characterID, []string{"vulnerable"}, []string{"hidden"}); err != nil {
		t.Fatalf("find condition change event: %v", err)
	}
}

func TestDaggerheartApplyAdversaryConditions(t *testing.T) {
	grpcAddr, authAddr, stopServer := startGRPCServer(t)
	defer stopServer()

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	defer conn.Close()

	campaignClient := gamev1.NewCampaignServiceClient(conn)
	characterClient := gamev1.NewCharacterServiceClient(conn)
	participantClient := gamev1.NewParticipantServiceClient(conn)
	sessionClient := gamev1.NewSessionServiceClient(conn)
	sceneClient := gamev1.NewSceneServiceClient(conn)
	eventClient := gamev1.NewEventServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()
	userID := createAuthUser(t, authAddr, "adversary-conditions-gm")
	ctx = withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:        "Adversary Conditions Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      gamev1.GmMode_HUMAN,
		ThemePrompt: "conditions",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()
	ownerParticipantID := createCampaign.GetOwnerParticipant().GetId()
	sceneAnchor := createCharacter(t, ctx, characterClient, campaignID, "Adversary Condition Anchor")
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, sceneAnchor)
	ensureSessionStartReadiness(t, ctx, participantClient, characterClient, campaignID, ownerParticipantID, sceneAnchor)

	startSession := startSessionWithDefaultControllers(t, ctx, sessionClient, characterClient, campaignID, "Condition Session")
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()
	sessionCtx := withSessionID(ctx, sessionID)
	createScene, err := sceneClient.CreateScene(ctx, &gamev1.CreateSceneRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		Name:       "Condition Scene",
		CharacterIds: []string{
			sceneAnchor,
		},
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	sceneID := createScene.GetSceneId()
	if sceneID == "" {
		t.Fatal("expected scene")
	}

	createAdversary, err := daggerheartClient.CreateAdversary(ctx, &daggerheartv1.DaggerheartCreateAdversaryRequest{
		CampaignId:       campaignID,
		SessionId:        sessionID,
		SceneId:          sceneID,
		AdversaryEntryId: "adversary.integration-foe",
	})
	if err != nil {
		t.Fatalf("create adversary: %v", err)
	}
	if createAdversary.GetAdversary() == nil {
		t.Fatal("expected adversary")
	}
	adversaryID := createAdversary.GetAdversary().GetId()

	addResp, err := daggerheartClient.ApplyAdversaryConditions(sessionCtx, &daggerheartv1.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:    campaignID,
		AdversaryId:   adversaryID,
		AddConditions: []*daggerheartv1.DaggerheartConditionState{protoStandardConditionState(daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)},
		Source:        "test",
	})
	if err != nil {
		t.Fatalf("apply adversary conditions add: %v", err)
	}
	if addResp.GetAdversary() == nil {
		t.Fatal("expected adversary in add response")
	}
	if !hasCondition(addResp.GetAdversary().GetConditionStates(), daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN) {
		t.Fatal("expected hidden condition after add")
	}

	changeResp, err := daggerheartClient.ApplyAdversaryConditions(sessionCtx, &daggerheartv1.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:         campaignID,
		AdversaryId:        adversaryID,
		AddConditions:      []*daggerheartv1.DaggerheartConditionState{protoStandardConditionState(daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)},
		RemoveConditionIds: []string{conditionCode(daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)},
		Source:             "test",
	})
	if err != nil {
		t.Fatalf("apply adversary conditions change: %v", err)
	}
	if changeResp.GetAdversary() == nil {
		t.Fatal("expected adversary in change response")
	}
	if hasCondition(changeResp.GetAdversary().GetConditionStates(), daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN) {
		t.Fatal("expected hidden condition cleared")
	}
	if !hasCondition(changeResp.GetAdversary().GetConditionStates(), daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE) {
		t.Fatal("expected vulnerable condition after change")
	}

	if err := findAdversaryConditionChange(ctx, eventClient, campaignID, adversaryID, []string{"vulnerable"}, []string{"hidden"}); err != nil {
		t.Fatalf("find adversary condition change event: %v", err)
	}
}

func hasCondition(conditions []*daggerheartv1.DaggerheartConditionState, target daggerheartv1.DaggerheartCondition) bool {
	for _, condition := range conditions {
		if condition != nil && condition.GetStandard() == target {
			return true
		}
	}
	return false
}

func findConditionChange(ctx context.Context, client gamev1.EventServiceClient, campaignID, characterID string, added, removed []string) error {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "entity_id = \"" + characterID + "\" AND type = \"sys.daggerheart.condition_changed\"",
	})
	if err != nil {
		return err
	}
	for _, evt := range response.GetEvents() {
		var payload conditionChangedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return err
		}
		if conditionPayloadCodesEqual(payload.Added, added) && conditionPayloadCodesEqual(payload.Removed, removed) {
			return nil
		}
	}
	return fmt.Errorf("matching condition change event not found")
}

func findAdversaryConditionChange(ctx context.Context, client gamev1.EventServiceClient, campaignID, adversaryID string, added, removed []string) error {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "entity_id = \"" + adversaryID + "\" AND type = \"sys.daggerheart.adversary_condition_changed\"",
	})
	if err != nil {
		return err
	}
	for _, evt := range response.GetEvents() {
		var payload adversaryConditionChangedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return err
		}
		if conditionPayloadCodesEqual(payload.Added, added) && conditionPayloadCodesEqual(payload.Removed, removed) {
			return nil
		}
	}
	return fmt.Errorf("matching adversary condition change event not found")
}

func protoStandardConditionState(condition daggerheartv1.DaggerheartCondition) *daggerheartv1.DaggerheartConditionState {
	code := conditionCode(condition)
	return &daggerheartv1.DaggerheartConditionState{
		Id:       code,
		Code:     code,
		Label:    code,
		Class:    daggerheartv1.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD,
		Standard: condition,
	}
}

func conditionCode(condition daggerheartv1.DaggerheartCondition) string {
	switch condition {
	case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN:
		return "hidden"
	case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED:
		return "restrained"
	case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE:
		return "vulnerable"
	case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_CLOAKED:
		return "cloaked"
	default:
		return ""
	}
}

func conditionPayloadCodesEqual(states []conditionStatePayload, expected []string) bool {
	actual := make([]string, 0, len(states))
	for _, state := range states {
		code := state.Code
		if code == "" {
			code = state.Standard
		}
		if code != "" {
			actual = append(actual, code)
		}
	}
	return stringSliceEqual(actual, expected)
}

func stringSliceEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]int, len(left))
	for _, value := range left {
		seen[value]++
	}
	for _, value := range right {
		seen[value]--
		if seen[value] < 0 {
			return false
		}
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}
