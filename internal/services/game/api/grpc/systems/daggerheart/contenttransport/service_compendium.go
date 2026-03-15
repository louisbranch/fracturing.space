package contenttransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (h *Handler) GetAdversary(ctx context.Context, in *pb.GetDaggerheartAdversaryRequest) (*pb.GetDaggerheartAdversaryResponse, error) {
	return newContentApplication(h).runGetAdversary(ctx, in)
}

func (h *Handler) ListAdversaries(ctx context.Context, in *pb.ListDaggerheartAdversariesRequest) (*pb.ListDaggerheartAdversariesResponse, error) {
	return newContentApplication(h).runListAdversaries(ctx, in)
}

func (h *Handler) GetBeastform(ctx context.Context, in *pb.GetDaggerheartBeastformRequest) (*pb.GetDaggerheartBeastformResponse, error) {
	return newContentApplication(h).runGetBeastform(ctx, in)
}

func (h *Handler) ListBeastforms(ctx context.Context, in *pb.ListDaggerheartBeastformsRequest) (*pb.ListDaggerheartBeastformsResponse, error) {
	return newContentApplication(h).runListBeastforms(ctx, in)
}

func (h *Handler) GetCompanionExperience(ctx context.Context, in *pb.GetDaggerheartCompanionExperienceRequest) (*pb.GetDaggerheartCompanionExperienceResponse, error) {
	return newContentApplication(h).runGetCompanionExperience(ctx, in)
}

func (h *Handler) ListCompanionExperiences(ctx context.Context, in *pb.ListDaggerheartCompanionExperiencesRequest) (*pb.ListDaggerheartCompanionExperiencesResponse, error) {
	return newContentApplication(h).runListCompanionExperiences(ctx, in)
}

func (h *Handler) GetLootEntry(ctx context.Context, in *pb.GetDaggerheartLootEntryRequest) (*pb.GetDaggerheartLootEntryResponse, error) {
	return newContentApplication(h).runGetLootEntry(ctx, in)
}

func (h *Handler) ListLootEntries(ctx context.Context, in *pb.ListDaggerheartLootEntriesRequest) (*pb.ListDaggerheartLootEntriesResponse, error) {
	return newContentApplication(h).runListLootEntries(ctx, in)
}

func (h *Handler) GetDamageType(ctx context.Context, in *pb.GetDaggerheartDamageTypeRequest) (*pb.GetDaggerheartDamageTypeResponse, error) {
	return newContentApplication(h).runGetDamageType(ctx, in)
}

func (h *Handler) ListDamageTypes(ctx context.Context, in *pb.ListDaggerheartDamageTypesRequest) (*pb.ListDaggerheartDamageTypesResponse, error) {
	return newContentApplication(h).runListDamageTypes(ctx, in)
}

func (h *Handler) GetDomain(ctx context.Context, in *pb.GetDaggerheartDomainRequest) (*pb.GetDaggerheartDomainResponse, error) {
	return newContentApplication(h).runGetDomain(ctx, in)
}

func (h *Handler) ListDomains(ctx context.Context, in *pb.ListDaggerheartDomainsRequest) (*pb.ListDaggerheartDomainsResponse, error) {
	return newContentApplication(h).runListDomains(ctx, in)
}

func (h *Handler) GetDomainCard(ctx context.Context, in *pb.GetDaggerheartDomainCardRequest) (*pb.GetDaggerheartDomainCardResponse, error) {
	return newContentApplication(h).runGetDomainCard(ctx, in)
}

func (h *Handler) ListDomainCards(ctx context.Context, in *pb.ListDaggerheartDomainCardsRequest) (*pb.ListDaggerheartDomainCardsResponse, error) {
	return newContentApplication(h).runListDomainCards(ctx, in)
}

func (h *Handler) GetWeapon(ctx context.Context, in *pb.GetDaggerheartWeaponRequest) (*pb.GetDaggerheartWeaponResponse, error) {
	return newContentApplication(h).runGetWeapon(ctx, in)
}

func (h *Handler) ListWeapons(ctx context.Context, in *pb.ListDaggerheartWeaponsRequest) (*pb.ListDaggerheartWeaponsResponse, error) {
	return newContentApplication(h).runListWeapons(ctx, in)
}

func (h *Handler) GetArmor(ctx context.Context, in *pb.GetDaggerheartArmorRequest) (*pb.GetDaggerheartArmorResponse, error) {
	return newContentApplication(h).runGetArmor(ctx, in)
}

func (h *Handler) ListArmor(ctx context.Context, in *pb.ListDaggerheartArmorRequest) (*pb.ListDaggerheartArmorResponse, error) {
	return newContentApplication(h).runListArmor(ctx, in)
}

func (h *Handler) GetItem(ctx context.Context, in *pb.GetDaggerheartItemRequest) (*pb.GetDaggerheartItemResponse, error) {
	return newContentApplication(h).runGetItem(ctx, in)
}

func (h *Handler) ListItems(ctx context.Context, in *pb.ListDaggerheartItemsRequest) (*pb.ListDaggerheartItemsResponse, error) {
	return newContentApplication(h).runListItems(ctx, in)
}

func (h *Handler) GetEnvironment(ctx context.Context, in *pb.GetDaggerheartEnvironmentRequest) (*pb.GetDaggerheartEnvironmentResponse, error) {
	return newContentApplication(h).runGetEnvironment(ctx, in)
}

func (h *Handler) ListEnvironments(ctx context.Context, in *pb.ListDaggerheartEnvironmentsRequest) (*pb.ListDaggerheartEnvironmentsResponse, error) {
	return newContentApplication(h).runListEnvironments(ctx, in)
}
