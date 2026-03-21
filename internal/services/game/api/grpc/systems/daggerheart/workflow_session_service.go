package daggerheart

import (
	"context"
	"encoding/json"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) requireSessionFlowStores() error {
	switch {
	case s.stores.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case s.stores.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case s.stores.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case s.stores.Content == nil:
		return status.Error(codes.Internal, "daggerheart content store is not configured")
	case s.stores.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case s.seedFunc == nil:
		return status.Error(codes.Internal, "seed generator is not configured")
	case s.stores.Write.Executor == nil:
		return status.Error(codes.Internal, "domain engine is not configured")
	default:
		return nil
	}
}

func (s *DaggerheartService) requireSessionAdversaryFlowStores() error {
	switch {
	case s.stores.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case s.stores.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case s.stores.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case s.stores.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case s.stores.Content == nil:
		return status.Error(codes.Internal, "daggerheart content store is not configured")
	case s.seedFunc == nil:
		return status.Error(codes.Internal, "seed generator is not configured")
	case s.stores.Write.Executor == nil:
		return status.Error(codes.Internal, "domain engine is not configured")
	default:
		return nil
	}
}

func (s *DaggerheartService) sessionFlowHandler() *sessionflowtransport.Handler {
	rolls := s.sessionRollHandler()
	return sessionflowtransport.NewHandler(sessionflowtransport.Dependencies{
		SessionActionRoll:           rolls.SessionActionRoll,
		SessionDamageRoll:           rolls.SessionDamageRoll,
		SessionAdversaryAttackRoll:  rolls.SessionAdversaryAttackRoll,
		ApplyRollOutcome:            s.outcomeHandler().ApplyRollOutcome,
		ApplyAttackOutcome:          s.outcomeHandler().ApplyAttackOutcome,
		ApplyReactionOutcome:        s.outcomeHandler().ApplyReactionOutcome,
		ApplyAdversaryAttackOutcome: s.outcomeHandler().ApplyAdversaryAttackOutcome,
		ApplyDamage:                 s.ApplyDamage,
		ApplyAdversaryDamage:        s.ApplyAdversaryDamage,
		LoadCharacterProfile: func(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
			return s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
		},
		LoadCharacterState: func(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
			return s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
		},
		LoadAdversary: func(ctx context.Context, campaignID, adversaryID, sessionID string) (projectionstore.DaggerheartAdversary, error) {
			adversary, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
			if err != nil {
				return projectionstore.DaggerheartAdversary{}, err
			}
			if adversary.SessionID != sessionID {
				return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
			}
			return adversary, nil
		},
		LoadAdversaryEntry:         s.stores.Content.GetDaggerheartAdversaryEntry,
		LoadSubclass:               s.stores.Content.GetDaggerheartSubclass,
		LoadArmor:                  s.stores.Content.GetDaggerheartArmor,
		ExecuteCharacterStatePatch: s.executeSessionFlowCharacterStatePatch,
		ExecuteAdversaryUpdate:     s.executeSessionFlowAdversaryUpdate,
		AdjustGMFear:               s.executeSessionFlowGMFearAdjust,
		SeedFunc:                   s.seedFunc,
	})
}

func (s *DaggerheartService) executeSessionFlowCharacterStatePatch(ctx context.Context, in sessionflowtransport.CharacterStatePatchInput) error {
	runtime := workflowwrite.NewRuntime(s.stores.Write, s.stores.Event, s.stores.Daggerheart)
	payloadJSON, err := json.Marshal(daggerheart.CharacterStatePatchPayload{
		CharacterID:                         ids.CharacterID(in.CharacterID),
		Source:                              in.Source,
		HopeBefore:                          in.HopeBefore,
		HopeAfter:                           in.HopeAfter,
		ArmorBefore:                         in.ArmorBefore,
		ArmorAfter:                          in.ArmorAfter,
		ClassStateBefore:                    in.ClassStateBefore,
		ClassStateAfter:                     in.ClassStateAfter,
		SubclassStateBefore:                 in.SubclassStateBefore,
		SubclassStateAfter:                  in.SubclassStateAfter,
		ImpenetrableUsedThisShortRestBefore: in.ImpenetrableUsedThisShortRestBefore,
		ImpenetrableUsedThisShortRestAfter:  in.ImpenetrableUsedThisShortRestAfter,
	})
	if err != nil {
		return err
	}
	return runtime.ExecuteSystemCommand(ctx, workflowruntime.SystemCommandInput{
		CampaignID:      in.CampaignID,
		CommandType:     commandids.DaggerheartCharacterStatePatch,
		SessionID:       in.SessionID,
		SceneID:         in.SceneID,
		RequestID:       in.RequestID,
		InvocationID:    in.InvocationID,
		EntityType:      "character",
		EntityID:        in.CharacterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "character state patch did not emit an event",
		ApplyErrMessage: "execute domain command",
	})
}

func (s *DaggerheartService) executeSessionFlowAdversaryUpdate(ctx context.Context, in sessionflowtransport.AdversaryUpdateInput) error {
	payloadJSON, err := json.Marshal(daggerheart.AdversaryUpdatePayload{
		AdversaryID:      ids.AdversaryID(in.Adversary.AdversaryID),
		AdversaryEntryID: in.Adversary.AdversaryEntryID,
		Name:             in.Adversary.Name,
		Kind:             in.Adversary.Kind,
		SessionID:        ids.SessionID(in.Adversary.SessionID),
		SceneID:          ids.SceneID(in.Adversary.SceneID),
		Notes:            in.Adversary.Notes,
		HP:               in.Adversary.HP,
		HPMax:            in.Adversary.HPMax,
		Stress:           in.UpdatedStress,
		StressMax:        in.Adversary.StressMax,
		Evasion:          in.Adversary.Evasion,
		Major:            in.Adversary.Major,
		Severe:           in.Adversary.Severe,
		Armor:            in.Adversary.Armor,
		FeatureStates: func() []daggerheart.AdversaryFeatureState {
			source := in.UpdatedFeatureStates
			if source == nil {
				source = in.Adversary.FeatureStates
			}
			out := make([]daggerheart.AdversaryFeatureState, 0, len(source))
			for _, state := range source {
				out = append(out, daggerheart.AdversaryFeatureState{
					FeatureID:       state.FeatureID,
					Status:          state.Status,
					FocusedTargetID: state.FocusedTargetID,
				})
			}
			return out
		}(),
		PendingExperience: func() *daggerheart.AdversaryPendingExperience {
			if in.ClearPendingExperience {
				return nil
			}
			source := in.UpdatedPendingExperience
			if source == nil {
				source = in.Adversary.PendingExperience
			}
			if source == nil {
				return nil
			}
			return &daggerheart.AdversaryPendingExperience{
				Name:     source.Name,
				Modifier: source.Modifier,
			}
		}(),
		SpotlightGateID: ids.GateID(in.Adversary.SpotlightGateID),
		SpotlightCount:  in.Adversary.SpotlightCount,
	})
	if err != nil {
		return err
	}
	runtime := workflowwrite.NewRuntime(s.stores.Write, s.stores.Event, s.stores.Daggerheart)
	return runtime.ExecuteSystemCommand(ctx, workflowruntime.SystemCommandInput{
		CampaignID:      in.CampaignID,
		CommandType:     commandids.DaggerheartAdversaryUpdate,
		SessionID:       in.SessionID,
		SceneID:         in.SceneID,
		RequestID:       in.RequestID,
		InvocationID:    in.InvocationID,
		EntityType:      "adversary",
		EntityID:        in.Adversary.AdversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary update did not emit an event",
		ApplyErrMessage: "execute domain command",
	})
}

func (s *DaggerheartService) executeSessionFlowGMFearAdjust(ctx context.Context, in sessionflowtransport.GMFearAdjustInput) error {
	snapshot, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, in.CampaignID)
	if err != nil {
		return err
	}
	nextFear := snapshot.GMFear + in.Delta
	if nextFear < daggerheart.GMFearMin {
		nextFear = daggerheart.GMFearMin
	}
	if nextFear > daggerheart.GMFearMax {
		nextFear = daggerheart.GMFearMax
	}
	if nextFear == snapshot.GMFear {
		return nil
	}
	runtime := workflowwrite.NewRuntime(s.stores.Write, s.stores.Event, s.stores.Daggerheart)
	payloadJSON, err := json.Marshal(daggerheart.GMFearSetPayload{
		After:  &nextFear,
		Reason: in.Reason,
	})
	if err != nil {
		return err
	}
	return runtime.ExecuteSystemCommand(ctx, workflowruntime.SystemCommandInput{
		CampaignID:      in.CampaignID,
		CommandType:     commandids.DaggerheartGMFearSet,
		SessionID:       in.SessionID,
		SceneID:         in.SceneID,
		RequestID:       in.RequestID,
		InvocationID:    in.InvocationID,
		EntityType:      "campaign",
		EntityID:        in.CampaignID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "gm fear adjust did not emit an event",
		ApplyErrMessage: "execute domain command",
	})
}

func (s *DaggerheartService) SessionActionRoll(ctx context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
	return s.sessionRollHandler().SessionActionRoll(ctx, in)
}

func (s *DaggerheartService) SessionDamageRoll(ctx context.Context, in *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
	return s.sessionRollHandler().SessionDamageRoll(ctx, in)
}

func (s *DaggerheartService) SessionAttackFlow(ctx context.Context, in *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session attack flow request is required")
	}
	if err := s.requireSessionFlowStores(); err != nil {
		return nil, err
	}
	return s.sessionFlowHandler().SessionAttackFlow(ctx, in)
}

func (s *DaggerheartService) SessionReactionFlow(ctx context.Context, in *pb.SessionReactionFlowRequest) (*pb.SessionReactionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session reaction flow request is required")
	}
	if err := s.requireSessionFlowStores(); err != nil {
		return nil, err
	}
	return s.sessionFlowHandler().SessionReactionFlow(ctx, in)
}

func (s *DaggerheartService) SessionAdversaryAttackRoll(ctx context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
	return s.sessionRollHandler().SessionAdversaryAttackRoll(ctx, in)
}

func (s *DaggerheartService) SessionAdversaryActionCheck(ctx context.Context, in *pb.SessionAdversaryActionCheckRequest) (*pb.SessionAdversaryActionCheckResponse, error) {
	return s.sessionRollHandler().SessionAdversaryActionCheck(ctx, in)
}

func (s *DaggerheartService) SessionAdversaryAttackFlow(ctx context.Context, in *pb.SessionAdversaryAttackFlowRequest) (*pb.SessionAdversaryAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack flow request is required")
	}
	if err := s.requireSessionAdversaryFlowStores(); err != nil {
		return nil, err
	}
	return s.sessionFlowHandler().SessionAdversaryAttackFlow(ctx, in)
}

func (s *DaggerheartService) SessionGroupActionFlow(ctx context.Context, in *pb.SessionGroupActionFlowRequest) (*pb.SessionGroupActionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session group action flow request is required")
	}
	if err := s.requireSessionFlowStores(); err != nil {
		return nil, err
	}
	return s.sessionFlowHandler().SessionGroupActionFlow(ctx, in)
}

func (s *DaggerheartService) SessionTagTeamFlow(ctx context.Context, in *pb.SessionTagTeamFlowRequest) (*pb.SessionTagTeamFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session tag team flow request is required")
	}
	if err := s.requireSessionFlowStores(); err != nil {
		return nil, err
	}
	return s.sessionFlowHandler().SessionTagTeamFlow(ctx, in)
}
