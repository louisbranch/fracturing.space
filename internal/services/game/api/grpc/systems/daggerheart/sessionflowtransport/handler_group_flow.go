package sessionflowtransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) SessionGroupActionFlow(ctx context.Context, in *pb.SessionGroupActionFlowRequest) (*pb.SessionGroupActionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session group action flow request is required")
	}
	if h.deps.SessionActionRoll == nil || h.deps.ApplyRollOutcome == nil {
		return nil, status.Error(codes.Internal, "session workflow dependencies are not configured")
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
	leaderID, err := validate.RequiredID(in.GetLeaderCharacterId(), "leader character id")
	if err != nil {
		return nil, err
	}
	leaderTrait, err := validate.RequiredID(in.GetLeaderTrait(), "leader trait")
	if err != nil {
		return nil, err
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
		supporterID, err := validate.RequiredID(supporter.GetCharacterId(), "supporter character id")
		if err != nil {
			return nil, err
		}
		supporterTrait, err := validate.RequiredID(supporter.GetTrait(), "supporter trait")
		if err != nil {
			return nil, err
		}

		rollResp, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
			CampaignId:  campaignID,
			SessionId:   sessionID,
			SceneId:     sceneID,
			CharacterId: supporterID,
			Trait:       supporterTrait,
			RollKind:    pb.RollKind_ROLL_KIND_REACTION,
			Difficulty:  in.GetDifficulty(),
			Modifiers:   supporter.GetModifiers(),
			Context:     supporter.GetContext(),
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

	leaderRoll, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CharacterId: leaderID,
		Trait:       leaderTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   leaderModifiers,
		HopeSpends:  in.GetLeaderHopeSpends(),
		Context:     in.GetLeaderContext(),
		Rng:         in.GetLeaderRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	leaderOutcome, err := h.deps.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
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

// SessionTagTeamFlow runs the tag-team orchestration by resolving both action
// rolls and then applying the chosen result to the combined targets.
func (h *Handler) SessionTagTeamFlow(ctx context.Context, in *pb.SessionTagTeamFlowRequest) (*pb.SessionTagTeamFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session tag team flow request is required")
	}
	if h.deps.SessionActionRoll == nil || h.deps.ApplyRollOutcome == nil {
		return nil, status.Error(codes.Internal, "session workflow dependencies are not configured")
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
	firstID, err := validate.RequiredID(first.GetCharacterId(), "first character id")
	if err != nil {
		return nil, err
	}
	secondID, err := validate.RequiredID(second.GetCharacterId(), "second character id")
	if err != nil {
		return nil, err
	}
	if firstID == secondID {
		return nil, status.Error(codes.InvalidArgument, "tag team participants must be distinct")
	}
	firstTrait, err := validate.RequiredID(first.GetTrait(), "first trait")
	if err != nil {
		return nil, err
	}
	secondTrait, err := validate.RequiredID(second.GetTrait(), "second trait")
	if err != nil {
		return nil, err
	}
	selectedID, err := validate.RequiredID(in.GetSelectedCharacterId(), "selected character id")
	if err != nil {
		return nil, err
	}
	if selectedID != firstID && selectedID != secondID {
		return nil, status.Error(codes.InvalidArgument, "selected character id must match a participant")
	}

	firstRoll, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CharacterId: firstID,
		Trait:       firstTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   first.GetModifiers(),
		HopeSpends:  first.GetHopeSpends(),
		Rng:         first.GetRng(),
	})
	if err != nil {
		return nil, err
	}
	secondRoll, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CharacterId: secondID,
		Trait:       secondTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   second.GetModifiers(),
		HopeSpends:  second.GetHopeSpends(),
		Rng:         second.GetRng(),
	})
	if err != nil {
		return nil, err
	}

	selectedRoll := firstRoll
	if selectedID == secondID {
		selectedRoll = secondRoll
	}

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	selectedOutcome, err := h.deps.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   selectedRoll.GetRollSeq(),
		Targets:   []string{firstID, secondID},
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

// SessionAdversaryAttackFlow runs the adversary attack orchestration by
