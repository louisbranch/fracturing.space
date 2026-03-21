package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/content/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

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
