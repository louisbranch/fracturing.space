package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/content/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

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
