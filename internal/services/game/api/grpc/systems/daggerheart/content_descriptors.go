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
	get            func(context.Context, storage.DaggerheartContentStore, string) (T, error)
	list           func(context.Context, storage.DaggerheartContentStore) ([]T, error)
	localize       func(context.Context, storage.DaggerheartContentStore, commonv1.Locale, []T) error
	toProto        func(T) *P
	toProtoList    func([]T) []*P
	listConfig     contentListConfig[T]
}

func getContentEntry[T any, P any](
	ctx context.Context,
	store storage.DaggerheartContentStore,
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
	store storage.DaggerheartContentStore,
	req contentListRequest,
	locale commonv1.Locale,
	descriptor contentDescriptor[T, P],
) ([]*P, contentPage[T], error) {
	items, err := descriptor.list(ctx, store)
	if err != nil {
		return nil, contentPage[T]{}, status.Errorf(codes.Internal, "%s: %v", descriptor.listAction, err)
	}
	page, err := listContentPage(items, req, descriptor.listConfig)
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
	get: func(ctx context.Context, store storage.DaggerheartContentStore, id string) (storage.DaggerheartClass, error) {
		return store.GetDaggerheartClass(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentStore) ([]storage.DaggerheartClass, error) {
		return store.ListDaggerheartClasses(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, items []storage.DaggerheartClass) error {
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
	get: func(ctx context.Context, store storage.DaggerheartContentStore, id string) (storage.DaggerheartSubclass, error) {
		return store.GetDaggerheartSubclass(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentStore) ([]storage.DaggerheartSubclass, error) {
		return store.ListDaggerheartSubclasses(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, items []storage.DaggerheartSubclass) error {
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
	},
}

var heritageDescriptor = contentDescriptor[storage.DaggerheartHeritage, pb.DaggerheartHeritage]{
	getAction:      "get heritage",
	listAction:     "list heritages",
	localizeAction: "localize heritages",
	get: func(ctx context.Context, store storage.DaggerheartContentStore, id string) (storage.DaggerheartHeritage, error) {
		return store.GetDaggerheartHeritage(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentStore) ([]storage.DaggerheartHeritage, error) {
		return store.ListDaggerheartHeritages(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, items []storage.DaggerheartHeritage) error {
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
	get: func(ctx context.Context, store storage.DaggerheartContentStore, id string) (storage.DaggerheartExperienceEntry, error) {
		return store.GetDaggerheartExperience(ctx, id)
	},
	list: func(ctx context.Context, store storage.DaggerheartContentStore) ([]storage.DaggerheartExperienceEntry, error) {
		return store.ListDaggerheartExperiences(ctx)
	},
	localize: func(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, items []storage.DaggerheartExperienceEntry) error {
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
