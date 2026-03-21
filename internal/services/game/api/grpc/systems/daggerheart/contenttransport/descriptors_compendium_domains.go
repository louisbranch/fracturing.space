package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/content/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

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
