package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/content/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

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
