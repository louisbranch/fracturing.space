package sqlite

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type listEventsPageSQLPlan struct {
	whereClause      string
	params           []any
	orderClause      string
	limitClause      string
	countWhereClause string
	countParams      []any
}

func buildListEventsPageSQLPlan(req storage.ListEventsPageRequest) listEventsPageSQLPlan {
	whereClause := "campaign_id = ?"
	params := []any{req.CampaignID}
	if req.AfterSeq > 0 {
		whereClause += " AND seq > ?"
		params = append(params, req.AfterSeq)
	}

	// The cursor direction determines comparison operators; sort order is applied separately.
	if req.CursorSeq > 0 {
		if req.CursorDir == "bwd" {
			whereClause += " AND seq < ?"
		} else {
			whereClause += " AND seq > ?"
		}
		params = append(params, req.CursorSeq)
	}

	if req.FilterClause != "" {
		whereClause += " AND " + req.FilterClause
		params = append(params, req.FilterParams...)
	}

	orderClause := "ORDER BY seq ASC"
	if req.Descending {
		orderClause = "ORDER BY seq DESC"
	}
	// Reverse sort temporarily for previous-page queries so near-edge rows are fetched first.
	if req.CursorReverse {
		if req.Descending {
			orderClause = "ORDER BY seq ASC"
		} else {
			orderClause = "ORDER BY seq DESC"
		}
	}

	countWhereClause := "campaign_id = ?"
	countParams := []any{req.CampaignID}
	if req.AfterSeq > 0 {
		countWhereClause += " AND seq > ?"
		countParams = append(countParams, req.AfterSeq)
	}
	if req.FilterClause != "" {
		countWhereClause += " AND " + req.FilterClause
		countParams = append(countParams, req.FilterParams...)
	}

	return listEventsPageSQLPlan{
		whereClause:      whereClause,
		params:           params,
		orderClause:      orderClause,
		limitClause:      fmt.Sprintf("LIMIT %d", req.PageSize+1),
		countWhereClause: countWhereClause,
		countParams:      countParams,
	}
}
