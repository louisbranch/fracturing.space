package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ListTimelineEntries returns a paginated timeline view for a campaign.
func (s *EventService) ListTimelineEntries(ctx context.Context, in *campaignv1.ListTimelineEntriesRequest) (*campaignv1.ListTimelineEntriesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign_id is required")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListEventsPageSize,
		Max:     maxListEventsPageSize,
	})

	orderBy, err := pagination.NormalizeOrderBy(strings.TrimSpace(in.GetOrderBy()), pagination.OrderByConfig{
		Default: "seq",
		Allowed: []string{"seq", "seq desc"},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_by: %s (must be 'seq' or 'seq desc')", strings.TrimSpace(in.GetOrderBy()))
	}
	descending := orderBy == "seq desc"

	filterStr := strings.TrimSpace(in.GetFilter())
	var filterClause string
	var filterParams []any
	if filterStr != "" {
		cond, err := filter.ParseEventFilter(filterStr)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
		}
		filterClause = cond.Clause
		filterParams = cond.Params
	}

	var cursorSeq uint64
	var cursorDir string
	var cursorReverse bool
	pageToken := strings.TrimSpace(in.GetPageToken())
	if pageToken != "" {
		c, err := pagination.Decode(pageToken)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid page_token: %v", err)
		}
		if err := pagination.ValidateFilterHash(c, filterStr); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		if err := pagination.ValidateOrderHash(c, orderBy); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		seqValue, err := pagination.ValueUint(c, "seq")
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		cursorSeq = seqValue
		cursorDir = string(c.Dir)
		cursorReverse = c.Reverse
	}

	req := storage.ListEventsPageRequest{
		CampaignID:    campaignID,
		PageSize:      pageSize,
		CursorSeq:     cursorSeq,
		CursorDir:     cursorDir,
		CursorReverse: cursorReverse,
		Descending:    descending,
		FilterClause:  filterClause,
		FilterParams:  filterParams,
	}

	result, err := s.stores.Event.ListEventsPage(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list timeline entries: %v", err)
	}

	resolver := newTimelineProjectionResolver(s.stores)
	response := &campaignv1.ListTimelineEntriesResponse{
		Entries:   make([]*campaignv1.TimelineEntry, 0, len(result.Events)),
		TotalSize: int32(result.TotalCount),
	}

	for _, evt := range result.Events {
		entry, err := timelineEntryFromEvent(ctx, resolver, evt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "resolve timeline entry: %v", err)
		}
		response.Entries = append(response.Entries, entry)
	}

	if len(result.Events) > 0 {
		if result.HasNextPage {
			lastSeq := result.Events[len(result.Events)-1].Seq
			nextCursor := pagination.NewNextPageCursor(
				[]pagination.CursorValue{pagination.UintValue("seq", lastSeq)},
				descending,
				filterStr,
				orderBy,
			)
			token, err := pagination.Encode(nextCursor)
			if err == nil {
				response.NextPageToken = token
			}
		}
		if result.HasPrevPage {
			firstSeq := result.Events[0].Seq
			prevCursor := pagination.NewPrevPageCursor(
				[]pagination.CursorValue{pagination.UintValue("seq", firstSeq)},
				descending,
				filterStr,
				orderBy,
			)
			token, err := pagination.Encode(prevCursor)
			if err == nil {
				response.PreviousPageToken = token
			}
		}
	}

	return response, nil
}

// timelineProjectionResolver caches projection reads to keep timeline responses consistent.
type timelineProjectionResolver struct {
	campaignStore    storage.CampaignStore
	participantStore storage.ParticipantStore
	characterStore   storage.CharacterStore
	sessionStore     storage.SessionStore

	campaignCache    map[string]storage.CampaignRecord
	participantCache map[string]storage.ParticipantRecord
	characterCache   map[string]storage.CharacterRecord
	sessionCache     map[string]storage.SessionRecord
}

// newTimelineProjectionResolver prepares a resolver for projection lookups.
func newTimelineProjectionResolver(stores Stores) *timelineProjectionResolver {
	return &timelineProjectionResolver{
		campaignStore:    stores.Campaign,
		participantStore: stores.Participant,
		characterStore:   stores.Character,
		sessionStore:     stores.Session,
		campaignCache:    make(map[string]storage.CampaignRecord),
		participantCache: make(map[string]storage.ParticipantRecord),
		characterCache:   make(map[string]storage.CharacterRecord),
		sessionCache:     make(map[string]storage.SessionRecord),
	}
}

// timelineEntryFromEvent builds a timeline entry with projection context.
func timelineEntryFromEvent(ctx context.Context, resolver *timelineProjectionResolver, evt event.Event) (*campaignv1.TimelineEntry, error) {
	iconID, projection, err := resolver.resolve(ctx, evt)
	if err != nil {
		return nil, err
	}
	changeFields := timelineChangeFields(evt)
	if len(changeFields) > 0 {
		if projection == nil {
			projection = &campaignv1.ProjectionDisplay{}
		}
		projection.Fields = append(projection.Fields, changeFields...)
	}
	return &campaignv1.TimelineEntry{
		Seq:              evt.Seq,
		EventType:        string(evt.Type),
		EventTime:        timestamppb.New(evt.Timestamp),
		IconId:           iconID,
		Projection:       projection,
		EventPayloadJson: string(evt.PayloadJSON),
	}, nil
}

func timelineChangeFields(evt event.Event) []*campaignv1.ProjectionField {
	switch evt.Type {
	case event.Type("sys.daggerheart.character_state_patched"):
		return daggerheartStateChangeFields(evt.PayloadJSON)
	default:
		return nil
	}
}

func daggerheartStateChangeFields(payloadJSON []byte) []*campaignv1.ProjectionField {
	if len(payloadJSON) == 0 {
		return nil
	}
	var payload daggerheart.CharacterStatePatchedPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil
	}
	fields := make([]*campaignv1.ProjectionField, 0, 6)
	appendIntChange(&fields, "HP", payload.HPBefore, payload.HPAfter)
	appendIntChange(&fields, "Hope", payload.HopeBefore, payload.HopeAfter)
	appendIntChange(&fields, "Hope Max", payload.HopeMaxBefore, payload.HopeMaxAfter)
	appendIntChange(&fields, "Stress", payload.StressBefore, payload.StressAfter)
	appendIntChange(&fields, "Armor", payload.ArmorBefore, payload.ArmorAfter)
	appendStringChange(&fields, "Life State", payload.LifeStateBefore, payload.LifeStateAfter)
	return fields
}

func appendIntChange(fields *[]*campaignv1.ProjectionField, label string, before, after *int) {
	if after == nil {
		return
	}
	if before != nil {
		if *before == *after {
			return
		}
		*fields = append(*fields, &campaignv1.ProjectionField{
			Label: label,
			Value: strconv.Itoa(*before) + " -> " + strconv.Itoa(*after),
		})
		return
	}
	*fields = append(*fields, &campaignv1.ProjectionField{
		Label: label,
		Value: "= " + strconv.Itoa(*after),
	})
}

func appendStringChange(fields *[]*campaignv1.ProjectionField, label string, before, after *string) {
	if after == nil {
		return
	}
	afterValue := strings.TrimSpace(*after)
	if afterValue == "" {
		return
	}
	if before != nil {
		beforeValue := strings.TrimSpace(*before)
		if beforeValue == afterValue {
			return
		}
		if beforeValue != "" {
			*fields = append(*fields, &campaignv1.ProjectionField{
				Label: label,
				Value: beforeValue + " -> " + afterValue,
			})
			return
		}
	}
	*fields = append(*fields, &campaignv1.ProjectionField{
		Label: label,
		Value: "= " + afterValue,
	})
}

func eventDomainFromType(evtType event.Type) string {
	value := strings.TrimSpace(string(evtType))
	if value == "" {
		return ""
	}
	parts := strings.SplitN(value, ".", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// resolve maps an event to the icon and projection display used in the timeline.
func (r *timelineProjectionResolver) resolve(ctx context.Context, evt event.Event) (commonv1.IconId, *campaignv1.ProjectionDisplay, error) {
	domain := strings.TrimSpace(evt.EntityType)
	if domain == "" {
		domain = strings.TrimSpace(eventDomainFromType(evt.Type))
	}
	domain = strings.ToLower(domain)

	entityID := strings.TrimSpace(evt.EntityID)
	if entityID == "" {
		switch domain {
		case "campaign":
			entityID = strings.TrimSpace(evt.CampaignID)
		case "session":
			entityID = strings.TrimSpace(evt.SessionID)
		}
	}

	switch domain {
	case "campaign":
		entry, ok, err := r.lookupCampaign(ctx, entityID)
		if err != nil {
			return commonv1.IconId_ICON_ID_CAMPAIGN, nil, err
		}
		if !ok {
			return commonv1.IconId_ICON_ID_CAMPAIGN, nil, nil
		}
		return commonv1.IconId_ICON_ID_CAMPAIGN, campaignProjectionDisplay(entry), nil
	case "participant":
		entry, ok, err := r.lookupParticipant(ctx, evt.CampaignID, entityID)
		if err != nil {
			return commonv1.IconId_ICON_ID_PARTICIPANT, nil, err
		}
		if !ok {
			return commonv1.IconId_ICON_ID_PARTICIPANT, nil, nil
		}
		return commonv1.IconId_ICON_ID_PARTICIPANT, participantProjectionDisplay(entry), nil
	case "character":
		entry, ok, err := r.lookupCharacter(ctx, evt.CampaignID, entityID)
		if err != nil {
			return commonv1.IconId_ICON_ID_CHARACTER, nil, err
		}
		if !ok {
			return commonv1.IconId_ICON_ID_CHARACTER, nil, nil
		}
		return commonv1.IconId_ICON_ID_CHARACTER, characterProjectionDisplay(entry), nil
	case "session":
		entry, ok, err := r.lookupSession(ctx, evt.CampaignID, entityID)
		if err != nil {
			return commonv1.IconId_ICON_ID_SESSION, nil, err
		}
		if !ok {
			return commonv1.IconId_ICON_ID_SESSION, nil, nil
		}
		return commonv1.IconId_ICON_ID_SESSION, sessionProjectionDisplay(entry), nil
	default:
		return commonv1.IconId_ICON_ID_GENERIC, nil, nil
	}
}

// lookupCampaign fetches campaign projection data, using cache when available.
func (r *timelineProjectionResolver) lookupCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, bool, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return storage.CampaignRecord{}, false, nil
	}
	if r.campaignStore == nil {
		return storage.CampaignRecord{}, false, fmt.Errorf("campaign store is not configured")
	}
	if cached, ok := r.campaignCache[campaignID]; ok {
		return cached, true, nil
	}
	entry, err := r.campaignStore.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.CampaignRecord{}, false, nil
		}
		return storage.CampaignRecord{}, false, err
	}
	r.campaignCache[campaignID] = entry
	return entry, true, nil
}

// lookupParticipant fetches participant projection data, using cache when available.
func (r *timelineProjectionResolver) lookupParticipant(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, bool, error) {
	campaignID = strings.TrimSpace(campaignID)
	participantID = strings.TrimSpace(participantID)
	if campaignID == "" || participantID == "" {
		return storage.ParticipantRecord{}, false, nil
	}
	if r.participantStore == nil {
		return storage.ParticipantRecord{}, false, fmt.Errorf("participant store is not configured")
	}
	key := campaignID + ":" + participantID
	if cached, ok := r.participantCache[key]; ok {
		return cached, true, nil
	}
	entry, err := r.participantStore.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.ParticipantRecord{}, false, nil
		}
		return storage.ParticipantRecord{}, false, err
	}
	r.participantCache[key] = entry
	return entry, true, nil
}

// lookupCharacter fetches character projection data, using cache when available.
func (r *timelineProjectionResolver) lookupCharacter(ctx context.Context, campaignID, characterID string) (storage.CharacterRecord, bool, error) {
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return storage.CharacterRecord{}, false, nil
	}
	if r.characterStore == nil {
		return storage.CharacterRecord{}, false, fmt.Errorf("character store is not configured")
	}
	key := campaignID + ":" + characterID
	if cached, ok := r.characterCache[key]; ok {
		return cached, true, nil
	}
	entry, err := r.characterStore.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.CharacterRecord{}, false, nil
		}
		return storage.CharacterRecord{}, false, err
	}
	r.characterCache[key] = entry
	return entry, true, nil
}

// lookupSession fetches session projection data, using cache when available.
func (r *timelineProjectionResolver) lookupSession(ctx context.Context, campaignID, sessionID string) (storage.SessionRecord, bool, error) {
	campaignID = strings.TrimSpace(campaignID)
	sessionID = strings.TrimSpace(sessionID)
	if campaignID == "" || sessionID == "" {
		return storage.SessionRecord{}, false, nil
	}
	if r.sessionStore == nil {
		return storage.SessionRecord{}, false, fmt.Errorf("session store is not configured")
	}
	key := campaignID + ":" + sessionID
	if cached, ok := r.sessionCache[key]; ok {
		return cached, true, nil
	}
	entry, err := r.sessionStore.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.SessionRecord{}, false, nil
		}
		return storage.SessionRecord{}, false, err
	}
	r.sessionCache[key] = entry
	return entry, true, nil
}

// campaignProjectionDisplay builds a display summary for campaign projections.
func campaignProjectionDisplay(entry storage.CampaignRecord) *campaignv1.ProjectionDisplay {
	display := &campaignv1.ProjectionDisplay{
		Title: entry.Name,
	}
	switch entry.System {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		display.Subtitle = "DAGGERHEART"
	}
	switch entry.Status {
	case campaign.StatusDraft:
		display.Status = "DRAFT"
	case campaign.StatusActive:
		display.Status = "ACTIVE"
	case campaign.StatusCompleted:
		display.Status = "COMPLETED"
	case campaign.StatusArchived:
		display.Status = "ARCHIVED"
	}
	return display
}

// participantProjectionDisplay builds a display summary for participant projections.
func participantProjectionDisplay(entry storage.ParticipantRecord) *campaignv1.ProjectionDisplay {
	display := &campaignv1.ProjectionDisplay{
		Title: entry.Name,
	}
	switch entry.Role {
	case participant.RoleGM:
		display.Subtitle = "GM"
	case participant.RolePlayer:
		display.Subtitle = "PLAYER"
	}
	switch entry.Controller {
	case participant.ControllerHuman:
		display.Status = "HUMAN"
	case participant.ControllerAI:
		display.Status = "AI"
	}
	return display
}

// characterProjectionDisplay builds a display summary for character projections.
func characterProjectionDisplay(entry storage.CharacterRecord) *campaignv1.ProjectionDisplay {
	display := &campaignv1.ProjectionDisplay{
		Title: entry.Name,
	}
	switch entry.Kind {
	case character.KindPC:
		display.Subtitle = "PC"
	case character.KindNPC:
		display.Subtitle = "NPC"
	}
	return display
}

// sessionProjectionDisplay builds a display summary for session projections.
func sessionProjectionDisplay(entry storage.SessionRecord) *campaignv1.ProjectionDisplay {
	display := &campaignv1.ProjectionDisplay{
		Title: entry.Name,
	}
	switch entry.Status {
	case session.StatusActive:
		display.Status = "ACTIVE"
	case session.StatusEnded:
		display.Status = "ENDED"
	}
	return display
}
