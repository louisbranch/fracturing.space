package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/content/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contentDescriptor[T any, P any] struct {
	getAction      string
	listAction     string
	localizeAction string
	get            func(context.Context, contentstore.DaggerheartContentReadStore, string) (T, error)
	list           func(context.Context, contentstore.DaggerheartContentReadStore) ([]T, error)
	listByRequest  func(context.Context, contentstore.DaggerheartContentReadStore, contentListRequest) ([]T, error)
	filterHashSeed func(contentListRequest) string
	localize       func(context.Context, contentstore.DaggerheartContentReadStore, commonv1.Locale, []T) error
	toProto        func(T) *P
	toProtoList    func([]T) []*P
	listConfig     contentListConfig[T]
}

func getContentEntry[T any, P any](
	ctx context.Context,
	store contentstore.DaggerheartContentReadStore,
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
		return nil, grpcerror.Internal(descriptor.localizeAction, err)
	}
	return descriptor.toProto(items[0]), nil
}

func listContentEntries[T any, P any](
	ctx context.Context,
	store contentstore.DaggerheartContentReadStore,
	req contentListRequest,
	locale commonv1.Locale,
	descriptor contentDescriptor[T, P],
) ([]*P, contentPage[T], error) {
	listFunc := descriptor.list
	if descriptor.listByRequest != nil {
		listFunc = func(listCtx context.Context, listStore contentstore.DaggerheartContentReadStore) ([]T, error) {
			return descriptor.listByRequest(listCtx, listStore, req)
		}
	}
	items, err := listFunc(ctx, store)
	if err != nil {
		return nil, contentPage[T]{}, grpcerror.Internal(descriptor.listAction, err)
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
		return nil, contentPage[T]{}, grpcerror.Internal(descriptor.localizeAction, err)
	}
	return descriptor.toProtoList(page.Items), page, nil
}

var classDescriptor = contentDescriptor[contentstore.DaggerheartClass, pb.DaggerheartClass]{
	getAction:      "get class",
	listAction:     "list classes",
	localizeAction: "localize classes",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartClass, error) {
		return store.GetDaggerheartClass(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartClass, error) {
		return store.ListDaggerheartClasses(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartClass) error {
		return localizeClasses(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartClass,
	toProtoList: toProtoDaggerheartClasses,
	listConfig: contentListConfig[contentstore.DaggerheartClass]{
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
		KeyFunc: func(item contentstore.DaggerheartClass) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartClass, field string) (any, bool) {
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

var subclassDescriptor = contentDescriptor[contentstore.DaggerheartSubclass, pb.DaggerheartSubclass]{
	getAction:      "get subclass",
	listAction:     "list subclasses",
	localizeAction: "localize subclasses",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartSubclass, error) {
		return store.GetDaggerheartSubclass(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartSubclass, error) {
		return store.ListDaggerheartSubclasses(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartSubclass) error {
		return localizeSubclasses(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartSubclass,
	toProtoList: toProtoDaggerheartSubclasses,
	listConfig: contentListConfig[contentstore.DaggerheartSubclass]{
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
		KeyFunc: func(item contentstore.DaggerheartSubclass) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartSubclass, field string) (any, bool) {
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

var heritageDescriptor = contentDescriptor[contentstore.DaggerheartHeritage, pb.DaggerheartHeritage]{
	getAction:      "get heritage",
	listAction:     "list heritages",
	localizeAction: "localize heritages",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartHeritage, error) {
		return store.GetDaggerheartHeritage(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartHeritage, error) {
		return store.ListDaggerheartHeritages(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartHeritage) error {
		return localizeHeritages(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartHeritage,
	toProtoList: toProtoDaggerheartHeritages,
	listConfig: contentListConfig[contentstore.DaggerheartHeritage]{
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
		KeyFunc: func(item contentstore.DaggerheartHeritage) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartHeritage, field string) (any, bool) {
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

var experienceDescriptor = contentDescriptor[contentstore.DaggerheartExperienceEntry, pb.DaggerheartExperienceEntry]{
	getAction:      "get experience",
	listAction:     "list experiences",
	localizeAction: "localize experiences",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartExperienceEntry, error) {
		return store.GetDaggerheartExperience(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartExperienceEntry, error) {
		return store.ListDaggerheartExperiences(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartExperienceEntry) error {
		return localizeExperiences(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartExperience,
	toProtoList: toProtoDaggerheartExperiences,
	listConfig: contentListConfig[contentstore.DaggerheartExperienceEntry]{
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
		KeyFunc: func(item contentstore.DaggerheartExperienceEntry) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartExperienceEntry, field string) (any, bool) {
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

var adversaryDescriptor = contentDescriptor[contentstore.DaggerheartAdversaryEntry, pb.DaggerheartAdversaryEntry]{
	getAction:      "get adversary",
	listAction:     "list adversaries",
	localizeAction: "localize adversaries",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartAdversaryEntry, error) {
		return store.GetDaggerheartAdversaryEntry(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartAdversaryEntry, error) {
		return store.ListDaggerheartAdversaryEntries(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartAdversaryEntry) error {
		return localizeAdversaries(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartAdversaryEntry,
	toProtoList: toProtoDaggerheartAdversaryEntries,
	listConfig: contentListConfig[contentstore.DaggerheartAdversaryEntry]{
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
		KeyFunc: func(item contentstore.DaggerheartAdversaryEntry) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartAdversaryEntry, field string) (any, bool) {
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

var beastformDescriptor = contentDescriptor[contentstore.DaggerheartBeastformEntry, pb.DaggerheartBeastformEntry]{
	getAction:      "get beastform",
	listAction:     "list beastforms",
	localizeAction: "localize beastforms",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartBeastformEntry, error) {
		return store.GetDaggerheartBeastform(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartBeastformEntry, error) {
		return store.ListDaggerheartBeastforms(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartBeastformEntry) error {
		return localizeBeastforms(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartBeastform,
	toProtoList: toProtoDaggerheartBeastforms,
	listConfig: contentListConfig[contentstore.DaggerheartBeastformEntry]{
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
		KeyFunc: func(item contentstore.DaggerheartBeastformEntry) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartBeastformEntry, field string) (any, bool) {
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

var companionExperienceDescriptor = contentDescriptor[contentstore.DaggerheartCompanionExperienceEntry, pb.DaggerheartCompanionExperienceEntry]{
	getAction:      "get companion experience",
	listAction:     "list companion experiences",
	localizeAction: "localize companion experiences",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartCompanionExperienceEntry, error) {
		return store.GetDaggerheartCompanionExperience(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartCompanionExperienceEntry, error) {
		return store.ListDaggerheartCompanionExperiences(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartCompanionExperienceEntry) error {
		return localizeCompanionExperiences(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartCompanionExperience,
	toProtoList: toProtoDaggerheartCompanionExperiences,
	listConfig: contentListConfig[contentstore.DaggerheartCompanionExperienceEntry]{
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
		KeyFunc: func(item contentstore.DaggerheartCompanionExperienceEntry) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartCompanionExperienceEntry, field string) (any, bool) {
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

var lootEntryDescriptor = contentDescriptor[contentstore.DaggerheartLootEntry, pb.DaggerheartLootEntry]{
	getAction:      "get loot entry",
	listAction:     "list loot entries",
	localizeAction: "localize loot entries",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartLootEntry, error) {
		return store.GetDaggerheartLootEntry(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartLootEntry, error) {
		return store.ListDaggerheartLootEntries(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartLootEntry) error {
		return localizeLootEntries(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartLootEntry,
	toProtoList: toProtoDaggerheartLootEntries,
	listConfig: contentListConfig[contentstore.DaggerheartLootEntry]{
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
		KeyFunc: func(item contentstore.DaggerheartLootEntry) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.IntValue("roll", int64(item.Roll)),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartLootEntry, field string) (any, bool) {
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

var damageTypeDescriptor = contentDescriptor[contentstore.DaggerheartDamageTypeEntry, pb.DaggerheartDamageTypeEntry]{
	getAction:      "get damage type",
	listAction:     "list damage types",
	localizeAction: "localize damage types",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartDamageTypeEntry, error) {
		return store.GetDaggerheartDamageType(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartDamageTypeEntry, error) {
		return store.ListDaggerheartDamageTypes(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartDamageTypeEntry) error {
		return localizeDamageTypes(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartDamageType,
	toProtoList: toProtoDaggerheartDamageTypes,
	listConfig: contentListConfig[contentstore.DaggerheartDamageTypeEntry]{
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
		KeyFunc: func(item contentstore.DaggerheartDamageTypeEntry) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartDamageTypeEntry, field string) (any, bool) {
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

var domainDescriptor = contentDescriptor[contentstore.DaggerheartDomain, pb.DaggerheartDomain]{
	getAction:      "get domain",
	listAction:     "list domains",
	localizeAction: "localize domains",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartDomain, error) {
		return store.GetDaggerheartDomain(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartDomain, error) {
		return store.ListDaggerheartDomains(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartDomain) error {
		return localizeDomains(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartDomain,
	toProtoList: toProtoDaggerheartDomains,
	listConfig: contentListConfig[contentstore.DaggerheartDomain]{
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
		KeyFunc: func(item contentstore.DaggerheartDomain) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartDomain, field string) (any, bool) {
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

var domainCardDescriptor = contentDescriptor[contentstore.DaggerheartDomainCard, pb.DaggerheartDomainCard]{
	getAction:      "get domain card",
	listAction:     "list domain cards",
	localizeAction: "localize domain cards",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartDomainCard, error) {
		return store.GetDaggerheartDomainCard(ctx, id)
	},
	listByRequest: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, req contentListRequest) ([]contentstore.DaggerheartDomainCard, error) {
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
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartDomainCard) error {
		return localizeDomainCards(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartDomainCard,
	toProtoList: toProtoDaggerheartDomainCards,
	listConfig: contentListConfig[contentstore.DaggerheartDomainCard]{
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
		KeyFunc: func(item contentstore.DaggerheartDomainCard) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.IntValue("level", int64(item.Level)),
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartDomainCard, field string) (any, bool) {
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

var weaponDescriptor = contentDescriptor[contentstore.DaggerheartWeapon, pb.DaggerheartWeapon]{
	getAction:      "get weapon",
	listAction:     "list weapons",
	localizeAction: "localize weapons",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartWeapon, error) {
		return store.GetDaggerheartWeapon(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartWeapon, error) {
		return store.ListDaggerheartWeapons(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartWeapon) error {
		return localizeWeapons(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartWeapon,
	toProtoList: toProtoDaggerheartWeapons,
	listConfig: contentListConfig[contentstore.DaggerheartWeapon]{
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
		KeyFunc: func(item contentstore.DaggerheartWeapon) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartWeapon, field string) (any, bool) {
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

var armorDescriptor = contentDescriptor[contentstore.DaggerheartArmor, pb.DaggerheartArmor]{
	getAction:      "get armor",
	listAction:     "list armor",
	localizeAction: "localize armor",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartArmor, error) {
		return store.GetDaggerheartArmor(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartArmor, error) {
		return store.ListDaggerheartArmor(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartArmor) error {
		return localizeArmor(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartArmor,
	toProtoList: toProtoDaggerheartArmorList,
	listConfig: contentListConfig[contentstore.DaggerheartArmor]{
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
		KeyFunc: func(item contentstore.DaggerheartArmor) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartArmor, field string) (any, bool) {
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

var itemDescriptor = contentDescriptor[contentstore.DaggerheartItem, pb.DaggerheartItem]{
	getAction:      "get item",
	listAction:     "list items",
	localizeAction: "localize items",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartItem, error) {
		return store.GetDaggerheartItem(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartItem, error) {
		return store.ListDaggerheartItems(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartItem) error {
		return localizeItems(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartItem,
	toProtoList: toProtoDaggerheartItems,
	listConfig: contentListConfig[contentstore.DaggerheartItem]{
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
		KeyFunc: func(item contentstore.DaggerheartItem) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartItem, field string) (any, bool) {
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

var environmentDescriptor = contentDescriptor[contentstore.DaggerheartEnvironment, pb.DaggerheartEnvironment]{
	getAction:      "get environment",
	listAction:     "list environments",
	localizeAction: "localize environments",
	get: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, id string) (contentstore.DaggerheartEnvironment, error) {
		return store.GetDaggerheartEnvironment(ctx, id)
	},
	list: func(ctx context.Context, store contentstore.DaggerheartContentReadStore) ([]contentstore.DaggerheartEnvironment, error) {
		return store.ListDaggerheartEnvironments(ctx)
	},
	localize: func(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartEnvironment) error {
		return localizeEnvironments(ctx, store, locale, items)
	},
	toProto:     toProtoDaggerheartEnvironment,
	toProtoList: toProtoDaggerheartEnvironments,
	listConfig: contentListConfig[contentstore.DaggerheartEnvironment]{
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
		KeyFunc: func(item contentstore.DaggerheartEnvironment) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item contentstore.DaggerheartEnvironment, field string) (any, bool) {
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
