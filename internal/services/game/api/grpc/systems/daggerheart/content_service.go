package daggerheart

import (
	"context"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/content/filter"
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
func NewDaggerheartContentService(stores Stores) *DaggerheartContentService {
	return &DaggerheartContentService{stores: stores}
}

// GetContentCatalog returns the entire Daggerheart content catalog.
func (s *DaggerheartContentService) GetContentCatalog(ctx context.Context, in *pb.GetDaggerheartContentCatalogRequest) (*pb.GetDaggerheartContentCatalogResponse, error) {
	store, err := s.contentStore()
	if err != nil {
		return nil, err
	}

	classes, err := store.ListDaggerheartClasses(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list classes: %v", err)
	}
	subclasses, err := store.ListDaggerheartSubclasses(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list subclasses: %v", err)
	}
	heritages, err := store.ListDaggerheartHeritages(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list heritages: %v", err)
	}
	experiences, err := store.ListDaggerheartExperiences(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list experiences: %v", err)
	}
	adversaries, err := store.ListDaggerheartAdversaryEntries(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list adversaries: %v", err)
	}
	beastforms, err := store.ListDaggerheartBeastforms(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list beastforms: %v", err)
	}
	companionExperiences, err := store.ListDaggerheartCompanionExperiences(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list companion experiences: %v", err)
	}
	lootEntries, err := store.ListDaggerheartLootEntries(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list loot entries: %v", err)
	}
	damageTypes, err := store.ListDaggerheartDamageTypes(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list damage types: %v", err)
	}
	domains, err := store.ListDaggerheartDomains(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list domains: %v", err)
	}
	domainCards, err := store.ListDaggerheartDomainCards(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list domain cards: %v", err)
	}
	weapons, err := store.ListDaggerheartWeapons(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list weapons: %v", err)
	}
	armor, err := store.ListDaggerheartArmor(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list armor: %v", err)
	}
	items, err := store.ListDaggerheartItems(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list items: %v", err)
	}
	environments, err := store.ListDaggerheartEnvironments(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list environments: %v", err)
	}

	if err := localizeClasses(ctx, store, in.GetLocale(), classes); err != nil {
		return nil, status.Errorf(codes.Internal, "localize classes: %v", err)
	}
	if err := localizeSubclasses(ctx, store, in.GetLocale(), subclasses); err != nil {
		return nil, status.Errorf(codes.Internal, "localize subclasses: %v", err)
	}
	if err := localizeHeritages(ctx, store, in.GetLocale(), heritages); err != nil {
		return nil, status.Errorf(codes.Internal, "localize heritages: %v", err)
	}
	if err := localizeExperiences(ctx, store, in.GetLocale(), experiences); err != nil {
		return nil, status.Errorf(codes.Internal, "localize experiences: %v", err)
	}
	if err := localizeAdversaries(ctx, store, in.GetLocale(), adversaries); err != nil {
		return nil, status.Errorf(codes.Internal, "localize adversaries: %v", err)
	}
	if err := localizeBeastforms(ctx, store, in.GetLocale(), beastforms); err != nil {
		return nil, status.Errorf(codes.Internal, "localize beastforms: %v", err)
	}
	if err := localizeCompanionExperiences(ctx, store, in.GetLocale(), companionExperiences); err != nil {
		return nil, status.Errorf(codes.Internal, "localize companion experiences: %v", err)
	}
	if err := localizeLootEntries(ctx, store, in.GetLocale(), lootEntries); err != nil {
		return nil, status.Errorf(codes.Internal, "localize loot entries: %v", err)
	}
	if err := localizeDamageTypes(ctx, store, in.GetLocale(), damageTypes); err != nil {
		return nil, status.Errorf(codes.Internal, "localize damage types: %v", err)
	}
	if err := localizeDomains(ctx, store, in.GetLocale(), domains); err != nil {
		return nil, status.Errorf(codes.Internal, "localize domains: %v", err)
	}
	if err := localizeDomainCards(ctx, store, in.GetLocale(), domainCards); err != nil {
		return nil, status.Errorf(codes.Internal, "localize domain cards: %v", err)
	}
	if err := localizeWeapons(ctx, store, in.GetLocale(), weapons); err != nil {
		return nil, status.Errorf(codes.Internal, "localize weapons: %v", err)
	}
	if err := localizeArmor(ctx, store, in.GetLocale(), armor); err != nil {
		return nil, status.Errorf(codes.Internal, "localize armor: %v", err)
	}
	if err := localizeItems(ctx, store, in.GetLocale(), items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize items: %v", err)
	}
	if err := localizeEnvironments(ctx, store, in.GetLocale(), environments); err != nil {
		return nil, status.Errorf(codes.Internal, "localize environments: %v", err)
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

	class, err := store.GetDaggerheartClass(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get class", err)
	}
	classes := []storage.DaggerheartClass{class}
	if err := localizeClasses(ctx, store, in.GetLocale(), classes); err != nil {
		return nil, status.Errorf(codes.Internal, "localize classes: %v", err)
	}
	class = classes[0]

	return &pb.GetDaggerheartClassResponse{Class: toProtoDaggerheartClass(class)}, nil
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

	classes, err := store.ListDaggerheartClasses(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list classes: %v", err)
	}

	page, err := listContentPage(classes, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartClass]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":   contentfilter.FieldString,
			"name": contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartClass) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartClass, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list classes: %v", err)
	}
	if err := localizeClasses(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize classes: %v", err)
	}

	return &pb.ListDaggerheartClassesResponse{
		Classes:           toProtoDaggerheartClasses(page.Items),
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

	subclass, err := store.GetDaggerheartSubclass(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get subclass", err)
	}
	subclasses := []storage.DaggerheartSubclass{subclass}
	if err := localizeSubclasses(ctx, store, in.GetLocale(), subclasses); err != nil {
		return nil, status.Errorf(codes.Internal, "localize subclasses: %v", err)
	}
	subclass = subclasses[0]

	return &pb.GetDaggerheartSubclassResponse{Subclass: toProtoDaggerheartSubclass(subclass)}, nil
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

	subclasses, err := store.ListDaggerheartSubclasses(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list subclasses: %v", err)
	}

	page, err := listContentPage(subclasses, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartSubclass]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":              contentfilter.FieldString,
			"name":            contentfilter.FieldString,
			"spellcast_trait": contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartSubclass) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartSubclass, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			case "spellcast_trait":
				return item.SpellcastTrait, true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list subclasses: %v", err)
	}
	if err := localizeSubclasses(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize subclasses: %v", err)
	}

	return &pb.ListDaggerheartSubclassesResponse{
		Subclasses:        toProtoDaggerheartSubclasses(page.Items),
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

	heritage, err := store.GetDaggerheartHeritage(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get heritage", err)
	}
	heritages := []storage.DaggerheartHeritage{heritage}
	if err := localizeHeritages(ctx, store, in.GetLocale(), heritages); err != nil {
		return nil, status.Errorf(codes.Internal, "localize heritages: %v", err)
	}
	heritage = heritages[0]

	return &pb.GetDaggerheartHeritageResponse{Heritage: toProtoDaggerheartHeritage(heritage)}, nil
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

	heritages, err := store.ListDaggerheartHeritages(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list heritages: %v", err)
	}

	page, err := listContentPage(heritages, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartHeritage]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":   contentfilter.FieldString,
			"name": contentfilter.FieldString,
			"kind": contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartHeritage) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartHeritage, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			case "kind":
				return item.Kind, true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list heritages: %v", err)
	}
	if err := localizeHeritages(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize heritages: %v", err)
	}

	return &pb.ListDaggerheartHeritagesResponse{
		Heritages:         toProtoDaggerheartHeritages(page.Items),
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

	experience, err := store.GetDaggerheartExperience(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get experience", err)
	}
	experiences := []storage.DaggerheartExperienceEntry{experience}
	if err := localizeExperiences(ctx, store, in.GetLocale(), experiences); err != nil {
		return nil, status.Errorf(codes.Internal, "localize experiences: %v", err)
	}
	experience = experiences[0]

	return &pb.GetDaggerheartExperienceResponse{Experience: toProtoDaggerheartExperience(experience)}, nil
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

	experiences, err := store.ListDaggerheartExperiences(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list experiences: %v", err)
	}

	page, err := listContentPage(experiences, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartExperienceEntry]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":   contentfilter.FieldString,
			"name": contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartExperienceEntry) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartExperienceEntry, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list experiences: %v", err)
	}
	if err := localizeExperiences(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize experiences: %v", err)
	}

	return &pb.ListDaggerheartExperiencesResponse{
		Experiences:       toProtoDaggerheartExperiences(page.Items),
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

	adversary, err := store.GetDaggerheartAdversaryEntry(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get adversary", err)
	}
	adversaries := []storage.DaggerheartAdversaryEntry{adversary}
	if err := localizeAdversaries(ctx, store, in.GetLocale(), adversaries); err != nil {
		return nil, status.Errorf(codes.Internal, "localize adversaries: %v", err)
	}
	adversary = adversaries[0]

	return &pb.GetDaggerheartAdversaryResponse{Adversary: toProtoDaggerheartAdversaryEntry(adversary)}, nil
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

	adversaries, err := store.ListDaggerheartAdversaryEntries(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list adversaries: %v", err)
	}

	page, err := listContentPage(adversaries, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartAdversaryEntry]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":   contentfilter.FieldString,
			"name": contentfilter.FieldString,
			"tier": contentfilter.FieldInt,
			"role": contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartAdversaryEntry) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartAdversaryEntry, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			case "tier":
				return int64(item.Tier), true
			case "role":
				return item.Role, true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list adversaries: %v", err)
	}
	if err := localizeAdversaries(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize adversaries: %v", err)
	}

	return &pb.ListDaggerheartAdversariesResponse{
		Adversaries:       toProtoDaggerheartAdversaryEntries(page.Items),
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

	beastform, err := store.GetDaggerheartBeastform(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get beastform", err)
	}
	beastforms := []storage.DaggerheartBeastformEntry{beastform}
	if err := localizeBeastforms(ctx, store, in.GetLocale(), beastforms); err != nil {
		return nil, status.Errorf(codes.Internal, "localize beastforms: %v", err)
	}
	beastform = beastforms[0]

	return &pb.GetDaggerheartBeastformResponse{Beastform: toProtoDaggerheartBeastform(beastform)}, nil
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

	beastforms, err := store.ListDaggerheartBeastforms(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list beastforms: %v", err)
	}

	page, err := listContentPage(beastforms, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartBeastformEntry]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":    contentfilter.FieldString,
			"name":  contentfilter.FieldString,
			"tier":  contentfilter.FieldInt,
			"trait": contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartBeastformEntry) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartBeastformEntry, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			case "tier":
				return int64(item.Tier), true
			case "trait":
				return item.Trait, true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list beastforms: %v", err)
	}
	if err := localizeBeastforms(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize beastforms: %v", err)
	}

	return &pb.ListDaggerheartBeastformsResponse{
		Beastforms:        toProtoDaggerheartBeastforms(page.Items),
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

	experience, err := store.GetDaggerheartCompanionExperience(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get companion experience", err)
	}
	experiences := []storage.DaggerheartCompanionExperienceEntry{experience}
	if err := localizeCompanionExperiences(ctx, store, in.GetLocale(), experiences); err != nil {
		return nil, status.Errorf(codes.Internal, "localize companion experiences: %v", err)
	}
	experience = experiences[0]

	return &pb.GetDaggerheartCompanionExperienceResponse{Experience: toProtoDaggerheartCompanionExperience(experience)}, nil
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

	experiences, err := store.ListDaggerheartCompanionExperiences(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list companion experiences: %v", err)
	}

	page, err := listContentPage(experiences, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartCompanionExperienceEntry]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":   contentfilter.FieldString,
			"name": contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartCompanionExperienceEntry) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartCompanionExperienceEntry, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list companion experiences: %v", err)
	}
	if err := localizeCompanionExperiences(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize companion experiences: %v", err)
	}

	return &pb.ListDaggerheartCompanionExperiencesResponse{
		Experiences:       toProtoDaggerheartCompanionExperiences(page.Items),
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

	entry, err := store.GetDaggerheartLootEntry(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get loot entry", err)
	}
	entries := []storage.DaggerheartLootEntry{entry}
	if err := localizeLootEntries(ctx, store, in.GetLocale(), entries); err != nil {
		return nil, status.Errorf(codes.Internal, "localize loot entries: %v", err)
	}
	entry = entries[0]

	return &pb.GetDaggerheartLootEntryResponse{Entry: toProtoDaggerheartLootEntry(entry)}, nil
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

	entries, err := store.ListDaggerheartLootEntries(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list loot entries: %v", err)
	}

	page, err := listContentPage(entries, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartLootEntry]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "roll",
			Allowed: []string{"roll", "roll desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":   contentfilter.FieldString,
			"name": contentfilter.FieldString,
			"roll": contentfilter.FieldInt,
		},
		KeySpec: []contentKeySpec{
			{Name: "roll", Kind: pagination.CursorValueInt},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartLootEntry) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.IntValue("roll", int64(item.Roll)),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartLootEntry, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			case "roll":
				return int64(item.Roll), true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list loot entries: %v", err)
	}
	if err := localizeLootEntries(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize loot entries: %v", err)
	}

	return &pb.ListDaggerheartLootEntriesResponse{
		Entries:           toProtoDaggerheartLootEntries(page.Items),
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

	entry, err := store.GetDaggerheartDamageType(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get damage type", err)
	}
	entries := []storage.DaggerheartDamageTypeEntry{entry}
	if err := localizeDamageTypes(ctx, store, in.GetLocale(), entries); err != nil {
		return nil, status.Errorf(codes.Internal, "localize damage types: %v", err)
	}
	entry = entries[0]

	return &pb.GetDaggerheartDamageTypeResponse{DamageType: toProtoDaggerheartDamageType(entry)}, nil
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

	entries, err := store.ListDaggerheartDamageTypes(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list damage types: %v", err)
	}

	page, err := listContentPage(entries, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartDamageTypeEntry]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":   contentfilter.FieldString,
			"name": contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartDamageTypeEntry) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartDamageTypeEntry, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list damage types: %v", err)
	}
	if err := localizeDamageTypes(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize damage types: %v", err)
	}

	return &pb.ListDaggerheartDamageTypesResponse{
		DamageTypes:       toProtoDaggerheartDamageTypes(page.Items),
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

	domain, err := store.GetDaggerheartDomain(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get domain", err)
	}
	domains := []storage.DaggerheartDomain{domain}
	if err := localizeDomains(ctx, store, in.GetLocale(), domains); err != nil {
		return nil, status.Errorf(codes.Internal, "localize domains: %v", err)
	}
	domain = domains[0]

	return &pb.GetDaggerheartDomainResponse{Domain: toProtoDaggerheartDomain(domain)}, nil
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

	domains, err := store.ListDaggerheartDomains(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list domains: %v", err)
	}

	page, err := listContentPage(domains, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartDomain]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":   contentfilter.FieldString,
			"name": contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartDomain) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartDomain, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list domains: %v", err)
	}
	if err := localizeDomains(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize domains: %v", err)
	}

	return &pb.ListDaggerheartDomainsResponse{
		Domains:           toProtoDaggerheartDomains(page.Items),
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

	card, err := store.GetDaggerheartDomainCard(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get domain card", err)
	}
	cards := []storage.DaggerheartDomainCard{card}
	if err := localizeDomainCards(ctx, store, in.GetLocale(), cards); err != nil {
		return nil, status.Errorf(codes.Internal, "localize domain cards: %v", err)
	}
	card = cards[0]

	return &pb.GetDaggerheartDomainCardResponse{DomainCard: toProtoDaggerheartDomainCard(card)}, nil
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

	var cards []storage.DaggerheartDomainCard
	domainID := strings.TrimSpace(in.GetDomainId())
	if domainID != "" {
		cards, err = store.ListDaggerheartDomainCardsByDomain(ctx, domainID)
	} else {
		cards, err = store.ListDaggerheartDomainCards(ctx)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list domain cards: %v", err)
	}

	filterHashSeed := ""
	if domainID != "" {
		filterHashSeed = "domain_id=" + domainID
	}

	page, err := listContentPage(cards, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartDomainCard]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "level",
			Allowed: []string{"level", "level desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":        contentfilter.FieldString,
			"name":      contentfilter.FieldString,
			"domain_id": contentfilter.FieldString,
			"level":     contentfilter.FieldInt,
			"type":      contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{
			{Name: "level", Kind: pagination.CursorValueInt},
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartDomainCard) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.IntValue("level", int64(item.Level)),
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartDomainCard, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			case "domain_id":
				return item.DomainID, true
			case "level":
				return int64(item.Level), true
			case "type":
				return item.Type, true
			default:
				return nil, false
			}
		},
		FilterHashSeed: filterHashSeed,
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list domain cards: %v", err)
	}
	if err := localizeDomainCards(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize domain cards: %v", err)
	}

	return &pb.ListDaggerheartDomainCardsResponse{
		DomainCards:       toProtoDaggerheartDomainCards(page.Items),
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

	weapon, err := store.GetDaggerheartWeapon(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get weapon", err)
	}
	weapons := []storage.DaggerheartWeapon{weapon}
	if err := localizeWeapons(ctx, store, in.GetLocale(), weapons); err != nil {
		return nil, status.Errorf(codes.Internal, "localize weapons: %v", err)
	}
	weapon = weapons[0]

	return &pb.GetDaggerheartWeaponResponse{Weapon: toProtoDaggerheartWeapon(weapon)}, nil
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

	weapons, err := store.ListDaggerheartWeapons(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list weapons: %v", err)
	}

	page, err := listContentPage(weapons, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartWeapon]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":          contentfilter.FieldString,
			"name":        contentfilter.FieldString,
			"category":    contentfilter.FieldString,
			"tier":        contentfilter.FieldInt,
			"trait":       contentfilter.FieldString,
			"damage_type": contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartWeapon) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartWeapon, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			case "category":
				return item.Category, true
			case "tier":
				return int64(item.Tier), true
			case "trait":
				return item.Trait, true
			case "damage_type":
				return item.DamageType, true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list weapons: %v", err)
	}
	if err := localizeWeapons(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize weapons: %v", err)
	}

	return &pb.ListDaggerheartWeaponsResponse{
		Weapons:           toProtoDaggerheartWeapons(page.Items),
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

	armor, err := store.GetDaggerheartArmor(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get armor", err)
	}
	armorItems := []storage.DaggerheartArmor{armor}
	if err := localizeArmor(ctx, store, in.GetLocale(), armorItems); err != nil {
		return nil, status.Errorf(codes.Internal, "localize armor: %v", err)
	}
	armor = armorItems[0]

	return &pb.GetDaggerheartArmorResponse{Armor: toProtoDaggerheartArmor(armor)}, nil
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

	armor, err := store.ListDaggerheartArmor(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list armor: %v", err)
	}

	page, err := listContentPage(armor, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartArmor]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":   contentfilter.FieldString,
			"name": contentfilter.FieldString,
			"tier": contentfilter.FieldInt,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartArmor) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartArmor, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			case "tier":
				return int64(item.Tier), true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list armor: %v", err)
	}
	if err := localizeArmor(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize armor: %v", err)
	}

	return &pb.ListDaggerheartArmorResponse{
		Armor:             toProtoDaggerheartArmorList(page.Items),
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

	item, err := store.GetDaggerheartItem(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get item", err)
	}
	items := []storage.DaggerheartItem{item}
	if err := localizeItems(ctx, store, in.GetLocale(), items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize items: %v", err)
	}
	item = items[0]

	return &pb.GetDaggerheartItemResponse{Item: toProtoDaggerheartItem(item)}, nil
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

	items, err := store.ListDaggerheartItems(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list items: %v", err)
	}

	page, err := listContentPage(items, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartItem]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":     contentfilter.FieldString,
			"name":   contentfilter.FieldString,
			"rarity": contentfilter.FieldString,
			"kind":   contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartItem) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartItem, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			case "rarity":
				return item.Rarity, true
			case "kind":
				return item.Kind, true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list items: %v", err)
	}
	if err := localizeItems(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize items: %v", err)
	}

	return &pb.ListDaggerheartItemsResponse{
		Items:             toProtoDaggerheartItems(page.Items),
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

	env, err := store.GetDaggerheartEnvironment(ctx, in.GetId())
	if err != nil {
		return nil, mapContentErr("get environment", err)
	}
	envs := []storage.DaggerheartEnvironment{env}
	if err := localizeEnvironments(ctx, store, in.GetLocale(), envs); err != nil {
		return nil, status.Errorf(codes.Internal, "localize environments: %v", err)
	}
	env = envs[0]

	return &pb.GetDaggerheartEnvironmentResponse{Environment: toProtoDaggerheartEnvironment(env)}, nil
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

	items, err := store.ListDaggerheartEnvironments(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list environments: %v", err)
	}

	page, err := listContentPage(items, contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}, contentListConfig[storage.DaggerheartEnvironment]{
		PageSizeConfig: pagination.PageSizeConfig{
			Default: defaultListDaggerheartContentPageSize,
			Max:     maxListDaggerheartContentPageSize,
		},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":         contentfilter.FieldString,
			"name":       contentfilter.FieldString,
			"tier":       contentfilter.FieldInt,
			"type":       contentfilter.FieldString,
			"difficulty": contentfilter.FieldInt,
		},
		KeySpec: []contentKeySpec{
			{Name: "name", Kind: pagination.CursorValueString},
			{Name: "id", Kind: pagination.CursorValueString},
		},
		KeyFunc: func(item storage.DaggerheartEnvironment) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item storage.DaggerheartEnvironment, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			case "tier":
				return int64(item.Tier), true
			case "type":
				return item.Type, true
			case "difficulty":
				return int64(item.Difficulty), true
			default:
				return nil, false
			}
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list environments: %v", err)
	}
	if err := localizeEnvironments(ctx, store, in.GetLocale(), page.Items); err != nil {
		return nil, status.Errorf(codes.Internal, "localize environments: %v", err)
	}

	return &pb.ListDaggerheartEnvironmentsResponse{
		Environments:      toProtoDaggerheartEnvironments(page.Items),
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
