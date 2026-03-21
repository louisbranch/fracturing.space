package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/content/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

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
