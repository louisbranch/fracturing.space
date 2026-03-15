package contenttransport

import (
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/content/filter"
)

type testContentItem struct {
	ID   string
	Name string
}

func testContentConfig() contentListConfig[testContentItem] {
	return contentListConfig[testContentItem]{
		PageSizeConfig: pagination.PageSizeConfig{Default: 2, Max: 5},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":   contentfilter.FieldString,
			"name": contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{{Name: "name", Kind: pagination.CursorValueString}, {Name: "id", Kind: pagination.CursorValueString}},
		KeyFunc: func(item testContentItem) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item testContentItem, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			default:
				return nil, false
			}
		},
	}
}
