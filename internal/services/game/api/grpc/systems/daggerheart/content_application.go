package daggerheart

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contentApplication struct {
	service *DaggerheartContentService
}

func newContentApplication(service *DaggerheartContentService) contentApplication {
	return contentApplication{service: service}
}

// GetContentCatalog returns the entire Daggerheart content catalog.
func (a contentApplication) runGetContentCatalog(ctx context.Context, in *pb.GetDaggerheartContentCatalogRequest) (*pb.GetDaggerheartContentCatalogResponse, error) {
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	catalog := newContentCatalog(store, in.GetLocale())
	if err := catalog.run(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "content catalog pipeline: %v", err)
	}
	return &pb.GetDaggerheartContentCatalogResponse{Catalog: catalog.proto()}, nil
}

// GetClass returns a single Daggerheart class.
func (a contentApplication) runGetClass(ctx context.Context, in *pb.GetDaggerheartClassRequest) (*pb.GetDaggerheartClassResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "class request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "class id is required")
	}

	class, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), classDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartClassResponse{Class: class}, nil
}

// ListClasses returns Daggerheart classes.
func (a contentApplication) runListClasses(ctx context.Context, in *pb.ListDaggerheartClassesRequest) (*pb.ListDaggerheartClassesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list classes request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	classes, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), classDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartClassesResponse{
		Classes:           classes,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetSubclass returns a single Daggerheart subclass.
func (a contentApplication) runGetSubclass(ctx context.Context, in *pb.GetDaggerheartSubclassRequest) (*pb.GetDaggerheartSubclassResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "subclass request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "subclass id is required")
	}

	subclass, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), subclassDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartSubclassResponse{Subclass: subclass}, nil
}

// ListSubclasses returns Daggerheart subclasses.
func (a contentApplication) runListSubclasses(ctx context.Context, in *pb.ListDaggerheartSubclassesRequest) (*pb.ListDaggerheartSubclassesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list subclasses request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	subclasses, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), subclassDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartSubclassesResponse{
		Subclasses:        subclasses,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetHeritage returns a single Daggerheart heritage.
func (a contentApplication) runGetHeritage(ctx context.Context, in *pb.GetDaggerheartHeritageRequest) (*pb.GetDaggerheartHeritageResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "heritage request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "heritage id is required")
	}

	heritage, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), heritageDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartHeritageResponse{Heritage: heritage}, nil
}

// ListHeritages returns Daggerheart heritages.
func (a contentApplication) runListHeritages(ctx context.Context, in *pb.ListDaggerheartHeritagesRequest) (*pb.ListDaggerheartHeritagesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list heritages request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	heritages, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), heritageDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartHeritagesResponse{
		Heritages:         heritages,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetExperience returns a single Daggerheart experience.
func (a contentApplication) runGetExperience(ctx context.Context, in *pb.GetDaggerheartExperienceRequest) (*pb.GetDaggerheartExperienceResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "experience request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "experience id is required")
	}

	experience, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), experienceDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartExperienceResponse{Experience: experience}, nil
}

// ListExperiences returns Daggerheart experiences.
func (a contentApplication) runListExperiences(ctx context.Context, in *pb.ListDaggerheartExperiencesRequest) (*pb.ListDaggerheartExperiencesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list experiences request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	experiences, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), experienceDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartExperiencesResponse{
		Experiences:       experiences,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetAdversary returns a single Daggerheart adversary catalog entry.
func (a contentApplication) runGetAdversary(ctx context.Context, in *pb.GetDaggerheartAdversaryRequest) (*pb.GetDaggerheartAdversaryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "adversary request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}

	adversary, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), adversaryDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartAdversaryResponse{Adversary: adversary}, nil
}

// ListAdversaries returns Daggerheart adversary catalog entries.
func (a contentApplication) runListAdversaries(ctx context.Context, in *pb.ListDaggerheartAdversariesRequest) (*pb.ListDaggerheartAdversariesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list adversaries request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}

	adversaries, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), adversaryDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartAdversariesResponse{
		Adversaries:       adversaries,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetBeastform returns a single Daggerheart beastform catalog entry.
func (a contentApplication) runGetBeastform(ctx context.Context, in *pb.GetDaggerheartBeastformRequest) (*pb.GetDaggerheartBeastformResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "beastform request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "beastform id is required")
	}

	beastform, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), beastformDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartBeastformResponse{Beastform: beastform}, nil
}

// ListBeastforms returns Daggerheart beastform catalog entries.
func (a contentApplication) runListBeastforms(ctx context.Context, in *pb.ListDaggerheartBeastformsRequest) (*pb.ListDaggerheartBeastformsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list beastforms request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}

	beastforms, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), beastformDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartBeastformsResponse{
		Beastforms:        beastforms,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetCompanionExperience returns a single Daggerheart companion experience catalog entry.
func (a contentApplication) runGetCompanionExperience(ctx context.Context, in *pb.GetDaggerheartCompanionExperienceRequest) (*pb.GetDaggerheartCompanionExperienceResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "companion experience request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "companion experience id is required")
	}

	experience, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), companionExperienceDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartCompanionExperienceResponse{Experience: experience}, nil
}

// ListCompanionExperiences returns Daggerheart companion experience catalog entries.
func (a contentApplication) runListCompanionExperiences(ctx context.Context, in *pb.ListDaggerheartCompanionExperiencesRequest) (*pb.ListDaggerheartCompanionExperiencesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list companion experiences request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}

	experiences, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), companionExperienceDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartCompanionExperiencesResponse{
		Experiences:       experiences,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetLootEntry returns a single Daggerheart loot catalog entry.
func (a contentApplication) runGetLootEntry(ctx context.Context, in *pb.GetDaggerheartLootEntryRequest) (*pb.GetDaggerheartLootEntryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "loot entry request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "loot entry id is required")
	}

	entry, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), lootEntryDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartLootEntryResponse{Entry: entry}, nil
}

// ListLootEntries returns Daggerheart loot catalog entries.
func (a contentApplication) runListLootEntries(ctx context.Context, in *pb.ListDaggerheartLootEntriesRequest) (*pb.ListDaggerheartLootEntriesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list loot entries request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}

	entries, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), lootEntryDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartLootEntriesResponse{
		Entries:           entries,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetDamageType returns a single Daggerheart damage type catalog entry.
func (a contentApplication) runGetDamageType(ctx context.Context, in *pb.GetDaggerheartDamageTypeRequest) (*pb.GetDaggerheartDamageTypeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "damage type request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "damage type id is required")
	}

	entry, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), damageTypeDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartDamageTypeResponse{DamageType: entry}, nil
}

// ListDamageTypes returns Daggerheart damage type catalog entries.
func (a contentApplication) runListDamageTypes(ctx context.Context, in *pb.ListDaggerheartDamageTypesRequest) (*pb.ListDaggerheartDamageTypesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list damage types request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}

	entries, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), damageTypeDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartDamageTypesResponse{
		DamageTypes:       entries,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetDomain returns a single Daggerheart domain.
func (a contentApplication) runGetDomain(ctx context.Context, in *pb.GetDaggerheartDomainRequest) (*pb.GetDaggerheartDomainResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "domain request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "domain id is required")
	}

	domain, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), domainDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartDomainResponse{Domain: domain}, nil
}

// ListDomains returns Daggerheart domains.
func (a contentApplication) runListDomains(ctx context.Context, in *pb.ListDaggerheartDomainsRequest) (*pb.ListDaggerheartDomainsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list domains request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}

	domains, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), domainDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartDomainsResponse{
		Domains:           domains,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetDomainCard returns a single Daggerheart domain card.
func (a contentApplication) runGetDomainCard(ctx context.Context, in *pb.GetDaggerheartDomainCardRequest) (*pb.GetDaggerheartDomainCardResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "domain card request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "domain card id is required")
	}

	card, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), domainCardDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartDomainCardResponse{DomainCard: card}, nil
}

// ListDomainCards returns Daggerheart domain cards, optionally filtered by domain.
func (a contentApplication) runListDomainCards(ctx context.Context, in *pb.ListDaggerheartDomainCardsRequest) (*pb.ListDaggerheartDomainCardsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list domain cards request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}

	domainID := strings.TrimSpace(in.GetDomainId())
	req := newContentListRequest(in)
	req.DomainID = domainID
	cards, page, err := listContentEntries(ctx, store, req, in.GetLocale(), domainCardDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartDomainCardsResponse{
		DomainCards:       cards,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetWeapon returns a single Daggerheart weapon.
func (a contentApplication) runGetWeapon(ctx context.Context, in *pb.GetDaggerheartWeaponRequest) (*pb.GetDaggerheartWeaponResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "weapon request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "weapon id is required")
	}

	weapon, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), weaponDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartWeaponResponse{Weapon: weapon}, nil
}

// ListWeapons returns Daggerheart weapons.
func (a contentApplication) runListWeapons(ctx context.Context, in *pb.ListDaggerheartWeaponsRequest) (*pb.ListDaggerheartWeaponsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list weapons request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}

	weapons, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), weaponDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartWeaponsResponse{
		Weapons:           weapons,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetArmor returns a single Daggerheart armor entry.
func (a contentApplication) runGetArmor(ctx context.Context, in *pb.GetDaggerheartArmorRequest) (*pb.GetDaggerheartArmorResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "armor request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "armor id is required")
	}

	armor, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), armorDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartArmorResponse{Armor: armor}, nil
}

// ListArmor returns Daggerheart armor entries.
func (a contentApplication) runListArmor(ctx context.Context, in *pb.ListDaggerheartArmorRequest) (*pb.ListDaggerheartArmorResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list armor request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}

	armor, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), armorDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartArmorResponse{
		Armor:             armor,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetItem returns a single Daggerheart item.
func (a contentApplication) runGetItem(ctx context.Context, in *pb.GetDaggerheartItemRequest) (*pb.GetDaggerheartItemResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "item request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "item id is required")
	}

	item, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), itemDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartItemResponse{Item: item}, nil
}

// ListItems returns Daggerheart items.
func (a contentApplication) runListItems(ctx context.Context, in *pb.ListDaggerheartItemsRequest) (*pb.ListDaggerheartItemsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list items request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}

	items, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), itemDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartItemsResponse{
		Items:             items,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetEnvironment returns a single Daggerheart environment.
func (a contentApplication) runGetEnvironment(ctx context.Context, in *pb.GetDaggerheartEnvironmentRequest) (*pb.GetDaggerheartEnvironmentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "environment request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "environment id is required")
	}

	env, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), environmentDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartEnvironmentResponse{Environment: env}, nil
}

// ListEnvironments returns Daggerheart environments.
func (a contentApplication) runListEnvironments(ctx context.Context, in *pb.ListDaggerheartEnvironmentsRequest) (*pb.ListDaggerheartEnvironmentsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list environments request is required")
	}
	store, err := a.service.contentStore()
	if err != nil {
		return nil, err
	}

	items, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), environmentDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartEnvironmentsResponse{
		Environments:      items,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}
