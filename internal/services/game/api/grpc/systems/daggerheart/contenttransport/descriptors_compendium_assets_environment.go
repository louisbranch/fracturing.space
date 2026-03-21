package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/content/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

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
