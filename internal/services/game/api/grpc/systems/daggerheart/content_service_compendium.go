package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartContentService) GetAdversary(ctx context.Context, in *pb.GetDaggerheartAdversaryRequest) (*pb.GetDaggerheartAdversaryResponse, error) {
	return s.handler().GetAdversary(ctx, in)
}

func (s *DaggerheartContentService) ListAdversaries(ctx context.Context, in *pb.ListDaggerheartAdversariesRequest) (*pb.ListDaggerheartAdversariesResponse, error) {
	return s.handler().ListAdversaries(ctx, in)
}

func (s *DaggerheartContentService) GetBeastform(ctx context.Context, in *pb.GetDaggerheartBeastformRequest) (*pb.GetDaggerheartBeastformResponse, error) {
	return s.handler().GetBeastform(ctx, in)
}

func (s *DaggerheartContentService) ListBeastforms(ctx context.Context, in *pb.ListDaggerheartBeastformsRequest) (*pb.ListDaggerheartBeastformsResponse, error) {
	return s.handler().ListBeastforms(ctx, in)
}

func (s *DaggerheartContentService) GetCompanionExperience(ctx context.Context, in *pb.GetDaggerheartCompanionExperienceRequest) (*pb.GetDaggerheartCompanionExperienceResponse, error) {
	return s.handler().GetCompanionExperience(ctx, in)
}

func (s *DaggerheartContentService) ListCompanionExperiences(ctx context.Context, in *pb.ListDaggerheartCompanionExperiencesRequest) (*pb.ListDaggerheartCompanionExperiencesResponse, error) {
	return s.handler().ListCompanionExperiences(ctx, in)
}

func (s *DaggerheartContentService) GetLootEntry(ctx context.Context, in *pb.GetDaggerheartLootEntryRequest) (*pb.GetDaggerheartLootEntryResponse, error) {
	return s.handler().GetLootEntry(ctx, in)
}

func (s *DaggerheartContentService) ListLootEntries(ctx context.Context, in *pb.ListDaggerheartLootEntriesRequest) (*pb.ListDaggerheartLootEntriesResponse, error) {
	return s.handler().ListLootEntries(ctx, in)
}

func (s *DaggerheartContentService) GetDamageType(ctx context.Context, in *pb.GetDaggerheartDamageTypeRequest) (*pb.GetDaggerheartDamageTypeResponse, error) {
	return s.handler().GetDamageType(ctx, in)
}

func (s *DaggerheartContentService) ListDamageTypes(ctx context.Context, in *pb.ListDaggerheartDamageTypesRequest) (*pb.ListDaggerheartDamageTypesResponse, error) {
	return s.handler().ListDamageTypes(ctx, in)
}

func (s *DaggerheartContentService) GetDomain(ctx context.Context, in *pb.GetDaggerheartDomainRequest) (*pb.GetDaggerheartDomainResponse, error) {
	return s.handler().GetDomain(ctx, in)
}

func (s *DaggerheartContentService) ListDomains(ctx context.Context, in *pb.ListDaggerheartDomainsRequest) (*pb.ListDaggerheartDomainsResponse, error) {
	return s.handler().ListDomains(ctx, in)
}

func (s *DaggerheartContentService) GetDomainCard(ctx context.Context, in *pb.GetDaggerheartDomainCardRequest) (*pb.GetDaggerheartDomainCardResponse, error) {
	return s.handler().GetDomainCard(ctx, in)
}

func (s *DaggerheartContentService) ListDomainCards(ctx context.Context, in *pb.ListDaggerheartDomainCardsRequest) (*pb.ListDaggerheartDomainCardsResponse, error) {
	return s.handler().ListDomainCards(ctx, in)
}

func (s *DaggerheartContentService) GetWeapon(ctx context.Context, in *pb.GetDaggerheartWeaponRequest) (*pb.GetDaggerheartWeaponResponse, error) {
	return s.handler().GetWeapon(ctx, in)
}

func (s *DaggerheartContentService) ListWeapons(ctx context.Context, in *pb.ListDaggerheartWeaponsRequest) (*pb.ListDaggerheartWeaponsResponse, error) {
	return s.handler().ListWeapons(ctx, in)
}

func (s *DaggerheartContentService) GetArmor(ctx context.Context, in *pb.GetDaggerheartArmorRequest) (*pb.GetDaggerheartArmorResponse, error) {
	return s.handler().GetArmor(ctx, in)
}

func (s *DaggerheartContentService) ListArmor(ctx context.Context, in *pb.ListDaggerheartArmorRequest) (*pb.ListDaggerheartArmorResponse, error) {
	return s.handler().ListArmor(ctx, in)
}

func (s *DaggerheartContentService) GetItem(ctx context.Context, in *pb.GetDaggerheartItemRequest) (*pb.GetDaggerheartItemResponse, error) {
	return s.handler().GetItem(ctx, in)
}

func (s *DaggerheartContentService) ListItems(ctx context.Context, in *pb.ListDaggerheartItemsRequest) (*pb.ListDaggerheartItemsResponse, error) {
	return s.handler().ListItems(ctx, in)
}

func (s *DaggerheartContentService) GetEnvironment(ctx context.Context, in *pb.GetDaggerheartEnvironmentRequest) (*pb.GetDaggerheartEnvironmentResponse, error) {
	return s.handler().GetEnvironment(ctx, in)
}

func (s *DaggerheartContentService) ListEnvironments(ctx context.Context, in *pb.ListDaggerheartEnvironmentsRequest) (*pb.ListDaggerheartEnvironmentsResponse, error) {
	return s.handler().ListEnvironments(ctx, in)
}
