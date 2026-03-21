package daggerheart

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func (s *DaggerheartService) characterMutationHandler() *charactermutationtransport.Handler {
	return charactermutationtransport.NewHandler(charactermutationtransport.Dependencies{
		Campaign:    s.stores.Campaign,
		Daggerheart: s.stores.Daggerheart,
		Content:     s.stores.Content,
		ExecuteCharacterCommand: func(ctx context.Context, in charactermutationtransport.CharacterCommandInput) error {
			adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
			_, err := workflowwrite.ExecuteAndApply(ctx, s.stores.Write, adapter, command.Command{
				CampaignID:    ids.CampaignID(in.CampaignID),
				Type:          in.CommandType,
				ActorType:     command.ActorTypeSystem,
				SessionID:     ids.SessionID(strings.TrimSpace(in.SessionID)),
				RequestID:     in.RequestID,
				InvocationID:  in.InvocationID,
				EntityType:    "character",
				EntityID:      in.CharacterID,
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   in.PayloadJSON,
			}, domainwrite.RequireEventsWithDiagnostics(in.MissingEventMsg, in.ApplyErrMessage))
			return err
		},
	})
}

func (s *DaggerheartService) ApplyLevelUp(ctx context.Context, in *pb.DaggerheartApplyLevelUpRequest) (*pb.DaggerheartApplyLevelUpResponse, error) {
	return s.characterMutationHandler().ApplyLevelUp(ctx, in)
}

func (s *DaggerheartService) ApplyClassFeature(ctx context.Context, in *pb.DaggerheartApplyClassFeatureRequest) (*pb.DaggerheartApplyClassFeatureResponse, error) {
	return s.characterMutationHandler().ApplyClassFeature(ctx, in)
}

func (s *DaggerheartService) ApplySubclassFeature(ctx context.Context, in *pb.DaggerheartApplySubclassFeatureRequest) (*pb.DaggerheartApplySubclassFeatureResponse, error) {
	return s.characterMutationHandler().ApplySubclassFeature(ctx, in)
}

func (s *DaggerheartService) UpdateGold(ctx context.Context, in *pb.DaggerheartUpdateGoldRequest) (*pb.DaggerheartUpdateGoldResponse, error) {
	return s.characterMutationHandler().UpdateGold(ctx, in)
}

func (s *DaggerheartService) AcquireDomainCard(ctx context.Context, in *pb.DaggerheartAcquireDomainCardRequest) (*pb.DaggerheartAcquireDomainCardResponse, error) {
	return s.characterMutationHandler().AcquireDomainCard(ctx, in)
}

func (s *DaggerheartService) SwapEquipment(ctx context.Context, in *pb.DaggerheartSwapEquipmentRequest) (*pb.DaggerheartSwapEquipmentResponse, error) {
	return s.characterMutationHandler().SwapEquipment(ctx, in)
}

func (s *DaggerheartService) UseConsumable(ctx context.Context, in *pb.DaggerheartUseConsumableRequest) (*pb.DaggerheartUseConsumableResponse, error) {
	return s.characterMutationHandler().UseConsumable(ctx, in)
}

func (s *DaggerheartService) AcquireConsumable(ctx context.Context, in *pb.DaggerheartAcquireConsumableRequest) (*pb.DaggerheartAcquireConsumableResponse, error) {
	return s.characterMutationHandler().AcquireConsumable(ctx, in)
}
