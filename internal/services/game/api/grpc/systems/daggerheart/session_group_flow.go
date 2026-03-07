package daggerheart

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) runSessionGroupActionFlow(ctx context.Context, in *pb.SessionGroupActionFlowRequest) (*pb.SessionGroupActionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session group action flow request is required")
	}
	if err := s.requireDependencies(
		dependencyCampaignStore,
		dependencySessionStore,
		dependencyDaggerheartStore,
		dependencyEventStore,
		dependencySeedGenerator,
	); err != nil {
		return nil, err
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	leaderID := strings.TrimSpace(in.GetLeaderCharacterId())
	if leaderID == "" {
		return nil, status.Error(codes.InvalidArgument, "leader character id is required")
	}
	leaderTrait := strings.TrimSpace(in.GetLeaderTrait())
	if leaderTrait == "" {
		return nil, status.Error(codes.InvalidArgument, "leader trait is required")
	}
	if in.GetDifficulty() == 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty is required")
	}
	supporters := in.GetSupporters()
	if len(supporters) == 0 {
		return nil, status.Error(codes.InvalidArgument, "supporters are required")
	}

	supportRolls := make([]*pb.GroupActionSupporterRoll, 0, len(supporters))
	supportSuccesses := 0
	supportFailures := 0
	for _, supporter := range supporters {
		if supporter == nil {
			return nil, status.Error(codes.InvalidArgument, "supporter is required")
		}
		supporterID := strings.TrimSpace(supporter.GetCharacterId())
		if supporterID == "" {
			return nil, status.Error(codes.InvalidArgument, "supporter character id is required")
		}
		supporterTrait := strings.TrimSpace(supporter.GetTrait())
		if supporterTrait == "" {
			return nil, status.Error(codes.InvalidArgument, "supporter trait is required")
		}

		rollResp, err := s.runSessionActionRoll(ctx, &pb.SessionActionRollRequest{
			CampaignId:  campaignID,
			SessionId:   sessionID,
			SceneId:     sceneID,
			CharacterId: supporterID,
			Trait:       supporterTrait,
			RollKind:    pb.RollKind_ROLL_KIND_REACTION,
			Difficulty:  in.GetDifficulty(),
			Modifiers:   supporter.GetModifiers(),
			Rng:         supporter.GetRng(),
		})
		if err != nil {
			return nil, err
		}
		if rollResp.GetSuccess() {
			supportSuccesses++
		} else {
			supportFailures++
		}

		supportRolls = append(supportRolls, &pb.GroupActionSupporterRoll{
			CharacterId: supporterID,
			ActionRoll:  rollResp,
			Success:     rollResp.GetSuccess(),
		})
	}

	supportModifier := supportSuccesses - supportFailures
	leaderModifiers := append([]*pb.ActionRollModifier{}, in.GetLeaderModifiers()...)
	if supportModifier != 0 {
		leaderModifiers = append(leaderModifiers, &pb.ActionRollModifier{
			Value:  int32(supportModifier),
			Source: "group_action_support",
		})
	}

	leaderRoll, err := s.runSessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CharacterId: leaderID,
		Trait:       leaderTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   leaderModifiers,
		Rng:         in.GetLeaderRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	leaderOutcome, err := s.runApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   leaderRoll.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	return &pb.SessionGroupActionFlowResponse{
		LeaderRoll:       leaderRoll,
		LeaderOutcome:    leaderOutcome,
		SupporterRolls:   supportRolls,
		SupportModifier:  int32(supportModifier),
		SupportSuccesses: int32(supportSuccesses),
		SupportFailures:  int32(supportFailures),
	}, nil
}

func (s *DaggerheartService) runSessionTagTeamFlow(ctx context.Context, in *pb.SessionTagTeamFlowRequest) (*pb.SessionTagTeamFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session tag team flow request is required")
	}
	if err := s.requireDependencies(
		dependencyCampaignStore,
		dependencySessionStore,
		dependencyDaggerheartStore,
		dependencyEventStore,
		dependencySeedGenerator,
	); err != nil {
		return nil, err
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if in.GetDifficulty() == 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty is required")
	}
	first := in.GetFirst()
	if first == nil {
		return nil, status.Error(codes.InvalidArgument, "first participant is required")
	}
	second := in.GetSecond()
	if second == nil {
		return nil, status.Error(codes.InvalidArgument, "second participant is required")
	}
	firstID := strings.TrimSpace(first.GetCharacterId())
	if firstID == "" {
		return nil, status.Error(codes.InvalidArgument, "first character id is required")
	}
	secondID := strings.TrimSpace(second.GetCharacterId())
	if secondID == "" {
		return nil, status.Error(codes.InvalidArgument, "second character id is required")
	}
	if firstID == secondID {
		return nil, status.Error(codes.InvalidArgument, "tag team participants must be distinct")
	}
	firstTrait := strings.TrimSpace(first.GetTrait())
	if firstTrait == "" {
		return nil, status.Error(codes.InvalidArgument, "first trait is required")
	}
	secondTrait := strings.TrimSpace(second.GetTrait())
	if secondTrait == "" {
		return nil, status.Error(codes.InvalidArgument, "second trait is required")
	}
	selectedID := strings.TrimSpace(in.GetSelectedCharacterId())
	if selectedID == "" {
		return nil, status.Error(codes.InvalidArgument, "selected character id is required")
	}
	if selectedID != firstID && selectedID != secondID {
		return nil, status.Error(codes.InvalidArgument, "selected character id must match a participant")
	}

	firstRoll, err := s.runSessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CharacterId: firstID,
		Trait:       firstTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   first.GetModifiers(),
		Rng:         first.GetRng(),
	})
	if err != nil {
		return nil, err
	}

	secondRoll, err := s.runSessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CharacterId: secondID,
		Trait:       secondTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   second.GetModifiers(),
		Rng:         second.GetRng(),
	})
	if err != nil {
		return nil, err
	}

	selectedRoll := firstRoll
	if selectedID == secondID {
		selectedRoll = secondRoll
	}

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	applyTargets := []string{firstID, secondID}
	selectedOutcome, err := s.runApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   selectedRoll.GetRollSeq(),
		Targets:   applyTargets,
	})
	if err != nil {
		return nil, err
	}

	return &pb.SessionTagTeamFlowResponse{
		FirstRoll:           firstRoll,
		SecondRoll:          secondRoll,
		SelectedOutcome:     selectedOutcome,
		SelectedCharacterId: selectedID,
		SelectedRollSeq:     selectedRoll.GetRollSeq(),
	}, nil
}
