package contenttransport

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/content/filter"
)

type contentKeySpec struct {
	Name string
	Kind pagination.CursorValueKind
}

type contentListConfig[T any] struct {
	PageSizeConfig pagination.PageSizeConfig
	OrderByConfig  pagination.OrderByConfig
	FilterFields   contentfilter.Fields
	KeySpec        []contentKeySpec
	KeyFunc        func(T) []pagination.CursorValue
	Resolver       func(T, string) (any, bool)
	FilterHashSeed string
}

type contentListRequest struct {
	PageSize  int32
	PageToken string
	OrderBy   string
	Filter    string
	DomainID  string
}

type contentPage[T any] struct {
	Items             []T
	TotalSize         int
	NextPageToken     string
	PreviousPageToken string
}

func listContentPage[T any](items []T, req contentListRequest, cfg contentListConfig[T]) (contentPage[T], error) {
	pageSize := pagination.ClampPageSize(req.PageSize, cfg.PageSizeConfig)

	orderByInput := strings.TrimSpace(req.OrderBy)
	orderBy, err := pagination.NormalizeOrderBy(orderByInput, cfg.OrderByConfig)
	if err != nil {
		return contentPage[T]{}, err
	}
	descending := strings.HasSuffix(orderBy, " desc")

	filterStr := strings.TrimSpace(req.Filter)
	filterExpr, err := contentfilter.Parse(filterStr, cfg.FilterFields)
	if err != nil {
		return contentPage[T]{}, err
	}

	filtered, err := applyContentFilter(items, filterExpr, cfg.Resolver)
	if err != nil {
		return contentPage[T]{}, err
	}

	filterHashInput := filterStr
	if cfg.FilterHashSeed != "" {
		filterHashInput = filterHashInput + "\n" + cfg.FilterHashSeed
	}

	var cursor *pagination.Cursor
	var cursorKeys []pagination.CursorValue
	pageToken := strings.TrimSpace(req.PageToken)
	if pageToken != "" {
		decoded, err := pagination.Decode(pageToken)
		if err != nil {
			return contentPage[T]{}, err
		}
		if err := pagination.ValidateFilterHash(decoded, filterHashInput); err != nil {
			return contentPage[T]{}, err
		}
		if err := pagination.ValidateOrderHash(decoded, orderBy); err != nil {
			return contentPage[T]{}, err
		}
		cursor = &decoded
		cursorKeys, err = cursorKeysFromToken(decoded, cfg.KeySpec)
		if err != nil {
			return contentPage[T]{}, err
		}
	}

	pageItems, hasNext, hasPrev, err := paginateContent(filtered, cursor, cursorKeys, descending, pageSize, cfg.KeyFunc)
	if err != nil {
		return contentPage[T]{}, err
	}

	response := contentPage[T]{
		Items:     pageItems,
		TotalSize: len(filtered),
	}
	if len(pageItems) == 0 {
		return response, nil
	}

	if hasNext {
		last := pageItems[len(pageItems)-1]
		nextCursor := pagination.NewNextPageCursor(cfg.KeyFunc(last), descending, filterHashInput, orderBy)
		token, err := pagination.Encode(nextCursor)
		if err != nil {
			return contentPage[T]{}, err
		}
		response.NextPageToken = token
	}
	if hasPrev {
		first := pageItems[0]
		prevCursor := pagination.NewPrevPageCursor(cfg.KeyFunc(first), descending, filterHashInput, orderBy)
		token, err := pagination.Encode(prevCursor)
		if err != nil {
			return contentPage[T]{}, err
		}
		response.PreviousPageToken = token
	}

	return response, nil
}
