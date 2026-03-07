package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) UpdateGold(ctx context.Context, in *pb.DaggerheartUpdateGoldRequest) (*pb.DaggerheartUpdateGoldResponse, error) {
	return newInventoryApplication(s).runUpdateGold(ctx, in)
}

func (s *DaggerheartService) AcquireDomainCard(ctx context.Context, in *pb.DaggerheartAcquireDomainCardRequest) (*pb.DaggerheartAcquireDomainCardResponse, error) {
	return newInventoryApplication(s).runAcquireDomainCard(ctx, in)
}

func (s *DaggerheartService) SwapEquipment(ctx context.Context, in *pb.DaggerheartSwapEquipmentRequest) (*pb.DaggerheartSwapEquipmentResponse, error) {
	return newInventoryApplication(s).runSwapEquipment(ctx, in)
}

func (s *DaggerheartService) UseConsumable(ctx context.Context, in *pb.DaggerheartUseConsumableRequest) (*pb.DaggerheartUseConsumableResponse, error) {
	return newInventoryApplication(s).runUseConsumable(ctx, in)
}

func (s *DaggerheartService) AcquireConsumable(ctx context.Context, in *pb.DaggerheartAcquireConsumableRequest) (*pb.DaggerheartAcquireConsumableResponse, error) {
	return newInventoryApplication(s).runAcquireConsumable(ctx, in)
}
