package sqlite

import (
	"fmt"
	"strings"

	corefilter "github.com/louisbranch/fracturing.space/internal/services/game/core/filter"
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

func buildListEventsPageSQLPlan(req storage.ListEventsPageRequest) (listEventsPageSQLPlan, error) {
	whereClause := "campaign_id = ?"
	params := []any{req.CampaignID}
	if req.AfterSeq > 0 {
		whereClause += " AND seq > ?"
		params = append(params, req.AfterSeq)
	}
	if req.CursorSeq > 0 {
		if req.CursorDir == "bwd" {
			whereClause += " AND seq < ?"
		} else {
			whereClause += " AND seq > ?"
		}
		params = append(params, req.CursorSeq)
	}

	orderClause := "ORDER BY seq ASC"
	if req.Descending {
		orderClause = "ORDER BY seq DESC"
	}
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

	filterClause, filterParams, err := compileEventQueryFilter(req.Filter)
	if err != nil {
		return listEventsPageSQLPlan{}, err
	}
	if filterClause != "" {
		whereClause += " AND " + filterClause
		params = append(params, filterParams...)
		countWhereClause += " AND " + filterClause
		countParams = append(countParams, filterParams...)
	}

	return listEventsPageSQLPlan{
		whereClause:      whereClause,
		params:           params,
		orderClause:      orderClause,
		limitClause:      fmt.Sprintf("LIMIT %d", req.PageSize+1),
		countWhereClause: countWhereClause,
		countParams:      countParams,
	}, nil
}

func compileEventQueryFilter(filter storage.EventQueryFilter) (string, []any, error) {
	var (
		clauses []string
		params  []any
	)

	if expression := strings.TrimSpace(filter.Expression); expression != "" {
		cond, err := corefilter.ParseEventFilter(expression)
		if err != nil {
			return "", nil, fmt.Errorf("parse event filter expression: %w", err)
		}
		if cond.Clause != "" {
			clauses = append(clauses, "("+cond.Clause+")")
			params = append(params, cond.Params...)
		}
	}

	appendExact := func(value, column string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		clauses = append(clauses, column+" = ?")
		params = append(params, value)
	}

	appendExact(filter.EventType, "event_type")
	appendExact(filter.SessionID, "session_id")
	appendExact(filter.SceneID, "scene_id")
	appendExact(filter.RequestID, "request_id")
	appendExact(filter.InvocationID, "invocation_id")
	appendExact(filter.ActorType, "actor_type")
	appendExact(filter.ActorID, "actor_id")
	appendExact(filter.SystemID, "system_id")
	appendExact(filter.SystemVersion, "system_version")
	appendExact(filter.EntityType, "entity_type")
	appendExact(filter.EntityID, "entity_id")

	if len(clauses) == 0 {
		return "", nil, nil
	}
	return strings.Join(clauses, " AND "), params, nil
}
