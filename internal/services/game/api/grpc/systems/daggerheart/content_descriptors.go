package daggerheart

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/content/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contentDescriptor[T any, P any] struct {
	getAction      string
	listAction     string
	localizeAction string
	get            func(context.Context, storage.DaggerheartContentReadStore, string) (T, error)
	list           func(context.Context, storage.DaggerheartContentReadStore) ([]T, error)
	listByRequest  func(context.Context, storage.DaggerheartContentReadStore, contentListRequest) ([]T, error)
	filterHashSeed func(contentListRequest) string
	localize       func(context.Context, storage.DaggerheartContentReadStore, commonv1.Locale, []T) error
	toProto        func(T) *P
	toProtoList    func([]T) []*P
	listConfig     contentListConfig[T]
}

func getContentEntry[T any, P any](
	ctx context.Context,
	store storage.DaggerheartContentReadStore,
	id string,
	locale commonv1.Locale,
	descriptor contentDescriptor[T, P],
) (*P, error) {
	item, err := descriptor.get(ctx, store, id)
	if err != nil {
		return nil, mapContentErr(descriptor.getAction, err)
	}
	items := []T{item}
	if err := descriptor.localize(ctx, store, locale, items); err != nil {
		return nil, status.Errorf(codes.Internal, "%s: %v", descriptor.localizeAction, err)
	}
	return descriptor.toProto(items[0]), nil
}

func listContentEntries[T any, P any](
	ctx context.Context,
	store storage.DaggerheartContentReadStore,
	req contentListRequest,
	locale commonv1.Locale,
	descriptor contentDescriptor[T, P],
) ([]*P, contentPage[T], error) {
	listFunc := descriptor.list
	if descriptor.listByRequest != nil {
		listFunc = func(listCtx context.Context, listStore storage.DaggerheartContentReadStore) ([]T, error) {
			return descriptor.listByRequest(listCtx, listStore, req)
		}
	}
	items, err := listFunc(ctx, store)
	if err != nil {
		return nil, contentPage[T]{}, status.Errorf(codes.Internal, "%s: %v", descriptor.listAction, err)
	}
	listConfig := descriptor.listConfig
	if descriptor.filterHashSeed != nil {
		listConfig.FilterHashSeed = descriptor.filterHashSeed(req)
	}
	page, err := listContentPage(items, req, listConfig)
	if err != nil {
		return nil, contentPage[T]{}, status.Errorf(codes.InvalidArgument, "%s: %v", descriptor.listAction, err)
	}
	if err := descriptor.localize(ctx, store, locale, page.Items); err != nil {
		return nil, contentPage[T]{}, status.Errorf(codes.Internal, "%s: %v", descriptor.localizeAction, err)
	}
	return descriptor.toProtoList(page.Items), page, nil
}

var classDescriptor = contentDescriptor[storage.DaggerheartClass, pb.DaggerheartClass]{
	getAction:      "get class",
	listAction:     "list classes",
	localizeAction: "localize classes",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartClass, error) {
		return store.GetDaggerheartClass(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartClass, error) {
		return store.ListDaggerheartClasses(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartClass) error {
		return localizeClasses(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartClass,
	toProtoList: toProtoDaggerheartClasses,
	listConfig: contentListConfig[storage.DaggerheartClass]{
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
	},
}

var subclassDescriptor = contentDescriptor[storage.DaggerheartSubclass, pb.DaggerheartSubclass]{
	getAction:      "get subclass",
	listAction:     "list subclasses",
	localizeAction: "localize subclasses",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartSubclass, error) {
		return store.GetDaggerheartSubclass(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartSubclass, error) {
		return store.ListDaggerheartSubclasses(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartSubclass) error {
		return localizeSubclasses(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartSubclass,
	toProtoList: toProtoDaggerheartSubclasses,
	listConfig: contentListConfig[storage.DaggerheartSubclass]{
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
			"class_id":        contentfilter.FieldString,
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
			case "class_id":
				return item.ClassID, true
			case "spellcast_trait":
				return item.SpellcastTrait, true
			default:
				return nil, false
			}
		},
	},
}

var heritageDescriptor = contentDescriptor[storage.DaggerheartHeritage, pb.DaggerheartHeritage]{
	getAction:      "get heritage",
	listAction:     "list heritages",
	localizeAction: "localize heritages",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartHeritage, error) {
		return store.GetDaggerheartHeritage(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartHeritage, error) {
		return store.ListDaggerheartHeritages(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartHeritage) error {
		return localizeHeritages(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartHeritage,
	toProtoList: toProtoDaggerheartHeritages,
	listConfig: contentListConfig[storage.DaggerheartHeritage]{
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
	},
}

var experienceDescriptor = contentDescriptor[storage.DaggerheartExperienceEntry, pb.DaggerheartExperienceEntry]{
	getAction:      "get experience",
	listAction:     "list experiences",
	localizeAction: "localize experiences",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartExperienceEntry, error) {
		return store.GetDaggerheartExperience(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartExperienceEntry, error) {
		return store.ListDaggerheartExperiences(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartExperienceEntry) error {
		return localizeExperiences(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartExperience,
	toProtoList: toProtoDaggerheartExperiences,
	listConfig: contentListConfig[storage.DaggerheartExperienceEntry]{
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
	},
}

var adversaryDescriptor = contentDescriptor[storage.DaggerheartAdversaryEntry, pb.DaggerheartAdversaryEntry]{
	getAction:      "get adversary",
	listAction:     "list adversaries",
	localizeAction: "localize adversaries",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartAdversaryEntry, error) {
		return store.GetDaggerheartAdversaryEntry(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartAdversaryEntry, error) {
		return store.ListDaggerheartAdversaryEntries(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartAdversaryEntry) error {
		return localizeAdversaries(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartAdversaryEntry,
	toProtoList: toProtoDaggerheartAdversaryEntries,
	listConfig: contentListConfig[storage.DaggerheartAdversaryEntry]{
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
	},
}

var beastformDescriptor = contentDescriptor[storage.DaggerheartBeastformEntry, pb.DaggerheartBeastformEntry]{
	getAction:      "get beastform",
	listAction:     "list beastforms",
	localizeAction: "localize beastforms",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartBeastformEntry, error) {
		return store.GetDaggerheartBeastform(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartBeastformEntry, error) {
		return store.ListDaggerheartBeastforms(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartBeastformEntry) error {
		return localizeBeastforms(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartBeastform,
	toProtoList: toProtoDaggerheartBeastforms,
	listConfig: contentListConfig[storage.DaggerheartBeastformEntry]{
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
	},
}

var companionExperienceDescriptor = contentDescriptor[storage.DaggerheartCompanionExperienceEntry, pb.DaggerheartCompanionExperienceEntry]{
	getAction:      "get companion experience",
	listAction:     "list companion experiences",
	localizeAction: "localize companion experiences",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartCompanionExperienceEntry, error) {
		return store.GetDaggerheartCompanionExperience(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartCompanionExperienceEntry, error) {
		return store.ListDaggerheartCompanionExperiences(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartCompanionExperienceEntry) error {
		return localizeCompanionExperiences(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartCompanionExperience,
	toProtoList: toProtoDaggerheartCompanionExperiences,
	listConfig: contentListConfig[storage.DaggerheartCompanionExperienceEntry]{
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
	},
}

var lootEntryDescriptor = contentDescriptor[storage.DaggerheartLootEntry, pb.DaggerheartLootEntry]{
	getAction:      "get loot entry",
	listAction:     "list loot entries",
	localizeAction: "localize loot entries",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartLootEntry, error) {
		return store.GetDaggerheartLootEntry(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartLootEntry, error) {
		return store.ListDaggerheartLootEntries(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartLootEntry) error {
		return localizeLootEntries(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartLootEntry,
	toProtoList: toProtoDaggerheartLootEntries,
	listConfig: contentListConfig[storage.DaggerheartLootEntry]{
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
	},
}

var damageTypeDescriptor = contentDescriptor[storage.DaggerheartDamageTypeEntry, pb.DaggerheartDamageTypeEntry]{
	getAction:      "get damage type",
	listAction:     "list damage types",
	localizeAction: "localize damage types",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartDamageTypeEntry, error) {
		return store.GetDaggerheartDamageType(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartDamageTypeEntry, error) {
		return store.ListDaggerheartDamageTypes(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartDamageTypeEntry) error {
		return localizeDamageTypes(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartDamageType,
	toProtoList: toProtoDaggerheartDamageTypes,
	listConfig: contentListConfig[storage.DaggerheartDamageTypeEntry]{
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
	},
}

var domainDescriptor = contentDescriptor[storage.DaggerheartDomain, pb.DaggerheartDomain]{
	getAction:      "get domain",
	listAction:     "list domains",
	localizeAction: "localize domains",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartDomain, error) {
		return store.GetDaggerheartDomain(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartDomain, error) {
		return store.ListDaggerheartDomains(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartDomain) error {
		return localizeDomains(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartDomain,
	toProtoList: toProtoDaggerheartDomains,
	listConfig: contentListConfig[storage.DaggerheartDomain]{
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
	},
}

var domainCardDescriptor = contentDescriptor[storage.DaggerheartDomainCard, pb.DaggerheartDomainCard]{
	getAction:      "get domain card",
	listAction:     "list domain cards",
	localizeAction: "localize domain cards",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartDomainCard, error) {
		return store.GetDaggerheartDomainCard(ctx, id)
	},
	listByRequest: func(ctx context.Context, store storage.DaggerheartContentReadStore, req contentListRequest) ([]storage.DaggerheartDomainCard, error) {
		if req.DomainID != "" {
			return store.ListDaggerheartDomainCardsByDomain(ctx, req.DomainID)
		}
		return store.ListDaggerheartDomainCards(ctx)
	},
	filterHashSeed: func(req contentListRequest) string {
		if req.DomainID == "" {
			return ""
		}
		return "domain_id=" + req.DomainID
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartDomainCard) error {
		return localizeDomainCards(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartDomainCard,
	toProtoList: toProtoDaggerheartDomainCards,
	listConfig: contentListConfig[storage.DaggerheartDomainCard]{
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
	},
}

var weaponDescriptor = contentDescriptor[storage.DaggerheartWeapon, pb.DaggerheartWeapon]{
	getAction:      "get weapon",
	listAction:     "list weapons",
	localizeAction: "localize weapons",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartWeapon, error) {
		return store.GetDaggerheartWeapon(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartWeapon, error) {
		return store.ListDaggerheartWeapons(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartWeapon) error {
		return localizeWeapons(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartWeapon,
	toProtoList: toProtoDaggerheartWeapons,
	listConfig: contentListConfig[storage.DaggerheartWeapon]{
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
	},
}

var armorDescriptor = contentDescriptor[storage.DaggerheartArmor, pb.DaggerheartArmor]{
	getAction:      "get armor",
	listAction:     "list armor",
	localizeAction: "localize armor",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartArmor, error) {
		return store.GetDaggerheartArmor(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartArmor, error) {
		return store.ListDaggerheartArmor(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartArmor) error {
		return localizeArmor(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartArmor,
	toProtoList: toProtoDaggerheartArmorList,
	listConfig: contentListConfig[storage.DaggerheartArmor]{
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
	},
}

var itemDescriptor = contentDescriptor[storage.DaggerheartItem, pb.DaggerheartItem]{
	getAction:      "get item",
	listAction:     "list items",
	localizeAction: "localize items",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartItem, error) {
		return store.GetDaggerheartItem(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartItem, error) {
		return store.ListDaggerheartItems(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartItem) error {
		return localizeItems(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartItem,
	toProtoList: toProtoDaggerheartItems,
	listConfig: contentListConfig[storage.DaggerheartItem]{
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
	},
}

var environmentDescriptor = contentDescriptor[storage.DaggerheartEnvironment, pb.DaggerheartEnvironment]{
	getAction:      "get environment",
	listAction:     "list environments",
	localizeAction: "localize environments",
	get: func(ctx context.Context, store storage.DaggerheartContentReadStore, id string) (storage.DaggerheartEnvironment, error) {
		return store.GetDaggerheartEnvironment(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentReadStore) ([]storage.DaggerheartEnvironment, error) {
		return store.ListDaggerheartEnvironments(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentReadStore, locale commonv1.Locale, items []storage.DaggerheartEnvironment) error {
		return localizeEnvironments(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartEnvironment,
	toProtoList: toProtoDaggerheartEnvironments,
	listConfig: contentListConfig[storage.DaggerheartEnvironment]{
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
	},
}
