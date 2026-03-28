package daggerheart

import (
	"context"
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/adversarytransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/sessionrolltransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func (s *DaggerheartService) sessionRollHandler() *sessionrolltransport.Handler {
	return sessionrolltransport.NewHandler(sessionrolltransport.Dependencies{
		Campaign:                     s.stores.Campaign,
		Session:                      s.stores.Session,
		SessionGate:                  s.stores.SessionGate,
		Daggerheart:                  s.stores.Daggerheart,
		Content:                      s.stores.Content,
		Event:                        s.stores.Event,
		SeedFunc:                     s.seedFunc,
		ExecuteActionRollResolve:     s.executeSessionRollResolve,
		ExecuteDamageRollResolve:     s.executeSessionRollResolve,
		ExecuteAdversaryRollResolve:  s.executeSessionRollResolve,
		ExecuteHopeSpend:             s.executeSessionHopeSpend,
		ExecuteArmorBackedHopeSpend:  s.executeArmorBackedHopeSpend,
		ExecuteAdversaryFeatureApply: s.executeSessionRollAdversaryFeatureApply,
		AdvanceBreathCountdown:       s.workflowEffectsHandler().AdvanceBreathCountdown,
		LoadAdversaryForSession: func(ctx context.Context, campaignID, sessionID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
			return adversarytransport.LoadAdversaryForSession(ctx, s.stores.Daggerheart, campaignID, sessionID, adversaryID)
		},
	})
}

func (s *DaggerheartService) executeSessionRollResolve(ctx context.Context, in sessionrolltransport.RollResolveInput) (uint64, error) {
	domainResult, err := s.executeWorkflowCoreCommand(ctx, workflowwrite.CoreCommandInput{
		CampaignID:      in.CampaignID,
		CommandType:     commandTypeActionRollResolve,
		SessionID:       in.SessionID,
		SceneID:         in.SceneID,
		RequestID:       in.RequestID,
		InvocationID:    in.InvocationID,
		EntityType:      in.EntityType,
		EntityID:        in.EntityID,
		PayloadJSON:     in.PayloadJSON,
		MissingEventMsg: in.MissingEventMsg,
		ApplyErrMessage: "execute domain command",
	})
	if err != nil {
		return 0, err
	}
	return domainResult.Decision.Events[0].Seq, nil
}

func (s *DaggerheartService) executeSessionHopeSpend(ctx context.Context, in sessionrolltransport.HopeSpendInput) error {
	payloadJSON, err := json.Marshal(daggerheartpayload.HopeSpendPayload{
		CharacterID: ids.CharacterID(in.CharacterID),
		Amount:      in.Amount,
		Before:      in.HopeBefore,
		After:       in.HopeAfter,
		RollSeq:     &in.RollSeq,
		Source:      in.Source,
	})
	if err != nil {
		return err
	}
	return s.executeWorkflowSystemCommand(ctx, workflowruntime.SystemCommandInput{
		CampaignID:      in.CampaignID,
		CommandType:     commandTypeDaggerheartHopeSpend,
		SessionID:       in.SessionID,
		SceneID:         in.SceneID,
		RequestID:       in.RequestID,
		InvocationID:    in.InvocationID,
		EntityType:      "character",
		EntityID:        in.CharacterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "hope spend did not emit an event",
		ApplyErrMessage: "execute domain command",
	})
}

func (s *DaggerheartService) executeArmorBackedHopeSpend(ctx context.Context, in sessionrolltransport.ArmorBackedHopeSpendInput) error {
	payloadJSON, err := json.Marshal(daggerheartpayload.CharacterStatePatchPayload{
		CharacterID: ids.CharacterID(in.CharacterID),
		Source:      "armor.hopeful",
		ArmorBefore: &in.ArmorBefore,
		ArmorAfter:  &in.ArmorAfter,
	})
	if err != nil {
		return err
	}
	return s.executeWorkflowSystemCommand(ctx, workflowruntime.SystemCommandInput{
		CampaignID:      in.CampaignID,
		CommandType:     commandTypeDaggerheartCharacterStatePatch,
		SessionID:       in.SessionID,
		SceneID:         in.SceneID,
		RequestID:       in.RequestID,
		InvocationID:    in.InvocationID,
		EntityType:      "character",
		EntityID:        in.CharacterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "armor-backed hope spend did not emit an event",
		ApplyErrMessage: "execute domain command",
	})
}

func (s *DaggerheartService) executeSessionRollAdversaryFeatureApply(ctx context.Context, in sessionrolltransport.AdversaryFeatureApplyInput) error {
	payloadJSON, err := json.Marshal(daggerheartpayload.AdversaryFeatureApplyPayload{
		ActorAdversaryID:    dhids.AdversaryID(in.Adversary.AdversaryID),
		AdversaryID:         dhids.AdversaryID(in.Adversary.AdversaryID),
		FeatureID:           in.FeatureID,
		FeatureStatesBefore: nil,
		FeatureStatesAfter:  nil,
		PendingExperienceBefore: func() *rules.AdversaryPendingExperience {
			if in.PendingExperienceBefore == nil {
				return nil
			}
			return &rules.AdversaryPendingExperience{
				Name:     in.PendingExperienceBefore.Name,
				Modifier: in.PendingExperienceBefore.Modifier,
			}
		}(),
		PendingExperienceAfter: func() *rules.AdversaryPendingExperience {
			if in.PendingExperienceAfter == nil {
				return nil
			}
			return &rules.AdversaryPendingExperience{
				Name:     in.PendingExperienceAfter.Name,
				Modifier: in.PendingExperienceAfter.Modifier,
			}
		}(),
	})
	if err != nil {
		return err
	}
	return s.executeWorkflowSystemCommand(ctx, workflowruntime.SystemCommandInput{
		CampaignID:      in.CampaignID,
		CommandType:     commandids.DaggerheartAdversaryFeatureApply,
		SessionID:       in.SessionID,
		SceneID:         in.SceneID,
		RequestID:       in.RequestID,
		InvocationID:    in.InvocationID,
		EntityType:      "adversary",
		EntityID:        in.Adversary.AdversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "session roll adversary feature apply did not emit an event",
		ApplyErrMessage: "execute domain command",
	})
}
