package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/content/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

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
