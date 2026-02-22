package daggerheart

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListDaggerheartContentPageSize = 50
	maxListDaggerheartContentPageSize     = 200
)

// DaggerheartContentService implements the Daggerheart content gRPC API.
type DaggerheartContentService struct {
	pb.UnimplementedDaggerheartContentServiceServer
	stores Stores
}

// NewDaggerheartContentService creates a configured gRPC handler for content catalog APIs.
func NewDaggerheartContentService(stores Stores) (*DaggerheartContentService, error) {
	if err := stores.ValidateContent(); err != nil {
		return nil, fmt.Errorf("validate stores: %w", err)
	}
	return &DaggerheartContentService{stores: stores}, nil
}

// GetContentCatalog returns the entire Daggerheart content catalog.
func (s *DaggerheartContentService) GetContentCatalog(ctx context.Context, in *pb.GetDaggerheartContentCatalogRequest) (*pb.GetDaggerheartContentCatalogResponse, error) {
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	var (
		classes              []storage.DaggerheartClass
		subclasses           []storage.DaggerheartSubclass
		heritages            []storage.DaggerheartHeritage
		experiences          []storage.DaggerheartExperienceEntry
		adversaries          []storage.DaggerheartAdversaryEntry
		beastforms           []storage.DaggerheartBeastformEntry
		companionExperiences []storage.DaggerheartCompanionExperienceEntry
		lootEntries          []storage.DaggerheartLootEntry
		damageTypes          []storage.DaggerheartDamageTypeEntry
		domains              []storage.DaggerheartDomain
		domainCards          []storage.DaggerheartDomainCard
		weapons              []storage.DaggerheartWeapon
		armor                []storage.DaggerheartArmor
		items                []storage.DaggerheartItem
		environments         []storage.DaggerheartEnvironment
	)

	steps := []contentCatalogStep{
		{name: "list classes", run: func() error { var stepErr error; classes, stepErr = store.ListDaggerheartClasses(ctx); return stepErr }},
		{name: "list subclasses", run: func() error {
			var stepErr error
			subclasses, stepErr = store.ListDaggerheartSubclasses(ctx)
			return stepErr
		}},
		{name: "list heritages", run: func() error {
			var stepErr error
			heritages, stepErr = store.ListDaggerheartHeritages(ctx)
			return stepErr
		}},
		{name: "list experiences", run: func() error {
			var stepErr error
			experiences, stepErr = store.ListDaggerheartExperiences(ctx)
			return stepErr
		}},
		{name: "list adversaries", run: func() error {
			var stepErr error
			adversaries, stepErr = store.ListDaggerheartAdversaryEntries(ctx)
			return stepErr
		}},
		{name: "list beastforms", run: func() error {
			var stepErr error
			beastforms, stepErr = store.ListDaggerheartBeastforms(ctx)
			return stepErr
		}},
		{name: "list companion experiences", run: func() error {
			var stepErr error
			companionExperiences, stepErr = store.ListDaggerheartCompanionExperiences(ctx)
			return stepErr
		}},
		{name: "list loot entries", run: func() error {
			var stepErr error
			lootEntries, stepErr = store.ListDaggerheartLootEntries(ctx)
			return stepErr
		}},
		{name: "list damage types", run: func() error {
			var stepErr error
			damageTypes, stepErr = store.ListDaggerheartDamageTypes(ctx)
			return stepErr
		}},
		{name: "list domains", run: func() error { var stepErr error; domains, stepErr = store.ListDaggerheartDomains(ctx); return stepErr }},
		{name: "list domain cards", run: func() error {
			var stepErr error
			domainCards, stepErr = store.ListDaggerheartDomainCards(ctx)
			return stepErr
		}},
		{name: "list weapons", run: func() error { var stepErr error; weapons, stepErr = store.ListDaggerheartWeapons(ctx); return stepErr }},
		{name: "list armor", run: func() error { var stepErr error; armor, stepErr = store.ListDaggerheartArmor(ctx); return stepErr }},
		{name: "list items", run: func() error { var stepErr error; items, stepErr = store.ListDaggerheartItems(ctx); return stepErr }},
		{name: "list environments", run: func() error {
			var stepErr error
			environments, stepErr = store.ListDaggerheartEnvironments(ctx)
			return stepErr
		}},
		{name: "localize classes", run: func() error { return localizeClasses(ctx, store, in.GetLocale(), classes) }},
		{name: "localize subclasses", run: func() error { return localizeSubclasses(ctx, store, in.GetLocale(), subclasses) }},
		{name: "localize heritages", run: func() error { return localizeHeritages(ctx, store, in.GetLocale(), heritages) }},
		{name: "localize experiences", run: func() error { return localizeExperiences(ctx, store, in.GetLocale(), experiences) }},
		{name: "localize adversaries", run: func() error { return localizeAdversaries(ctx, store, in.GetLocale(), adversaries) }},
		{name: "localize beastforms", run: func() error { return localizeBeastforms(ctx, store, in.GetLocale(), beastforms) }},
		{name: "localize companion experiences", run: func() error { return localizeCompanionExperiences(ctx, store, in.GetLocale(), companionExperiences) }},
		{name: "localize loot entries", run: func() error { return localizeLootEntries(ctx, store, in.GetLocale(), lootEntries) }},
		{name: "localize damage types", run: func() error { return localizeDamageTypes(ctx, store, in.GetLocale(), damageTypes) }},
		{name: "localize domains", run: func() error { return localizeDomains(ctx, store, in.GetLocale(), domains) }},
		{name: "localize domain cards", run: func() error { return localizeDomainCards(ctx, store, in.GetLocale(), domainCards) }},
		{name: "localize weapons", run: func() error { return localizeWeapons(ctx, store, in.GetLocale(), weapons) }},
		{name: "localize armor", run: func() error { return localizeArmor(ctx, store, in.GetLocale(), armor) }},
		{name: "localize items", run: func() error { return localizeItems(ctx, store, in.GetLocale(), items) }},
		{name: "localize environments", run: func() error { return localizeEnvironments(ctx, store, in.GetLocale(), environments) }},
	}
	if err := runContentCatalogSteps(steps); err != nil {
		return nil, status.Errorf(codes.Internal, "content catalog pipeline: %v", err)
	}

	return &pb.GetDaggerheartContentCatalogResponse{
		Catalog: &pb.DaggerheartContentCatalog{
			Classes:              toProtoDaggerheartClasses(classes),
			Subclasses:           toProtoDaggerheartSubclasses(subclasses),
			Heritages:            toProtoDaggerheartHeritages(heritages),
			Experiences:          toProtoDaggerheartExperiences(experiences),
			Adversaries:          toProtoDaggerheartAdversaryEntries(adversaries),
			Beastforms:           toProtoDaggerheartBeastforms(beastforms),
			CompanionExperiences: toProtoDaggerheartCompanionExperiences(companionExperiences),
			LootEntries:          toProtoDaggerheartLootEntries(lootEntries),
			DamageTypes:          toProtoDaggerheartDamageTypes(damageTypes),
			Domains:              toProtoDaggerheartDomains(domains),
			DomainCards:          toProtoDaggerheartDomainCards(domainCards),
			Weapons:              toProtoDaggerheartWeapons(weapons),
			Armor:                toProtoDaggerheartArmorList(armor),
			Items:                toProtoDaggerheartItems(items),
			Environments:         toProtoDaggerheartEnvironments(environments),
		},
	}, nil
}

// GetClass returns a single Daggerheart class.
func (s *DaggerheartContentService) GetClass(ctx context.Context, in *pb.GetDaggerheartClassRequest) (*pb.GetDaggerheartClassResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "class request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListClasses(ctx context.Context, in *pb.ListDaggerheartClassesRequest) (*pb.ListDaggerheartClassesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list classes request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}
	classes, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), classDescriptor)
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
func (s *DaggerheartContentService) GetSubclass(ctx context.Context, in *pb.GetDaggerheartSubclassRequest) (*pb.GetDaggerheartSubclassResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "subclass request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListSubclasses(ctx context.Context, in *pb.ListDaggerheartSubclassesRequest) (*pb.ListDaggerheartSubclassesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list subclasses request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}
	subclasses, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), subclassDescriptor)
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
func (s *DaggerheartContentService) GetHeritage(ctx context.Context, in *pb.GetDaggerheartHeritageRequest) (*pb.GetDaggerheartHeritageResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "heritage request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListHeritages(ctx context.Context, in *pb.ListDaggerheartHeritagesRequest) (*pb.ListDaggerheartHeritagesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list heritages request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}
	heritages, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), heritageDescriptor)
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
func (s *DaggerheartContentService) GetExperience(ctx context.Context, in *pb.GetDaggerheartExperienceRequest) (*pb.GetDaggerheartExperienceResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "experience request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListExperiences(ctx context.Context, in *pb.ListDaggerheartExperiencesRequest) (*pb.ListDaggerheartExperiencesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list experiences request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}
	experiences, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), experienceDescriptor)
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
func (s *DaggerheartContentService) GetAdversary(ctx context.Context, in *pb.GetDaggerheartAdversaryRequest) (*pb.GetDaggerheartAdversaryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "adversary request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListAdversaries(ctx context.Context, in *pb.ListDaggerheartAdversariesRequest) (*pb.ListDaggerheartAdversariesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list adversaries request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	adversaries, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), adversaryDescriptor)
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
func (s *DaggerheartContentService) GetBeastform(ctx context.Context, in *pb.GetDaggerheartBeastformRequest) (*pb.GetDaggerheartBeastformResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "beastform request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListBeastforms(ctx context.Context, in *pb.ListDaggerheartBeastformsRequest) (*pb.ListDaggerheartBeastformsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list beastforms request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	beastforms, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), beastformDescriptor)
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
func (s *DaggerheartContentService) GetCompanionExperience(ctx context.Context, in *pb.GetDaggerheartCompanionExperienceRequest) (*pb.GetDaggerheartCompanionExperienceResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "companion experience request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListCompanionExperiences(ctx context.Context, in *pb.ListDaggerheartCompanionExperiencesRequest) (*pb.ListDaggerheartCompanionExperiencesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list companion experiences request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	experiences, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), companionExperienceDescriptor)
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
func (s *DaggerheartContentService) GetLootEntry(ctx context.Context, in *pb.GetDaggerheartLootEntryRequest) (*pb.GetDaggerheartLootEntryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "loot entry request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListLootEntries(ctx context.Context, in *pb.ListDaggerheartLootEntriesRequest) (*pb.ListDaggerheartLootEntriesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list loot entries request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	entries, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), lootEntryDescriptor)
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
func (s *DaggerheartContentService) GetDamageType(ctx context.Context, in *pb.GetDaggerheartDamageTypeRequest) (*pb.GetDaggerheartDamageTypeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "damage type request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListDamageTypes(ctx context.Context, in *pb.ListDaggerheartDamageTypesRequest) (*pb.ListDaggerheartDamageTypesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list damage types request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	entries, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), damageTypeDescriptor)
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
func (s *DaggerheartContentService) GetDomain(ctx context.Context, in *pb.GetDaggerheartDomainRequest) (*pb.GetDaggerheartDomainResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "domain request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListDomains(ctx context.Context, in *pb.ListDaggerheartDomainsRequest) (*pb.ListDaggerheartDomainsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list domains request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	domains, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), domainDescriptor)
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
func (s *DaggerheartContentService) GetDomainCard(ctx context.Context, in *pb.GetDaggerheartDomainCardRequest) (*pb.GetDaggerheartDomainCardResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "domain card request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListDomainCards(ctx context.Context, in *pb.ListDaggerheartDomainCardsRequest) (*pb.ListDaggerheartDomainCardsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list domain cards request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	domainID := strings.TrimSpace(in.GetDomainId())
	cards, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
		DomainID:  domainID,
	}, in.GetLocale(), domainCardDescriptor)
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
func (s *DaggerheartContentService) GetWeapon(ctx context.Context, in *pb.GetDaggerheartWeaponRequest) (*pb.GetDaggerheartWeaponResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "weapon request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListWeapons(ctx context.Context, in *pb.ListDaggerheartWeaponsRequest) (*pb.ListDaggerheartWeaponsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list weapons request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	weapons, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), weaponDescriptor)
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
func (s *DaggerheartContentService) GetArmor(ctx context.Context, in *pb.GetDaggerheartArmorRequest) (*pb.GetDaggerheartArmorResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "armor request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListArmor(ctx context.Context, in *pb.ListDaggerheartArmorRequest) (*pb.ListDaggerheartArmorResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list armor request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	armor, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), armorDescriptor)
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
func (s *DaggerheartContentService) GetItem(ctx context.Context, in *pb.GetDaggerheartItemRequest) (*pb.GetDaggerheartItemResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "item request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListItems(ctx context.Context, in *pb.ListDaggerheartItemsRequest) (*pb.ListDaggerheartItemsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list items request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	items, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), itemDescriptor)
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
func (s *DaggerheartContentService) GetEnvironment(ctx context.Context, in *pb.GetDaggerheartEnvironmentRequest) (*pb.GetDaggerheartEnvironmentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "environment request is required")
	}
	store, err := s.contentStore()
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
func (s *DaggerheartContentService) ListEnvironments(ctx context.Context, in *pb.ListDaggerheartEnvironmentsRequest) (*pb.ListDaggerheartEnvironmentsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list environments request is required")
	}
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	items, page, err := listContentEntries(ctx, store, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, in.GetLocale(), environmentDescriptor)
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

func (s *DaggerheartContentService) contentStore() (storage.DaggerheartContentStore, error) {
	if s == nil || s.stores.DaggerheartContent == nil {
		return nil, status.Error(codes.Internal, "content store is not configured")
	}
	return s.stores.DaggerheartContent, nil
}

func mapContentErr(action string, err error) error {
	if errors.Is(err, storage.ErrNotFound) {
		return status.Error(codes.NotFound, "content not found")
	}
	return status.Errorf(codes.Internal, "%s: %v", action, err)
}

func toProtoDaggerheartClass(class storage.DaggerheartClass) *pb.DaggerheartClass {
	return &pb.DaggerheartClass{
		Id:              class.ID,
		Name:            class.Name,
		StartingEvasion: int32(class.StartingEvasion),
		StartingHp:      int32(class.StartingHP),
		StartingItems:   append([]string{}, class.StartingItems...),
		Features:        toProtoDaggerheartFeatures(class.Features),
		HopeFeature:     toProtoDaggerheartHopeFeature(class.HopeFeature),
		DomainIds:       append([]string{}, class.DomainIDs...),
	}
}

func toProtoDaggerheartClasses(classes []storage.DaggerheartClass) []*pb.DaggerheartClass {
	items := make([]*pb.DaggerheartClass, 0, len(classes))
	for _, class := range classes {
		items = append(items, toProtoDaggerheartClass(class))
	}
	return items
}

func toProtoDaggerheartSubclass(subclass storage.DaggerheartSubclass) *pb.DaggerheartSubclass {
	return &pb.DaggerheartSubclass{
		Id:                     subclass.ID,
		Name:                   subclass.Name,
		SpellcastTrait:         subclass.SpellcastTrait,
		FoundationFeatures:     toProtoDaggerheartFeatures(subclass.FoundationFeatures),
		SpecializationFeatures: toProtoDaggerheartFeatures(subclass.SpecializationFeatures),
		MasteryFeatures:        toProtoDaggerheartFeatures(subclass.MasteryFeatures),
	}
}

func toProtoDaggerheartSubclasses(subclasses []storage.DaggerheartSubclass) []*pb.DaggerheartSubclass {
	items := make([]*pb.DaggerheartSubclass, 0, len(subclasses))
	for _, subclass := range subclasses {
		items = append(items, toProtoDaggerheartSubclass(subclass))
	}
	return items
}

func toProtoDaggerheartHeritage(heritage storage.DaggerheartHeritage) *pb.DaggerheartHeritage {
	return &pb.DaggerheartHeritage{
		Id:       heritage.ID,
		Name:     heritage.Name,
		Kind:     heritageKindToProto(heritage.Kind),
		Features: toProtoDaggerheartFeatures(heritage.Features),
	}
}

func toProtoDaggerheartExperience(experience storage.DaggerheartExperienceEntry) *pb.DaggerheartExperienceEntry {
	return &pb.DaggerheartExperienceEntry{
		Id:          experience.ID,
		Name:        experience.Name,
		Description: experience.Description,
	}
}

func toProtoDaggerheartExperiences(experiences []storage.DaggerheartExperienceEntry) []*pb.DaggerheartExperienceEntry {
	items := make([]*pb.DaggerheartExperienceEntry, 0, len(experiences))
	for _, experience := range experiences {
		items = append(items, toProtoDaggerheartExperience(experience))
	}
	return items
}

func toProtoDaggerheartAdversaryAttack(attack storage.DaggerheartAdversaryAttack) *pb.DaggerheartAdversaryAttack {
	return &pb.DaggerheartAdversaryAttack{
		Name:        attack.Name,
		Range:       attack.Range,
		DamageDice:  toProtoDaggerheartDamageDice(attack.DamageDice),
		DamageBonus: int32(attack.DamageBonus),
		DamageType:  damageTypeToProto(attack.DamageType),
	}
}

func toProtoDaggerheartAdversaryExperiences(experiences []storage.DaggerheartAdversaryExperience) []*pb.DaggerheartAdversaryExperience {
	items := make([]*pb.DaggerheartAdversaryExperience, 0, len(experiences))
	for _, experience := range experiences {
		items = append(items, &pb.DaggerheartAdversaryExperience{
			Name:     experience.Name,
			Modifier: int32(experience.Modifier),
		})
	}
	return items
}

func toProtoDaggerheartAdversaryFeatures(features []storage.DaggerheartAdversaryFeature) []*pb.DaggerheartAdversaryFeature {
	items := make([]*pb.DaggerheartAdversaryFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, &pb.DaggerheartAdversaryFeature{
			Id:          feature.ID,
			Name:        feature.Name,
			Kind:        feature.Kind,
			Description: feature.Description,
			CostType:    feature.CostType,
			Cost:        int32(feature.Cost),
		})
	}
	return items
}

func toProtoDaggerheartAdversaryEntry(entry storage.DaggerheartAdversaryEntry) *pb.DaggerheartAdversaryEntry {
	return &pb.DaggerheartAdversaryEntry{
		Id:              entry.ID,
		Name:            entry.Name,
		Tier:            int32(entry.Tier),
		Role:            entry.Role,
		Description:     entry.Description,
		Motives:         entry.Motives,
		Difficulty:      int32(entry.Difficulty),
		MajorThreshold:  int32(entry.MajorThreshold),
		SevereThreshold: int32(entry.SevereThreshold),
		Hp:              int32(entry.HP),
		Stress:          int32(entry.Stress),
		Armor:           int32(entry.Armor),
		AttackModifier:  int32(entry.AttackModifier),
		StandardAttack:  toProtoDaggerheartAdversaryAttack(entry.StandardAttack),
		Experiences:     toProtoDaggerheartAdversaryExperiences(entry.Experiences),
		Features:        toProtoDaggerheartAdversaryFeatures(entry.Features),
	}
}

func toProtoDaggerheartAdversaryEntries(entries []storage.DaggerheartAdversaryEntry) []*pb.DaggerheartAdversaryEntry {
	items := make([]*pb.DaggerheartAdversaryEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toProtoDaggerheartAdversaryEntry(entry))
	}
	return items
}

func toProtoDaggerheartBeastformAttack(attack storage.DaggerheartBeastformAttack) *pb.DaggerheartBeastformAttack {
	return &pb.DaggerheartBeastformAttack{
		Range:       attack.Range,
		Trait:       attack.Trait,
		DamageDice:  toProtoDaggerheartDamageDice(attack.DamageDice),
		DamageBonus: int32(attack.DamageBonus),
		DamageType:  damageTypeToProto(attack.DamageType),
	}
}

func toProtoDaggerheartBeastformFeatures(features []storage.DaggerheartBeastformFeature) []*pb.DaggerheartBeastformFeature {
	items := make([]*pb.DaggerheartBeastformFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, &pb.DaggerheartBeastformFeature{
			Id:          feature.ID,
			Name:        feature.Name,
			Description: feature.Description,
		})
	}
	return items
}

func toProtoDaggerheartBeastform(beastform storage.DaggerheartBeastformEntry) *pb.DaggerheartBeastformEntry {
	return &pb.DaggerheartBeastformEntry{
		Id:           beastform.ID,
		Name:         beastform.Name,
		Tier:         int32(beastform.Tier),
		Examples:     beastform.Examples,
		Trait:        beastform.Trait,
		TraitBonus:   int32(beastform.TraitBonus),
		EvasionBonus: int32(beastform.EvasionBonus),
		Attack:       toProtoDaggerheartBeastformAttack(beastform.Attack),
		Advantages:   append([]string{}, beastform.Advantages...),
		Features:     toProtoDaggerheartBeastformFeatures(beastform.Features),
	}
}

func toProtoDaggerheartBeastforms(beastforms []storage.DaggerheartBeastformEntry) []*pb.DaggerheartBeastformEntry {
	items := make([]*pb.DaggerheartBeastformEntry, 0, len(beastforms))
	for _, beastform := range beastforms {
		items = append(items, toProtoDaggerheartBeastform(beastform))
	}
	return items
}

func toProtoDaggerheartCompanionExperience(experience storage.DaggerheartCompanionExperienceEntry) *pb.DaggerheartCompanionExperienceEntry {
	return &pb.DaggerheartCompanionExperienceEntry{
		Id:          experience.ID,
		Name:        experience.Name,
		Description: experience.Description,
	}
}

func toProtoDaggerheartCompanionExperiences(experiences []storage.DaggerheartCompanionExperienceEntry) []*pb.DaggerheartCompanionExperienceEntry {
	items := make([]*pb.DaggerheartCompanionExperienceEntry, 0, len(experiences))
	for _, experience := range experiences {
		items = append(items, toProtoDaggerheartCompanionExperience(experience))
	}
	return items
}

func toProtoDaggerheartLootEntry(entry storage.DaggerheartLootEntry) *pb.DaggerheartLootEntry {
	return &pb.DaggerheartLootEntry{
		Id:          entry.ID,
		Name:        entry.Name,
		Roll:        int32(entry.Roll),
		Description: entry.Description,
	}
}

func toProtoDaggerheartLootEntries(entries []storage.DaggerheartLootEntry) []*pb.DaggerheartLootEntry {
	items := make([]*pb.DaggerheartLootEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toProtoDaggerheartLootEntry(entry))
	}
	return items
}

func toProtoDaggerheartDamageType(entry storage.DaggerheartDamageTypeEntry) *pb.DaggerheartDamageTypeEntry {
	return &pb.DaggerheartDamageTypeEntry{
		Id:          entry.ID,
		Name:        entry.Name,
		Description: entry.Description,
	}
}

func toProtoDaggerheartDamageTypes(entries []storage.DaggerheartDamageTypeEntry) []*pb.DaggerheartDamageTypeEntry {
	items := make([]*pb.DaggerheartDamageTypeEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toProtoDaggerheartDamageType(entry))
	}
	return items
}

func toProtoDaggerheartHeritages(heritages []storage.DaggerheartHeritage) []*pb.DaggerheartHeritage {
	items := make([]*pb.DaggerheartHeritage, 0, len(heritages))
	for _, heritage := range heritages {
		items = append(items, toProtoDaggerheartHeritage(heritage))
	}
	return items
}

func toProtoDaggerheartDomain(domain storage.DaggerheartDomain) *pb.DaggerheartDomain {
	return &pb.DaggerheartDomain{
		Id:          domain.ID,
		Name:        domain.Name,
		Description: domain.Description,
	}
}

func toProtoDaggerheartDomains(domains []storage.DaggerheartDomain) []*pb.DaggerheartDomain {
	items := make([]*pb.DaggerheartDomain, 0, len(domains))
	for _, domain := range domains {
		items = append(items, toProtoDaggerheartDomain(domain))
	}
	return items
}

func toProtoDaggerheartDomainCard(card storage.DaggerheartDomainCard) *pb.DaggerheartDomainCard {
	return &pb.DaggerheartDomainCard{
		Id:          card.ID,
		Name:        card.Name,
		DomainId:    card.DomainID,
		Level:       int32(card.Level),
		Type:        domainCardTypeToProto(card.Type),
		RecallCost:  int32(card.RecallCost),
		UsageLimit:  card.UsageLimit,
		FeatureText: card.FeatureText,
	}
}

func toProtoDaggerheartDomainCards(cards []storage.DaggerheartDomainCard) []*pb.DaggerheartDomainCard {
	items := make([]*pb.DaggerheartDomainCard, 0, len(cards))
	for _, card := range cards {
		items = append(items, toProtoDaggerheartDomainCard(card))
	}
	return items
}

func toProtoDaggerheartWeapon(weapon storage.DaggerheartWeapon) *pb.DaggerheartWeapon {
	return &pb.DaggerheartWeapon{
		Id:         weapon.ID,
		Name:       weapon.Name,
		Category:   weaponCategoryToProto(weapon.Category),
		Tier:       int32(weapon.Tier),
		Trait:      weapon.Trait,
		Range:      weapon.Range,
		DamageDice: toProtoDaggerheartDamageDice(weapon.DamageDice),
		DamageType: damageTypeToProto(weapon.DamageType),
		Burden:     int32(weapon.Burden),
		Feature:    weapon.Feature,
	}
}

func toProtoDaggerheartWeapons(weapons []storage.DaggerheartWeapon) []*pb.DaggerheartWeapon {
	items := make([]*pb.DaggerheartWeapon, 0, len(weapons))
	for _, weapon := range weapons {
		items = append(items, toProtoDaggerheartWeapon(weapon))
	}
	return items
}

func toProtoDaggerheartArmor(armor storage.DaggerheartArmor) *pb.DaggerheartArmor {
	return &pb.DaggerheartArmor{
		Id:                  armor.ID,
		Name:                armor.Name,
		Tier:                int32(armor.Tier),
		BaseMajorThreshold:  int32(armor.BaseMajorThreshold),
		BaseSevereThreshold: int32(armor.BaseSevereThreshold),
		ArmorScore:          int32(armor.ArmorScore),
		Feature:             armor.Feature,
	}
}

func toProtoDaggerheartArmorList(items []storage.DaggerheartArmor) []*pb.DaggerheartArmor {
	armor := make([]*pb.DaggerheartArmor, 0, len(items))
	for _, item := range items {
		armor = append(armor, toProtoDaggerheartArmor(item))
	}
	return armor
}

func toProtoDaggerheartItem(item storage.DaggerheartItem) *pb.DaggerheartItem {
	return &pb.DaggerheartItem{
		Id:          item.ID,
		Name:        item.Name,
		Rarity:      itemRarityToProto(item.Rarity),
		Kind:        itemKindToProto(item.Kind),
		StackMax:    int32(item.StackMax),
		Description: item.Description,
		EffectText:  item.EffectText,
	}
}

func toProtoDaggerheartItems(items []storage.DaggerheartItem) []*pb.DaggerheartItem {
	results := make([]*pb.DaggerheartItem, 0, len(items))
	for _, item := range items {
		results = append(results, toProtoDaggerheartItem(item))
	}
	return results
}

func toProtoDaggerheartEnvironment(env storage.DaggerheartEnvironment) *pb.DaggerheartEnvironment {
	return &pb.DaggerheartEnvironment{
		Id:                    env.ID,
		Name:                  env.Name,
		Tier:                  int32(env.Tier),
		Type:                  environmentTypeToProto(env.Type),
		Difficulty:            int32(env.Difficulty),
		Impulses:              append([]string{}, env.Impulses...),
		PotentialAdversaryIds: append([]string{}, env.PotentialAdversaryIDs...),
		Features:              toProtoDaggerheartFeatures(env.Features),
		Prompts:               append([]string{}, env.Prompts...),
	}
}

func toProtoDaggerheartEnvironments(envs []storage.DaggerheartEnvironment) []*pb.DaggerheartEnvironment {
	results := make([]*pb.DaggerheartEnvironment, 0, len(envs))
	for _, env := range envs {
		results = append(results, toProtoDaggerheartEnvironment(env))
	}
	return results
}

func toProtoDaggerheartFeatures(features []storage.DaggerheartFeature) []*pb.DaggerheartFeature {
	items := make([]*pb.DaggerheartFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, &pb.DaggerheartFeature{
			Id:          feature.ID,
			Name:        feature.Name,
			Description: feature.Description,
			Level:       int32(feature.Level),
		})
	}
	return items
}

func toProtoDaggerheartHopeFeature(feature storage.DaggerheartHopeFeature) *pb.DaggerheartHopeFeature {
	return &pb.DaggerheartHopeFeature{
		Name:        feature.Name,
		Description: feature.Description,
		HopeCost:    int32(feature.HopeCost),
	}
}

func toProtoDaggerheartDamageDice(dice []storage.DaggerheartDamageDie) []*pb.DiceSpec {
	results := make([]*pb.DiceSpec, 0, len(dice))
	for _, die := range dice {
		results = append(results, &pb.DiceSpec{
			Sides: int32(die.Sides),
			Count: int32(die.Count),
		})
	}
	return results
}

func heritageKindToProto(kind string) pb.DaggerheartHeritageKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "ancestry":
		return pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY
	case "community":
		return pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_COMMUNITY
	default:
		return pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_UNSPECIFIED
	}
}

func domainCardTypeToProto(kind string) pb.DaggerheartDomainCardType {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "ability":
		return pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_ABILITY
	case "spell":
		return pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_SPELL
	case "grimoire":
		return pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_GRIMOIRE
	default:
		return pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_UNSPECIFIED
	}
}

func weaponCategoryToProto(kind string) pb.DaggerheartWeaponCategory {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "primary":
		return pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_PRIMARY
	case "secondary":
		return pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_SECONDARY
	default:
		return pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_UNSPECIFIED
	}
}

func itemRarityToProto(kind string) pb.DaggerheartItemRarity {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "common":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_COMMON
	case "uncommon":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNCOMMON
	case "rare":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_RARE
	case "unique":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNIQUE
	case "legendary":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_LEGENDARY
	default:
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNSPECIFIED
	}
}

func itemKindToProto(kind string) pb.DaggerheartItemKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "consumable":
		return pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_CONSUMABLE
	case "equipment":
		return pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_EQUIPMENT
	case "treasure":
		return pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_TREASURE
	default:
		return pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_UNSPECIFIED
	}
}

func environmentTypeToProto(kind string) pb.DaggerheartEnvironmentType {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "exploration":
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_EXPLORATION
	case "social":
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_SOCIAL
	case "traversal":
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_TRAVERSAL
	case "event":
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_EVENT
	default:
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_UNSPECIFIED
	}
}

func damageTypeToProto(kind string) pb.DaggerheartDamageType {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "physical":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL
	case "magic":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC
	case "mixed":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED
	default:
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED
	}
}
